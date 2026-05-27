import {NotificationService} from './notification-service';

const mockGet = jest.fn();

jest.mock('./requests', () => ({
    __esModule: true,
    default: {
        get: (...args: any[]) => mockGet(...args)
    }
}));

const requestWithBody = (body: any) => {
    const request: any = {
        abort: jest.fn(),
        query: jest.fn(() => request),
        then: (resolve: any) => Promise.resolve(resolve({body}))
    };
    return request;
};

describe('notifications service', () => {
    beforeEach(() => mockGet.mockReset());

    it('sends list filters as query parameters', async () => {
        const request = requestWithBody({
            items: [
                {
                    id: 7,
                    source: 'worm',
                    severity: 'warning',
                    title: 'scan',
                    body: 'body',
                    channel: 'telegram',
                    status: 'sent',
                    provider_message_id: '123',
                    created_at: '2026-05-27T12:00:00Z'
                }
            ],
            total: 1,
            page: 2,
            page_size: 20
        });
        mockGet.mockReturnValue(request);

        const result = await new NotificationService().listNotifications({
            page: 2,
            pageSize: 20,
            status: 'sent',
            severity: 'warning',
            source: 'worm',
            keyword: 'scan'
        });

        expect(mockGet).toHaveBeenCalledWith('/notifications');
        expect(request.query).toHaveBeenCalledWith({
            page: 2,
            page_size: 20,
            status: 'sent',
            severity: 'warning',
            source: 'worm',
            keyword: 'scan'
        });
        expect(result).toMatchObject({
            total: 1,
            page: 2,
            pageSize: 20,
            items: [{id: 7, providerMessageId: '123', createdAt: '2026-05-27T12:00:00Z'}]
        });
    });

    it('loads notification details', async () => {
        mockGet.mockReturnValue(
            requestWithBody({
                item: {
                    id: '9',
                    source: 'application',
                    status: 'failed',
                    error_message: 'telegram unavailable'
                }
            })
        );

        const item = await new NotificationService().getNotification(9);

        expect(mockGet).toHaveBeenCalledWith('/notifications/9');
        expect(item.id).toBe(9);
        expect(item.errorMessage).toBe('telegram unavailable');
    });
});
