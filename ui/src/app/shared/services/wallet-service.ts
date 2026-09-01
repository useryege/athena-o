import {AccountDataModule} from '../access-modules';
import requests from './requests';

const readScope = {module: AccountDataModule.Wallet, mode: 'read' as const};
const writeScope = {module: AccountDataModule.Wallet, mode: 'write' as const};

export type AbortablePromise<T> = Promise<T> & {abort?: () => void};

export type WalletType = 'EVM' | 'SOLANA';
export type WalletSource = 'created' | 'imported';
export type WalletAvatarKind = 'default' | 'preset' | 'upload';

export const walletTypeOptions: {value: WalletType; label: string}[] = [
    {value: 'EVM', label: 'EVM'},
    {value: 'SOLANA', label: 'Solana'}
];

export const walletAvatarPresets = ['star-violet', 'bolt-blue', 'gem-cyan', 'leaf-green', 'sun-amber', 'flame-orange', 'heart-rose', 'moon-indigo'] as const;

export type WalletAvatarPresetID = (typeof walletAvatarPresets)[number];

export interface WalletItem {
    id: number;
    walletType: WalletType;
    address: string;
    remark: string;
    source: WalletSource;
    avatarKind: WalletAvatarKind;
    avatarPresetId: string;
    avatarUrl: string;
    revision: number;
    createdAt: string;
    updatedAt: string;
}

export interface ListWalletsOptions {
    walletType?: WalletType;
    query?: string;
    page?: number;
    pageSize?: number;
}

export interface ListWalletsResult {
    items: WalletItem[];
    total: number;
    page: number;
    pageSize: number;
}

export interface CreateWalletInput {
    walletType: WalletType;
    remark: string;
    avatarPresetId?: string;
}

export interface ImportWalletInput extends CreateWalletInput {
    privateKey: string;
}

export interface CreateWalletResult {
    item: WalletItem;
    privateKey: string;
}

export interface WalletSecretChallenge {
    message: string;
    expiresAt: number;
}

export interface WalletSecretLease {
    expiresAt: number;
}

export const WALLET_LOGIN_SESSION_REQUIRED = 'WALLET_LOGIN_SESSION_REQUIRED';
export const WALLET_REAUTH_REQUIRED = 'WALLET_REAUTH_REQUIRED';
export const WALLET_REAUTH_UNAVAILABLE = 'WALLET_REAUTH_UNAVAILABLE';

const readValue = (item: any, ...names: string[]) => {
    for (const name of names) {
        if (item?.[name] !== undefined && item?.[name] !== null) {
            return item[name];
        }
    }
    return undefined;
};

const readString = (item: any, ...names: string[]) => String(readValue(item, ...names) ?? '');
const readNumber = (item: any, ...names: string[]) => Number(readValue(item, ...names) ?? 0) || 0;

const normalizeWalletType = (value: unknown): WalletType => {
    switch (String(value || '').toUpperCase()) {
        case '2':
        case 'SOLANA':
        case 'WALLET_TYPE_SOLANA':
            return 'SOLANA';
        default:
            return 'EVM';
    }
};

const normalizeWalletSource = (value: unknown): WalletSource => {
    switch (String(value || '').toLowerCase()) {
        case '2':
        case 'imported':
        case 'wallet_source_imported':
            return 'imported';
        default:
            return 'created';
    }
};

const normalizeAvatarKind = (value: unknown, avatarPresetId: string, avatarUrl: string): WalletAvatarKind => {
    switch (String(value || '').toLowerCase()) {
        case '2':
        case 'preset':
        case 'wallet_avatar_kind_preset':
            return 'preset';
        case '3':
        case 'upload':
        case 'uploaded':
        case 'wallet_avatar_kind_uploaded':
            return 'upload';
        default:
            return avatarUrl ? 'upload' : avatarPresetId ? 'preset' : 'default';
    }
};

export const normalizeWallet = (item: any = {}): WalletItem => {
    const avatarPresetId = readString(item, 'avatarPresetId', 'avatar_preset_id');
    const avatarUrl = readString(item, 'avatarUrl', 'avatar_url');
    return {
        id: readNumber(item, 'id'),
        walletType: normalizeWalletType(readValue(item, 'walletType', 'wallet_type')),
        address: readString(item, 'address'),
        remark: readString(item, 'remark'),
        source: normalizeWalletSource(readValue(item, 'source')),
        avatarKind: normalizeAvatarKind(readValue(item, 'avatarKind', 'avatar_kind'), avatarPresetId, avatarUrl),
        avatarPresetId,
        avatarUrl,
        revision: readNumber(item, 'revision'),
        createdAt: readString(item, 'createdAt', 'created_at'),
        updatedAt: readString(item, 'updatedAt', 'updated_at')
    };
};

const abortableRequest = <T>(req: any, map: (response: any) => T): AbortablePromise<T> => {
    const promise = req.then(map) as AbortablePromise<T>;
    promise.abort = () => req.abort();
    return promise;
};

const isRecord = (value: unknown): value is Record<string, unknown> => Boolean(value) && typeof value === 'object';

