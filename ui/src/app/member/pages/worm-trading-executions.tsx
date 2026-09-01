import {
    ArrowLeftOutlined,
    CheckCircleOutlined,
    ClockCircleOutlined,
    ExclamationCircleOutlined,
    PauseCircleOutlined,
    PlayCircleOutlined,
    ReloadOutlined,
    SafetyCertificateOutlined,
    StopOutlined
} from '@ant-design/icons';
import {Alert, Button, Card, Descriptions, Empty, Progress, Space, Tag, Tooltip, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {useNavigate, useParams} from 'react-router-dom';
import {AppPage, ResourceTable, useAsyncData} from '../../components';
import {AccountDataModule} from '../../shared/access-modules';
import {Context, useAuthorization} from '../../shared/context';
import {formatBeijingUnixSeconds} from '../../shared/format';
import {AccountIdentityProvider} from '../../shared/models';
import {memberServices as services} from '../services';
import {
    AbortableWormTradingPromise,
    WormExecutionAllowedAction,
    WormExecutionCommandResult,
    WormExecutionRun,
    WormExecutionRunState,
    WormExecutionRunStep,
    WormExecutionStepState
} from '../../shared/services/worm-trading-service';
import {requestErrorDetails, requestErrorMessage} from '../../shared/services/requests';
import {short, usePagedParams} from '../../shared/pages/shared';
import {wormExecutionMandatoryGuardDefinitions} from './worm-execution-preflight';

const runPageSizes = [20, 50, 100];
const stepPageSizes = [20, 50, 100];
const detailPollIntervalMS = 1_500;
const heartbeatIntervalMS = 10_000;
const terminalRunStates = new Set<WormExecutionRunState>(['COMPLETED', 'TERMINATED', 'FAILED']);
const activeStepStates = new Set<WormExecutionStepState>(['PREFLIGHTING', 'OPENING', 'OPENED', 'SIGNING', 'FINALIZING', 'AWAITING_COMPLETION']);
const acronyms: Record<string, string> = {api: 'API', hmac: 'HMAC', http: 'HTTP', id: 'ID', jwt: 'JWT', rpc: 'RPC', sol: 'SOL', usdc: 'USDC'};

const displayCode = (value: string, fallback = '—') =>
    value
        .trim()
        .toLowerCase()
        .split(/[_-]+/)
        .filter(Boolean)
        .map(part => acronyms[part] || `${part.slice(0, 1).toUpperCase()}${part.slice(1)}`)
        .join(' ') || fallback;

const runStatusColor = (state: WormExecutionRunState) => {
    switch (state) {
        case 'COMPLETED':
            return 'success';
        case 'FAILED':
        case 'RECONCILIATION_REQUIRED':
            return 'error';
        case 'AUTHORIZED':
        case 'RUNNING':
            return 'processing';
        case 'PAUSE_REQUESTED':
        case 'PAUSED':
        case 'TERMINATE_REQUESTED':
        case 'AWAITING_AUTHORIZATION':
            return 'warning';
        default:
            return 'default';
    }
};

const stepStatusColor = (state: WormExecutionStepState) => {
    switch (state) {
        case 'COMPLETED':
        case 'SATISFIED':
            return 'success';
        case 'FAILED':
        case 'OUTCOME_UNKNOWN':
            return 'error';
        case 'SKIPPED':
        case 'NOT_EXECUTED':
            return 'warning';
        case 'PREFLIGHTING':
        case 'OPENING':
        case 'OPENED':
        case 'SIGNING':
        case 'FINALIZING':
        case 'AWAITING_COMPLETION':
            return 'processing';
        default:
            return 'default';
    }
};

const RunStatusTag = ({state}: {state: WormExecutionRunState}) => <Tag color={runStatusColor(state)}>{displayCode(state)}</Tag>;
const stepStatusLabel = (step: WormExecutionRunStep) =>
    step.state === 'COMPLETED' && step.completionSource === 'OPEN_POSITION' ? 'Completed · Open position observed' : displayCode(step.state);
const StepStatusTag = ({step}: {step: WormExecutionRunStep}) => <Tag color={stepStatusColor(step.state)}>{stepStatusLabel(step)}</Tag>;
const runProgress = (run: WormExecutionRun) => (run.counts.total > 0 ? Math.round((run.counts.terminal / run.counts.total) * 100) : 0);
const sideLabel = (step: WormExecutionRunStep) => step.side;

interface ExecutionRunIntent {
    accountId: string;
    id: string;
    planId: string;
    combinationId: string;
    combinationRevision: number;
}

interface ExecutionDriverInterruption {
    reason: string;
    authoritativeStateReloaded: boolean;
}

const executionRunIntent = (run: WormExecutionRun, accountId: string): ExecutionRunIntent => ({
    accountId,
    id: run.id,
    planId: run.planId,
    combinationId: run.combinationId,
    combinationRevision: run.combinationRevision
});

const executionRunMatchesIntent = (run: WormExecutionRun, intent: ExecutionRunIntent) =>
    run.id === intent.id && run.planId === intent.planId && run.combinationId === intent.combinationId && run.combinationRevision === intent.combinationRevision;

const RunListCard = ({run, onOpen}: {run: WormExecutionRun; onOpen: () => void}) => (
    <Card size='small' className='worm-execution-list-card'>
        <div className='worm-execution-list-card__heading'>
            <div>
                <Typography.Text strong={true}>{run.combinationName}</Typography.Text>
                <Typography.Text type='secondary'>Run {short(run.id, 8, 6)}</Typography.Text>
            </div>
            <RunStatusTag state={run.state} />
        </div>
        <Progress percent={runProgress(run)} size='small' status={run.state === 'FAILED' || run.state === 'RECONCILIATION_REQUIRED' ? 'exception' : 'normal'} />
        <Typography.Text type='secondary'>
            {run.counts.terminal} of {run.counts.total} steps terminal · {run.counts.completed} completed
        </Typography.Text>
        <Button block={true} onClick={onOpen}>
            View execution
        </Button>
    </Card>
);

const ExecutionEmptyState = (props: {accountID: string; accessRevision: number; canWrite: boolean}) => {
    const navigate = useNavigate();
    const combinations = useAsyncData(() => services.wormTrading.listMarketCombinations(1, 1), [props.accountID, props.accessRevision]);
    const hasCombinations = (combinations.data?.total || 0) > 0;

    let guidance: React.ReactNode;
    let actions: React.ReactNode;
    if (combinations.loading) {
        guidance = <Typography.Text type='secondary'>Checking saved combinations…</Typography.Text>;
    } else if (combinations.error) {
        guidance = (
            <Space orientation='vertical' size='small' align='center'>
                <Typography.Text type='warning'>
                    <ExclamationCircleOutlined /> Could not check saved combinations.
                </Typography.Text>
                <Typography.Text type='secondary'>Execution runs appear only after a usable preview is prepared for live execution.</Typography.Text>
            </Space>
        );
        actions = (
            <Space wrap={true}>
                <Button icon={<ReloadOutlined />} onClick={combinations.reload}>
                    Retry
                </Button>
                <Button onClick={() => navigate('/worm-trading/combinations')}>Open combinations</Button>
            </Space>
        );
    } else if (hasCombinations && props.canWrite) {
        guidance = <Typography.Paragraph type='secondary'>Saved combinations are templates. Choose one, build a preview, then prepare it for live execution.</Typography.Paragraph>;
        actions = (
            <Button type='primary' onClick={() => navigate('/worm-trading/combinations')}>
                Choose a saved combination
            </Button>
        );
    } else if (props.canWrite) {
        guidance = <Typography.Paragraph type='secondary'>Create a combination, build an actionable preview, then prepare it for live execution.</Typography.Paragraph>;
        actions = (
            <Button type='primary' onClick={() => navigate('/worm-trading/combinations/new')}>
                Create combination
            </Button>
        );
    } else if (hasCombinations) {
        guidance = (
            <Typography.Paragraph type='secondary'>You can review saved combinations, but preparing a live execution requires Worm Trading write access.</Typography.Paragraph>
        );
        actions = <Button onClick={() => navigate('/worm-trading/combinations')}>View saved combinations</Button>;
    } else {
        guidance = (
            <Typography.Paragraph type='secondary'>
                No execution runs are available. Creating combinations and preparing live executions requires Worm Trading write access.
            </Typography.Paragraph>
        );
    }

    return (
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='No execution runs yet'>
            <Space orientation='vertical' size='middle' align='center'>
                <div aria-live='polite' aria-atomic='true' aria-busy={combinations.loading}>
                    {guidance}
                </div>
                {actions}
            </Space>
        </Empty>
    );
};

export const WormTradingExecutionsPage = () => {
    const authorization = useAuthorization();
    const canWrite = authorization.canWrite(AccountDataModule.WormTrading);
    const navigate = useNavigate();
    useExecutionAuthorizationReturn();
    const {page, pageSize, setPage} = usePagedParams(20, runPageSizes);
    const data = useAsyncData(() => services.wormTrading.listExecutionRuns(page, pageSize), [authorization.user.accountId, authorization.revision, page, pageSize]);
    const items = data.data?.items || [];
    const columns: ColumnsType<WormExecutionRun> = [
        {
            title: 'Combination',
            render: run => (
                <Button type='link' className='worm-execution-name-link' onClick={() => navigate(`/worm-trading/executions/${encodeURIComponent(run.id)}`)}>
                    {run.combinationName}
                </Button>
            )
        },
        {title: 'Status', width: 190, render: run => <RunStatusTag state={run.state} />},
        {
            title: 'Progress',
            width: 230,
            render: run => (
                <div className='worm-execution-progress-cell'>
                    <Progress percent={runProgress(run)} size='small' showInfo={false} />
                    <Typography.Text type='secondary'>
                        {run.counts.terminal}/{run.counts.total}
                    </Typography.Text>
                </div>
            )
        },
        {title: 'Completed', width: 110, render: run => run.counts.completed},
        {title: 'Updated', width: 190, render: run => formatBeijingUnixSeconds(run.updatedAt) || '—'},
        {
            title: 'Action',
            width: 140,
            render: run => <Button onClick={() => navigate(`/worm-trading/executions/${encodeURIComponent(run.id)}`)}>View</Button>
        }
    ];
    const empty = !data.loading && !data.error && (data.data?.total || 0) === 0;
    return (
        <AppPage
            title='Worm Trading Executions'
            subtitle='Review live execution runs prepared from actionable previews. Runs are permanent and cannot be deleted.'
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}>
            {empty ? (
                <ExecutionEmptyState
                    key={`${authorization.user.accountId}:${authorization.revision}`}
                    accountID={authorization.user.accountId}
                    accessRevision={authorization.revision}
                    canWrite={canWrite}
                />
            ) : (
                <ResourceTable<WormExecutionRun>
                    rowKey='id'
                    label='Worm execution history'
                    items={items}
                    columns={columns}
                    loading={data.loading}
                    total={data.data?.total}
                    page={page}
                    pageSize={pageSize}
                    pageSizeOptions={runPageSizes}
                    onPageChange={setPage}
                    scrollX={1050}
                    compactEmptyDescription='No executions on this page'
                    compactRender={run => <RunListCard run={run} onOpen={() => navigate(`/worm-trading/executions/${encodeURIComponent(run.id)}`)} />}
                />
            )}
        </AppPage>
    );
};

