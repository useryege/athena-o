import {UserInfo} from '../models';
import requests from './requests';

export class UserService {
    public login(username: string, password: string): Promise<{token: string}> {
        return requests
            .post('/session')
            .send({username, password})
            .then(res => ({token: res.body.token}));
    }

    public logout(): Promise<boolean> {
        return fetch(requests.toAbsURL('/auth/logout'), {credentials: 'same-origin', redirect: 'manual'}).then(res => {
            if (res.status >= 400) {
                throw new Error(res.statusText || `Logout failed (${res.status})`);
            }
            return true;
        });
    }

    public get(): Promise<UserInfo> {
        return requests.get('/session/userinfo').then(res => res.body as UserInfo);
    }
}
