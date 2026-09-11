/** Administrator-only projections. Never retain arbitrary gateway fields. */
export type SubscriptionState = 'pending_baseline' | 'healthy' | 'interrupted' | 'paused' | 'permission_disabled' | 'cancelled';
export interface Counts {
    total?: string;
    pending?: string;
    sending?: string;
    sent?: string;
    failed?: string;
    unknown?: string;
    cancelled?: string;
}
export interface SubscriptionSummary {
    subscriptionId: string;
    accountId: string;
    username: string;
    email: string;
    wallet: string;
    status: SubscriptionState;
    createdAt: string;
    updatedAt: string;
    pausedAt?: string;
    cancelledAt?: string;
    permissionDisabledAt?: string;
    observation: {
        state: string;
        reason: string;
        lastReliableAt?: string;
        interruptionCount?: string;
        latestInterruption?: {start?: string; end?: string; recoveredAt?: string; reason: string; uncertainty: string; possibleMissing: boolean};
    };
    activityCount?: string;
    associatedDeliveryCounts: Counts;
    asOf: string;
}
export interface SummaryPage {
    summaries: SubscriptionSummary[];
    page: {nextCursor?: string};
    asOf: string;
}
export interface RuntimeMetric {
    name: string;
    value?: string;
    unit: string;
    kind: 'gauge' | 'window' | 'epoch';
    windowStart?: string;
    windowEnd?: string;
    serviceEpoch?: string;
}
export interface RuntimeStatus {
    collectorConnected?: boolean;
    collectorEpoch: string;
    filterRevision: string;
    metrics: RuntimeMetric[];
    asOf: string;
}
const record = (value: unknown): Record<string, unknown> => (value !== null && typeof value === 'object' && !Array.isArray(value) ? (value as Record<string, unknown>) : {});
const optionalString = (value: unknown) => (typeof value === 'string' ? value : undefined);
const string = (value: unknown) => optionalString(value) ?? '';
const state = (value: unknown): SubscriptionState => {
    if (!['pending_baseline', 'healthy', 'interrupted', 'paused', 'permission_disabled', 'cancelled'].includes(string(value))) throw new Error('Invalid subscription status');
    return value as SubscriptionState;
};
export const normalizeSubscriptionSummary = (value: unknown): SubscriptionSummary => {
    const item = record(value),
        observation = record(item.observation),
        latest = record(observation.latestInterruption),
        counts = record(item.associatedDeliveryCounts);
    return {
        subscriptionId: string(item.subscriptionId),
        accountId: string(item.accountId),
        username: string(item.username),
        email: string(item.email),
        wallet: string(item.wallet),
        status: state(item.status),
        createdAt: string(item.createdAt),
        updatedAt: string(item.updatedAt),
        pausedAt: optionalString(item.pausedAt),
        cancelledAt: optionalString(item.cancelledAt),
        permissionDisabledAt: optionalString(item.permissionDisabledAt),
        observation: {
            state: string(observation.state),
            reason: string(observation.reason),
            lastReliableAt: optionalString(observation.lastReliableAt),
            interruptionCount: optionalString(observation.interruptionCount),
            latestInterruption:
                observation.latestInterruption == null
                    ? undefined
                    : {
                          start: optionalString(latest.start),
                          end: optionalString(latest.end),
                          recoveredAt: optionalString(latest.recoveredAt),
                          reason: string(latest.reason),
                          uncertainty: string(latest.uncertainty),
                          possibleMissing: latest.possibleMissing === true
                      }
        },
        activityCount: optionalString(item.activityCount),
        associatedDeliveryCounts: {
            total: optionalString(counts.total),
            pending: optionalString(counts.pending),
            sending: optionalString(counts.sending),
            sent: optionalString(counts.sent),
            failed: optionalString(counts.failed),
            unknown: optionalString(counts.unknown),
            cancelled: optionalString(counts.cancelled)
        },
        asOf: string(item.asOf)
    };
};
export const normalizeSummaryPage = (value: unknown): SummaryPage => {
    const item = record(value);
    return {
        summaries: Array.isArray(item.summaries) ? item.summaries.map(normalizeSubscriptionSummary) : [],
        page: {nextCursor: optionalString(record(item.page).nextCursor)},
        asOf: string(item.asOf)
    };
};
export const normalizeRuntimeStatus = (value: unknown): RuntimeStatus => {
    const item = record(value);
    return {
        collectorConnected: typeof item.collectorConnected === 'boolean' ? item.collectorConnected : undefined,
        collectorEpoch: string(item.collectorEpoch),
        filterRevision: string(item.filterRevision),
        asOf: string(item.asOf),
        metrics: Array.isArray(item.metrics)
            ? item.metrics.map(value => {
                  const metric = record(value);
                  if (metric.kind !== 'gauge' && metric.kind !== 'window' && metric.kind !== 'epoch') throw new Error('Invalid runtime metric kind');
                  return {
                      name: string(metric.name),
                      value: optionalString(metric.value),
                      unit: string(metric.unit),
                      kind: metric.kind,
                      windowStart: optionalString(metric.windowStart),
                      windowEnd: optionalString(metric.windowEnd),
                      serviceEpoch: optionalString(metric.serviceEpoch)
                  };
              })
            : []
    };
};
