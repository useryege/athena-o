import * as models from './trader-sync-models';
import {MemberTraderSyncService} from './trader-sync-service';
import requests from '../shared/services/requests';
import {AccountDataModule} from '../shared/models';

const fixture = (name: string): any => JSON.parse(JSON.stringify(require(`./testdata/trader-sync/${name}.json`)));
const target = () => fixture('resolve-saved-note-existing').target;
const activity = () => fixture('05').activity;
const sub = () => fixture('01').subscription;
const available = {availability: 'available', reasonCode: '', source: '', queriedAt: ''};
afterEach(() => jest.restoreAllMocks());

test('recorded resolve preserves false, zero, long count, note and six independent periods', () => {
    const value = target();
    value.pnl.reverse();
    const result = models.normalizeResolvedTarget(value);
    expect(result.verified.value).toBe(false);
    expect(result.positionValue.value).toBe('0');
    expect(result.predictions.value).toBe('9007199254740993');
    expect(result.canonicalProfileURL).toBe('');
    expect(result.savedNote).toEqual(value.savedNote);
    expect(result.existingSubscription).toEqual(value.existingSubscription);
    expect(result.pnl.map(p => p.period)).toEqual(['1D', '1W', '1M', '1Y', 'YTD', 'ALL']);
    expect(result.pnl[0].curve.points).toEqual([]);
    expect(result.pnl[0].curve.evidence.availability).toBe('unavailable');
    expect(result.pnl[0].amount.value).toBeUndefined();
});
test('synthetic mixed PnL keeps curve independent from unavailable amount and allows empty available text', () => {
    const value = target();
    value.displayName.value = '';
    delete value.savedNote;
    delete value.existingSubscription;
    value.pnl[0].amount = {evidence: available, value: '-123.000000000000001'};
    value.pnl[1].amount = {evidence: available, value: '0'};
    value.pnl[5].curve = {
        evidence: available,
        points: [
            {t: '1700000000', p: '-0.000001'},
            {t: '1700000001', p: '0'}
        ]
    };
    const result = models.normalizeResolvedTarget(value);
    expect(result.displayName.value).toBe('');
    expect(result.savedNote).toBeUndefined();
    expect(result.existingSubscription).toBeUndefined();
    expect(result.pnl[0].amount.value).toBe('-123.000000000000001');
    expect(result.pnl[1].amount.value).toBe('0');
    expect(result.pnl[5].amount.value).toBeUndefined();
    expect(result.pnl[5].curve.points).toEqual(value.pnl[5].curve.points);
});
test('recorded subscription and history preserve revision and unknown optional facts', () => {
    const result = models.normalizeSubscription(sub());
    expect(result.revision).toBe('9007199254740993');
    expect(result.currentInterval).toBeUndefined();
    expect(result.pausedAt).toBeUndefined();
    expect(result.targetDisplay.displayName.value).toBeUndefined();
    expect(models.normalizeSubscriptionPage(fixture('03'))).toMatchObject({quota: {used: 2, limit: 10}, subscriptions: [{note: '', noteRevision: '0'}]});
    expect(models.normalizeHistoryPage(fixture('12'))).toEqual(fixture('12'));
});
test.each(['pending_baseline', 'healthy', 'interrupted', 'paused', 'permission_disabled', 'cancelled'])('synthetic wire subscription state %s is retained', status => {
    expect(models.normalizeSubscription({...sub(), status}).status).toBe(status);
});
test('synthetic observation preserves unknown interruption bounds and same-generation checkpoint', () => {
    const value = sub();
    value.status = 'interrupted';
    value.observation = {
        state: 'interrupted',
        reason: 'disconnect',
        lastReliableAt: '2026-09-11T00:00:00Z',
        interruptionCount: '2',
        latestInterruption: {reason: 'disconnect', uncertainty: 'actual_start_unknown', possibleMissing: true}
    };
    value.currentInterval = fixture('12').entries[0].interval;
    const result = models.normalizeSubscription(value);
    expect(result.observation).toEqual(value.observation);
    expect(result.currentInterval).toEqual(value.currentInterval);
    const interruption = {id: 'interruption/9007199254740993', kind: 'interruption', sortAt: '2026-09-11T00:00:00Z', interruption: value.observation.latestInterruption};
    expect(models.normalizeHistoryPage({entries: [interruption], page: {}, asOf: 'now'}).entries[0]).toEqual(interruption);
});
test('recorded activity keeps exact raw facts and source location, unknown legs and optional notification', () => {
    const result = models.normalizeActivity(activity());
    expect(result.positionId).toBe('90071992547409931234567890');
    expect(result.collateralRaw).toBe('0');
    expect(result.priceNumerator).toBe('0');
    expect(result.metadata.legs).toEqual([]);
    expect(result.metadata.legsEvidence.availability).toBe('unavailable');
    expect(result.delivery).toBeUndefined();
    expect(result.summaryProgress).toBeUndefined();
    expect(result.sourceLocation).toEqual(activity().sourceLocation);
    expect(result.publicTimeEvidence.reasonCode).toBe('public_time_unobservable');
});
test('recorded delivery sent without startedAt remains sent and attempt index stays string', () => {
    const result = models.normalizeActivity(fixture('06').activity).delivery!;
    expect(result.status).toBe('sent');
    expect(result.startedAt).toBeUndefined();
    expect(result.latestAttempt!.index).toBe('1');
    expect(result.latestAttempt!.startedAt).toBeUndefined();
});
test.each(['pending', 'sending', 'sent', 'failed', 'unknown', 'cancelled'])('synthetic delivery %s and retryable attempt retain separate states', status => {
    const value = fixture('06').activity;
    value.delivery.status = status;
    value.delivery.latestAttempt.status = 'retryable';
    value.delivery.latestAttempt.index = '9007199254740993';
    expect(models.normalizeActivity(value).delivery).toMatchObject({status, latestAttempt: {status: 'retryable', index: '9007199254740993'}});
});
test.each(['waiting', 'frozen', 'cancelled_before_freeze'])('summary %s keeps related versus batch counts separate', phase => {
    const value = fixture('07').activity;
    value.summaryProgress.phase = phase;
    if (phase === 'frozen') value.summaryProgress.batchId = '9007199254740993';
    expect(models.normalizeActivity(value).summaryProgress).toEqual(value.summaryProgress);
});
test('recorded batch, parts and activity page retain different envelopes', () => {
    expect(models.normalizeSummaryBatch(fixture('09').batch)).toEqual(fixture('09').batch);
    expect(models.normalizePartPage(fixture('10'))).toEqual(fixture('10'));
    const page = models.normalizeActivityPage(fixture('15'));
    expect(page.page).toEqual(fixture('15').page);
    expect(page.page.hasNewer).toBe(false);
    expect(models.normalizePartPage(fixture('10')).parts[0].index).toBe(1);
});
test('synthetic empty lists use omitted repeated fields from Go omitempty without losing required page', () => {
    expect(models.normalizeActivityPage({page: {hasNewer: false}})).toEqual({activities: [], page: {hasNewer: false}});
    expect(models.normalizeSubscriptionPage({page: {}, quota: {used: 0, limit: 10}, asOf: 'now'}).subscriptions).toEqual([]);
    expect(models.normalizePartPage({page: {}, asOf: 'now'}).parts).toEqual([]);
    expect(models.normalizeHistoryPage({page: {}, asOf: 'now'}).entries).toEqual([]);
});
test('recorded anomaly retains both hashes; synthetic absent conflict stays absent', () => {
    const value = fixture('17').activity;
    expect(models.normalizeActivity(value).finalityAnomaly).toEqual(value.finalityAnomaly);
    delete value.finalityAnomaly.conflictingBlockHash;
    expect(models.normalizeActivity(value).finalityAnomaly!.conflictingBlockHash).toBeUndefined();
});
test('synthetic SELL Combo preserves full markets and exact IDs without inventing links', () => {
    const value = activity();
    value.side = 'SELL';
    value.id = '9007199254740993';
    value.sourceLocation.logIndex = '0';
    value.metadata.relationship = 'NOT(AND(legs))';
    value.metadata.legsEvidence = available;
    value.metadata.legs = [
        {
            positionId: '9007199254740993123',
            market: {
                evidence: available,
                id: 'market-slug',
                title: 'Leg',
                url: 'https://example.com/market',
                conditionId: 'condition',
                positionId: '9007199254740993123',
                outcome: 'NO'
            }
        }
    ];
    const result = models.normalizeActivity(value);
    expect(result.id).toBe('9007199254740993');
    expect(result.sourceLocation.logIndex).toBe('0');
    expect(result.metadata).toEqual(value.metadata);
    expect(result.side).toBe('SELL');
});
// These are deliberately malformed synthetic protocol inputs, not captured responses.
test.each([
    ['id', ''],
    ['id', 1],
    ['id', '0'],
    ['id', '-1'],
    ['subscriptionId', 'bogus'],
    ['side', 'buy'],
    ['collateralRaw', 9007199254740992],
    ['collateralRaw', '1e9'],
    ['collateralDecimals', '6'],
    ['sharesDecimals', -1],
    ['sourceLocation', {}],
    ['priceEvidence', {availability: 'unknown'}],
    ['metadata', null],
    ['notificationMode', 'email'],
    ['targetDisplaySnapshot', undefined],
    ['recordedAt', undefined]
])('rejects malformed required activity %s = %s', (key, value) => {
    expect(() => models.normalizeActivity({...activity(), [key]: value})).toThrow(/protocol/i);
});
test.each([
    (v: any) => {
        v.revision = 9007199254740993;
    },
    (v: any) => {
        v.status = 'monitoring';
    },
    (v: any) => {
        v.noteRevision = undefined;
    },
    (v: any) => {
        v.observation.state = 'error';
    },
    (v: any) => {
        v.queueCounts.total = '-1';
    },
    (v: any) => {
        v.wallet = '0xabc';
    }
])('rejects malformed required subscription fields', mutate => {
    const value = sub();
    mutate(value);
    expect(() => models.normalizeSubscription(value)).toThrow(/protocol/i);
});
test.each([
    (v: any) => {
        v.verified.value = 'false';
    },
    (v: any) => {
        v.positionValue.value = 0;
    },
    (v: any) => {
        v.pnl.pop();
    },
    (v: any) => {
        v.pnl[0].period = 'ALL';
    },
    (v: any) => {
        v.quota.used = 1.1;
    },
    (v: any) => {
        v.savedNote.revision = '-1';
    },
    (v: any) => {
        v.displayName = {value: 'missing evidence'};
    }
])('rejects malformed required resolved fields', mutate => {
    const value = target();
    mutate(value);
    expect(() => models.normalizeResolvedTarget(value)).toThrow(/protocol/i);
});
test('rejects snake_case and administrator envelopes instead of manufacturing member resources', () => {
    const value = activity();
    value.source_record_id = value.sourceRecordId;
    delete value.sourceRecordId;
    expect(() => models.normalizeActivity(value)).toThrow(/protocol/i);
    expect(() => models.normalizeSubscription({subscriptionId: sub().id, accountId: 'owner', status: 'pending_baseline'})).toThrow(/protocol/i);
    expect(() => models.normalizeActivityPage({activities: []})).toThrow(/protocol/i);
    expect(() => models.normalizePartPage({...fixture('10'), parts: [{...fixture('10').parts[0], index: '1'}]})).toThrow(/protocol/i);
});

