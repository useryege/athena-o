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

export interface PolymarketSportsLiveMarketOptionItem {
    conditionId: string;
    marketSlug: string;
    question: string;
    outcomes: string[];
    outcomePrices: string[];
    bestBid?: number;
    bestAsk?: number;
    lastTradePrice?: number;
    volumeNum?: number;
    liquidityNum?: number;
}

export interface PolymarketSportsLiveMarketGroupItem {
    type: string;
    title: string;
    markets: PolymarketSportsLiveMarketOptionItem[];
}

export interface PolymarketSportsLiveTeamItem {
    name: string;
    logo?: string;
    ordering?: string;
}

export interface PolymarketSportsLiveEventItem {
    eventSlug: string;
    title: string;
    image?: string;
    score?: string;
    period?: string;
    elapsed?: string;
    lastUpdate?: string;
    live?: boolean;
    ended?: boolean;
    gameStatus?: string;
    startTime?: string;
    markets: PolymarketSportsLiveMarketGroupItem[];
    teams: PolymarketSportsLiveTeamItem[];
}

export interface GetPolymarketSportsLiveSnapshotResult {
    events: PolymarketSportsLiveEventItem[];
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
const readStringArray = (item: any, ...names: string[]) => {
    const value = readValue(item, ...names);
    if (!Array.isArray(value)) {
        return [];
    }
    return value.map(v => String(v ?? ''));
};

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

const normalizeMarketOption = (item: any): PolymarketSportsLiveMarketOptionItem => ({
    conditionId: readString(item, 'conditionId', 'condition_id'),
    marketSlug: readString(item, 'marketSlug', 'market_slug'),
    question: readString(item, 'question'),
    outcomes: readStringArray(item, 'outcomes'),
    outcomePrices: readStringArray(item, 'outcomePrices', 'outcome_prices'),
    bestBid: readNumber(item, 'bestBid', 'best_bid'),
    bestAsk: readNumber(item, 'bestAsk', 'best_ask'),
    lastTradePrice: readNumber(item, 'lastTradePrice', 'last_trade_price'),
    volumeNum: readNumber(item, 'volumeNum', 'volume_num'),
    liquidityNum: readNumber(item, 'liquidityNum', 'liquidity_num')
});

const normalizeMarketGroup = (item: any): PolymarketSportsLiveMarketGroupItem => {
    const markets = readValue(item, 'markets');
    return {
        type: readString(item, 'type'),
        title: readString(item, 'title'),
        markets: Array.isArray(markets) ? markets.map(normalizeMarketOption) : []
    };
};

const normalizeTeam = (item: any): PolymarketSportsLiveTeamItem => ({
    name: readString(item, 'name'),
    logo: readString(item, 'logo'),
    ordering: readString(item, 'ordering')
});

const normalizeEvent = (item: any): PolymarketSportsLiveEventItem => {
    const markets = readValue(item, 'markets');
    const teams = readValue(item, 'teams');
    return {
        eventSlug: readString(item, 'eventSlug', 'event_slug'),
        title: readString(item, 'title'),
        image: readString(item, 'image'),
        score: readString(item, 'score'),
        period: readString(item, 'period'),
        elapsed: readString(item, 'elapsed'),
        lastUpdate: readString(item, 'lastUpdate', 'last_update'),
        live: readBoolean(item, 'live'),
        ended: readBoolean(item, 'ended'),
        gameStatus: readString(item, 'gameStatus', 'game_status'),
        startTime: readString(item, 'startTime', 'start_time'),
        markets: Array.isArray(markets) ? markets.map(normalizeMarketGroup) : [],
        teams: Array.isArray(teams) ? teams.map(normalizeTeam) : []
    };
};

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

    public getSportsLiveSnapshot(limit = 30): Promise<GetPolymarketSportsLiveSnapshotResult> & {abort?: () => void} {
        const req = requests.get('/polymarket/sports/live/snapshot').query({limit});
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                events: (body.events || []).map(normalizeEvent),
                fetchedAt: readNumber(body, 'fetchedAt', 'fetched_at'),
                stale: readBoolean(body, 'stale')
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }
}
