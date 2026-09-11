import * as React from 'react';
import {Link, useParams} from 'react-router-dom';
import {Alert} from 'antd';
import {AppPage, Section} from '../../../components';
import {useVisibleQuery, AbortablePromise} from '../../../shared/use-visible-query';
import {requestErrorDetails, requestErrorMessage} from '../../../shared/services/requests';
import {memberServices as services} from '../../services';
import {blankDetailSession, captureTraderSyncScope, DetailSession, readDetailSession, saveDetailSession} from './state';
import {applyActivityRefresh, validateActivityPage} from './activity-session';
import {ActivityFeed} from './activity-feed';
import {subscriptionTime} from './subscription-state';
import {PartCounts, SummaryParts} from './notification-result';
import {CursorNavigation} from './cursor-navigation';
import './trader-sync.css';

/** Preserve backend error codes through the polling hook, which accepts Error only. */
export const detailRead = <T,>(request: AbortablePromise<T>): AbortablePromise<T> =>
    Object.assign(
        request.catch(error => {
            throw Object.assign(new Error(requestErrorMessage(error)), {code: requestErrorDetails(error).code});
        }),
        {abort: () => request.abort?.()}
    );
export const isDetailNotFound = (error?: Error) => (error as (Error & {code?: number}) | undefined)?.code === 5;
export const DetailReadError = ({error, reload, label, lastUpdated}: {error?: Error; reload: () => void; label: string; lastUpdated?: string}) =>
    error ? (
        <Alert
            type='error'
            title={isDetailNotFound(error) ? `${label} not found` : `Could not refresh ${label.toLowerCase()}`}
            description={
                isDetailNotFound(error) ? (
                    'This item is unavailable.'
                ) : (
                    <>
                        {error.message}
                        {lastUpdated && <p>Data may be out of date. Last updated: {subscriptionTime(lastUpdated)}</p>}
                    </>
                )
            }
            action={
                <button type='button' onClick={reload}>
                    Refresh
                </button>
            }
        />
    ) : null;

export const invalidDetailCursor = (error?: Error) => (error as (Error & {code?: number}) | undefined)?.code === 3;
type Lane = 'parts' | 'activities';
export const useDetailSession = (ownerId: string, resource: string, ready: boolean) => {
    const scope = React.useMemo(() => captureTraderSyncScope(ownerId), [ownerId]);
    const [session, setSession] = React.useState(() => readDetailSession(ownerId, resource) || blankDetailSession());
    const ref = React.useRef(session);
    const mounted = React.useRef(true);
    const activeReads = React.useRef(new Map<Lane, symbol>());
    const queuedMoves = React.useRef(new Map<Lane, -1 | 1>());
    const pendingScroll = React.useRef<{parts: number; activities: number; top: number} | undefined>({
        parts: session.parts.index,
        activities: session.activities.index,
        top: session.scrollY
    });
    const save = () => {
        if (mounted.current && scope.isCurrent()) saveDetailSession(ownerId, resource, scope, ref.current);
    };
    const update = (value: DetailSession) => {
        if (!mounted.current || !scope.isCurrent()) return;
        ref.current = value;
        save();
        setSession(value);
    };
    const remember = () => {
        if (!pendingScroll.current) {
            ref.current = {...ref.current, scrollY: window.scrollY};
            save();
        }
    };
    React.useLayoutEffect(() => {
        mounted.current = true;
        const invalidation = scope.subscribeInvalidation?.(() => {
            pendingScroll.current = undefined;
            activeReads.current.clear();
            queuedMoves.current.clear();
            ref.current = blankDetailSession();
            setSession(ref.current);
        });
        window.addEventListener('scroll', remember);
        return () => {
            remember();
            mounted.current = false;
            pendingScroll.current = undefined;
            invalidation?.();
            window.removeEventListener('scroll', remember);
        };
    }, [scope]);
    React.useLayoutEffect(() => {
        const pending = pendingScroll.current;
        if (
            !pending ||
            !ready ||
            !mounted.current ||
            !scope.isCurrent() ||
            session !== ref.current ||
            pending.parts !== session.parts.index ||
            pending.activities !== session.activities.index
        )
            return;
        const partsReady = resource.startsWith('activity/') || !!session.parts.pages[session.parts.index];
        const activitiesReady = resource.startsWith('activity/') || !!session.activities.pages[session.activities.index];
        if (!partsReady || !activitiesReady) return;
        window.scrollTo({top: pending.top, behavior: 'instant'});
        pendingScroll.current = undefined;
        ref.current = {...ref.current, scrollY: pending.top};
        save();
    }, [session, ready, scope]);
    const move = (lane: Lane, direction: -1 | 1) => {
        if (!scope.isCurrent()) return;
        if (activeReads.current.has(lane)) {
            queuedMoves.current.set(lane, direction);
            return;
        }
        const value = ref.current;
        const current = value[lane];
        const nextCursor = current.pages[current.index]?.page.nextCursor;
        if ((direction === -1 && current.index === 0) || (direction === 1 && !nextCursor)) return;
        const index = current.index + direction;
        const cursors = direction === 1 ? [...current.cursors.slice(0, index), nextCursor] : current.cursors;
        const next: DetailSession = {...value, [lane]: {...current, index, cursors, positions: {...current.positions, [current.index]: window.scrollY}}};
        const top = direction === -1 ? current.positions[index] || 0 : 0;
        pendingScroll.current = {parts: next.parts.index, activities: next.activities.index, top};
        update({...next, scrollY: top});
    };
    const trackRead = <T,>(lane: Lane, request: AbortablePromise<T>): AbortablePromise<T> => {
        const token = Symbol();
        activeReads.current.set(lane, token);
        return Object.assign(
            request.finally(() => {
                if (activeReads.current.get(lane) !== token) return;
                activeReads.current.delete(lane);
                const direction = queuedMoves.current.get(lane);
                queuedMoves.current.delete(lane);
                if (direction)
                    window.setTimeout(() => {
                        if (mounted.current && scope.isCurrent()) move(lane, direction);
                    }, 0);
            }),
            {abort: () => request.abort?.()}
        );
    };
    const restart = (lane: Lane) => {
        queuedMoves.current.delete(lane);
        const value = ref.current;
        const next: DetailSession = {...value, [lane]: {index: 0, cursors: [undefined], pages: {}, positions: {}, revision: (value[lane].revision || 0) + 1}};
        pendingScroll.current = {parts: next.parts.index, activities: next.activities.index, top: 0};
        update({...next, scrollY: 0});
    };
    return {session, ref, scope, update, remember, move, trackRead, restart};
};
type DetailController = ReturnType<typeof useDetailSession>;

