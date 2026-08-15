import requests from './requests';
import {readBoolean, readNumber} from './api-values';
import {normalizeSportsLiveEventCard, normalizeSportsPriceHistorySeries, SportsLiveEventCardItem, SportsPriceHistorySeriesItem} from './sports-models';

export interface ListSportsLiveEventsResult {
    items: SportsLiveEventCardItem[];
    fetchedAt?: number;
    stale?: boolean;
}

export interface BatchGetSportsLivePriceHistoriesResult {
    items: SportsPriceHistorySeriesItem[];
}

export class SportsLiveService {
    public listEvents(limit = 200): Promise<ListSportsLiveEventsResult> & {abort?: () => void} {
        const req = requests.get('/sports-live/events').query({limit});
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: (body.items || []).map(normalizeSportsLiveEventCard),
                fetchedAt: readNumber(body, 'fetchedAt', 'fetched_at'),
                stale: readBoolean(body, 'stale')
            };
        }) as Promise<ListSportsLiveEventsResult> & {abort?: () => void};
        promise.abort = () => req.abort();
        return promise;
    }

    public batchGetPriceHistories(marketKeys: string[], limitPerToken = 360): Promise<BatchGetSportsLivePriceHistoriesResult> & {abort?: () => void} {
        const req = requests.post('/sports-live/price-history:batchGet').send({market_keys: marketKeys, limit_per_token: limitPerToken});
        const promise = req.then(res => ({items: (res.body?.items || []).map(normalizeSportsPriceHistorySeries)})) as Promise<BatchGetSportsLivePriceHistoriesResult> & {
            abort?: () => void;
        };
        promise.abort = () => req.abort();
        return promise;
    }
}
