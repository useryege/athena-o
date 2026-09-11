// Public camelCase JSON contract: application/v1alpha1/trader_sync_types.go.
// No numeric coercion: IDs, revisions, counts and amounts retain their wire strings.
export type Availability = 'available' | 'unavailable';
export type SubscriptionStatus = 'pending_baseline' | 'healthy' | 'interrupted' | 'paused' | 'permission_disabled' | 'cancelled';
export type ObservationState = 'pending_baseline' | 'healthy' | 'interrupted';
export type DeliveryStatus = 'pending' | 'sending' | 'sent' | 'failed' | 'unknown' | 'cancelled';
export type AttemptStatus = 'sending' | 'sent' | 'retryable' | 'failed' | 'unknown';
export type SummaryPhase = 'waiting' | 'frozen' | 'cancelled_before_freeze';
export type PnLPeriod = '1D' | '1W' | '1M' | '1Y' | 'YTD' | 'ALL';

export interface FieldEvidence {
    availability: Availability;
    reasonCode: string;
    source: string;
    queriedAt: string;
}

export interface StringField {
    evidence: FieldEvidence;
    value?: string;
}

export interface DecimalField {
    evidence: FieldEvidence;
    value?: string;
}

export interface BoolField {
    evidence: FieldEvidence;
    value?: boolean;
}

export interface TimeField {
    evidence: FieldEvidence;
    value?: string;
}

export interface CurvePoint {
    t: string;
    p: string;
}

export interface Curve {
    evidence: FieldEvidence;
    points: CurvePoint[];
}

export interface PnLView {
    period: PnLPeriod;
    amount: DecimalField;
    curve: Curve;
    interval: string;
    fidelity: string;
    referenceTime: TimeField;
    timezone: StringField;
}

export interface ResolvedTarget {
    wallet: string;
    canonicalProfileURL: string;
    avatar: StringField;
    displayName: StringField;
    verified: BoolField;
    joinedAt: TimeField;
    positionValue: DecimalField;
    largestWin: DecimalField;
    predictions: DecimalField;
    pnl: PnLView[];
    defaultPeriod: PnLPeriod;
    confirmationToken: string;
    expiresAt: string;
    usageNotice: string;
    savedNote?: TargetNote;
    existingSubscription?: ExistingSubscription;
    quota: Quota;
}

export interface TargetNote {
    wallet: string;
    note: string;
    revision: string;
}

export interface Quota {
    used: number;
    limit: number;
}

export interface ExistingSubscription {
    id: string;
    status: SubscriptionStatus;
    revision: string;
}

export interface TargetDisplay {
    displayName: StringField;
    avatar: StringField;
    profileURL: StringField;
}

export interface Subscription {
    id: string;
    wallet: string;
    status: SubscriptionStatus;
    revision: string;
    generation: string;
    note: string;
    noteRevision: string;
    createdAt: string;
    updatedAt: string;
    pausedAt?: string;
    cancelledAt?: string;
    permissionDisabledAt?: string;
    currentInterval?: Interval;
    observation: Observation;
    bindingStatus: 'connected' | 'unreachable' | 'unbound';
    queueNotice: string;
    queueCounts: StatusCounts;
    targetDisplay: TargetDisplay;
}

export interface Interval {
    effectiveAt: string;
    endedAt?: string;
    generation: string;
    epoch: string;
}

export interface Observation {
    state: ObservationState;
    reason: string;
    lastReliableAt?: string;
    latestInterruption?: Interruption;
    interruptionCount: string;
}

export interface Interruption {
    start?: string;
    end?: string;
    recoveredAt?: string;
    reason: string;
    uncertainty: string;
    possibleMissing: boolean;
}

export interface HistoryEntry {
    id: string;
    kind: 'interval' | 'interruption';
    sortAt: string;
    interval?: Interval;
    interruption?: Interruption;
}

export interface Activity {
    id: string;
    subscriptionId: string;
    sourceRecordId: string;
    wallet: string;
    side: 'BUY' | 'SELL';
    positionId: string;
    collateralRaw: string;
    sharesRaw: string;
    feeRaw: string;
    collateralSymbol: string;
    collateralDecimals: number;
    sharesDecimals: number;
    priceNumerator: string;
    priceDenominator: string;
    priceEvidence: FieldEvidence;
    sourceVersion: string;
    settledAt: string;
    receivedAt: string;
    recordedAt: string;
    publicTimeEvidence: FieldEvidence;
    metadata: TradeMetadata;
    noteSnapshot: string;
    notificationMode: 'in_app_only' | 'ordinary' | 'summary';
    notificationReason: string;
    delivery?: Delivery;
    summaryProgress?: SummaryProgress;
    targetDisplaySnapshot: TargetDisplay;
    finalityAnomaly?: FinalityAnomaly;
    sourceLocation: SourceLocation;
}

