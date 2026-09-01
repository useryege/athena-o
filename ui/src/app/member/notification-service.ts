import requests from '../shared/services/requests';

type AbortablePromise<T> = Promise<T> & {abort?: () => void};

const notificationReadScope = {feature: 'member-notifications' as const, mode: 'read' as const};
const notificationWriteScope = {feature: 'member-notifications' as const, mode: 'write' as const};

export type TelegramBindingStatus = 'connected' | 'unreachable';
export type TelegramBindingAttemptStatus = 'pending' | 'failed';

export interface TelegramNotificationBinding {
    status: TelegramBindingStatus;
    telegramUsername: string;
    telegramDisplayName: string;
    boundAt: string;
    revision: number;
}

export interface TelegramBindingAttempt {
    id: string;
    status: TelegramBindingAttemptStatus;
    expiresAt: string;
    failureReason: string;
}

export interface TelegramNotificationSettings {
    botAvailable: boolean;
    botUsername: string;
    binding?: TelegramNotificationBinding;
    attempt?: TelegramBindingAttempt;
}

export interface BeginTelegramBindingResult {
    attempt: TelegramBindingAttempt;
    botUsername: string;
    deepLink: string;
    fallbackCommand: string;
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

const normalizeBinding = (value: any): TelegramNotificationBinding | undefined => {
    if (!value) {
        return undefined;
    }
    const status = readString(value, 'status').toLowerCase();
    if (status !== 'connected' && status !== 'unreachable') {
        throw new Error(`Unsupported Telegram notification binding status: ${status || 'empty'}`);
    }
    return {
        status,
        telegramUsername: readString(value, 'telegramUsername', 'telegram_username'),
        telegramDisplayName: readString(value, 'telegramDisplayName', 'telegram_display_name'),
        boundAt: readString(value, 'boundAt', 'bound_at'),
        revision: readNumber(value, 'revision')
    };
};

const normalizeAttempt = (value: any): TelegramBindingAttempt | undefined => {
    if (!value) {
        return undefined;
    }
    const status = readString(value, 'status').toLowerCase();
    if (status !== 'pending' && status !== 'failed') {
        throw new Error(`Unsupported Telegram notification binding attempt status: ${status || 'empty'}`);
    }
    return {
        id: readString(value, 'id'),
        status,
        expiresAt: readString(value, 'expiresAt', 'expires_at'),
        failureReason: readString(value, 'failureReason', 'failure_reason')
    };
};

const normalizeSettings = (value: any = {}): TelegramNotificationSettings => ({
    botAvailable: readBoolean(value, 'botAvailable', 'bot_available'),
    botUsername: readString(value, 'botUsername', 'bot_username'),
    binding: normalizeBinding(readValue(value, 'binding')),
    attempt: normalizeAttempt(readValue(value, 'attempt'))
});

const normalizeBeginResult = (value: any = {}): BeginTelegramBindingResult => {
    const attempt = normalizeAttempt(readValue(value, 'attempt'));
    if (!attempt || attempt.status !== 'pending' || !attempt.id) {
        throw new Error('Telegram notification binding did not return a pending attempt');
    }
    const result = {
        attempt,
        botUsername: readString(value, 'botUsername', 'bot_username'),
        deepLink: readString(value, 'deepLink', 'deep_link'),
        fallbackCommand: readString(value, 'fallbackCommand', 'fallback_command')
    };
    if (!result.deepLink || !result.fallbackCommand) {
        throw new Error('Telegram notification binding instructions are incomplete');
    }
    return result;
};

const abortable = <T>(request: any, normalize: (body: any) => T): AbortablePromise<T> => {
    const promise = request.then((response: any) => normalize(response.body || {})) as AbortablePromise<T>;
    promise.abort = () => request.abort();
    return promise;
};

export class MemberNotificationService {
    public getTelegramSettings(): AbortablePromise<TelegramNotificationSettings> {
        const request = requests.get('/notification-bindings/telegram', notificationReadScope);
        return abortable(request, normalizeSettings);
    }

    public beginTelegramBinding(): AbortablePromise<BeginTelegramBindingResult> {
        const request = requests.post('/notification-bindings/telegram/attempt', notificationWriteScope).send({});
        return abortable(request, normalizeBeginResult);
    }

    public cancelTelegramBindingAttempt(): AbortablePromise<void> {
        const request = requests.delete('/notification-bindings/telegram/attempt', notificationWriteScope);
        return abortable(request, () => undefined);
    }

    public disconnectTelegram(): AbortablePromise<void> {
        const request = requests.delete('/notification-bindings/telegram', notificationWriteScope);
        return abortable(request, () => undefined);
    }
}
