import type {ResolvedTarget} from '../../trader-sync-models';
import type {CreateRequest} from '../../trader-sync-service';
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
    addDraft = undefined;
    newSubscriptionFocus = undefined;
    const listeners = Array.from(invalidations.values());
    invalidations.clear();
    listeners.forEach(listener => listener());
};

export interface AddDraft {
    ownerId: string;
    input: string;
    note: string;
    noteEdited: boolean;
    wallet?: string;
    target?: ResolvedTarget;
    createRequest?: CreateRequest;
    returnPath: string;
    scrollY: number;
}
let addDraft: AddDraft | undefined;
let newSubscriptionFocus: {ownerId: string; subscriptionId: string} | undefined;
export const readAddDraft = (ownerId: string): AddDraft | undefined => (addDraft?.ownerId === ownerId ? addDraft : undefined);
export const saveAddDraft = (draft: AddDraft): void => {
    addDraft = draft;
};
/** Task17 reads this without changing its existing activity filter. */
export const saveNewSubscriptionFocus = (ownerId: string, subscriptionId: string): void => {
    newSubscriptionFocus = {ownerId, subscriptionId};
};
export const readNewSubscriptionFocus = (ownerId: string): string | undefined => (newSubscriptionFocus?.ownerId === ownerId ? newSubscriptionFocus.subscriptionId : undefined);
