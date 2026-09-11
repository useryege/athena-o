import * as React from 'react';
import {Link} from 'react-router-dom';
import {Typography} from 'antd';
import type {Activity, TradeMetadata} from '../../trader-sync-models';
import {SubscriptionIdentity, subscriptionTime} from './subscription-state';
import {formatRaw} from './precision';

/** Selection can span rows; freeze every metadata region intersected by any range. */
export const selectionIntersects = (node: Node | null, selection: Selection | null): boolean => {
    if (!node || !selection || selection.isCollapsed) return false;
    for (let index = 0; index < selection.rangeCount; index++) {
        try {
            if (selection.getRangeAt(index).intersectsNode(node)) return true;
        } catch {
            /* Detached selection ranges do not own this row. */
        }
    }
    return false;
};
const Metadata = React.memo(({metadata, positionId}: {metadata: TradeMetadata; positionId: string}) => {
    const market = metadata.market;
    return (
        <>
            {market.evidence.availability === 'available' ? (
                <>
                    <strong>{market.title || 'Market title unavailable'}</strong>
                    <p>Outcome: {market.outcome || 'Unavailable'}</p>
                    {market.url && (
                        <a href={market.url} target='_blank' rel='noreferrer'>
                            View market
                        </a>
                    )}
                </>
            ) : (
                <p>Metadata unavailable: {market.evidence.reasonCode || 'not provided'}</p>
            )}
            <p>
                Position ID: <Typography.Text copyable={{text: positionId}}>{positionId}</Typography.Text>
            </p>
            {metadata.relationship && <p>Combo: {metadata.relationship === 'AND(legs)' ? 'All conditions must hold (YES)' : 'Not all conditions hold (NO)'}</p>}
            {metadata.relationship &&
                (metadata.legsEvidence.availability === 'available' ? (
                    <details>
                        <summary>{metadata.legs.length} conditions</summary>
                        <ul>
                            {metadata.legs.map((leg, index) => (
                                <li key={`${index}-${leg.positionId}`}>
                                    {leg.market.evidence.availability === 'available'
                                        ? `${leg.market.title} — ${leg.market.outcome}`
                                        : `Metadata unavailable · Position ${leg.positionId}`}
                                </li>
                            ))}
                        </ul>
                    </details>
                ) : (
                    <p>Conditions unavailable; count unknown</p>
                ))}
        </>
    );
});
const Notification = ({activity}: {activity: Activity}) => {
    if (activity.notificationMode === 'in_app_only') return <p>In-app only · Unbound when recorded; later binding does not send this activity.</p>;
    if (activity.notificationMode === 'ordinary')
        return (
            <p>
                Telegram: {activity.delivery?.status || 'Result unavailable'}
                {activity.delivery?.status === 'sent' ? ' · Telegram confirmed receipt' : ''}
                {activity.delivery?.status === 'unknown' ? ' · May have been received; no automatic resend' : ''}
                {activity.delivery?.reason ? ` · ${activity.delivery.reason}` : ''}
            </p>
        );
    const progress = activity.summaryProgress;
    if (!progress) return <p>Summary status unavailable</p>;
    if (progress.phase === 'waiting') return <p>Waiting for summary</p>;
    if (progress.phase === 'cancelled_before_freeze') return <p>Summary cancelled before freeze: {progress.reason}</p>;
    const counts = progress.relatedPartCounts;
    return (
        <>
            <p>
                Related summary parts: pending {counts.pending}; sending {counts.sending}; sent {counts.sent}; failed {counts.failed}; unknown {counts.unknown}; cancelled{' '}
                {counts.cancelled}
            </p>
            {progress.batchId && <Link to={`/trader-sync/summaries/${progress.batchId}`}>View summary batch</Link>}
        </>
    );
};
const ActivityRow = React.memo(({activity, onOpen, summaryBatchId}: {activity: Activity; onOpen: () => void; summaryBatchId?: string}) => {
    const row = React.useRef<HTMLElement>(null);
    const latest = React.useRef(activity.metadata);
    latest.current = activity.metadata;
    const [metadata, setMetadata] = React.useState(activity.metadata);
    React.useLayoutEffect(() => {
        if (!selectionIntersects(row.current, window.getSelection())) setMetadata(activity.metadata);
    }, [activity.metadata]);
    React.useEffect(() => {
        const apply = () => {
            if (!selectionIntersects(row.current, window.getSelection())) setMetadata(latest.current);
        };
        document.addEventListener('selectionchange', apply);
        return () => document.removeEventListener('selectionchange', apply);
    }, []);
    return (
        <article ref={row} className='trader-sync-activity' data-activity-id={activity.id}>
            <div className='trader-sync-activity-identity'>
                <SubscriptionIdentity wallet={activity.wallet} note={activity.noteSnapshot} display={activity.targetDisplaySnapshot} />
                <Link to={`/trader-sync/subscriptions/${activity.subscriptionId}`} onClick={onOpen}>
                    View original subscription
                </Link>
                <p>
                    <strong>{activity.side}</strong> · Settled at {subscriptionTime(activity.settledAt)}
                </p>
            </div>
            <div className='trader-sync-activity-metadata'>
                <Metadata metadata={metadata} positionId={activity.positionId} />
            </div>
            <div className='trader-sync-activity-amounts'>
                <p>
                    Trade value{' '}
                    <Typography.Text copyable={{text: formatRaw(activity.collateralRaw, activity.collateralDecimals)}}>
                        {formatRaw(activity.collateralRaw, activity.collateralDecimals)}
                    </Typography.Text>{' '}
                    {activity.collateralSymbol}
                </p>
                <p>
                    Shares{' '}
                    <Typography.Text copyable={{text: formatRaw(activity.sharesRaw, activity.sharesDecimals)}}>
                        {formatRaw(activity.sharesRaw, activity.sharesDecimals)}
                    </Typography.Text>
                </p>
            </div>
            <div className='trader-sync-activity-notification'>
                <Notification activity={activity} />
                {activity.finalityAnomaly && <p className='trader-sync-anomaly'>Finality anomaly</p>}
                <Link
                    to={`/trader-sync/activities/${activity.id}`}
                    state={summaryBatchId && /^[1-9]\d*$/.test(summaryBatchId) ? {traderSyncSummaryBatchId: summaryBatchId} : undefined}
                    onClick={onOpen}>
                    View activity
                </Link>
            </div>
        </article>
    );
});
export const ActivityFeed = ({activities, onOpen, summaryBatchId}: {activities: Activity[]; onOpen: () => void; summaryBatchId?: string}) => (
    <div className='trader-sync-feed'>
        {activities.map(activity => (
            <ActivityRow key={activity.id} activity={activity} onOpen={onOpen} summaryBatchId={summaryBatchId} />
        ))}
    </div>
);
