import {OperationFacts} from '../components/operation-facts';
import {Alert, Button, Empty, Form, InputNumber, Progress, Tabs, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {AppPage, ResourceTable, Section, StatusTag} from '../../components';
import {useAdminReadScope} from '../read-scope';
import {useVisibleQuery} from '../../shared/use-visible-query';
import {Context, useAuthorization} from '../../shared/context';
import {formatBeijingUnixSeconds} from '../../shared/format';
import {adminServices as services} from '../services';
import type {
    EtherscanGatewayProbeCounts,
    EtherscanGatewayProbeGatewaySummary,
    EtherscanGatewayProbeKeySummary,
    EtherscanGatewayProbeRun,
    EtherscanGatewayStatus
} from '../../shared/services/service-status-service';

type ProbeTone = 'good' | 'bad' | 'warn' | 'neutral' | 'running';
type FailureCategoryKey = Exclude<keyof EtherscanGatewayProbeCounts, 'success'>;

const failureCategories: Array<{key: FailureCategoryKey; label: string; tone: Exclude<ProbeTone, 'good' | 'running'>}> = [
    {key: 'rateLimit', label: 'Rate Limit', tone: 'warn'},
    {key: 'authentication', label: 'Auth', tone: 'bad'},
    {key: 'plan', label: 'Plan', tone: 'bad'},
    {key: 'invalidRequest', label: 'Invalid', tone: 'bad'},
    {key: 'malformed', label: 'Malformed', tone: 'bad'},
    {key: 'upstream', label: 'Upstream', tone: 'bad'},
    {key: 'other', label: 'Other', tone: 'bad'}
];

const formatUnixSeconds = (value?: number) => formatBeijingUnixSeconds(value) || '-';
const formatLatency = (value?: number) => (value === undefined ? '-' : `${value} ms`);
const formatDuration = (value?: number) => {
    if (!value) {
        return '-';
    }
    return value < 1000 ? `${value} ms` : `${(value / 1000).toFixed(3)} s`;
};

const isRunning = (item: EtherscanGatewayStatus) => item.reachable && item.started && item.status === 'running';
const isRuntimeError = (item: EtherscanGatewayStatus) => !isRunning(item) && item.status !== 'unreachable';

const runtimeTag = (item: EtherscanGatewayStatus) => (
    <StatusTag value={item.status || '-'} positive={isRunning(item)} negative={item.status === 'unreachable' || isRuntimeError(item)} />
);

const countsTotal = (counts?: EtherscanGatewayProbeCounts) =>
    counts ? counts.success + counts.rateLimit + counts.authentication + counts.plan + counts.invalidRequest + counts.malformed + counts.upstream + counts.other : 0;

const successRateValue = (counts?: EtherscanGatewayProbeCounts, fallbackTotal?: number) => {
    const total = fallbackTotal || countsTotal(counts);
    if (!counts || total <= 0) {
        return 0;
    }
    return Number(((counts.success * 100) / total).toFixed(2));
};

const successRate = (counts?: EtherscanGatewayProbeCounts, fallbackTotal?: number) => {
    const total = fallbackTotal || countsTotal(counts);
    if (!counts || total <= 0) {
        return '-';
    }
    return `${successRateValue(counts, fallbackTotal).toFixed(2)}%`;
};

const failureCount = (counts?: EtherscanGatewayProbeCounts) => {
    if (!counts) {
        return 0;
    }
    return Math.max(countsTotal(counts) - counts.success, 0);
};

const severeFailureCount = (counts?: EtherscanGatewayProbeCounts) =>
    counts ? counts.authentication + counts.plan + counts.invalidRequest + counts.malformed + counts.upstream + counts.other : 0;

const mainFailureCause = (counts?: EtherscanGatewayProbeCounts): {label: string; value: number; tone: ProbeTone} => {
    if (!counts) {
        return {label: 'None', value: 0, tone: 'neutral'};
    }
    const top = failureCategories.reduce(
        (best, category) => {
            const value = counts[category.key];
            return value > best.value ? {label: category.label, value, tone: category.tone as ProbeTone} : best;
        },
        {label: 'None', value: 0, tone: 'neutral' as ProbeTone}
    );
    return top.value > 0 ? top : {label: 'None', value: 0, tone: 'neutral'};
};

const rowTone = (counts?: EtherscanGatewayProbeCounts): ProbeTone => {
    const total = countsTotal(counts);
    if (!counts || total <= 0) {
        return 'neutral';
    }
    if (severeFailureCount(counts) > 0) {
        return 'bad';
    }
    if (successRateValue(counts) < 90) {
        return 'warn';
    }
    return 'good';
};

const rowResultLabel = (counts?: EtherscanGatewayProbeCounts) => {
    const tone = rowTone(counts);
    if (tone === 'good') {
        return 'ok';
    }
    if (tone === 'warn') {
        return 'below 90%';
    }
    if (tone === 'bad') {
        return 'needs check';
    }
    return '-';
};

const probeTone = (run?: EtherscanGatewayProbeRun): ProbeTone => {
    if (run?.status === 'running') {
        return 'running';
    }
    if (run?.result === 'pass') {
        return 'good';
    }
    if (run?.result === 'fail' || run?.result === 'error' || run?.status === 'error') {
        return 'bad';
    }
    return 'neutral';
};

const probeResultLabel = (run?: EtherscanGatewayProbeRun) => {
    if (run?.status === 'running') {
        return 'RUNNING';
    }
    if (run?.result === 'pass') {
        return 'PASS';
    }
    if (run?.result === 'fail') {
        return 'FAIL';
    }
    if (run?.result === 'error' || run?.status === 'error') {
        return 'ERROR';
    }
    return 'IDLE';
};

const requiredDeltaLabel = (run: EtherscanGatewayProbeRun) => {
    if (run.status === 'running') {
        return 'pending';
    }
    const delta = run.counts.success - run.requiredSuccess;
    if (delta >= 0) {
        return `+${delta} over line`;
    }
    return `${Math.abs(delta)} short`;
};

const toneClass = (tone: ProbeTone) => `etherscan-probe-tone--${tone}`;

const resultTag = (run?: EtherscanGatewayProbeRun) => (
    <StatusTag
        value={run?.status === 'running' ? 'running' : run?.result || run?.status || 'idle'}
        positive={run?.result === 'pass'}
        negative={run?.result === 'fail' || run?.result === 'error' || run?.status === 'error'}
    />
);

const TonePill = (props: {tone: ProbeTone; children: React.ReactNode}) => <span className={`etherscan-probe-pill ${toneClass(props.tone)}`}>{props.children}</span>;

const ProbeRate = (props: {counts?: EtherscanGatewayProbeCounts; total?: number}) => {
    const tone = rowTone(props.counts);
    const rate = successRateValue(props.counts, props.total);
    const status = tone === 'bad' ? 'exception' : tone === 'good' ? 'success' : 'normal';
    return (
        <div className='etherscan-probe-rate'>
            <Progress percent={rate} size='small' showInfo={false} status={status} strokeColor={tone === 'warn' ? 'var(--athena-amber)' : undefined} />
            <Typography.Text className={`etherscan-probe-rate__value ${toneClass(tone)}`}>{successRate(props.counts, props.total)}</Typography.Text>
        </div>
    );
};

const sampleTone = (sample: string): ProbeTone => {
    const lower = sample.toLowerCase();
    if (lower.includes('rate_limit') || lower.includes('rate limit')) {
        return 'warn';
    }
    if (
        lower.includes('authentication') ||
        lower.includes('permission') ||
        lower.includes('invalid') ||
        lower.includes('malformed') ||
        lower.includes('upstream') ||
        lower.includes('other')
    ) {
        return 'bad';
    }
    return 'neutral';
};

export const EtherscanGatewaysPage = () => {
    const scope = useAdminReadScope('etherscan-workspace');
    return scope.isCurrent() ? (
        <EtherscanWorkspace key={scope.key} />
    ) : (
        <AppPage title='Etherscan Gateways' loading>
            <p>Checking administrator access…</p>
        </AppPage>
    );
};
const EtherscanWorkspace = () => {
    const authorization = useAuthorization();
    const scope = useAdminReadScope('etherscan');
    const ctx = React.useContext(Context);
    const [form] = Form.useForm();
    const data = useVisibleQuery(() => services.serviceStatus.listEtherscanGatewayStatuses(), useAdminReadScope('etherscan-health'), 10000);
    const latestProbe = useVisibleQuery(() => services.serviceStatus.getLatestEtherscanGatewayProbeRun(), useAdminReadScope('etherscan-latest'), 10000);
    const [probeRun, setProbeRun] = React.useState<EtherscanGatewayProbeRun>();
    const [probeSubmitting, setProbeSubmitting] = React.useState(false);
    const [startError, setStartError] = React.useState('');
    const requestRef = React.useRef<ReturnType<typeof services.serviceStatus.runEtherscanGatewayProbe>>();
    const mounted = React.useRef(true);
    React.useEffect(
        () => () => {
            mounted.current = false;
            requestRef.current?.abort?.();
        },
        []
    );
    const seedRun = probeRun || latestProbe.data;
    const runScope = {...scope, key: JSON.stringify([scope.key, seedRun?.runID, seedRun?.status]), isCurrent: () => scope.isCurrent() && seedRun?.status === 'running'};
    const polled = useVisibleQuery(() => services.serviceStatus.getEtherscanGatewayProbeRun(seedRun!.runID), runScope, 1000);
    React.useEffect(() => {
        if (polled.data && polled.data.status !== 'running') setProbeRun(polled.data);
    }, [polled.data]);
    const activeRun = polled.data || seedRun;
    const probeRunning = activeRun?.status === 'running';
    const hasProbeResult = Boolean(activeRun?.status && activeRun.status !== 'idle');
    const items = data.data?.items || [];
    const checkedAt = formatBeijingUnixSeconds(data.data?.checkedAt) || 'Not checked';
    const running = items.filter(isRunning).length;
    const unreachable = items.filter(item => item.status === 'unreachable').length;
    const errors = items.filter(isRuntimeError).length;
    const runProbe = async (values: {intervalMS: number; requestsPerKey: number}) => {
        if (!authorization.isAdmin || !scope.isCurrent() || requestRef.current || probeRunning) return;
        setProbeSubmitting(true);
        setStartError('');
        const request = services.serviceStatus.runEtherscanGatewayProbe({intervalMS: values.intervalMS, requestsPerKey: values.requestsPerKey});
        requestRef.current = request;
        try {
            const result = await request;
            if (!mounted.current || !scope.isCurrent()) return;
            setProbeRun(result);
            ctx.notifications.info('Probe started', result.runID);
        } catch (error: any) {
            if (mounted.current && scope.isCurrent()) setStartError(error?.message || 'Could not start the Etherscan Gateway probe.');
        } finally {
            requestRef.current = undefined;
            if (mounted.current && scope.isCurrent()) setProbeSubmitting(false);
        }
    };
    const summaryColumns: ColumnsType<EtherscanGatewayProbeGatewaySummary | EtherscanGatewayProbeKeySummary> = [
        {
            title: 'Gateway / Key',
            render: item => (
                <Typography.Text className='athena-identifier' copyable>
                    {'gateway' in item ? item.gateway : item.keyLabel}
                </Typography.Text>
            )
        },
        {title: 'Result', render: item => <TonePill tone={rowTone(item.counts)}>{rowResultLabel(item.counts)}</TonePill>},
        {title: 'Success rate', className: 'athena-numeric-column', render: item => <ProbeRate counts={item.counts} />},
        {title: 'Success', className: 'athena-numeric-column', render: item => `${item.counts.success}/${countsTotal(item.counts)}`},
        {title: 'Failures', className: 'athena-numeric-column', render: item => failureCount(item.counts)},
        {
            title: 'Main cause',
            render: item => {
                const cause = mainFailureCause(item.counts);
                return (
                    <TonePill tone={cause.tone}>
                        {cause.label} {cause.value || ''}
                    </TonePill>
                );
            }
        }
    ];
    const summary = (item: EtherscanGatewayProbeGatewaySummary | EtherscanGatewayProbeKeySummary) => (
        <article className='etherscan-summary-row'>
            <Typography.Text className='athena-identifier' copyable>
                {'gateway' in item ? item.gateway : item.keyLabel}
            </Typography.Text>
            <OperationFacts
                columns={1}
                items={[
                    {label: 'Result', value: <TonePill tone={rowTone(item.counts)}>{rowResultLabel(item.counts)}</TonePill>},
                    {label: 'Success rate', value: successRate(item.counts)},
                    {label: 'Success', value: `${item.counts.success}/${countsTotal(item.counts)}`},
                    {label: 'Failures', value: failureCount(item.counts)},
                    {label: 'Main cause', value: `${mainFailureCause(item.counts).label} ${mainFailureCause(item.counts).value}`}
                ]}
            />
        </article>
    );
    return (
        <AppPage title='Etherscan Gateways' subtitle='Gateway health and Etherscan request testing.' onRefresh={data.reload}>
            <Tabs
                className='admin-source-tabs'
                defaultActiveKey='gateways'
                items={[
                    {
                        key: 'gateways',
                        label: (
                            <span>
                                Gateways<small>{data.loading ? 'Loading' : data.error ? 'Unavailable' : `${running} running · ${unreachable} unreachable`}</small>
                            </span>
                        ),
                        children: (
                            <Section title='Runtime status'>
                                {data.error && <Alert type='error' title='Gateway health unavailable' description={data.error.message} />}
                                {data.stale && <Alert type='warning' title='Stale gateway health — showing the last successful read' />}
                                <p className='admin-source-note'>Last checked: {checkedAt} (UTC+8)</p>
                                <OperationFacts
                                    items={[
                                        {label: 'Total', value: items.length},
                                        {label: 'Running', value: <TonePill tone='good'>{running}</TonePill>},
                                        {label: 'Unreachable', value: <TonePill tone={unreachable ? 'bad' : 'neutral'}>{unreachable}</TonePill>},
                                        {label: 'Errors', value: errors}
                                    ]}
                                />
                                <div className='etherscan-gateway-records'>
                                    {items.map(item => (
                                        <article className='etherscan-gateway-record' key={item.address}>
                                            <div className='etherscan-gateway-facts'>
                                                <Typography.Text className='athena-identifier' copyable>
                                                    {item.address}
                                                </Typography.Text>
                                                <div>
                                                    <span>Reachable</span>
                                                    <StatusTag value={item.reachable ? 'Yes' : 'No'} positive={item.reachable} negative={!item.reachable} />
                                                </div>
                                                <div>
                                                    <span>Runtime</span>
                                                    {runtimeTag(item)}
                                                </div>
                                                <div>
                                                    <span>Latency</span>
                                                    <span className='athena-number'>{formatLatency(item.latencyMS)}</span>
                                                </div>
                                            </div>
                                            <p className='admin-source-note'>Checked {formatUnixSeconds(item.checkedAt)} (UTC+8)</p>
                                            <details className='admin-operation-details'>
                                                <summary>Connection details</summary>
                                                <Typography.Text className='athena-identifier' copyable={Boolean(item.etherscanBaseURL)}>
                                                    {item.etherscanBaseURL || 'Base URL unavailable'}
                                                </Typography.Text>
                                            </details>
                                            {item.errorMessage && (
                                                <p className='etherscan-gateway-error'>
                                                    <Typography.Text className='athena-identifier' type='danger' copyable>
                                                        {item.errorMessage}
                                                    </Typography.Text>
                                                </p>
                                            )}
                                        </article>
                                    ))}
                                </div>
                                {!items.length && !data.loading && <Empty description='No Etherscan gateway IPs configured' />}
                                <p className='admin-source-note'>Runtime health is separate from request test results. Open Live Probe to inspect the latest test.</p>
                            </Section>
                        )
                    },
                    {
                        key: 'probe',
                        label: (
                            <span>
                                Live Probe<small>{latestProbe.loading ? 'Loading' : latestProbe.error ? 'Unavailable' : probeResultLabel(activeRun)}</small>
                            </span>
                        ),
                        children: (
                            <>
                                <Section title='Run a request test'>
                                    <p>Sends requests using the configured API keys and gateways.</p>
                                    <Form className='etherscan-probe-form' form={form} layout='vertical' initialValues={{intervalMS: 10, requestsPerKey: 6}} onFinish={runProbe}>
                                        <Form.Item name='intervalMS' label='Interval (ms)' extra='1–1,000 ms' rules={[{required: true}]}>
                                            <InputNumber min={1} max={1000} precision={0} disabled={probeSubmitting || probeRunning} />
                                        </Form.Item>
                                        <Form.Item name='requestsPerKey' label='Requests per API key' extra='1–20 requests' rules={[{required: true}]}>
                                            <InputNumber min={1} max={20} precision={0} disabled={probeSubmitting || probeRunning} />
                                        </Form.Item>
                                        <Form.Item>
                                            <Button type='primary' htmlType='submit' disabled={!authorization.isAdmin} loading={probeSubmitting || probeRunning}>
                                                Run Probe
                                            </Button>
                                        </Form.Item>
                                    </Form>
                                    {startError && <Alert type='error' title='Probe failed to start' description={startError} />}
                                </Section>
                                <Section title='Latest probe' extra={resultTag(activeRun)}>
                                    {latestProbe.error && (
                                        <Alert
                                            type='error'
                                            title='Latest probe unavailable'
                                            description={latestProbe.error.message}
                                            action={<Button onClick={latestProbe.reload}>Retry</Button>}
                                        />
                                    )}
                                    {polled.error && (
                                        <Alert
                                            type='error'
                                            title='Probe refresh failed'
                                            description={polled.error.message}
                                            action={<Button onClick={polled.reload}>Retry</Button>}
                                        />
                                    )}
                                    {!hasProbeResult && !latestProbe.loading && <Empty description='No probe has run yet' />}
                                    {hasProbeResult && activeRun && (
                                        <>
                                            <p className='admin-source-note'>Finished {formatUnixSeconds(activeRun.finishedAt)} (UTC+8)</p>
                                            <div className='etherscan-probe-result-line'>
                                                <strong className='etherscan-probe-kpi__value'>
                                                    {activeRun.counts.success} / {activeRun.total}
                                                </strong>
                                                <span>successful requests</span>
                                                <TonePill tone={probeTone(activeRun)}>{successRate(activeRun.counts, activeRun.total)}</TonePill>
                                            </div>
                                            <p>
                                                {activeRun.requiredSuccess} successful requests required · {requiredDeltaLabel(activeRun)}
                                            </p>
                                            <p className='admin-source-note'>
                                                {activeRun.keyCount} API keys · {activeRun.gatewayCount} gateways · {activeRun.requestsPerKey} requests / key ·{' '}
                                                {activeRun.intervalMS} ms interval
                                            </p>
                                            <h3>Failures {failureCount(activeRun.counts)}</h3>
                                            <div className='etherscan-probe-breakdown'>
                                                {failureCategories.map(category => (
                                                    <div
                                                        className={`etherscan-probe-breakdown__item ${toneClass(activeRun.counts[category.key] ? category.tone : 'neutral')}`}
                                                        key={category.key}>
                                                        <span>{category.label}</span>
                                                        <strong>{activeRun.counts[category.key]}</strong>
                                                    </div>
                                                ))}
                                            </div>
                                            <Tabs
                                                defaultActiveKey='gateway'
                                                items={[
                                                    {
                                                        key: 'gateway',
                                                        label: 'By Gateway',
                                                        children: (
                                                            <ResourceTable<EtherscanGatewayProbeGatewaySummary | EtherscanGatewayProbeKeySummary>
                                                                rowKey={item => ('gateway' in item ? item.gateway : item.keyLabel)}
                                                                label='Probe gateway summaries'
                                                                items={activeRun.gatewaySummaries}
                                                                columns={summaryColumns}
                                                                compactRender={summary}
                                                            />
                                                        )
                                                    },
                                                    {
                                                        key: 'key',
                                                        label: 'By API Key',
                                                        children: (
                                                            <ResourceTable<EtherscanGatewayProbeGatewaySummary | EtherscanGatewayProbeKeySummary>
                                                                rowKey={item => ('gateway' in item ? item.gateway : item.keyLabel)}
                                                                label='Probe API key summaries'
                                                                items={activeRun.keySummaries}
                                                                columns={summaryColumns}
                                                                compactRender={summary}
                                                            />
                                                        )
                                                    }
                                                ]}
                                            />
                                            <p className='admin-source-note'>Row status checks failure types and the 90% threshold independently of the overall result.</p>
                                            <details className='admin-operation-details'>
                                                <summary>Timing &amp; run details</summary>
                                                <OperationFacts
                                                    items={[
                                                        {label: 'Run ID', value: <span className='athena-identifier'>{activeRun.runID}</span>},
                                                        {label: 'Elapsed', value: formatDuration(activeRun.elapsedMS)},
                                                        {label: 'Start spread', value: formatDuration(activeRun.startSpreadMS)},
                                                        {label: 'Created', value: formatUnixSeconds(activeRun.createdAt)},
                                                        {label: 'Started', value: formatUnixSeconds(activeRun.startedAt)},
                                                        {label: 'Finished', value: formatUnixSeconds(activeRun.finishedAt)}
                                                    ]}
                                                />
                                            </details>
                                            {activeRun.errorMessage && <Alert type='error' title={activeRun.errorMessage} />}
                                            {activeRun.samples.length > 0 && (
                                                <details className='admin-operation-details' open>
                                                    <summary>Error samples</summary>
                                                    {activeRun.samples.map(sample => (
                                                        <div className='etherscan-probe-sample' key={sample}>
                                                            <TonePill tone={sampleTone(sample)}>{sampleTone(sample) === 'warn' ? 'Rate limit' : 'Error'}</TonePill>
                                                            <Typography.Text className='athena-identifier' copyable>
                                                                {sample}
                                                            </Typography.Text>
                                                        </div>
                                                    ))}
                                                </details>
                                            )}
                                        </>
                                    )}
                                </Section>
                            </>
                        )
                    }
                ]}
            />
        </AppPage>
    );
};
