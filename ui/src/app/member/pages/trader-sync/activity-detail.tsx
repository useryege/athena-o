import * as React from 'react';
import {Link, useParams, useLocation} from 'react-router-dom';
import {AppPage, Section} from '../../../components';
import {useVisibleQuery} from '../../../shared/use-visible-query';
import {memberServices as services} from '../../services';
import {captureTraderSyncScope} from './state';
import {TradeFacts} from './trade-facts';
import {NotificationResult, PartCounts, SummaryParts} from './notification-result';
import {CursorNavigation} from './cursor-navigation';
import {detailRead, invalidDetailCursor, DetailReadError, isDetailNotFound, useDetailParts, useDetailSession} from './summary-detail';
import {subscriptionTime} from './subscription-state';
import './trader-sync.css';

export const TraderSyncActivityPage = ({ownerId}: {ownerId: string}) => {
    const {activityId = ''} = useParams();
    return <ActivityDetail key={JSON.stringify([ownerId, activityId])} ownerId={ownerId} activityId={activityId} />;
};
const RelatedParts = ({controller, batchId, activityId}: {controller: ReturnType<typeof useDetailSession>; batchId: string; activityId: string}) => {
    const parts = useDetailParts(controller, batchId, activityId);
    return (
        <div>
            {invalidDetailCursor(parts.error) && (
                <button type='button' onClick={() => controller.restart('parts')}>
                    Return to first part page
                </button>
            )}
            <DetailReadError error={parts.error} reload={parts.reload} lastUpdated={parts.page?.asOf} label='Related summary parts' />
            {parts.page ? <SummaryParts parts={parts.page.parts} /> : parts.loading ? <p role='status'>Loading related parts…</p> : null}
            {parts.page?.parts.length === 0 && <p>No related parts.</p>}
            <CursorNavigation
                ariaLabel='Related summary part pages'
                canPrevious={controller.session.parts.index > 0}
                nextCursor={parts.page?.page.nextCursor}
                onPrevious={() => controller.move('parts', -1)}
                onNext={() => controller.move('parts', 1)}
            />
        </div>
    );
};
const ActivityDetail = ({ownerId, activityId}: {ownerId: string; activityId: string}) => {
    const scope = React.useMemo(() => captureTraderSyncScope(ownerId), [ownerId]);
    const query = useVisibleQuery(() => detailRead(services.traderSync.getActivity(activityId)), {...scope, key: JSON.stringify([scope.key, 'activity', activityId])}, 5000);
    const [lastUpdated, setLastUpdated] = React.useState<string>();
    React.useEffect(() => {
        if (query.data) setLastUpdated(new Date().toISOString());
    }, [query.data]);
    const item = scope.isCurrent() && !isDetailNotFound(query.error) ? query.data : undefined;
    const controller = useDetailSession(ownerId, `activity/${activityId}`, !!item);
    const progress = item?.summaryProgress;
    // Source belongs to this history entry; resource sessions retain only pages and scroll.
    const navigationState: unknown = useLocation().state;
    const summarySource = navigationState && typeof navigationState === 'object' ? (navigationState as {traderSyncSummaryBatchId?: unknown}).traderSyncSummaryBatchId : undefined;
    const returnPath = typeof summarySource === 'string' && /^[1-9]\d*$/.test(summarySource) ? `/trader-sync/summaries/${summarySource}` : undefined;
    return (
        <AppPage
            title='Activity'
            subtitle='The published trade, its original evidence and notification results.'
            extra={
                <Link to={returnPath || '/trader-sync'} onClick={controller.remember}>
                    {returnPath ? 'Back to summary batch' : 'Back to Trader Sync'}
                </Link>
            }>
            <div className='trader-sync-detail trader-sync-home'>
                <DetailReadError error={query.error} reload={query.reload} lastUpdated={item ? lastUpdated : undefined} label='Activity' />
                {!item && query.loading && <p role='status'>Loading activity…</p>}
                {item && (
                    <>
                        <Section title='Trade facts'>
                            <TradeFacts activity={item} />
                            <Link to={`/trader-sync/subscriptions/${item.subscriptionId}`} onClick={controller.remember}>
                                View original subscription
                            </Link>
                        </Section>
                        <Section title='Notification results'>
                            {item.notificationMode === 'in_app_only' && (
                                <p>In-app only · Telegram was not bound when this activity formed. Later binding does not send this activity.</p>
                            )}
                            {item.notificationReason && <p>Notification reason: {item.notificationReason}</p>}
                            {item.notificationMode === 'ordinary' && (item.delivery ? <NotificationResult delivery={item.delivery} /> : <p>Delivery result unavailable</p>)}
                            {item.notificationMode === 'summary' && (
                                <>
                                    {!progress && <p>Summary status unavailable</p>}
                                    {progress?.phase === 'waiting' && <p>Waiting for summary</p>}
                                    {progress?.phase === 'cancelled_before_freeze' && <p>Summary cancelled before freeze: {progress.reason || 'Reason not recorded'}</p>}
                                    {progress && <p>Oldest waiting activity: {subscriptionTime(progress.oldestAt)}</p>}
                                    {progress?.phase === 'frozen' && (
                                        <>
                                            <p>Summary membership frozen</p>
                                            <PartCounts counts={progress.relatedPartCounts} label='Parts related to this activity' />
                                            <PartCounts counts={progress.batchPartCounts} label='Entire batch' />
                                            <p>First part submitted: {subscriptionTime(progress.firstStartedAt)}</p>
                                            {progress.batchId ? (
                                                <>
                                                    <Link to={`/trader-sync/summaries/${progress.batchId}`} onClick={controller.remember}>
                                                        View summary batch
                                                    </Link>
                                                    <RelatedParts key={progress.batchId} controller={controller} batchId={progress.batchId} activityId={activityId} />
                                                </>
                                            ) : (
                                                <p>Batch reference unavailable</p>
                                            )}
                                        </>
                                    )}
                                </>
                            )}
                        </Section>
                    </>
                )}
            </div>
        </AppPage>
    );
};
