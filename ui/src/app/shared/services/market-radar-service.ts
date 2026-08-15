import requests from './requests';
import {readBoolean, readNumber, readString, readValue} from './api-values';

export interface MarketRadarHotMarketTokenItem {
    tokenId: string;
    outcome: string;
    price?: number;
}

export interface MarketRadarHotMarketItem {
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
    tokens: MarketRadarHotMarketTokenItem[];
}

export interface ListMarketRadarHotMarketsResult {
    items: MarketRadarHotMarketItem[];
    fetchedAt?: number;
    stale?: boolean;
    monitoredMarkets?: number;
    monitoredTokens?: number;
    candidateCount?: number;
}

export interface MarketRadarRealtimeWindowItem {
    window: string;
    priceChangePp?: number;
    warmup?: boolean;
    sampleCount?: number;
}

export interface MarketRadarRealtimeTokenItem {
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
    windows: MarketRadarRealtimeWindowItem[];
    warmup?: boolean;
}

export interface MarketRadarRealtimeMarketItem {
    conditionId: string;
    marketSlug: string;
    eventSlug: string;
    question: string;
    image?: string;
    volume24hr?: number;
    volumeNum?: number;
    liquidityNum?: number;
    updatedAt?: string;
    tokens: MarketRadarRealtimeTokenItem[];
}

export interface ListMarketRadarRealtimeMarketsResult {
    items: MarketRadarRealtimeMarketItem[];
    fetchedAt?: number;
    stale?: boolean;
    subscribedMarkets?: number;
    subscribedTokens?: number;
    connected?: boolean;
    lastEventAt?: number;
    candidateCount?: number;
}

export interface MarketRadarMoverWindowItem {
    window: string;
    priceChangePp?: number;
    warmup?: boolean;
    sampleCount?: number;
}

export interface MarketRadarMoverTokenItem {
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
    windows: MarketRadarMoverWindowItem[];
    warmup?: boolean;
    score?: number;
    direction?: string;
}

export interface MarketRadarMoverMarketItem {
    conditionId: string;
    marketSlug: string;
    eventSlug: string;
    question: string;
    image?: string;
    volume24hr?: number;
    volumeNum?: number;
    liquidityNum?: number;
    updatedAt?: string;
    tokens: MarketRadarMoverTokenItem[];
    leader?: MarketRadarMoverTokenItem;
    score?: number;
    direction?: string;
}

export interface ListMarketRadarMoversResult {
    items: MarketRadarMoverMarketItem[];
    fetchedAt?: number;
    stale?: boolean;
    connected?: boolean;
    lastEventAt?: number;
    monitoredMarkets?: number;
    monitoredTokens?: number;
    candidateCount?: number;
}

const normalizeHotMarketToken = (item: any): MarketRadarHotMarketTokenItem => ({
    tokenId: readString(item, 'tokenId', 'token_id'),
    outcome: readString(item, 'outcome'),
    price: readNumber(item, 'price')
});

const normalizeHotMarket = (item: any): MarketRadarHotMarketItem => {
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

const normalizeRealtimeWindow = (item: any): MarketRadarRealtimeWindowItem => ({
    window: readString(item, 'window'),
    priceChangePp: readNumber(item, 'priceChangePp', 'price_change_pp'),
    warmup: readBoolean(item, 'warmup'),
    sampleCount: readNumber(item, 'sampleCount', 'sample_count')
});

const normalizeRealtimeToken = (item: any): MarketRadarRealtimeTokenItem => {
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

const normalizeRealtimeMarket = (item: any): MarketRadarRealtimeMarketItem => {
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

const normalizeMoverWindow = (item: any): MarketRadarMoverWindowItem => ({
    window: readString(item, 'window'),
    priceChangePp: readNumber(item, 'priceChangePp', 'price_change_pp'),
    warmup: readBoolean(item, 'warmup'),
    sampleCount: readNumber(item, 'sampleCount', 'sample_count')
});

const normalizeMoverToken = (item: any): MarketRadarMoverTokenItem => {
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

const normalizeMoverMarket = (item: any): MarketRadarMoverMarketItem => {
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

export class MarketRadarService {
    public listHotMarkets(limit = 100): Promise<ListMarketRadarHotMarketsResult> & {abort?: () => void} {
        const req = requests.get('/market-radar/hot-markets').query({limit});
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
        }) as Promise<ListMarketRadarHotMarketsResult> & {abort?: () => void};
        promise.abort = () => req.abort();
        return promise;
    }

    public listRealtimeMarkets(limit = 100): Promise<ListMarketRadarRealtimeMarketsResult> & {abort?: () => void} {
        const req = requests.get('/market-radar/realtime-markets').query({limit});
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
        }) as Promise<ListMarketRadarRealtimeMarketsResult> & {abort?: () => void};
        promise.abort = () => req.abort();
        return promise;
    }

    public listMovers(limit = 100): Promise<ListMarketRadarMoversResult> & {abort?: () => void} {
        const req = requests.get('/market-radar/movers').query({limit});
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
        }) as Promise<ListMarketRadarMoversResult> & {abort?: () => void};
        promise.abort = () => req.abort();
        return promise;
    }
}
