import './trader-sync.css';
import * as React from 'react';
import {Alert, Button, Space} from 'antd';
import {Link} from 'react-router-dom';
import {AppPage, ResourceTable} from '../../../components';
import {requestErrorMessage} from '../../../shared/services/requests';
import {useVisibleQuery} from '../../../shared/use-visible-query';
import {memberServices as services} from '../../services';
import type {Subscription, SubscriptionPage} from '../../trader-sync-models';
import {blankCursorSession, captureTraderSyncScope, readSubscriptionSession, saveSubscriptionSession, SubscriptionListSession} from './state';
import {SubscriptionIdentity, subscriptionStatusLabel, subscriptionTime, subscriptionQueueNotice} from './subscription-state';
export const TraderSyncSubscriptionsPage = ({ownerId}: {ownerId: string}) => <Subscriptions key={ownerId} ownerId={ownerId} />;
const Subscriptions = ({ownerId}: {ownerId: string}) => {
    const scope = React.useMemo(() => captureTraderSyncScope(ownerId), [ownerId]);
    const [session, setSession] = React.useState<SubscriptionListSession>(
        () => readSubscriptionSession(ownerId) || {view: 'current', current: blankCursorSession<SubscriptionPage>(), cancelled: blankCursorSession<SubscriptionPage>()}
    );
    const sessionRef = React.useRef(session);
    const update = (next: SubscriptionListSession) => {
        if (!scope.isCurrent()) return;
        sessionRef.current = next;
        saveSubscriptionSession(ownerId, scope, next);
        setSession(next);
    };
    const current = session[session.view];
    const query = useVisibleQuery(
        () => {
            const request = services.traderSync.listSubscriptions({view: session.view, cursor: current.cursors[current.index], pageSize: 50});
            return Object.assign(
                request.catch(error => {
                    throw new Error(requestErrorMessage(error));
                }),
                {abort: () => request.abort?.()}
            );
        },
        {...scope, key: JSON.stringify([scope.key, 'subscriptions', session.view, current.cursors[current.index]])},
        5000
    );
    React.useEffect(() => {
        if (query.data && scope.isCurrent())
            update({...sessionRef.current, [session.view]: {...sessionRef.current[session.view], pages: {...sessionRef.current[session.view].pages, [current.index]: query.data}}});
    }, [query.data]);
    React.useLayoutEffect(() => {
        const saved = sessionRef.current[sessionRef.current.view].scrollY;
        if (saved > 0) window.scrollTo({top: saved, behavior: 'instant'});
        const unsubscribe = scope.subscribeInvalidation?.(() => {
            const empty: SubscriptionListSession = {view: 'current', current: blankCursorSession(), cancelled: blankCursorSession()};
            sessionRef.current = empty;
            setSession(empty);
        });
        return () => {
            const value = sessionRef.current;
            saveSubscriptionSession(ownerId, scope, {...value, [value.view]: {...value[value.view], scrollY: window.scrollY}});
            unsubscribe?.();
        };
    }, [scope]);
    React.useLayoutEffect(() => {
        if (scope.isCurrent()) window.scrollTo({top: sessionRef.current[session.view].scrollY, behavior: 'instant'});
    }, [session.view]);
    const page = scope.isCurrent() ? query.data || current.pages[current.index] : undefined;
    const identity = (item: Subscription) => <SubscriptionIdentity wallet={item.wallet} note={item.note} display={item.targetDisplay} />;
    const facts = (item: Subscription) => (
        <>
            <strong>{subscriptionStatusLabel(item.status)}</strong>
            <p>Effective: {subscriptionTime(item.currentInterval?.effectiveAt)}</p>
            <p>Last reliable observation: {subscriptionTime(item.observation.lastReliableAt)}</p>
        </>
    );
    const queue = (item: Subscription) => (
        <>
            <p>
                Queued {item.queueCounts.pending}; sending {item.queueCounts.sending}; sent {item.queueCounts.sent}; failed {item.queueCounts.failed}; unknown{' '}
                {item.queueCounts.unknown}; cancelled {item.queueCounts.cancelled}
            </p>
            <p>{subscriptionQueueNotice(item.queueNotice)}</p>
        </>
    );
    const details = (item: Subscription) => <Link to={`/trader-sync/subscriptions/${item.id}`}>View subscription</Link>;
    return (
        <AppPage title='Subscriptions' subtitle='Manage current traders and retained cancellation history.' extra={<Link to='/trader-sync/add'>Add trader</Link>}>
            <Space wrap={true}>
                {(['current', 'cancelled'] as const).map(view => (
                    <Button
                        key={view}
                        className='trader-sync-view'
                        aria-pressed={session.view === view}
                        type={session.view === view ? 'primary' : 'default'}
                        onClick={() => update({...session, [session.view]: {...current, scrollY: window.scrollY}, view})}
                    >
                        {view === 'current' ? 'Current' : 'Cancelled'}
                    </Button>
                ))}
            </Space>
            {query.error && (
                <Alert
                    type='error'
                    title={query.stale || page ? 'Subscription data may be out of date' : 'Could not load subscriptions'}
                    description={query.error.message}
                    action={<Button onClick={query.reload}>Retry</Button>}
                />
            )}
            {page && (
                <p>
                    {page.quota.used} / {page.quota.limit} current subscriptions. All current states use a slot. As of {subscriptionTime(page.asOf)}
                </p>
            )}
            {page && page.subscriptions.length === 0 && <p>{session.view === 'current' ? 'No current subscriptions. Add a trader to begin.' : 'No cancelled subscriptions.'}</p>}
            <ResourceTable<Subscription>
                rowKey='id'
                items={page?.subscriptions || []}
                loading={!page && query.loading}
                label='Subscriptions'
                columns={[
                    {title: 'Trader', key: 'identity', render: (_value, item) => identity(item)},
                    {title: 'Monitoring', key: 'state', render: (_value, item) => facts(item)},
                    {title: 'Old notifications', key: 'queue', render: (_value, item) => queue(item)},
                    {title: 'Manage', key: 'details', render: (_value, item) => details(item)}
                ]}
                compactRender={item => (
                    <div className='trader-sync-subscription-row'>
                        {identity(item)}
                        {facts(item)}
                        {queue(item)}
                        {details(item)}
                    </div>
                )}
                compactEmptyDescription={session.view === 'current' ? 'No current subscriptions. Add a trader to begin.' : 'No cancelled subscriptions.'}
            />
            <Space wrap={true}>
                <Button disabled={current.index === 0 || query.loading} onClick={() => update({...session, [session.view]: {...current, index: current.index - 1, scrollY: 0}})}>
                    Previous
                </Button>
                <Button
                    disabled={!page?.page.nextCursor || query.loading}
                    onClick={() =>
                        update({
                            ...session,
                            [session.view]: {...current, index: current.index + 1, cursors: [...current.cursors.slice(0, current.index + 1), page?.page.nextCursor], scrollY: 0}
                        })
                    }
                >
                    Next
                </Button>
                <Button onClick={() => update({...session, [session.view]: blankCursorSession<SubscriptionPage>()})}>Latest subscriptions</Button>
            </Space>
        </AppPage>
    );
};
