import * as React from 'react';

export interface AsyncState<T> {
    data?: T;
    loading: boolean;
    error?: Error;
    reload: () => void;
}

export interface CachedAsyncState<T> extends AsyncState<T> {
    refreshing: boolean;
}

export interface CachedAsyncDataOptions {
    staleTimeMs: number;
}

type AbortablePromise<T> = Promise<T> & {abort?: () => void};

interface CacheSnapshot<T> {
    data?: T;
    hasData: boolean;
    loading: boolean;
    refreshing: boolean;
    error?: Error;
    updatedAt: number;
    revision: number;
}

interface CacheRequest {
    id: number;
    generation: number;
    abort?: () => void;
}

interface CacheEntry {
    key: string;
    generation: number;
    snapshot: CacheSnapshot<unknown>;
    listeners: Set<() => void>;
    request?: CacheRequest;
    requestSequence: number;
    lastUsed: number;
    hasSubscribed: boolean;
}

const MAX_INACTIVE_CACHE_ENTRIES = 20;
const cacheEntries = new Map<string, CacheEntry>();
let cacheGeneration = 0;
let cacheAccessSequence = 0;
let cachePruneScheduled = false;

const normalizeError = (error: unknown): Error => {
    if (error instanceof Error) {
        return error;
    }
    if (error && typeof error === 'object' && 'message' in error) {
        return new Error(String((error as {message?: unknown}).message));
    }
    return new Error(String(error));
};

const touchCacheEntry = (entry: CacheEntry) => {
    entry.lastUsed = ++cacheAccessSequence;
};

const publishCacheSnapshot = (entry: CacheEntry, update: Partial<CacheSnapshot<unknown>>) => {
    entry.snapshot = {
        ...entry.snapshot,
        ...update,
        revision: entry.snapshot.revision + 1
    };
    entry.listeners.forEach(listener => listener());
};

const abortCacheRequest = (entry: CacheEntry) => {
    const request = entry.request;
    if (!request) {
        return;
    }

    entry.request = undefined;
    entry.requestSequence++;
    try {
        request.abort?.();
    } catch {
        // An abort failure cannot make an invalidated request current again.
    }
    publishCacheSnapshot(entry, {
        loading: !entry.snapshot.hasData,
        refreshing: false
    });
};

const pruneInactiveCacheEntries = () => {
    const removable = Array.from(cacheEntries.values()).filter(entry => entry.listeners.size === 0 && !entry.request);
    const abandoned = removable.filter(entry => !entry.hasSubscribed);
    const inactive = removable.filter(entry => entry.hasSubscribed).sort((left, right) => left.lastUsed - right.lastUsed);

    for (const entry of [...abandoned, ...inactive.slice(0, Math.max(0, inactive.length - MAX_INACTIVE_CACHE_ENTRIES))]) {
        if (cacheEntries.get(entry.key) !== entry) {
            continue;
        }
        cacheEntries.delete(entry.key);
        publishCacheSnapshot(entry, {});
    }
};

const scheduleCachePrune = () => {
    if (cachePruneScheduled) {
        return;
    }
    cachePruneScheduled = true;
    queueMicrotask(() => {
        cachePruneScheduled = false;
        pruneInactiveCacheEntries();
    });
};

const getCacheEntry = (key: string): CacheEntry => {
    let entry = cacheEntries.get(key);
    if (!entry) {
        entry = {
            key,
            generation: cacheGeneration,
            snapshot: {
                hasData: false,
                loading: true,
                refreshing: false,
                updatedAt: 0,
                revision: 0
            },
            listeners: new Set(),
            requestSequence: 0,
            lastUsed: ++cacheAccessSequence,
            hasSubscribed: false
        };
        cacheEntries.set(key, entry);
    }
    return entry;
};

const subscribeCacheEntry = (entry: CacheEntry, listener: () => void) => {
    if (cacheEntries.get(entry.key) !== entry || entry.generation !== cacheGeneration) {
        let active = true;
        queueMicrotask(() => {
            if (active) {
                listener();
            }
        });
        return () => {
            active = false;
        };
    }
    entry.hasSubscribed = true;
    entry.listeners.add(listener);
    touchCacheEntry(entry);
    scheduleCachePrune();

    return () => {
        entry.listeners.delete(listener);
        touchCacheEntry(entry);
        const request = entry.request;
        queueMicrotask(() => {
            if (entry.listeners.size === 0 && entry.request === request) {
                abortCacheRequest(entry);
            }
            scheduleCachePrune();
        });
    };
};