type PhantomProvider = {
    isPhantom?: boolean;
    publicKey?: {toString: () => string};
    connect: () => Promise<{publicKey: {toString: () => string}}>;
    signMessage: (message: Uint8Array, encoding: 'utf8') => Promise<{signature: Uint8Array}>;
};

const phantomProvider = () => (window as Window & {phantom?: {solana?: PhantomProvider}}).phantom?.solana;
const rawBase64URL = (bytes: Uint8Array) => {
    let binary = '';
    for (const byte of bytes) {
        binary += String.fromCharCode(byte);
    }
    return window.btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '');
};
const newCommandID = () => window.crypto.randomUUID();
const wait = (milliseconds: number) => new Promise(resolve => window.setTimeout(resolve, milliseconds));

const useExecutionAuthorizationReturn = (expectedRunID = '') => {
    const ctx = React.useContext(Context);
    React.useEffect(() => {
        let storedRunID = '';
        try {
            const stored = window.sessionStorage.getItem('athena.member.worm-execution.authorization');
            if (stored) {
                storedRunID = String((JSON.parse(stored) as {runId?: unknown}).runId || '');
            }
        } catch {
            // Invalid tab-local state is discarded when this is an authorization return.
        }
        const url = new URL(window.location.href);
        const reason = url.searchParams.get('wormExecutionReason') || '';
        if (!reason && (!expectedRunID || storedRunID !== expectedRunID)) return;
        window.sessionStorage.removeItem('athena.member.worm-execution.authorization');
        if (reason) {
            ctx.notifications.error('Could not authorize execution', displayCode(reason));
            url.searchParams.delete('wormExecutionReason');
            window.history.replaceState(window.history.state, '', `${url.pathname}${url.search}${url.hash}`);
        }
    }, [ctx.notifications, expectedRunID]);
};

