import requests from './requests';

export interface PolymarketHotMarketTokenItem {
    tokenId: string;
    outcome: string;
    price?: number;
}

export interface PolymarketHotMarketItem {
    conditionId: string;
    marketSlug: string;
    eventSlug: string;
    question: string;
    image?: string;
    volume24hr?: number;
    volumeNum?: number;
    liquidityNum?: number;
    spread?: number;
    bestBid?: number;
    bestAsk?: number;
    lastTradePrice?: number;
    updatedAt?: string;
    tokens: PolymarketHotMarketTokenItem[];
}

export interface ListPolymarketHotMarketsResult {
    items: PolymarketHotMarketItem[];
    fetchedAt?: number;
    stale?: boolean;
    monitoredMarkets?: number;
    monitoredTokens?: number;
    candidateCount?: number;
}

export interface PolymarketRealtimeWindowItem {
    window: string;
    priceChangePp?: number;
    warmup?: boolean;
    sampleCount?: number;
}

export interface PolymarketRealtimeTokenItem {
    tokenId: string;
    outcome: string;
    price?: number;
    bestBid?: number;
    bestAsk?: number;
    spread?: number;
    lastTradePrice?: number;
    lastTradeSize?: number;
    lastTradeSide?: string;
    lastEventAt?: number;
    windows: PolymarketRealtimeWindowItem[];
    warmup?: boolean;
}

export interface PolymarketRealtimeMarketItem {
    conditionId: string;
    marketSlug: string;
    eventSlug: string;
    question: string;
    image?: string;
    volume24hr?: number;
    volumeNum?: number;
    liquidityNum?: number;
    updatedAt?: string;
    tokens: PolymarketRealtimeTokenItem[];
}

export interface ListPolymarketRealtimeMarketsResult {
    items: PolymarketRealtimeMarketItem[];
    fetchedAt?: number;
    stale?: boolean;
    subscribedMarkets?: number;
    subscribedTokens?: number;
    connected?: boolean;
    lastEventAt?: number;
    candidateCount?: number;
}

export interface PolymarketMoverWindowItem {
    window: string;
    priceChangePp?: number;
    warmup?: boolean;
    sampleCount?: number;
}

export interface PolymarketMoverTokenItem {
    tokenId: string;
    outcome: string;
    price?: number;
    bestBid?: number;
    bestAsk?: number;
    spread?: number;
    lastTradePrice?: number;
    lastTradeSize?: number;
    lastTradeSide?: string;
    lastEventAt?: number;
    windows: PolymarketMoverWindowItem[];
    warmup?: boolean;
    score?: number;
    direction?: string;
}

export interface PolymarketMoverMarketItem {
    conditionId: string;
    marketSlug: string;
    eventSlug: string;
    question: string;
    image?: string;
    volume24hr?: number;
    volumeNum?: number;
    liquidityNum?: number;
    updatedAt?: string;
    tokens: PolymarketMoverTokenItem[];
    leader?: PolymarketMoverTokenItem;
    score?: number;
    direction?: string;
}

export interface ListPolymarketMoversResult {
    items: PolymarketMoverMarketItem[];
    fetchedAt?: number;
    stale?: boolean;
    connected?: boolean;
    lastEventAt?: number;
    monitoredMarkets?: number;
    monitoredTokens?: number;
    candidateCount?: number;
}

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

const normalizeHotMarketToken = (item: any): PolymarketHotMarketTokenItem => ({
    tokenId: readString(item, 'tokenId', 'token_id'),
    outcome: readString(item, 'outcome'),
    price: readNumber(item, 'price')
});

