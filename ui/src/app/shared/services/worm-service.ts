import requests from './requests';

export interface WormMarketItem {
    conditionId: string;
    title: string;
    description?: string;
    logo?: string;
    lastTradePrice?: string;
    state: string;
    category: string;
    created?: number;
    eventTitle?: string;
    eventConditionId?: string;
    eventLogo?: string;
    marginEnabled: boolean;
    liveState?: 'live' | 'not_live' | 'unknown';
    liveCheckedAt?: number;
    livePriceChange?: string;
    ignored: boolean;
}

export interface ListWormMarketsResult {
    items: WormMarketItem[];
    nextCursor?: string;
    fetchedAt?: number;
    stale?: boolean;
}

export type WormMarketSortOption = 'new' | 'trending' | 'ending_soon' | 'leverage';
export type WormMarketCategorySlug = 'all' | 'politics' | 'sports' | 'crypto' | 'tech' | 'finance' | 'wtf';
export type WormMarketIgnoredFilter = 'active' | 'ignored' | 'all';

export const DEFAULT_WORM_MARKET_SORT: WormMarketSortOption = 'leverage';
export const DEFAULT_WORM_MARKET_CATEGORY: WormMarketCategorySlug = 'sports';
export const DEFAULT_WORM_MARKET_IGNORED_FILTER: WormMarketIgnoredFilter = 'active';
export const DEFAULT_WORM_MARKET_STATE = 'open';

const isOpenWormMarket = (item: Pick<WormMarketItem, 'state'>) => item.state?.toLowerCase() === DEFAULT_WORM_MARKET_STATE;

export interface ListWormMarketsOptions {
    limit?: number;
    cursor?: string;
    sortOption?: WormMarketSortOption;
    categorySlug?: WormMarketCategorySlug;
    ignoredFilter?: WormMarketIgnoredFilter;
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
const readNumber = (item: any, ...names: string[]) => Number(readValue(item, ...names) || 0) || undefined;
const readBoolean = (item: any, ...names: string[]) => Boolean(readValue(item, ...names));
const isLiveWormMarket = (item: Pick<WormMarketItem, 'liveState'>) => item.liveState === 'live';

const normalizeMarket = (item: any): WormMarketItem => ({
    conditionId: readString(item, 'conditionId', 'condition_id'),
    title: readString(item, 'title', 'title'),
    description: readString(item, 'description', 'description'),
    logo: readString(item, 'logo', 'logo'),
    lastTradePrice: readString(item, 'lastTradePrice', 'last_trade_price'),
    state: readString(item, 'state', 'state'),
    category: readString(item, 'category', 'category'),
    created: Number(item?.created || 0) || undefined,
    eventTitle: readString(item, 'eventTitle', 'event_title'),
    eventConditionId: readString(item, 'eventConditionId', 'event_condition_id'),
    eventLogo: readString(item, 'eventLogo', 'event_logo'),
    marginEnabled: readBoolean(item, 'marginEnabled', 'margin_enabled'),
    liveState: readString(item, 'liveState', 'live_state') as WormMarketItem['liveState'],
    liveCheckedAt: readNumber(item, 'liveCheckedAt', 'live_checked_at'),
    livePriceChange: readString(item, 'livePriceChange', 'live_price_change'),
    ignored: readBoolean(item, 'ignored', 'ignored')
});

const sortLiveFirst = (items: WormMarketItem[]) =>
    items
        .map((item, index) => ({item, index}))
        .sort((left, right) => {
            const liveDelta = Number(isLiveWormMarket(right.item)) - Number(isLiveWormMarket(left.item));
            return liveDelta || left.index - right.index;
        })
        .map(entry => entry.item);

export class WormService {
    public listMarkets(options: ListWormMarketsOptions = {}): Promise<ListWormMarketsResult> & {abort?: () => void} {
        const query: any = {
            limit: options.limit || 20,
            cursor: options.cursor || ''
        };
        if (options.sortOption) {
            query.sort_option = options.sortOption;
        }
        if (options.categorySlug) {
            query.category_slug = options.categorySlug;
        }
        query.ignored_filter = options.ignoredFilter || DEFAULT_WORM_MARKET_IGNORED_FILTER;
        const req = requests.get('/worm/markets').query(query);
        const promise = req.then(res => {
            const body = res.body || {};
            const items = (body.items || []).map(normalizeMarket).filter(isOpenWormMarket);
            return {
                items: sortLiveFirst(items),
                nextCursor: body.nextCursor || body.next_cursor || '',
                fetchedAt: readNumber(body, 'fetchedAt', 'fetched_at'),
                stale: readBoolean(body, 'stale', 'stale')
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public batchUpdateMarketsIgnored(conditionIds: string[], ignored: boolean): Promise<{updated: number}> & {abort?: () => void} {
        const req = requests.post('/worm/markets/ignored').send({
            conditionIds,
            condition_ids: conditionIds,
            ignored
        });
        const promise = req.then(res => ({updated: readNumber(res.body || {}, 'updated', 'updated') || 0})) as any;
        promise.abort = () => req.abort();
        return promise;
    }
}
