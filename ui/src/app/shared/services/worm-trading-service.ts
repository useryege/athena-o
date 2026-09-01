import {AccountDataModule} from '../access-modules';
import requests, {APPLICATION_REALM_QUERY} from './requests';

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

export type WormPositionCashOutState =
    | 'AWAITING_AUTHORIZATION'
    | 'QUEUED'
    | 'PREFLIGHTING'
    | 'CLOSING'
    | 'AWAITING_COMPLETION'
    | 'COMPLETED'
    | 'FAILED'
    | 'RECONCILIATION_REQUIRED'
    | 'EXPIRED';

export type WormPositionCashOutAllowedAction = 'CASH_OUT' | 'AUTHORIZE_CASH_OUT' | 'CHECK_STATUS' | 'NONE';

export interface WormPositionCashOutProjection {
    operationId: string;
    state: WormPositionCashOutState | '';
    reasonCode: string;
    allowedAction: WormPositionCashOutAllowedAction;
    batchId: string;
    batchState: WormPositionCashOutBatchState | '';
    batchItemState: WormPositionCashOutBatchItemState | '';
    batchLockReasonCode: string;
    revision: number;
    updatedAt: number;
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
    cashOut: WormPositionCashOutProjection;
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

export type WormExecutionPlanState = 'BUILDING' | 'READY' | 'FAILED' | 'EXPIRED';
export type WormExecutionPlanStepDisposition = 'READY' | 'SKIPPED';

export interface WormExecutionPlanAssetBalance {
    atomicAmount: string;
    amount: string;
    decimals: number;
    observedSlot: string;
    availability: string;
    errorCode: string;
}

export interface WormExecutionPlanWallet {
    ordinal: number;
    wallet: WormTradingWalletSummary;
    connection?: WormWalletConnection;
    sol: WormExecutionPlanAssetBalance;
    usdc: WormExecutionPlanAssetBalance;
    status: string;
    reasonCode: string;
}

export interface WormExecutionPlanEstimate {
    averagePrice: string;
    totalShares: string;
    totalCost: string;
    bestAsk: string;
    worstFillPrice: string;
    isFullyFilled: boolean;
    feeAmount: string;
    userFundsNeeded: string;
    liquidationPrice: string;
}

export interface WormExecutionPlanItem extends WormMarketCombinationItem {
    backend: string;
    funds: string;
    leverage: string;
    state: string;
    reasonCode: string;
    estimate?: WormExecutionPlanEstimate;
}

export interface WormExecutionPlan {
    id: string;
    combinationId: string;
    combinationName: string;
    combinationRevision: number;
    state: WormExecutionPlanState;
    buildStage: string;
    failureCode: string;
    usabilityCode: string;
    walletCount: number;
    itemCount: number;
    totalStepCount: number;
    completedStepCount: number;
    readyStepCount: number;
    skippedStepCount: number;
    reasonCounts: Record<string, number>;
    maximumCollateral: string;
    openingFeeEstimate: string;
    totalUSDCNeeded: string;
    requestedAt: number;
    completedAt: number;
    expiresAt: number;
    retentionUntil: number;
    createdAt: number;
    updatedAt: number;
    wallets: WormExecutionPlanWallet[];
    items: WormExecutionPlanItem[];
}

export interface CreateWormExecutionPlanInput {
    combinationId: string;
    expectedCombinationRevision: number;
    walletIds: number[];
}

export interface WormExecutionPlanStep {
    ordinal: number;
    walletOrdinal: number;
    itemOrdinal: number;
    disposition: WormExecutionPlanStepDisposition;
    reasonCode: string;
    projectedUsdcBefore: string;
    projectedUsdcAfter: string;
}

export interface ListWormExecutionPlanStepsResult {
    items: WormExecutionPlanStep[];
    total: number;
    page: number;
    pageSize: number;
}

export type WormExecutionRunState =
    | 'AWAITING_AUTHORIZATION'
    | 'AUTHORIZED'
    | 'RUNNING'
    | 'PAUSE_REQUESTED'
    | 'PAUSED'
    | 'TERMINATE_REQUESTED'
    | 'RECONCILIATION_REQUIRED'
    | 'COMPLETED'
    | 'TERMINATED'
    | 'FAILED';

export type WormExecutionStepState =
    | 'PENDING'
    | 'PREFLIGHTING'
    | 'OPENING'
    | 'OPENED'
    | 'SIGNING'
    | 'FINALIZING'
    | 'AWAITING_COMPLETION'
    | 'COMPLETED'
    | 'SATISFIED'
    | 'SKIPPED'
    | 'FAILED'
    | 'NOT_EXECUTED'
    | 'OUTCOME_UNKNOWN';

export type WormExecutionAllowedAction = 'AUTHORIZE' | 'START' | 'PAUSE' | 'CONTINUE' | 'TERMINATE' | 'HEARTBEAT' | 'EXECUTE_NEXT' | 'RECONCILE';

export interface WormExecutionRunCounts {
    total: number;
    actionable: number;
    terminal: number;
    completed: number;
    satisfied: number;
    skipped: number;
    failed: number;
    notExecuted: number;
}

export interface WormExecutionRunAuthorization {
    state: string;
    proofKind: string;
    authorizedAt: number;
    requiresReauthorization: boolean;
}

export interface WormExecutionRunCoordinator {
    state: string;
    generation: number;
    leaseExpiresAt: number;
}

export interface WormExecutionRunStep {
    id: string;
    ordinal: number;
    wallet: WormTradingWalletSummary;
    market: {
        eventConditionId: string;
        eventTitle: string;
        eventLogo: string;
        marketConditionId: string;
        marketTitle: string;
        marketLogo: string;
        outcomeLabel: string;
        backend: string;
    };
    side: WormMarketOutcomeSide;
    funds: string;
    leverage: string;
    state: WormExecutionStepState;
    reasonCode: string;
    positionRequestId: string;
    providerState: string;
    providerOrderState: string;
    completionSource: string;
    completionPositionPubkey: string;
    completionPositionRequestPubkey: string;
    completionPositionCreatedAt: number;
    startedAt: number;
    updatedAt: number;
    completedAt: number;
}

export interface WormExecutionRun {
    id: string;
    planId: string;
    combinationId: string;
    combinationName: string;
    combinationRevision: number;
    state: WormExecutionRunState;
    revision: number;
    counts: WormExecutionRunCounts;
    nextStepOrdinal: number;
    currentStep?: WormExecutionRunStep;
    pauseCode: string;
    failureCode: string;
    blockCode: string;
    requestedAt: number;
    authorizedAt: number;
    startedAt: number;
    pausedAt: number;
    completedAt: number;
    updatedAt: number;
    allowedActions: WormExecutionAllowedAction[];
    authorization: WormExecutionRunAuthorization;
    coordinator: WormExecutionRunCoordinator;
}

export interface ListWormExecutionRunsResult {
    items: WormExecutionRun[];
    total: number;
    page: number;
    pageSize: number;
}

export interface ListWormExecutionRunStepsResult {
    items: WormExecutionRunStep[];
    total: number;
    page: number;
    pageSize: number;
}

export interface WormExecutionCommandInput {
    commandId: string;
    expectedRevision: number;
}

export interface WormExecutionCoordinatorCommandInput extends WormExecutionCommandInput {
    coordinatorToken: string;
}

export interface WormExecutionCommandResult {
    run: WormExecutionRun;
    coordinatorToken: string;
}

export interface WormExecutionAuthorizationChallenge {
    message: string;
    expiresAt: number;
}

export type WormPositionCashOutOperationAllowedAction = 'AUTHORIZE_CASH_OUT' | 'CHECK_STATUS';

export interface WormPositionCashOutOperation {
    id: string;
    walletId: number;
    walletAddress: string;
    positionPubkey: string;
    positionRequestPubkey: string;
    marketConditionId: string;
    isYes: boolean;
    positionCreatedAt: number;
    totalShares: string;
    state: WormPositionCashOutState;
    stage: string;
    reasonCode: string;
    revision: number;
    proofKind: string;
    providerState: string;
    observedClosed: boolean;
    observedLiquidated: boolean;
    completionSource: string;
    authorizationExpiresAt: number;
    dispatchExpiresAt: number;
    requestedAt: number;
    authorizedAt: number;
    dispatchedAt: number;
    completedAt: number;
    updatedAt: number;
    allowedActions: WormPositionCashOutOperationAllowedAction[];
}

export interface CreateWormPositionCashOutInput {
    commandId: string;
    walletId: number;
    positionPubkey: string;
}

export interface WormPositionCashOutCommandInput {
    commandId: string;
    expectedRevision: number;
}

export interface WormPositionCashOutAuthorizationChallenge {
    message: string;
    expiresAt: number;
}

export type WormPositionCashOutBatchState =
    | 'BUILDING'
    | 'AWAITING_AUTHORIZATION'
    | 'QUEUED'
    | 'RUNNING'
    | 'PAUSE_REQUESTED'
    | 'PAUSED'
    | 'TERMINATE_REQUESTED'
    | 'TERMINATED'
    | 'COMPLETED'
    | 'FAILED'
    | 'RECONCILIATION_REQUIRED'
    | 'CANCELLED'
    | 'EXPIRED';

export type WormPositionCashOutBatchAllowedAction = 'AUTHORIZE_BATCH' | 'CANCEL' | 'PAUSE' | 'CONTINUE' | 'TERMINATE' | 'CHECK_STATUS';

export type WormPositionCashOutBatchItemState =
    | 'PENDING'
    | 'PREFLIGHTING'
    | 'CLOSING'
    | 'AWAITING_POSITION'
    | 'AWAITING_BALANCE'
    | 'COMPLETED'
    | 'FAILED'
    | 'RECONCILIATION_REQUIRED'
    | 'NOT_EXECUTED';

export interface WormPositionCashOutBatchBalanceEvidence {
    mint: string;
    decimals: number;
    atomicAmount: string;
    observedSlot: string;
}

export interface WormPositionCashOutBatchItem {
    id: string;
    ordinal: number;
    walletOrdinal: number;
    positionOrdinal: number;
    walletId: number;
    walletAddress: string;
    walletRemark: string;
    positionPubkey: string;
    positionRequestPubkey: string;
    marketConditionId: string;
    marketTitle: string;
    isYes: boolean;
    shares: string;
    positionCreatedAt: number;
    state: WormPositionCashOutBatchItemState;
    reasonCode: string;
    childOperationId: string;
    baseline?: WormPositionCashOutBatchBalanceEvidence;
    observed?: WormPositionCashOutBatchBalanceEvidence;
    deltaAtomicAmount: string;
    balanceDeadlineAt: number;
    completedAt: number;
    updatedAt: number;
}

export interface WormPositionCashOutBatchWallet {
    ordinal: number;
    walletId: number;
    address: string;
    remark: string;
    avatarKind: string;
    avatarPresetId: string;
    avatarUrl: string;
    positionCount: number;
    completedCount: number;
}

export interface WormPositionCashOutBatch {
    id: string;
    state: WormPositionCashOutBatchState;
    revision: number;
    reasonCode: string;
    walletCount: number;
    positionCount: number;
    completedCount: number;
    notExecutedCount: number;
    currentWalletOrdinal: number;
    currentPositionOrdinal: number;
    currentItem?: WormPositionCashOutBatchItem;
    wallets: WormPositionCashOutBatchWallet[];
    authorizationExpiresAt: number;
    executionStartedAt: number;
    completedAt: number;
    createdAt: number;
    updatedAt: number;
    allowedActions: WormPositionCashOutBatchAllowedAction[];
}

export interface CreateWormPositionCashOutBatchInput {
    commandId: string;
    walletIds: number[];
}

export interface WormPositionCashOutBatchCommandInput {
    commandId: string;
    expectedRevision: number;
}

export interface ListWormPositionCashOutBatchItemsResult {
    items: WormPositionCashOutBatchItem[];
    total: number;
    page: number;
    pageSize: number;
}

export interface WormPositionCashOutBatchAuthorizationChallenge {
    message: string;
    expiresAt: number;
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

const normalizePositionCashOutState = (value: unknown): WormPositionCashOutState => {
    switch (value) {
        case 'AWAITING_AUTHORIZATION':
        case 'QUEUED':
        case 'PREFLIGHTING':
        case 'CLOSING':
        case 'AWAITING_COMPLETION':
        case 'COMPLETED':
        case 'FAILED':
        case 'RECONCILIATION_REQUIRED':
        case 'EXPIRED':
            return value;
        default:
            return invalidWormTradingResponse();
    }
};

const normalizePositionCashOutAllowedAction = (value: unknown): WormPositionCashOutAllowedAction => {
    switch (value) {
        case 'CASH_OUT':
        case 'AUTHORIZE_CASH_OUT':
        case 'CHECK_STATUS':
        case 'NONE':
            return value;
        default:
            return invalidWormTradingResponse();
    }
};

const normalizePositionCashOutProjection = (value: unknown): WormPositionCashOutProjection => {
    const item = requireRecord(value);
    const operationId = readString(item, 'operationId');
    const rawState = readValue(item, 'state');
    const state = rawState === '' || rawState === undefined || rawState === null ? '' : normalizePositionCashOutState(rawState);
    const reasonCode = readString(item, 'reasonCode');
    const allowedAction = normalizePositionCashOutAllowedAction(item.allowedAction);
    const batchId = readString(item, 'batchId');
    const rawBatchState = readValue(item, 'batchState');
    const batchState = rawBatchState === '' || rawBatchState === undefined || rawBatchState === null ? '' : normalizePositionCashOutBatchState(rawBatchState);
    const rawBatchItemState = readValue(item, 'batchItemState');
    const batchItemState =
        rawBatchItemState === '' || rawBatchItemState === undefined || rawBatchItemState === null ? '' : normalizePositionCashOutBatchItemState(rawBatchItemState);
    const batchLockReasonCode = readString(item, 'batchLockReasonCode');
    const revision = optionalInteger(item, 0, 0, 'revision');
    const updatedAt = optionalInteger(item, 0, 0, 'updatedAt');
    if (
        (reasonCode && !/^[A-Z][A-Z0-9_]{0,127}$/.test(reasonCode)) ||
        (batchLockReasonCode && !/^[A-Z][A-Z0-9_]{0,127}$/.test(batchLockReasonCode)) ||
        (!batchId && (batchState !== '' || batchItemState !== '' || batchLockReasonCode !== '')) ||
        (batchId && (batchState === '' || !canonicalWormTradingUUIDPattern.test(batchId))) ||
        (!operationId && !batchId && (state !== '' || revision !== 0 || updatedAt !== 0 || (allowedAction !== 'CASH_OUT' && allowedAction !== 'NONE'))) ||
        (!operationId && batchId && (state !== '' || revision < 1 || updatedAt < 1 || allowedAction !== 'NONE')) ||
        (operationId && (state === '' || revision < 1 || updatedAt < 1 || allowedAction === 'CASH_OUT'))
    ) {
        return invalidWormTradingResponse();
    }
    return {
        operationId,
        state,
        reasonCode,
        allowedAction,
        batchId,
        batchState,
        batchItemState,
        batchLockReasonCode,
        revision,
        updatedAt
    };
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
        liquidationPrice: readString(item, 'liquidationPrice', 'liquidation_price'),
        unrealizedPnL: readString(item, 'unrealizedPnl', 'unrealized_pnl'),
        realizedPnL: requireString(item, 'realizedPnl', 'realized_pnl'),
        isClosed: readBoolean(item, 'isClosed', 'is_closed'),
        isLiquidated: readBoolean(item, 'isLiquidated', 'is_liquidated'),
        isClaimed: readBoolean(item, 'isClaimed', 'is_claimed'),
        createdAt: optionalInteger(item, 0, 0, 'createdAt', 'created_at'),
        cashOut: normalizePositionCashOutProjection(readValue(item, 'cashOut'))
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
    map: (value: Record<string, unknown>) => T,
    scope = method === 'GET' ? readScope : writeScope
): AbortableWormTradingPromise<T> => {
    const request = requests.scopedFetch(
        requests.toAbsURL(path),
        {
            method,
            credentials: 'same-origin',
            mode: 'same-origin',
            redirect: 'error',
            headers: {'Accept': 'application/json', 'Content-Type': 'application/json'},
            body: body ? JSON.stringify(body) : undefined
        },
        scope
    );
    const promise = request.then(async response => {
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
    promise.abort = request.abort;
    return promise;
};

const rawReauthenticationPost = <T>(path: string, body: Record<string, unknown>, map: (value: Record<string, unknown>) => T): AbortableWormTradingPromise<T> =>
    rawSameOriginRequest('POST', path, body, 'Worm credential reauthentication failed', map);

const rawOptionalSameOriginGet = <T>(path: string, fallbackError: string, map: (value: Record<string, unknown>) => T): AbortableWormTradingPromise<T | undefined> => {
    const request = rawSameOriginRequest('GET', path, undefined, fallbackError, map);
    const promise = request.catch(reason => {
        if (isRecord(reason) && reason.status === 404) {
            return undefined;
        }
        throw reason;
    }) as AbortableWormTradingPromise<T | undefined>;
    promise.abort = request.abort;
    return promise;
};

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

const optionalExactBoolean = (item: unknown, name: string, fallback = false): boolean => {
    const value = requireRecord(item)[name];
    if (value === undefined || value === null) {
        return fallback;
    }
    if (typeof value !== 'boolean') {
        return invalidWormTradingResponse();
    }
    return value;
};

const executionCodePattern = /^[A-Z][A-Z0-9_]{0,127}$/;
const canonicalWormTradingUUIDPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;

const normalizePositionCashOutOperationAllowedAction = (value: unknown): WormPositionCashOutOperationAllowedAction => {
    if (value === 'AUTHORIZE_CASH_OUT' || value === 'CHECK_STATUS') {
        return value;
    }
    return invalidWormTradingResponse();
};

const normalizePositionCashOutOperation = (value: unknown, expected?: {id?: string; walletId?: number; positionPubkey?: string}): WormPositionCashOutOperation => {
    const item = requireRecord(value);
    const id = requireExactString(item, 'id');
    const walletId = requireInteger(item, 1, 'walletId');
    const positionPubkey = requireExactString(item, 'positionPubkey');
    const reasonCode = optionalExactString(item, 'reasonCode');
    const allowedActions = requireExactArray(item, 'allowedActions').map(normalizePositionCashOutOperationAllowedAction);
    if (
        (expected?.id && id !== expected.id) ||
        (expected?.walletId && walletId !== expected.walletId) ||
        (expected?.positionPubkey && positionPubkey !== expected.positionPubkey) ||
        (reasonCode && !executionCodePattern.test(reasonCode)) ||
        new Set(allowedActions).size !== allowedActions.length
    ) {
        return invalidWormTradingResponse();
    }
    return {
        id,
        walletId,
        walletAddress: requireExactString(item, 'walletAddress'),
        positionPubkey,
        positionRequestPubkey: optionalExactString(item, 'positionRequestPubkey'),
        marketConditionId: requireExactString(item, 'marketConditionId'),
        isYes: requireExactBoolean(item, 'isYes'),
        positionCreatedAt: requireInteger(item, 1, 'positionCreatedAt'),
        totalShares: requireExactString(item, 'totalShares'),
        state: normalizePositionCashOutState(item.state),
        stage: requireExactString(item, 'stage'),
        reasonCode,
        revision: requireInteger(item, 1, 'revision'),
        proofKind: optionalExactString(item, 'proofKind'),
        providerState: optionalExactString(item, 'providerState'),
        observedClosed: requireExactBoolean(item, 'observedClosed'),
        observedLiquidated: requireExactBoolean(item, 'observedLiquidated'),
        completionSource: optionalExactString(item, 'completionSource'),
        authorizationExpiresAt: optionalInteger(item, 0, 0, 'authorizationExpiresAt'),
        dispatchExpiresAt: optionalInteger(item, 0, 0, 'dispatchExpiresAt'),
        requestedAt: requireInteger(item, 1, 'requestedAt'),
        authorizedAt: optionalInteger(item, 0, 0, 'authorizedAt'),
        dispatchedAt: optionalInteger(item, 0, 0, 'dispatchedAt'),
        completedAt: optionalInteger(item, 0, 0, 'completedAt'),
        updatedAt: requireInteger(item, 1, 'updatedAt'),
        allowedActions
    };
};

const normalizePositionCashOutBatchState = (value: unknown): WormPositionCashOutBatchState => {
    switch (value) {
        case 'BUILDING':
        case 'AWAITING_AUTHORIZATION':
        case 'QUEUED':
        case 'RUNNING':
        case 'PAUSE_REQUESTED':
        case 'PAUSED':
        case 'TERMINATE_REQUESTED':
        case 'TERMINATED':
        case 'COMPLETED':
        case 'FAILED':
        case 'RECONCILIATION_REQUIRED':
        case 'CANCELLED':
        case 'EXPIRED':
            return value;
        default:
            return invalidWormTradingResponse();
    }
};

const normalizePositionCashOutBatchAllowedAction = (value: unknown): WormPositionCashOutBatchAllowedAction => {
    switch (value) {
        case 'AUTHORIZE_BATCH':
        case 'CANCEL':
        case 'PAUSE':
        case 'CONTINUE':
        case 'TERMINATE':
        case 'CHECK_STATUS':
            return value;
        default:
            return invalidWormTradingResponse();
    }
};

const normalizePositionCashOutBatchItemState = (value: unknown): WormPositionCashOutBatchItemState => {
    switch (value) {
        case 'PENDING':
        case 'PREFLIGHTING':
        case 'CLOSING':
        case 'AWAITING_POSITION':
        case 'AWAITING_BALANCE':
        case 'COMPLETED':
        case 'FAILED':
        case 'RECONCILIATION_REQUIRED':
        case 'NOT_EXECUTED':
            return value;
        default:
            return invalidWormTradingResponse();
    }
};

const normalizePositionCashOutBatchBalanceEvidence = (value: unknown): WormPositionCashOutBatchBalanceEvidence | undefined => {
    if (value === undefined || value === null) {
        return undefined;
    }
    const item = requireRecord(value);
    const atomicAmount = requireExactString(item, 'atomicAmount');
    const observedSlot = requireExactString(item, 'observedSlot');
    const evidence = {
        mint: requireExactString(item, 'mint'),
        decimals: requireInteger(item, 0, 'decimals'),
        atomicAmount,
        observedSlot
    };
    if (evidence.decimals !== 6 || !/^\d+$/.test(atomicAmount) || !/^[1-9]\d*$/.test(observedSlot)) {
        return invalidWormTradingResponse();
    }
    return evidence;
};

const normalizePositionCashOutBatchItem = (value: unknown): WormPositionCashOutBatchItem => {
    const item = requireRecord(value);
    const id = requireExactString(item, 'id');
    const childOperationId = optionalExactString(item, 'childOperationId');
    const reasonCode = optionalExactString(item, 'reasonCode');
    const shares = requireExactString(item, 'shares');
    const deltaAtomicAmount = optionalExactString(item, 'deltaAtomicAmount');
    if (
        !canonicalWormTradingUUIDPattern.test(id) ||
        (childOperationId !== '' && !canonicalWormTradingUUIDPattern.test(childOperationId)) ||
        (reasonCode !== '' && !executionCodePattern.test(reasonCode)) ||
        !/^\d+(?:\.\d+)?$/.test(shares) ||
        /^0(?:\.0+)?$/.test(shares) ||
        (deltaAtomicAmount !== '' && !/^[1-9]\d*$/.test(deltaAtomicAmount))
    ) {
        return invalidWormTradingResponse();
    }
    return {
        id,
        ordinal: requireInteger(item, 1, 'ordinal'),
        walletOrdinal: requireInteger(item, 1, 'walletOrdinal'),
        positionOrdinal: requireInteger(item, 1, 'positionOrdinal'),
        walletId: requireInteger(item, 1, 'walletId'),
        walletAddress: requireExactString(item, 'walletAddress'),
        walletRemark: optionalExactString(item, 'walletRemark'),
        positionPubkey: requireExactString(item, 'positionPubkey'),
        positionRequestPubkey: optionalExactString(item, 'positionRequestPubkey'),
        marketConditionId: requireExactString(item, 'marketConditionId'),
        marketTitle: optionalExactString(item, 'marketTitle'),
        isYes: requireExactBoolean(item, 'isYes'),
        shares,
        positionCreatedAt: requireInteger(item, 1, 'positionCreatedAt'),
        state: normalizePositionCashOutBatchItemState(item.state),
        reasonCode,
        childOperationId,
        baseline: normalizePositionCashOutBatchBalanceEvidence(item.baseline),
        observed: normalizePositionCashOutBatchBalanceEvidence(item.observed),
        deltaAtomicAmount,
        balanceDeadlineAt: optionalInteger(item, 0, 0, 'balanceDeadlineAt'),
        completedAt: optionalInteger(item, 0, 0, 'completedAt'),
        updatedAt: requireInteger(item, 1, 'updatedAt')
    };
};

const normalizePositionCashOutBatchWallet = (value: unknown): WormPositionCashOutBatchWallet => {
    const item = requireRecord(value);
    const positionCount = requireInteger(item, 0, 'positionCount');
    const completedCount = requireInteger(item, 0, 'completedCount');
    if (completedCount > positionCount) {
        return invalidWormTradingResponse();
    }
    return {
        ordinal: requireInteger(item, 1, 'ordinal'),
        walletId: requireInteger(item, 1, 'walletId'),
        address: requireExactString(item, 'address'),
        remark: optionalExactString(item, 'remark'),
        avatarKind: optionalExactString(item, 'avatarKind'),
        avatarPresetId: optionalExactString(item, 'avatarPresetId'),
        avatarUrl: optionalExactString(item, 'avatarUrl'),
        positionCount,
        completedCount
    };
};

const normalizePositionCashOutBatch = (value: unknown, expectedID?: string): WormPositionCashOutBatch => {
    const item = requireRecord(value);
    const id = requireExactString(item, 'id');
    const state = normalizePositionCashOutBatchState(item.state);
    const reasonCode = optionalExactString(item, 'reasonCode');
    const walletCount = requireInteger(item, 1, 'walletCount');
    const positionCount = requireInteger(item, 0, 'positionCount');
    const completedCount = requireInteger(item, 0, 'completedCount');
    const notExecutedCount = requireInteger(item, 0, 'notExecutedCount');
    const currentWalletOrdinal = optionalInteger(item, 0, 0, 'currentWalletOrdinal');
    const currentPositionOrdinal = optionalInteger(item, 0, 0, 'currentPositionOrdinal');
    const wallets = requireExactArray(item, 'wallets').map(normalizePositionCashOutBatchWallet);
    const allowedActions = requireExactArray(item, 'allowedActions').map(normalizePositionCashOutBatchAllowedAction);
    const rawCurrentItem = item.currentItem;
    const currentItem = rawCurrentItem === undefined || rawCurrentItem === null ? undefined : normalizePositionCashOutBatchItem(rawCurrentItem);
    const walletIDs = new Set(wallets.map(wallet => wallet.walletId));
    const walletAddresses = new Set(wallets.map(wallet => wallet.address));
    if (
        !canonicalWormTradingUUIDPattern.test(id) ||
        (expectedID && id !== expectedID) ||
        (reasonCode !== '' && !executionCodePattern.test(reasonCode)) ||
        wallets.length !== walletCount ||
        walletIDs.size !== wallets.length ||
        walletAddresses.size !== wallets.length ||
        wallets.some((wallet, index) => wallet.ordinal !== index + 1) ||
        wallets.reduce((total, wallet) => total + wallet.positionCount, 0) !== positionCount ||
        completedCount > positionCount ||
        notExecutedCount > positionCount - completedCount ||
        wallets.reduce((total, wallet) => total + wallet.completedCount, 0) !== completedCount ||
        new Set(allowedActions).size !== allowedActions.length ||
        currentWalletOrdinal > walletCount ||
        (currentItem !== undefined &&
            (currentItem.walletOrdinal !== currentWalletOrdinal || currentItem.positionOrdinal !== currentPositionOrdinal || !walletIDs.has(currentItem.walletId))) ||
        (currentItem === undefined && (currentWalletOrdinal !== 0 || currentPositionOrdinal !== 0))
    ) {
        return invalidWormTradingResponse();
    }
    return {
        id,
        state,
        revision: requireInteger(item, 1, 'revision'),
        reasonCode,
        walletCount,
        positionCount,
        completedCount,
        notExecutedCount,
        currentWalletOrdinal,
        currentPositionOrdinal,
        currentItem,
        wallets,
        authorizationExpiresAt: optionalInteger(item, 0, 0, 'authorizationExpiresAt'),
        executionStartedAt: optionalInteger(item, 0, 0, 'executionStartedAt'),
        completedAt: optionalInteger(item, 0, 0, 'completedAt'),
        createdAt: requireInteger(item, 1, 'createdAt'),
        updatedAt: requireInteger(item, 1, 'updatedAt'),
        allowedActions
    };
};

const normalizePositionCashOutBatchItems = (value: unknown, expectedPage: number, expectedPageSize: number): ListWormPositionCashOutBatchItemsResult => {
    const item = requireRecord(value);
    const items = requireExactArray(item, 'items').map(normalizePositionCashOutBatchItem);
    const total = requireInteger(item, 0, 'total');
    const page = requireInteger(item, 1, 'page');
    const pageSize = requireInteger(item, 1, 'pageSize');
    const offset = (page - 1) * pageSize;
    if (
        page !== expectedPage ||
        pageSize !== expectedPageSize ||
        items.length > pageSize ||
        (items.length === 0 ? offset < total : offset >= total || offset + items.length > total) ||
        items.some((batchItem, index) => batchItem.ordinal !== offset + index + 1) ||
        new Set(items.map(batchItem => batchItem.id)).size !== items.length
    ) {
        return invalidWormTradingResponse();
    }
    return {items, total, page, pageSize};
};

const normalizeExecutionCodeCounts = (value: unknown): Record<string, number> => {
    if (value === undefined || value === null) {
        return {};
    }
    const record = requireRecord(value);
    const result: Record<string, number> = {};
    for (const [code, count] of Object.entries(record)) {
        if (!executionCodePattern.test(code) || !Number.isSafeInteger(count) || Number(count) < 0) {
            return invalidWormTradingResponse();
        }
        result[code] = Number(count);
    }
    return result;
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

const nonNegativeExactDecimalPattern = /^\d+(?:\.\d+)?$/;

const optionalExactDecimal = (item: unknown, name: string): string => {
    const value = optionalExactString(item, name);
    if (value !== '' && (value.length > 128 || !nonNegativeExactDecimalPattern.test(value))) {
        return invalidWormTradingResponse();
    }
    return value;
};

const normalizeExecutionPlanState = (value: unknown): WormExecutionPlanState => {
    switch (value) {
        case 'BUILDING':
        case 'READY':
        case 'FAILED':
        case 'EXPIRED':
            return value;
        default:
            return invalidWormTradingResponse();
    }
};

const normalizeExecutionPlanAsset = (value: unknown, incompleteAllowed: boolean): WormExecutionPlanAssetBalance => {
    const item = requireRecord(value);
    const availability = optionalExactString(item, 'availability');
    const normalizedAvailability = availability.toUpperCase();
    if (incompleteAllowed && normalizedAvailability === '') {
        const atomicAmount = optionalExactString(item, 'atomicAmount');
        const observedSlot = optionalExactString(item, 'observedSlot');
        if ((atomicAmount !== '' && !/^\d+$/.test(atomicAmount)) || (observedSlot !== '' && !/^\d+$/.test(observedSlot))) {
            return invalidWormTradingResponse();
        }
        return {
            atomicAmount,
            amount: optionalExactDecimal(item, 'amount'),
            decimals: optionalInteger(item, 0, 0, 'decimals'),
            observedSlot,
            availability: '',
            errorCode: optionalExactString(item, 'errorCode')
        };
    }
    if (
        normalizedAvailability !== 'AVAILABLE' &&
        normalizedAvailability !== 'UNAVAILABLE' &&
        normalizedAvailability !== 'BALANCE_AVAILABILITY_AVAILABLE' &&
        normalizedAvailability !== 'BALANCE_AVAILABILITY_UNAVAILABLE'
    ) {
        return invalidWormTradingResponse();
    }
    const available = normalizedAvailability.endsWith('_AVAILABLE') || normalizedAvailability === 'AVAILABLE';
    const atomicAmount = optionalExactString(item, 'atomicAmount');
    const amount = optionalExactDecimal(item, 'amount');
    const observedSlot = optionalExactString(item, 'observedSlot');
    if (available && (!/^\d+$/.test(atomicAmount) || amount === '' || !/^\d+$/.test(observedSlot))) {
        return invalidWormTradingResponse();
    }
    if (atomicAmount !== '' && !/^\d+$/.test(atomicAmount)) {
        return invalidWormTradingResponse();
    }
    if (observedSlot !== '' && !/^\d+$/.test(observedSlot)) {
        return invalidWormTradingResponse();
    }
    return {
        atomicAmount,
        amount,
        decimals: optionalInteger(item, 0, 0, 'decimals'),
        observedSlot,
        availability: normalizedAvailability,
        errorCode: optionalExactString(item, 'errorCode')
    };
};

const normalizeExecutionPlanWallet = (value: unknown, incompleteAllowed: boolean): WormExecutionPlanWallet => {
    const item = requireRecord(value);
    const wallet = normalizeWalletSummary(item.wallet);
    const connectionRecord = requireRecord(item.connection);
    const connectionState = readString(connectionRecord, 'state');
    const connection = incompleteAllowed && connectionState === '' ? undefined : normalizeConnection(connectionRecord);
    const sol = normalizeExecutionPlanAsset(item.sol, incompleteAllowed);
    const usdc = normalizeExecutionPlanAsset(item.usdc, incompleteAllowed);
    return {
        ordinal: requireInteger(item, 1, 'ordinal'),
        wallet,
        connection,
        sol,
        usdc,
        status: optionalExactString(item, 'status'),
        reasonCode: optionalExactString(item, 'reasonCode')
    };
};

const normalizeExecutionPlanEstimate = (value: unknown): WormExecutionPlanEstimate | undefined => {
    if (value === undefined || value === null) {
        return undefined;
    }
    const item = requireRecord(value);
    return {
        averagePrice: optionalExactDecimal(item, 'averagePrice'),
        totalShares: optionalExactDecimal(item, 'totalShares'),
        totalCost: optionalExactDecimal(item, 'totalCost'),
        bestAsk: optionalExactDecimal(item, 'bestAsk'),
        worstFillPrice: optionalExactDecimal(item, 'worstFillPrice'),
        isFullyFilled: optionalExactBoolean(item, 'isFullyFilled'),
        feeAmount: optionalExactDecimal(item, 'feeAmount'),
        userFundsNeeded: optionalExactDecimal(item, 'userFundsNeeded'),
        liquidationPrice: optionalExactDecimal(item, 'liquidationPrice')
    };
};

const normalizeExecutionPlanItem = (value: unknown): WormExecutionPlanItem => {
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
        outcomeLabel: requireExactString(item, 'outcomeLabel'),
        backend: optionalExactString(item, 'backend'),
        funds: optionalExactDecimal(item, 'funds'),
        leverage: optionalExactDecimal(item, 'leverage'),
        state: optionalExactString(item, 'state'),
        reasonCode: optionalExactString(item, 'reasonCode'),
        estimate: normalizeExecutionPlanEstimate(item.estimate)
    };
};

const normalizeExecutionPlan = (value: unknown, expectedPlanID?: string): WormExecutionPlan => {
    const item = requireRecord(value);
    const combination = requireRecord(item.combination);
    const id = requireExactString(item, 'id');
    const state = normalizeExecutionPlanState(item.state);
    const wallets = readRepeatedArray(item, 'wallets').map(wallet => normalizeExecutionPlanWallet(wallet, state === 'BUILDING' || state === 'FAILED'));
    const items = readRepeatedArray(item, 'items').map(normalizeExecutionPlanItem);
    const walletCount = requireInteger(item, 1, 'walletCount');
    const itemCount = requireInteger(item, 1, 'itemCount');
    const totalStepCount = requireInteger(item, 1, 'totalStepCount');
    const completedStepCount = optionalInteger(item, 0, 0, 'completedStepCount');
    const readyStepCount = optionalInteger(item, 0, 0, 'readyStepCount');
    const skippedStepCount = optionalInteger(item, 0, 0, 'skippedStepCount');
    const reasonCounts = normalizeExecutionCodeCounts(item.reasonCounts);
    const classifiedReasonCount = Object.values(reasonCounts).reduce((sum, count) => sum + count, 0);
    if (
        (expectedPlanID && id !== expectedPlanID) ||
        new Set(wallets.map(wallet => wallet.wallet.walletId)).size !== wallets.length ||
        new Set(wallets.map(wallet => wallet.wallet.address)).size !== wallets.length ||
        new Set(items.map(planItem => planItem.marketConditionId)).size !== items.length ||
        wallets.some((wallet, index) => wallet.ordinal !== index + 1) ||
        items.some((planItem, index) => planItem.ordinal !== index + 1) ||
        wallets.length !== walletCount ||
        items.length !== itemCount ||
        totalStepCount !== walletCount * itemCount ||
        completedStepCount > totalStepCount ||
        readyStepCount + skippedStepCount > totalStepCount ||
        ((state === 'READY' || state === 'EXPIRED') && (completedStepCount !== totalStepCount || readyStepCount + skippedStepCount !== totalStepCount)) ||
        classifiedReasonCount > readyStepCount + skippedStepCount ||
        ((state === 'READY' || state === 'EXPIRED') && classifiedReasonCount !== readyStepCount + skippedStepCount) ||
        (state === 'FAILED' && optionalExactString(item, 'failureCode') === '')
    ) {
        return invalidWormTradingResponse();
    }
    return {
        id,
        combinationId: requireExactString(combination, 'id'),
        combinationName: requireExactString(combination, 'name'),
        combinationRevision: requireInteger(combination, 1, 'revision'),
        state,
        buildStage: optionalExactString(item, 'buildStage'),
        failureCode: optionalExactString(item, 'failureCode'),
        usabilityCode: optionalExactString(item, 'usabilityCode'),
        walletCount,
        itemCount,
        totalStepCount,
        completedStepCount,
        readyStepCount,
        skippedStepCount,
        reasonCounts,
        maximumCollateral: optionalExactDecimal(item, 'maximumCollateral'),
        openingFeeEstimate: optionalExactDecimal(item, 'openingFeeEstimate'),
        totalUSDCNeeded: optionalExactDecimal(item, 'totalUSDCNeeded'),
        requestedAt: requireInteger(item, 0, 'requestedAt'),
        completedAt: optionalInteger(item, 0, 0, 'completedAt'),
        expiresAt: optionalInteger(item, 0, 0, 'expiresAt'),
        retentionUntil: optionalInteger(item, 0, 0, 'retentionUntil'),
        createdAt: optionalInteger(item, 0, 0, 'createdAt'),
        updatedAt: optionalInteger(item, 0, 0, 'updatedAt'),
        wallets,
        items
    };
};

const validateCreatedExecutionPlan = (plan: WormExecutionPlan, input: CreateWormExecutionPlanInput): WormExecutionPlan => {
    const walletIDs = plan.wallets.map(wallet => wallet.wallet.walletId);
    if (
        plan.state !== 'BUILDING' ||
        plan.combinationId !== input.combinationId.trim() ||
        plan.combinationRevision !== input.expectedCombinationRevision ||
        walletIDs.length !== input.walletIds.length ||
        walletIDs.some((walletID, index) => walletID !== input.walletIds[index])
    ) {
        return invalidWormTradingResponse();
    }
    return plan;
};

const normalizeExecutionPlanStepDisposition = (value: unknown): WormExecutionPlanStepDisposition => {
    if (value === 'READY' || value === 'SKIPPED') {
        return value;
    }
    return invalidWormTradingResponse();
};

const normalizeExecutionPlanStep = (value: unknown): WormExecutionPlanStep => {
    const item = requireRecord(value);
    const disposition = normalizeExecutionPlanStepDisposition(item.disposition);
    const reasonCode = optionalExactString(item, 'reasonCode');
    if ((disposition === 'READY' && reasonCode !== '') || (disposition === 'SKIPPED' && reasonCode === '')) {
        return invalidWormTradingResponse();
    }
    return {
        ordinal: requireInteger(item, 1, 'ordinal'),
        walletOrdinal: requireInteger(item, 1, 'walletOrdinal'),
        itemOrdinal: requireInteger(item, 1, 'itemOrdinal'),
        disposition,
        reasonCode,
        projectedUsdcBefore: optionalExactDecimal(item, 'projectedUSDCBefore'),
        projectedUsdcAfter: optionalExactDecimal(item, 'projectedUSDCAfter')
    };
};

const normalizeExecutionRunState = (value: unknown): WormExecutionRunState => {
    switch (value) {
        case 'AWAITING_AUTHORIZATION':
        case 'AUTHORIZED':
        case 'RUNNING':
        case 'PAUSE_REQUESTED':
        case 'PAUSED':
        case 'TERMINATE_REQUESTED':
        case 'RECONCILIATION_REQUIRED':
        case 'COMPLETED':
        case 'TERMINATED':
        case 'FAILED':
            return value;
        default:
            return invalidWormTradingResponse();
    }
};

const normalizeExecutionStepState = (value: unknown): WormExecutionStepState => {
    switch (value) {
        case 'PENDING':
        case 'PREFLIGHTING':
        case 'OPENING':
        case 'OPENED':
        case 'SIGNING':
        case 'FINALIZING':
        case 'AWAITING_COMPLETION':
        case 'COMPLETED':
        case 'SATISFIED':
        case 'SKIPPED':
        case 'FAILED':
        case 'NOT_EXECUTED':
        case 'OUTCOME_UNKNOWN':
            return value;
        default:
            return invalidWormTradingResponse();
    }
};

const normalizeExecutionAllowedAction = (value: unknown): WormExecutionAllowedAction => {
    switch (value) {
        case 'AUTHORIZE':
        case 'START':
        case 'PAUSE':
        case 'CONTINUE':
        case 'TERMINATE':
        case 'HEARTBEAT':
        case 'EXECUTE_NEXT':
        case 'RECONCILE':
            return value;
        default:
            return invalidWormTradingResponse();
    }
};

const optionalUnsignedIntegerText = (item: unknown, name: string): string => {
    const value = requireRecord(item)[name];
    if (value === undefined || value === null || value === '' || value === 0 || value === '0') {
        return '';
    }
    const normalized = typeof value === 'number' && Number.isSafeInteger(value) ? String(value) : typeof value === 'string' ? value : '';
    if (!/^[1-9]\d*$/.test(normalized)) {
        return invalidWormTradingResponse();
    }
    return normalized;
};

const normalizeExecutionRunStep = (value: unknown): WormExecutionRunStep => {
    const item = requireRecord(value);
    const wallet = normalizeWalletSummary(item.wallet);
    const marketValue = requireRecord(item.market);
    const market = {
        eventConditionId: requireExactString(marketValue, 'eventConditionId'),
        eventTitle: requireExactString(marketValue, 'eventTitle'),
        eventLogo: optionalExactString(marketValue, 'eventLogo'),
        marketConditionId: requireExactString(marketValue, 'marketConditionId'),
        marketTitle: requireExactString(marketValue, 'marketTitle'),
        marketLogo: optionalExactString(marketValue, 'marketLogo'),
        outcomeLabel: requireExactString(marketValue, 'outcomeLabel'),
        backend: requireExactString(marketValue, 'backend')
    };
    const state = normalizeExecutionStepState(item.state);
    const reasonCode = optionalExactString(item, 'reasonCode');
    const providerState = optionalExactString(item, 'providerState');
    const completionSource = optionalExactString(item, 'completionSource');
    const completionPositionPubkey = optionalExactString(item, 'completionPositionPubkey');
    const completionPositionRequestPubkey = optionalExactString(item, 'completionPositionRequestPubkey');
    const completionPositionCreatedAt = optionalInteger(item, 0, 0, 'completionPositionCreatedAt');
    if (state === 'OUTCOME_UNKNOWN' && reasonCode === '') {
        return invalidWormTradingResponse();
    }
    if (
        (state === 'COMPLETED' && (completionSource !== 'OPEN_POSITION' || completionPositionPubkey.trim() === '' || completionPositionCreatedAt <= 0)) ||
        (state !== 'COMPLETED' && (completionSource !== '' || completionPositionPubkey !== '' || completionPositionRequestPubkey !== '' || completionPositionCreatedAt !== 0))
    ) {
        return invalidWormTradingResponse();
    }
    return {
        id: requireExactString(item, 'id'),
        ordinal: requireInteger(item, 1, 'ordinal'),
        wallet,
        market,
        side: normalizeOutcomeSide(item.side),
        funds: optionalExactDecimal(item, 'funds'),
        leverage: optionalExactDecimal(item, 'leverage'),
        state,
        reasonCode,
        positionRequestId: optionalUnsignedIntegerText(item, 'positionRequestId'),
        providerState,
        providerOrderState: optionalExactString(item, 'providerOrderState'),
        completionSource,
        completionPositionPubkey,
        completionPositionRequestPubkey,
        completionPositionCreatedAt,
        startedAt: optionalInteger(item, 0, 0, 'startedAt'),
        updatedAt: optionalInteger(item, 0, 0, 'updatedAt'),
        completedAt: optionalInteger(item, 0, 0, 'completedAt')
    };
};

const normalizeExecutionRun = (value: unknown, expectedRunID?: string): WormExecutionRun => {
    const item = requireRecord(value);
    const combination = requireRecord(item.combination);
    const countsValue = requireRecord(item.counts);
    const authorizationValue = requireRecord(item.authorization);
    const coordinatorValue = requireRecord(item.coordinator);
    const id = requireExactString(item, 'id');
    const state = normalizeExecutionRunState(item.state);
    const counts: WormExecutionRunCounts = {
        total: requireInteger(countsValue, 1, 'total'),
        actionable: requireInteger(countsValue, 1, 'actionable'),
        terminal: optionalInteger(countsValue, 0, 0, 'terminal'),
        completed: optionalInteger(countsValue, 0, 0, 'completed'),
        satisfied: optionalInteger(countsValue, 0, 0, 'satisfied'),
        skipped: optionalInteger(countsValue, 0, 0, 'skipped'),
        failed: optionalInteger(countsValue, 0, 0, 'failed'),
        notExecuted: optionalInteger(countsValue, 0, 0, 'notExecuted')
    };
    if (
        (expectedRunID && id !== expectedRunID) ||
        counts.actionable > counts.total ||
        counts.terminal > counts.total ||
        counts.completed + counts.satisfied + counts.skipped + counts.failed + counts.notExecuted !== counts.terminal
    ) {
        return invalidWormTradingResponse();
    }
    const allowedActions = requireExactArray(item, 'allowedActions').map(normalizeExecutionAllowedAction);
    if (new Set(allowedActions).size !== allowedActions.length) {
        return invalidWormTradingResponse();
    }
    const nextStepOrdinal = optionalInteger(item, 0, 0, 'nextStepOrdinal');
    if (nextStepOrdinal > counts.total || (allowedActions.includes('EXECUTE_NEXT') && nextStepOrdinal === 0)) {
        return invalidWormTradingResponse();
    }
    const currentStepValue = item.currentStep;
    const currentStep = currentStepValue === undefined || currentStepValue === null ? undefined : normalizeExecutionRunStep(currentStepValue);
    if (currentStep && currentStep.ordinal > counts.total) {
        return invalidWormTradingResponse();
    }
    return {
        id,
        planId: requireExactString(item, 'planId'),
        combinationId: requireExactString(combination, 'id'),
        combinationName: requireExactString(combination, 'name'),
        combinationRevision: requireInteger(combination, 1, 'revision'),
        state,
        revision: requireInteger(item, 1, 'revision'),
        counts,
        nextStepOrdinal,
        currentStep,
        pauseCode: optionalExactString(item, 'pauseCode'),
        failureCode: optionalExactString(item, 'failureCode'),
        blockCode: optionalExactString(item, 'blockCode'),
        requestedAt: requireInteger(item, 1, 'requestedAt'),
        authorizedAt: optionalInteger(item, 0, 0, 'authorizedAt'),
        startedAt: optionalInteger(item, 0, 0, 'startedAt'),
        pausedAt: optionalInteger(item, 0, 0, 'pausedAt'),
        completedAt: optionalInteger(item, 0, 0, 'completedAt'),
        updatedAt: requireInteger(item, 1, 'updatedAt'),
        allowedActions,
        authorization: {
            state: optionalExactString(authorizationValue, 'state'),
            proofKind: optionalExactString(authorizationValue, 'proofKind'),
            authorizedAt: optionalInteger(authorizationValue, 0, 0, 'authorizedAt'),
            requiresReauthorization: optionalExactBoolean(authorizationValue, 'requiresReauthorization')
        },
        coordinator: {
            state: optionalExactString(coordinatorValue, 'state'),
            generation: optionalInteger(coordinatorValue, 0, 0, 'generation'),
            leaseExpiresAt: optionalInteger(coordinatorValue, 0, 0, 'leaseExpiresAt')
        }
    };
};

const normalizeExecutionCommandResult = (value: unknown, expectedRunID: string): WormExecutionCommandResult => {
    const item = requireRecord(value);
    return {
        run: normalizeExecutionRun(item.run || item, expectedRunID),
        coordinatorToken: optionalExactString(item, 'coordinatorToken')
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

    public createPositionCashOut(input: CreateWormPositionCashOutInput): AbortableWormTradingPromise<WormPositionCashOutOperation> {
        return rawSameOriginRequest('POST', '/api/v1/worm-trading/position-cash-outs', {...input}, 'Worm position Cash Out could not be prepared', body =>
            normalizePositionCashOutOperation(body, {walletId: input.walletId, positionPubkey: input.positionPubkey})
        );
    }

    public getPositionCashOut(id: string): AbortableWormTradingPromise<WormPositionCashOutOperation> {
        const expectedID = id.trim();
        return rawSameOriginRequest(
            'GET',
            `/api/v1/worm-trading/position-cash-outs/${encodeURIComponent(expectedID)}`,
            undefined,
            'Worm position Cash Out status could not be loaded',
            body => normalizePositionCashOutOperation(body, {id: expectedID})
        );
    }

    public reconcilePositionCashOut(id: string, command: WormPositionCashOutCommandInput): AbortableWormTradingPromise<WormPositionCashOutOperation> {
        const expectedID = id.trim();
        return rawSameOriginRequest(
            'POST',
            `/api/v1/worm-trading/position-cash-outs/${encodeURIComponent(expectedID)}:reconcile`,
            {...command},
            'Worm position Cash Out status could not be reconciled',
            body => normalizePositionCashOutOperation(body, {id: expectedID})
        );
    }

    public googlePositionCashOutAuthorizationURL(id: string, command: WormPositionCashOutCommandInput, returnTo: string): string {
        const query = new URLSearchParams({
            cashOutId: id,
            commandId: command.commandId,
            expectedRevision: String(command.expectedRevision),
            returnTo,
            [APPLICATION_REALM_QUERY]: 'member'
        });
        return `${requests.toAbsURL('/auth/worm-trading/position-cash-outs/google')}?${query.toString()}`;
    }

    public createSolanaPositionCashOutAuthorizationChallenge(
        id: string,
        command: WormPositionCashOutCommandInput
    ): AbortableWormTradingPromise<WormPositionCashOutAuthorizationChallenge> {
        return rawReauthenticationPost(`/auth/worm-trading/position-cash-outs/${encodeURIComponent(id)}/solana/challenge`, {...command}, body => ({
            message: requireExactString(body, 'message'),
            expiresAt: requireInteger(body, 1, 'expiresAt')
        }));
    }

    public verifySolanaPositionCashOutAuthorization(id: string, signature: string): AbortableWormTradingPromise<WormPositionCashOutOperation> {
        const expectedID = id.trim();
        return rawReauthenticationPost(`/auth/worm-trading/position-cash-outs/${encodeURIComponent(expectedID)}/solana/verify`, {signature}, body =>
            normalizePositionCashOutOperation(body, {id: expectedID})
        );
    }

    public authorizeDevelopmentPositionCashOut(id: string, command: WormPositionCashOutCommandInput): AbortableWormTradingPromise<WormPositionCashOutOperation> {
        const expectedID = id.trim();
        return rawReauthenticationPost(`/auth/worm-trading/position-cash-outs/${encodeURIComponent(expectedID)}/development`, {...command}, body =>
            normalizePositionCashOutOperation(body, {id: expectedID})
        );
    }

    public createPositionCashOutBatch(input: CreateWormPositionCashOutBatchInput): AbortableWormTradingPromise<WormPositionCashOutBatch> {
        if (
            !canonicalWormTradingUUIDPattern.test(input.commandId) ||
            input.walletIds.length < 1 ||
            input.walletIds.length > 100 ||
            input.walletIds.some(walletId => !Number.isSafeInteger(walletId) || walletId < 1) ||
            new Set(input.walletIds).size !== input.walletIds.length
        ) {
            throw new Error('A Cash Out batch requires between 1 and 100 unique wallets.');
        }
        return rawSameOriginRequest(
            'POST',
            '/api/v1/worm-trading/position-cash-out-batches',
            {commandId: input.commandId, walletIds: [...input.walletIds]},
            'Worm position Cash Out batch could not be prepared',
            normalizePositionCashOutBatch
        );
    }

    public getActivePositionCashOutBatch(): AbortableWormTradingPromise<WormPositionCashOutBatch | undefined> {
        return rawOptionalSameOriginGet(
            '/api/v1/worm-trading/position-cash-out-batches/active',
            'Active Worm position Cash Out batch could not be loaded',
            normalizePositionCashOutBatch
        );
    }

    public getPositionCashOutBatch(id: string): AbortableWormTradingPromise<WormPositionCashOutBatch> {
        const expectedID = id.trim();
        return rawSameOriginRequest(
            'GET',
            `/api/v1/worm-trading/position-cash-out-batches/${encodeURIComponent(expectedID)}`,
            undefined,
            'Worm position Cash Out batch status could not be loaded',
            body => normalizePositionCashOutBatch(body, expectedID)
        );
    }

    public listPositionCashOutBatchItems(id: string, page = 1, pageSize = 20): AbortableWormTradingPromise<ListWormPositionCashOutBatchItemsResult> {
        const expectedID = id.trim();
        if (!canonicalWormTradingUUIDPattern.test(expectedID) || !Number.isSafeInteger(page) || page < 1 || !Number.isSafeInteger(pageSize) || pageSize < 1 || pageSize > 100) {
            throw new Error('Invalid Cash Out batch item page.');
        }
        const query = new URLSearchParams({page: String(page), pageSize: String(pageSize)});
        return rawSameOriginRequest(
            'GET',
            `/api/v1/worm-trading/position-cash-out-batches/${encodeURIComponent(expectedID)}/items?${query.toString()}`,
            undefined,
            'Worm position Cash Out batch items could not be loaded',
            body => normalizePositionCashOutBatchItems(body, page, pageSize)
        );
    }

    public commandPositionCashOutBatch(
        id: string,
        action: 'cancel' | 'pause' | 'continue' | 'terminate' | 'check-status',
        command: WormPositionCashOutBatchCommandInput
    ): AbortableWormTradingPromise<WormPositionCashOutBatch> {
        const expectedID = id.trim();
        if (
            !canonicalWormTradingUUIDPattern.test(expectedID) ||
            !canonicalWormTradingUUIDPattern.test(command.commandId) ||
            !Number.isSafeInteger(command.expectedRevision) ||
            command.expectedRevision < 1
        ) {
            throw new Error('Invalid Cash Out batch command.');
        }
        return rawSameOriginRequest(
            'POST',
            `/api/v1/worm-trading/position-cash-out-batches/${encodeURIComponent(expectedID)}:${action}`,
            {...command},
            'Worm position Cash Out batch command could not be applied',
            body => normalizePositionCashOutBatch(body, expectedID)
        );
    }

    public googlePositionCashOutBatchAuthorizationURL(id: string, command: WormPositionCashOutBatchCommandInput, returnTo: string): string {
        const query = new URLSearchParams({
            batchId: id,
            commandId: command.commandId,
            expectedRevision: String(command.expectedRevision),
            returnTo,
            [APPLICATION_REALM_QUERY]: 'member'
        });
        return `${requests.toAbsURL('/auth/worm-trading/position-cash-out-batches/google')}?${query.toString()}`;
    }

    public createSolanaPositionCashOutBatchAuthorizationChallenge(
        id: string,
        command: WormPositionCashOutBatchCommandInput
    ): AbortableWormTradingPromise<WormPositionCashOutBatchAuthorizationChallenge> {
        return rawReauthenticationPost(`/auth/worm-trading/position-cash-out-batches/${encodeURIComponent(id)}/solana/challenge`, {...command}, body => ({
            message: requireExactString(body, 'message'),
            expiresAt: requireInteger(body, 1, 'expiresAt')
        }));
    }

    public verifySolanaPositionCashOutBatchAuthorization(id: string, signature: string): AbortableWormTradingPromise<WormPositionCashOutBatch> {
        const expectedID = id.trim();
        return rawReauthenticationPost(`/auth/worm-trading/position-cash-out-batches/${encodeURIComponent(expectedID)}/solana/verify`, {signature}, body =>
            normalizePositionCashOutBatch(body, expectedID)
        );
    }

    public authorizeDevelopmentPositionCashOutBatch(id: string, command: WormPositionCashOutBatchCommandInput): AbortableWormTradingPromise<WormPositionCashOutBatch> {
        const expectedID = id.trim();
        return rawReauthenticationPost(`/auth/worm-trading/position-cash-out-batches/${encodeURIComponent(expectedID)}/development`, {...command}, body =>
            normalizePositionCashOutBatch(body, expectedID)
        );
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

    public regenerateWallet(walletId: number): AbortableWormTradingPromise<WormWalletConnection> {
        return rawSameOriginRequest(
            'POST',
            `/api/v1/worm-trading/wallet-connections/${encodeURIComponent(String(walletId))}:regenerate`,
            {acknowledgeUnknownCredentialMayRemain: true},
            'Worm wallet credential regeneration failed',
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
        const query = new URLSearchParams({returnTo, [APPLICATION_REALM_QUERY]: 'member'});
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

    public createExecutionPlan(input: CreateWormExecutionPlanInput): AbortableWormTradingPromise<WormExecutionPlan> {
        const request = requests.post('/worm-trading/execution-plans', writeScope).send(input);
        return abortableRequest(request, body => validateCreatedExecutionPlan(normalizeExecutionPlan(requireRecord(body).plan || body), input));
    }

    public getExecutionPlan(id: string): AbortableWormTradingPromise<WormExecutionPlan> {
        const expectedPlanID = id.trim();
        const request = requests.get(`/worm-trading/execution-plans/${encodeURIComponent(expectedPlanID)}`, readScope);
        return abortableRequest(request, body => normalizeExecutionPlan(requireRecord(body).plan || body, expectedPlanID));
    }

    public listExecutionPlanSteps(id: string, page = 1, pageSize = 50): AbortableWormTradingPromise<ListWormExecutionPlanStepsResult> {
        const request = requests.get(`/worm-trading/execution-plans/${encodeURIComponent(id)}/steps`, readScope).query({page, pageSize});
        return abortableRequest(request, value => {
            const body = requireRecord(value);
            const items = readRepeatedArray(body, 'items').map(normalizeExecutionPlanStep);
            const total = optionalInteger(body, 0, 0, 'total');
            const responsePage = requireInteger(body, 1, 'page');
            const responsePageSize = requireInteger(body, 1, 'pageSize');
            const offset = (responsePage - 1) * responsePageSize;
            const expectedItemCount = Math.min(responsePageSize, Math.max(0, total - offset));
            if (
                responsePage !== page ||
                responsePageSize !== pageSize ||
                responsePageSize > 100 ||
                items.length !== expectedItemCount ||
                items.some((step, index) => step.ordinal !== offset + index + 1) ||
                (items.length > 0 && offset + items.length > total)
            ) {
                return invalidWormTradingResponse();
            }
            return {items, total, page: responsePage, pageSize: responsePageSize};
        });
    }

    public createExecutionRun(planId: string, command: WormExecutionCommandInput): AbortableWormTradingPromise<WormExecutionRun> {
        const expectedPlanID = planId.trim();
        return rawSameOriginRequest(
            'POST',
            '/api/v1/worm-trading/executions',
            {planId, commandId: command.commandId, expectedRevision: command.expectedRevision},
            'Worm execution could not be prepared',
            body => {
                const run = normalizeExecutionRun(body.run || body);
                return run.planId === expectedPlanID ? run : invalidWormTradingResponse();
            }
        );
    }

    public listExecutionRuns(page = 1, pageSize = 20): AbortableWormTradingPromise<ListWormExecutionRunsResult> {
        const query = new URLSearchParams({page: String(page), pageSize: String(pageSize)});
        return rawSameOriginRequest('GET', `/api/v1/worm-trading/executions?${query.toString()}`, undefined, 'Worm executions could not be loaded', value => {
            const body = requireRecord(value);
            const items = readRepeatedArray(body, 'items').map(run => normalizeExecutionRun(run));
            const total = optionalInteger(body, 0, 0, 'total');
            const responsePage = requireInteger(body, 1, 'page');
            const responsePageSize = requireInteger(body, 1, 'pageSize');
            const offset = (responsePage - 1) * responsePageSize;
            const expectedItemCount = Math.min(responsePageSize, Math.max(0, total - offset));
            if (
                responsePage !== page ||
                responsePageSize !== pageSize ||
                responsePageSize > 100 ||
                items.length !== expectedItemCount ||
                new Set(items.map(run => run.id)).size !== items.length
            ) {
                return invalidWormTradingResponse();
            }
            return {items, total, page: responsePage, pageSize: responsePageSize};
        });
    }

    public getExecutionRun(id: string): AbortableWormTradingPromise<WormExecutionRun> {
        const expectedRunID = id.trim();
        return rawSameOriginRequest('GET', `/api/v1/worm-trading/executions/${encodeURIComponent(expectedRunID)}`, undefined, 'Worm execution could not be loaded', body =>
            normalizeExecutionRun(body.run || body, expectedRunID)
        );
    }

    public listExecutionRunSteps(id: string, page = 1, pageSize = 50): AbortableWormTradingPromise<ListWormExecutionRunStepsResult> {
        const query = new URLSearchParams({page: String(page), pageSize: String(pageSize)});
        return rawSameOriginRequest(
            'GET',
            `/api/v1/worm-trading/executions/${encodeURIComponent(id)}/steps?${query.toString()}`,
            undefined,
            'Worm execution steps could not be loaded',
            value => {
                const body = requireRecord(value);
                const items = readRepeatedArray(body, 'items').map(normalizeExecutionRunStep);
                const total = optionalInteger(body, 0, 0, 'total');
                const responsePage = requireInteger(body, 1, 'page');
                const responsePageSize = requireInteger(body, 1, 'pageSize');
                const offset = (responsePage - 1) * responsePageSize;
                const expectedItemCount = Math.min(responsePageSize, Math.max(0, total - offset));
                if (
                    responsePage !== page ||
                    responsePageSize !== pageSize ||
                    responsePageSize > 100 ||
                    items.length !== expectedItemCount ||
                    items.some((step, index) => step.ordinal !== offset + index + 1)
                ) {
                    return invalidWormTradingResponse();
                }
                return {items, total, page: responsePage, pageSize: responsePageSize};
            }
        );
    }

    public startExecutionRun(id: string, command: WormExecutionCommandInput): AbortableWormTradingPromise<WormExecutionCommandResult> {
        return this.executionRunCommand(id, 'start', command, 'Worm execution could not be started');
    }

    public pauseExecutionRun(id: string, command: WormExecutionCommandInput): AbortableWormTradingPromise<WormExecutionCommandResult> {
        return this.executionRunCommand(id, 'pause', command, 'Worm execution could not be paused');
    }

    public continueExecutionRun(id: string, command: WormExecutionCommandInput): AbortableWormTradingPromise<WormExecutionCommandResult> {
        return this.executionRunCommand(id, 'continue', command, 'Worm execution could not be continued');
    }

    public terminateExecutionRun(id: string, command: WormExecutionCommandInput): AbortableWormTradingPromise<WormExecutionCommandResult> {
        return this.executionRunCommand(id, 'terminate', command, 'Worm execution could not be terminated');
    }

    public heartbeatExecutionRun(id: string, command: WormExecutionCoordinatorCommandInput): AbortableWormTradingPromise<WormExecutionCommandResult> {
        return this.executionRunCommand(id, 'heartbeat', command, 'Worm execution coordinator heartbeat failed');
    }

    public executeNextExecutionStep(
        id: string,
        command: WormExecutionCoordinatorCommandInput & {expectedStepOrdinal: number}
    ): AbortableWormTradingPromise<WormExecutionCommandResult> {
        return this.executionRunCommand(id, 'execute-next', command, 'The next Worm execution step could not be started');
    }

    public reconcileExecutionStep(runId: string, stepId: string, command: WormExecutionCommandInput): AbortableWormTradingPromise<WormExecutionCommandResult> {
        return rawSameOriginRequest(
            'POST',
            `/api/v1/worm-trading/executions/${encodeURIComponent(runId)}/steps/${encodeURIComponent(stepId)}:reconcile`,
            {...command},
            'Worm execution reconciliation could not be started',
            body => normalizeExecutionCommandResult(body, runId)
        );
    }

    public googleExecutionAuthorizationURL(runId: string, command: WormExecutionCommandInput, returnTo: string): string {
        const query = new URLSearchParams({
            runId,
            commandId: command.commandId,
            expectedRevision: String(command.expectedRevision),
            returnTo,
            [APPLICATION_REALM_QUERY]: 'member'
        });
        return `${requests.toAbsURL('/auth/worm-trading/executions/google')}?${query.toString()}`;
    }

    public createSolanaExecutionAuthorizationChallenge(runId: string, command: WormExecutionCommandInput): AbortableWormTradingPromise<WormExecutionAuthorizationChallenge> {
        return rawReauthenticationPost(`/auth/worm-trading/executions/${encodeURIComponent(runId)}/solana/challenge`, {...command}, body => ({
            message: requireExactString(body, 'message'),
            expiresAt: requireInteger(body, 1, 'expiresAt')
        }));
    }

    public verifySolanaExecutionAuthorization(runId: string, signature: string): AbortableWormTradingPromise<WormExecutionRun> {
        return rawReauthenticationPost(`/auth/worm-trading/executions/${encodeURIComponent(runId)}/solana/verify`, {signature}, body =>
            normalizeExecutionRun(body.run || body, runId)
        );
    }

    public authorizeDevelopmentExecutionRun(runId: string, command: WormExecutionCommandInput): AbortableWormTradingPromise<WormExecutionRun> {
        return rawReauthenticationPost(`/auth/worm-trading/executions/${encodeURIComponent(runId)}/development`, {...command}, body =>
            normalizeExecutionRun(body.run || body, runId)
        );
    }

    private executionRunCommand(
        id: string,
        action: 'start' | 'pause' | 'continue' | 'terminate' | 'heartbeat' | 'execute-next',
        body: WormExecutionCommandInput | WormExecutionCoordinatorCommandInput | (WormExecutionCoordinatorCommandInput & {expectedStepOrdinal: number}),
        fallbackError: string
    ): AbortableWormTradingPromise<WormExecutionCommandResult> {
        return rawSameOriginRequest('POST', `/api/v1/worm-trading/executions/${encodeURIComponent(id)}:${action}`, {...body}, fallbackError, response =>
            normalizeExecutionCommandResult(response, id)
        );
    }
}
