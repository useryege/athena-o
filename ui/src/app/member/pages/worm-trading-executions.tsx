import {useModuleAccessLease} from '../../shared/module-access';
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
import {Alert, Button, Card, Descriptions, Empty, Progress, Skeleton, Space, Tag, Tooltip, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {useNavigate, useParams} from 'react-router-dom';
import {AppPage, ResourceTable, Section, useAsyncData} from '../../components';
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
import {usePagedParams} from '../../shared/pages/shared';
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
            return 'info';
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
            return 'info';
        default:
            return 'default';
    }
};

const RunStatusTag = ({state}: {state: WormExecutionRunState}) => <Tag className={`worm-status worm-status--${runStatusColor(state)}`}>{displayCode(state)}</Tag>;
const stepStatusLabel = (step: WormExecutionRunStep) =>
    step.state === 'COMPLETED' && step.completionSource === 'OPEN_POSITION' ? 'Completed · Open position observed' : displayCode(step.state);
const StepStatusTag = ({step}: {step: WormExecutionRunStep}) => <Tag className={`worm-status worm-status--${stepStatusColor(step.state)}`}>{stepStatusLabel(step)}</Tag>;
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
    <div className='worm-execution-list-card'>
        <Typography.Text strong>{run.combinationName}</Typography.Text>
        <code className='athena-identifier'>{run.id}</code>
        <RunStatusTag state={run.state} />
        {run.state === 'AWAITING_AUTHORIZATION' && <Typography.Text type='secondary'>Frozen · No order submitted</Typography.Text>}
        <div className='worm-execution-list-facts'>
            <div>
                <span>Step progress</span>
                <RunProgress run={run} />
            </div>
            <div>
                <span>Updated · UTC+8</span>
                <span>{formatBeijingUnixSeconds(run.updatedAt) || '—'}</span>
            </div>
        </div>
        <Button block onClick={onOpen}>
            View execution
        </Button>
    </div>
);

