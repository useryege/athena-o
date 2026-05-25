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

export type WormMarketSortOption = 'new' | 'trending' | 'ending_soon' | 'leverage';
export type WormMarketCategorySlug = 'all' | 'politics' | 'sports' | 'crypto' | 'tech' | 'finance' | 'wtf';

export interface ListWormMarketsOptions {
    limit?: number;
    cursor?: string;
    sortOption?: WormMarketSortOption;
    categorySlug?: WormMarketCategorySlug;
}

export interface WormMarketOutcome {
    isYes: boolean;
    text: string;
}

export interface WormMarketConfig {
    kind?: string;
    maxLeverage?: string;
    openingFee?: string;
    closingFee?: string;
    annualFeeRate?: string;
    orderMinSize?: string;
    priceDecimals?: number;
    sharesDecimals?: number;
    minPrice?: string;
    maxPrice?: string;
    minAmount?: string;
    maxAmount?: string;
    minFunds?: string;
    maxFunds?: string;
    pricePrecision?: number;
    amountPrecision?: number;
    fundsPrecision?: number;
    makerFeeRate?: string;
    takerFeeRate?: string;
    defaultSlippageRate?: string;
}

export interface WormMarketStats {
    totalVolume?: string;
    totalVolume24H?: string;
    marketCap?: string;
    tradeCount?: number;
}

export interface WormMarketPrice {
    conditionId: string;
    price?: string;
    priceKind?: string;
    isYes: boolean;
}

export interface WormOrderBookLevel {
    price: string;
    totalAmount: string;
}

export interface WormMarketOrderBook {
    market: string;
    isYes: boolean;
    bid: WormOrderBookLevel[];
    ask: WormOrderBookLevel[];
}

export interface WormMarketDetail {
    market: WormMarketItem;
    yesOutcomeLabel?: string;
    noOutcomeLabel?: string;
    outcomes: WormMarketOutcome[];
    rules: string[];
    resolutionDate?: number;
    makerFee?: string;
    takerFee?: string;
    config: WormMarketConfig;
    stats: WormMarketStats;
    prices: WormMarketPrice[];
    orderBooks: WormMarketOrderBook[];
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
    marginEnabled: readBoolean(item, 'marginEnabled', 'margin_enabled')
});

const normalizeOutcome = (item: any): WormMarketOutcome => ({
    isYes: readBoolean(item, 'isYes', 'is_yes'),
    text: readString(item, 'text', 'text')
});

const normalizeConfig = (item: any = {}): WormMarketConfig => ({
    kind: readString(item, 'kind', 'kind'),
    maxLeverage: readString(item, 'maxLeverage', 'max_leverage'),
    openingFee: readString(item, 'openingFee', 'opening_fee'),
    closingFee: readString(item, 'closingFee', 'closing_fee'),
    annualFeeRate: readString(item, 'annualFeeRate', 'annual_fee_rate'),
    orderMinSize: readString(item, 'orderMinSize', 'order_min_size'),
    priceDecimals: readNumber(item, 'priceDecimals', 'price_decimals'),
    sharesDecimals: readNumber(item, 'sharesDecimals', 'shares_decimals'),
    minPrice: readString(item, 'minPrice', 'min_price'),
    maxPrice: readString(item, 'maxPrice', 'max_price'),
    minAmount: readString(item, 'minAmount', 'min_amount'),
    maxAmount: readString(item, 'maxAmount', 'max_amount'),
    minFunds: readString(item, 'minFunds', 'min_funds'),
    maxFunds: readString(item, 'maxFunds', 'max_funds'),
    pricePrecision: readNumber(item, 'pricePrecision', 'price_precision'),
    amountPrecision: readNumber(item, 'amountPrecision', 'amount_precision'),
    fundsPrecision: readNumber(item, 'fundsPrecision', 'funds_precision'),
    makerFeeRate: readString(item, 'makerFeeRate', 'maker_fee_rate'),
    takerFeeRate: readString(item, 'takerFeeRate', 'taker_fee_rate'),
    defaultSlippageRate: readString(item, 'defaultSlippageRate', 'default_slippage_rate')
});

const normalizeStats = (item: any = {}): WormMarketStats => ({
    totalVolume: readString(item, 'totalVolume', 'total_volume'),
    totalVolume24H: readString(item, 'totalVolume24H', 'totalVolume24h', 'total_volume_24h'),
    marketCap: readString(item, 'marketCap', 'market_cap'),
    tradeCount: readNumber(item, 'tradeCount', 'trade_count')
});

const normalizePrice = (item: any): WormMarketPrice => ({
    conditionId: readString(item, 'conditionId', 'condition_id'),
    price: readString(item, 'price', 'price'),
    priceKind: readString(item, 'priceKind', 'price_kind'),
    isYes: readBoolean(item, 'isYes', 'is_yes')
});

const normalizeOrderBookLevel = (item: any): WormOrderBookLevel => ({
    price: readString(item, 'price', 'price'),
    totalAmount: readString(item, 'totalAmount', 'total_amount')
});

const normalizeOrderBook = (item: any): WormMarketOrderBook => ({
    market: readString(item, 'market', 'market'),
    isYes: readBoolean(item, 'isYes', 'is_yes'),
    bid: (item?.bid || []).map(normalizeOrderBookLevel),
    ask: (item?.ask || []).map(normalizeOrderBookLevel)
});

const normalizeMarketDetail = (item: any): WormMarketDetail => ({
    market: normalizeMarket(item?.market || {}),
    yesOutcomeLabel: readString(item, 'yesOutcomeLabel', 'yes_outcome_label'),
    noOutcomeLabel: readString(item, 'noOutcomeLabel', 'no_outcome_label'),
    outcomes: (item?.outcomes || []).map(normalizeOutcome),
    rules: (item?.rules || []).map((rule: any) => String(rule || '')).filter((rule: string) => rule),
    resolutionDate: readNumber(item, 'resolutionDate', 'resolution_date'),
    makerFee: readString(item, 'makerFee', 'maker_fee'),
    takerFee: readString(item, 'takerFee', 'taker_fee'),
    config: normalizeConfig(item?.config || {}),
    stats: normalizeStats(item?.stats || {}),
    prices: (item?.prices || []).map(normalizePrice),
    orderBooks: (item?.orderBooks || item?.order_books || []).map(normalizeOrderBook)
});

export class WormService {
    public listMarkets(options: ListWormMarketsOptions = {}): Promise<ListWormMarketsResult> & {abort?: () => void} {
        const query: any = {
            limit: options.limit || 20,
            cursor: options.cursor || ''
        };
        if (options.sortOption && options.sortOption !== 'new') {
            query.sort_option = options.sortOption;
        }
        if (options.categorySlug && options.categorySlug !== 'all') {
            query.category_slug = options.categorySlug;
        }
        const req = requests.get('/worm/markets').query(query);
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

    public getMarket(conditionId: string): Promise<WormMarketDetail> & {abort?: () => void} {
        const req = requests.get(`/worm/markets/${encodeURIComponent(conditionId)}`);
        const promise = req.then(res => normalizeMarketDetail((res.body || {}).market || {})) as any;
        promise.abort = () => req.abort();
        return promise;
    }
}
