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