const normalizeHotMarket = (item: any): PolymarketHotMarketItem => {
    const tokens = readValue(item, 'tokens');
    return {
        conditionId: readString(item, 'conditionId', 'condition_id'),
        marketSlug: readString(item, 'marketSlug', 'market_slug'),
        eventSlug: readString(item, 'eventSlug', 'event_slug'),
        question: readString(item, 'question'),
        image: readString(item, 'image'),
        volume24hr: readNumber(item, 'volume24hr', 'volume_24hr'),
        volumeNum: readNumber(item, 'volumeNum', 'volume_num'),
        liquidityNum: readNumber(item, 'liquidityNum', 'liquidity_num'),
        spread: readNumber(item, 'spread'),
        bestBid: readNumber(item, 'bestBid', 'best_bid'),
        bestAsk: readNumber(item, 'bestAsk', 'best_ask'),
        lastTradePrice: readNumber(item, 'lastTradePrice', 'last_trade_price'),
        updatedAt: readString(item, 'updatedAt', 'updated_at'),
        tokens: Array.isArray(tokens) ? tokens.map(normalizeHotMarketToken) : []
    };
};

const normalizeRealtimeWindow = (item: any): PolymarketRealtimeWindowItem => ({
    window: readString(item, 'window'),
    priceChangePp: readNumber(item, 'priceChangePp', 'price_change_pp'),
    warmup: readBoolean(item, 'warmup'),
    sampleCount: readNumber(item, 'sampleCount', 'sample_count')
});

const normalizeRealtimeToken = (item: any): PolymarketRealtimeTokenItem => {
    const windows = readValue(item, 'windows');
    return {
        tokenId: readString(item, 'tokenId', 'token_id'),
        outcome: readString(item, 'outcome'),
        price: readNumber(item, 'price'),
        bestBid: readNumber(item, 'bestBid', 'best_bid'),
        bestAsk: readNumber(item, 'bestAsk', 'best_ask'),
        spread: readNumber(item, 'spread'),
        lastTradePrice: readNumber(item, 'lastTradePrice', 'last_trade_price'),
        lastTradeSize: readNumber(item, 'lastTradeSize', 'last_trade_size'),
        lastTradeSide: readString(item, 'lastTradeSide', 'last_trade_side'),
        lastEventAt: readNumber(item, 'lastEventAt', 'last_event_at'),
        windows: Array.isArray(windows) ? windows.map(normalizeRealtimeWindow) : [],
        warmup: readBoolean(item, 'warmup')
    };
};

const normalizeRealtimeMarket = (item: any): PolymarketRealtimeMarketItem => {
    const tokens = readValue(item, 'tokens');
    return {
        conditionId: readString(item, 'conditionId', 'condition_id'),
        marketSlug: readString(item, 'marketSlug', 'market_slug'),
        eventSlug: readString(item, 'eventSlug', 'event_slug'),
        question: readString(item, 'question'),
        image: readString(item, 'image'),
        volume24hr: readNumber(item, 'volume24hr', 'volume_24hr'),
        volumeNum: readNumber(item, 'volumeNum', 'volume_num'),
        liquidityNum: readNumber(item, 'liquidityNum', 'liquidity_num'),
        updatedAt: readString(item, 'updatedAt', 'updated_at'),
        tokens: Array.isArray(tokens) ? tokens.map(normalizeRealtimeToken) : []
    };
};

const normalizeMoverWindow = (item: any): PolymarketMoverWindowItem => ({
    window: readString(item, 'window'),
    priceChangePp: readNumber(item, 'priceChangePp', 'price_change_pp'),
    warmup: readBoolean(item, 'warmup'),
    sampleCount: readNumber(item, 'sampleCount', 'sample_count')
});

