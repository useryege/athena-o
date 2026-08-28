import {AccountDataModule} from '../access-modules';
import requests from './requests';

const readScope = {module: AccountDataModule.WormTrading, mode: 'read' as const};
const writeScope = {module: AccountDataModule.WormTrading, mode: 'write' as const};

export type AbortableWormTradingPromise<T> = Promise<T> & {abort?: () => void};
export type WormTradingWalletBalanceStatus = 'COMPLETE' | 'PARTIAL' | 'UNAVAILABLE';
export type WormTradingActivityStatus = 'COMPLETE' | 'PARTIAL' | 'UNAVAILABLE';
export type WormWalletConnectionState = 'NOT_CONNECTED' | 'CONNECTING' | 'CONNECTED' | 'RECONNECT_REQUIRED' | 'DISCONNECTING' | 'REVOCATION_REQUIRED';

export const WORM_TRADING_LOGIN_SESSION_REQUIRED = 'WORM_TRADING_LOGIN_SESSION_REQUIRED';
export const WORM_TRADING_REAUTH_REQUIRED = 'WORM_TRADING_REAUTH_REQUIRED';
export const WORM_TRADING_REAUTH_UNAVAILABLE = 'WORM_TRADING_REAUTH_UNAVAILABLE';

export interface WormTradingStatus {
    started: boolean;
    status: string;
    network: string;
    commitment: string;
    rpcReachable: boolean;
    batchSupported: boolean;
    genesisVerified: boolean;
    genesisHash: string;
    usdcMint: string;
    usdcVerified: boolean;
    latestConfirmedSlot: string;
    lastProbeAt: number;
    lastSuccessAt: number;
    latencyMS: number;
    consecutiveFailures: number;
    lastErrorCategory: string;
    credentialStoreReady: boolean;
    wormAPIStatus: string;
    wormAPILastSuccessAt: number;
    wormAPILastErrorCategory: string;
}

export interface WormTradingWalletSummary {
    walletId: number;
    address: string;
    remark: string;
    avatarKind: string;
    avatarPresetId: string;
    avatarUrl: string;
}

export interface WormTradingAssetBalance {
    atomicAmount: string;
    amount: string;
    decimals: number;
    observedSlot: string;
    availability: string;
    errorCode: string;
}

export interface WormTradingTokenAssetBalance extends WormTradingAssetBalance {
    mint: string;
    tokenAccountCount: number;
}

export interface WormTradingWalletBalanceItem {
    wallet: WormTradingWalletSummary;
    sol: WormTradingAssetBalance;
    usdc: WormTradingTokenAssetBalance;
    status: WormTradingWalletBalanceStatus;
}

export interface ListWormTradingWalletBalancesResult {
    items: WormTradingWalletBalanceItem[];
    total: number;
    page: number;
    pageSize: number;
    network: string;
    commitment: string;
    fetchedAt: number;
}

export interface WormWalletConnection {
    state: WormWalletConnectionState;
    warningCode: string;
    connectedAt: number;
}

export interface WormTradingWalletConnectionItem {
    wallet: WormTradingWalletSummary;
    connection: WormWalletConnection;
}

export interface ListWormTradingWalletConnectionsResult {
    items: WormTradingWalletConnectionItem[];
    total: number;
    page: number;
    pageSize: number;
    fetchedAt: number;
}

export interface WormMarketReference {
    conditionId: string;
    title: string;
    logo: string;
    lastTradePrice: string;
    eventConditionId: string;
    eventTitle: string;
    eventLogo: string;
}

export interface WormOpenPosition {
    pubkey: string;
    positionRequestPubkey: string;
    market: WormMarketReference;
    side: 'YES' | 'NO' | string;
    leverage: string;
    totalShares: string;
    averageEntryPrice: string;
    userLiquidity: string;
    totalLiquidity: string;
    liquidationPrice: string;
    unrealizedPnL: string;
    realizedPnL: string;
    isClosed: boolean;
    isLiquidated: boolean;
    isClaimed: boolean;
    createdAt: number;
}

export interface WormInFlightRequest {
    pubkey: string;
    type: string;
    state: string;
    orderState: string;
    market: WormMarketReference;
    side: 'YES' | 'NO' | string;
    leverage: string;
    funds: string;
    price: string;
    shares: string;
    createdAt: number;
}

export interface WormActivityStreamState {
    availability: string;
    errorCode: string;
    truncated: boolean;
}

export interface WormTradingWalletActivityItem {
    wallet: WormTradingWalletSummary;
    connection: WormWalletConnection;
    openPositions: WormOpenPosition[];
    inFlightRequests: WormInFlightRequest[];
    positions: WormActivityStreamState;
    requests: WormActivityStreamState;
    status: WormTradingActivityStatus;
    observedAt: number;
}

