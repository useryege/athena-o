import {Alert, Spin, Tag} from 'antd';
import * as React from 'react';
import {AppPage, KeyValueGrid, ResourceTable, Section, StatusTag, useAsyncData} from '../../components';
import {formatBeijingDateTime, formatBeijingUnixSeconds} from '../../shared/format';
import {adminServices as services} from '../services';
import type {ServiceHealthStatus, ServiceStatus} from '../../shared/services/service-status-service';

const serviceLabels: Record<string, string> = {
    'notification': 'Notification',
    'wallet': 'Wallet',
    'market-radar': 'Market Radar',
    'sports-live': 'Sports Live',
    'sports-history': 'Sports History',
    'managed-oo': 'Managed OO',
    'worm-markets': 'Worm Markets',
    'worm-trading': 'Worm Trading',
    'profit-sharing': 'Profit Sharing',
    'token-api': 'Token API'
};

const statusTag = (status: ServiceHealthStatus) => (
    <StatusTag value={status} positive={status === 'SERVING'} negative={status === 'NOT_SERVING' || status === 'SERVICE_UNKNOWN' || status === 'UNREACHABLE'} />
);

const runtimeStatusTag = (status: string) => {
    const value = status.toLowerCase() || 'unknown';
    return <Tag color={value === 'running' ? 'green' : value === 'degraded' ? 'orange' : 'red'}>{value}</Tag>;
};

export const ServiceStatusPage = () => {
    const data = useAsyncData(() => services.serviceStatus.list(), []);
    const notificationRuntime = useAsyncData(() => services.adminNotifications.getRuntimeStatus(), []);
    const reloadAll = () => {
        data.reload();
        notificationRuntime.reload();
    };
    const reloadRef = React.useRef(reloadAll);
    reloadRef.current = reloadAll;

    React.useEffect(() => {
        const timer = window.setInterval(() => reloadRef.current(), 10000);
        return () => window.clearInterval(timer);
    }, []);

    const checkedAt = formatBeijingUnixSeconds(data.data?.checkedAt) || 'Not checked';
    const runtime = notificationRuntime.data;
    const botUsername = (runtime?.botUsername || '').replace(/^@+/, '');
    const botIdentity = runtime?.botAvailable
        ? [botUsername ? `@${botUsername}` : 'Telegram bot', runtime.botId ? `ID ${runtime.botId}` : ''].filter(Boolean).join(' · ')
        : 'Unavailable';
    return (
        <AppPage
            title='Service Status'
            subtitle={`gRPC health of Athena services · Last checked ${checkedAt}`}
            loading={data.loading || notificationRuntime.loading}
            error={data.error}
            onRefresh={reloadAll}>
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
            <Section
                title='Notification Runtime'
                extra={
                    notificationRuntime.loading ? (
                        <span role='status' aria-label='Refreshing notification runtime'>
                            <Spin size='small' />
                        </span>
                    ) : runtime ? (
                        runtimeStatusTag(runtime.status)
                    ) : null
                }>
                {notificationRuntime.error && <Alert type='error' showIcon={true} title='Notification runtime unavailable' description={notificationRuntime.error.message} />}
                {runtime && (
                    <KeyValueGrid
                        columns={3}
                        items={[
                            {label: 'Runtime', value: runtimeStatusTag(runtime.status)},
                            {label: 'Started', value: <StatusTag value={runtime.started ? 'Yes' : 'No'} positive={runtime.started} negative={!runtime.started} />},
                            {
                                label: 'Bot available',
                                value: <StatusTag value={runtime.botAvailable ? 'Yes' : 'No'} positive={runtime.botAvailable} negative={!runtime.botAvailable} />
                            },
                            {label: 'Bot identity', value: botIdentity},
                            {
                                label: 'Poller',
                                value: <StatusTag value={runtime.pollerActive ? 'Active' : 'Inactive'} positive={runtime.pollerActive} negative={!runtime.pollerActive} />
                            },
                            {label: 'Last poll', value: formatBeijingDateTime(runtime.lastPollAt) || 'Never'},
                            {label: 'Last update', value: formatBeijingDateTime(runtime.lastUpdateAt) || 'Never'},
                            {label: 'System pending', value: runtime.systemPendingCount},
                            {label: 'System retry', value: runtime.systemRetryCount},
                            {label: 'System failed', value: runtime.systemFailedCount},
                            {label: 'System sending', value: runtime.systemSendingCount},
                            {label: 'System unknown', value: runtime.systemUnknownCount},
                            {label: 'Account pending', value: runtime.accountPendingCount},
                            {label: 'Account retry', value: runtime.accountRetryCount},
                            {label: 'Account failed', value: runtime.accountFailedCount},
                            {label: 'Account sending', value: runtime.accountSendingCount},
                            {label: 'Account unknown', value: runtime.accountUnknownCount},
                            {label: 'Unreachable bindings', value: runtime.unreachableBindingCount}
                        ]}
                    />
                )}
            </Section>
        </AppPage>
    );
};
