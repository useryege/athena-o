import {Empty} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {AppPage, MetricRow, ResourceTable, Section, StatusTag, TruncatedText, useAsyncData} from '../components';
import {services} from '../shared/services';
import type {EtherscanGatewayStatus} from '../shared/services';

const formatUnixSeconds = (value?: number) => (value ? new Date(value * 1000).toLocaleString() : '-');
const formatLatency = (value?: number) => (value === undefined ? '-' : `${value} ms`);

const isRunning = (item: EtherscanGatewayStatus) => item.reachable && item.started && item.status === 'running';
const isRuntimeError = (item: EtherscanGatewayStatus) => !isRunning(item) && item.status !== 'unreachable';

const runtimeTag = (item: EtherscanGatewayStatus) => (
    <StatusTag value={item.status || '-'} positive={isRunning(item)} negative={item.status === 'unreachable' || isRuntimeError(item)} />
);

export const EtherscanGatewaysPage = () => {
    const data = useAsyncData(() => services.serviceStatus.listEtherscanGatewayStatuses(), []);
    const reloadRef = React.useRef(data.reload);
    reloadRef.current = data.reload;

    React.useEffect(() => {
        const timer = window.setInterval(() => reloadRef.current(), 10000);
        return () => window.clearInterval(timer);
    }, []);

    const items = data.data?.items || [];
    const checkedAt = data.data?.checkedAt ? new Date(data.data.checkedAt * 1000).toLocaleString() : 'Not checked';
    const running = items.filter(isRunning).length;
    const unreachable = items.filter(item => item.status === 'unreachable').length;
    const errors = items.filter(isRuntimeError).length;

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
        </AppPage>
    );
};