export interface ListWormTradingWalletActivityResult {
    items: WormTradingWalletActivityItem[];
    total: number;
    page: number;
    pageSize: number;
    fetchedAt: number;
    openPositionCount: number;
    inFlightRequestCount: number;
    status: WormTradingActivityStatus;
}

export interface WormTradingReauthenticationChallenge {
    message: string;
    expiresAt: number;
}

export interface WormTradingReauthenticationLease {
    expiresAt: number;
}

export type WormMarketOutcomeSide = 'YES' | 'NO';

export interface WormTradingEventOutcome {
    side: WormMarketOutcomeSide;
    label: string;
    maxLeverage: string;
    lastTradePrice: string;
    selectable: boolean;
    unavailableCode: string;
}

export interface WormTradingEventMarket {
    marketConditionId: string;
    title: string;
    logo: string;
    state: string;
    marginEnabled: boolean;
    backend: string;
    unavailableCode: string;
    outcomes: WormTradingEventOutcome[];
}

export interface WormTradingEvent {
    eventConditionId: string;
    title: string;
    logo: string;
    markets: WormTradingEventMarket[];
    fetchedAt: number;
}

export interface WormMarketCombinationItem {
    ordinal: number;
    eventConditionId: string;
    eventTitle: string;
    eventLogo: string;
    marketConditionId: string;
    marketTitle: string;
    marketLogo: string;
    side: WormMarketOutcomeSide;
    outcomeLabel: string;
}

export interface WormMarketCombination {
    id: string;
    name: string;
    revision: number;
    items: WormMarketCombinationItem[];
    createdAt: number;
    updatedAt: number;
}

export interface ListWormMarketCombinationsResult {
    items: WormMarketCombination[];
    total: number;
    page: number;
    pageSize: number;
}

export interface SaveWormMarketCombinationItem {
    eventConditionId: string;
    marketConditionId: string;
    side: WormMarketOutcomeSide;
}

export interface CreateWormMarketCombinationInput {
    name: string;
    items: SaveWormMarketCombinationItem[];
}

export interface UpdateWormMarketCombinationInput extends CreateWormMarketCombinationInput {
    expectedRevision: number;
}

const readValue = (item: any, ...names: string[]) => {
    for (const name of names) {
        if (item?.[name] !== undefined && item?.[name] !== null) {
            return item[name];
        }
    }
    return undefined;
};

const readOptionalStringValue = (value: unknown): string => {
    if (value === undefined || value === null) {
        return '';
    }
    if (typeof value === 'object') {
        return String(readValue(value, 'value') ?? '');
    }
    return String(value);
};

const readString = (item: any, ...names: string[]) => readOptionalStringValue(readValue(item, ...names));
const readNumber = (item: any, ...names: string[]) => Number(readValue(item, ...names) ?? 0) || 0;
const readBoolean = (item: any, ...names: string[]) => {
    const value = readValue(item, ...names);
    return value === true || value === 1 || String(value).toLowerCase() === 'true';
};

const isRecord = (value: unknown): value is Record<string, any> => Boolean(value) && typeof value === 'object' && !Array.isArray(value);

const invalidWormTradingResponse = (): never => {
    throw new Error('Worm Trading returned an invalid response. Existing data was not replaced.');
};

const requireRecord = (value: unknown): Record<string, any> => {
    if (!isRecord(value)) {
        return invalidWormTradingResponse();
    }
    return value;
};

const readRepeatedArray = (item: unknown, ...names: string[]): unknown[] => {
    const value = readValue(requireRecord(item), ...names);
    if (value === undefined || value === null) {
        return [];
    }
    if (!Array.isArray(value)) {
        return invalidWormTradingResponse();
    }
    return value;
};

const requireString = (item: unknown, ...names: string[]): string => {
    const value = readValue(requireRecord(item), ...names);
    if (typeof value !== 'string' || value.trim() === '') {
        return invalidWormTradingResponse();
    }
    return value;
};

const requireInteger = (item: unknown, minimum: number, ...names: string[]): number => {
    const value = readValue(requireRecord(item), ...names);
    const parsed = typeof value === 'number' || typeof value === 'string' ? Number(value) : Number.NaN;
    if (!Number.isSafeInteger(parsed) || parsed < minimum) {
        return invalidWormTradingResponse();
    }
    return parsed;
};

const optionalInteger = (item: unknown, fallback: number, minimum: number, ...names: string[]): number => {
    const record = requireRecord(item);
    const value = readValue(record, ...names);
    if (value === undefined || value === null) {
        return fallback;
    }
    return requireInteger(record, minimum, ...names);
};

const requireUnsignedIntegerString = (item: unknown, ...names: string[]): string => {
    const value = readString(requireRecord(item), ...names);
    if (!/^\d+$/.test(value)) {
        return invalidWormTradingResponse();
    }
    return value;
};

