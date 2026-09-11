import type {Delivery, StatusCounts, SummaryPart} from '../../trader-sync-models';
import {subscriptionTime} from './subscription-state';
export const deliveryLabel = (delivery: Delivery): string =>
    ({pending: 'Queued', sending: 'Sending', sent: 'Telegram accepted', failed: 'Delivery failed', unknown: 'Delivery unknown', cancelled: 'Delivery cancelled'})[delivery.status];
export const NotificationResult = ({delivery}: {delivery: Delivery}) => (
    <div className='trader-sync-delivery' data-delivery-status={delivery.status}>
        <h3>{deliveryLabel(delivery)}</h3>
        {delivery.status === 'sent' && <p>Telegram accepted this message. This is not a read receipt.</p>}
        {delivery.status === 'unknown' && <p>Telegram may have received this message. No automatic resend.</p>}
        {delivery.reason && <p>Reason: {delivery.reason}</p>}
        <p>Attempts recorded: {delivery.attemptCount}</p>
        <details>
            <summary>Delivery evidence</summary>
            <p>Delivery ID: {delivery.id}</p>
            <p>Authorized at: {subscriptionTime(delivery.authorizedAt)}</p>
            <p>{delivery.startedAt ? `Submitted at: ${subscriptionTime(delivery.startedAt)}` : 'Submit time not recorded'}</p>
            <p>Result at: {subscriptionTime(delivery.resultAt)}</p>
            {(!delivery.startedAt || !delivery.resultAt) && <p>Submission-to-result duration cannot be determined.</p>}
            <p>Message ID: {delivery.messageId || 'Not recorded'}</p>
            {delivery.latestAttempt && (
                <div>
                    <h4>Latest attempt · {delivery.latestAttempt.index}</h4>
                    <p>
                        Result:{' '}
                        {delivery.latestAttempt.status === 'retryable'
                            ? 'Retryable response'
                            : delivery.latestAttempt.status === 'sent'
                              ? 'Telegram accepted'
                              : delivery.latestAttempt.status}
                    </p>
                    <p>Authorized at: {subscriptionTime(delivery.latestAttempt.authorizedAt)}</p>
                    <p>{delivery.latestAttempt.startedAt ? `Submitted at: ${subscriptionTime(delivery.latestAttempt.startedAt)}` : 'Submit time not recorded'}</p>
                    <p>Result at: {subscriptionTime(delivery.latestAttempt.resultAt)}</p>
                    {delivery.latestAttempt.reason && <p>Reason: {delivery.latestAttempt.reason}</p>}
                </div>
            )}
        </details>
    </div>
);
export const PartCounts = ({counts, label}: {counts: StatusCounts; label: string}) => (
    <div className='trader-sync-part-counts'>
        <p>
            {label}: {counts.total} parts
        </p>
        <dl>
            {(['pending', 'sending', 'sent', 'failed', 'unknown', 'cancelled'] as const).map(status => (
                <div key={status}>
                    <dt>{status}</dt>
                    <dd>{counts[status]}</dd>
                </div>
            ))}
        </dl>
    </div>
);
export const SummaryParts = ({parts}: {parts: SummaryPart[]}) => (
    <div>
        {parts.map(part => (
            <article className='trader-sync-summary-part' key={part.id} data-part-id={part.id}>
                <h3>
                    Part {part.index} of {part.total}
                </h3>
                <p>Associated activities: {part.associatedActivityCount}</p>
                <NotificationResult delivery={part.delivery} />
            </article>
        ))}
    </div>
);
