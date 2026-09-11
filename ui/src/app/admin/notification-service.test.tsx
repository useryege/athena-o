import requests from '../shared/services/requests';
import {AdminNotificationService} from './notification-service';

afterEach(() => jest.restoreAllMocks());

// Mock only the network request; normalize the real public service response.
const responseRequest = (body: unknown) => Object.assign(Promise.resolve({body}), {abort: jest.fn()});

test('unknown delivery keeps its result and missing HTTP start evidence', async () => {
    const request = responseRequest({item: {id: 1, status: 'unknown', authorizedAt: '2026-09-10T01:00:00Z', startedAt: '', resultAt: '2026-09-10T01:00:05Z', sentAt: ''}});
    jest.spyOn(requests, 'get').mockReturnValue(request as any);
    const item = await new AdminNotificationService().getNotification(1);
    expect(item.status).toBe('unknown');
    expect(item.authorizedAt).toBe('2026-09-10T01:00:00Z');
    expect(item.startedAt).toBe('');
    expect(item.resultAt).toBe('2026-09-10T01:00:05Z');
    expect(item.sentAt).toBe('');
});

test('runtime counts distinguish unknown and sending from failed and pending', async () => {
    const request = responseRequest({
        systemSendingCount: '2',
        systemUnknownCount: '3',
        accountSendingCount: '4',
        accountUnknownCount: '5',
        systemFailedCount: '1',
        accountFailedCount: '6'
    });
    jest.spyOn(requests, 'get').mockReturnValue(request as any);
    const item = await new AdminNotificationService().getRuntimeStatus();
    expect(item).toMatchObject({
        systemSendingCount: 2,
        systemUnknownCount: 3,
        accountSendingCount: 4,
        accountUnknownCount: 5,
        systemFailedCount: 1,
        accountFailedCount: 6,
        systemPendingCount: 0,
        accountPendingCount: 0
    });
});

test.each([
    ['01-not-entered', 'stopped', undefined, undefined, undefined],
    ['02-initializing', 'recovering', 'initializing', undefined, '0'],
    ['03-first-completed', 'running', 'completed', '0', '0'],
    ['04-waiting', 'recovering', 'waiting', '55000', '5000'],
    ['05-cancelled', 'degraded', 'cancelled', '55000', '5000'],
    ['06-fatal', 'failed', 'cancelled', '55000', '5000'],
    ['07-stopped', 'stopped', 'cancelled', '55000', '5000'],
    ['08-recovery-failed', 'failed', 'failed', undefined, '0']
])('runtime preserves frozen gateway recovery %s without starting a browser budget', async (name, status, state, remainingMillis, elapsedMillis) => {
    const body = require(`./__fixtures__/trader-sync/notification-${name}.json`);
    jest.spyOn(requests, 'get').mockReturnValue(responseRequest(body) as any);
    const result = await new AdminNotificationService().getRuntimeStatus();
    expect(result.status).toBe(status);
    expect((result as any).recovery?.state).toBe(state);
    expect((result as any).recovery?.remainingMillis).toBe(remainingMillis);
    expect((result as any).recovery?.elapsedMillis).toBe(elapsedMillis);
    expect(result.pollerActive).toBe(body.poller_active === true);
    if (state === undefined) expect((result as any).recovery).toBeUndefined();
});