/** Each lane owns its input cursor; activities additionally use their signed refresh cursor. */
export const useDetailParts = (controller: DetailController, batchId: string, activityId?: string) => {
    const {scope, session, ref, update} = controller;
    const index = session.parts.index;
    const query = useVisibleQuery(
        () => {
            const current = ref.current.parts;
            const request = detailRead(services.traderSync.listSummaryParts(batchId, {pageSize: 50, cursor: current.cursors[index], activityId}));
            return controller.trackRead(
                'parts',
                Object.assign(
                    request.then(page => {
                        const old = current.pages[index];
                        if (old && (old.parts.length !== page.parts.length || old.parts.some((part, i) => part.id !== page.parts[i].id)))
                            throw new Error('Summary part membership changed; the previous page is retained.');
                        return page;
                    }),
                    {abort: () => request.abort?.()}
                )
            );
        },
        {...scope, key: JSON.stringify([scope.key, batchId, activityId, 'parts', index, session.parts.revision])},
        5000
    );
    React.useEffect(() => {
        if (query.data && scope.isCurrent()) update({...ref.current, parts: {...ref.current.parts, pages: {...ref.current.parts.pages, [index]: query.data}}});
    }, [query.data]);
    const page = scope.isCurrent() && !isDetailNotFound(query.error) ? session.parts.pages[index] : undefined;
    return {...query, page};
};
const useSummaryActivities = (controller: DetailController, batchId: string) => {
    const {scope, session, ref, update} = controller;
    const index = session.activities.index;
    const query = useVisibleQuery(
        () => {
            const current = ref.current.activities;
            const old = current.pages[index];
            const input = old ? {refreshCursor: old.page.refreshCursor} : {cursor: current.cursors[index]};
            const request = detailRead(services.traderSync.listActivities({summaryBatchId: batchId, pageSize: 50, ...input}));
            return controller.trackRead(
                'activities',
                Object.assign(
                    request.then(page => {
                        if (old) return applyActivityRefresh(old, page);
                        validateActivityPage(page);
                        if (index > 0 && current.pages[index - 1]?.page.snapshot !== page.page.snapshot)
                            throw new Error('Summary activity snapshot changed; return to the previous page.');
                        return page;
                    }),
                    {abort: () => request.abort?.()}
                )
            );
        },
        {...scope, key: JSON.stringify([scope.key, batchId, 'activities', index, session.activities.revision])},
        5000
    );
    React.useEffect(() => {
        if (query.data && scope.isCurrent()) update({...ref.current, activities: {...ref.current.activities, pages: {...ref.current.activities.pages, [index]: query.data}}});
    }, [query.data]);
    const page = scope.isCurrent() && !isDetailNotFound(query.error) ? session.activities.pages[index] : undefined;
    return {...query, page};
};
export const TraderSyncSummaryPage = ({ownerId}: {ownerId: string}) => {
    const {batchId = ''} = useParams();
    return <SummaryDetail key={JSON.stringify([ownerId, batchId])} ownerId={ownerId} batchId={batchId} />;
};
const SummaryDetail = ({ownerId, batchId}: {ownerId: string; batchId: string}) => {
    const scope = React.useMemo(() => captureTraderSyncScope(ownerId), [ownerId]);
    const batch = useVisibleQuery(() => detailRead(services.traderSync.getSummaryBatch(batchId)), {...scope, key: JSON.stringify([scope.key, 'batch', batchId])}, 5000);
    const controller = useDetailSession(ownerId, `summary/${batchId}`, !!batch.data && !isDetailNotFound(batch.error));
    const parts = useDetailParts(controller, batchId);
    const activities = useSummaryActivities(controller, batchId);
    const value = scope.isCurrent() && !isDetailNotFound(batch.error) ? batch.data : undefined;
    const counts = value?.partCounts;
    const complete =
        counts &&
        BigInt(counts.total) > 0n &&
        counts.sent === counts.total &&
        (['pending', 'sending', 'failed', 'unknown', 'cancelled'] as const).every(key => BigInt(counts[key]) === 0n);
    return (
        <AppPage
            title='Summary batch'
            subtitle='Fixed trade membership and a separate delivery result for every part.'
            extra={
                <Link to='/trader-sync' onClick={controller.remember}>
                    Back to Trader Sync
                </Link>
            }>
            <div className='trader-sync-detail trader-sync-home'>
                <DetailReadError error={batch.error} reload={batch.reload} lastUpdated={value?.asOf} label='Summary batch' />
                {!value && batch.loading && <p role='status'>Loading summary batch…</p>}
                {value && (
                    <>
                        <Section title='Batch facts'>
                            <p>Batch ID: {value.id}</p>
                            <p>Activities in this batch: {value.activityCount}</p>
                            <PartCounts counts={value.partCounts} label='Entire batch' />
                            {complete && <p>All parts accepted by Telegram</p>}
                            <p>
                                Settled from: {subscriptionTime(value.settledFrom)} · Through: {subscriptionTime(value.settledTo)}
                            </p>
                            <p>
                                Recorded from: {subscriptionTime(value.recordedFrom)} · Through: {subscriptionTime(value.recordedTo)}
                            </p>
                            <p>Oldest waiting activity: {subscriptionTime(value.oldestAt)}</p>
                            <p>First part submitted: {subscriptionTime(value.firstStartedAt)}</p>
                            <details>
                                <summary>Activities by target</summary>
                                <ul>
                                    {value.targetCounts.map(target => (
                                        <li key={target.wallet}>
                                            {target.wallet}: {target.count} activities
                                        </li>
                                    ))}
                                </ul>
                            </details>
                            <p>Batch updated: {subscriptionTime(value.asOf)}. Parts and activities update independently.</p>
                        </Section>
                        <Section title='Summary parts'>
                            {invalidDetailCursor(parts.error) && (
                                <button type='button' onClick={() => controller.restart('parts')}>
                                    Return to first part page
                                </button>
                            )}
                            <DetailReadError error={parts.error} reload={parts.reload} lastUpdated={parts.page?.asOf} label='Summary parts' />
                            {parts.page ? <SummaryParts parts={parts.page.parts} /> : parts.loading ? <p role='status'>Loading parts…</p> : null}
                            {parts.page?.parts.length === 0 && <p>No parts.</p>}
                            <CursorNavigation
                                ariaLabel='Summary part pages'
                                canPrevious={controller.session.parts.index > 0}
                                nextCursor={parts.page?.page.nextCursor}
                                onPrevious={() => controller.move('parts', -1)}
                                onNext={() => controller.move('parts', 1)}
                            />
                        </Section>
                        <Section title='Batch activities'>
                            {invalidDetailCursor(activities.error) && (
                                <button type='button' onClick={() => controller.restart('activities')}>
                                    Return to first activity page
                                </button>
                            )}
                            <DetailReadError error={activities.error} reload={activities.reload} lastUpdated={activities.page?.page.asOf} label='Batch activities' />
                            {activities.page ? (
                                <ActivityFeed activities={activities.page.activities} onOpen={controller.remember} summaryBatchId={batchId} />
                            ) : activities.loading ? (
                                <p role='status'>Loading activities…</p>
                            ) : null}
                            {activities.page?.activities.length === 0 && <p>No activities.</p>}
                            <CursorNavigation
                                ariaLabel='Summary activity pages'
                                canPrevious={controller.session.activities.index > 0}
                                nextCursor={activities.page?.page.nextCursor}
                                onPrevious={() => controller.move('activities', -1)}
                                onNext={() => controller.move('activities', 1)}
                            />
                        </Section>
                    </>
                )}
            </div>
        </AppPage>
    );
};
