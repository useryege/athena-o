import {
    Account,
    AccountAccess,
    AccountPreferences,
    AccountProfile,
    AccountThemeMode,
    AccountTier,
    parseAccountAccess,
    parseAccountPreferences,
    parseAccountProfile,
    Token
} from '../models';
import requests from './requests';

const account = (value: any): Account => ({
    name: value?.name || '',
    administrator: Boolean(value?.administrator),
    access: parseAccountAccess(value?.access),
    capabilities: value?.capabilities || [],
    profile: parseAccountProfile(value?.profile, value?.name || '')
});

const token = (value: any): Token => ({
    id: String(value?.id || ''),
    issuedAt: Number(value?.issuedAt ?? value?.issued_at ?? 0),
    expiresAt: Number(value?.expiresAt ?? value?.expires_at ?? 0)
});

const avatarProfile = (value: any, name: string): AccountProfile => parseAccountProfile(value?.profile || value, name);
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
    public list(): Promise<Account[]> {
        return requests.get('/account').then(res => (res.body.items || []).map(account));
    }

    public get(name: string): Promise<Account> {
        return requests.get(`/account/${encodeURIComponent(name)}`).then(res => account(res.body));
    }

    public updateAccess(name: string, access: AccountAccess): Promise<Account> {
        return requests
            .put(`/account/${encodeURIComponent(name)}/access`)
            .send({
                loginEnabled: access.loginEnabled,
                revision: access.revision,
                moduleAccess: access.moduleAccess.map(item => ({module: item.module, dataAccess: item.dataAccess}))
            })
            .then(res => account(res.body));
    }

    public updateProfile(name: string, displayName: string, expectedRevision: number): Promise<AccountProfile> {
        return requests
            .put(`/account/${encodeURIComponent(name)}/profile`)
            .send({displayName, expectedRevision})
            .then(res => parseAccountProfile(res.body, name));
    }

    public updateTier(name: string, tier: AccountTier, expectedRevision: number): Promise<AccountProfile> {
        return requests
            .put(`/account/${encodeURIComponent(name)}/tier`)
            .send({tier: accountTierAPIValue(tier), expectedRevision})
            .then(res => parseAccountProfile(res.body, name));
    }

    public updatePreferences(theme: AccountThemeMode, expectedRevision: number): Promise<AccountPreferences> {
        return requests
            .put('/account/preferences')
            .send({theme: accountThemeAPIValue(theme), expectedRevision})
            .then(res => parseAccountPreferences(res.body));
    }

    public listTokens(): Promise<Token[]> {
        return requests.get('/account/security/tokens').then(res => (res.body?.items || []).map(token));
    }

    public createToken(tokenId: string, expiresIn: number): Promise<string> {
        return requests
            .post('/account/security/tokens')
            .send({expiresIn, id: tokenId})
            .then(res => String(res.body?.token || ''));
    }

    public deleteToken(id: string): Promise<void> {
        return requests.delete(`/account/security/tokens/${encodeURIComponent(id)}`).then(() => undefined);
    }

    public uploadAvatar(name: string, file: File, expectedRevision: number): Promise<AccountProfile> {
        return requests
            .rawPut(`/api/v1/account/${encodeURIComponent(name)}/avatar`)
            .field('expectedRevision', String(expectedRevision))
            .attach('file', file as any)
            .then(res => avatarProfile(res.body, name));
    }

    public deleteAvatar(name: string, expectedRevision: number): Promise<AccountProfile> {
        return requests
            .rawDelete(`/api/v1/account/${encodeURIComponent(name)}/avatar`)
            .query({expectedRevision})
            .then(res => avatarProfile(res.body, name));
    }
}