export interface SourceLocation {
    chainId: string;
    exchangeAddress: string;
    transactionHash: string;
    blockHash: string;
    blockNumber: string;
    logIndex: string;
}

export interface FinalityAnomaly {
    reason: string;
    detectedAt: string;
    publishedBlockHash: string;
    conflictingBlockHash?: string;
}

export interface MarketRef {
    evidence: FieldEvidence;
    id: string;
    title: string;
    url: string;
    conditionId: string;
    positionId: string;
    outcome: string;
}

export interface ComboLeg {
    positionId: string;
    market: MarketRef;
}

export interface TradeMetadata {
    market: MarketRef;
    legsEvidence: FieldEvidence;
    legs: ComboLeg[];
    relationship: '' | 'AND(legs)' | 'NOT(AND(legs))';
}

export interface Delivery {
    id: string;
    status: DeliveryStatus;
    reason: string;
    authorizedAt?: string;
    startedAt?: string;
    resultAt?: string;
    messageId?: string;
    attemptCount: string;
    latestAttempt?: Attempt;
}

export interface Attempt {
    index: string;
    authorizedAt: string;
    startedAt?: string;
    resultAt?: string;
    status: AttemptStatus;
    reason: string;
}

export interface StatusCounts {
    total: string;
    pending: string;
    sending: string;
    sent: string;
    failed: string;
    unknown: string;
    cancelled: string;
}

export interface SummaryProgress {
    phase: SummaryPhase;
    reason: string;
    batchId?: string;
    relatedPartCounts: StatusCounts;
    batchPartCounts: StatusCounts;
    oldestAt: string;
    firstStartedAt?: string;
}

export interface TargetCount {
    wallet: string;
    count: string;
}

export interface SummaryBatch {
    id: string;
    oldestAt: string;
    settledFrom: string;
    settledTo: string;
    recordedFrom: string;
    recordedTo: string;
    firstStartedAt?: string;
    activityCount: string;
    targetCounts: TargetCount[];
    partCounts: StatusCounts;
    asOf: string;
}

export interface SummaryPart {
    id: string;
    index: number;
    total: number;
    delivery: Delivery;
    associatedActivityCount: string;
}

export interface PageInfo {
    nextCursor?: string;
}
export interface ActivityPageInfo extends PageInfo {
    refreshCursor?: string;
    snapshot?: string;
    asOf?: string;
    hasNewer: boolean;
}
export interface SubscriptionPage {
    subscriptions: Subscription[];
    page: PageInfo;
    quota: Quota;
    asOf: string;
}
export interface HistoryPage {
    entries: HistoryEntry[];
    page: PageInfo;
    asOf: string;
}
export interface ActivityPage {
    activities: Activity[];
    page: ActivityPageInfo;
}
export interface PartPage {
    parts: SummaryPart[];
    page: PageInfo;
    asOf: string;
}

type Decoder<T> = (value: unknown, path?: string) => T;
const invalid = (path: string): never => {
    throw new Error(`Trader Sync protocol error at ${path}`);
};
const string: Decoder<string> = (value, path = 'value') => (typeof value === 'string' ? value : invalid(path));
const boolean: Decoder<boolean> = (value, path = 'value') => (typeof value === 'boolean' ? value : invalid(path));
const pattern =
    (expression: RegExp): Decoder<string> =>
    (value, path = 'value') => {
        const text = string(value, path);
        return expression.test(text) ? text : invalid(path);
    };
const uint = pattern(/^(0|[1-9]\d*)$/);
const positive = pattern(/^[1-9]\d*$/);
const signed = pattern(/^-?(0|[1-9]\d*)$/);
const decimal = pattern(/^-?\d+(?:\.\d+)?$/);
const uuid = pattern(/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i);
const wallet = pattern(/^0x[0-9a-f]{40}$/i);
const hash = pattern(/^0x[0-9a-f]{64}$/i);
const nonempty = pattern(/\S/);
const int32: Decoder<number> = (value, path = 'value') => (typeof value === 'number' && Number.isInteger(value) && value >= 0 && value <= 2147483647 ? value : invalid(path));
const positiveInt32: Decoder<number> = (value, path = 'value') => {
    const result = int32(value, path);
    return result > 0 ? result : invalid(path);
};
const choice =
    <const V extends string[]>(...values: V): Decoder<V[number]> =>
    (value, path = 'value') => {
        const text = string(value, path);
        return values.includes(text) ? text : invalid(path);
    };
