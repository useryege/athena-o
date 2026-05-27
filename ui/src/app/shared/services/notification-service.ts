import requests from './requests';

export interface NotificationDelivery {
    id: number;
    source: string;
    severity: string;
    title: string;
    body: string;
    link: string;
    channel: string;
    status: string;
    providerMessageId: string;
    errorMessage: string;
    createdAt: string;
    sentAt: string;
}

export interface ListNotificationsOptions {
    page?: number;
    pageSize?: number;
    status?: string;
    severity?: string;
    source?: string;
    keyword?: string;
}

export interface ListNotificationsResult {
    items: NotificationDelivery[];
    total: number;
    page: number;
    pageSize: number;
}

export interface SendTestNotificationResult {
    notificationId: number;
    status: string;
    providerMessageId: string;
    errorMessage: string;
}

const readValue = (item: any, ...names: string[]) => {
    for (const name of names) {
        if (item?.[name] !== undefined && item?.[name] !== null) {
            return item[name];
        }
    }
    return undefined;
};

const readString = (item: any, ...names: string[]) => String(readValue(item, ...names) || '');
const readNumber = (item: any, ...names: string[]) => Number(readValue(item, ...names) || 0) || 0;

const normalizeDelivery = (item: any = {}): NotificationDelivery => ({
    id: readNumber(item, 'id'),
    source: readString(item, 'source'),
    severity: readString(item, 'severity'),
    title: readString(item, 'title'),
    body: readString(item, 'body'),
    link: readString(item, 'link'),
    channel: readString(item, 'channel'),
    status: readString(item, 'status'),
    providerMessageId: readString(item, 'providerMessageId', 'provider_message_id'),
    errorMessage: readString(item, 'errorMessage', 'error_message'),
    createdAt: readString(item, 'createdAt', 'created_at'),
    sentAt: readString(item, 'sentAt', 'sent_at')
});

export class NotificationService {
    public listNotifications(options: ListNotificationsOptions = {}): Promise<ListNotificationsResult> & {abort?: () => void} {
        const query: any = {
            page: options.page || 1,
            page_size: options.pageSize || 20
        };
        if (options.status) {
            query.status = options.status;
        }
        if (options.severity) {
            query.severity = options.severity;
        }
        if (options.source) {
            query.source = options.source;
        }
        if (options.keyword) {
            query.keyword = options.keyword;
        }

        const req = requests.get('/notifications').query(query);
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: (body.items || []).map(normalizeDelivery),
                total: readNumber(body, 'total'),
                page: readNumber(body, 'page') || query.page,
                pageSize: readNumber(body, 'pageSize', 'page_size') || query.page_size
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public getNotification(id: number | string): Promise<NotificationDelivery> & {abort?: () => void} {
        const req = requests.get(`/notifications/${encodeURIComponent(String(id))}`);
        const promise = req.then(res => normalizeDelivery((res.body || {}).item || res.body || {})) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public sendTestNotification(): Promise<SendTestNotificationResult> & {abort?: () => void} {
        const req = requests.post('/notifications/test').send({});
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                notificationId: readNumber(body, 'notificationId', 'notification_id'),
                status: readString(body, 'status'),
                providerMessageId: readString(body, 'providerMessageId', 'provider_message_id'),
                errorMessage: readString(body, 'errorMessage', 'error_message')
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }
}
