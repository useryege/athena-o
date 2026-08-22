import {Account, AccountAccess, parseAccountAccess} from '../models';
import requests from './requests';

const account = (value: any): Account => ({
    name: value?.name || '',
    administrator: Boolean(value?.administrator),
    access: parseAccountAccess(value?.access),
    capabilities: value?.capabilities || [],
    tokens: value?.tokens || []
});

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

    public changePassword(name: string, currentPassword: string, newPassword: string): Promise<boolean> {
        return requests
            .put('/account/password')
            .send({currentPassword, name, newPassword})
            .then(res => res.status === 200);
    }

    public createToken(name: string, tokenId: string, expiresIn: number): Promise<string> {
        return requests
            .post(`/account/${name}/token`)
            .send({expiresIn, id: tokenId})
            .then(res => res.body.token as string);
    }

    public deleteToken(name: string, id: string): Promise<any> {
        return requests.delete(`/account/${name}/token/${id}`);
    }
}
