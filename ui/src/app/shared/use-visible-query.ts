import {useCallback, useLayoutEffect, useRef, useState} from 'react';

export type AbortablePromise<T> = Promise<T> & {abort?: () => void};

export interface ReadScope {
    key: string;
    isCurrent: () => boolean;
    subscribeInvalidation?: (listener: () => void) => () => void;
}

export interface VisibleQuery<T> {
    data?: T;
    error?: Error;
    loading: boolean;
    stale: boolean;
    reload: () => void;
}

type Snapshot<T> = Omit<VisibleQuery<T>, 'reload'> & {key: string};

/** Visible, single-flight reads. Scope owners control identity invalidation. */
export const useVisibleQuery = <T>(load: () => AbortablePromise<T>, scope: ReadScope, intervalMs: number): VisibleQuery<T> => {
    const loadRef = useRef(load);
    const scopeRef = useRef(scope);
    loadRef.current = load;
    scopeRef.current = scope;
    const controller = useRef<{reload: () => void}>();
    const [snapshot, setSnapshot] = useState<Snapshot<T>>({key: scope.key, loading: false, stale: false});
    const reload = useCallback(() => controller.current?.reload(), []);

    useLayoutEffect(() => {
        const key = scope.key;
        let active = true;
        let invalidated = false;
        let sequence = 0;
        let pending: AbortablePromise<T> | undefined;
        let busy = false;
        const current = () => active && !invalidated && scope.isCurrent() && scopeRef.current.key === key && scopeRef.current.isCurrent();
        const empty = () => setSnapshot({key, loading: false, stale: false});
        const abort = () => {
            sequence++;
            const previous = pending;
            pending = undefined;
            busy = false;
            previous?.abort?.();
        };
        const invalidate = () => {
            invalidated = true;
            abort();
            if (active) empty();
        };
        const refresh = () => {
            if (!current() || busy || document.visibilityState === 'hidden') return;
            busy = true;
            const requestSequence = ++sequence;
            const canPublish = () => current() && requestSequence === sequence;
            setSnapshot(previous => ({...previous, loading: previous.data === undefined}));
            let request: AbortablePromise<T>;
            try {
                request = loadRef.current();
            } catch (error) {
                if (canPublish()) {
                    busy = false;
                    setSnapshot(previous => ({...previous, loading: false, stale: previous.data !== undefined, error: error instanceof Error ? error : new Error(String(error))}));
                }
                return;
            }
            // load itself may synchronously invalidate the scope before returning.
            if (canPublish()) pending = request;
            else request.abort?.();
            void request
                .then(
                    data => {
                        if (canPublish()) setSnapshot({key, data, loading: false, stale: false});
                    },
                    error => {
                        if (canPublish())
                            setSnapshot(previous => ({
                                ...previous,
                                loading: false,
                                stale: previous.data !== undefined,
                                error: error instanceof Error ? error : new Error(String(error))
                            }));
                    }
                )
                .finally(() => {
                    if (canPublish()) {
                        pending = undefined;
                        busy = false;
                    }
                });
        };
        empty();
        controller.current = {reload: refresh};
        const unsubscribe = scope.subscribeInvalidation?.(invalidate);
        if (!current()) invalidate();
        document.addEventListener('visibilitychange', refresh);
        window.addEventListener('focus', refresh);
        const timer = window.setInterval(refresh, intervalMs);
        refresh();
        return () => {
            active = false;
            abort();
            unsubscribe?.();
            window.clearInterval(timer);
            document.removeEventListener('visibilitychange', refresh);
            window.removeEventListener('focus', refresh);
            controller.current = undefined;
        };
        // Scope/load object identities change on normal renders; key owns the lifecycle.
    }, [scope.key, intervalMs]);

    // Never render the previous resource even before its layout cleanup runs.
    if (snapshot.key !== scope.key || !scope.isCurrent()) return {loading: false, stale: false, reload};
    return {...snapshot, reload};
};
