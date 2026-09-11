import * as React from 'react';
import {Alert, Button, Checkbox, Input, Select, Space} from 'antd';
import {Link} from 'react-router-dom';
import {AppPage, KeyValueGrid, ResourceTable, Section, StatusTag} from '../../../components';
import {formatBeijingDateTime} from '../../../shared/format';
import {useVisibleQuery} from '../../../shared/use-visible-query';
import {useAdminReadScope} from '../../read-scope';
import {adminServices as services} from '../../services';
import type {Counts, SubscriptionState, SubscriptionSummary} from '../../trader-sync-models';

export const summaryStatus = (state: string) => (state === 'healthy' ? 'Monitoring' : state === 'interrupted' ? 'Monitoring interrupted' : state);
export const summaryTime = (value?: string) => formatBeijingDateTime(value) || 'Unknown';
export const deliveryCounts = (counts: Counts) =>
    ['total', 'pending', 'sending', 'sent', 'failed', 'unknown', 'cancelled'].map(key => `${key}: ${counts[key as keyof Counts] ?? 'Unavailable'}`).join(' · ');
export const summaryLifecycle = (item: SubscriptionSummary) => [
    {label: 'Created', value: summaryTime(item.createdAt)},
    {label: 'Updated', value: summaryTime(item.updatedAt)},
    ...(item.status === 'paused' ? [{label: 'Paused', value: summaryTime(item.pausedAt)}] : []),
    ...(item.status === 'cancelled' ? [{label: 'Cancelled', value: summaryTime(item.cancelledAt)}] : []),
    ...(item.status === 'permission_disabled' ? [{label: 'Permission disabled', value: summaryTime(item.permissionDisabledAt)}] : [])
];
const status = (item: SubscriptionSummary) => <StatusTag value={summaryStatus(item.status)} positive={item.status === 'healthy'} negative={item.status === 'interrupted'} />;
const identity = (item: SubscriptionSummary) => (
    <div className='break-value'>
        <div>{item.username || 'Unavailable'}</div>
        <div>{item.email || 'Unavailable'}</div>
        <div>{item.accountId}</div>
    </div>
);
const target = (item: SubscriptionSummary) => (
    <Link className='break-value' to={`/trader-sync/subscriptions/${encodeURIComponent(item.subscriptionId)}`}>
        {item.wallet}
    </Link>
);
const states: SubscriptionState[] = ['pending_baseline', 'healthy', 'interrupted', 'paused', 'permission_disabled', 'cancelled'];
const emptyFilters = {accountId: '', wallet: '', state: '', includeCancelled: false};