const startCacheRequest = <T>(entry: CacheEntry, load: () => AbortablePromise<T>, staleTimeMs: number, force: boolean) => {
    if (cacheEntries.get(entry.key) !== entry || entry.generation !== cacheGeneration) {
        return;
    }
    if (entry.request) {
        return;
    }

    const snapshot = entry.snapshot;
    if (!force && snapshot.hasData && Date.now() - snapshot.updatedAt < staleTimeMs) {
        return;
    }

    const id = ++entry.requestSequence;
    const generation = entry.generation;
    touchCacheEntry(entry);
    publishCacheSnapshot(entry, {
        loading: !snapshot.hasData,
        refreshing: snapshot.hasData,
        error: undefined
    });

    let promise: AbortablePromise<T>;
    try {
        promise = load();
    } catch (error) {
        if (generation === cacheGeneration && entry.requestSequence === id) {
            publishCacheSnapshot(entry, {
                loading: false,
                refreshing: false,
                error: normalizeError(error)
            });
        }
        return;
    }

    entry.request = {id, generation, abort: promise.abort?.bind(promise)};
    promise.then(
        data => {
            if (generation !== cacheGeneration || entry.request?.id !== id || entry.request.generation !== generation) {
                return;
            }
            entry.request = undefined;
            touchCacheEntry(entry);
            publishCacheSnapshot(entry, {
                data,
                hasData: true,
                loading: false,
                refreshing: false,
                error: undefined,
                updatedAt: Date.now()
            });
            scheduleCachePrune();
        },
        error => {
            if (generation !== cacheGeneration || entry.request?.id !== id || entry.request.generation !== generation) {
                return;
            }
            entry.request = undefined;
            touchCacheEntry(entry);
            publishCacheSnapshot(entry, {
                loading: false,
                refreshing: false,
                error: normalizeError(error)
            });
            scheduleCachePrune();
        }
    );
};

export const clearAsyncDataCache = () => {
    cacheGeneration++;
    const entries = Array.from(cacheEntries.values());
    cacheEntries.clear();

    entries.forEach(entry => {
        abortCacheRequest(entry);
        publishCacheSnapshot(entry, {
            data: undefined,
            hasData: false,
            loading: true,
            refreshing: false,
            error: undefined,
            updatedAt: 0
        });
    });
};

export const useAsyncData = <T>(load: () => Promise<T> & {abort?: () => void}, deps: React.DependencyList): AsyncState<T> => {
    const [data, setData] = React.useState<T>();
    const [loading, setLoading] = React.useState(true);
    const [error, setError] = React.useState<Error>();
    const [version, setVersion] = React.useState(0);

    React.useEffect(() => {
        let active = true;
        const req = load();
        setLoading(true);
        setError(undefined);
        req.then(
            next => {
                if (active) {
                    setData(next);
                    setLoading(false);
                }
            },
            err => {
                if (active) {
                    setError(err instanceof Error ? err : new Error(String(err?.message || err)));
                    setLoading(false);
                }
            }
        );
        return () => {
            active = false;
            req.abort?.();
        };
    }, [...deps, version]);

    return {data, loading, error, reload: () => setVersion(current => current + 1)};
};

export const useCachedAsyncData = <T>(cacheKey: string, load: () => AbortablePromise<T>, options: CachedAsyncDataOptions): CachedAsyncState<T> => {
    const loadRef = React.useRef(load);
    React.useLayoutEffect(() => {
        loadRef.current = load;
    }, [load]);

    const entry = getCacheEntry(cacheKey);
    const subscribe = React.useCallback((listener: () => void) => subscribeCacheEntry(entry, listener), [entry]);
    const getSnapshot = React.useCallback(() => entry.snapshot, [entry]);
    const snapshot = React.useSyncExternalStore(subscribe, getSnapshot, getSnapshot) as CacheSnapshot<T>;
    const staleTimeMs = Number.isFinite(options.staleTimeMs) ? Math.max(0, options.staleTimeMs) : 0;

    React.useEffect(() => {
        startCacheRequest(entry, () => loadRef.current(), staleTimeMs, false);
    }, [entry, staleTimeMs]);

    const reload = React.useCallback(() => {
        startCacheRequest(entry, () => loadRef.current(), staleTimeMs, true);
    }, [entry, staleTimeMs]);

    return {
        data: snapshot.data,
        loading: snapshot.loading,
        refreshing: snapshot.refreshing,
        error: snapshot.error,
        reload
    };
};