const CurrentStepPanel = ({step}: {step?: WormExecutionRunStep}) => {
    if (!step) {
        return null;
    }
    return (
        <Card size='small' className='worm-execution-current-card' title='Current step'>
            <div className='worm-execution-current-card__headline'>
                <div>
                    <Typography.Text strong={true}>Step {step.ordinal}</Typography.Text>
                    <Typography.Text type='secondary'>{step.wallet.remark || short(step.wallet.address, 8, 6)}</Typography.Text>
                </div>
                <StepStatusTag step={step} />
            </div>
            <Typography.Title level={4}>{step.market.marketTitle}</Typography.Title>
            <Space wrap={true}>
                <Tag color={sideLabel(step) === 'YES' ? 'green' : 'red'}>{sideLabel(step)}</Tag>
                <Tag>{step.funds} USDC</Tag>
                <Tag>{step.leverage}×</Tag>
            </Space>
            <Descriptions size='small' column={{xs: 1, sm: 2, lg: 4}}>
                <Descriptions.Item label='Worm request ID'>{step.positionRequestId || 'Not assigned'}</Descriptions.Item>
                <Descriptions.Item label='Provider state'>{displayCode(step.providerState, 'Not observed')}</Descriptions.Item>
                <Descriptions.Item label='Order state'>{displayCode(step.providerOrderState, 'Not observed')}</Descriptions.Item>
                <Descriptions.Item label='Updated'>{formatBeijingUnixSeconds(step.updatedAt) || '—'}</Descriptions.Item>
            </Descriptions>
        </Card>
    );
};

const StepCard = ({step}: {step: WormExecutionRunStep}) => (
    <Card size='small' className='worm-execution-step-card'>
        <div className='worm-execution-step-card__heading'>
            <Typography.Text strong={true}>Step {step.ordinal}</Typography.Text>
            <StepStatusTag step={step} />
        </div>
        <Typography.Text>{step.market.marketTitle}</Typography.Text>
        <Typography.Text type='secondary'>{step.wallet.remark || short(step.wallet.address, 8, 6)}</Typography.Text>
        <Space wrap={true}>
            <Tag color={sideLabel(step) === 'YES' ? 'green' : 'red'}>{sideLabel(step)}</Tag>
            <Tag>{step.funds} USDC</Tag>
        </Space>
        {(step.reasonCode || step.providerState) && (
            <Typography.Text type='secondary'>{step.reasonCode ? displayCode(step.reasonCode) : `Provider: ${displayCode(step.providerState)}`}</Typography.Text>
        )}
    </Card>
);

