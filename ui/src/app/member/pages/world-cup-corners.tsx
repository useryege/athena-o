import {InfoCircleOutlined, SearchOutlined} from '@ant-design/icons';
import type {ColumnsType} from 'antd/es/table';
import {Alert, Button, Input, Progress, Select, Tag, Typography} from 'antd';
import * as React from 'react';
import {AppPage, ResourceTable, useAsyncData} from '../../components';
import {memberServices as services} from '../services';
import type {WorldCupCornerMatch, WorldCupCornerStageKey} from '../../shared/services/world-cup-corners-service';

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
        {props.item.hasPenaltyShootout && <span className='world-cup-corners-score__penalties'>PEN {formatPair(props.item.homePenaltyScore, props.item.awayPenaltyScore)}</span>}
    </span>
);

export const WorldCupCornersPage = () => {
    const dataset = useAsyncData(() => services.worldCupCorners.getDataset(), []);
    const stages = dataset.data?.stages || [];
    const matches = dataset.data?.matches || [];
    const [search, setSearch] = React.useState('');
    const [stage, setStage] = React.useState<WorldCupCornerStageKey | 'all'>('all');
    const [outcome, setOutcome] = React.useState<OutcomeFilter>('all');
    const [sort, setSort] = React.useState('tournament');
    const [methodOpen, setMethodOpen] = React.useState(false);
    const stageHeadingID = React.useId();
    const tableHeadingID = React.useId();
    const stageByKey = React.useMemo(() => new Map(stages.map(item => [item.key, item])), [stages]);
    const tournamentSummary = React.useMemo(() => summarize(matches), [matches]);
    const knockoutSummary = React.useMemo(() => summarize(matches.filter(item => stageByKey.get(item.stage)?.knockout)), [matches, stageByKey]);
    const stageSummaries = React.useMemo(
        () => stages.map(stageItem => ({stage: stageItem, summary: summarize(matches.filter(item => item.stage === stageItem.key))})),
        [matches, stages]
    );

    const filteredItems = React.useMemo(() => {
        const query = search.trim().toLowerCase();
        const filtered = matches.filter(item => {
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
        if (sort === 'corners90') filtered.sort((a, b) => corners90Total(b) - corners90Total(a));
        if (sort === 'corners90Asc') filtered.sort((a, b) => corners90Total(a) - corners90Total(b));
        if (sort === 'cornersFullAsc') filtered.sort((a, b) => cornersFullTotal(a) - cornersFullTotal(b));
        if (sort === 'cornersFull') filtered.sort((a, b) => cornersFullTotal(b) - cornersFullTotal(a));
        return filtered;
    }, [matches, outcome, search, stage, sort]);

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
            render: item => <span className='world-cup-corners-number world-cup-corners-number--strong'>{corners90Total(item)}</span>
        },
        {
            title: 'O6.5',
            key: 'over65',
            width: 105,
            align: 'center',
            render: item => (
                <Tag color={hitsOver(item) ? 'success' : 'default'} className='world-cup-corners-result'>
                    {hitsOver(item) ? 'HIT' : 'MISS'}
                </Tag>
            )
        },
        {
            title: 'Full-match corners',
            key: 'cornersFull',
            width: 165,
            align: 'center',
            render: item => <span className='world-cup-corners-number'>{formatPair(item.homeCornersFull, item.awayCornersFull)}</span>
        }
    ];

    const resetFilters = () => {
        setSearch('');
        setStage('all');
        setOutcome('all');
        setSort('tournament');
    };
    const filtersActive = Boolean(search || stage !== 'all' || outcome !== 'all' || sort !== 'tournament');

    return (
        <div className='market-intelligence-page corners-page'>
            <AppPage
                title='2022 World Cup Corner Analysis'
                loading={dataset.loading}
                error={dataset.error}
                onRefresh={dataset.reload}
                subtitle={
                    <span>
                        {dataset.data ? `${matches.length} matches · ` : ''}Data derived from{' '}
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

                <Button onClick={() => setMethodOpen(!methodOpen)} aria-expanded={methodOpen}>
                    View methodology &amp; sources
                </Button>
                {methodOpen && (
                    <p className='world-cup-corners-footnote'>
                        90-minute corners count periods 1–2; full-match corners also include periods 3–4. Penalty shootouts are separate from the score after extra time. The
                        displayed sample and hit rates use the returned match dataset.
                    </p>
                )}
                {dataset.error && dataset.data && <p className='radar-warmup'>Stale dataset. Refresh to retry.</p>}
                {dataset.data && (
                    <>
                        <section className='world-cup-corners-kpis' aria-label='Tournament summary'>
                            <KpiCard label='Sample size' value={`${tournamentSummary.matches}`} detail='matches in the 2022 tournament' />
                            <KpiCard
                                label='O6.5 overall'
                                value={formatRate(tournamentSummary.hitRate)}
                                detail={`${tournamentSummary.hits}/${tournamentSummary.matches} matches`}
                                tone='good'
                            />
                            <KpiCard
                                label='O6.5 knockout'
                                value={formatRate(knockoutSummary.hitRate)}
                                detail={`${knockoutSummary.hits}/${knockoutSummary.matches} matches`}
                                tone='good'
                            />
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
                                            aria-label={`${stageItem.label} O6.5 hit rate`}
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
                                        Showing {filteredItems.length} of {matches.length} matches
                                    </Typography.Text>
                                </div>
                                <div className='world-cup-corners-filters'>
                                    <label>
                                        Team
                                        <Input
                                            aria-label='Search by team'
                                            allowClear
                                            prefix={<SearchOutlined />}
                                            placeholder='Search team'
                                            value={search}
                                            onChange={event => setSearch(event.target.value)}
                                        />
                                    </label>
                                    <label>
                                        Stage
                                        <Select
                                            aria-label='Filter by stage'
                                            value={stage}
                                            options={[{value: 'all', label: 'All stages'}, ...stages.map(item => ({value: item.key, label: item.label}))]}
                                            onChange={setStage}
                                        />
                                    </label>
                                    <label>
                                        O6.5 outcome
                                        <Select
                                            aria-label='Filter by O6.5 outcome'
                                            value={outcome}
                                            onChange={setOutcome}
                                            options={[
                                                {value: 'all', label: 'All outcomes'},
                                                {value: 'hit', label: 'Hit'},
                                                {value: 'miss', label: 'Miss'}
                                            ]}
                                        />
                                    </label>
                                    <Button disabled={!filtersActive} onClick={resetFilters}>
                                        Reset
                                    </Button>
                                    <label>
                                        Sort matches
                                        <Select
                                            aria-label='Sort matches'
                                            value={sort}
                                            onChange={setSort}
                                            options={[
                                                {value: 'tournament', label: 'Tournament order'},
                                                {value: 'corners90', label: '90-min total, highest first'},
                                                {value: 'corners90Asc', label: '90-min total, lowest first'},
                                                {value: 'cornersFullAsc', label: 'Full-match total, lowest first'},
                                                {value: 'cornersFull', label: 'Full-match total, highest first'}
                                            ]}
                                        />
                                    </label>
                                </div>
                            </header>
                            <ResourceTable
                                rowKey='id'
                                label='2022 World Cup corner data'
                                items={filteredItems}
                                columns={columns}
                                scrollX={1090}
                                stickyHeader={true}
                                compactRender={item => (
                                    <article>
                                        <span className='radar-description'>{stageByKey.get(item.stage)?.shortLabel || item.stage}</span>
                                        <h3 className='radar-heading'>
                                            {item.homeTeam} – {item.awayTeam}
                                        </h3>
                                        <dl className='market-fact-grid'>
                                            <div>
                                                <dt>90-min corners</dt>
                                                <dd>{formatPair(item.homeCorners90, item.awayCorners90)}</dd>
                                            </div>
                                            <div>
                                                <dt>90-min total / O6.5</dt>
                                                <dd>
                                                    {corners90Total(item)} · <span className={hitsOver(item) ? 'radar-up' : ''}>{hitsOver(item) ? 'HIT' : 'MISS'}</span>
                                                </dd>
                                            </div>
                                            <div>
                                                <dt>Score after ET</dt>
                                                <dd>
                                                    <Score item={item} />
                                                </dd>
                                            </div>
                                            <div>
                                                <dt>Full-match corners</dt>
                                                <dd>{formatPair(item.homeCornersFull, item.awayCornersFull)}</dd>
                                            </div>
                                        </dl>
                                    </article>
                                )}
                            />
                        </section>

                        <Typography.Paragraph className='world-cup-corners-footnote' type='secondary'>
                            Method: 90-minute corners count StatsBomb corner events in periods 1–2; full-match corners also include periods 3–4. Penalty shootout results are shown
                            separately from the score after extra time.
                        </Typography.Paragraph>
                    </>
                )}
            </AppPage>
        </div>
    );
};
