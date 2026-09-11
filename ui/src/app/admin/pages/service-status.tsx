import './trader-sync/trader-sync.css';
import {Alert, Spin, Tag} from 'antd';
import {Link} from 'react-router-dom';
import {useVisibleQuery, type AbortablePromise} from '../../shared/use-visible-query';
import {useAdminReadScope} from '../read-scope';
import type {RuntimeMetric} from '../trader-sync-models';
import {AppPage, KeyValueGrid, ResourceTable, Section, StatusTag} from '../../components';
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
    return (
        <Tag className='trader-sync-runtime-status' color={value === 'running' ? 'green' : value === 'degraded' || value === 'recovering' ? 'orange' : 'red'}>
            {value}
        </Tag>
    );
};

const receivedAt = <T,>(request: AbortablePromise<T>) =>
    Object.assign(
        request.then(value => ({value, receivedAt: Date.now()})),
        {abort: () => request.abort?.()}
    );
const metricScope = (metric: RuntimeMetric) =>
    metric.kind === 'window'
        ? `Window: ${formatBeijingDateTime(metric.windowStart) || 'Unavailable'} – ${formatBeijingDateTime(metric.windowEnd) || 'Unavailable'} (UTC+8)`
        : metric.kind === 'epoch'
          ? `Service epoch: ${metric.serviceEpoch || 'Unavailable'}`
          : `Current gauge${metric.serviceEpoch ? ` · Service epoch: ${metric.serviceEpoch}` : ''}`;

