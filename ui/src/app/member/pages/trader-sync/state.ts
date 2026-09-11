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
    activitySessions.clear();
    detailSessions.clear();
    subscriptionSessions.clear();
    historySessions.clear();
    subscriptionEdits.clear();
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

export interface CursorSession<T> {
    index: number;
    cursors: Array<string | undefined>;
    pages: Record<number, T>;
    scrollY: number;
}
export interface SubscriptionListSession {
    view: 'current' | 'cancelled';
    current: CursorSession<import('../../trader-sync-models').SubscriptionPage>;
    cancelled: CursorSession<import('../../trader-sync-models').SubscriptionPage>;
}
export const blankCursorSession = <T>(): CursorSession<T> => ({index: 0, cursors: [undefined], pages: {}, scrollY: 0});
const subscriptionSessions = new Map<string, SubscriptionListSession>();
const historySessions = new Map<string, CursorSession<import('../../trader-sync-models').HistoryPage>>();
export const readSubscriptionSession = (ownerId: string) => subscriptionSessions.get(ownerId);
export const saveSubscriptionSession = (ownerId: string, scope: ReadScope, value: SubscriptionListSession): void => {
    if (scope.isCurrent() && scope.key === captureTraderSyncScope(ownerId).key) subscriptionSessions.set(ownerId, value);
};
export const readHistorySession = (ownerId: string, id: string) => historySessions.get(JSON.stringify([ownerId, id]));
export const saveHistorySession = (ownerId: string, id: string, scope: ReadScope, value: CursorSession<import('../../trader-sync-models').HistoryPage>): void => {
    if (scope.isCurrent() && scope.key === captureTraderSyncScope(ownerId).key) historySessions.set(JSON.stringify([ownerId, id]), value);
};

export type SubscriptionIntent =
    | {action: 'pause' | 'resume' | 'cancel'; payload: import('../../trader-sync-service').ChangeRequest}
    | {action: 'note'; payload: import('../../trader-sync-service').NoteRequest};
export interface SubscriptionEdit {
    intent?: SubscriptionIntent;
    note?: string;
    noteRevision?: string;
    serverNote?: string;
    needsRefresh?: 'note' | 'change';
}
const subscriptionEdits = new Map<string, SubscriptionEdit>();
export const readSubscriptionEdit = (ownerId: string, id: string): SubscriptionEdit => subscriptionEdits.get(JSON.stringify([ownerId, id])) || {};
export const saveSubscriptionEdit = (ownerId: string, id: string, scope: ReadScope, value: SubscriptionEdit): void => {
    if (scope.isCurrent() && scope.key === captureTraderSyncScope(ownerId).key) subscriptionEdits.set(JSON.stringify([ownerId, id]), value);
};

const activitySessions = new Map<string, import('./activity-session').ActivitySession>();
export const readActivitySession = (ownerId: string) => activitySessions.get(ownerId);
/** Async consumers pass their original scope so cleanup cannot be undone by a late save. */
export const saveActivitySession = (ownerId: string, value: import('./activity-session').ActivitySession, scope: ReadScope = captureTraderSyncScope(ownerId)): void => {
    if (scope.isCurrent() && scope.key === captureTraderSyncScope(ownerId).key) activitySessions.set(ownerId, value);
};
export const consumeNewSubscriptionFocus = (ownerId: string, subscriptionId: string): boolean => {
    if (readNewSubscriptionFocus(ownerId) !== subscriptionId) return false;
    newSubscriptionFocus = undefined;
    return true;
};

export interface DetailCursorSession<T> {
    revision?: number;
    index: number;
    cursors: Array<string | undefined>;
    pages: Record<number, T>;
    positions: Record<number, number>;
}
export interface DetailSession {
    scrollY: number;
    returnPath?: string;
    parts: DetailCursorSession<import('../../trader-sync-models').PartPage>;
    activities: DetailCursorSession<import('../../trader-sync-models').ActivityPage>;
}
export const blankDetailSession = (): DetailSession => ({
    scrollY: 0,
    parts: {index: 0, cursors: [undefined], pages: {}, positions: {}},
    activities: {index: 0, cursors: [undefined], pages: {}, positions: {}}
});
const detailSessions = new Map<string, DetailSession>();
export const readDetailSession = (ownerId: string, resource: string): DetailSession | undefined => detailSessions.get(JSON.stringify([ownerId, resource]));
export const saveDetailSession = (ownerId: string, resource: string, scope: ReadScope, session: DetailSession): void => {
    if (scope.isCurrent() && scope.key === captureTraderSyncScope(ownerId).key) detailSessions.set(JSON.stringify([ownerId, resource]), session);
};
