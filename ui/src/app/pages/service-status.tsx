import * as React from 'react';
import {AppPage, ResourceTable, Section, StatusTag, useAsyncData} from '../components';
import {formatBeijingUnixSeconds} from '../shared/format';
import {services, ServiceHealthStatus, ServiceStatus} from '../shared/services';

const serviceLabels: Record<string, string> = {
    'notification': 'Notification',
    'wallet': 'Wallet',
    'market-radar': 'Market Radar',
    'sports-live': 'Sports Live',
    'sports-history': 'Sports History',
    'managed-oo': 'Managed OO',
    'worm-markets': 'Worm Markets',
    'profit-sharing': 'Profit Sharing',
    'token-api': 'Token API'
};

const statusTag = (status: ServiceHealthStatus) => (
    <StatusTag value={status} positive={status === 'SERVING'} negative={status === 'NOT_SERVING' || status === 'SERVICE_UNKNOWN' || status === 'UNREACHABLE'} />
);

export const ServiceStatusPage = () => {
    const data = useAsyncData(() => services.serviceStatus.list(), []);
    const reloadRef = React.useRef(data.reload);
    reloadRef.current = data.reload;

    React.useEffect(() => {
        const timer = window.setInterval(() => reloadRef.current(), 10000);
        return () => window.clearInterval(timer);
    }, []);

    const checkedAt = formatBeijingUnixSeconds(data.data?.checkedAt) || 'Not checked';
    return (
        <AppPage title='Service Status' subtitle={`gRPC health of Athena services · Last checked ${checkedAt}`} error={data.error} onRefresh={data.reload}>
            <Section title='Services'>
                <ResourceTable<ServiceStatus>
                    rowKey='name'
                    items={data.data?.items || []}
                    loading={data.loading}
                    columns={[
                        {title: 'Service', render: item => serviceLabels[item.name] || item.name},
                        {title: 'Status', render: item => statusTag(item.status)},
                        {title: 'Error', dataIndex: 'errorMessage'}
                    ]}
                />
            </Section>
        </AppPage>
    );
};
