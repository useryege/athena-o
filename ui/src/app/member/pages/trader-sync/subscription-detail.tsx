import * as React from 'react';
import {Alert, Button} from 'antd';
import {Link, useParams} from 'react-router-dom';
import {AppPage, Section} from '../../../components';
import {requestErrorDetails, requestErrorMessage} from '../../../shared/services/requests';
import {useVisibleQuery} from '../../../shared/use-visible-query';
import {memberServices as services} from '../../services';
import type {Subscription} from '../../trader-sync-models';
import {captureTraderSyncScope, readHistorySession} from './state';
import {SubscriptionControls, SubscriptionIdentity, subscriptionStatusLabel, subscriptionTime} from './subscription-state';
import {ObservationHistory} from './observation-history';
export const TraderSyncSubscriptionPage = ({ownerId}: {ownerId: string}) => {
    const {subscriptionId = ''} = useParams();
    return <SubscriptionDetail key={JSON.stringify([ownerId, subscriptionId])} ownerId={ownerId} subscriptionId={subscriptionId} />;
};
const SubscriptionDetail = ({ownerId, subscriptionId}: {ownerId: string; subscriptionId: string}) => {
    const scope = React.useMemo(() => captureTraderSyncScope(ownerId), [ownerId]);
    const [subscription, setSubscription] = React.useState<Subscription>();
    const latest = React.useRef<Subscription>();
    const mutationVersion = React.useRef(0);
    const query = useVisibleQuery(
        () => {
            const version = mutationVersion.current;
            const request = services.traderSync.getSubscription(subscriptionId);
            return Object.assign(
                request.then(
                    value => (version === mutationVersion.current ? value : latest.current || value),
                    error => {
                        throw Object.assign(new Error(requestErrorMessage(error)), {code: requestErrorDetails(error).code});
                    }
                ),
                {abort: () => request.abort?.()}
            );
        },
        {...scope, key: JSON.stringify([scope.key, 'subscription', subscriptionId])},
        5000
    );
    const update = (value: Subscription) => {
        if (!scope.isCurrent()) return;
        mutationVersion.current++;
        latest.current = value;
        setSubscription(value);
    };
    React.useEffect(() => {
        if (query.data) update(query.data);
    }, [query.data]);
    React.useLayoutEffect(() => {
        const unsubscribe = scope.subscribeInvalidation?.(() => {
            latest.current = undefined;
            setSubscription(undefined);
        });
        return () => unsubscribe?.();
    }, [scope]);
    const restored = React.useRef(false);
    React.useEffect(() => {
        if (subscription && !restored.current) {
            restored.current = true;
            const scrollY = readHistorySession(ownerId, subscriptionId)?.scrollY;
            if (scrollY) window.scrollTo({top: scrollY, behavior: 'instant'});
        }
    }, [subscription]);
    const item = scope.isCurrent() ? subscription : undefined;
    return (
        <AppPage
            title='Subscription'
            subtitle='Monitoring state, retained wallet note and observation history.'
            extra={<Link to='/trader-sync/subscriptions'>Back to subscriptions</Link>}>
            {query.error && (
                <Alert
                    type='error'
                    title={
                        (query.error as Error & {code?: number}).code === 5
                            ? 'Subscription not found'
                            : query.stale
                              ? 'Subscription data may be out of date'
                              : 'Could not load subscription'
                    }
                    description={query.error.message}
                    action={<Button onClick={query.reload}>Refresh subscription</Button>}
                />
            )}
            {!item && query.loading && <p role='status'>Loading subscription…</p>}
            {item && (
                <>
                    <Section title='Trader'>
                        <SubscriptionIdentity wallet={item.wallet} note={item.note} display={item.targetDisplay} />
                    </Section>
                    <Section title='Current monitoring state'>
                        <h3>{subscriptionStatusLabel(item.status)}</h3>
                        <p>{item.observation.reason || 'No additional monitoring reason reported.'}</p>
                        <p>Effective from: {subscriptionTime(item.currentInterval?.effectiveAt)}</p>
                        <p>Last reliable observation: {subscriptionTime(item.observation.lastReliableAt)}</p>
                        <p>Related persistent interruption records: {item.observation.interruptionCount}</p>
                        {item.status === 'paused' && <p>Paused at: {subscriptionTime(item.pausedAt)}</p>}
                        {item.status === 'cancelled' && <p>Cancelled at: {subscriptionTime(item.cancelledAt)}</p>}
                        {item.status === 'permission_disabled' && (
                            <p>Disabled at: {subscriptionTime(item.permissionDisabledAt)}. Access restoration does not resume this subscription automatically.</p>
                        )}
                        <p>Subscription updated: {subscriptionTime(item.updatedAt)}</p>
                    </Section>
                    <Section title='Note and subscription actions'>
                        <SubscriptionControls ownerId={ownerId} scope={scope} subscription={item} onUpdated={update} editNote={true} />
                    </Section>
                    <ObservationHistory ownerId={ownerId} subscriptionId={subscriptionId} />
                    <Section title='Activities and old notifications'>
                        <p>{item.queueNotice}</p>
                        <p>
                            Queued {item.queueCounts.pending}; sending {item.queueCounts.sending}; sent {item.queueCounts.sent}; failed {item.queueCounts.failed}; unknown{' '}
                            {item.queueCounts.unknown}; cancelled {item.queueCounts.cancelled}
                        </p>
                        <p>Current Telegram binding: {item.bindingStatus}. Old queued notifications may still arrive later.</p>
                        <p>
                            <Link to={`/trader-sync?subscriptionId=${encodeURIComponent(subscriptionId)}`}>View activities and notification results</Link>
                        </p>
                        <Link to='/notifications'>Open Notifications</Link>
                    </Section>
                </>
            )}
        </AppPage>
    );
};