const normalizeBalanceStatus = (value: unknown): WormTradingWalletBalanceStatus => {
    switch (String(value || '').toUpperCase()) {
        case 'COMPLETE':
            return 'COMPLETE';
        case 'PARTIAL':
            return 'PARTIAL';
        case 'UNAVAILABLE':
            return 'UNAVAILABLE';
        default:
            return invalidWormTradingResponse();
    }
};

const normalizeConnectionState = (value: unknown): WormWalletConnectionState => {
    switch (String(value || '').toUpperCase()) {
        case 'CONNECTING':
            return 'CONNECTING';
        case 'CONNECTED':
            return 'CONNECTED';
        case 'RECONNECT_REQUIRED':
            return 'RECONNECT_REQUIRED';
        case 'DISCONNECTING':
            return 'DISCONNECTING';
        case 'REVOCATION_REQUIRED':
            return 'REVOCATION_REQUIRED';
        case 'NOT_CONNECTED':
            return 'NOT_CONNECTED';
        default:
            return invalidWormTradingResponse();
    }
};

const normalizeAvailability = (value: unknown): string => {
    const availability = String(value || '').toUpperCase();
    if (availability !== 'AVAILABLE' && availability !== 'UNAVAILABLE') {
        return invalidWormTradingResponse();
    }
    return availability;
};

const normalizeWalletSummary = (wallet: unknown): WormTradingWalletSummary => {
    const item = requireRecord(wallet);
    return {
        walletId: requireInteger(item, 1, 'walletId', 'wallet_id'),
        address: requireString(item, 'address'),
        remark: readString(item, 'remark'),
        avatarKind: readString(item, 'avatarKind', 'avatar_kind'),
        avatarPresetId: readString(item, 'avatarPresetId', 'avatar_preset_id'),
        avatarUrl: readString(item, 'avatarUrl', 'avatar_url')
    };
};

const normalizeAssetBalance = (value: unknown): WormTradingAssetBalance => {
    const item = requireRecord(value);
    const availability = normalizeAvailability(readValue(item, 'availability'));
    const atomicAmount = readString(item, 'atomicAmount', 'atomic_amount');
    const amount = readString(item, 'amount');
    if (availability === 'AVAILABLE' && (!/^\d+$/.test(atomicAmount) || amount === '')) {
        return invalidWormTradingResponse();
    }
    return {
        atomicAmount,
        amount,
        decimals: requireInteger(item, 0, 'decimals'),
        observedSlot: requireUnsignedIntegerString(item, 'observedSlot', 'observed_slot'),
        availability,
        errorCode: readString(item, 'errorCode', 'error_code')
    };
};

const normalizeWalletBalance = (value: unknown): WormTradingWalletBalanceItem => {
    const item = requireRecord(value);
    const usdc = requireRecord(item.usdc);
    return {
        wallet: normalizeWalletSummary(item.wallet),
        sol: normalizeAssetBalance(item.sol),
        usdc: {
            ...normalizeAssetBalance(usdc),
            mint: requireString(usdc, 'mint'),
            tokenAccountCount: requireInteger(usdc, 0, 'tokenAccountCount', 'token_account_count')
        },
        status: normalizeBalanceStatus(readValue(item, 'status'))
    };
};

const normalizeConnection = (value: unknown): WormWalletConnection => {
    const item = requireRecord(value);
    return {
        state: normalizeConnectionState(readValue(item, 'state')),
        warningCode: readString(item, 'warningCode', 'warning_code'),
        connectedAt: optionalInteger(item, 0, 0, 'connectedAt', 'connected_at')
    };
};

const normalizeWalletConnection = (value: unknown): WormTradingWalletConnectionItem => {
    const item = requireRecord(value);
    return {
        wallet: normalizeWalletSummary(readValue(item, 'wallet')),
        connection: normalizeConnection(readValue(item, 'connection'))
    };
};

const normalizeMarket = (value: unknown): WormMarketReference => {
    const item = requireRecord(value);
    const event = isRecord(item.event) ? item.event : {};
    return {
        conditionId: requireString(item, 'conditionId', 'condition_id'),
        title: requireString(item, 'title'),
        logo: readString(item, 'logo'),
        lastTradePrice: readString(item, 'lastTradePrice', 'last_trade_price', 'latestTradePrice', 'latest_trade_price'),
        eventConditionId: readString(item, 'eventConditionId', 'event_condition_id') || readString(event, 'conditionId', 'condition_id'),
        eventTitle: readString(item, 'eventTitle', 'event_title') || readString(event, 'title'),
        eventLogo: readString(item, 'eventLogo', 'event_logo') || readString(event, 'logo')
    };
};