const parseJSONResponse = async (response: Response): Promise<Record<string, unknown>> => {
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

const rawAuthPost = <T>(path: string, body: Record<string, unknown>, map: (value: Record<string, unknown>) => T): AbortablePromise<T> => {
    const request = requests.scopedFetch(
        requests.toAbsURL(path),
        {
            method: 'POST',
            credentials: 'same-origin',
            headers: {'Accept': 'application/json', 'Content-Type': 'application/json'},
            body: JSON.stringify(body)
        },
        writeScope
    );
    const promise = request.then(async response => {
        const responseBody = await parseJSONResponse(response);
        if (!response.ok) {
            const nestedError = isRecord(responseBody.error) ? responseBody.error : undefined;
            const error = new Error(String(nestedError?.message || responseBody.message || response.statusText || 'Wallet reauthentication failed')) as Error & {
                status: number;
                body: Record<string, unknown>;
                response: {status: number; body: Record<string, unknown>; headers: Record<string, string>};
            };
            const reason = String(responseBody.reason || nestedError?.reason || response.headers.get('X-Athena-Error-Reason') || '');
            const headers = reason ? {'x-athena-error-reason': reason} : {};
            error.status = response.status;
            error.body = responseBody;
            error.response = {status: response.status, body: responseBody, headers};
            throw error;
        }
        return map(responseBody);
    }) as AbortablePromise<T>;
    promise.abort = request.abort;
    return promise;
};

export class WalletService {
    public listWallets(options: ListWalletsOptions = {}): AbortablePromise<ListWalletsResult> {
        const query: Record<string, string | number> = {
            page: options.page || 1,
            pageSize: options.pageSize || 12
        };
        if (options.walletType) {
            query.walletType = options.walletType;
        }
        if (options.query) {
            query.query = options.query;
        }
        const req = requests.get('/wallets', readScope).query(query);
        return abortableRequest(req, response => {
            const body = response.body || {};
            return {
                items: (body.items || []).map(normalizeWallet),
                total: readNumber(body, 'total'),
                page: readNumber(body, 'page') || Number(query.page),
                pageSize: readNumber(body, 'pageSize', 'page_size') || Number(query.pageSize)
            };
        });
    }

    public getWallet(id: number): AbortablePromise<WalletItem> {
        const req = requests.get(`/wallets/${encodeURIComponent(String(id))}`, readScope);
        return abortableRequest(req, response => normalizeWallet((response.body || {}).item || response.body || {}));
    }

    public createWallet(input: CreateWalletInput): AbortablePromise<CreateWalletResult> {
        const req = requests.post('/wallets', writeScope).send({
            walletType: input.walletType,
            remark: input.remark,
            avatarPresetId: input.avatarPresetId || ''
        });
        return abortableRequest(req, response => ({
            item: normalizeWallet((response.body || {}).item || {}),
            privateKey: readString(response.body, 'privateKey', 'private_key')
        }));
    }

    public importWallet(input: ImportWalletInput): AbortablePromise<WalletItem> {
        const req = requests.post('/wallets:import', writeScope).send({
            walletType: input.walletType,
            privateKey: input.privateKey,
            remark: input.remark,
            avatarPresetId: input.avatarPresetId || ''
        });
        return abortableRequest(req, response => normalizeWallet((response.body || {}).item || {}));
    }

    public updateRemark(id: number, remark: string, expectedRevision: number): AbortablePromise<WalletItem> {
        const req = requests.patch(`/wallets/${encodeURIComponent(String(id))}/remark`, writeScope).send({remark, expectedRevision});
        return abortableRequest(req, response => normalizeWallet((response.body || {}).item || {}));
    }

    public updateAvatarPreset(id: number, avatarPresetId: string, expectedRevision: number): AbortablePromise<WalletItem> {
        const req = requests.patch(`/wallets/${encodeURIComponent(String(id))}/avatar-preset`, writeScope).send({avatarPresetId, expectedRevision});
        return abortableRequest(req, response => normalizeWallet((response.body || {}).item || {}));
    }

    public uploadAvatar(id: number, file: File, expectedRevision: number): AbortablePromise<WalletItem> {
        const req = requests
            .rawPut(`/api/v1/wallets/${encodeURIComponent(String(id))}/avatar`, writeScope)
            .field('expectedRevision', String(expectedRevision))
            .attach('file', file as any);
        return abortableRequest(req, response => normalizeWallet((response.body || {}).item || response.body || {}));
    }

    public deleteAvatar(id: number, expectedRevision: number): AbortablePromise<WalletItem> {
        const req = requests.rawDelete(`/api/v1/wallets/${encodeURIComponent(String(id))}/avatar`, writeScope).query({expectedRevision});
        return abortableRequest(req, response => normalizeWallet((response.body || {}).item || response.body || {}));
    }

    public revealPrivateKey(id: number): AbortablePromise<string> {
        const req = requests.post(`/wallets/${encodeURIComponent(String(id))}:revealPrivateKey`, writeScope).send({});
        return abortableRequest(req, response => readString(response.body, 'privateKey', 'private_key'));
    }

    public googleWalletSecretReauthenticationURL(returnTo: string): string {
        const query = new URLSearchParams({returnTo});
        return `${requests.toAbsURL('/auth/wallet-secrets/google')}?${query.toString()}`;
    }

    public createSolanaWalletSecretChallenge(): AbortablePromise<WalletSecretChallenge> {
        return rawAuthPost('/auth/wallet-secrets/solana/challenge', {}, body => ({
            message: readString(body, 'message'),
            expiresAt: readNumber(body, 'expiresAt', 'expires_at')
        }));
    }

    public verifySolanaWalletSecretSignature(signature: string): AbortablePromise<WalletSecretLease> {
        return rawAuthPost('/auth/wallet-secrets/solana/verify', {signature}, body => ({expiresAt: readNumber(body, 'expiresAt', 'expires_at')}));
    }

    public createDevelopmentWalletSecretLease(): AbortablePromise<WalletSecretLease> {
        return rawAuthPost('/auth/wallet-secrets/development', {}, body => ({expiresAt: readNumber(body, 'expiresAt', 'expires_at')}));
    }
}
