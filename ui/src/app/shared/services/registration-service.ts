import requests from './requests';

export type UsernameAvailability = 'available' | 'invalid' | 'unavailable';

export type RegistrationErrorReason = 'username_invalid' | 'username_unavailable' | 'registration_expired' | 'registration_unavailable' | 'google_not_allowed' | 'maintenance';

export type RegistrationIdentityProvider = 'google' | 'solana_wallet';

export interface Registration {
    provider: RegistrationIdentityProvider;
    verifiedEmail: string;
    solanaAddress: string;
    administrator: boolean;
    expiresAt: number;
    csrfToken: string;
}

export interface CreateRegistrationResult {
    redirectTo: string;
}

export class RegistrationRequestError extends Error {
    public readonly status: number;
    public readonly reason?: RegistrationErrorReason;

    constructor(status: number, message: string, reason?: RegistrationErrorReason) {
        super(message);
        this.name = 'RegistrationRequestError';
        this.status = status;
        this.reason = reason;
    }
}

const registrationPath = '/auth/registration';

const isRecord = (value: unknown): value is Record<string, unknown> => Boolean(value) && typeof value === 'object';

const readJSON = async (response: Response): Promise<Record<string, unknown>> => {
    const text = await response.text();
    if (!text) {
        return {};
    }
    try {
        const value: unknown = JSON.parse(text);
        return isRecord(value) ? value : {};
    } catch {
        return {};
    }
};

const registrationReason = (response: Response, body: Record<string, unknown>): RegistrationErrorReason | undefined => {
    const headerReason = response.headers.get('X-Athena-Error-Reason');
    const error = isRecord(body.error) ? body.error : undefined;
    const value = headerReason || body.reason || error?.reason;
    switch (value) {
        case 'username_invalid':
        case 'username_unavailable':
        case 'registration_expired':
        case 'registration_unavailable':
        case 'google_not_allowed':
        case 'maintenance':
            return value;
        default:
            return undefined;
    }
};

const requestJSON = async (path: string, init?: RequestInit): Promise<Record<string, unknown>> => {
    const headers = new Headers(init?.headers);
    headers.set('Accept', 'application/json');
    const response = await fetch(requests.toAbsURL(path), {
        credentials: 'same-origin',
        ...init,
        headers
    });
    const body = await readJSON(response);
    if (!response.ok) {
        const error = isRecord(body.error) ? body.error : undefined;
        const message = String(body.message || error?.message || response.statusText || `Request failed (${response.status})`);
        throw new RegistrationRequestError(response.status, message, registrationReason(response, body));
    }
    return body;
};

const stringValue = (value: unknown) => (typeof value === 'string' ? value : '');
const registrationProvider = (value: unknown): RegistrationIdentityProvider => {
    if (value === 1 || value === 'google' || value === 'ACCOUNT_IDENTITY_PROVIDER_GOOGLE') {
        return 'google';
    }
    if (value === 3 || value === 'solana_wallet' || value === 'ACCOUNT_IDENTITY_PROVIDER_SOLANA_WALLET') {
        return 'solana_wallet';
    }
    throw new Error('Registration identity provider is invalid');
};

export class RegistrationService {
    public async get(): Promise<Registration> {
        const body = await requestJSON(registrationPath);
        return {
            provider: registrationProvider(body.provider),
            verifiedEmail: stringValue(body.verifiedEmail ?? body.verified_email),
            solanaAddress: stringValue(body.solanaAddress ?? body.solana_address),
            administrator: Boolean(body.administrator),
            expiresAt: Number(body.expiresAt ?? body.expires_at ?? 0),
            csrfToken: stringValue(body.csrfToken ?? body.csrf_token)
        };
    }

    public async usernameAvailability(username: string, signal?: AbortSignal): Promise<UsernameAvailability> {
        const query = new URLSearchParams({username});
        const body = await requestJSON(`${registrationPath}/username-availability?${query.toString()}`, {signal});
        const status = body.status;
        if (status === 'available' || status === 'invalid' || status === 'unavailable') {
            return status;
        }
        throw new Error('Username availability response is invalid');
    }

    public async create(username: string, csrfToken: string): Promise<CreateRegistrationResult> {
        const body = await requestJSON(registrationPath, {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({username, csrfToken})
        });
        return {redirectTo: stringValue(body.redirectTo ?? body.redirect_to)};
    }

    public async cancel(csrfToken: string): Promise<CreateRegistrationResult> {
        const body = await requestJSON(registrationPath, {
            method: 'DELETE',
            headers: {'X-Athena-CSRF-Token': csrfToken}
        });
        return {redirectTo: stringValue(body.redirectTo ?? body.redirect_to)};
    }
}

export const registrationService = new RegistrationService();
