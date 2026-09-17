import requests from '../shared/services/requests';
import {normalizeSubscriptionSummary, normalizeSummaryPage, normalizeRuntimeStatus} from './trader-sync-models';
import {AdminTraderSyncService} from './trader-sync-service';

const summary = {
    subscriptionId: 'sub-2',
    accountId: 'owner',
    username: 'alice',
    email: 'alice@example.test',
    wallet: '0x123',
    status: 'healthy',
    createdAt: '2026-09-10T01:00:00Z',
    updatedAt: '2026-09-10T02:00:00Z',
    observation: {
        state: 'healthy',
        reason: '',
        interruptionCount: '9007199254740993',
        latestInterruption: {reason: 'disconnect', uncertainty: 'unknown_start', possibleMissing: true}
    },
    activityCount: '9007199254740995',
    associatedDeliveryCounts: {total: '7', pending: '1', sending: '1', sent: '2', failed: '1', unknown: '1', cancelled: '1'},
    asOf: '2026-09-10T03:00:00Z'
};
afterEach(() => jest.restoreAllMocks());
const response = (body: unknown) => {
    const request = Object.assign(Promise.resolve({body}), {abort: jest.fn(), query: jest.fn()});
    request.query.mockReturnValue(request);
    return request;
};
test('admin summary only selects public safe fields recursively and preserves exact counts and missing facts', () => {
    const result = normalizeSubscriptionSummary({
        ...summary,
        note: 'PRIVATE',
        activities: ['PRIVATE'],
        payload: 'PRIVATE',
        observation: {...summary.observation, note: 'PRIVATE', latestInterruption: {...summary.observation.latestInterruption, payload: 'PRIVATE'}},
        associatedDeliveryCounts: {...summary.associatedDeliveryCounts, deliveries: ['PRIVATE']}
    });
    expect(result).toEqual(summary);
    expect(result.pausedAt).toBeUndefined();
    expect(result.observation.lastReliableAt).toBeUndefined();
    expect(result.observation.latestInterruption.start).toBeUndefined();
});
test('summary page retains server order, next cursor and asOf without inventing totals', () => {
    expect(
        normalizeSummaryPage({summaries: [summary, {...summary, subscriptionId: 'sub-1'}], page: {nextCursor: 'opaque', total: 900}, asOf: summary.asOf, payload: 'PRIVATE'})
    ).toEqual({summaries: [summary, {...summary, subscriptionId: 'sub-1'}], page: {nextCursor: 'opaque'}, asOf: summary.asOf});
});
test('runtime keeps gauge/window/epoch provenance and absent raw observations without fabricated zero', () => {
    const runtime = {
        collectorConnected: false,
        collectorEpoch: '9007199254740993',
        filterRevision: '3',
        asOf: summary.asOf,
        metrics: [
            {name: 'raw_queue_depth', unit: 'raw_logs', kind: 'gauge'},
            {name: 'sent', value: '9007199254740999', unit: 'deliveries', kind: 'epoch', serviceEpoch: 'opaque-instance'},
            {name: 'recent', value: '0', unit: 'activities', kind: 'window', windowStart: summary.createdAt, windowEnd: summary.updatedAt}
        ]
    };
    expect(normalizeRuntimeStatus({...runtime, payload: 'PRIVATE', metrics: runtime.metrics.map(metric => ({...metric, note: 'PRIVATE'}))})).toEqual(runtime);
    expect(() => normalizeRuntimeStatus({...runtime, metrics: [{kind: 'counter'}]})).toThrow();
    expect(() => normalizeSubscriptionSummary({...summary, status: 'monitoring'})).toThrow();
});
test('admin list sends explicit read feature and gateway filters and forwards abort', async () => {
    const request = response({summaries: [summary], page: {}, asOf: summary.asOf});
    const get = jest.spyOn(requests, 'get').mockReturnValue(request as any);
    const pending = new AdminTraderSyncService().listSubscriptionSummaries({
        wallet: '0x123',
        includeCancelled: true,
        pageSize: 25,
        cursor: 'opaque',
        accountId: 'filter-owner',
        state: 'interrupted'
    });
    expect(await pending).toEqual({summaries: [summary], page: {}, asOf: summary.asOf});
    expect(get).toHaveBeenCalledWith('/admin/trader-sync/subscriptions', {feature: 'admin-trader-sync', mode: 'read'});
    expect(request.query).toHaveBeenCalledWith({
        'wallet': '0x123',
        'include_cancelled': true,
        'page.page_size': 25,
        'page.cursor': 'opaque',
        'account_id': 'filter-owner',
        'state': 'interrupted'
    });
    pending.abort();
    expect(request.abort).toHaveBeenCalledTimes(1);
});
test('detail and runtime unwrap their distinct gateway envelopes and retain abort', async () => {
    const detail = response({summary}),
        runtime = response({status: {collectorConnected: false, collectorEpoch: '1', filterRevision: '2', metrics: [], asOf: summary.asOf}});
    const get = jest
        .spyOn(requests, 'get')
        .mockReturnValueOnce(detail as any)
        .mockReturnValueOnce(runtime as any);
    const service = new AdminTraderSyncService();
    const first = service.getSubscriptionSummary('a/b');
    expect(await first).toEqual(summary);
    first.abort();
    const second = service.getRuntimeStatus();
    expect(await second).toMatchObject({collectorConnected: false, metrics: []});
    second.abort();
    expect(get.mock.calls).toEqual([
        ['/admin/trader-sync/subscriptions/a%2Fb', {feature: 'admin-trader-sync', mode: 'read'}],
        ['/admin/trader-sync/status', {feature: 'admin-service-status', mode: 'read'}]
    ]);
    expect(detail.abort).toHaveBeenCalledTimes(1);
    expect(runtime.abort).toHaveBeenCalledTimes(1);
});
test('frozen Task13 runtime gateway response retains its original exact values and service epochs', async () => {
    const body = require('./__fixtures__/trader-sync/tradersync-fix1-ordinary.json');
    jest.spyOn(requests, 'get').mockReturnValue(response(body) as any);
    const result = await new AdminTraderSyncService().getRuntimeStatus();
    expect(result).toEqual(body.status);
    expect(result.metrics.filter(metric => metric.kind === 'epoch')).toHaveLength(10);
    expect(result.metrics.find(metric => metric.name === 'raw_observation_available')?.value).toBe('1');
});
