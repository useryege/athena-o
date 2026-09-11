import {Alert, Space} from 'antd';
import {Link, useParams} from 'react-router-dom';
import {AppPage, KeyValueGrid, Section} from '../../../components';
import {useVisibleQuery} from '../../../shared/use-visible-query';
import {useAdminReadScope} from '../../read-scope';
import {adminServices as services} from '../../services';
import {deliveryCounts, summaryLifecycle, summaryStatus, summaryTime, SummaryStatus} from './subscriptions';

export const TraderSyncAdminSubscriptionPage = () => {
    const {id = ''} = useParams();
    const data = useVisibleQuery(() => services.adminTraderSync.getSubscriptionSummary(id), useAdminReadScope(`trader-sync-subscription:${id}`), 10000);
    const item = data.data,
        interruption = item?.observation.latestInterruption;
    return (
        <AppPage
            title='Subscription summary'
            subtitle='Administrator overview · Times in UTC+8'
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}
            extra={
                <Space wrap={true}>
                    <Link to='/trader-sync/subscriptions'>Subscriptions</Link>
                    <Link to='/service-status'>Service Status</Link>
                </Space>
            }
        >
            {data.stale && <Alert type='warning' title='Stale summary — showing the last successful read' />}
            {item && (
                <>
                    <Section title='Subscription'>
                        <KeyValueGrid
                            items={[
                                {label: 'Subscription ID', value: item.subscriptionId},
                                {label: 'Account ID', value: item.accountId},
                                {label: 'Username', value: item.username || 'Unavailable'},
                                {label: 'Email', value: item.email || 'Unavailable'},
                                {label: 'Wallet', value: item.wallet},
                                {
                                    label: 'Status',
                                    value: <SummaryStatus state={item.status} />
                                },
                                ...summaryLifecycle(item),
                                {label: 'As of', value: summaryTime(item.asOf)},
                                {label: 'Activities (lifetime)', value: item.activityCount ?? 'Unavailable'},
                                {label: 'Associated deliveries', value: deliveryCounts(item.associatedDeliveryCounts)}
                            ]}
                        />
                        <p>
                            Associated deliveries are distinct logical deliveries linked to this subscription. Do not add counts across subscriptions: a summary delivery can be
                            linked to more than one.
                        </p>
                    </Section>
                    <Section title='Observation'>
                        <KeyValueGrid
                            items={[
                                {label: 'Current observation', value: summaryStatus(item.observation.state)},
                                {label: 'Current reason', value: item.observation.reason || 'None reported'},
                                {label: 'Last reliable checkpoint', value: summaryTime(item.observation.lastReliableAt)},
                                {label: 'Recorded interruptions', value: item.observation.interruptionCount ?? 'Unavailable'}
                            ]}
                        />
                        <p>
                            The reliable checkpoint records persisted monitoring coverage in the current activation. It is not the last trade time; no trades does not mean
                            monitoring failed.
                        </p>
                    </Section>
                    {interruption && (
                        <Section title='Latest recorded interruption'>
                            <KeyValueGrid
                                items={[
                                    {label: 'Reason', value: interruption.reason || 'Unknown'},
                                    {label: 'Start', value: summaryTime(interruption.start)},
                                    {label: 'End', value: summaryTime(interruption.end)},
                                    {label: 'Recovered at', value: summaryTime(interruption.recoveredAt)},
                                    {label: 'Uncertainty', value: interruption.uncertainty || 'None reported'},
                                    {label: 'Possibly missing activities', value: interruption.possibleMissing ? 'Yes' : 'No'}
                                ]}
                            />
                            <p>
                                An unknown historical endpoint does not imply a current interruption. A later manual activation is separate from recovery in the earlier activation.
                                Missing periods are not backfilled.
                            </p>
                        </Section>
                    )}
                </>
            )}
        </AppPage>
    );
};
