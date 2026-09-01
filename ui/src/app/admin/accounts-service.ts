import {
    type Account,
    type AccountAccess,
    AccountStatus,
    type AccountProfile,
    AccountTier,
    parseAccountAccess,
    parseAccountIdentity,
    parseAccountStatus,
    parseAccountProfile
} from '../shared/models';
import requests from '../shared/services/requests';

type AbortablePromise<T> = Promise<T> & {abort?: () => void};

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

const avatarProfile = (value: any, username: string): AccountProfile => parseAccountProfile(value?.profile || value, username);
const accountTierAPIValue = (tier: AccountTier) => (tier === AccountTier.Pro ? 2 : 1);

export class AdminAccountsService {
    private readonly readScope = {feature: 'admin-accounts' as const, mode: 'read' as const};
    private readonly writeScope = {feature: 'admin-accounts' as const, mode: 'write' as const};

    public list(options: AccountListOptions = {}): AbortablePromise<AccountsPage> {
        const req = requests.get('/account', this.readScope).query({
            query: options.query || undefined,
            status: options.status === AccountStatus.Unspecified ? undefined : options.status,
            page: options.page || 1,
            pageSize: options.pageSize || 50,
            profitSharingEligibleOnly: options.profitSharingEligibleOnly || undefined
        });
        const promise = req.then(res => ({
            items: (res.body?.items || []).map(account),
            totalSize: Number(res.body?.totalSize ?? res.body?.total_size ?? 0)
        })) as AbortablePromise<AccountsPage>;
        promise.abort = () => req.abort();
        return promise;
    }

    public get(id: string): AbortablePromise<Account> {
        const req = requests.get(`/account/${encodeURIComponent(id)}`, this.readScope);
        const promise = req.then(res => account(res.body)) as AbortablePromise<Account>;
        promise.abort = () => req.abort();
        return promise;
    }

    public updateProfile(id: string, displayName: string, expectedRevision: number): Promise<AccountProfile> {
        return requests
            .put(`/account/${encodeURIComponent(id)}/profile`, this.writeScope)
            .send({displayName, expectedRevision})
            .then(res => parseAccountProfile(res.body));
    }

    public updateAccess(id: string, access: AccountAccess): Promise<Account> {
        return requests
            .put(`/account/${encodeURIComponent(id)}/access`, this.writeScope)
            .send({
                loginEnabled: access.loginEnabled,
                apiKeyEnabled: access.apiKeyEnabled,
                profitSharingEnabled: access.profitSharingEnabled,
                revision: access.revision,
                moduleAccess: access.moduleAccess.map(item => ({module: item.module, dataAccess: item.dataAccess}))
            })
            .then(res => account(res.body));
    }

    public updateTier(id: string, tier: AccountTier, expectedRevision: number): Promise<AccountProfile> {
        return requests
            .put(`/account/${encodeURIComponent(id)}/tier`, this.writeScope)
            .send({tier: accountTierAPIValue(tier), expectedRevision})
            .then(res => parseAccountProfile(res.body));
    }

    public uploadAvatar(id: string, username: string, file: File, expectedRevision: number): Promise<AccountProfile> {
        return requests
            .rawPut(`/api/v1/account/${encodeURIComponent(id)}/avatar`, this.writeScope)
            .field('expectedRevision', String(expectedRevision))
            .attach('file', file as any)
            .then(res => avatarProfile(res.body, username));
    }

    public deleteAvatar(id: string, username: string, expectedRevision: number): Promise<AccountProfile> {
        return requests
            .rawDelete(`/api/v1/account/${encodeURIComponent(id)}/avatar`, this.writeScope)
            .query({expectedRevision})
            .then(res => avatarProfile(res.body, username));
    }
}
