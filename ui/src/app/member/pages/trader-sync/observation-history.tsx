import * as React from 'react';
import {Alert, Button, Space} from 'antd';
import {Section} from '../../../components';
import {memberServices as services} from '../../services';
import type {HistoryPage} from '../../trader-sync-models';
import {AbortablePromise} from '../../../shared/use-visible-query';
import {requestErrorMessage} from '../../../shared/services/requests';
import {blankCursorSession, captureTraderSyncScope, readHistorySession, saveHistorySession} from './state';
import {subscriptionTime} from './subscription-state';
export const ObservationHistory = ({ownerId, subscriptionId}: {ownerId: string; subscriptionId: string}) => (
    <History key={JSON.stringify([ownerId, subscriptionId])} ownerId={ownerId} subscriptionId={subscriptionId} />
);
const History = ({ownerId, subscriptionId}: {ownerId: string; subscriptionId: string}) => {
    const scope = React.useMemo(() => captureTraderSyncScope(ownerId), [ownerId]);
    const [session, setSession] = React.useState(() => readHistorySession(ownerId, subscriptionId) || blankCursorSession<HistoryPage>());
    const sessionRef = React.useRef(session);
    const [error, setError] = React.useState('');
    const [loading, setLoading] = React.useState(false);
    const [retry, setRetry] = React.useState(0);
    const save = (next: typeof session) => {
        if (!scope.isCurrent()) return;
        sessionRef.current = next;
        saveHistorySession(ownerId, subscriptionId, scope, next);
        setSession(next);
    };
    React.useLayoutEffect(() => {
        const unsubscribe = scope.subscribeInvalidation?.(() => {
            sessionRef.current = blankCursorSession();
            setSession(sessionRef.current);
            setError('');
        });
        return () => {
            saveHistorySession(ownerId, subscriptionId, scope, {...sessionRef.current, scrollY: window.scrollY});
            unsubscribe?.();
        };
    }, [scope]);
    React.useEffect(() => {
        if (!scope.isCurrent()) return;
        if (session.pages[session.index]) {
            setError('');
            return;
        }
        let active = true;
        let request: AbortablePromise<HistoryPage> | undefined;
        setLoading(true);
        setError('');
        const unsubscribe = scope.subscribeInvalidation?.(() => {
            active = false;
            request?.abort?.();
            setLoading(false);
        });
        try {
            request = services.traderSync.listSubscriptionHistory(subscriptionId, {pageSize: 50, cursor: session.cursors[session.index]});
            void request
                .then(
                    page => {
                        if (active && scope.isCurrent()) save({...sessionRef.current, pages: {...sessionRef.current.pages, [session.index]: page}});
                    },
                    failure => {
                        if (active && scope.isCurrent()) setError(requestErrorMessage(failure));
                    }
                )
                .finally(() => {
                    if (active && scope.isCurrent()) setLoading(false);
                });
        } catch (failure) {
            setError(requestErrorMessage(failure));
            setLoading(false);
        }
        return () => {
            active = false;
            request?.abort?.();
            unsubscribe?.();
        };
    }, [scope.key, session.index, retry]);
    const page = scope.isCurrent() ? session.pages[session.index] : undefined;
    return (
        <Section title='Observation history'>
            <p>
                Confirmed observation intervals and interruption records. Missed activity is not backfilled. An unconfirmed recovery in an old record does not describe the current
                subscription state.
            </p>
            {loading && <p role='status'>Loading observation history…</p>}
            {error && <Alert type='error' title={error} action={<Button onClick={() => setRetry(value => value + 1)}>Retry history</Button>} />}
            {page && (
                <>
                    <p>As of {subscriptionTime(page.asOf)}</p>
                    {page.entries.length === 0 && <p>No observation history recorded yet.</p>}
                    <ol className='trader-sync-observation-history'>
                        {page.entries.map(entry => (
                            <li key={entry.id} data-history-id={entry.id}>
                                <strong>{entry.kind === 'interval' ? 'Monitoring interval' : 'Monitoring interruption'}</strong>
                                <p>Recorded ordering time: {subscriptionTime(entry.sortAt)}</p>
                                {entry.interval && (
                                    <>
                                        <p>Effective from (inclusive): {subscriptionTime(entry.interval.effectiveAt)}</p>
                                        <p>End (exclusive): {entry.interval.endedAt ? subscriptionTime(entry.interval.endedAt) : 'Not ended in this record'}</p>
                                        <p>
                                            Generation {entry.interval.generation}; collector epoch {entry.interval.epoch}
                                        </p>
                                    </>
                                )}
                                {entry.interruption && (
                                    <>
                                        <p>Actual start: {subscriptionTime(entry.interruption.start)}</p>
                                        <p>Recovery boundary: {subscriptionTime(entry.interruption.end)}</p>
                                        <p>Recovered at: {subscriptionTime(entry.interruption.recoveredAt)}</p>
                                        <p>Reason: {entry.interruption.reason || 'Not provided'}</p>
                                        <p>Uncertainty: {entry.interruption.uncertainty || 'None reported'}</p>
                                        <p>
                                            {entry.interruption.possibleMissing
                                                ? 'Activity may have been missed. The number is unknown.'
                                                : 'No possible missing activity reported for this record.'}
                                        </p>
                                    </>
                                )}
                            </li>
                        ))}
                    </ol>
                </>
            )}
            <Space wrap={true}>
                <Button disabled={loading || session.index === 0} onClick={() => save({...session, index: session.index - 1})}>
                    Previous history
                </Button>
                <Button
                    disabled={loading || !page?.page.nextCursor}
                    onClick={() => save({...session, index: session.index + 1, cursors: [...session.cursors.slice(0, session.index + 1), page?.page.nextCursor]})}>
                    More history
                </Button>
                <Button
                    disabled={loading}
                    onClick={() => {
                        save(blankCursorSession<HistoryPage>());
                        setRetry(value => value + 1);
                    }}>
                    Latest history
                </Button>
            </Space>
        </Section>
    );
};
