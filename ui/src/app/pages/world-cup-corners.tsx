import {InfoCircleOutlined, SearchOutlined} from '@ant-design/icons';
import type {ColumnsType} from 'antd/es/table';
import {Alert, Button, Input, Progress, Segmented, Select, Space, Tag, Typography} from 'antd';
import * as React from 'react';
import {AppPage, ResourceTable} from '../components';
import {WorldCupCornerMatch, WorldCupCornerStageKey, worldCupCornerMatches, worldCupCornerStages} from './world-cup-corners-data';

type OutcomeFilter = 'all' | 'hit' | 'miss';

const overThreshold = 6.5;
const corners90Total = (item: WorldCupCornerMatch) => item.homeCorners90 + item.awayCorners90;
const cornersFullTotal = (item: WorldCupCornerMatch) => item.homeCornersFull + item.awayCornersFull;
const hitsOver = (item: WorldCupCornerMatch) => corners90Total(item) > overThreshold;

const summarize = (items: WorldCupCornerMatch[]) => {
    const hits = items.filter(hitsOver).length;
    const average = items.length === 0 ? 0 : items.reduce((total, item) => total + corners90Total(item), 0) / items.length;
    return {
        matches: items.length,
        hits,
        hitRate: items.length === 0 ? 0 : (hits / items.length) * 100,
        average
    };
};

const stageByKey = new Map(worldCupCornerStages.map(stage => [stage.key, stage]));
const tournamentSummary = summarize(worldCupCornerMatches);
const knockoutSummary = summarize(worldCupCornerMatches.filter(item => stageByKey.get(item.stage)?.knockout));
const stageSummaries = worldCupCornerStages.map(stage => ({stage, summary: summarize(worldCupCornerMatches.filter(item => item.stage === stage.key))}));

const formatPair = (home: number, away: number) => `${home}–${away}`;
const formatRate = (value: number) => `${value.toFixed(1)}%`;

const KpiCard = (props: {label: string; value: string; detail: string; tone?: 'primary' | 'good'}) => (
    <article className={`world-cup-corners-kpi world-cup-corners-kpi--${props.tone || 'primary'}`}>
        <Typography.Text className='world-cup-corners-kpi__label'>{props.label}</Typography.Text>
        <Typography.Text className='world-cup-corners-kpi__value'>{props.value}</Typography.Text>
        <Typography.Text className='world-cup-corners-kpi__detail'>{props.detail}</Typography.Text>
    </article>
);

const Score = (props: {item: WorldCupCornerMatch}) => (
    <span className='world-cup-corners-score'>
        <span>{formatPair(props.item.homeScore, props.item.awayScore)}</span>
        {props.item.homePenaltyScore !== undefined && props.item.awayPenaltyScore !== undefined && (
            <span className='world-cup-corners-score__penalties'>PEN {formatPair(props.item.homePenaltyScore, props.item.awayPenaltyScore)}</span>
        )}
    </span>
);