const ExecutionSteps = ({run}: {run: WormExecutionRun}) => {
    const {page, pageSize, setPage} = usePagedParams(50, stepPageSizes);
    const data = useAsyncData(() => services.wormTrading.listExecutionRunSteps(run.id, page, pageSize), [run.id, run.updatedAt, page, pageSize]);
    const columns: ColumnsType<WormExecutionRunStep> = [
        {title: '#', dataIndex: 'ordinal', width: 70},
        {title: 'Wallet', width: 190, render: step => step.wallet.remark || short(step.wallet.address, 8, 6)},
        {title: 'Market', render: step => step.market.marketTitle},
        {title: 'Side', width: 90, render: step => <Tag color={sideLabel(step) === 'YES' ? 'green' : 'red'}>{sideLabel(step)}</Tag>},
        {title: 'Funds', width: 120, render: step => `${step.funds} USDC`},
        {title: 'State', width: 190, render: step => <StepStatusTag step={step} />},
        {title: 'Reason', width: 220, render: step => displayCode(step.reasonCode)},
        {title: 'Worm request ID', width: 170, render: step => step.positionRequestId || '—'}
    ];
    return (
        <Card
            className='worm-execution-steps-card'
            title='Steps'
            extra={
                <Button size='small' icon={<ReloadOutlined />} loading={data.loading} onClick={data.reload}>
                    Refresh
                </Button>
            }>
            {data.error && (
                <Alert type='warning' showIcon={true} title='Could not refresh steps' description={requestErrorMessage(data.error, 'The last step page remains visible.')} />
            )}
            <ResourceTable<WormExecutionRunStep>
                rowKey='ordinal'
                label='Worm execution steps'
                items={data.data?.items || []}
                columns={columns}
                loading={data.loading}
                total={data.data?.total}
                page={page}
                pageSize={pageSize}
                pageSizeOptions={stepPageSizes}
                onPageChange={setPage}
                scrollX={1250}
                compactRender={step => <StepCard step={step} />}
                compactEmptyDescription='No steps on this page'
            />
        </Card>
    );
};

const RunSummary = ({run}: {run: WormExecutionRun}) => (
    <Card className='worm-execution-summary-card'>
        <div className='worm-execution-summary-card__heading'>
            <div>
                <Typography.Text type='secondary'>Run {short(run.id, 8, 6)}</Typography.Text>
                <Typography.Title level={3}>{run.combinationName}</Typography.Title>
            </div>
            <RunStatusTag state={run.state} />
        </div>
        <Progress
            percent={runProgress(run)}
            status={run.state === 'FAILED' || run.state === 'RECONCILIATION_REQUIRED' ? 'exception' : run.state === 'COMPLETED' ? 'success' : 'normal'}
        />
        <div className='worm-execution-stat-grid'>
            <div>
                <span>Total</span>
                <strong>{run.counts.total}</strong>
            </div>
            <div>
                <span>Actionable</span>
                <strong>{run.counts.actionable}</strong>
            </div>
            <div>
                <span>Completed</span>
                <strong>{run.counts.completed}</strong>
            </div>
            <div>
                <span>Satisfied</span>
                <strong>{run.counts.satisfied}</strong>
            </div>
            <div>
                <span>Skipped</span>
                <strong>{run.counts.skipped}</strong>
            </div>
            <div>
                <span>Failed</span>
                <strong>{run.counts.failed}</strong>
            </div>
        </div>
    </Card>
);

const RunMandatoryGuards = () => (
    <div className='worm-execution-preflight-checks'>
        <div>
            <Typography.Text strong={true}>Mandatory execution guards</Typography.Text>
            <Typography.Text type='secondary'>Both guards are always applied during fresh preflight and immediately before Worm Open.</Typography.Text>
        </div>
        <div className='worm-execution-preflight-checks__tags'>
            {wormExecutionMandatoryGuardDefinitions.map(definition => (
                <Tag key={definition.key} color='blue'>
                    Always on · {definition.title}
                </Tag>
            ))}
        </div>
    </div>
);

const stateAlert = (run: WormExecutionRun) => {
    if (run.blockCode) {
        return {
            type: 'error' as const,
            title: 'Execution requires authoritative reconciliation',
            description: displayCode(run.blockCode, 'A provider mutation outcome is unknown.')
        };
    }
    if (run.state === 'FAILED') {
        return {type: 'error' as const, title: 'Execution failed', description: displayCode(run.failureCode)};
    }
    if (run.state === 'PAUSED' || run.state === 'PAUSE_REQUESTED') {
        return {type: 'warning' as const, title: displayCode(run.state), description: displayCode(run.pauseCode, 'No new step will start.')};
    }
    if (run.state === 'AWAITING_AUTHORIZATION') {
        return {type: 'warning' as const, title: 'Authorization required', description: 'Authorize this frozen run before Athena can sign or submit any Worm request.'};
    }
    if (run.state === 'COMPLETED') {
        return {type: 'success' as const, title: 'Execution completed', description: 'Every frozen step reached a terminal result.'};
    }
    return undefined;
};