export const ServiceStatusPage = () => {
    const data = useVisibleQuery(() => services.serviceStatus.list(), useAdminReadScope('service-status'), 10000);
    const notificationRuntime = useVisibleQuery(() => receivedAt(services.adminNotifications.getRuntimeStatus()), useAdminReadScope('notification-runtime'), 10000);
    const traderRuntime = useVisibleQuery(() => services.adminTraderSync.getRuntimeStatus(), useAdminReadScope('trader-sync-runtime'), 10000);
    const reloadAll = () => {
        data.reload();
        notificationRuntime.reload();
        traderRuntime.reload();
    };

    const checkedAt = formatBeijingUnixSeconds(data.data?.checkedAt) || 'Not checked';
    const runtime = notificationRuntime.data?.value;
    const recovery = runtime?.recovery;
    const notificationStatus =
        runtime && ['failed', 'stopped'].includes(runtime.status)
            ? runtime.status
            : recovery && ['initializing', 'waiting'].includes(recovery.state)
              ? 'recovering'
              : runtime?.status || 'unknown';
    const trader = traderRuntime.data;
    const botUsername = (runtime?.botUsername || '').replace(/^@+/, '');
    const botIdentity = runtime?.botAvailable
        ? [botUsername ? `@${botUsername}` : 'Telegram bot', runtime.botId ? `ID ${runtime.botId}` : ''].filter(Boolean).join(' · ')
        : 'Unavailable';
    return (
        <AppPage
            title='Service Status'
            subtitle={`gRPC health of Athena services · Last checked ${checkedAt}`}
            loading={data.loading || notificationRuntime.loading || traderRuntime.loading}
            error={data.error}
            onRefresh={reloadAll}
        >
            <Section title='Services'>
                {data.stale && <Alert type='warning' title='Stale service health — showing the last successful read' />}
                <p>Last checked: {checkedAt} (UTC+8)</p>
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
                        runtimeStatusTag(notificationStatus)
                    ) : null
                }
            >
                {notificationRuntime.error && <Alert type='error' showIcon={true} title='Notification runtime unavailable' description={notificationRuntime.error.message} />}
                {notificationRuntime.stale && <Alert type='warning' title='Stale notification runtime — showing the last successful read' />}
                <p>Last received: {notificationRuntime.data ? formatBeijingDateTime(new Date(notificationRuntime.data.receivedAt).toISOString()) : 'Unavailable'} (UTC+8)</p>
                {recovery && (
                    <KeyValueGrid
                        items={[
                            {label: 'Recovery', value: recovery.state || 'Unavailable'},
                            {label: 'Recovery reason', value: recovery.reason || 'Unavailable'},
                            {label: 'Recovery started', value: formatBeijingDateTime(recovery.startedAt) || 'Unavailable'},
                            {label: 'Recovery remaining', value: recovery.remainingMillis === undefined ? 'Unavailable' : `${recovery.remainingMillis} ms`},
                            {label: 'Recovery elapsed', value: recovery.elapsedMillis === undefined ? 'Unavailable' : `${recovery.elapsedMillis} ms`},
                            {label: 'Recovery clock', value: recovery.clockSource || 'Unavailable'}
                        ]}
                    />
                )}
                {recovery && (
                    <p>
                        Recovery timing is reported by the notification process at the last read. Remaining time includes any active sending barrier or Retry-After wait; this page
                        does not count it down.
                    </p>
                )}
                {runtime && (
                    <KeyValueGrid
                        columns={3}
                        items={[
                            {label: 'Runtime', value: runtimeStatusTag(notificationStatus)},
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
            <Section title='Trader Sync' extra={<Link to='/trader-sync/subscriptions'>Subscription summaries</Link>}>
                {traderRuntime.error && <Alert type='error' showIcon={true} title='Trader Sync runtime unavailable' description={traderRuntime.error.message} />}
                {traderRuntime.stale && <Alert type='warning' title='Stale Trader Sync runtime — showing the last successful read' />}
                <p>As of: {formatBeijingDateTime(trader?.asOf) || 'Unavailable'} (UTC+8)</p>
                {trader && (
                    <>
                        <KeyValueGrid
                            items={[
                                {label: 'Collector', value: trader.collectorConnected === undefined ? 'Unavailable' : trader.collectorConnected ? 'Connected' : 'Disconnected'},
                                {label: 'Collector epoch', value: trader.collectorEpoch || 'Unavailable'},
                                {label: 'Filter revision', value: trader.filterRevision || 'Unavailable'},
                                {
                                    label: 'Raw observation',
                                    value: trader.metrics.find(metric => metric.name === 'raw_observation_available')?.value === '1' ? 'Available' : 'Not observable'
                                },
                                {label: 'Raw queue depth (raw logs)', value: trader.metrics.find(metric => metric.name === 'raw_queue_depth')?.value ?? 'Not observable'},
                                {
                                    label: 'Raw persist in flight (raw logs)',
                                    value: trader.metrics.find(metric => metric.name === 'raw_persist_in_flight')?.value ?? 'Not observable'
                                }
                            ]}
                        />
                        <p>
                            Each metric has its own unit and scope. Window counts cover only the stated interval; epoch counts belong to the named service instance. Do not add
                            metrics across units, windows or service instances.
                        </p>
                        <ResourceTable<RuntimeMetric>
                            rowKey={metric => [metric.name, metric.kind, metric.serviceEpoch, metric.windowStart, metric.windowEnd].join(':')}
                            items={trader.metrics}
                            label='Trader Sync runtime metrics'
                            columns={[
                                {title: 'Metric', render: metric => <span className='break-value'>{metric.name}</span>},
                                {title: 'Value', render: metric => metric.value ?? 'Not observable'},
                                {title: 'Unit', render: metric => <span className='break-value'>{metric.unit || 'Unavailable'}</span>},
                                {title: 'Scope', render: metric => metricScope(metric)}
                            ]}
                            compactRender={metric => (
                                <KeyValueGrid
                                    columns={1}
                                    items={[
                                        {label: 'Metric', value: metric.name},
                                        {label: 'Value', value: metric.value ?? 'Not observable'},
                                        {label: 'Unit', value: metric.unit || 'Unavailable'},
                                        {label: 'Scope', value: metricScope(metric)}
                                    ]}
                                />
                            )}
                        />
                    </>
                )}
            </Section>
        </AppPage>
    );
};