export const TraderSyncAdminSubscriptionsPage = () => {
    const [draft, setDraft] = React.useState(emptyFilters);
    const [filters, setFilters] = React.useState(emptyFilters);
    const [cursors, setCursors] = React.useState<Array<string | undefined>>([undefined]);
    const input = {
        ...filters,
        accountId: filters.accountId || undefined,
        wallet: filters.wallet || undefined,
        state: filters.state || undefined,
        pageSize: 50,
        cursor: cursors[cursors.length - 1]
    };
    const data = useVisibleQuery(() => services.adminTraderSync.listSubscriptionSummaries(input), useAdminReadScope(`trader-sync-subscriptions:${JSON.stringify(input)}`), 10000);
    const apply = (event: React.FormEvent) => {
        event.preventDefault();
        setFilters({...draft, accountId: draft.accountId.trim(), wallet: draft.wallet.trim().toLowerCase()});
        setCursors([undefined]);
    };
    return (
        <AppPage
            title='Trader Sync'
            subtitle='Administrator subscription summaries · Times in UTC+8'
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}
            extra={<Link to='/service-status'>Service Status</Link>}
            filters={
                <form onSubmit={apply}>
                    <Space wrap={true} align='end'>
                        <label>
                            Account ID
                            <Input aria-label='Account ID' value={draft.accountId} onChange={event => setDraft(current => ({...current, accountId: event.target.value}))} />
                        </label>
                        <label>
                            Canonical wallet
                            <Input aria-label='Canonical wallet' value={draft.wallet} onChange={event => setDraft(current => ({...current, wallet: event.target.value}))} />
                        </label>
                        <label>
                            Status
                            <Select
                                aria-label='Subscription status'
                                style={{minWidth: 190}}
                                value={draft.state}
                                onChange={state => setDraft(current => ({...current, state}))}
                                options={[{value: '', label: 'All states'}, ...states.map(value => ({value, label: summaryStatus(value)}))]}
                            />
                        </label>
                        <Checkbox checked={draft.includeCancelled} onChange={event => setDraft(current => ({...current, includeCancelled: event.target.checked}))}>
                            Include cancelled
                        </Checkbox>
                        <Button htmlType='submit'>Apply filters</Button>
                    </Space>
                </form>
            }>
            {data.stale && <Alert type='warning' title='Stale summaries — showing the last successful read' />}
            <Section title={filters.includeCancelled ? 'Current and cancelled subscriptions' : 'Current subscriptions'}>
                <p>As of: {summaryTime(data.data?.asOf)} (UTC+8)</p>
                <p>
                    Activity counts cover each subscription’s lifetime. Associated deliveries count distinct logical deliveries linked to that subscription. Do not add counts
                    across rows: one summary delivery can be associated with several subscriptions.
                </p>
                <ResourceTable<SubscriptionSummary>
                    rowKey='subscriptionId'
                    label='Trader Sync subscription summaries'
                    items={data.data?.summaries || []}
                    loading={data.loading}
                    columns={[
                        {title: 'User', render: identity},
                        {title: 'Wallet', render: target},
                        {
                            title: 'Lifecycle',
                            render: item => (
                                <>
                                    {status(item)}
                                    {summaryLifecycle(item).map(fact => (
                                        <div key={fact.label}>
                                            {fact.label}: {fact.value}
                                        </div>
                                    ))}
                                </>
                            )
                        },
                        {
                            title: 'Observation',
                            render: item => (
                                <>
                                    <div>{summaryStatus(item.observation.state)}</div>
                                    <div>{item.observation.reason}</div>
                                    <div>Last reliable checkpoint: {summaryTime(item.observation.lastReliableAt)}</div>
                                    <div>Interruptions: {item.observation.interruptionCount ?? 'Unavailable'}</div>
                                </>
                            )
                        },
                        {title: 'Activities', render: item => item.activityCount ?? 'Unavailable'},
                        {title: 'Associated deliveries', render: item => deliveryCounts(item.associatedDeliveryCounts)}
                    ]}
                    compactEmptyDescription='No subscriptions match these filters.'
                    compactRender={item => (
                        <KeyValueGrid
                            columns={1}
                            items={[
                                {label: 'User', value: identity(item)},
                                {label: 'Wallet', value: target(item)},
                                {label: 'Status', value: status(item)},
                                ...summaryLifecycle(item),
                                {label: 'Observation', value: summaryStatus(item.observation.state)},
                                {label: 'Reason', value: item.observation.reason || 'None reported'},
                                {label: 'Last reliable checkpoint', value: summaryTime(item.observation.lastReliableAt)},
                                {label: 'Interruptions', value: item.observation.interruptionCount ?? 'Unavailable'},
                                {label: 'Activities', value: item.activityCount ?? 'Unavailable'},
                                {label: 'Associated deliveries', value: deliveryCounts(item.associatedDeliveryCounts)}
                            ]}
                        />
                    )}
                />
                <Space wrap={true}>
                    <Button disabled={cursors.length === 1 || data.loading} onClick={() => setCursors(current => current.slice(0, -1))}>
                        Previous
                    </Button>
                    <Button
                        disabled={!data.data?.page.nextCursor || data.loading}
                        onClick={() => {
                            const next = data.data?.page.nextCursor;
                            if (next) setCursors(current => [...current, next]);
                        }}>
                        Next
                    </Button>
                </Space>
            </Section>
        </AppPage>
    );
};
