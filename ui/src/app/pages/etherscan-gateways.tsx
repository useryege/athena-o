import {Button, Empty, Form, InputNumber, Progress, Space, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {AppPage, MetricRow, ResourceTable, Section, StatusTag, TruncatedText, useAsyncData} from '../components';
import {Context} from '../shared/context';
import {services} from '../shared/services';
import type {EtherscanGatewayProbeCounts, EtherscanGatewayProbeGatewaySummary, EtherscanGatewayProbeKeySummary, EtherscanGatewayProbeRun, EtherscanGatewayStatus} from '../shared/services';

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

const formatUnixSeconds = (value?: number) => (value ? new Date(value * 1000).toLocaleString() : '-');
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

const severeFailureCount = (counts?: EtherscanGatewayProbeCounts) => (counts ? counts.authentication + counts.plan + counts.invalidRequest + counts.malformed + counts.upstream + counts.other : 0);

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

const ProbeKPI = (props: {label: string; value: React.ReactNode; detail?: React.ReactNode; tone?: ProbeTone}) => (
    <div className={`etherscan-probe-kpi ${toneClass(props.tone || 'neutral')}`}>
        <span className='etherscan-probe-kpi__label'>{props.label}</span>
        <strong className='etherscan-probe-kpi__value'>{props.value ?? '-'}</strong>
        {props.detail && <span className='etherscan-probe-kpi__detail'>{props.detail}</span>}
    </div>
);

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

const gatewayStatusRowClassName = (item: EtherscanGatewayStatus) => {
    if (item.status === 'unreachable') {
        return 'etherscan-probe-row--bad';
    }
    if (isRuntimeError(item)) {
        return 'etherscan-probe-row--warn';
    }
    return '';
};

const probeSummaryRowClassName = (item: {counts: EtherscanGatewayProbeCounts}) => {
    const tone = rowTone(item.counts);
    return tone === 'bad' ? 'etherscan-probe-row--bad' : tone === 'warn' ? 'etherscan-probe-row--warn' : '';
};

const sampleTone = (sample: string): ProbeTone => {
    const lower = sample.toLowerCase();
    if (lower.includes('rate_limit') || lower.includes('rate limit')) {
        return 'warn';
    }
    if (lower.includes('authentication') || lower.includes('permission') || lower.includes('invalid') || lower.includes('malformed') || lower.includes('upstream') || lower.includes('other')) {
        return 'bad';
    }
    return 'neutral';
};

export const EtherscanGatewaysPage = (props: {canRunProbe: boolean}) => {
    const ctx = React.useContext(Context);
    const [form] = Form.useForm();
    const data = useAsyncData(() => services.serviceStatus.listEtherscanGatewayStatuses(), []);
    const latestProbe = useAsyncData(() => services.serviceStatus.getLatestEtherscanGatewayProbeRun(), []);
    const [probeRun, setProbeRun] = React.useState<EtherscanGatewayProbeRun>();
    const [probeSubmitting, setProbeSubmitting] = React.useState(false);
    const reloadRef = React.useRef(data.reload);
    reloadRef.current = data.reload;
    const latestProbeReloadRef = React.useRef(latestProbe.reload);
    latestProbeReloadRef.current = latestProbe.reload;

    React.useEffect(() => {
        const timer = window.setInterval(() => reloadRef.current(), 10000);
        return () => window.clearInterval(timer);
    }, []);

    const items = data.data?.items || [];
    const checkedAt = data.data?.checkedAt ? new Date(data.data.checkedAt * 1000).toLocaleString() : 'Not checked';
    const running = items.filter(isRunning).length;
    const unreachable = items.filter(item => item.status === 'unreachable').length;
    const errors = items.filter(isRuntimeError).length;
    const activeRun = probeRun || latestProbe.data;
    const hasProbeResult = Boolean(activeRun && activeRun.status && activeRun.status !== 'idle');
    const probeRunning = activeRun?.status === 'running';

    const columns: ColumnsType<EtherscanGatewayStatus> = [
        {title: 'Address', render: item => <TruncatedText value={item.address} copyable={true} />},
        {
            title: 'Reachable',
            render: item => <StatusTag value={item.reachable ? 'Yes' : 'No'} positive={item.reachable} negative={!item.reachable} />
        },
        {title: 'Runtime', render: runtimeTag},
        {title: 'Latency', render: item => formatLatency(item.latencyMS)},
        {title: 'Etherscan Base URL', render: item => <TruncatedText value={item.etherscanBaseURL} copyable={Boolean(item.etherscanBaseURL)} />},
        {title: 'Checked', render: item => formatUnixSeconds(item.checkedAt)},
        {title: 'Error', render: item => <TruncatedText value={item.errorMessage} copyable={Boolean(item.errorMessage)} />}
    ];

    const gatewayProbeColumns: ColumnsType<EtherscanGatewayProbeGatewaySummary> = [
        {title: 'Gateway', render: item => <TruncatedText value={item.gateway} copyable={true} />},
        {title: 'Result', render: item => <TonePill tone={rowTone(item.counts)}>{rowResultLabel(item.counts)}</TonePill>},
        {title: 'Success Rate', render: item => <ProbeRate counts={item.counts} />},
        {title: 'Success', render: item => `${item.counts.success}/${countsTotal(item.counts)}`},
        {title: 'Failures', render: item => failureCount(item.counts)},
        {
            title: 'Main Cause',
            render: item => {
                const cause = mainFailureCause(item.counts);
                return <TonePill tone={cause.tone}>{cause.value > 0 ? `${cause.label} ${cause.value}` : cause.label}</TonePill>;
            }
        }
    ];

    const keyProbeColumns: ColumnsType<EtherscanGatewayProbeKeySummary> = [
        {title: 'Key Label', render: item => <TruncatedText value={item.keyLabel} copyable={false} />},
        {title: 'Result', render: item => <TonePill tone={rowTone(item.counts)}>{rowResultLabel(item.counts)}</TonePill>},
        {title: 'Success Rate', render: item => <ProbeRate counts={item.counts} />},
        {title: 'Success', render: item => `${item.counts.success}/${countsTotal(item.counts)}`},
        {title: 'Failures', render: item => failureCount(item.counts)},
        {
            title: 'Main Cause',
            render: item => {
                const cause = mainFailureCause(item.counts);
                return <TonePill tone={cause.tone}>{cause.value > 0 ? `${cause.label} ${cause.value}` : cause.label}</TonePill>;
            }
        }
    ];

    React.useEffect(() => {
        if (!probeRun?.runID || probeRun.status !== 'running') {
            return;
        }

        let cancelled = false;
        const poll = async () => {
            try {
                const next = await services.serviceStatus.getEtherscanGatewayProbeRun(probeRun.runID);
                if (!cancelled) {
                    setProbeRun(next);
                    if (next.status !== 'running') {
                        latestProbeReloadRef.current();
                    }
                }
            } catch (err: any) {
                if (!cancelled) {
                    ctx.notifications.error('Probe refresh failed', err?.message || 'Could not refresh probe status.');
                }
            }
        };
        const timer = window.setInterval(poll, 1000);
        poll();
        return () => {
            cancelled = true;
            window.clearInterval(timer);
        };
    }, [ctx.notifications, probeRun?.runID, probeRun?.status]);

    const runProbe = async (values: {intervalMS?: number; requestsPerKey?: number}) => {
        const intervalMS = Number(values.intervalMS || 10);
        const requestsPerKey = Number(values.requestsPerKey || 6);
        setProbeSubmitting(true);
        try {
            const run = await services.serviceStatus.runEtherscanGatewayProbe({intervalMS, requestsPerKey});
            setProbeRun(run);
            ctx.notifications.info('Probe started', run.runID);
        } catch (err: any) {
            ctx.notifications.error('Probe failed to start', err?.message || 'Could not start the Etherscan Gateway probe.');
        } finally {
            setProbeSubmitting(false);
        }
    };

    const activeRunTone = probeTone(activeRun);
    const activeRunMainCause = mainFailureCause(activeRun?.counts);
    const activeRunFailures = failureCount(activeRun?.counts);
    const activeRunRate = successRateValue(activeRun?.counts, activeRun?.total);
    const activeRunProgressStatus = activeRunTone === 'bad' ? 'exception' : activeRunTone === 'good' ? 'success' : activeRunTone === 'running' ? 'active' : 'normal';

    return (
        <AppPage
            title='Etherscan Gateways'
            subtitle={`gRPC runtime status from ETHERSCAN_GATEWAY_IPS · Last checked ${checkedAt}`}
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}>
            <Section title='Summary'>
                <MetricRow
                    items={[
                        {label: 'Total', value: items.length},
                        {label: 'Running', value: running, tone: running === items.length && items.length > 0 ? 'good' : undefined},
                        {label: 'Unreachable', value: unreachable, tone: unreachable > 0 ? 'bad' : undefined},
                        {label: 'Errors', value: errors, tone: errors > 0 ? 'bad' : undefined}
                    ]}
                />
            </Section>
            <Section title='Gateways'>
                {items.length === 0 && !data.loading ? (
                    <Empty description='No Etherscan gateway IPs configured' />
                ) : (
                    <ResourceTable
                        rowKey='address'
                        label='Etherscan gateways'
                        items={items}
                        columns={columns}
                        loading={data.loading}
                        scrollX={1320}
                        stickyHeader={true}
                        rowClassName={gatewayStatusRowClassName}
                    />
                )}
            </Section>
            <Section title='Live Probe'>
                <Form form={form} layout='inline' initialValues={{intervalMS: 10, requestsPerKey: 6}} onFinish={runProbe}>
                    <Form.Item name='intervalMS' label='Interval ms' rules={[{required: true}]}>
                        <InputNumber min={1} max={1000} precision={0} disabled={probeSubmitting || probeRunning} style={{width: 120}} />
                    </Form.Item>
                    <Form.Item name='requestsPerKey' label='Requests / API key' rules={[{required: true}]}>
                        <InputNumber min={1} max={20} precision={0} disabled={probeSubmitting || probeRunning} style={{width: 140}} />
                    </Form.Item>
                    <Form.Item>
                        <Button type='primary' htmlType='submit' loading={probeSubmitting || probeRunning} disabled={!props.canRunProbe}>
                            Run Probe
                        </Button>
                    </Form.Item>
                    <Form.Item>
                        <Space>
                            <Typography.Text type='secondary'>Status</Typography.Text>
                            {resultTag(activeRun)}
                        </Space>
                    </Form.Item>
                </Form>
            </Section>
            {hasProbeResult && activeRun && (
                <Section title='Probe Result'>
                    <Space orientation='vertical' size='middle' className='etherscan-probe-stack'>
                        <div className={`etherscan-probe-hero ${toneClass(activeRunTone)}`}>
                            <div className='etherscan-probe-hero__main'>
                                <TonePill tone={activeRunTone}>{probeResultLabel(activeRun)}</TonePill>
                                <Typography.Text className='etherscan-probe-hero__title'>
                                    Success {activeRun.counts.success}/{activeRun.total} · {successRate(activeRun.counts, activeRun.total)}
                                </Typography.Text>
                                <Typography.Text className={`etherscan-probe-hero__delta ${toneClass(activeRunTone)}`}>{requiredDeltaLabel(activeRun)}</Typography.Text>
                            </div>
                            <Typography.Text className='etherscan-probe-hero__meta'>
                                {activeRun.keyCount} keys · {activeRun.gatewayCount} gateways · {activeRun.requestsPerKey} requests/key · {activeRun.intervalMS} ms interval
                            </Typography.Text>
                            {activeRun.total > 0 && (
                                <Progress
                                    percent={activeRunRate}
                                    status={activeRunProgressStatus}
                                    strokeColor={activeRunTone === 'warn' ? 'var(--athena-amber)' : undefined}
                                    showInfo={false}
                                />
                            )}
                        </div>
                        <div className='etherscan-probe-kpi-grid'>
                            <ProbeKPI label='Success %' value={successRate(activeRun.counts, activeRun.total)} detail={`${activeRun.counts.success}/${activeRun.total} success`} tone={activeRunTone} />
                            <ProbeKPI label='Success' value={`${activeRun.counts.success}/${activeRun.total}`} detail={`${activeRun.requiredSuccess} required`} tone={activeRunTone} />
                            <ProbeKPI label='Required' value={activeRun.requiredSuccess} detail={requiredDeltaLabel(activeRun)} tone={activeRunTone} />
                            <ProbeKPI label='Failures' value={activeRunFailures} detail={`${countsTotal(activeRun.counts)} completed`} tone={activeRunFailures > 0 ? 'warn' : 'good'} />
                            <ProbeKPI
                                label='Main Cause'
                                value={activeRunMainCause.label}
                                detail={activeRunMainCause.value > 0 ? `${activeRunMainCause.value} events` : 'no classified failures'}
                                tone={activeRunMainCause.tone}
                            />
                        </div>
                        <div className='etherscan-probe-breakdown' aria-label='Failure breakdown'>
                            {failureCategories.map(category => {
                                const value = activeRun.counts[category.key];
                                const tone = value > 0 ? category.tone : 'neutral';
                                return (
                                    <div key={category.key} className={`etherscan-probe-breakdown__item ${toneClass(tone)}`}>
                                        <span>{category.label}</span>
                                        <strong>{value}</strong>
                                    </div>
                                );
                            })}
                        </div>
                        <div className='etherscan-probe-timing'>
                            <MetricRow
                                items={[
                                    {label: 'Interval', value: `${activeRun.intervalMS} ms`},
                                    {label: 'Requests / Key', value: activeRun.requestsPerKey},
                                    {label: 'Elapsed', value: formatDuration(activeRun.elapsedMS)},
                                    {label: 'Start Spread', value: formatDuration(activeRun.startSpreadMS)},
                                    {label: 'Started', value: formatUnixSeconds(activeRun.startedAt)},
                                    {label: 'Finished', value: formatUnixSeconds(activeRun.finishedAt)}
                                ]}
                            />
                        </div>
                        {activeRun.errorMessage && <Typography.Text type='danger'>{activeRun.errorMessage}</Typography.Text>}
                    </Space>
                </Section>
            )}
            {hasProbeResult && activeRun && (
                <Section title='By Gateway'>
                    <ResourceTable
                        rowKey='gateway'
                        label='Etherscan Gateway probe gateway summaries'
                        items={activeRun.gatewaySummaries}
                        columns={gatewayProbeColumns}
                        scrollX={960}
                        rowClassName={probeSummaryRowClassName}
                    />
                </Section>
            )}
            {hasProbeResult && activeRun && (
                <Section title='By API Key'>
                    <ResourceTable
                        rowKey='keyLabel'
                        label='Etherscan Gateway probe API key summaries'
                        items={activeRun.keySummaries}
                        columns={keyProbeColumns}
                        scrollX={960}
                        rowClassName={probeSummaryRowClassName}
                    />
                </Section>
            )}
            {hasProbeResult && activeRun && activeRun.samples.length > 0 && (
                <Section title='Error Samples'>
                    <Space orientation='vertical' className='etherscan-probe-samples'>
                        {activeRun.samples.map(sample => {
                            const tone = sampleTone(sample);
                            return (
                                <div key={sample} className={`etherscan-probe-sample ${toneClass(tone)}`}>
                                    <TonePill tone={tone}>{tone === 'warn' ? 'rate limit' : tone === 'bad' ? 'error' : 'sample'}</TonePill>
                                    <TruncatedText value={sample} copyable={true} />
                                </div>
                            );
                        })}
                    </Space>
                </Section>
            )}
        </AppPage>
    );
};
