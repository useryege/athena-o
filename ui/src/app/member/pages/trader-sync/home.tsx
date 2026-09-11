import * as React from 'react';
import {Alert, Skeleton} from 'antd';
import {Link, useLocation} from 'react-router-dom';
import {AppPage} from '../../../components';
import {requestErrorMessage} from '../../../shared/services/requests';
import {AbortablePromise, useVisibleQuery} from '../../../shared/use-visible-query';
import {memberServices as services} from '../../services';
import type {ActivityQuery} from '../../trader-sync-service';
import {activityDateRange, ActivitySession, applyActivityRefresh, readActivitySession, saveActivitySession, validateActivityPage} from './activity-session';
import {captureTraderSyncScope} from './state';
import {subscriptionTime} from './subscription-state';
import {TargetSidebar} from './target-sidebar';
import {ActivityFeed} from './activity-feed';
import {CursorNavigation} from './cursor-navigation';
import './trader-sync.css';
const readable = <T,>(request: AbortablePromise<T>): AbortablePromise<T> =>
    Object.assign(
        request.catch(error => {
            throw new Error(requestErrorMessage(error));
        }),
        {abort: () => request.abort?.()}
    );
const dateInput = (iso?: string, end = false) => (iso ? new Date(Date.parse(iso) + 8 * 3600000 - (end ? 86400000 : 0)).toISOString().slice(0, 10) : '');
type Operation = {kind: 'latest' | 'next'; query: ActivityQuery};
export const TraderSyncHomePage = ({ownerId}: {ownerId: string}) => <Home key={ownerId} ownerId={ownerId} />;
const Home = ({ownerId}: {ownerId: string}) => {
    const location = useLocation();
    const scope = React.useMemo(() => captureTraderSyncScope(ownerId), [ownerId]);
    const [session, setSession] = React.useState<ActivitySession>(() => {
        const saved = readActivitySession(ownerId);
        const params = new URLSearchParams(location.search);
        const subscriptionId = params.get('subscriptionId') || undefined;
        if (saved && (!params.has('subscriptionId') || saved.query.subscriptionId === subscriptionId)) return saved;
        return {query: {pageSize: 50, subscriptionId}, previous: [], scrollY: 0};
    });
    const sessionRef = React.useRef(session);
    const pendingScroll = React.useRef<{session: ActivitySession; scopeKey: string}>();
    const mounted = React.useRef(true);
    const busy = React.useRef(false);
    const [reading, setReading] = React.useState(false);
    const operation = React.useRef<Operation>();
    const queuedAction = React.useRef<() => void>();
    const queuedTimer = React.useRef<number>();
    const [from, setFrom] = React.useState(() => dateInput(session.query.from));
    const [to, setTo] = React.useState(() => dateInput(session.query.to, true));
    const [filterError, setFilterError] = React.useState('');
    const heading = React.useRef<HTMLHeadingElement>(null);
    const save = React.useCallback(
        (value: ActivitySession) => {
            if (!mounted.current || !scope.isCurrent()) return;
            sessionRef.current = value;
            saveActivitySession(ownerId, value, scope);
            setSession(value);
        },
        [ownerId, scope]
    );
    const rememberScroll = React.useCallback(() => {
        if (scope.isCurrent() && !pendingScroll.current) {
            sessionRef.current = {...sessionRef.current, scrollY: window.scrollY};
            saveActivitySession(ownerId, sessionRef.current, scope);
        }
    }, [ownerId, scope]);
    React.useLayoutEffect(() => {
        mounted.current = true;
        if (sessionRef.current.scrollY > 0) window.scrollTo({top: sessionRef.current.scrollY, behavior: 'instant'});
        const unsubscribe = scope.subscribeInvalidation?.(() => {
            pendingScroll.current = undefined;
            operation.current = undefined;
            queuedAction.current = undefined;
            window.clearTimeout(queuedTimer.current);
            sessionRef.current = {query: {pageSize: 50}, previous: [], scrollY: 0};
            setSession(sessionRef.current);
            setFrom('');
            setTo('');
            setFilterError('');
        });
        window.addEventListener('scroll', rememberScroll, {passive: true});
        return () => {
            rememberScroll();
            mounted.current = false;
            pendingScroll.current = undefined;
            window.clearTimeout(queuedTimer.current);
            unsubscribe?.();
            window.removeEventListener('scroll', rememberScroll);
        };
    }, [scope, rememberScroll]);
    React.useLayoutEffect(() => {
        const restore = pendingScroll.current;
        if (!restore) return;
        pendingScroll.current = undefined;
        // Query and page-stack references identify this navigation; a normal
        // fixed-page refresh preserves both even when its metadata is replaced.
        if (mounted.current && scope.isCurrent() && restore.scopeKey === scope.key && session.query === restore.session.query && session.previous === restore.session.previous)
            window.scrollTo({top: restore.session.scrollY, behavior: 'instant'});
    }, [session, scope]);
    const activityRead = useVisibleQuery(
        () => {
            const base = sessionRef.current;
            const action = operation.current;
            const query: ActivityQuery =
                action?.query || (base.current ? {...base.query, cursor: undefined, refreshCursor: base.current.page.refreshCursor} : {...base.query, refreshCursor: undefined});
            busy.current = true;
            setReading(!!action || !base.current);
            let request: ReturnType<typeof services.traderSync.listActivities>;
            try {
                request = services.traderSync.listActivities(query);
            } catch (error) {
                busy.current = false;
                setReading(false);
                throw error;
            }
            let cancelled = false;
            const current = () => !cancelled && mounted.current && scope.isCurrent();
            const result = request
                .then(incoming => {
                    if (!current()) return incoming;
                    const page = !action && base.current ? applyActivityRefresh(base.current, incoming) : validateActivityPage(incoming);
                    if (action?.kind === 'next' && base.current?.page.snapshot !== page.page.snapshot)
                        throw new Error('Trader Sync protocol error: history snapshot changed. The previous page is retained.');
                    const previous =
                        action?.kind === 'latest'
                            ? []
                            : action?.kind === 'next' && base.current
                              ? [...base.previous, {query: base.query, page: base.current, scrollY: base.scrollY}]
                              : base.previous;
                    save({query: action?.query || base.query, current: page, previous, scrollY: action ? 0 : sessionRef.current.scrollY});
                    operation.current = undefined;
                    if (action) {
                        window.scrollTo({top: 0, behavior: 'instant'});
                        if (action.kind === 'latest') heading.current?.focus({preventScroll: true});
                    }
                    return page;
                })
                .catch(error => {
                    if (current()) operation.current = undefined;
                    throw new Error(requestErrorMessage(error));
                })
                .finally(() => {
                    if (current()) {
                        busy.current = false;
                        const queued = queuedAction.current;
                        queuedAction.current = undefined;
                        setReading(!!queued);
                        if (queued)
                            queuedTimer.current = window.setTimeout(() => {
                                if (current()) queued();
                            }, 0);
                    }
                });
            return Object.assign(result, {
                abort: () => {
                    cancelled = true;
                    request.abort?.();
                }
            });
        },
        {...scope, key: JSON.stringify([scope.key, 'activity-home'])},
        5000
    );
    const targets = useVisibleQuery(
        () => readable(services.traderSync.listSubscriptions({view: 'current', pageSize: 10})),
        {...scope, key: JSON.stringify([scope.key, 'targets'])},
        5000
    );
    const telegram = useVisibleQuery(() => readable(services.memberNotifications.getTelegramSettings()), {...scope, key: JSON.stringify([scope.key, 'binding'])}, 5000);
    const load = (kind: Operation['kind'], query: ActivityQuery) => {
        if (!scope.isCurrent()) return;
        if (busy.current) {
            queuedAction.current = () => load(kind, query);
            setReading(true);
            return;
        }
        operation.current = {kind, query};
        activityRead.reload();
    };
    const previousPage = () => {
        if (!scope.isCurrent()) return;
        if (busy.current) {
            queuedAction.current = previousPage;
            setReading(true);
            return;
        }
        const previous = sessionRef.current.previous.at(-1);
        if (previous) {
            const restored: ActivitySession = {query: previous.query, current: previous.page, previous: sessionRef.current.previous.slice(0, -1), scrollY: previous.scrollY};
            pendingScroll.current = {session: restored, scopeKey: scope.key};
            save(restored);
            activityRead.reload();
        }
    };
    const latest = (query = session.query) => load('latest', {...query, cursor: undefined, refreshCursor: undefined});
    const changeFilter = (subscriptionId?: string) => latest({...session.query, subscriptionId});
    const current = scope.isCurrent() ? session.current : undefined;
    const filtered = !!(session.query.subscriptionId || session.query.from || session.query.to);
    const targetPage = targets.data;
    return (
        <AppPage
            title='Trader Sync'
            subtitle='Activity Alerts · Follow confirmed trades across your selected traders.'
            extra={
                <Link className='trader-sync-add-action' to={`/trader-sync/add${location.search}`} onClick={rememberScroll}>
                    Add trader
                </Link>
            }>
            <div className='trader-sync-home'>
                <div className='trader-sync-binding'>
                    <span>Current Telegram: {telegram.data ? telegram.data.binding?.status || 'unbound' : telegram.error ? 'Unavailable' : 'Loading…'}</span> ·{' '}
                    <Link to='/notifications' onClick={rememberScroll}>
                        Manage notifications
                    </Link>
                    {telegram.data && !telegram.data.binding && <p>New activity remains available in-app. Binding later does not send earlier activity.</p>}
                    {telegram.error && (
                        <Alert
                            type='error'
                            title='Current binding data may be out of date'
                            description={telegram.error.message}
                            action={
                                <button type='button' onClick={telegram.reload}>
                                    Retry binding
                                </button>
                            }
                        />
                    )}
                </div>
                {targets.error && (
                    <Alert
                        type='error'
                        title='Target data may be out of date'
                        description={targets.error.message}
                        action={
                            <button type='button' onClick={targets.reload}>
                                Retry targets
                            </button>
                        }
                    />
                )}
                <div className='trader-sync-layout'>
                    <TargetSidebar ownerId={ownerId} page={targetPage} selected={session.query.subscriptionId} onSelect={changeFilter} disabled={reading} />
                    <section className='trader-sync-main' aria-labelledby='trader-sync-activity-title'>
                        <h2 id='trader-sync-activity-title' tabIndex={-1} ref={heading}>
                            Activity
                        </h2>
                        <form
                            className='trader-sync-filters'
                            onSubmit={event => {
                                event.preventDefault();
                                try {
                                    const dates = activityDateRange(from, to);
                                    setFilterError('');
                                    latest({...session.query, ...dates});
                                } catch (error) {
                                    setFilterError((error as Error).message);
                                }
                            }}>
                            <label>
                                Settlement from (UTC+8)
                                <input type='date' value={from} onChange={event => setFrom(event.target.value)} />
                            </label>
                            <label>
                                Settlement through (UTC+8)
                                <input type='date' value={to} onChange={event => setTo(event.target.value)} />
                            </label>
                            <button type='submit' disabled={reading}>
                                Apply dates
                            </button>
                            <label>
                                Page size
                                <select
                                    value={session.query.pageSize || 50}
                                    disabled={reading}
                                    onChange={event => latest({...session.query, pageSize: event.target.value === '100' ? 100 : 50})}>
                                    <option value='50'>50</option>
                                    <option value='100'>100</option>
                                </select>
                            </label>
                            {filtered && (
                                <button
                                    type='button'
                                    disabled={reading}
                                    onClick={() => {
                                        setFrom('');
                                        setTo('');
                                        latest({pageSize: session.query.pageSize});
                                    }}>
                                    Clear filters
                                </button>
                            )}
                        </form>
                        {filterError && <p role='alert'>{filterError}</p>}
                        <p>Settlement dates use UTC+8; the end date includes that day. Activities retain their recorded order.</p>
                        {activityRead.error && (
                            <Alert
                                type='error'
                                title={current ? 'Activity data may be out of date' : 'Could not load activity'}
                                description={activityRead.error.message}
                                action={
                                    <button type='button' disabled={reading} onClick={activityRead.reload}>
                                        Retry activity
                                    </button>
                                }
                            />
                        )}
                        {current && <p>Last checked: {subscriptionTime(current.page.asOf)}</p>}
                        <div className='trader-sync-update-slot'>
                            {current?.page.hasNewer && (
                                <button type='button' className='trader-sync-new-activity' disabled={reading} onClick={() => latest()}>
                                    New activity available
                                </button>
                            )}
                        </div>
                        {activityRead.error && (
                            <button type='button' disabled={reading} onClick={() => latest()}>
                                Load latest activities
                            </button>
                        )}
                        {!current && activityRead.loading && <Skeleton active={true} title={true} paragraph={{rows: 6}} />}
                        {current?.activities.length === 0 && (
                            <div className='trader-sync-empty'>
                                {filtered ? (
                                    <p>No activity matches these filters. Clear filters to see all retained activity.</p>
                                ) : targetPage?.subscriptions.length === 0 ? (
                                    <p>
                                        No current subscriptions. <Link to='/trader-sync/add'>Add a trader</Link> to begin. Retained activity stays available.
                                    </p>
                                ) : targetPage && !targetPage.subscriptions.some(item => item.status === 'healthy') ? (
                                    <p>
                                        No activity yet. Check target monitoring states in <Link to='/trader-sync/subscriptions'>Subscriptions</Link>.
                                    </p>
                                ) : (
                                    <p>No activity yet. Monitoring begins at the effective boundary; earlier or missed trades are not backfilled.</p>
                                )}
                            </div>
                        )}
                        <ActivityFeed activities={current?.activities || []} onOpen={rememberScroll} />
                        <CursorNavigation
                            canPrevious={!reading && session.previous.length > 0}
                            nextCursor={!reading ? current?.page.nextCursor : undefined}
                            onNext={() => {
                                if (current?.page.nextCursor) {
                                    sessionRef.current = {...sessionRef.current, scrollY: window.scrollY};
                                    load('next', {...session.query, cursor: current.page.nextCursor, refreshCursor: undefined});
                                }
                            }}
                            onPrevious={previousPage}
                        />
                    </section>
                </div>
            </div>
        </AppPage>
    );
};