const normalizeMoverToken = (item: any): PolymarketMoverTokenItem => {
    const windows = readValue(item, 'windows');
    return {
        tokenId: readString(item, 'tokenId', 'token_id'),
        outcome: readString(item, 'outcome'),
        price: readNumber(item, 'price'),
        bestBid: readNumber(item, 'bestBid', 'best_bid'),
        bestAsk: readNumber(item, 'bestAsk', 'best_ask'),
        spread: readNumber(item, 'spread'),
        lastTradePrice: readNumber(item, 'lastTradePrice', 'last_trade_price'),
        lastTradeSize: readNumber(item, 'lastTradeSize', 'last_trade_size'),
        lastTradeSide: readString(item, 'lastTradeSide', 'last_trade_side'),
        lastEventAt: readNumber(item, 'lastEventAt', 'last_event_at'),
        windows: Array.isArray(windows) ? windows.map(normalizeMoverWindow) : [],
        warmup: readBoolean(item, 'warmup'),
        score: readNumber(item, 'score'),
        direction: readString(item, 'direction')
    };
};

const normalizeMoverMarket = (item: any): PolymarketMoverMarketItem => {
    const tokens = readValue(item, 'tokens');
    const leader = readValue(item, 'leader');
    return {
        conditionId: readString(item, 'conditionId', 'condition_id'),
        marketSlug: readString(item, 'marketSlug', 'market_slug'),
        eventSlug: readString(item, 'eventSlug', 'event_slug'),
        question: readString(item, 'question'),
        image: readString(item, 'image'),
        volume24hr: readNumber(item, 'volume24hr', 'volume_24hr'),
        volumeNum: readNumber(item, 'volumeNum', 'volume_num'),
        liquidityNum: readNumber(item, 'liquidityNum', 'liquidity_num'),
        updatedAt: readString(item, 'updatedAt', 'updated_at'),
        tokens: Array.isArray(tokens) ? tokens.map(normalizeMoverToken) : [],
        leader: leader ? normalizeMoverToken(leader) : undefined,
        score: readNumber(item, 'score'),
        direction: readString(item, 'direction')
    };
};

export class PolymarketService {
    public listHotMarkets(limit = 100): Promise<ListPolymarketHotMarketsResult> & {abort?: () => void} {
        const req = requests.get('/polymarket/hot-markets').query({limit});
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: (body.items || []).map(normalizeHotMarket),
                fetchedAt: readNumber(body, 'fetchedAt', 'fetched_at'),
                stale: readBoolean(body, 'stale'),
                monitoredMarkets: readNumber(body, 'monitoredMarkets', 'monitored_markets'),
                monitoredTokens: readNumber(body, 'monitoredTokens', 'monitored_tokens'),
                candidateCount: readNumber(body, 'candidateCount', 'candidate_count')
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listRealtimeMarkets(limit = 100): Promise<ListPolymarketRealtimeMarketsResult> & {abort?: () => void} {
        const req = requests.get('/polymarket/realtime-markets').query({limit});
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: (body.items || []).map(normalizeRealtimeMarket),
                fetchedAt: readNumber(body, 'fetchedAt', 'fetched_at'),
                stale: readBoolean(body, 'stale'),
                subscribedMarkets: readNumber(body, 'subscribedMarkets', 'subscribed_markets'),
                subscribedTokens: readNumber(body, 'subscribedTokens', 'subscribed_tokens'),
                connected: readBoolean(body, 'connected'),
                lastEventAt: readNumber(body, 'lastEventAt', 'last_event_at'),
                candidateCount: readNumber(body, 'candidateCount', 'candidate_count')
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listMovers(limit = 100): Promise<ListPolymarketMoversResult> & {abort?: () => void} {
        const req = requests.get('/polymarket/movers').query({limit});
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: (body.items || []).map(normalizeMoverMarket),
                fetchedAt: readNumber(body, 'fetchedAt', 'fetched_at'),
                stale: readBoolean(body, 'stale'),
                connected: readBoolean(body, 'connected'),
                lastEventAt: readNumber(body, 'lastEventAt', 'last_event_at'),
                monitoredMarkets: readNumber(body, 'monitoredMarkets', 'monitored_markets'),
                monitoredTokens: readNumber(body, 'monitoredTokens', 'monitored_tokens'),
                candidateCount: readNumber(body, 'candidateCount', 'candidate_count')
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

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
