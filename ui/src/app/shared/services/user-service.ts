import {parseUserInfo, UserInfo} from '../models';
import requests from './requests';

export class UserService {
    public logout(): Promise<boolean> {
        return fetch(requests.toAbsURL('/auth/logout'), {credentials: 'same-origin', redirect: 'manual'}).then(res => {
            if (res.status >= 400) {
                throw new Error(res.statusText || `Logout failed (${res.status})`);
            }
            return true;
        });
    }

    public get(): Promise<UserInfo> {
        return requests.get('/session/userinfo').then(res => parseUserInfo(res.body));
    }
}
