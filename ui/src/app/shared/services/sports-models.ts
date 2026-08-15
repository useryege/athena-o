import {readNumber, readNumberArray, readString, readValue} from './api-values';

export interface SportsLiveMarketCardItem {
    marketKey: string;
    conditionId: string;
    slug: string;
    sportsMarketType: string;
    question: string;
    outcomes?: string;
    outcomePrices?: string;
    bestBid?: number;
    bestAsk?: number;
    lastTradePrice?: number;
    spread?: number;
    liquidityNum?: number;
    volumeNum?: number;
    updatedAt?: string;
}

export interface SportsLiveTeamItem {
    name: string;
    logo?: string;
    abbreviation?: string;
    alias?: string;
}

export interface SportsLiveEventCardItem {
    eventKey: string;
    eventId: string;
    slug: string;
    title: string;
    image?: string;
    score?: string;
    period?: string;
    elapsed?: string;
    gameStatus?: string;
    startTime?: string;
    updatedAt?: string;
    liquidity?: number;
    volume?: number;
    marketCount?: number;
    markets: SportsLiveMarketCardItem[];
    teams: SportsLiveTeamItem[];
}

export interface SportsHistoryEventCardItem extends SportsLiveEventCardItem {
    league: string;
    finishedAt?: string;
}

export interface SportsPriceHistorySeriesItem {
    marketKey: string;
    tokenId: string;
    outcome: string;
    timestamps: number[];
    prices: number[];
}

const normalizeSportsLiveMarketCard = (item: any): SportsLiveMarketCardItem => ({
    marketKey: readString(item, 'marketKey', 'market_key'),
    conditionId: readString(item, 'conditionId', 'condition_id'),
    slug: readString(item, 'slug'),
    sportsMarketType: readString(item, 'sportsMarketType', 'sports_market_type'),
    question: readString(item, 'question'),
    outcomes: readString(item, 'outcomes'),
    outcomePrices: readString(item, 'outcomePrices', 'outcome_prices'),
    bestBid: readNumber(item, 'bestBid', 'best_bid'),
    bestAsk: readNumber(item, 'bestAsk', 'best_ask'),
    lastTradePrice: readNumber(item, 'lastTradePrice', 'last_trade_price'),
    spread: readNumber(item, 'spread'),
    liquidityNum: readNumber(item, 'liquidityNum', 'liquidity_num'),
    volumeNum: readNumber(item, 'volumeNum', 'volume_num'),
    updatedAt: readString(item, 'updatedAt', 'updated_at')
});

const normalizeSportsLiveTeam = (item: any): SportsLiveTeamItem => ({
    name: readString(item, 'name'),
    logo: readString(item, 'logo'),
    abbreviation: readString(item, 'abbreviation'),
    alias: readString(item, 'alias')
});

export const normalizeSportsLiveEventCard = (item: any): SportsLiveEventCardItem => {
    const markets = readValue(item, 'markets');
    const teams = readValue(item, 'teams');
    return {
        eventKey: readString(item, 'eventKey', 'event_key'),
        eventId: readString(item, 'eventId', 'event_id'),
        slug: readString(item, 'slug'),
        title: readString(item, 'title'),
        image: readString(item, 'image'),
        score: readString(item, 'score'),
        period: readString(item, 'period'),
        elapsed: readString(item, 'elapsed'),
        gameStatus: readString(item, 'gameStatus', 'game_status'),
        startTime: readString(item, 'startTime', 'start_time'),
        updatedAt: readString(item, 'updatedAt', 'updated_at'),
        liquidity: readNumber(item, 'liquidity'),
        volume: readNumber(item, 'volume'),
        marketCount: readNumber(item, 'marketCount', 'market_count'),
        markets: Array.isArray(markets) ? markets.map(normalizeSportsLiveMarketCard) : [],
        teams: Array.isArray(teams) ? teams.map(normalizeSportsLiveTeam) : []
    };
};

export const normalizeSportsHistoryEventCard = (item: any): SportsHistoryEventCardItem => ({
    ...normalizeSportsLiveEventCard(item),
    league: readString(item, 'league'),
    finishedAt: readString(item, 'finishedAt', 'finished_at')
});

export const normalizeSportsPriceHistorySeries = (item: any): SportsPriceHistorySeriesItem => ({
    marketKey: readString(item, 'marketKey', 'market_key'),
    tokenId: readString(item, 'tokenId', 'token_id'),
    outcome: readString(item, 'outcome'),
    timestamps: readNumberArray(item, 'timestamps'),
    prices: readNumberArray(item, 'prices')
});