const driverRecoveryAlert = (run: WormExecutionRun, interruption?: ExecutionDriverInterruption) => {
    const coordinatorInactive = run.state === 'RUNNING' && run.coordinator.state !== 'ACTIVE';
    if (terminalRunStates.has(run.state) || (!interruption && !coordinatorInactive)) {
        return undefined;
    }
    const authoritativeState = interruption?.authoritativeStateReloaded
        ? 'Athena reloaded the authoritative Run state without retrying any execution command.'
        : 'Athena has not retried any execution command.';
    let stepState = 'No Step is currently active. This tab will not start a new Step.';
    const currentStepActive = Boolean(run.currentStep && activeStepStates.has(run.currentStep.state));
    if (run.currentStep) {
        stepState = currentStepActive
            ? `Step ${run.currentStep.ordinal} is already active. The backend will continue its safe processing until it reaches a definite result or Outcome Unknown; no following Step will start from this tab.`
            : `Step ${run.currentStep.ordinal} is in the authoritative ${displayCode(run.currentStep.state)} state. No following Step will start from this tab.`;
    }
    let recovery = '';
    if (run.allowedActions.includes('PAUSE')) {
        recovery = currentStepActive
            ? ' Select Pause and review, wait until the Run reaches Paused, then explicitly Continue when you are ready.'
            : ' Select Pause and review, then explicitly Continue when you are ready.';
    }
    return {
        title: interruption ? 'Execution driver stopped' : 'Execution coordinator is inactive',
        description: `${interruption?.reason ? `${interruption.reason} ` : ''}${authoritativeState} ${stepState}${recovery}`
    };
};