const optional =
    <T>(read: Decoder<T>): Decoder<T | undefined> =>
    (value, path) =>
        value === undefined ? undefined : read(value, path);
const array =
    <T>(read: Decoder<T>, empty: 'null' | 'omitted' | 'neither' = 'neither'): Decoder<T[]> =>
    (value, path = 'value') => {
        if ((empty === 'null' && value === null) || (empty === 'omitted' && value === undefined)) return [];
        if (!Array.isArray(value)) return invalid(path);
        return value.map((item, index) => read(item, `${path}[${index}]`));
    };
const object =
    <T>(shape: {[K in keyof T]-?: Decoder<T[K]>}): Decoder<T> =>
    (value, path = 'response') => {
        if (typeof value !== 'object' || value === null || Array.isArray(value)) return invalid(path);
        const source = value as Record<string, unknown>;
        const result: Record<string, unknown> = {};
        for (const key of Object.keys(shape) as Array<keyof T & string>) {
            const field = shape[key](source[key], `${path}.${key}`);
            if (field !== undefined) result[key] = field;
        }
        return result as T;
    };
const evidence = object<FieldEvidence>({availability: choice('available', 'unavailable'), reasonCode: string, source: string, queriedAt: string});
const stringField = object<StringField>({evidence, value: optional(string)});
const decimalField = object<DecimalField>({evidence, value: optional(decimal)});
const boolField = object<BoolField>({evidence, value: optional(boolean)});
const timeField = object<TimeField>({evidence, value: optional(string)});
const curve = object<Curve>({evidence, points: array(object<CurvePoint>({t: signed, p: decimal}), 'null')});
const periods = ['1D', '1W', '1M', '1Y', 'YTD', 'ALL'] as const;
const pnlView = object<PnLView>({period: choice(...periods), amount: decimalField, curve, interval: string, fidelity: string, referenceTime: timeField, timezone: stringField});
const pnl: Decoder<PnLView[]> = (value, path = 'pnl') => {
    const views = array(pnlView)(value, path);
    if (views.length !== periods.length || new Set(views.map(view => view.period)).size !== periods.length) return invalid(path);
    return periods.map(period => views.find(view => view.period === period)!);
};
const status = choice('pending_baseline', 'healthy', 'interrupted', 'paused', 'permission_disabled', 'cancelled');
const quota = object<Quota>({used: int32, limit: int32});
export const normalizeTargetNote = object<TargetNote>({wallet, note: string, revision: uint});
const existing = object<ExistingSubscription>({id: uuid, status, revision: uint});
const display = object<TargetDisplay>({displayName: stringField, avatar: stringField, profileURL: stringField});
export const normalizeResolvedTarget = object<ResolvedTarget>({
    wallet,
    canonicalProfileURL: string,
    avatar: stringField,
    displayName: stringField,
    verified: boolField,
    joinedAt: timeField,
    positionValue: decimalField,
    largestWin: decimalField,
    predictions: decimalField,
    pnl,
    defaultPeriod: choice(...periods),
    confirmationToken: nonempty,
    expiresAt: string,
    usageNotice: string,
    savedNote: optional(normalizeTargetNote),
    existingSubscription: optional(existing),
    quota
});
const counts = object<StatusCounts>({total: uint, pending: uint, sending: uint, sent: uint, failed: uint, unknown: uint, cancelled: uint});
const interruption = object<Interruption>({
    start: optional(string),
    end: optional(string),
    recoveredAt: optional(string),
    reason: string,
    uncertainty: string,
    possibleMissing: boolean
});
const observation = object<Observation>({
    state: choice('pending_baseline', 'healthy', 'interrupted'),
    reason: string,
    lastReliableAt: optional(string),
    latestInterruption: optional(interruption),
    interruptionCount: uint
});
const interval = object<Interval>({effectiveAt: string, endedAt: optional(string), generation: uint, epoch: uint});
export const normalizeSubscription = object<Subscription>({
    id: uuid,
    wallet,
    status,
    revision: uint,
    generation: uint,
    note: string,
    noteRevision: uint,
    createdAt: string,
    updatedAt: string,
    pausedAt: optional(string),
    cancelledAt: optional(string),
    permissionDisabledAt: optional(string),
    currentInterval: optional(interval),
    observation,
    bindingStatus: choice('connected', 'unreachable', 'unbound'),
    queueNotice: string,
    queueCounts: counts,
    targetDisplay: display
});
const historyShape = object<HistoryEntry>({
    id: nonempty,
    kind: choice('interval', 'interruption'),
    sortAt: string,
    interval: optional(interval),
    interruption: optional(interruption)
});
const history: Decoder<HistoryEntry> = (value, path) => {
    const entry = historyShape(value, path);
    const prefix = `${entry.kind}/`;
    if (!entry.id.startsWith(prefix)) return invalid(path || 'history.id');
    (entry.kind === 'interval' ? uuid : positive)(entry.id.slice(prefix.length), `${path || 'history'}.id`);
    if (entry.kind === 'interval' ? !entry.interval : !entry.interruption) return invalid(path || 'history');
    return entry;
};
const attempt = object<Attempt>({
    index: positive,
    authorizedAt: string,
    startedAt: optional(string),
    resultAt: optional(string),
    status: choice('sending', 'sent', 'retryable', 'failed', 'unknown'),
    reason: string
});
const delivery = object<Delivery>({
    id: positive,
    status: choice('pending', 'sending', 'sent', 'failed', 'unknown', 'cancelled'),
    reason: string,
    authorizedAt: optional(string),
    startedAt: optional(string),
    resultAt: optional(string),
    messageId: optional(string),
    attemptCount: uint,
    latestAttempt: optional(attempt)
});
const market = object<MarketRef>({evidence, id: string, title: string, url: string, conditionId: string, positionId: string, outcome: string});
const metadata = object<TradeMetadata>({
    market,
    legsEvidence: evidence,
    legs: array(object<ComboLeg>({positionId: uint, market}), 'null'),
    relationship: choice('', 'AND(legs)', 'NOT(AND(legs))')
});
const summary = object<SummaryProgress>({
    phase: choice('waiting', 'frozen', 'cancelled_before_freeze'),
    reason: string,
    batchId: optional(positive),
    relatedPartCounts: counts,
    batchPartCounts: counts,
    oldestAt: string,
    firstStartedAt: optional(string)
});
const anomaly = object<FinalityAnomaly>({reason: nonempty, detectedAt: string, publishedBlockHash: hash, conflictingBlockHash: optional(hash)});
const location = object<SourceLocation>({chainId: uint, exchangeAddress: wallet, transactionHash: hash, blockHash: hash, blockNumber: uint, logIndex: uint});
export const normalizeActivity = object<Activity>({
    id: positive,
    subscriptionId: uuid,
    sourceRecordId: positive,
    wallet,
    side: choice('BUY', 'SELL'),
    positionId: uint,
    collateralRaw: uint,
    sharesRaw: uint,
    feeRaw: uint,
    collateralSymbol: nonempty,
    collateralDecimals: int32,
    sharesDecimals: int32,
    priceNumerator: signed,
    priceDenominator: signed,
    priceEvidence: evidence,
    sourceVersion: nonempty,
    settledAt: string,
    receivedAt: string,
    recordedAt: string,
    publicTimeEvidence: evidence,
    metadata,
    noteSnapshot: string,
    notificationMode: choice('in_app_only', 'ordinary', 'summary'),
    notificationReason: string,
    delivery: optional(delivery),
    summaryProgress: optional(summary),
    targetDisplaySnapshot: display,
    finalityAnomaly: optional(anomaly),
    sourceLocation: location
});
export const normalizeSummaryBatch = object<SummaryBatch>({
    id: positive,
    oldestAt: string,
    settledFrom: string,
    settledTo: string,
    recordedFrom: string,
    recordedTo: string,
    firstStartedAt: optional(string),
    activityCount: uint,
    targetCounts: array(object<TargetCount>({wallet, count: uint}), 'null'),
    partCounts: counts,
    asOf: string
});
const part = object<SummaryPart>({id: positive, index: positiveInt32, total: positiveInt32, delivery, associatedActivityCount: uint});
const page = object<PageInfo>({nextCursor: optional(string)});
const activityPage = object<ActivityPageInfo>({
    nextCursor: optional(string),
    refreshCursor: optional(string),
    snapshot: optional(string),
    asOf: optional(string),
    hasNewer: boolean
});
// Go service repeated fields use omitempty: an absent list is the legal empty page.
export const normalizeSubscriptionPage = object<SubscriptionPage>({subscriptions: array(normalizeSubscription, 'omitted'), page, quota, asOf: string});
export const normalizeHistoryPage = object<HistoryPage>({entries: array(history, 'omitted'), page, asOf: string});
export const normalizeActivityPage = object<ActivityPage>({activities: array(normalizeActivity, 'omitted'), page: activityPage});
export const normalizePartPage = object<PartPage>({parts: array(part, 'omitted'), page, asOf: string});