export const WorldCupCornersPage = () => {
    const [search, setSearch] = React.useState('');
    const [stage, setStage] = React.useState<WorldCupCornerStageKey | 'all'>('all');
    const [outcome, setOutcome] = React.useState<OutcomeFilter>('all');
    const stageHeadingID = React.useId();
    const tableHeadingID = React.useId();

    const filteredItems = React.useMemo(() => {
        const query = search.trim().toLowerCase();
        return worldCupCornerMatches.filter(item => {
            if (stage !== 'all' && item.stage !== stage) {
                return false;
            }
            if (outcome === 'hit' && !hitsOver(item)) {
                return false;
            }
            if (outcome === 'miss' && hitsOver(item)) {
                return false;
            }
            return !query || item.homeTeam.toLowerCase().includes(query) || item.awayTeam.toLowerCase().includes(query);
        });
    }, [outcome, search, stage]);

    const columns: ColumnsType<WorldCupCornerMatch> = [
        {
            title: 'Stage',
            dataIndex: 'stage',
            width: 160,
            render: value => <Tag>{stageByKey.get(value)?.shortLabel || value}</Tag>
        },
        {
            title: 'Match',
            key: 'match',
            width: 250,
            render: item => (
                <span className='world-cup-corners-match'>
                    <strong>{item.homeTeam}</strong>
                    <span aria-hidden='true'>–</span>
                    <strong>{item.awayTeam}</strong>
                </span>
            )
        },
        {
            title: 'Score after ET',
            key: 'score',
            width: 150,
            align: 'center',
            render: item => <Score item={item} />
        },
        {
            title: '90-min corners',
            key: 'corners90',
            width: 140,
            align: 'center',
            render: item => <span className='world-cup-corners-number'>{formatPair(item.homeCorners90, item.awayCorners90)}</span>
        },
        {
            title: '90-min total',
            key: 'corners90Total',
            width: 120,
            align: 'center',
            sorter: (left, right) => corners90Total(left) - corners90Total(right),
            render: item => <span className='world-cup-corners-number world-cup-corners-number--strong'>{corners90Total(item)}</span>
        },
        {
            title: 'O6.5',
            key: 'over65',
            width: 105,
            align: 'center',
            filters: [
                {text: 'Hit', value: 'hit'},
                {text: 'Miss', value: 'miss'}
            ],
            onFilter: (value, item) => (value === 'hit' ? hitsOver(item) : !hitsOver(item)),
            render: item => (
                <Tag color={hitsOver(item) ? 'green' : 'default'} className='world-cup-corners-result'>
                    {hitsOver(item) ? 'HIT' : 'MISS'}
                </Tag>
            )
        },
        {
            title: 'Full-match corners',
            key: 'cornersFull',
            width: 165,
            align: 'center',
            sorter: (left, right) => cornersFullTotal(left) - cornersFullTotal(right),
            render: item => <span className='world-cup-corners-number'>{formatPair(item.homeCornersFull, item.awayCornersFull)}</span>
        }
    ];

    const resetFilters = () => {
        setSearch('');
        setStage('all');
        setOutcome('all');
    };
    const filtersActive = Boolean(search || stage !== 'all' || outcome !== 'all');

    return (
        <AppPage
            title='2022 World Cup Corner Analysis'
            subtitle={
                <span>
                    64 matches · Data derived from{' '}
                    <Typography.Link href='https://github.com/statsbomb/open-data' target='_blank' rel='noopener noreferrer'>
                        StatsBomb Open Data
                    </Typography.Link>
                </span>
            }>
            <Alert
                className='world-cup-corners-notice'
                type='info'
                showIcon={true}
                icon={<InfoCircleOutlined />}
                title='Settlement scope'
                description={
                    <span>
                        The O6.5 analysis uses corners taken during 90 minutes plus stoppage time. Extra-time corners appear only in the full-match column. Always confirm the{' '}
                        <Typography.Link href='https://docs.polymarket.us/faqs/sports-faqs' target='_blank' rel='noopener noreferrer'>
                            market-specific rules
                        </Typography.Link>{' '}
                        before trading.
                    </span>
                }
            />

            <section className='world-cup-corners-kpis' aria-label='Tournament summary'>
                <KpiCard label='Sample size' value={`${tournamentSummary.matches}`} detail='matches in the 2022 tournament' />
                <KpiCard label='O6.5 overall' value={formatRate(tournamentSummary.hitRate)} detail={`${tournamentSummary.hits}/${tournamentSummary.matches} matches`} tone='good' />
                <KpiCard label='O6.5 knockout' value={formatRate(knockoutSummary.hitRate)} detail={`${knockoutSummary.hits}/${knockoutSummary.matches} matches`} tone='good' />
                <KpiCard label='Average 90-min corners' value={tournamentSummary.average.toFixed(2)} detail='combined corners per match' />
            </section>

            <section className='world-cup-corners-stage-panel' aria-labelledby={stageHeadingID}>
                <header className='world-cup-corners-section-header'>
                    <div>
                        <Typography.Title id={stageHeadingID} level={2}>
                            O6.5 hit rate by stage
                        </Typography.Title>
                        <Typography.Text type='secondary'>Visible sample sizes keep small knockout rounds in context.</Typography.Text>
                    </div>
                </header>
                <div className='world-cup-corners-stage-list'>
                    {stageSummaries.map(({stage: stageItem, summary}) => (
                        <div className='world-cup-corners-stage-row' key={stageItem.key}>
                            <Typography.Text className='world-cup-corners-stage-row__label'>{stageItem.shortLabel}</Typography.Text>
                            <Progress
                                className='world-cup-corners-stage-row__progress'
                                percent={Number(summary.hitRate.toFixed(1))}
                                showInfo={false}
                                strokeColor='var(--athena-green)'
                                trailColor='var(--athena-bg-soft)'
                            />
                            <span className='world-cup-corners-stage-row__value'>
                                <strong>{formatRate(summary.hitRate)}</strong>
                                <small>
                                    {summary.hits}/{summary.matches}
                                </small>
                            </span>
                        </div>
                    ))}
                </div>
            </section>

            <section className='world-cup-corners-table-panel' aria-labelledby={tableHeadingID}>
                <header className='world-cup-corners-table-header'>
                    <div>
                        <Typography.Title id={tableHeadingID} level={2}>
                            Match data
                        </Typography.Title>
                        <Typography.Text type='secondary'>
                            Showing {filteredItems.length} of {worldCupCornerMatches.length} matches
                        </Typography.Text>
                    </div>
                    <Space className='world-cup-corners-filters' size={10} wrap={true}>
                        <Input
                            aria-label='Search by team'
                            allowClear={true}
                            prefix={<SearchOutlined />}
                            placeholder='Search team'
                            value={search}
                            onChange={event => setSearch(event.target.value)}
                        />
                        <Select<WorldCupCornerStageKey | 'all'>
                            aria-label='Filter by stage'
                            value={stage}
                            onChange={setStage}
                            options={[{value: 'all', label: 'All stages'}, ...worldCupCornerStages.map(item => ({value: item.key, label: item.label}))]}
                        />
                        <Segmented<OutcomeFilter>
                            aria-label='Filter by O6.5 outcome'
                            value={outcome}
                            onChange={setOutcome}
                            options={[
                                {label: 'All', value: 'all'},
                                {label: 'O6.5 Hit', value: 'hit'},
                                {label: 'Miss', value: 'miss'}
                            ]}
                        />
                        <Button disabled={!filtersActive} onClick={resetFilters}>
                            Reset
                        </Button>
                    </Space>
                </header>
                <ResourceTable rowKey='id' label='2022 World Cup corner data' items={filteredItems} columns={columns} scrollX={1090} stickyHeader={true} />
            </section>

            <Typography.Paragraph className='world-cup-corners-footnote' type='secondary'>
                Method: 90-minute corners count StatsBomb corner events in periods 1–2; full-match corners also include periods 3–4. Penalty shootout results are shown separately
                from the score after extra time.
            </Typography.Paragraph>
        </AppPage>
    );
};
