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
    configKind?: string;
    maxLeverageYes?: string;
    maxLeverageNo?: string;
    openingFee?: string;
    closingFee?: string;
    annualFeeRate?: string;
    orderMinSize?: string;
    priceDecimals?: number;
    sharesDecimals?: number;
    estimate?: WormMarginPositionEstimateItem;
    tradingDataError?: string;
}

export interface WormMarginPositionEstimateItem {
    funds: string;
    isYes: boolean;
    leverage: string;
    averagePrice?: string;
    totalShares?: string;
    totalCost?: string;
    bestAsk?: string;
    worstFillPrice?: string;
    isFullyFilled?: boolean;
    feeAmount?: string;
    userFundsNeeded?: string;
    liquidationPrice?: string;
}

export interface WormEventItem {
    conditionId: string;
    title: string;
    description?: string;
    logo?: string;
    category?: string;
    created?: number;
    live: boolean;
    marketCount: number;
    markets: WormMarketItem[];
}

export interface GetWormEventResult {
    item?: WormEventItem;
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
const readNumber = (item: any, ...names: string[]) => Number(readValue(item, ...names) || 0) || undefined;
const readBoolean = (item: any, ...names: string[]) => Boolean(readValue(item, ...names));

const normalizeEstimate = (item: any): WormMarginPositionEstimateItem | undefined => {
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
    configKind: readString(item, 'configKind', 'config_kind'),
    maxLeverageYes: readString(item, 'maxLeverageYes', 'max_leverage_yes'),
    maxLeverageNo: readString(item, 'maxLeverageNo', 'max_leverage_no'),
    openingFee: readString(item, 'openingFee', 'opening_fee'),
    closingFee: readString(item, 'closingFee', 'closing_fee'),
    annualFeeRate: readString(item, 'annualFeeRate', 'annual_fee_rate'),
    orderMinSize: readString(item, 'orderMinSize', 'order_min_size'),
    priceDecimals: readNumber(item, 'priceDecimals', 'price_decimals'),
    sharesDecimals: readNumber(item, 'sharesDecimals', 'shares_decimals'),
    estimate: normalizeEstimate(readValue(item, 'estimate')),
    tradingDataError: readString(item, 'tradingDataError', 'trading_data_error')
});

const normalizeEvent = (item: any): WormEventItem => {
    const markets = readValue(item, 'markets');
    const normalizedMarkets = Array.isArray(markets) ? markets.map(normalizeMarket) : [];
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

export class WormService {
    public getEvent(conditionId: string): Promise<GetWormEventResult> & {abort?: () => void} {
        const req = requests.get(`/worm/events/${encodeURIComponent(conditionId)}`);
        const promise = req.then(res => {
            const body = res.body || {};
            const item = body.item || body.event;
            return {
                item: item ? normalizeEvent(item) : undefined,
                fetchedAt: readNumber(body, 'fetchedAt', 'fetched_at')
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }
}
