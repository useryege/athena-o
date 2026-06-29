import requests from './requests';
import type {WormEventItem, WormMarketItem} from './worm-service';

export interface WormPolyFIFAEventConfig {
    wormEventId: string;
    eventRef: string;
}

export interface PolymarketFIFAMoneylineDirectionItem {
    tokenId: string;
    midPrice?: number;
    bestBid?: number;
    bestAsk?: number;
    spread?: number;
}

export interface PolymarketFIFAMoneylineOptionItem {
    outcomeKey: string;
    outcomeLabel: string;
    marketId: string;
    marketSlug: string;
    question: string;
    conditionId: string;
    yes: PolymarketFIFAMoneylineDirectionItem;
    no: PolymarketFIFAMoneylineDirectionItem;
    orderMinSize?: number;
    tickSize?: number;
    enableOrderBook?: boolean;
    acceptingOrders?: boolean;
    negRisk?: boolean;
}

export interface PolymarketSportsLiveTeamItem {
    name: string;
    logo?: string;
    abbreviation?: string;
    alias?: string;
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

export interface PolymarketFIFAWalletHoldingItem {
    walletId?: number;
    chain: string;
    type: string;
    alias: string;
    walletAddress: string;
    solRawAmount: string;
    solAmount: string;
    usdcRawAmount: string;
    usdcAmount: string;
    explorerUrl: string;
    ok?: boolean;
    errorMessage?: string;
}

export interface WormPolyFIFADashboard {
    config?: WormPolyFIFAEventConfig;
    wormEvent?: WormEventItem;
    wormFetchedAt?: number;
    wormError?: string;
    polymarketEvent?: PolymarketFIFAMoneylineEventItem;
    polymarketFetchedAt?: number;
    polymarketError?: string;
    walletBalances: PolymarketFIFAWalletBalanceItem[];
    walletBalancesFetchedAt?: number;
    walletBalancesError?: string;
    walletHoldings: PolymarketFIFAWalletHoldingItem[];
    walletHoldingsFetchedAt?: number;
    walletHoldingsError?: string;
    fetchedAt?: number;
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

const readBoolean = (item: any, ...names: string[]) => {
    const value = readValue(item, ...names);
    return value === true || value === 'true' || value === 1 || value === '1';
};

const normalizeFIFAEventConfig = (item: any): WormPolyFIFAEventConfig => ({
    wormEventId: readString(item, 'wormEventId', 'worm_event_id'),
    eventRef: readString(item, 'eventRef', 'event_ref')
});

const normalizeWormEstimate = (item: any) => {
    if (!item) {
        return undefined;
    }
    return {
        funds: readString(item, 'funds'),
        isYes: readBoolean(item, 'isYes', 'is_yes'),
        leverage: readString(item, 'leverage'),
        averagePrice: readString(item, 'averagePrice', 'average_price'),
        totalShares: readString(item, 'totalShares', 'total_shares'),
        totalCost: readString(item, 'totalCost', 'total_cost'),
        bestAsk: readString(item, 'bestAsk', 'best_ask'),
        worstFillPrice: readString(item, 'worstFillPrice', 'worst_fill_price'),
        isFullyFilled: readBoolean(item, 'isFullyFilled', 'is_fully_filled'),
        feeAmount: readString(item, 'feeAmount', 'fee_amount'),
        userFundsNeeded: readString(item, 'userFundsNeeded', 'user_funds_needed'),
        liquidationPrice: readString(item, 'liquidationPrice', 'liquidation_price')
    };
};

const normalizeWormMarket = (item: any): WormMarketItem => ({
    conditionId: readString(item, 'conditionId', 'condition_id'),
    title: readString(item, 'title'),
    description: readString(item, 'description'),
    logo: readString(item, 'logo'),
    lastTradePrice: readString(item, 'lastTradePrice', 'last_trade_price'),
    state: readString(item, 'state'),
    category: readString(item, 'category'),
    created: readNumber(item, 'created'),
    eventTitle: readString(item, 'eventTitle', 'event_title'),
    eventConditionId: readString(item, 'eventConditionId', 'event_condition_id'),
    eventLogo: readString(item, 'eventLogo', 'event_logo'),
    marginEnabled: readBoolean(item, 'marginEnabled', 'margin_enabled'),
    liveState: readString(item, 'liveState', 'live_state') as WormMarketItem['liveState'],
    liveCheckedAt: readNumber(item, 'liveCheckedAt', 'live_checked_at'),
    livePriceChange: readString(item, 'livePriceChange', 'live_price_change'),
    configKind: readString(item, 'configKind', 'config_kind'),
    maxLeverageYes: readString(item, 'maxLeverageYes', 'max_leverage_yes'),
    maxLeverageNo: readString(item, 'maxLeverageNo', 'max_leverage_no'),
    openingFee: readString(item, 'openingFee', 'opening_fee'),
    closingFee: readString(item, 'closingFee', 'closing_fee'),
    annualFeeRate: readString(item, 'annualFeeRate', 'annual_fee_rate'),
    orderMinSize: readString(item, 'orderMinSize', 'order_min_size'),
    priceDecimals: readNumber(item, 'priceDecimals', 'price_decimals'),
    sharesDecimals: readNumber(item, 'sharesDecimals', 'shares_decimals'),
    estimate: normalizeWormEstimate(readValue(item, 'estimate')),
    tradingDataError: readString(item, 'tradingDataError', 'trading_data_error')
});

const normalizeWormEvent = (item: any): WormEventItem => {
    const markets = readValue(item, 'markets');
    const normalizedMarkets = Array.isArray(markets) ? markets.map(normalizeWormMarket) : [];
    return {
        conditionId: readString(item, 'conditionId', 'condition_id'),
        title: readString(item, 'title'),
        description: readString(item, 'description'),
        logo: readString(item, 'logo'),
        category: readString(item, 'category'),
        created: readNumber(item, 'created'),
        live: readBoolean(item, 'live'),
        marketCount: readNumber(item, 'marketCount', 'market_count') || normalizedMarkets.length,
        markets: normalizedMarkets
    };
};

const normalizeSportsLiveTeam = (item: any): PolymarketSportsLiveTeamItem => ({
    name: readString(item, 'name'),
    logo: readString(item, 'logo'),
    abbreviation: readString(item, 'abbreviation'),
    alias: readString(item, 'alias')
});

const normalizeFIFAMoneylineDirection = (item: any): PolymarketFIFAMoneylineDirectionItem => ({
    tokenId: readString(item, 'tokenId', 'token_id'),
    midPrice: readNumber(item, 'midPrice', 'mid_price'),
    bestBid: readNumber(item, 'bestBid', 'best_bid'),
    bestAsk: readNumber(item, 'bestAsk', 'best_ask'),
    spread: readNumber(item, 'spread')
});

const normalizeFIFAMoneylineOption = (item: any): PolymarketFIFAMoneylineOptionItem => ({
    outcomeKey: readString(item, 'outcomeKey', 'outcome_key'),
    outcomeLabel: readString(item, 'outcomeLabel', 'outcome_label'),
    marketId: readString(item, 'marketId', 'market_id'),
    marketSlug: readString(item, 'marketSlug', 'market_slug'),
    question: readString(item, 'question'),
    conditionId: readString(item, 'conditionId', 'condition_id'),
    yes: normalizeFIFAMoneylineDirection(readValue(item, 'yes')),
    no: normalizeFIFAMoneylineDirection(readValue(item, 'no')),
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

const normalizeFIFAWalletHolding = (item: any): PolymarketFIFAWalletHoldingItem => ({
    walletId: readNumber(item, 'walletId', 'wallet_id'),
    chain: readString(item, 'chain'),
    type: readString(item, 'type'),
    alias: readString(item, 'alias'),
    walletAddress: readString(item, 'walletAddress', 'wallet_address'),
    solRawAmount: readString(item, 'solRawAmount', 'sol_raw_amount'),
    solAmount: readString(item, 'solAmount', 'sol_amount'),
    usdcRawAmount: readString(item, 'usdcRawAmount', 'usdc_raw_amount'),
    usdcAmount: readString(item, 'usdcAmount', 'usdc_amount'),
    explorerUrl: readString(item, 'explorerUrl', 'explorer_url'),
    ok: readBoolean(item, 'ok'),
    errorMessage: readString(item, 'errorMessage', 'error_message')
});

const normalizeDashboard = (item: any): WormPolyFIFADashboard => {
    const walletBalances = readValue(item, 'walletBalances', 'wallet_balances');
    const walletHoldings = readValue(item, 'walletHoldings', 'wallet_holdings');
    return {
        config: readValue(item, 'config') ? normalizeFIFAEventConfig(readValue(item, 'config')) : undefined,
        wormEvent: readValue(item, 'wormEvent', 'worm_event') ? normalizeWormEvent(readValue(item, 'wormEvent', 'worm_event')) : undefined,
        wormFetchedAt: readNumber(item, 'wormFetchedAt', 'worm_fetched_at'),
        wormError: readString(item, 'wormError', 'worm_error'),
        polymarketEvent: readValue(item, 'polymarketEvent', 'polymarket_event') ? normalizeFIFAMoneylineEvent(readValue(item, 'polymarketEvent', 'polymarket_event')) : undefined,
        polymarketFetchedAt: readNumber(item, 'polymarketFetchedAt', 'polymarket_fetched_at'),
        polymarketError: readString(item, 'polymarketError', 'polymarket_error'),
        walletBalances: Array.isArray(walletBalances) ? walletBalances.map(normalizeFIFAWalletBalance) : [],
        walletBalancesFetchedAt: readNumber(item, 'walletBalancesFetchedAt', 'wallet_balances_fetched_at'),
        walletBalancesError: readString(item, 'walletBalancesError', 'wallet_balances_error'),
        walletHoldings: Array.isArray(walletHoldings) ? walletHoldings.map(normalizeFIFAWalletHolding) : [],
        walletHoldingsFetchedAt: readNumber(item, 'walletHoldingsFetchedAt', 'wallet_holdings_fetched_at'),
        walletHoldingsError: readString(item, 'walletHoldingsError', 'wallet_holdings_error'),
        fetchedAt: readNumber(item, 'fetchedAt', 'fetched_at')
    };
};

export class WormPolyService {
    public getFIFADashboard(): Promise<WormPolyFIFADashboard> & {abort?: () => void} {
        const req = requests.get('/worm-poly/fifa/dashboard');
        const promise = req.then(res => normalizeDashboard(res.body?.dashboard || {})) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public updateFIFAEventConfig(config: WormPolyFIFAEventConfig): Promise<WormPolyFIFAEventConfig> & {abort?: () => void} {
        const req = requests.put('/worm-poly/fifa/event-config').send({wormEventId: config.wormEventId, eventRef: config.eventRef});
        const promise = req.then(res => normalizeFIFAEventConfig(res.body?.config || {})) as any;
        promise.abort = () => req.abort();
        return promise;
    }
}
