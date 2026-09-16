import {OperationFacts} from '../components/operation-facts';
import './trader-sync/trader-sync.css';
import {Alert, Button, Empty, Spin, Tabs, Tag} from 'antd';
import {Link} from 'react-router-dom';
import {useVisibleQuery, type AbortablePromise} from '../../shared/use-visible-query';
import {useAdminReadScope} from '../read-scope';
import type {RuntimeMetric} from '../trader-sync-models';
import {AppPage, ResourceTable, Section, StatusTag} from '../../components';
import {formatBeijingDateTime, formatBeijingUnixSeconds} from '../../shared/format';
import {adminServices as services} from '../services';
import type {ServiceHealthStatus} from '../../shared/services/service-status-service';

const serviceLabels: Record<string, string> = {
    'notification': 'Notification',
    'wallet': 'Wallet',
    'market-radar': 'Market Radar',
    'managed-oo': 'Managed OO',
    'worm-markets': 'Worm Markets',
    'worm-trading': 'Worm Trading',
    'profit-sharing': 'Profit Sharing',
    'token-api': 'Token API'
};

const statusTag = (status: ServiceHealthStatus) => (
    <StatusTag
        value={status
            .toLowerCase()
            .replace(/_/g, ' ')
            .replace(/^./, letter => letter.toUpperCase())}
        positive={status === 'SERVING'}
        negative={status === 'NOT_SERVING' || status === 'SERVICE_UNKNOWN' || status === 'UNREACHABLE'}
    />
);