const normalizeSide = (item: any): 'YES' | 'NO' | string => {
    const explicit = readString(item, 'side').toUpperCase();
    if (explicit) {
        return explicit;
    }
    const isYes = readValue(item, 'isYes', 'is_yes');
    return isYes === undefined || isYes === null ? '' : readBoolean(item, 'isYes', 'is_yes') ? 'YES' : 'NO';
};

const normalizeOpenPosition = (value: unknown): WormOpenPosition => {
    const item = requireRecord(value);
    const side = normalizeSide(item);
    if (side !== 'YES' && side !== 'NO') {
        return invalidWormTradingResponse();
    }
    return {
        pubkey: requireString(item, 'pubkey'),
        positionRequestPubkey: readString(item, 'positionRequestPubkey', 'position_request_pubkey'),
        market: normalizeMarket(item.market),
        side,
        leverage: requireString(item, 'leverage'),
        totalShares: requireString(item, 'totalShares', 'total_shares'),
        averageEntryPrice: requireString(item, 'averageEntryPrice', 'average_entry_price', 'avgEntryPrice', 'avg_entry_price'),
        userLiquidity: requireString(item, 'userLiquidity', 'user_liquidity'),
        totalLiquidity: requireString(item, 'totalLiquidity', 'total_liquidity'),
        liquidationPrice: requireString(item, 'liquidationPrice', 'liquidation_price'),
        unrealizedPnL: readString(item, 'unrealizedPnl', 'unrealized_pnl'),
        realizedPnL: requireString(item, 'realizedPnl', 'realized_pnl'),
        isClosed: readBoolean(item, 'isClosed', 'is_closed'),
        isLiquidated: readBoolean(item, 'isLiquidated', 'is_liquidated'),
        isClaimed: readBoolean(item, 'isClaimed', 'is_claimed'),
        createdAt: optionalInteger(item, 0, 0, 'createdAt', 'created_at')
    };
};

const normalizeInFlightRequest = (value: unknown): WormInFlightRequest => {
    const item = requireRecord(value);
    const type = requireString(item, 'type').toLowerCase();
    const side = normalizeSide(item);
    if ((type !== 'market' && type !== 'limit') || (side !== 'YES' && side !== 'NO')) {
        return invalidWormTradingResponse();
    }
    return {
        pubkey: requireString(item, 'pubkey'),
        type,
        state: requireString(item, 'state'),
        orderState: readString(item, 'orderState', 'order_state'),
        market: normalizeMarket(item.market),
        side,
        leverage: requireString(item, 'leverage'),
        funds: requireString(item, 'funds'),
        price: readString(item, 'price'),
        shares: readString(item, 'shares'),
        createdAt: optionalInteger(item, 0, 0, 'createdAt', 'created_at')
    };
};

const normalizeStreamState = (value: unknown): WormActivityStreamState => {
    const item = requireRecord(value);
    return {
        availability: normalizeAvailability(readValue(item, 'availability')),
        errorCode: readString(item, 'errorCode', 'error_code'),
        truncated: readBoolean(item, 'truncated')
    };
};

const normalizeWalletActivity = (value: unknown): WormTradingWalletActivityItem => {
    const item = requireRecord(value);
    return {
        wallet: normalizeWalletSummary(item.wallet),
        connection: normalizeConnection(item.connection),
        openPositions: readRepeatedArray(item, 'openPositions', 'open_positions').map(normalizeOpenPosition),
        inFlightRequests: readRepeatedArray(item, 'inFlightRequests', 'in_flight_requests').map(normalizeInFlightRequest),
        positions: normalizeStreamState(readValue(item, 'positions', 'openPositionsStatus', 'open_positions_status')),
        requests: normalizeStreamState(readValue(item, 'requests', 'inFlightRequestsStatus', 'in_flight_requests_status')),
        status: normalizeBalanceStatus(readValue(item, 'status')),
        observedAt: optionalInteger(item, 0, 0, 'observedAt', 'observed_at')
    };
};

const abortableRequest = <T>(request: any, map: (body: any) => T): AbortableWormTradingPromise<T> => {
    const promise = request.then((response: any) => map(response.body)) as AbortableWormTradingPromise<T>;
    promise.abort = () => request.abort();
    return promise;
};

const parseJSONResponse = async (response: Response): Promise<unknown> => {
    const text = await response.text();
    if (!text) {
        return undefined;
    }
    try {
        const value: unknown = JSON.parse(text);
        return value;
    } catch {
        return undefined;
    }
};

