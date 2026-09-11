import type {ReadScope} from '../../../shared/use-visible-query';

let generation = 0;
const invalidations = new Map<symbol, () => void>();

/** Capture once per identity; callers append resource/filter/cursor to this key. */
export const captureTraderSyncScope = (ownerId: string): ReadScope => {
    const captured = generation;
    const isCurrent = () => captured === generation;
    return {
        key: JSON.stringify([ownerId, captured]),
        isCurrent,
        subscribeInvalidation: listener => {
            if (!isCurrent()) {
                listener();
                return () => undefined;
            }
            const token = Symbol();
            invalidations.set(token, listener);
            return () => {
                invalidations.delete(token);
            };
        }
    };
};

/** Sole module cleanup boundary; future drafts/page caches must be cleared here too. */
export const clearTraderSyncState = (): void => {
    generation++;
    const listeners = Array.from(invalidations.values());
    invalidations.clear();
    listeners.forEach(listener => listener());
};
