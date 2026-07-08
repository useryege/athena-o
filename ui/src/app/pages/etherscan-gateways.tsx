import {Button, Empty, Form, InputNumber, Progress, Space, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {AppPage, MetricRow, ResourceTable, Section, StatusTag, TruncatedText, useAsyncData} from '../components';
import {Context} from '../shared/context';
import {services} from '../shared/services';
import type {EtherscanGatewayProbeCounts, EtherscanGatewayProbeGatewaySummary, EtherscanGatewayProbeKeySummary, EtherscanGatewayProbeRun, EtherscanGatewayStatus} from '../shared/services';

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

const successRate = (counts?: EtherscanGatewayProbeCounts, fallbackTotal?: number) => {
    const total = fallbackTotal || countsTotal(counts);
    if (!counts || total <= 0) {
        return '-';
    }
    return `${((counts.success * 100) / total).toFixed(2)}%`;
};

const groupedAuthInvalid = (counts?: EtherscanGatewayProbeCounts) => (counts ? counts.authentication + counts.plan + counts.invalidRequest + counts.malformed : 0);

const resultTag = (run?: EtherscanGatewayProbeRun) => (
    <StatusTag
        value={run?.status === 'running' ? 'running' : run?.result || run?.status || 'idle'}
        positive={run?.result === 'pass'}
        negative={run?.result === 'fail' || run?.result === 'error' || run?.status === 'error'}
    />
);

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
        {title: 'Success', render: item => item.counts.success},
        {title: 'Rate Limit', render: item => item.counts.rateLimit},
        {title: 'Auth / Invalid', render: item => groupedAuthInvalid(item.counts)},
        {title: 'Upstream', render: item => item.counts.upstream},
        {title: 'Other', render: item => item.counts.other},
        {title: 'Success Rate', render: item => successRate(item.counts)}
    ];

    const keyProbeColumns: ColumnsType<EtherscanGatewayProbeKeySummary> = [
        {title: 'Key Label', render: item => <TruncatedText value={item.keyLabel} copyable={false} />},
        {title: 'Success', render: item => item.counts.success},
        {title: 'Rate Limit', render: item => item.counts.rateLimit},
        {title: 'Auth / Invalid', render: item => groupedAuthInvalid(item.counts)},
        {title: 'Upstream', render: item => item.counts.upstream},
        {title: 'Other', render: item => item.counts.other},
        {title: 'Success Rate', render: item => successRate(item.counts)}
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
                    <ResourceTable rowKey='address' label='Etherscan gateways' items={items} columns={columns} loading={data.loading} scrollX={1320} stickyHeader={true} />
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
                    <Space direction='vertical' size='middle' style={{width: '100%'}}>
                        <MetricRow
                            items={[
                                {label: 'Keys', value: activeRun.keyCount},
                                {label: 'Gateways', value: activeRun.gatewayCount},
                                {label: 'Total', value: activeRun.total},
                                {label: 'Required', value: activeRun.requiredSuccess},
                                {label: 'Success', value: activeRun.counts.success, tone: activeRun.result === 'pass' ? 'good' : activeRun.result === 'fail' ? 'bad' : undefined},
                                {label: 'Rate', value: successRate(activeRun.counts, activeRun.total), tone: activeRun.result === 'pass' ? 'good' : activeRun.result === 'fail' ? 'bad' : undefined}
                            ]}
                        />
                        <MetricRow
                            items={[
                                {label: 'RateLimit', value: activeRun.counts.rateLimit, tone: activeRun.counts.rateLimit > 0 ? 'warn' : undefined},
                                {label: 'Auth', value: activeRun.counts.authentication, tone: activeRun.counts.authentication > 0 ? 'bad' : undefined},
                                {label: 'Plan', value: activeRun.counts.plan, tone: activeRun.counts.plan > 0 ? 'bad' : undefined},
                                {label: 'Invalid', value: activeRun.counts.invalidRequest, tone: activeRun.counts.invalidRequest > 0 ? 'bad' : undefined},
                                {label: 'Upstream', value: activeRun.counts.upstream, tone: activeRun.counts.upstream > 0 ? 'bad' : undefined},
                                {label: 'Other', value: activeRun.counts.other, tone: activeRun.counts.other > 0 ? 'bad' : undefined}
                            ]}
                        />
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
                        {activeRun.total > 0 && <Progress percent={Number(((activeRun.counts.success * 100) / activeRun.total).toFixed(2))} status={activeRun.result === 'fail' || activeRun.result === 'error' ? 'exception' : activeRun.result === 'pass' ? 'success' : 'active'} />}
                        {activeRun.errorMessage && <Typography.Text type='danger'>{activeRun.errorMessage}</Typography.Text>}
                    </Space>
                </Section>
            )}
            {hasProbeResult && activeRun && (
                <Section title='By Gateway'>
                    <ResourceTable rowKey='gateway' label='Etherscan Gateway probe gateway summaries' items={activeRun.gatewaySummaries} columns={gatewayProbeColumns} scrollX={960} />
                </Section>
            )}
            {hasProbeResult && activeRun && (
                <Section title='By API Key'>
                    <ResourceTable rowKey='keyLabel' label='Etherscan Gateway probe API key summaries' items={activeRun.keySummaries} columns={keyProbeColumns} scrollX={960} />
                </Section>
            )}
            {hasProbeResult && activeRun && activeRun.samples.length > 0 && (
                <Section title='Error Samples'>
                    <Space direction='vertical' style={{width: '100%'}}>
                        {activeRun.samples.map(sample => (
                            <TruncatedText key={sample} value={sample} copyable={true} />
                        ))}
                    </Space>
                </Section>
            )}
        </AppPage>
    );
};