const RunProgress = ({run}: {run: WormExecutionRun}) => (
    <div className='worm-execution-progress-cell'>
        <span>
            {run.counts.terminal} of {run.counts.total} terminal
        </span>
        <Progress
            aria-label={`${run.counts.terminal} of ${run.counts.total} steps terminal`}
            strokeColor='var(--athena-muted)'
            percent={runProgress(run)}
            size='small'
            showInfo={false}
        />
        <Typography.Text type='secondary'>
            {run.counts.completed} completed · {run.counts.skipped} skipped
        </Typography.Text>
    </div>
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
                <Button icon={<ReloadOutlined aria-hidden='true' />} onClick={combinations.reload}>
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
            title: 'Combination / Run',
            render: run => (
                <div className='worm-execution-list-identity'>
                    <Button type='link' className='worm-execution-name-link' onClick={() => navigate(`/worm-trading/executions/${encodeURIComponent(run.id)}`)}>
                        {run.combinationName}
                    </Button>
                    <code className='athena-identifier'>{run.id}</code>
                </div>
            )
        },
        {
            title: 'Status',
            width: 220,
            render: run => (
                <div>
                    <RunStatusTag state={run.state} />
                    {run.state === 'AWAITING_AUTHORIZATION' && <p className='worm-execution-note'>Frozen · No order submitted</p>}
                </div>
            )
        },
        {title: 'Step progress', width: 230, render: run => <RunProgress run={run} />},
        {title: 'Updated · UTC+8', width: 180, render: run => formatBeijingUnixSeconds(run.updatedAt) || '—'},
        {title: 'Action', width: 100, render: run => <Button onClick={() => navigate(`/worm-trading/executions/${encodeURIComponent(run.id)}`)}>View</Button>}
    ];
    const empty = !data.loading && !data.error && (data.data?.total || 0) === 0;
    return (
        <div className='worm-execution-theme'>
            <AppPage
                title='Worm Trading Executions'
                subtitle='Frozen runs, their progress and permanent execution history.'
                loading={data.loading}
                error={data.error}
                onRefresh={data.reload}>
                {data.error && data.data && (
                    <Alert type='warning' showIcon title='Execution history is stale' description='Showing the last confirmed response. Refresh to check current state.' />
                )}
                {!data.data ? (
                    data.loading ? (
                        <Skeleton active />
                    ) : null
                ) : empty ? (
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
                        scrollX={950}
                        compactEmptyDescription='No executions on this page'
                        compactRender={run => <RunListCard run={run} onOpen={() => navigate(`/worm-trading/executions/${encodeURIComponent(run.id)}`)} />}
                    />
                )}
                <p className='worm-execution-note'>
                    Runs are prepared from actionable previews. Creating a Run freezes intent; it does not submit an order. Terminal counts include skipped, failed and not-executed
                    steps.
                </p>
            </AppPage>
        </div>
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
                    <Typography.Text type='secondary'>{step.wallet.remark || step.wallet.address}</Typography.Text>
                </div>
                <StepStatusTag step={step} />
            </div>
            <Typography.Title level={4}>{step.market.marketTitle}</Typography.Title>
            <Space wrap={true}>
                <Tag className='worm-outcome'>{sideLabel(step)}</Tag>
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

const StepEvidence = ({step}: {step: WormExecutionRunStep}) => (
    <details className='worm-step-evidence'>
        <summary>Identity and evidence · Step {step.ordinal}</summary>
        <dl>
            <div>
                <dt>Step ID</dt>
                <dd className='athena-identifier'>{step.id}</dd>
            </div>
            <div>
                <dt>Wallet address</dt>
                <dd className='athena-identifier'>{step.wallet.address}</dd>
            </div>
            <div>
                <dt>Event ID</dt>
                <dd className='athena-identifier'>{step.market.eventConditionId}</dd>
            </div>
            <div>
                <dt>Market ID</dt>
                <dd className='athena-identifier'>{step.market.marketConditionId}</dd>
            </div>
            <div>
                <dt>Worm request ID</dt>
                <dd className='athena-identifier'>{step.positionRequestId || 'Not assigned'}</dd>
            </div>
            <div>
                <dt>Provider / order state</dt>
                <dd>
                    {displayCode(step.providerState, 'Not observed')} / {displayCode(step.providerOrderState, 'Not observed')}
                </dd>
            </div>
            {step.completionSource === 'OPEN_POSITION' && (
                <>
                    <div>
                        <dt>Completion source</dt>
                        <dd>Open position observed</dd>
                    </div>
                    <div>
                        <dt>Position public key</dt>
                        <dd className='athena-identifier'>{step.completionPositionPubkey}</dd>
                    </div>
                    <div>
                        <dt>Position request public key</dt>
                        <dd className='athena-identifier'>{step.completionPositionRequestPubkey || 'Not provided'}</dd>
                    </div>
                    <div>
                        <dt>Position created · UTC+8</dt>
                        <dd>{formatBeijingUnixSeconds(step.completionPositionCreatedAt)}</dd>
                    </div>
                    <div>
                        <dt>Completion observed · UTC+8</dt>
                        <dd>{formatBeijingUnixSeconds(step.completedAt) || '—'}</dd>
                    </div>
                </>
            )}
            <div>
                <dt>Last update · UTC+8</dt>
                <dd>{formatBeijingUnixSeconds(step.updatedAt) || '—'}</dd>
            </div>
        </dl>
    </details>
);

const StepCard = ({step}: {step: WormExecutionRunStep}) => (
    <div className='worm-execution-step-card'>
        <span className='worm-execution-note'>Step {step.ordinal}</span>
        <Typography.Text strong>{step.wallet.remark || 'Solana wallet'}</Typography.Text>
        <Typography.Text strong>{step.market.marketTitle}</Typography.Text>
        <span className='worm-execution-note'>
            {step.market.backend} · {step.side}
        </span>
        <span className='athena-numeric'>{step.funds ? `${step.funds} USDC` : 'Unavailable'}</span>
        <span className='worm-execution-note'>Market · {step.leverage}×</span>
        <StepStatusTag step={step} />
        {step.reasonCode && <span className='worm-execution-note'>{displayCode(step.reasonCode)}</span>}
        <StepEvidence step={step} />
    </div>
);

const ExecutionSteps = ({run}: {run: WormExecutionRun}) => {
    const {page, pageSize, setPage} = usePagedParams(50, stepPageSizes);
    const data = useAsyncData(() => services.wormTrading.listExecutionRunSteps(run.id, page, pageSize), [run.id, run.updatedAt, page, pageSize]);
    const columns: ColumnsType<WormExecutionRunStep> = [
        {
            title: 'Step / Wallet',
            width: 170,
            render: step => (
                <div>
                    <span className='worm-execution-note'>Step {step.ordinal}</span>
                    <p>{step.wallet.remark || 'Solana wallet'}</p>
                </div>
            )
        },
        {
            title: 'Market / Evidence',
            render: step => (
                <div>
                    <Typography.Text strong>{step.market.marketTitle}</Typography.Text>
                    <p className='worm-execution-note'>
                        {step.market.backend} · {step.side}
                    </p>
                    <StepEvidence step={step} />
                </div>
            )
        },
        {
            title: 'Funds',
            width: 140,
            className: 'athena-numeric-column',
            align: 'right',
            render: step => (
                <div>
                    <span className='athena-numeric'>{step.funds ? `${step.funds} USDC` : 'Unavailable'}</span>
                    <p className='worm-execution-note'>Market · {step.leverage}×</p>
                </div>
            )
        },
        {
            title: 'State',
            width: 210,
            render: step => (
                <div>
                    <StepStatusTag step={step} />
                    {step.reasonCode && <p className='worm-execution-note'>{displayCode(step.reasonCode)}</p>}
                </div>
            )
        }
    ];
    return (
        <Section
            title='Steps'
            extra={
                <Button icon={<ReloadOutlined aria-hidden='true' />} loading={data.loading} onClick={data.reload}>
                    Refresh steps
                </Button>
            }>
            <p className='worm-execution-note'>Wallet order first. Expand a step for identity and evidence.</p>
            {data.error && <Alert type='warning' showIcon title={data.data ? 'Steps are stale' : 'Could not load steps'} description={requestErrorMessage(data.error)} />}
            {!data.data ? (
                data.loading ? (
                    <Skeleton active />
                ) : null
            ) : (
                <ResourceTable<WormExecutionRunStep>
                    rowKey='ordinal'
                    label='Worm execution steps'
                    items={data.data.items}
                    columns={columns}
                    loading={data.loading}
                    total={data.data.total}
                    page={page}
                    pageSize={pageSize}
                    pageSizeOptions={stepPageSizes}
                    onPageChange={setPage}
                    scrollX={900}
                    compactRender={step => <StepCard step={step} />}
                    compactEmptyDescription='No steps on this page'
                />
            )}
        </Section>
    );
};

const RunSummary = ({run}: {run: WormExecutionRun}) => {
    const navigate = useNavigate();
    return (
        <Card className='worm-execution-summary-card'>
            <div className='worm-execution-summary-card__heading'>
                <div>
                    <Typography.Title level={2}>{run.combinationName}</Typography.Title>
                    <code className='athena-identifier'>{run.id}</code>
                </div>
                <RunStatusTag state={run.state} />
            </div>
            <div className='worm-execution-stat-grid'>
                <div>
                    <span>Frozen combination</span>
                    <strong>Revision {run.combinationRevision}</strong>
                </div>
                <div>
                    <span>Order type</span>
                    <strong>Market · 1×</strong>
                </div>
                <div>
                    <span>Prepared · UTC+8</span>
                    <strong>{formatBeijingUnixSeconds(run.requestedAt)}</strong>
                </div>
            </div>
            <div className='worm-execution-progress-heading'>
                <span>
                    {run.counts.terminal} of {run.counts.total} steps terminal
                </span>
                <span>Completed (open position observed): {run.counts.completed}</span>
            </div>
            <Progress aria-label={`${run.counts.terminal} of ${run.counts.total} steps terminal`} strokeColor='var(--athena-muted)' percent={runProgress(run)} showInfo={false} />
            <p className='worm-execution-counts'>
                {run.counts.actionable} initially actionable · {run.counts.skipped} skipped · {run.counts.satisfied} satisfied · {run.counts.failed} failed ·{' '}
                {run.counts.notExecuted} not executed
            </p>
            <Button type='link' onClick={() => navigate('/worm-trading/combinations')}>
                Saved combinations
            </Button>
            <details className='worm-step-evidence'>
                <summary>Frozen snapshot identity</summary>
                <dl>
                    <div>
                        <dt>Plan ID</dt>
                        <dd className='athena-identifier'>{run.planId}</dd>
                    </div>
                    <div>
                        <dt>Combination ID</dt>
                        <dd className='athena-identifier'>{run.combinationId}</dd>
                    </div>
                </dl>
            </details>
        </Card>
    );
};

const RunMandatoryGuards = () => (
    <div className='worm-execution-guard-copy'>
        {wormExecutionMandatoryGuardDefinitions.map(definition => (
            <div key={definition.key}>
                <strong>{definition.title} · Always on</strong>
                <p className='worm-execution-note'>{definition.description}</p>
            </div>
        ))}
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
    const captureModuleAccess = useModuleAccessLease('worm');
    const accessCurrent = React.useMemo(() => captureModuleAccess(), [captureModuleAccess]);
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
    const operationRef = React.useRef(false);
    const scope = JSON.stringify([authorization.user.accountId, authorization.user.iss, authorization.revision, id, canWrite]);
    const scopeRef = React.useRef(scope);
    scopeRef.current = scope;
    const mountedRef = React.useRef(true);
    const dialogRef = React.useRef<{destroy(): void}>();
    React.useEffect(() => {
        mountedRef.current = true;
        operationRef.current = false;
        setOperation('');
        return () => {
            mountedRef.current = false;
            dialogRef.current?.destroy();
        };
    }, [scope]);
    const operationCurrent = () => mountedRef.current && scopeRef.current === scope && accessCurrent();

    const [driverActive, setDriverActive] = React.useState(false);
    const [driverInterruption, setDriverInterruption] = React.useState<ExecutionDriverInterruption>();
    const driverActiveRef = React.useRef(false);
    const driverPromiseRef = React.useRef<Promise<void> | undefined>(undefined);
    const driverWaitRef = React.useRef<{timer: number; resolve(): void}>();
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
            if (driverWaitRef.current) {
                window.clearTimeout(driverWaitRef.current.timer);
                driverWaitRef.current.resolve();
                driverWaitRef.current = undefined;
            }
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
                    await new Promise<void>(resolve => {
                        driverWaitRef.current = {
                            timer: window.setTimeout(() => {
                                driverWaitRef.current = undefined;
                                resolve();
                            }, detailPollIntervalMS),
                            resolve
                        };
                    });
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
        const requiredAction = action === 'continue' ? 'CONTINUE' : (action.toUpperCase() as WormExecutionAllowedAction);
        if (!current || operationRef.current || !canWrite || !operationCurrent() || !current.allowedActions.includes(requiredAction)) return;
        operationRef.current = true;
        setOperation(action);
        try {
            if (action === 'pause' || action === 'terminate') {
                stopDriver();
                await waitForDriverToSettle();
                if (!operationCurrent()) return;
                current = await fetchRun();
                if (!operationCurrent()) return;
                publishRun(current);
                if (!current.allowedActions.includes(requiredAction)) return;
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
            if (!operationCurrent()) return;
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
            if (!operationCurrent()) return;
            ctx.notifications.error(`Could not ${action} execution`, requestErrorMessage(reason, 'The execution state was not changed.'));
            try {
                await loadRun();
            } catch {
                // Keep the command error visible.
            }
        } finally {
            if (operationCurrent()) {
                operationRef.current = false;
                setOperation('');
            }
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
        if (!current || operationRef.current || !canWrite || !operationCurrent() || !current.allowedActions.includes('AUTHORIZE')) return;
        operationRef.current = true;
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
                if (!operationCurrent()) return;
                if (connected.publicKey.toString() !== identity.solanaAddress) throw new Error('Phantom is connected to a different login address.');
                const challenge = await services.wormTrading.createSolanaExecutionAuthorizationChallenge(current.id, command);
                if (!operationCurrent()) return;
                const signed = await provider.signMessage(new TextEncoder().encode(challenge.message), 'utf8');
                if (!operationCurrent()) return;
                next = await services.wormTrading.verifySolanaExecutionAuthorization(current.id, rawBase64URL(signed.signature));
            } else {
                next = await services.wormTrading.authorizeDevelopmentExecutionRun(current.id, command);
            }
            if (!operationCurrent()) return;
            publishRun(next);
            setDriverInterruption(undefined);
            ctx.notifications.success('Execution authorized', 'Review the frozen run, then start it explicitly.');
        } catch (reason) {
            if (!operationCurrent()) return;
            ctx.notifications.error('Could not authorize execution', requestErrorMessage(reason, 'No execution authorization was recorded.'));
            try {
                await loadRun();
            } catch {
                // Keep the authorization error visible.
            }
        } finally {
            if (operationCurrent()) {
                operationRef.current = false;
                setOperation('');
            }
        }
    };

    const confirmAuthorize = () => {
        const current = runRef.current;
        if (!current) return;
        dialogRef.current = ctx.modal.confirm({
            className: 'worm-execution-confirm',
            width: 'min(36rem, calc(100vw - 2rem))',
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
        if (!current || !step || operationRef.current || !canWrite || !operationCurrent() || !current.allowedActions.includes('RECONCILE')) return;
        operationRef.current = true;
        setOperation('reconcile');
        try {
            const result = await services.wormTrading.reconcileExecutionStep(current.id, step.id, {
                commandId: newCommandID(),
                expectedRevision: current.revision
            });
            if (!operationCurrent()) return;
            publishRun(result.run);
            setDriverInterruption(undefined);
        } catch (reason) {
            if (!operationCurrent()) return;
            ctx.notifications.error('Could not check authoritative status', requestErrorMessage(reason, 'No Worm mutation was replayed.'));
        } finally {
            if (operationCurrent()) {
                operationRef.current = false;
                setOperation('');
            }
        }
    };

    const primaryAction = (current: WormExecutionRun, recoveryRequired: boolean) => {
        const has = (action: WormExecutionAllowedAction) => current.allowedActions.includes(action);
        if (has('AUTHORIZE'))
            return (
                <Button type='primary' icon={<SafetyCertificateOutlined aria-hidden='true' />} loading={operation === 'authorize'} onClick={confirmAuthorize}>
                    Authorize
                </Button>
            );
        if (has('START'))
            return (
                <Button type='primary' icon={<PlayCircleOutlined aria-hidden='true' />} loading={operation === 'start'} onClick={() => void runCommand('start')}>
                    Start
                </Button>
            );
        if (has('PAUSE'))
            return (
                <Button type='primary' icon={<PauseCircleOutlined aria-hidden='true' />} loading={operation === 'pause'} onClick={() => void runCommand('pause')}>
                    {recoveryRequired ? 'Pause and review' : 'Pause'}
                </Button>
            );
        if (has('CONTINUE'))
            return (
                <Button type='primary' icon={<PlayCircleOutlined aria-hidden='true' />} loading={operation === 'continue'} onClick={() => void runCommand('continue')}>
                    Continue
                </Button>
            );
        return null;
    };

    const alert = run ? stateAlert(run) : undefined;
    const recoveryAlert = run ? driverRecoveryAlert(run, driverInterruption) : undefined;
    const recoveryRequired = Boolean(recoveryAlert);
    const actions = run && canWrite && !error && (
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
                            icon={<StopOutlined aria-hidden='true' />}
                            disabled={Boolean(operation) && operation !== 'terminate'}
                            loading={operation === 'terminate'}
                            onClick={() =>
                                (dialogRef.current = ctx.modal.confirm({
                                    className: 'worm-execution-confirm',
                                    width: 'min(36rem, calc(100vw - 2rem))',
                                    okButtonProps: {danger: true},
                                    autoFocusButton: 'cancel',
                                    title: 'Terminate this execution?',
                                    content:
                                        'Unstarted steps become Not executed. Any step already sent to Worm continues to an authoritative result or unknown state; Athena will not cancel it.',
                                    okText: 'Terminate execution',
                                    onOk: () => runCommand('terminate')
                                }))
                            }>
                            Terminate
                        </Button>
                    </Tooltip>
                )}
                {primaryAction(run, recoveryRequired)}
            </Space>
        </div>
    );
    return (
        <div className='worm-execution-theme'>
            <Button className='worm-execution-back' type='text' icon={<ArrowLeftOutlined aria-hidden='true' />} onClick={() => navigate('/worm-trading/executions')}>
                All executions
            </Button>
            <AppPage
                title='Worm Trading Execution'
                subtitle='Review the frozen intent, authorize this Run, then start explicitly.'
                loading={loading}
                error={error}
                onRefresh={() => {
                    void refreshRun();
                }}>
                {error && run && <Alert type='warning' showIcon title='Execution status is stale' description='Showing the last confirmed Run. Refresh before acting.' />}
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
                        <Card
                            className='worm-execution-control-card'
                            title={run.state === 'AWAITING_AUTHORIZATION' ? 'Authorize this frozen Run' : 'Authorization and control'}
                            extra={actions}>
                            <p className='worm-execution-note'>Preparing this Run submitted no order. Authorize separately, then choose Start.</p>
                            <p className='worm-execution-note'>
                                <strong>Worm transaction trust boundary.</strong> Athena signs the exact Solana transaction returned by Worm. The frozen funds request is capped at
                                10 USDC per step, but the signer does not inspect programs, accounts, instructions, or independently prove the actual chain spend.
                            </p>
                            <RunMandatoryGuards />
                            <p className='worm-execution-note'>Run updated · {formatBeijingUnixSeconds(run.updatedAt)} UTC+8</p>
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
                                icon={<ExclamationCircleOutlined aria-hidden='true' />}
                                title='Mutation outcome unknown'
                                description='This Wallet and market remain isolated. Athena will only perform authoritative read checks; Open, Finalize, and cancel are never replayed automatically.'
                                action={
                                    canWrite &&
                                    !error && (
                                        <Button icon={<ReloadOutlined aria-hidden='true' />} loading={operation === 'reconcile'} onClick={() => void reconcile()}>
                                            Check authoritative status
                                        </Button>
                                    )
                                }
                            />
                        )}
                        <CurrentStepPanel step={run.currentStep} />
                        <ExecutionSteps run={run} />
                        <div className='worm-execution-action-spacer' aria-hidden='true' />
                    </div>
                )}
            </AppPage>
        </div>
    );
};