export const WormTradingExecutionDetailPage = () => {
    const ctx = React.useContext(Context);
    const authorization = useAuthorization();
    const canWrite = authorization.canWrite(AccountDataModule.WormTrading);
    const navigate = useNavigate();
    const {id = ''} = useParams();
    useExecutionAuthorizationReturn(id);
    const [run, setRun] = React.useState<WormExecutionRun>();
    const [loading, setLoading] = React.useState(true);
    const [error, setError] = React.useState<Error>();
    const [operation, setOperation] = React.useState('');
    const [driverActive, setDriverActive] = React.useState(false);
    const [driverInterruption, setDriverInterruption] = React.useState<ExecutionDriverInterruption>();
    const driverActiveRef = React.useRef(false);
    const driverPromiseRef = React.useRef<Promise<void> | undefined>(undefined);
    const driverRequestRef = React.useRef<AbortableWormTradingPromise<unknown> | undefined>(undefined);
    const epochRef = React.useRef(0);
    const accountRef = React.useRef(authorization.user.accountId);
    const runRef = React.useRef<WormExecutionRun>();
    const runIntentRef = React.useRef<ExecutionRunIntent>();
    accountRef.current = authorization.user.accountId;
    runRef.current = run;

    const publishRun = React.useCallback((next: WormExecutionRun) => {
        const intent = runIntentRef.current;
        if (intent) {
            if (intent.accountId !== accountRef.current || !executionRunMatchesIntent(next, intent)) {
                throw new Error('Worm Trading changed immutable execution inputs. Existing data was not replaced.');
            }
        } else {
            runIntentRef.current = executionRunIntent(next, accountRef.current);
        }
        const current = runRef.current;
        if (current?.id === next.id && current.revision > next.revision) {
            return current;
        }
        setRun(next);
        runRef.current = next;
        setError(undefined);
        return next;
    }, []);

    const fetchRun = React.useCallback(() => services.wormTrading.getExecutionRun(id), [id]);

    const loadRun = React.useCallback(async () => {
        const requestEpoch = epochRef.current;
        const requestAccountID = accountRef.current;
        const next = await fetchRun();
        if (requestEpoch === epochRef.current && requestAccountID === accountRef.current) {
            publishRun(next);
        }
        return next;
    }, [fetchRun, publishRun]);

    React.useEffect(() => {
        epochRef.current += 1;
        driverActiveRef.current = false;
        setDriverActive(false);
        setDriverInterruption(undefined);
        if (runIntentRef.current?.accountId !== authorization.user.accountId || runIntentRef.current?.id !== id) {
            runIntentRef.current = undefined;
        }
        runRef.current = undefined;
        setRun(undefined);
        setLoading(true);
        setError(undefined);
        let active = true;
        let timer: number | undefined;
        let request: ReturnType<typeof services.wormTrading.getExecutionRun> | undefined;
        const poll = async () => {
            if (driverActiveRef.current) {
                timer = window.setTimeout(poll, detailPollIntervalMS);
                return;
            }
            request = services.wormTrading.getExecutionRun(id);
            try {
                const next = await request;
                if (!active) return;
                publishRun(next);
                setLoading(false);
                if (!terminalRunStates.has(next.state) || next.blockCode !== '') {
                    timer = window.setTimeout(poll, detailPollIntervalMS);
                }
            } catch (reason) {
                if (!active) return;
                setError(reason instanceof Error ? reason : new Error(String(reason)));
                setLoading(false);
            }
        };
        void poll();
        return () => {
            active = false;
            epochRef.current += 1;
            driverRequestRef.current?.abort?.();
            request?.abort?.();
            if (timer !== undefined) window.clearTimeout(timer);
        };
    }, [authorization.user.accountId, authorization.revision, id, publishRun]);

    const stopDriver = React.useCallback(() => {
        epochRef.current += 1;
        driverRequestRef.current?.abort?.();
        driverActiveRef.current = false;
        setDriverActive(false);
    }, []);

    const awaitDriverRequest = React.useCallback(async <T,>(request: AbortableWormTradingPromise<T>): Promise<T> => {
        driverRequestRef.current = request;
        try {
            return await request;
        } finally {
            if (driverRequestRef.current === request) driverRequestRef.current = undefined;
        }
    }, []);

    const waitForDriverToSettle = React.useCallback(async () => {
        const pending = driverPromiseRef.current;
        if (pending) await pending;
    }, []);

    const drive = React.useCallback(
        async (initial: WormExecutionCommandResult) => {
            const epoch = ++epochRef.current;
            let coordinatorToken = initial.coordinatorToken;
            let current = initial.run;
            let lastHeartbeat = Date.now();
            try {
                current = publishRun(initial.run);
                driverActiveRef.current = true;
                setDriverActive(true);
                if (!coordinatorToken) {
                    throw new Error('Worm Trading did not return an execution coordinator token.');
                }
                while (epochRef.current === epoch && accountRef.current === authorization.user.accountId) {
                    if (terminalRunStates.has(current.state) || current.state !== 'RUNNING') {
                        break;
                    }
                    if (Date.now() - lastHeartbeat >= heartbeatIntervalMS) {
                        const heartbeat = await awaitDriverRequest(
                            services.wormTrading.heartbeatExecutionRun(current.id, {
                                commandId: newCommandID(),
                                expectedRevision: current.revision,
                                coordinatorToken
                            })
                        );
                        if (epochRef.current !== epoch) break;
                        current = publishRun(heartbeat.run);
                        coordinatorToken = heartbeat.coordinatorToken || coordinatorToken;
                        lastHeartbeat = Date.now();
                        continue;
                    }
                    if (current.allowedActions.includes('EXECUTE_NEXT')) {
                        if (current.nextStepOrdinal <= 0) {
                            throw new Error('Worm Trading did not identify the next frozen step.');
                        }
                        const advanced = await awaitDriverRequest(
                            services.wormTrading.executeNextExecutionStep(current.id, {
                                commandId: newCommandID(),
                                expectedRevision: current.revision,
                                expectedStepOrdinal: current.nextStepOrdinal,
                                coordinatorToken
                            })
                        );
                        if (epochRef.current !== epoch) break;
                        current = publishRun(advanced.run);
                        coordinatorToken = advanced.coordinatorToken || coordinatorToken;
                    }
                    await wait(detailPollIntervalMS);
                    if (epochRef.current !== epoch) break;
                    const next = await awaitDriverRequest(fetchRun());
                    if (epochRef.current !== epoch || accountRef.current !== authorization.user.accountId) break;
                    current = publishRun(next);
                }
            } catch (reason) {
                if (epochRef.current === epoch) {
                    const details = requestErrorDetails(reason);
                    const interruptionReason =
                        details.status === 409
                            ? 'The execution revision changed. Review the authoritative state before continuing.'
                            : requestErrorMessage(reason, 'The execution driver could not continue.');
                    let authoritativeStateReloaded = false;
                    try {
                        await loadRun();
                        authoritativeStateReloaded = true;
                    } catch {
                        // The primary error is already visible and no mutation is retried.
                    }
                    if (epochRef.current === epoch && accountRef.current === authorization.user.accountId) {
                        setDriverInterruption({reason: interruptionReason, authoritativeStateReloaded});
                        ctx.notifications.error(
                            'Execution driver stopped',
                            authoritativeStateReloaded
                                ? `${interruptionReason} Athena reloaded the authoritative Run state without retrying any execution command.`
                                : `${interruptionReason} Athena did not retry any execution command; refresh the authoritative Run state before continuing.`
                        );
                    }
                }
            } finally {
                if (epochRef.current === epoch) {
                    driverActiveRef.current = false;
                    setDriverActive(false);
                }
            }
        },
        [authorization.user.accountId, awaitDriverRequest, ctx.notifications, fetchRun, loadRun, publishRun]
    );

    const runCommand = async (action: 'start' | 'continue' | 'pause' | 'terminate') => {
        let current = runRef.current;
        if (!current || operation) return;
        setOperation(action);
        try {
            if (action === 'pause' || action === 'terminate') {
                stopDriver();
                await waitForDriverToSettle();
                current = await fetchRun();
                publishRun(current);
            }
            const command = {commandId: newCommandID(), expectedRevision: current.revision};
            let result: WormExecutionCommandResult;
            switch (action) {
                case 'start':
                    result = await services.wormTrading.startExecutionRun(current.id, command);
                    break;
                case 'continue':
                    result = await services.wormTrading.continueExecutionRun(current.id, command);
                    break;
                case 'pause':
                    result = await services.wormTrading.pauseExecutionRun(current.id, command);
                    break;
                default:
                    result = await services.wormTrading.terminateExecutionRun(current.id, command);
            }
            publishRun(result.run);
            setDriverInterruption(undefined);
            if (action === 'start' || action === 'continue') {
                const pending = drive(result);
                driverPromiseRef.current = pending;
                void pending.then(
                    () => {
                        if (driverPromiseRef.current === pending) driverPromiseRef.current = undefined;
                    },
                    () => {
                        if (driverPromiseRef.current === pending) driverPromiseRef.current = undefined;
                    }
                );
            }
        } catch (reason) {
            ctx.notifications.error(`Could not ${action} execution`, requestErrorMessage(reason, 'The execution state was not changed.'));
            try {
                await loadRun();
            } catch {
                // Keep the command error visible.
            }
        } finally {
            setOperation('');
        }
    };

    const refreshRun = async () => {
        stopDriver();
        await waitForDriverToSettle();
        await loadRun();
        setDriverInterruption(undefined);
    };

    const authorize = async () => {
        const current = runRef.current;
        if (!current || operation) return;
        const command = {commandId: newCommandID(), expectedRevision: current.revision};
        const identity = authorization.user.identity;
        if (identity.provider === AccountIdentityProvider.Google) {
            window.sessionStorage.setItem('athena.member.worm-execution.authorization', JSON.stringify({runId: current.id}));
            window.location.assign(services.wormTrading.googleExecutionAuthorizationURL(current.id, command, `/worm-trading/executions/${current.id}`));
            return;
        }
        setOperation('authorize');
        try {
            let next: WormExecutionRun;
            if (identity.provider === AccountIdentityProvider.SolanaWallet) {
                const provider = phantomProvider();
                if (!provider?.isPhantom) throw new Error('Phantom is required to authorize this execution.');
                const connected = provider.publicKey ? {publicKey: provider.publicKey} : await provider.connect();
                if (connected.publicKey.toString() !== identity.solanaAddress) throw new Error('Phantom is connected to a different login address.');
                const challenge = await services.wormTrading.createSolanaExecutionAuthorizationChallenge(current.id, command);
                const signed = await provider.signMessage(new TextEncoder().encode(challenge.message), 'utf8');
                next = await services.wormTrading.verifySolanaExecutionAuthorization(current.id, rawBase64URL(signed.signature));
            } else {
                next = await services.wormTrading.authorizeDevelopmentExecutionRun(current.id, command);
            }
            publishRun(next);
            setDriverInterruption(undefined);
            ctx.notifications.success('Execution authorized', 'Review the frozen run, then start it explicitly.');
        } catch (reason) {
            ctx.notifications.error('Could not authorize execution', requestErrorMessage(reason, 'No execution authorization was recorded.'));
            try {
                await loadRun();
            } catch {
                // Keep the authorization error visible.
            }
        } finally {
            setOperation('');
        }
    };

    const confirmAuthorize = () => {
        const current = runRef.current;
        if (!current) return;
        ctx.modal.confirm({
            title: 'Authorize this live Worm execution?',
            content: (
                <div className='worm-execution-authorization-copy'>
                    <p>Athena will ask each selected custodial Wallet to sign the exact Solana transaction returned by Worm for this frozen run.</p>
                    <p>
                        The frozen request limits the funds value sent to Worm to at most 10 USDC per step. Athena does not inspect Worm&apos;s programs, accounts, instructions, or
                        cryptographically prove the transaction&apos;s actual chain spend.
                    </p>
                    <p>Every Open is a market order at fixed 1× leverage.</p>
                    <p>
                        Immediately before Open, Athena always checks for any-direction Open Position in the target market and any remaining uncovered wallet-wide in-flight market
                        or limit request across all markets and directions. Either match skips this order and every remaining order for that Wallet.
                    </p>
                </div>
            ),
            okText: 'Authorize frozen run',
            onOk: authorize
        });
    };

    const reconcile = async () => {
        const current = runRef.current;
        const step = current?.currentStep;
        if (!current || !step || operation) return;
        setOperation('reconcile');
        try {
            const result = await services.wormTrading.reconcileExecutionStep(current.id, step.id, {
                commandId: newCommandID(),
                expectedRevision: current.revision
            });
            publishRun(result.run);
            setDriverInterruption(undefined);
        } catch (reason) {
            ctx.notifications.error('Could not check authoritative status', requestErrorMessage(reason, 'No Worm mutation was replayed.'));
        } finally {
            setOperation('');
        }
    };

    const primaryAction = (current: WormExecutionRun, recoveryRequired: boolean) => {
        const has = (action: WormExecutionAllowedAction) => current.allowedActions.includes(action);
        if (has('AUTHORIZE'))
            return (
                <Button type='primary' icon={<SafetyCertificateOutlined />} loading={operation === 'authorize'} onClick={confirmAuthorize}>
                    Authorize
                </Button>
            );
        if (has('START'))
            return (
                <Button type='primary' icon={<PlayCircleOutlined />} loading={operation === 'start'} onClick={() => void runCommand('start')}>
                    Start
                </Button>
            );
        if (has('PAUSE'))
            return (
                <Button type='primary' icon={<PauseCircleOutlined />} loading={operation === 'pause'} onClick={() => void runCommand('pause')}>
                    {recoveryRequired ? 'Pause and review' : 'Pause'}
                </Button>
            );
        if (has('CONTINUE'))
            return (
                <Button type='primary' icon={<PlayCircleOutlined />} loading={operation === 'continue'} onClick={() => void runCommand('continue')}>
                    Continue
                </Button>
            );
        return null;
    };

    const alert = run ? stateAlert(run) : undefined;
    const recoveryAlert = run ? driverRecoveryAlert(run, driverInterruption) : undefined;
    const recoveryRequired = Boolean(recoveryAlert);
    return (
        <AppPage
            title='Worm Trading Execution'
            subtitle='One Wallet × Market step runs at a time. Refreshing or leaving this page never starts the next step.'
            loading={loading}
            error={error}
            onRefresh={() => {
                void refreshRun();
            }}
            extra={
                <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/worm-trading/executions')}>
                    Executions
                </Button>
            }>
            {run && (
                <div className='worm-execution-detail'>
                    <div className='worm-execution-live-region' aria-live='polite'>
                        {driverActive
                            ? `Execution driver active. ${run.counts.terminal} of ${run.counts.total} steps are terminal.`
                            : `${displayCode(run.state)}. ${run.counts.terminal} of ${run.counts.total} steps are terminal.`}
                    </div>
                    {alert && <Alert type={alert.type} showIcon={true} title={alert.title} description={alert.description} />}
                    {recoveryAlert && <Alert type='warning' showIcon={true} title={recoveryAlert.title} description={recoveryAlert.description} />}
                    <RunSummary run={run} />
                    <Card className='worm-execution-control-card' title='Authorization and control'>
                        <Alert
                            type='warning'
                            showIcon={true}
                            title='Worm transaction trust boundary'
                            description='Athena signs the exact Solana transaction returned by Worm. The frozen funds request is capped at 10 USDC per step, but the signer does not inspect programs, accounts, instructions, or independently prove the actual chain spend.'
                        />
                        <RunMandatoryGuards />
                        <Descriptions size='small' column={{xs: 1, sm: 2, lg: 3}}>
                            <Descriptions.Item label='Authorization'>
                                {run.authorization.requiresReauthorization ? 'Fresh proof required' : displayCode(run.authorization.state)}
                            </Descriptions.Item>
                            <Descriptions.Item label='Proof'>{displayCode(run.authorization.proofKind)}</Descriptions.Item>
                            <Descriptions.Item label='Coordinator'>{driverActive ? 'This tab is driving' : displayCode(run.coordinator.state, 'Inactive')}</Descriptions.Item>
                        </Descriptions>
                    </Card>
                    {run.blockCode && run.allowedActions.includes('RECONCILE') && (
                        <Alert
                            className='worm-execution-unknown-alert'
                            type='error'
                            showIcon={true}
                            icon={<ExclamationCircleOutlined />}
                            title='Mutation outcome unknown'
                            description='This Wallet and market remain isolated. Athena will only perform authoritative read checks; Open, Finalize, and cancel are never replayed automatically.'
                            action={
                                <Button icon={<ReloadOutlined />} loading={operation === 'reconcile'} onClick={() => void reconcile()}>
                                    Check authoritative status
                                </Button>
                            }
                        />
                    )}
                    <CurrentStepPanel step={run.currentStep} />
                    <ExecutionSteps run={run} />
                    <div className='worm-execution-action-spacer' aria-hidden='true' />
                    {canWrite && (
                        <div className='worm-execution-sticky-actions'>
                            <div className='worm-execution-sticky-actions__state'>
                                {driverActive ? <ClockCircleOutlined spin={true} /> : run.state === 'COMPLETED' ? <CheckCircleOutlined /> : null}
                                <span>{driverActive ? 'Waiting for the current authoritative step result' : displayCode(run.state)}</span>
                            </div>
                            <Space>
                                {run.allowedActions.includes('TERMINATE') && (
                                    <Tooltip title='Does not cancel an already submitted Worm request'>
                                        <Button
                                            danger={true}
                                            icon={<StopOutlined />}
                                            disabled={Boolean(operation) && operation !== 'terminate'}
                                            loading={operation === 'terminate'}
                                            onClick={() =>
                                                ctx.modal.confirm({
                                                    title: 'Terminate this execution?',
                                                    content:
                                                        'Unstarted steps become Not executed. Any step already sent to Worm continues to an authoritative result or unknown state; Athena will not cancel it.',
                                                    okText: 'Terminate execution',
                                                    onOk: () => runCommand('terminate')
                                                })
                                            }>
                                            Terminate
                                        </Button>
                                    </Tooltip>
                                )}
                                {primaryAction(run, recoveryRequired)}
                            </Space>
                        </div>
                    )}
                </div>
            )}
        </AppPage>
    );
};
