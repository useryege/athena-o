import requests from './requests';
import {readBoolean, readNumber, readString} from './api-values';
import {normalizeSportsHistoryEventCard, normalizeSportsPriceHistorySeries, SportsHistoryEventCardItem, SportsPriceHistorySeriesItem} from './sports-models';

export interface ListSportsHistoryEventsResult {
    items: SportsHistoryEventCardItem[];
    fetchedAt?: number;
    stale?: boolean;
}

export interface BatchGetSportsHistoryPriceHistoriesResult {
    items: SportsPriceHistorySeriesItem[];
}

export type SportsHistorySyncState = 'idle' | 'syncing' | 'succeeded' | 'failed';

export interface SportsHistorySyncStatus {
    state: SportsHistorySyncState;
    startedAt?: number;
    completedAt?: number;
    lastSuccessAt?: number;
    errorMessage?: string;
}

const normalizeSyncStatus = (item: any): SportsHistorySyncStatus => {
    const state = readString(item, 'state').toLowerCase();
    return {
        state: state === 'syncing' || state === 'succeeded' || state === 'failed' ? state : 'idle',
        startedAt: readNumber(item, 'startedAt', 'started_at'),
        completedAt: readNumber(item, 'completedAt', 'completed_at'),
        lastSuccessAt: readNumber(item, 'lastSuccessAt', 'last_success_at'),
        errorMessage: readString(item, 'errorMessage', 'error_message')
    };
};

export class SportsHistoryService {
    public listEvents(limit = 200): Promise<ListSportsHistoryEventsResult> & {abort?: () => void} {
        const req = requests.get('/sports-history/events').query({limit});
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: (body.items || []).map(normalizeSportsHistoryEventCard),
                fetchedAt: readNumber(body, 'fetchedAt', 'fetched_at'),
                stale: readBoolean(body, 'stale')
            };
        }) as Promise<ListSportsHistoryEventsResult> & {abort?: () => void};
        promise.abort = () => req.abort();
        return promise;
    }

    public batchGetPriceHistories(marketKeys: string[], limitPerToken = 360): Promise<BatchGetSportsHistoryPriceHistoriesResult> & {abort?: () => void} {
        const req = requests.post('/sports-history/price-history:batchGet').send({market_keys: marketKeys, limit_per_token: limitPerToken});
        const promise = req.then(res => ({items: (res.body?.items || []).map(normalizeSportsPriceHistorySeries)})) as Promise<BatchGetSportsHistoryPriceHistoriesResult> & {
            abort?: () => void;
        };
        promise.abort = () => req.abort();
        return promise;
    }

    public getSyncStatus(): Promise<SportsHistorySyncStatus> & {abort?: () => void} {
        const req = requests.get('/sports-history/sync-status');
        const promise = req.then(res => normalizeSyncStatus(res.body?.status || {})) as Promise<SportsHistorySyncStatus> & {abort?: () => void};
        promise.abort = () => req.abort();
        return promise;
    }

    public refresh(): Promise<SportsHistorySyncStatus> & {abort?: () => void} {
        const req = requests.post('/sports-history:refresh').send({});
        const promise = req.then(res => normalizeSyncStatus(res.body?.status || {})) as Promise<SportsHistorySyncStatus> & {abort?: () => void};
        promise.abort = () => req.abort();
        return promise;
    }
}
