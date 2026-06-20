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

export interface PolymarketSportsLiveMarketCardItem {
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

export interface PolymarketSportsLiveTeamItem {
    name: string;
    logo?: string;
    abbreviation?: string;
    alias?: string;
}

export interface PolymarketFIFAMoneylineOptionItem {
    outcomeKey: string;
    outcomeLabel: string;
    marketId: string;
    marketSlug: string;
    question: string;
    conditionId: string;
    yesTokenId: string;
    noTokenId?: string;
    outcomePrice?: number;
    midPrice?: number;
    bestBid?: number;
    bestAsk?: number;
    lastTradePrice?: number;
    spread?: number;
    orderMinSize?: number;
    tickSize?: number;
    enableOrderBook?: boolean;
    acceptingOrders?: boolean;
    negRisk?: boolean;
}

export interface PolymarketFIFAMoneylineEventItem {
    eventId: string;
    eventSlug: string;
    title: string;
    image?: string;
    sport?: string;
    score?: string;
    gameStatus?: string;
    startTime?: string;
    updatedAt?: string;
    polymarketUrl?: string;
    active?: boolean;
    closed?: boolean;
    live?: boolean;
    ended?: boolean;
    teams: PolymarketSportsLiveTeamItem[];
    options: PolymarketFIFAMoneylineOptionItem[];
}

export interface GetPolymarketFIFAMoneylineEventResult {
    item?: PolymarketFIFAMoneylineEventItem;
    fetchedAt?: number;
}

export interface PolymarketFIFAWalletBalanceItem {
    chain: string;
    label: string;
    walletAddress: string;
    tokenAddress: string;
    tokenSymbol: string;
    decimals?: number;
    rawAmount: string;
    amount: string;
    explorerUrl: string;
    ok?: boolean;
    errorMessage?: string;
}

export interface ListPolymarketFIFAWalletBalancesResult {
    items: PolymarketFIFAWalletBalanceItem[];
    fetchedAt?: number;
}

export interface PolymarketUMABaseItem {
    txHash: string;
    logIndex: number;
    blockNumber: number;
    blockHash: string;
    txIndex: number;
    contractAddress: string;
    topic: string;
    requester: string;
    proposer: string;
    identifier: string;
    requestTimestamp: number;
    ancillaryDataHex: string;
    ancillaryDataText: string;
    marketId: string;
    proposedPrice: string;
    rawTopics: string;
    rawData: string;
    fetchedAt: string;
    conditionId: string;
    eventSlug: string;
    marketSlug: string;
    question: string;
    polymarketUrl: string;
}

export interface PolymarketUMAProposalItem extends PolymarketUMABaseItem {
    expirationTimestamp: number;
    currency: string;
}

export interface PolymarketUMADisputeItem extends PolymarketUMABaseItem {
    disputer: string;
}

export interface ListPolymarketUMAResult<T extends PolymarketUMABaseItem> {
    items: T[];
    total: number;
    page: number;
    pageSize: number;
}

export interface ScanPolymarketManagedOOBlockResult {
    blockNumber: number;
    proposalCount: number;
    disputeCount: number;
}

export interface PolymarketSportsLiveEventCardItem {
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
    markets: PolymarketSportsLiveMarketCardItem[];
    teams: PolymarketSportsLiveTeamItem[];
}

export interface PolymarketSportsHistoryEventCardItem extends PolymarketSportsLiveEventCardItem {
    league: string;
    finishedAt?: string;
}

export interface PolymarketSportsLivePriceHistorySeriesItem {
    marketKey: string;
    tokenId: string;
    outcome: string;
    timestamps: number[];
    prices: number[];
}

export interface ListPolymarketSportsLiveEventsResult {
    items: PolymarketSportsLiveEventCardItem[];
    fetchedAt?: number;
    stale?: boolean;
}

export interface BatchGetPolymarketSportsLivePriceHistoryResult {
    items: PolymarketSportsLivePriceHistorySeriesItem[];
}

export interface ListPolymarketSportsHistoryEventsResult {
    items: PolymarketSportsHistoryEventCardItem[];
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
const readNumberArray = (item: any, ...names: string[]) => {
    const value = readValue(item, ...names);
    if (!Array.isArray(value)) {
        return [];
    }
    return value.map(next => Number(next)).filter(next => Number.isFinite(next));
};

const normalizeSportsLiveMarketCard = (item: any): PolymarketSportsLiveMarketCardItem => ({
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

const normalizeSportsLiveTeam = (item: any): PolymarketSportsLiveTeamItem => ({
    name: readString(item, 'name'),
    logo: readString(item, 'logo'),
    abbreviation: readString(item, 'abbreviation'),
    alias: readString(item, 'alias')
});

const normalizeFIFAMoneylineOption = (item: any): PolymarketFIFAMoneylineOptionItem => ({
    outcomeKey: readString(item, 'outcomeKey', 'outcome_key'),
    outcomeLabel: readString(item, 'outcomeLabel', 'outcome_label'),
    marketId: readString(item, 'marketId', 'market_id'),
    marketSlug: readString(item, 'marketSlug', 'market_slug'),
    question: readString(item, 'question'),
    conditionId: readString(item, 'conditionId', 'condition_id'),
    yesTokenId: readString(item, 'yesTokenId', 'yes_token_id'),
    noTokenId: readString(item, 'noTokenId', 'no_token_id'),
    outcomePrice: readNumber(item, 'outcomePrice', 'outcome_price'),
    midPrice: readNumber(item, 'midPrice', 'mid_price'),
    bestBid: readNumber(item, 'bestBid', 'best_bid'),
    bestAsk: readNumber(item, 'bestAsk', 'best_ask'),
    lastTradePrice: readNumber(item, 'lastTradePrice', 'last_trade_price'),
    spread: readNumber(item, 'spread'),
    orderMinSize: readNumber(item, 'orderMinSize', 'order_min_size'),
    tickSize: readNumber(item, 'tickSize', 'tick_size'),
    enableOrderBook: readBoolean(item, 'enableOrderBook', 'enable_order_book'),
    acceptingOrders: readBoolean(item, 'acceptingOrders', 'accepting_orders'),
    negRisk: readBoolean(item, 'negRisk', 'neg_risk')
});

const normalizeFIFAMoneylineEvent = (item: any): PolymarketFIFAMoneylineEventItem => {
    const teams = readValue(item, 'teams');
    const options = readValue(item, 'options');
    return {
        eventId: readString(item, 'eventId', 'event_id'),
        eventSlug: readString(item, 'eventSlug', 'event_slug'),
        title: readString(item, 'title'),
        image: readString(item, 'image'),
        sport: readString(item, 'sport'),
        score: readString(item, 'score'),
        gameStatus: readString(item, 'gameStatus', 'game_status'),
        startTime: readString(item, 'startTime', 'start_time'),
        updatedAt: readString(item, 'updatedAt', 'updated_at'),
        polymarketUrl: readString(item, 'polymarketUrl', 'polymarket_url'),
        active: readBoolean(item, 'active'),
        closed: readBoolean(item, 'closed'),
        live: readBoolean(item, 'live'),
        ended: readBoolean(item, 'ended'),
        teams: Array.isArray(teams) ? teams.map(normalizeSportsLiveTeam) : [],
        options: Array.isArray(options) ? options.map(normalizeFIFAMoneylineOption) : []
    };
};

const normalizeFIFAWalletBalance = (item: any): PolymarketFIFAWalletBalanceItem => ({
    chain: readString(item, 'chain'),
    label: readString(item, 'label'),
    walletAddress: readString(item, 'walletAddress', 'wallet_address'),
    tokenAddress: readString(item, 'tokenAddress', 'token_address'),
    tokenSymbol: readString(item, 'tokenSymbol', 'token_symbol'),
    decimals: readNumber(item, 'decimals'),
    rawAmount: readString(item, 'rawAmount', 'raw_amount'),
    amount: readString(item, 'amount'),
    explorerUrl: readString(item, 'explorerUrl', 'explorer_url'),
    ok: readBoolean(item, 'ok'),
    errorMessage: readString(item, 'errorMessage', 'error_message')
});

const normalizeSportsLiveEventCard = (item: any): PolymarketSportsLiveEventCardItem => {
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

const normalizeSportsHistoryEventCard = (item: any): PolymarketSportsHistoryEventCardItem => ({
    ...normalizeSportsLiveEventCard(item),
    league: readString(item, 'league'),
    finishedAt: readString(item, 'finishedAt', 'finished_at')
});

const normalizeSportsLivePriceHistorySeries = (item: any): PolymarketSportsLivePriceHistorySeriesItem => ({
    marketKey: readString(item, 'marketKey', 'market_key'),
    tokenId: readString(item, 'tokenId', 'token_id'),
    outcome: readString(item, 'outcome'),
    timestamps: readNumberArray(item, 'timestamps'),
    prices: readNumberArray(item, 'prices')
});

const normalizeUMABase = (item: any): PolymarketUMABaseItem => ({
    txHash: readString(item, 'txHash', 'tx_hash'),
    logIndex: readNumber(item, 'logIndex', 'log_index') || 0,
    blockNumber: readNumber(item, 'blockNumber', 'block_number') || 0,
    blockHash: readString(item, 'blockHash', 'block_hash'),
    txIndex: readNumber(item, 'txIndex', 'tx_index') || 0,
    contractAddress: readString(item, 'contractAddress', 'contract_address'),
    topic: readString(item, 'topic'),
    requester: readString(item, 'requester'),
    proposer: readString(item, 'proposer'),
    identifier: readString(item, 'identifier'),
    requestTimestamp: readNumber(item, 'requestTimestamp', 'request_timestamp') || 0,
    ancillaryDataHex: readString(item, 'ancillaryDataHex', 'ancillary_data_hex'),
    ancillaryDataText: readString(item, 'ancillaryDataText', 'ancillary_data_text'),
    marketId: readString(item, 'marketId', 'market_id'),
    proposedPrice: readString(item, 'proposedPrice', 'proposed_price'),
    rawTopics: readString(item, 'rawTopics', 'raw_topics'),
    rawData: readString(item, 'rawData', 'raw_data'),
    fetchedAt: readString(item, 'fetchedAt', 'fetched_at'),
    conditionId: readString(item, 'conditionId', 'condition_id'),
    eventSlug: readString(item, 'eventSlug', 'event_slug'),
    marketSlug: readString(item, 'marketSlug', 'market_slug'),
    question: readString(item, 'question'),
    polymarketUrl: readString(item, 'polymarketUrl', 'polymarket_url')
});

const normalizeUMAProposal = (item: any): PolymarketUMAProposalItem => ({
    ...normalizeUMABase(item),
    expirationTimestamp: readNumber(item, 'expirationTimestamp', 'expiration_timestamp') || 0,
    currency: readString(item, 'currency')
});

const normalizeUMADispute = (item: any): PolymarketUMADisputeItem => ({
    ...normalizeUMABase(item),
    disputer: readString(item, 'disputer')
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
    public scanManagedOOBlock(blockNumber: number): Promise<ScanPolymarketManagedOOBlockResult> & {abort?: () => void} {
        const req = requests.post(`/polymarket/uma/blocks/${blockNumber}:scan`).send({});
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                blockNumber: readNumber(body, 'blockNumber', 'block_number') || blockNumber,
                proposalCount: readNumber(body, 'proposalCount', 'proposal_count') || 0,
                disputeCount: readNumber(body, 'disputeCount', 'dispute_count') || 0
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listUMAProposals(page = 1, pageSize = 20, blockNumber?: number): Promise<ListPolymarketUMAResult<PolymarketUMAProposalItem>> & {abort?: () => void} {
        return this.listUMA('/polymarket/uma/proposals', normalizeUMAProposal, page, pageSize, blockNumber);
    }

    public listUMADisputes(page = 1, pageSize = 20, blockNumber?: number): Promise<ListPolymarketUMAResult<PolymarketUMADisputeItem>> & {abort?: () => void} {
        return this.listUMA('/polymarket/uma/disputes', normalizeUMADispute, page, pageSize, blockNumber);
    }

    private listUMA<T extends PolymarketUMABaseItem>(
        path: string,
        normalize: (item: any) => T,
        page: number,
        pageSize: number,
        blockNumber?: number
    ): Promise<ListPolymarketUMAResult<T>> & {abort?: () => void} {
        const query: Record<string, number> = {page, page_size: pageSize};
        if (blockNumber) {
            query.block_number = blockNumber;
        }
        const req = requests.get(path).query(query);
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: (body.items || []).map(normalize),
                total: readNumber(body, 'total') || 0,
                page: readNumber(body, 'page') || page,
                pageSize: readNumber(body, 'pageSize', 'page_size') || pageSize
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

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

    public listSportsLiveEvents(limit = 200): Promise<ListPolymarketSportsLiveEventsResult> & {abort?: () => void} {
        const req = requests.get('/polymarket/sports/live/events').query({limit});
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: (body.items || []).map(normalizeSportsLiveEventCard),
                fetchedAt: readNumber(body, 'fetchedAt', 'fetched_at'),
                stale: readBoolean(body, 'stale')
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public batchGetSportsLivePriceHistory(marketKeys: string[], limitPerToken = 360): Promise<BatchGetPolymarketSportsLivePriceHistoryResult> & {abort?: () => void} {
        const req = requests.post('/polymarket/sports/live/price-history:batchGet').send({market_keys: marketKeys, limit_per_token: limitPerToken});
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: (body.items || []).map(normalizeSportsLivePriceHistorySeries)
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listSportsHistoryEvents(limit = 200): Promise<ListPolymarketSportsHistoryEventsResult> & {abort?: () => void} {
        const req = requests.get('/polymarket/sports/history/events').query({limit});
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: (body.items || []).map(normalizeSportsHistoryEventCard),
                fetchedAt: readNumber(body, 'fetchedAt', 'fetched_at'),
                stale: readBoolean(body, 'stale')
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public batchGetSportsHistoryPriceHistory(marketKeys: string[], limitPerToken = 360): Promise<BatchGetPolymarketSportsLivePriceHistoryResult> & {abort?: () => void} {
        const req = requests.post('/polymarket/sports/history/price-history:batchGet').send({market_keys: marketKeys, limit_per_token: limitPerToken});
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: (body.items || []).map(normalizeSportsLivePriceHistorySeries)
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public getFIFAMoneylineEvent(eventRef: string): Promise<GetPolymarketFIFAMoneylineEventResult> & {abort?: () => void} {
        const req = requests.get('/polymarket/fifa/moneyline-event').query({event_ref: eventRef});
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                item: body.item ? normalizeFIFAMoneylineEvent(body.item) : undefined,
                fetchedAt: readNumber(body, 'fetchedAt', 'fetched_at')
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listFIFAWalletBalances(): Promise<ListPolymarketFIFAWalletBalancesResult> & {abort?: () => void} {
        const req = requests.get('/polymarket/fifa/wallet-balances');
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: (body.items || []).map(normalizeFIFAWalletBalance),
                fetchedAt: readNumber(body, 'fetchedAt', 'fetched_at')
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }
}
