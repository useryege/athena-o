import requests from '../shared/services/requests';

type AbortablePromise<T> = Promise<T> & {abort?: () => void};

const notificationReadScope = {feature: 'admin-notifications' as const, mode: 'read' as const};
const notificationWriteScope = {feature: 'admin-notifications' as const, mode: 'write' as const};

export interface NotificationDelivery {
    id: number;
    source: string;
    severity: string;
    topicLabel: string;
    title: string;
    body: string;
    link: string;
    channel: string;
    status: string;
    telegramChat: string;
    providerMessageId: string;
    errorMessage: string;
    createdAt: string;
    sentAt: string;
    authorizedAt: string;
    startedAt: string;
    resultAt: string;
}

export interface ListNotificationsOptions {
    page?: number;
    pageSize?: number;
    status?: string;
    severity?: string;
    telegramChat?: string;
    topicLabel?: string;
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

export interface NotificationRecovery {
    state: string;
    reason: string;
    startedAt?: string;
    remainingMillis?: string;
    elapsedMillis?: string;
    clockSource: string;
}

export interface SystemNotificationRuntimeStatus {
    recovery?: NotificationRecovery;
    started: boolean;
    status: string;
    botAvailable: boolean;
    botId: number;
    botUsername: string;
    pollerActive: boolean;
    lastPollAt: string;
    lastUpdateAt: string;
    systemPendingCount: number;
    systemRetryCount: number;
    systemFailedCount: number;
    accountPendingCount: number;
    accountRetryCount: number;
    accountFailedCount: number;
    unreachableBindingCount: number;
    systemSendingCount: number;
    systemUnknownCount: number;
    accountSendingCount: number;
    accountUnknownCount: number;
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
const readBoolean = (item: any, ...names: string[]) => Boolean(readValue(item, ...names));

const normalizeDelivery = (item: any = {}): NotificationDelivery => ({
    id: readNumber(item, 'id'),
    source: readString(item, 'source'),
    severity: readString(item, 'severity'),
    topicLabel: readString(item, 'topicLabel', 'topic_label'),
    title: readString(item, 'title'),
    body: readString(item, 'body'),
    link: readString(item, 'link'),
    channel: readString(item, 'channel'),
    status: readString(item, 'status'),
    telegramChat: readString(item, 'telegramChat', 'telegram_chat'),
    providerMessageId: readString(item, 'providerMessageId', 'provider_message_id'),
    errorMessage: readString(item, 'errorMessage', 'error_message'),
    createdAt: readString(item, 'createdAt', 'created_at'),
    sentAt: readString(item, 'sentAt', 'sent_at'),
    authorizedAt: readString(item, 'authorizedAt', 'authorized_at'),
    startedAt: readString(item, 'startedAt', 'started_at'),
    resultAt: readString(item, 'resultAt', 'result_at')
});

const normalizeRuntimeStatus = (item: any = {}): SystemNotificationRuntimeStatus => ({
    recovery:
        item.recovery && typeof item.recovery === 'object'
            ? {
                  state: readString(item.recovery, 'state'),
                  reason: readString(item.recovery, 'reason'),
                  startedAt: typeof item.recovery.startedAt === 'string' ? item.recovery.startedAt : undefined,
                  remainingMillis: typeof item.recovery.remainingMillis === 'string' ? item.recovery.remainingMillis : undefined,
                  elapsedMillis: typeof item.recovery.elapsedMillis === 'string' ? item.recovery.elapsedMillis : undefined,
                  clockSource: readString(item.recovery, 'clockSource')
              }
            : undefined,
    started: readBoolean(item, 'started'),
    status: readString(item, 'status'),
    botAvailable: readBoolean(item, 'botAvailable', 'bot_available'),
    botId: readNumber(item, 'botId', 'bot_id'),
    botUsername: readString(item, 'botUsername', 'bot_username'),
    pollerActive: readBoolean(item, 'pollerActive', 'poller_active'),
    lastPollAt: readString(item, 'lastPollAt', 'last_poll_at'),
    lastUpdateAt: readString(item, 'lastUpdateAt', 'last_update_at'),
    systemPendingCount: readNumber(item, 'systemPendingCount', 'system_pending_count'),
    systemRetryCount: readNumber(item, 'systemRetryCount', 'system_retry_count'),
    systemFailedCount: readNumber(item, 'systemFailedCount', 'system_failed_count'),
    accountPendingCount: readNumber(item, 'accountPendingCount', 'account_pending_count'),
    accountRetryCount: readNumber(item, 'accountRetryCount', 'account_retry_count'),
    accountFailedCount: readNumber(item, 'accountFailedCount', 'account_failed_count'),
    systemSendingCount: readNumber(item, 'systemSendingCount', 'system_sending_count'),
    systemUnknownCount: readNumber(item, 'systemUnknownCount', 'system_unknown_count'),
    accountSendingCount: readNumber(item, 'accountSendingCount', 'account_sending_count'),
    accountUnknownCount: readNumber(item, 'accountUnknownCount', 'account_unknown_count'),
    unreachableBindingCount: readNumber(item, 'unreachableBindingCount', 'unreachable_binding_count')
});

export class AdminNotificationService {
    public getRuntimeStatus(): AbortablePromise<SystemNotificationRuntimeStatus> {
        const request = requests.get('/admin/notification-runtime/status', notificationReadScope);
        const promise = request.then((response: any) => normalizeRuntimeStatus(response.body || {})) as AbortablePromise<SystemNotificationRuntimeStatus>;
        promise.abort = () => request.abort();
        return promise;
    }

    public listNotifications(options: ListNotificationsOptions = {}): AbortablePromise<ListNotificationsResult> {
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
        if (options.telegramChat) {
            query.telegram_chat = options.telegramChat;
        }
        if (options.topicLabel) {
            query.topic_label = options.topicLabel;
        }
        if (options.source) {
            query.source = options.source;
        }
        if (options.keyword) {
            query.keyword = options.keyword;
        }

        const request = requests.get('/admin/system-notification-deliveries', notificationReadScope).query(query);
        const promise = request.then((response: any) => {
            const body = response.body || {};
            return {
                items: (body.items || []).map(normalizeDelivery),
                total: readNumber(body, 'total'),
                page: readNumber(body, 'page') || query.page,
                pageSize: readNumber(body, 'pageSize', 'page_size') || query.page_size
            };
        }) as AbortablePromise<ListNotificationsResult>;
        promise.abort = () => request.abort();
        return promise;
    }

    public getNotification(id: number | string): AbortablePromise<NotificationDelivery> {
        const request = requests.get(`/admin/system-notification-deliveries/${encodeURIComponent(String(id))}`, notificationReadScope);
        const promise = request.then((response: any) => normalizeDelivery((response.body || {}).item || response.body || {})) as AbortablePromise<NotificationDelivery>;
        promise.abort = () => request.abort();
        return promise;
    }

    public sendTestNotification(topicLabel: string): AbortablePromise<SendTestNotificationResult> {
        const request = requests.post('/admin/system-notification-tests', notificationWriteScope).send({topicLabel});
        const promise = request.then((response: any) => {
            const body = response.body || {};
            return {
                notificationId: readNumber(body, 'notificationId', 'notification_id'),
                status: readString(body, 'status'),
                providerMessageId: readString(body, 'providerMessageId', 'provider_message_id'),
                errorMessage: readString(body, 'errorMessage', 'error_message')
            };
        }) as AbortablePromise<SendTestNotificationResult>;
        promise.abort = () => request.abort();
        return promise;
    }
}