const rawSameOriginRequest = <T>(
    method: 'GET' | 'POST' | 'DELETE',
    path: string,
    body: Record<string, unknown> | undefined,
    fallbackError: string,
    map: (value: Record<string, unknown>) => T
): AbortableWormTradingPromise<T> => {
    const controller = new AbortController();
    const promise = fetch(requests.toAbsURL(path), {
        method,
        credentials: 'same-origin',
        mode: 'same-origin',
        redirect: 'error',
        headers: {'Accept': 'application/json', 'Content-Type': 'application/json'},
        body: body ? JSON.stringify(body) : undefined,
        signal: controller.signal
    }).then(async response => {
        const parsedBody = await parseJSONResponse(response);
        const responseBody = isRecord(parsedBody) ? parsedBody : {};
        if (!response.ok) {
            const nestedError = isRecord(responseBody.error) ? responseBody.error : undefined;
            const error = new Error(String(nestedError?.message || responseBody.message || response.statusText || fallbackError)) as Error & {
                status: number;
                body: Record<string, unknown>;
                response: {status: number; body: Record<string, unknown>; headers: Record<string, string>};
            };
            const reason = String(responseBody.reason || nestedError?.reason || response.headers.get('X-Athena-Error-Reason') || '');
            const headers = reason ? {'x-athena-error-reason': reason} : {};
            error.status = response.status;
            error.body = responseBody;
            error.response = {status: response.status, body: responseBody, headers};
            throw error;
        }
        if (!isRecord(parsedBody)) {
            return invalidWormTradingResponse();
        }
        return map(parsedBody);
    }) as AbortableWormTradingPromise<T>;
    promise.abort = () => controller.abort();
    return promise;
};

const rawReauthenticationPost = <T>(path: string, body: Record<string, unknown>, map: (value: Record<string, unknown>) => T): AbortableWormTradingPromise<T> =>
    rawSameOriginRequest('POST', path, body, 'Worm credential reauthentication failed', map);

const normalizeManagedConnection = (value: unknown, expectedWalletID: number): WormWalletConnection => {
    const item = requireRecord(value);
    if (requireInteger(item, 1, 'walletId', 'wallet_id') !== expectedWalletID) {
        return invalidWormTradingResponse();
    }
    requireString(item, 'address');
    return normalizeConnection(item);
};

const requireExactString = (item: unknown, name: string): string => {
    const value = requireRecord(item)[name];
    if (typeof value !== 'string' || value.trim() === '') {
        return invalidWormTradingResponse();
    }
    return value;
};

const optionalExactString = (item: unknown, name: string): string => {
    const value = requireRecord(item)[name];
    if (value === undefined || value === null) {
        return '';
    }
    if (typeof value !== 'string') {
        return invalidWormTradingResponse();
    }
    return value;
};

const requireExactBoolean = (item: unknown, name: string): boolean => {
    const value = requireRecord(item)[name];
    if (typeof value !== 'boolean') {
        return invalidWormTradingResponse();
    }
    return value;
};

const requireExactArray = (item: unknown, name: string): unknown[] => {
    const value = requireRecord(item)[name];
    if (!Array.isArray(value)) {
        return invalidWormTradingResponse();
    }
    return value;
};

const normalizeOutcomeSide = (value: unknown): WormMarketOutcomeSide => {
    if (value === 'YES' || value === 'NO') {
        return value;
    }
    return invalidWormTradingResponse();
};

const exactUnitDecimalMaximumLength = 128;

const parseExactUnitDecimal = (value: string): {integer: bigint; scale: number} | undefined => {
    if (value.length > exactUnitDecimalMaximumLength || !/^\d+(?:\.\d+)?$/.test(value)) {
        return undefined;
    }
    const [whole, fraction = ''] = value.split('.');
    const scale = fraction.length;
    const denominator = 10n ** BigInt(scale);
    const integer = BigInt(`${whole}${fraction}`);
    if (integer > denominator) {
        return undefined;
    }
    return {integer, scale};
};

const exactUnitDecimalsAreComplements = (yes: string, no: string): boolean => {
    const yesPrice = parseExactUnitDecimal(yes);
    const noPrice = parseExactUnitDecimal(no);
    if (!yesPrice || !noPrice) {
        return false;
    }
    const scale = Math.max(yesPrice.scale, noPrice.scale);
    const yesInteger = yesPrice.integer * 10n ** BigInt(scale - yesPrice.scale);
    const noInteger = noPrice.integer * 10n ** BigInt(scale - noPrice.scale);
    return yesInteger + noInteger === 10n ** BigInt(scale);
};

