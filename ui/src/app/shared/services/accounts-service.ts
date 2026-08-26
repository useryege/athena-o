import {
    Account,
    AccountAccess,
    AccountStatus,
    AccountPreferences,
    AccountProfile,
    AccountThemeMode,
    AccountTier,
    parseAccountAccess,
    parseAccountIdentity,
    parseAccountStatus,
    parseAccountPreferences,
    parseAccountProfile,
    Token
} from '../models';
import requests from './requests';

const apiKeyReadScope = {feature: 'api-key' as const, mode: 'read' as const};
const apiKeyWriteScope = {feature: 'api-key' as const, mode: 'write' as const};

const account = (value: any): Account => ({
    id: String(value?.id || ''),
    username: String(value?.username || ''),
    administrator: Boolean(value?.administrator),
    access: parseAccountAccess(value?.access),
    profile: parseAccountProfile(value?.profile, value?.username || ''),
    identity: parseAccountIdentity(value?.identity),
    status: parseAccountStatus(value?.status)
});

export interface AccountListOptions {
    query?: string;
    status?: AccountStatus;
    page?: number;
    pageSize?: number;
    profitSharingEligibleOnly?: boolean;
}

export interface AccountsPage {
    items: Account[];
    totalSize: number;
}

const token = (value: any): Token => ({
    id: String(value?.id || ''),
    issuedAt: Number(value?.issuedAt ?? value?.issued_at ?? 0),
    expiresAt: Number(value?.expiresAt ?? value?.expires_at ?? 0)
});

const avatarProfile = (value: any, username: string): AccountProfile => parseAccountProfile(value?.profile || value, username);
const accountTierAPIValue = (tier: AccountTier) => (tier === AccountTier.Pro ? 2 : 1);
const accountThemeAPIValue = (theme: AccountThemeMode) => {
    if (theme === AccountThemeMode.Dark) {
        return 3;
    }
    if (theme === AccountThemeMode.Light) {
        return 2;
    }
    return 1;
};

export class AccountsService {
    public list(options: AccountListOptions = {}): Promise<AccountsPage> {
        return requests
            .get('/account')
            .query({
                query: options.query || undefined,
                status: options.status === AccountStatus.Unspecified ? undefined : options.status,
                page: options.page || 1,
                pageSize: options.pageSize || 50,
                profitSharingEligibleOnly: options.profitSharingEligibleOnly || undefined
            })
            .then(res => ({items: (res.body?.items || []).map(account), totalSize: Number(res.body?.totalSize ?? res.body?.total_size ?? 0)}));
    }

    public get(id: string): Promise<Account> {
        return requests.get(`/account/${encodeURIComponent(id)}`).then(res => account(res.body));
    }

    public updateAccess(id: string, access: AccountAccess): Promise<Account> {
        return requests
            .put(`/account/${encodeURIComponent(id)}/access`)
            .send({
                loginEnabled: access.loginEnabled,
                apiKeyEnabled: access.apiKeyEnabled,
                profitSharingEnabled: access.profitSharingEnabled,
                revision: access.revision,
                moduleAccess: access.moduleAccess.map(item => ({module: item.module, dataAccess: item.dataAccess}))
            })
            .then(res => account(res.body));
    }

    public updateProfile(id: string, displayName: string, expectedRevision: number): Promise<AccountProfile> {
        return requests
            .put(`/account/${encodeURIComponent(id)}/profile`)
            .send({displayName, expectedRevision})
            .then(res => parseAccountProfile(res.body));
    }

    public updateTier(id: string, tier: AccountTier, expectedRevision: number): Promise<AccountProfile> {
        return requests
            .put(`/account/${encodeURIComponent(id)}/tier`)
            .send({tier: accountTierAPIValue(tier), expectedRevision})
            .then(res => parseAccountProfile(res.body));
    }

    public updatePreferences(theme: AccountThemeMode, expectedRevision: number): Promise<AccountPreferences> {
        return requests
            .put('/account/preferences')
            .send({theme: accountThemeAPIValue(theme), expectedRevision})
            .then(res => parseAccountPreferences(res.body));
    }

    public listTokens(): Promise<Token[]> {
        return requests.get('/account/security/tokens', apiKeyReadScope).then(res => (res.body?.items || []).map(token));
    }

    public createToken(tokenId: string, expiresIn: number): Promise<string> {
        return requests
            .post('/account/security/tokens', apiKeyWriteScope)
            .send({expiresIn, id: tokenId})
            .then(res => String(res.body?.token || ''));
    }

    public deleteToken(id: string): Promise<void> {
        return requests.delete(`/account/security/tokens/${encodeURIComponent(id)}`, apiKeyWriteScope).then(() => undefined);
    }

    public uploadAvatar(id: string, username: string, file: File, expectedRevision: number): Promise<AccountProfile> {
        return requests
            .rawPut(`/api/v1/account/${encodeURIComponent(id)}/avatar`)
            .field('expectedRevision', String(expectedRevision))
            .attach('file', file as any)
            .then(res => avatarProfile(res.body, username));
    }

    public deleteAvatar(id: string, username: string, expectedRevision: number): Promise<AccountProfile> {
        return requests
            .rawDelete(`/api/v1/account/${encodeURIComponent(id)}/avatar`)
            .query({expectedRevision})
            .then(res => avatarProfile(res.body, username));
    }
}
