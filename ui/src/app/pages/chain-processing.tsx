import {PlayCircleOutlined, SearchOutlined, StopOutlined} from '@ant-design/icons';
import {Button, Card, InputNumber, Select, Space, Tag, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {useSearchParams} from 'react-router-dom';
import {AppPage, ChoiceGroup, KeyValueGrid, ResourceTable, Section, StatusTag, useAsyncData} from '../components';
import {formatBeijingDateTime, formatBeijingUnixSeconds, formatBlockNumber} from '../shared/format';
import {PAGE_SIZE_OPTIONS} from '../shared/pagination';
import {services} from '../shared/services';
import {useAuthorization} from '../shared/context';
import {TokenChainCheckpoint, TokenChainProcessingAttempt, TokenChainProcessingSummary} from '../shared/services/token-service';
import {boolTag} from './shared';
import {ChainBadge} from './token-shared';

type ChainProcessingStatus = 'running' | 'stopped';
type AttemptStatus = 'running' | 'succeeded' | 'failed' | 'cancelled' | 'interrupted';

const DEFAULT_WINDOW_SECONDS = 24 * 60 * 60;
const WINDOW_OPTIONS = [
    {label: '1h', value: 60 * 60},
    {label: '24h', value: DEFAULT_WINDOW_SECONDS},
    {label: '72h', value: 72 * 60 * 60}
];
const ATTEMPT_STATUSES: AttemptStatus[] = ['running', 'succeeded', 'failed', 'cancelled', 'interrupted'];
const STAGE_ORDER = ['checkpoint_read', 'candidate_discovery', 'candidate_validation', 'persistence'] as const;

const positiveInteger = (value: string | null) => {
    const parsed = Number(value || 0);
    return Number.isSafeInteger(parsed) && parsed > 0 ? parsed : undefined;
};

const allowedWindow = (value: string | null) => {
    const parsed = Number(value || 0);
    return WINDOW_OPTIONS.some(option => option.value === parsed) ? parsed : DEFAULT_WINDOW_SECONDS;
};

const allowedPageSize = (value: string | null) => {
    const parsed = Number(value || 0);
    return PAGE_SIZE_OPTIONS.includes(parsed) ? parsed : PAGE_SIZE_OPTIONS[0];
};

const resolved = <T,>(value: T) => Promise.resolve(value) as Promise<T> & {abort?: () => void};

const formatDuration = (microseconds?: number, available = true) => {
    if (!available || microseconds === undefined) {
        return '-';
    }
    if (microseconds < 1000) {
        return `${microseconds} μs`;
    }
    if (microseconds < 1_000_000) {
        const milliseconds = microseconds / 1000;
        return `${milliseconds >= 100 ? milliseconds.toFixed(0) : milliseconds.toFixed(1)} ms`;
    }
    const seconds = microseconds / 1_000_000;
    return `${seconds >= 10 ? seconds.toFixed(1) : seconds.toFixed(2)} s`;
};

const formatInteger = (value?: number) => new Intl.NumberFormat().format(value || 0);

const attemptTone = (status?: string) => ({
    positive: status === 'succeeded',
    negative: status === 'failed' || status === 'interrupted'
});

const stageAvailable = (item: TokenChainProcessingAttempt, stage: (typeof STAGE_ORDER)[number]) => {
    if (item.status === 'succeeded' && item.timingComplete) {
        return true;
    }
    const terminalIndex = STAGE_ORDER.indexOf(item.terminalStage as (typeof STAGE_ORDER)[number]);
    return terminalIndex >= STAGE_ORDER.indexOf(stage);
};

const AttemptDetails = (props: {item: TokenChainProcessingAttempt}) => (
    <div className='chain-processing-attempt-details'>
        <KeyValueGrid
            columns={3}
            items={[
                {label: 'Started', value: formatBeijingDateTime(props.item.startedAt) || '-'},
                {label: 'Completed', value: formatBeijingDateTime(props.item.completedAt) || '-'},
                {label: 'Terminal stage', value: props.item.terminalStage || '-'},
                {label: 'Validated', value: formatInteger(props.item.validatedCount)},
                {label: 'Rejected', value: formatInteger(props.item.rejectedCount)},
                {label: 'Research expired', value: formatInteger(props.item.expiredResearchStateCount)},
                {label: 'Timing', value: props.item.timingComplete ? 'Complete' : 'Incomplete'},
                {label: 'Last updated', value: formatBeijingDateTime(props.item.updatedAt) || '-'},
                {label: 'Error', value: props.item.errorMessage || '-'}
            ]}
        />
    </div>
);

const AttemptCard = (props: {item: TokenChainProcessingAttempt}) => {
    const tone = attemptTone(props.item.status);
    return (
        <Card className='chain-processing-attempt-card' size='small'>
            <div className='chain-processing-attempt-card__header'>
                <div>
                    <Typography.Text strong={true}>Block {formatBlockNumber(props.item.blockNumber)}</Typography.Text>
                    <Typography.Text type='secondary'>Attempt {props.item.attemptNumber || '-'}</Typography.Text>
                </div>
                <StatusTag value={props.item.status} {...tone} />
            </div>
            <div className='chain-processing-attempt-card__metrics'>
                <span>
                    Checkpoint<strong>{formatDuration(props.item.checkpointReadDurationUS, stageAvailable(props.item, 'checkpoint_read'))}</strong>
                </span>
                <span>
                    Discovery<strong>{formatDuration(props.item.discoveryDurationUS, stageAvailable(props.item, 'candidate_discovery'))}</strong>
                </span>
                <span>
                    Validation<strong>{formatDuration(props.item.validationDurationUS, stageAvailable(props.item, 'candidate_validation'))}</strong>
                </span>
                <span>
                    Persistence<strong>{formatDuration(props.item.persistenceDurationUS, stageAvailable(props.item, 'persistence'))}</strong>
                </span>
            </div>
            <div className='chain-processing-attempt-card__footer'>
                <Typography.Text type='secondary'>{formatBeijingUnixSeconds(props.item.blockTime) || 'Block time unavailable'}</Typography.Text>
                <Typography.Text strong={true}>{formatDuration(props.item.totalDurationUS, Boolean(props.item.terminalStage))}</Typography.Text>
            </div>
            <details className='chain-processing-attempt-card__details'>
                <summary>Timing and error details</summary>
                <AttemptDetails item={props.item} />
            </details>
        </Card>
    );
};

const CheckpointCard = (props: {item: TokenChainCheckpoint; action: React.ReactNode}) => (
    <Card className='chain-processing-checkpoint-card' size='small'>
        <div className='chain-processing-checkpoint-card__header'>
            <ChainBadge chainID={props.item.chainID} />
            <StatusTag value={props.item.status} positive={props.item.status === 'running'} />
        </div>
        <KeyValueGrid
            columns={1}
            items={[
                {label: 'Enabled', value: boolTag(props.item.enabled)},
                {label: 'Cursor', value: formatBlockNumber(props.item.cursorBlockNumber)},
                {label: 'Updated', value: formatBeijingDateTime(props.item.updatedAt) || '-'}
            ]}
        />
        <div className='chain-processing-checkpoint-card__actions'>{props.action}</div>
    </Card>
);

const Kpi = (props: {label: string; value: React.ReactNode; detail: React.ReactNode; onClick?: () => void}) => {
    const content = (
        <>
            <span>{props.label}</span>
            <strong>{props.value}</strong>
            <small>{props.detail}</small>
        </>
    );
    return props.onClick ? (
        <button type='button' className='chain-processing-kpi chain-processing-kpi--interactive' onClick={props.onClick}>
            {content}
        </button>
    ) : (
        <div className='chain-processing-kpi'>{content}</div>
    );
};

const StageBreakdown = (props: {summary?: TokenChainProcessingSummary}) => {
    const items = [
        {label: 'Checkpoint read', value: props.summary?.averageCheckpointReadDurationUS || 0},
        {label: 'Block discovery', value: props.summary?.averageDiscoveryDurationUS || 0},
        {label: 'Candidate validation', value: props.summary?.averageValidationDurationUS || 0},
        {label: 'Transaction persistence', value: props.summary?.averagePersistenceDurationUS || 0}
    ];
    const max = Math.max(1, ...items.map(item => item.value));
    return (
        <div className='chain-processing-stages'>
            {items.map(item => (
                <div className='chain-processing-stage' key={item.label}>
                    <span>{item.label}</span>
                    <div className='chain-processing-stage__track' aria-hidden='true'>
                        <i style={{width: `${Math.max(item.value > 0 ? 4 : 0, (item.value / max) * 100)}%`}} />
                    </div>
                    <strong>{formatDuration(item.value, Boolean(props.summary?.measuredSucceededCount))}</strong>
                </div>
            ))}
        </div>
    );
};

export const ChainProcessingPage = () => {
    const authorization = useAuthorization();
    const [params, setParams] = useSearchParams();
    const checkpoints = useAsyncData(() => services.tokenapi.listChainCheckpoints(), []);
    const requestedChainID = positiveInteger(params.get('chain'));
    const chainID = requestedChainID || checkpoints.data?.find(item => item.enabled)?.chainID || checkpoints.data?.[0]?.chainID;
    const windowSeconds = allowedWindow(params.get('window'));
    const blockNumber = positiveInteger(params.get('block'));
    const requestedStatus = params.get('status') || '';
    const status = ATTEMPT_STATUSES.includes(requestedStatus as AttemptStatus) ? requestedStatus : '';
    const page = positiveInteger(params.get('page')) || 1;
    const pageSize = allowedPageSize(params.get('pageSize') || params.get('page_size'));
    const [blockDraft, setBlockDraft] = React.useState<number | null>(blockNumber || null);
    const [updatingStatusByChainID, setUpdatingStatusByChainID] = React.useState<Record<number, ChainProcessingStatus>>({});

    React.useEffect(() => setBlockDraft(blockNumber || null), [blockNumber]);
    React.useEffect(() => {
        if (!chainID) {
            return;
        }
        const next = new URLSearchParams(params);
        let changed = false;
        if (!requestedChainID) {
            next.set('chain', String(chainID));
            changed = true;
        }
        if (!next.has('window')) {
            next.set('window', String(DEFAULT_WINDOW_SECONDS));
            changed = true;
        }
        if (changed) {
            setParams(next, {replace: true});
        }
    }, [chainID, params, requestedChainID, setParams]);

    const summary = useAsyncData(
        () => (chainID ? services.tokenapi.getChainProcessingSummary({chainID, windowSeconds, blockNumber}) : resolved<TokenChainProcessingSummary | undefined>(undefined)),
        [chainID, windowSeconds, blockNumber]
    );
    const attempts = useAsyncData(
        () =>
            chainID
                ? services.tokenapi.listChainProcessingAttempts({chainID, windowSeconds, blockNumber, status: status || undefined, page, pageSize})
                : resolved({items: [] as TokenChainProcessingAttempt[], total: 0, page: 1, pageSize}),
        [chainID, windowSeconds, blockNumber, status, page, pageSize]
    );

    const updateParams = (values: Record<string, string | number | undefined>) => {
        const next = new URLSearchParams(params);
        Object.entries(values).forEach(([key, value]) => {
            if (value === undefined || value === '') {
                next.delete(key);
            } else {
                next.set(key, String(value));
            }
        });
        setParams(next);
    };
    const applyBlock = (value = blockDraft) => updateParams({block: value || undefined, page: 1});
    const selectBlock = (value?: number) => {
        setBlockDraft(value || null);
        updateParams({block: value || undefined, page: 1});
    };
    const updateStatus = async (item: TokenChainCheckpoint, nextStatus: ChainProcessingStatus) => {
        if (!authorization.canWriteData || !item.chainID) {
            return;
        }
        const targetChainID = item.chainID;
        setUpdatingStatusByChainID(current => ({...current, [targetChainID]: nextStatus}));
        try {
            await services.tokenapi.updateChainCheckpoint(targetChainID, nextStatus);
            checkpoints.reload();
        } finally {
            setUpdatingStatusByChainID(current => {
                const next = {...current};
                delete next[targetChainID];
                return next;
            });
        }
    };
    const checkpointAction = (item: TokenChainCheckpoint) => {
        const updatingStatus = item.chainID ? updatingStatusByChainID[item.chainID] : undefined;
        return (
            <Space.Compact>
                <Button
                    size='small'
                    icon={<PlayCircleOutlined />}
                    disabled={!item.chainID || Boolean(updatingStatus) || item.status === 'running'}
                    loading={updatingStatus === 'running'}
                    aria-label={`Start chain ${item.chainID || ''} ingest`}
                    onClick={() => void updateStatus(item, 'running')}>
                    Start
                </Button>
                <Button
                    size='small'
                    danger={true}
                    icon={<StopOutlined />}
                    disabled={!item.chainID || Boolean(updatingStatus) || item.status === 'stopped'}
                    loading={updatingStatus === 'stopped'}
                    aria-label={`Stop chain ${item.chainID || ''} ingest`}
                    onClick={() => void updateStatus(item, 'stopped')}>
                    Stop
                </Button>
            </Space.Compact>
        );
    };

    const checkpointColumns: ColumnsType<TokenChainCheckpoint> = [
        {title: 'Chain', render: item => <ChainBadge chainID={item.chainID} />},
        {title: 'Enabled', render: item => boolTag(item.enabled)},
        {title: 'Cursor', render: item => formatBlockNumber(item.cursorBlockNumber)},
        {title: 'Current status', render: item => <StatusTag value={item.status} positive={item.status === 'running'} />},
        {title: 'Updated', render: item => formatBeijingDateTime(item.updatedAt) || '-'}
    ];
    if (authorization.canWriteData) {
        checkpointColumns.push({title: 'Actions', render: checkpointAction});
    }
    const attemptColumns: ColumnsType<TokenChainProcessingAttempt> = [
        {
            title: 'Block',
            render: item => (
                <Button type='link' size='small' onClick={() => selectBlock(item.blockNumber)}>
                    {formatBlockNumber(item.blockNumber)}
                </Button>
            )
        },
        {title: 'Try', dataIndex: 'attemptNumber'},
        {title: 'Status', render: item => <StatusTag value={item.status} {...attemptTone(item.status)} />},
        {title: 'Block time', render: item => formatBeijingUnixSeconds(item.blockTime) || '-'},
        {title: 'Checkpoint', render: item => formatDuration(item.checkpointReadDurationUS, stageAvailable(item, 'checkpoint_read'))},
        {title: 'Discovery', render: item => formatDuration(item.discoveryDurationUS, stageAvailable(item, 'candidate_discovery'))},
        {title: 'Validation', render: item => formatDuration(item.validationDurationUS, stageAvailable(item, 'candidate_validation'))},
        {title: 'Persistence', render: item => formatDuration(item.persistenceDurationUS, stageAvailable(item, 'persistence'))},
        {title: 'Total', render: item => formatDuration(item.totalDurationUS, Boolean(item.terminalStage))},
        {title: 'Candidates', render: item => formatInteger(item.candidateCount)}
    ];
    const summaryValue = summary.data;
    const rangeText = blockNumber
        ? `Exact block ${formatBlockNumber(blockNumber)}`
        : `${formatBeijingUnixSeconds(summaryValue?.rangeStartBlockTime) || '-'} – ${formatBeijingUnixSeconds(summaryValue?.rangeEndBlockTime) || '-'}`;
    const requestError = checkpoints.error || summary.error || attempts.error;
    const loading = checkpoints.loading || summary.loading || attempts.loading;
    const reload = () => {
        checkpoints.reload();
        summary.reload();
        attempts.reload();
    };

    return (
        <AppPage
            title='Chain Processing'
            subtitle='Block checkpoints, controls, and processing performance retained for 72 hours of chain time.'
            loading={loading}
            error={requestError}
            onRefresh={reload}
            filters={
                <Space className='chain-processing-filters' wrap={true} align='center'>
                    <Select
                        aria-label='Select chain'
                        value={chainID}
                        placeholder='Chain'
                        options={(checkpoints.data || []).map(item => ({value: item.chainID, label: item.chainName || `Chain ${item.chainID}`}))}
                        onChange={value => updateParams({chain: value, page: 1})}
                    />
                    <ChoiceGroup<number>
                        ariaLabel='Select chain-time window'
                        value={windowSeconds}
                        options={WINDOW_OPTIONS}
                        onChange={value => updateParams({window: value, block: undefined, page: 1})}
                    />
                    <Space.Compact>
                        <InputNumber
                            aria-label='Query exact block number'
                            min={1}
                            precision={0}
                            value={blockDraft}
                            placeholder='Block number'
                            onChange={value => setBlockDraft(typeof value === 'number' ? value : null)}
                            onKeyDown={event => {
                                if (event.key === 'Enter') {
                                    applyBlock();
                                }
                            }}
                        />
                        <Button aria-label='Apply exact block query' icon={<SearchOutlined />} onClick={() => applyBlock()} />
                        {blockNumber && <Button onClick={() => selectBlock(undefined)}>Clear</Button>}
                    </Space.Compact>
                    <Select
                        aria-label='Filter attempt status'
                        value={status || 'all'}
                        options={[{label: 'All statuses', value: 'all'}, ...ATTEMPT_STATUSES.map(value => ({label: value, value}))]}
                        onChange={value => updateParams({status: value === 'all' ? undefined : value, page: 1})}
                    />
                </Space>
            }>
            <Section title='Checkpoints'>
                <ResourceTable
                    rowKey={item => item.chainID || item.chainName || 'chain'}
                    items={checkpoints.data || []}
                    columns={checkpointColumns}
                    loading={checkpoints.loading}
                    compactRender={item => <CheckpointCard item={item} action={checkpointAction(item)} />}
                    label='Chain processing checkpoints'
                />
            </Section>

            <section className='chain-processing-summary' aria-label='Chain processing performance summary'>
                <div className='chain-processing-kpis'>
                    <Kpi label='Successful blocks' value={formatInteger(summaryValue?.succeededCount)} detail={`${formatInteger(summaryValue?.measuredSucceededCount)} measured`} />
                    <Kpi label='Average duration' value={formatDuration(summaryValue?.averageDurationUS, Boolean(summaryValue?.measuredSucceededCount))} detail={rangeText} />
                    <Kpi
                        label='Fastest block'
                        value={summaryValue?.fastestBlockNumber ? `#${formatBlockNumber(summaryValue.fastestBlockNumber)}` : '-'}
                        detail={formatDuration(summaryValue?.fastestDurationUS, Boolean(summaryValue?.fastestBlockNumber))}
                        onClick={summaryValue?.fastestBlockNumber ? () => selectBlock(summaryValue.fastestBlockNumber) : undefined}
                    />
                    <Kpi
                        label='Slowest block'
                        value={summaryValue?.slowestBlockNumber ? `#${formatBlockNumber(summaryValue.slowestBlockNumber)}` : '-'}
                        detail={formatDuration(summaryValue?.slowestDurationUS, Boolean(summaryValue?.slowestBlockNumber))}
                        onClick={summaryValue?.slowestBlockNumber ? () => selectBlock(summaryValue.slowestBlockNumber) : undefined}
                    />
                    <Kpi
                        label='Failure rate'
                        value={`${((summaryValue?.failureRateBPS || 0) / 100).toFixed(2)}%`}
                        detail={`${formatInteger(summaryValue?.failedCount)} failed · ${formatInteger(
                            (summaryValue?.succeededCount || 0) + (summaryValue?.failedCount || 0)
                        )} decided`}
                    />
                </div>
                <div className='chain-processing-summary__meta'>
                    <Typography.Text type='secondary'>{rangeText}</Typography.Text>
                    <Space size={[4, 4]} wrap={true}>
                        <Tag>{formatInteger(summaryValue?.runningCount)} running</Tag>
                        <Tag>{formatInteger(summaryValue?.cancelledCount)} cancelled</Tag>
                        <Tag>{formatInteger(summaryValue?.interruptedCount)} interrupted</Tag>
                        <Tag>{formatInteger(summaryValue?.incompleteSucceededCount)} incomplete timings</Tag>
                    </Space>
                </div>
            </section>

            <Section title='Average main-stage duration'>
                <StageBreakdown summary={summaryValue} />
            </Section>

            <Section title='Processing attempts'>
                <ResourceTable
                    rowKey={item => item.attemptID || `${item.chainID}-${item.blockNumber}-${item.attemptNumber}`}
                    items={attempts.data?.items || []}
                    columns={attemptColumns}
                    loading={attempts.loading}
                    total={attempts.data?.total || 0}
                    page={page}
                    pageSize={pageSize}
                    onPageChange={(nextPage, nextPageSize) => updateParams({page: nextPage, pageSize: nextPageSize})}
                    scrollX={1280}
                    stickyHeader={true}
                    compactRender={item => <AttemptCard item={item} />}
                    expandable={{expandedRowRender: item => <AttemptDetails item={item} />}}
                    label='Chain block processing attempts'
                />
            </Section>
        </AppPage>
    );
};
