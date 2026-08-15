export interface WormMarketsMarketItem {
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
    estimate?: WormMarketsMarginPositionEstimateItem;
    tradingDataError?: string;
}

export interface WormMarketsMarginPositionEstimateItem {
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

export interface WormMarketsEventItem {
    conditionId: string;
    title: string;
    description?: string;
    logo?: string;
    category?: string;
    created?: number;
    live: boolean;
    marketCount: number;
    markets: WormMarketsMarketItem[];
}

export interface GetWormMarketsEventResult {
    item?: WormMarketsEventItem;
    fetchedAt?: number;
}
