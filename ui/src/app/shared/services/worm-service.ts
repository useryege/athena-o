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
}

export interface ListWormMarketsResult {
    items: WormMarketItem[];
    nextCursor?: string;
}

const readString = (item: any, camelName: string, snakeName: string) => item?.[camelName] || item?.[snakeName] || '';

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
    marginEnabled: Boolean(item?.marginEnabled ?? item?.margin_enabled)
});

export class WormService {
    public listMarkets(limit = 20, cursor = ''): Promise<ListWormMarketsResult> & {abort?: () => void} {
        const req = requests.get('/worm/markets').query({limit, cursor});
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: (body.items || []).map(normalizeMarket),
                nextCursor: body.nextCursor || body.next_cursor || ''
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }
}