const normalizeEventOutcome = (value: unknown): WormTradingEventOutcome => {
    const item = requireRecord(value);
    const side = normalizeOutcomeSide(item.side);
    const selectable = requireExactBoolean(item, 'selectable');
    const unavailableCode = optionalExactString(item, 'unavailableCode');
    const maxLeverage = optionalExactString(item, 'maxLeverage');
    const lastTradePrice = optionalExactString(item, 'lastTradePrice');
    if ((selectable && unavailableCode !== '') || (!selectable && unavailableCode === '')) {
        return invalidWormTradingResponse();
    }
    if (selectable) {
        const parsedLeverage = Number(maxLeverage);
        if (!Number.isFinite(parsedLeverage) || parsedLeverage < 1) {
            return invalidWormTradingResponse();
        }
    }
    if (lastTradePrice !== '' && !parseExactUnitDecimal(lastTradePrice)) {
        return invalidWormTradingResponse();
    }
    return {
        side,
        label: requireExactString(item, 'label'),
        maxLeverage,
        lastTradePrice,
        selectable,
        unavailableCode
    };
};

const normalizeEventMarket = (value: unknown): WormTradingEventMarket => {
    const item = requireRecord(value);
    const outcomes = requireExactArray(item, 'outcomes').map(normalizeEventOutcome);
    const sides = new Set(outcomes.map(outcome => outcome.side));
    if (outcomes.length !== 2 || sides.size !== 2 || !sides.has('YES') || !sides.has('NO')) {
        return invalidWormTradingResponse();
    }
    const yesPrice = outcomes.find(outcome => outcome.side === 'YES')?.lastTradePrice || '';
    const noPrice = outcomes.find(outcome => outcome.side === 'NO')?.lastTradePrice || '';
    if ((yesPrice === '') !== (noPrice === '') || (yesPrice !== '' && !exactUnitDecimalsAreComplements(yesPrice, noPrice))) {
        return invalidWormTradingResponse();
    }
    return {
        marketConditionId: requireExactString(item, 'marketConditionId'),
        title: requireExactString(item, 'title'),
        logo: optionalExactString(item, 'logo'),
        state: requireExactString(item, 'state'),
        marginEnabled: requireExactBoolean(item, 'marginEnabled'),
        backend: optionalExactString(item, 'backend'),
        unavailableCode: optionalExactString(item, 'unavailableCode'),
        outcomes
    };
};

const normalizeTradingEvent = (value: unknown, expectedEventConditionID?: string): WormTradingEvent => {
    const item = requireRecord(value);
    const eventConditionId = requireExactString(item, 'eventConditionId');
    const markets = requireExactArray(item, 'markets').map(normalizeEventMarket);
    if (expectedEventConditionID && eventConditionId !== expectedEventConditionID) {
        return invalidWormTradingResponse();
    }
    if (new Set(markets.map(market => market.marketConditionId)).size !== markets.length) {
        return invalidWormTradingResponse();
    }
    return {
        eventConditionId,
        title: requireExactString(item, 'title'),
        logo: optionalExactString(item, 'logo'),
        markets,
        fetchedAt: requireInteger(item, 1, 'fetchedAt')
    };
};

const normalizeCombinationItem = (value: unknown): WormMarketCombinationItem => {
    const item = requireRecord(value);
    return {
        ordinal: requireInteger(item, 1, 'ordinal'),
        eventConditionId: requireExactString(item, 'eventConditionId'),
        eventTitle: requireExactString(item, 'eventTitle'),
        eventLogo: optionalExactString(item, 'eventLogo'),
        marketConditionId: requireExactString(item, 'marketConditionId'),
        marketTitle: requireExactString(item, 'marketTitle'),
        marketLogo: optionalExactString(item, 'marketLogo'),
        side: normalizeOutcomeSide(item.side),
        outcomeLabel: requireExactString(item, 'outcomeLabel')
    };
};

const normalizeMarketCombination = (value: unknown): WormMarketCombination => {
    const item = requireRecord(value);
    const items = requireExactArray(item, 'items').map(normalizeCombinationItem);
    const marketIDs = new Set(items.map(selection => selection.marketConditionId));
    if (items.length === 0 || marketIDs.size !== items.length || items.some((selection, index) => selection.ordinal !== index + 1)) {
        return invalidWormTradingResponse();
    }
    return {
        id: requireExactString(item, 'id'),
        name: requireExactString(item, 'name'),
        revision: requireInteger(item, 1, 'revision'),
        items,
        createdAt: requireInteger(item, 0, 'createdAt'),
        updatedAt: requireInteger(item, 0, 'updatedAt')
    };
};