const fakeRequest = (body: unknown) => Object.assign(Promise.resolve({body}), {abort: jest.fn(), query: jest.fn().mockReturnThis(), send: jest.fn().mockReturnThis()});
const svc = new MemberTraderSyncService();
const change = {expectedRevision: '9007199254740993', requestId: 'request-1'};
const page = {'page.page_size': 20, 'page.cursor': 'opaque/+=cursor'};
const input = {pageSize: 20, cursor: 'opaque/+=cursor'};
const cases: Array<[string, 'get' | 'post' | 'patch', string, 'read' | 'write', any, any, any, any]> = [
    ['resolveTarget', 'post', '/targets:resolve', 'read', ['wallet'], {input: 'wallet'}, undefined, fixture('resolve-saved-note-existing')],
    [
        'createSubscription',
        'post',
        '/subscriptions',
        'write',
        [{confirmationToken: 'token', requestId: 'r'}],
        {confirmationToken: 'token', requestId: 'r'},
        undefined,
        fixture('create-real-confirmation')
    ],
    ['listSubscriptions', 'get', '/subscriptions', 'read', [{...input, view: 'current', state: 'healthy'}], undefined, {...page, view: 'current', state: 'healthy'}, fixture('03')],
    ['getSubscription', 'get', '/subscriptions/a%2Fb', 'read', ['a/b'], undefined, undefined, fixture('01')],
    ['listSubscriptionHistory', 'get', '/subscriptions/a%2Fb/history', 'read', ['a/b', input], undefined, page, fixture('12')],
    ['pauseSubscription', 'post', '/subscriptions/a%2Fb:pause', 'write', ['a/b', change], change, undefined, fixture('01')],
    ['resumeSubscription', 'post', '/subscriptions/a%2Fb:resume', 'write', ['a/b', change], change, undefined, fixture('01')],
    ['cancelSubscription', 'post', '/subscriptions/a%2Fb:cancel', 'write', ['a/b', change], change, undefined, fixture('01')],
    ['updateTargetNote', 'patch', '/targets/a%2Fb/note', 'write', ['a/b', {...change, note: ''}], {...change, note: ''}, undefined, {note: target().savedNote}],
    [
        'listActivities',
        'get',
        '/activities',
        'read',
        [{...input, subscriptionId: 'sub', summaryBatchId: 'batch', from: 'from', to: 'to'}],
        undefined,
        {...page, subscription_id: 'sub', summary_batch_id: 'batch', from: 'from', to: 'to'},
        fixture('15')
    ],
    ['getActivity', 'get', '/activities/a%2Fb', 'read', ['a/b'], undefined, undefined, fixture('05')],
    ['getSummaryBatch', 'get', '/summaries/a%2Fb', 'read', ['a/b'], undefined, undefined, fixture('09')],
    ['listSummaryParts', 'get', '/summaries/a%2Fb/parts', 'read', ['a/b', {...input, activityId: 'long-id'}], undefined, {...page, activity_id: 'long-id'}, fixture('10')]
];
test.each(cases)('%s sends exact route, scope, body/query and forwards abort', async (method, verb, path, mode, args, body, query, response) => {
    const request = fakeRequest(response);
    const spy = jest.spyOn(requests, verb).mockReturnValue(request as any);
    const promise = (svc as any)[method](...args);
    const result = await promise;
    expect(spy).toHaveBeenCalledWith('/trader-sync' + path, {module: AccountDataModule.TraderSync, mode});
    if (body) expect(request.send).toHaveBeenCalledWith(body);
    else expect(request.send).not.toHaveBeenCalled();
    if (query) expect(request.query).toHaveBeenCalledWith(query);
    else expect(request.query).not.toHaveBeenCalled();
    expect(result).toBeDefined();
    promise.abort();
    expect(request.abort).toHaveBeenCalledTimes(1);
});
test('create preserves absent versus explicitly empty note wrapper', async () => {
    const request = fakeRequest(fixture('01'));
    jest.spyOn(requests, 'post').mockReturnValue(request as any);
    await svc.createSubscription({confirmationToken: 't', requestId: 'r', note: {value: ''}});
    expect(request.send).toHaveBeenCalledWith({confirmationToken: 't', requestId: 'r', note: {value: ''}});
});
test('activity refresh uses only refresh_cursor and drops undefined query values', async () => {
    const request = fakeRequest(fixture('15'));
    jest.spyOn(requests, 'get').mockReturnValue(request as any);
    await svc.listActivities({refreshCursor: 'opaque-refresh', from: undefined});
    expect(request.query).toHaveBeenCalledWith({refresh_cursor: 'opaque-refresh'});
    expect(() => svc.listActivities({cursor: 'page', refreshCursor: 'refresh'})).toThrow(/cursor/i);
});
test('service rejects malformed DTO instead of accepting a resolved HTTP promise', async () => {
    jest.spyOn(requests, 'get').mockReturnValue(fakeRequest({activity: {}}) as any);
    await expect(svc.getActivity('1')).rejects.toThrow(/protocol/i);
});

test.each(['garbage', 'interval/123', 'interruption/0', 'interruption/uuid'])('rejects synthetic malformed history id %s', id => {
    const value = fixture('12');
    value.entries[0].id = id;
    expect(() => models.normalizeHistoryPage(value)).toThrow(/protocol/i);
});
test.each([0, -1, 1.5, 2147483648])('rejects synthetic out-of-range part index %s', index => {
    const value = fixture('10');
    value.parts[0].index = index;
    expect(() => models.normalizePartPage(value)).toThrow(/protocol/i);
});
