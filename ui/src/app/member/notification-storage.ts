import type {BeginTelegramBindingResult} from './notification-service';

const telegramAttemptStorageKey = 'athena.member.notifications.telegram-attempt';

export interface StoredTelegramInstructions {
    attemptId: string;
    deepLink: string;
    fallbackCommand: string;
}

const isSafeTelegramDeepLink = (value: string) => {
    try {
        const url = new URL(value);
        return url.protocol === 'https:' || url.protocol === 'tg:';
    } catch {
        return false;
    }
};

const validInstructions = (value: Partial<StoredTelegramInstructions>): value is StoredTelegramInstructions =>
    typeof value.attemptId === 'string' &&
    Boolean(value.attemptId) &&
    typeof value.deepLink === 'string' &&
    isSafeTelegramDeepLink(value.deepLink) &&
    typeof value.fallbackCommand === 'string' &&
    Boolean(value.fallbackCommand);

export const readTelegramBindingInstructions = (): StoredTelegramInstructions | undefined => {
    try {
        const raw = window.sessionStorage.getItem(telegramAttemptStorageKey);
        if (!raw) {
            return undefined;
        }
        const value = JSON.parse(raw) as Partial<StoredTelegramInstructions>;
        if (!validInstructions(value)) {
            window.sessionStorage.removeItem(telegramAttemptStorageKey);
            return undefined;
        }
        return {attemptId: value.attemptId, deepLink: value.deepLink, fallbackCommand: value.fallbackCommand};
    } catch {
        clearTelegramBindingInstructions();
        return undefined;
    }
};

export const storeTelegramBindingInstructions = (result: BeginTelegramBindingResult): StoredTelegramInstructions => {
    const value = {attemptId: result.attempt.id, deepLink: result.deepLink, fallbackCommand: result.fallbackCommand};
    if (!validInstructions(value)) {
        throw new Error('Telegram setup returned an unsupported one-time link');
    }
    try {
        window.sessionStorage.setItem(telegramAttemptStorageKey, JSON.stringify(value));
    } catch {
        // The setup remains usable in this render even when storage is disabled.
    }
    return value;
};

export const clearTelegramBindingInstructions = () => {
    try {
        window.sessionStorage.removeItem(telegramAttemptStorageKey);
    } catch {
        // Storage cleanup is best effort; the server attempt remains authoritative.
    }
};