export class WormTradingService {
    public getStatus(): AbortableWormTradingPromise<WormTradingStatus> {
        const request = requests.get('/worm-trading/status', readScope);
        return abortableRequest(request, body => ({
            started: readBoolean(body, 'started'),
            status: readString(body, 'status'),
            network: readString(body, 'network'),
            commitment: readString(body, 'commitment'),
            rpcReachable: readBoolean(body, 'rpcReachable', 'rpc_reachable'),
            batchSupported: readBoolean(body, 'batchSupported', 'batch_supported'),
            genesisVerified: readBoolean(body, 'genesisVerified', 'genesis_verified'),
            genesisHash: readString(body, 'genesisHash', 'genesis_hash'),
            usdcMint: readString(body, 'usdcMint', 'usdc_mint'),
            usdcVerified: readBoolean(body, 'usdcVerified', 'usdc_verified'),
            latestConfirmedSlot: readString(body, 'latestConfirmedSlot', 'latest_confirmed_slot'),
            lastProbeAt: readNumber(body, 'lastProbeAt', 'last_probe_at'),
            lastSuccessAt: readNumber(body, 'lastSuccessAt', 'last_success_at'),
            latencyMS: readNumber(body, 'latencyMs', 'latency_ms'),
            consecutiveFailures: readNumber(body, 'consecutiveFailures', 'consecutive_failures'),
            lastErrorCategory: readString(body, 'lastErrorCategory', 'last_error_category'),
            credentialStoreReady: readBoolean(body, 'credentialStoreReady', 'credential_store_ready'),
            wormAPIStatus: readString(body, 'wormApiStatus', 'worm_api_status'),
            wormAPILastSuccessAt: readNumber(body, 'wormApiLastSuccessAt', 'worm_api_last_success_at'),
            wormAPILastErrorCategory: readString(body, 'wormApiLastErrorCategory', 'worm_api_last_error_category')
        }));
    }

    public listWalletBalances(page = 1, pageSize = 20): AbortableWormTradingPromise<ListWormTradingWalletBalancesResult> {
        const request = requests.get('/worm-trading/wallet-balances', readScope).query({page, pageSize});
        return abortableRequest(request, value => {
            const body = requireRecord(value);
            return {
                items: readRepeatedArray(body, 'items').map(normalizeWalletBalance),
                total: optionalInteger(body, 0, 0, 'total'),
                page: requireInteger(body, 1, 'page'),
                pageSize: requireInteger(body, 1, 'pageSize', 'page_size'),
                network: requireString(body, 'network'),
                commitment: requireString(body, 'commitment'),
                fetchedAt: requireInteger(body, 0, 'fetchedAt', 'fetched_at')
            };
        });
    }

    public listWalletActivity(page = 1, pageSize = 20): AbortableWormTradingPromise<ListWormTradingWalletActivityResult> {
        const request = requests.get('/worm-trading/wallet-activity', readScope).query({page, pageSize});
        return abortableRequest(request, value => {
            const body = requireRecord(value);
            return {
                items: readRepeatedArray(body, 'items').map(normalizeWalletActivity),
                total: optionalInteger(body, 0, 0, 'total'),
                page: requireInteger(body, 1, 'page'),
                pageSize: requireInteger(body, 1, 'pageSize', 'page_size'),
                fetchedAt: requireInteger(body, 0, 'fetchedAt', 'fetched_at'),
                openPositionCount: requireInteger(body, 0, 'openPositionCount', 'open_position_count'),
                inFlightRequestCount: requireInteger(body, 0, 'inFlightRequestCount', 'in_flight_request_count'),
                status: normalizeBalanceStatus(readValue(body, 'status'))
            };
        });
    }

    public listWalletConnections(page = 1, pageSize = 100): AbortableWormTradingPromise<ListWormTradingWalletConnectionsResult> {
        const query = new URLSearchParams({page: String(page), pageSize: String(pageSize)});
        return rawSameOriginRequest('GET', `/api/v1/worm-trading/wallet-connections?${query.toString()}`, undefined, 'Worm wallet connection inventory failed', value => {
            const body = requireRecord(value);
            const items = readRepeatedArray(body, 'items').map(normalizeWalletConnection);
            const responsePage = requireInteger(body, 1, 'page');
            const responsePageSize = requireInteger(body, 1, 'pageSize', 'page_size');
            const total = requireInteger(body, 0, 'total');
            const walletIDs = new Set(items.map(item => item.wallet.walletId));
            const walletAddresses = new Set(items.map(item => item.wallet.address));
            const firstItemOffset = (responsePage - 1) * responsePageSize;
            if (
                responsePage !== page ||
                responsePageSize !== pageSize ||
                items.length > responsePageSize ||
                walletIDs.size !== items.length ||
                walletAddresses.size !== items.length ||
                (items.length > 0 && firstItemOffset + items.length > total)
            ) {
                return invalidWormTradingResponse();
            }
            return {
                items,
                total,
                page: responsePage,
                pageSize: responsePageSize,
                fetchedAt: requireInteger(body, 0, 'fetchedAt', 'fetched_at')
            };
        });
    }

    public connectWallet(walletId: number): AbortableWormTradingPromise<WormWalletConnection> {
        return rawSameOriginRequest('POST', `/api/v1/worm-trading/wallet-connections/${encodeURIComponent(String(walletId))}`, {}, 'Worm wallet connection failed', body =>
            normalizeManagedConnection(body.connection || body, walletId)
        );
    }

