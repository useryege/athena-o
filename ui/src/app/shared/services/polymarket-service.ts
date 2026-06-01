import requests from './requests';

export interface PolymarketSportsLiveMarketItem {
    conditionId: string;
    marketSlug: string;
    eventSlug: string;
    title: string;
    image?: string;
    score?: string;
    period?: string;
    elapsed?: string;
    lastUpdate?: string;
    liquidityNum?: number;
    volumeNum?: number;
}

export interface ListPolymarketSportsLiveMarketsResult {
    items: PolymarketSportsLiveMarketItem[];
    fetchedAt?: number;
    stale?: boolean;
}

const readValue = (item: any, ...names: string[]) => {
    for (const name of names) {
        if (item?.[name] !== undefined && item?.[name] !== null) {
            return item[name];
        }
    }
    return undefined;
};

const readString = (item: any, ...names: string[]) => String(readValue(item, ...names) || '');
const readNumber = (item: any, ...names: string[]) => {
    const value = Number(readValue(item, ...names));
    return Number.isFinite(value) ? value : undefined;
};
const readBoolean = (item: any, ...names: string[]) => Boolean(readValue(item, ...names));

const normalizeItem = (item: any): PolymarketSportsLiveMarketItem => ({
    conditionId: readString(item, 'conditionId', 'condition_id'),
    marketSlug: readString(item, 'marketSlug', 'market_slug'),
    eventSlug: readString(item, 'eventSlug', 'event_slug'),
    title: readString(item, 'title'),
    image: readString(item, 'image'),
    score: readString(item, 'score'),
    period: readString(item, 'period'),
    elapsed: readString(item, 'elapsed'),
    lastUpdate: readString(item, 'lastUpdate', 'last_update'),
    liquidityNum: readNumber(item, 'liquidityNum', 'liquidity_num'),
    volumeNum: readNumber(item, 'volumeNum', 'volume_num')
});

export class PolymarketService {
    public listSportsLiveMarkets(limit = 200): Promise<ListPolymarketSportsLiveMarketsResult> & {abort?: () => void} {
        const req = requests.get('/polymarket/sports/live/markets').query({limit});
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: (body.items || []).map(normalizeItem),
                fetchedAt: readNumber(body, 'fetchedAt', 'fetched_at'),
                stale: readBoolean(body, 'stale')
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }
}