const runtimeStatusTag = (status: string) => {
    const value = status.toLowerCase() || 'unknown';
    return (
        <Tag className='trader-sync-runtime-status' color={value === 'running' ? 'success' : value === 'degraded' || value === 'recovering' ? 'warning' : 'error'}>
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
        <AppPage title='Service Status' subtitle='Service health, notification delivery and collection activity.' onRefresh={reloadAll}>
            <Tabs
                className='admin-source-tabs'
                defaultActiveKey='services'
                items={[
                    {
                        key: 'services',
                        label: (
                            <span>
                                Services
                                <small>
                                    {data.loading
                                        ? 'Loading'
                                        : data.error
                                          ? 'Unavailable'
                                          : `${(data.data?.items || []).filter(item => item.status !== 'SERVING').length} needs attention`}
                                </small>
                            </span>
                        ),
                        forceRender: true,
                        children: (
                            <Section
                                title='Services'
                                extra={<span>{data.data ? `${data.data.items.length} services` : data.loading ? 'Loading services' : 'Service count unavailable'}</span>}>
                                {data.error && (
                                    <Alert
                                        type='error'
                                        showIcon
                                        title='Service health unavailable'
                                        description={data.error.message}
                                        action={<Button onClick={data.reload}>Retry</Button>}
                                    />
                                )}
                                {data.stale && <Alert type='warning' title='Stale service health — showing the last successful read' />}
                                <p>Last checked: {checkedAt} (UTC+8)</p>
                                {Boolean(data.data?.items.length) && (
                                    <div className='service-health-container'>
                                        <table className='service-health-table' aria-label='Service health'>
                                            <thead>
                                                <tr>
                                                    <th>Service</th>
                                                    <th>Status</th>
                                                    <th>Error</th>
                                                </tr>
                                            </thead>
                                            <tbody>
                                                {(data.data?.items || []).map(item => (
                                                    <tr key={item.name}>
                                                        <th scope='row'>{serviceLabels[item.name] || item.name}</th>
                                                        <td className='service-health-status'>{statusTag(item.status)}</td>
                                                        <td className='service-health-error' data-empty={!item.errorMessage || undefined}>
                                                            {item.errorMessage || '—'}
                                                        </td>
                                                    </tr>
                                                ))}
                                            </tbody>
                                        </table>
                                    </div>
                                )}
                                {!data.loading && data.data && !data.data.items.length && <Empty description='No service health records' />}
                                <p className='admin-source-note'>gRPC health checks · Service connectivity is separate from runtime activity.</p>
                            </Section>
                        )
                    },
                    {
                        key: 'notifications',
                        label: (
                            <span>
                                Notifications<small>{notificationRuntime.loading ? 'Loading' : notificationRuntime.error ? 'Unavailable' : notificationStatus}</small>
                            </span>
                        ),
                        forceRender: true,
                        children: (
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
                                }>
                                {notificationRuntime.error && (
                                    <Alert
                                        type='error'
                                        showIcon={true}
                                        title='Notification runtime unavailable'
                                        description={notificationRuntime.error.message}
                                        action={<Button onClick={notificationRuntime.reload}>Retry</Button>}
                                    />
                                )}
                                {notificationRuntime.stale && <Alert type='warning' title='Stale notification runtime — showing the last successful read' />}
                                <p>
                                    Last received: {notificationRuntime.data ? formatBeijingDateTime(new Date(notificationRuntime.data.receivedAt).toISOString()) : 'Unavailable'}{' '}
                                    (UTC+8)
                                </p>
                                {recovery && ['initializing', 'waiting'].includes(recovery.state) && (
                                    <Alert type='warning' showIcon title='Recovery in progress' description={`Recovery: ${recovery.state}`} />
                                )}
                                {recovery && (
                                    <OperationFacts
                                        items={[
                                            {label: 'Recovery', value: recovery.state || 'Unavailable'},

                                            {label: 'Recovery remaining', value: recovery.remainingMillis === undefined ? 'Unavailable' : `${recovery.remainingMillis} ms`},
                                            {label: 'Recovery elapsed', value: recovery.elapsedMillis === undefined ? 'Unavailable' : `${recovery.elapsedMillis} ms`}
                                        ]}
                                    />
                                )}
                                {recovery && (
                                    <details className='admin-operation-details'>
                                        <summary>Recovery details</summary>
                                        <OperationFacts
                                            items={[
                                                {label: 'Recovery reason', value: <span className='athena-identifier'>{recovery.reason || 'Unavailable'}</span>},
                                                {label: 'Recovery started', value: formatBeijingDateTime(recovery.startedAt) || 'Unavailable'},
                                                {label: 'Recovery clock', value: <span className='athena-identifier'>{recovery.clockSource || 'Unavailable'}</span>}
                                            ]}
                                        />
                                    </details>
                                )}
                                {recovery && (
                                    <p>
                                        Recovery timing is reported by the notification process at the last read. Remaining time includes any active sending barrier or Retry-After
                                        wait; this page does not count it down.
                                    </p>
                                )}
                                {runtime && (
                                    <>
                                        <h3>Delivery queues</h3>
                                        <table className='admin-queue-table' aria-label='Delivery queues'>
                                            <thead>
                                                <tr>
                                                    <th>State</th>
                                                    <th>System</th>
                                                    <th>Account</th>
                                                </tr>
                                            </thead>
                                            <tbody>
                                                {(['Pending', 'Retry', 'Failed', 'Sending', 'Unknown'] as const).map(state => (
                                                    <tr key={state}>
                                                        <th scope='row'>{state}</th>
                                                        <td>{runtime[`system${state}Count`]}</td>
                                                        <td>{runtime[`account${state}Count`]}</td>
                                                    </tr>
                                                ))}
                                            </tbody>
                                        </table>
                                        <h3>Runtime details</h3>
                                    </>
                                )}
                                {runtime && (
                                    <OperationFacts
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
                                                value: (
                                                    <StatusTag
                                                        value={runtime.pollerActive ? 'Active' : 'Inactive'}
                                                        positive={runtime.pollerActive}
                                                        negative={!runtime.pollerActive}
                                                    />
                                                )
                                            },
                                            {label: 'Last poll', value: formatBeijingDateTime(runtime.lastPollAt) || 'Never'},
                                            {label: 'Last update', value: formatBeijingDateTime(runtime.lastUpdateAt) || 'Never'},
                                            {label: 'Unreachable bindings', value: runtime.unreachableBindingCount}
                                        ]}
                                    />
                                )}
                            </Section>
                        )
                    },
                    {
                        key: 'trader',
                        label: (
                            <span>
                                Trader Sync
                                <small>
                                    {traderRuntime.loading
                                        ? 'Loading'
                                        : traderRuntime.error
                                          ? 'Unavailable'
                                          : trader?.collectorConnected === undefined
                                            ? 'Unavailable'
                                            : trader.collectorConnected
                                              ? 'Connected'
                                              : 'Disconnected'}
                                </small>
                            </span>
                        ),
                        forceRender: true,
                        children: (
                            <Section title='Trader Sync' extra={<Link to='/trader-sync/subscriptions'>Subscription summaries</Link>}>
                                {traderRuntime.error && (
                                    <Alert
                                        type='error'
                                        showIcon={true}
                                        title='Trader Sync runtime unavailable'
                                        description={traderRuntime.error.message}
                                        action={<Button onClick={traderRuntime.reload}>Retry</Button>}
                                    />
                                )}
                                {traderRuntime.stale && <Alert type='warning' title='Stale Trader Sync runtime — showing the last successful read' />}
                                <p>As of: {formatBeijingDateTime(trader?.asOf) || 'Unavailable'} (UTC+8)</p>
                                {trader && (
                                    <>
                                        <OperationFacts
                                            items={[
                                                {
                                                    label: 'Collector',
                                                    value: trader.collectorConnected === undefined ? 'Unavailable' : trader.collectorConnected ? 'Connected' : 'Disconnected'
                                                },
                                                {label: 'Collector epoch', value: trader.collectorEpoch || 'Unavailable'},
                                                {label: 'Filter revision', value: trader.filterRevision || 'Unavailable'},
                                                {
                                                    label: 'Raw observation',
                                                    value:
                                                        trader.metrics.find(metric => metric.name === 'raw_observation_available')?.value === '1' ? 'Available' : 'Not observable'
                                                },
                                                {
                                                    label: 'Raw queue depth (raw logs)',
                                                    value: trader.metrics.find(metric => metric.name === 'raw_queue_depth')?.value ?? 'Not observable'
                                                },
                                                {
                                                    label: 'Raw persist in flight (raw logs)',
                                                    value: trader.metrics.find(metric => metric.name === 'raw_persist_in_flight')?.value ?? 'Not observable'
                                                }
                                            ]}
                                        />
                                        <p>
                                            Each metric has its own unit and scope. Window counts cover only the stated interval; epoch counts belong to the named service instance.
                                            Do not add metrics across units, windows or service instances.
                                        </p>
                                        <ResourceTable<RuntimeMetric>
                                            rowKey={metric => [metric.name, metric.kind, metric.serviceEpoch, metric.windowStart, metric.windowEnd].join(':')}
                                            items={trader.metrics}
                                            label='Trader Sync runtime metrics'
                                            columns={[
                                                {title: 'Metric', render: metric => <span className='break-value'>{metric.name}</span>},
                                                {title: 'Value', className: 'athena-numeric-column', render: metric => metric.value ?? 'Not observable'},
                                                {title: 'Unit', render: metric => <span className='break-value'>{metric.unit || 'Unavailable'}</span>},
                                                {title: 'Scope', render: metric => metricScope(metric)}
                                            ]}
                                            compactRender={metric => (
                                                <OperationFacts
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
                        )
                    }
                ]}
            />
        </AppPage>
    );
};