    public reconnectWallet(walletId: number): AbortableWormTradingPromise<WormWalletConnection> {
        return rawSameOriginRequest(
            'POST',
            `/api/v1/worm-trading/wallet-connections/${encodeURIComponent(String(walletId))}:reconnect`,
            {},
            'Worm wallet reconnection failed',
            body => normalizeManagedConnection(body.connection || body, walletId)
        );
    }

    public disconnectWallet(walletId: number): AbortableWormTradingPromise<WormWalletConnection> {
        return rawSameOriginRequest(
            'DELETE',
            `/api/v1/worm-trading/wallet-connections/${encodeURIComponent(String(walletId))}`,
            undefined,
            'Worm wallet disconnection failed',
            body => normalizeManagedConnection(body.connection || body, walletId)
        );
    }

    public googleCredentialReauthenticationURL(returnTo: string): string {
        const query = new URLSearchParams({returnTo});
        return `${requests.toAbsURL('/auth/worm-trading/google')}?${query.toString()}`;
    }

    public createSolanaCredentialChallenge(): AbortableWormTradingPromise<WormTradingReauthenticationChallenge> {
        return rawReauthenticationPost('/auth/worm-trading/solana/challenge', {}, body => ({
            message: requireString(body, 'message'),
            expiresAt: requireInteger(body, 1, 'expiresAt', 'expires_at')
        }));
    }

    public verifySolanaCredentialSignature(signature: string): AbortableWormTradingPromise<WormTradingReauthenticationLease> {
        return rawReauthenticationPost('/auth/worm-trading/solana/verify', {signature}, body => ({expiresAt: requireInteger(body, 1, 'expiresAt', 'expires_at')}));
    }

    public createDevelopmentCredentialLease(): AbortableWormTradingPromise<WormTradingReauthenticationLease> {
        return rawReauthenticationPost('/auth/worm-trading/development', {}, body => ({expiresAt: requireInteger(body, 1, 'expiresAt', 'expires_at')}));
    }

    public getEvent(eventConditionId: string): AbortableWormTradingPromise<WormTradingEvent> {
        const expectedEventConditionID = eventConditionId.trim();
        const request = requests.get(`/worm-trading/events/${encodeURIComponent(expectedEventConditionID)}`, readScope);
        return abortableRequest(request, body => normalizeTradingEvent(requireRecord(body).event || body, expectedEventConditionID));
    }

    public listMarketCombinations(page = 1, pageSize = 20): AbortableWormTradingPromise<ListWormMarketCombinationsResult> {
        const request = requests.get('/worm-trading/combinations', readScope).query({page, pageSize});
        return abortableRequest(request, value => {
            const body = requireRecord(value);
            const items = requireExactArray(body, 'items').map(normalizeMarketCombination);
            const total = requireInteger(body, 0, 'total');
            const responsePage = requireInteger(body, 1, 'page');
            const responsePageSize = requireInteger(body, 1, 'pageSize');
            const offset = (responsePage - 1) * responsePageSize;
            if (
                responsePage !== page ||
                responsePageSize !== pageSize ||
                responsePageSize > 100 ||
                items.length > responsePageSize ||
                new Set(items.map(item => item.id)).size !== items.length ||
                (items.length > 0 && offset + items.length > total)
            ) {
                return invalidWormTradingResponse();
            }
            return {items, total, page: responsePage, pageSize: responsePageSize};
        });
    }

    public getMarketCombination(id: string): AbortableWormTradingPromise<WormMarketCombination> {
        const request = requests.get(`/worm-trading/combinations/${encodeURIComponent(id)}`, readScope);
        return abortableRequest(request, body => normalizeMarketCombination(requireRecord(body).combination || body));
    }

    public createMarketCombination(input: CreateWormMarketCombinationInput): AbortableWormTradingPromise<WormMarketCombination> {
        const request = requests.post('/worm-trading/combinations', writeScope).send(input);
        return abortableRequest(request, body => normalizeMarketCombination(requireRecord(body).combination || body));
    }

    public updateMarketCombination(id: string, input: UpdateWormMarketCombinationInput): AbortableWormTradingPromise<WormMarketCombination> {
        const request = requests.put(`/worm-trading/combinations/${encodeURIComponent(id)}`, writeScope).send(input);
        return abortableRequest(request, body => normalizeMarketCombination(requireRecord(body).combination || body));
    }

    public deleteMarketCombination(id: string, expectedRevision: number): AbortableWormTradingPromise<void> {
        const request = requests.delete(`/worm-trading/combinations/${encodeURIComponent(id)}`, writeScope).query({expectedRevision});
        return abortableRequest(request, () => undefined);
    }
}
