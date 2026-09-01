import {parseUserInfo, UserInfo} from '../models';
import requests from './requests';

export class UserService {
    public logout(): Promise<boolean> {
        return requests.scopedFetch(requests.toAbsURL('/auth/logout'), {credentials: 'same-origin', redirect: 'manual'}, {session: true}).then(res => {
            if (res.status >= 400) {
                throw new Error(res.statusText || `Logout failed (${res.status})`);
            }
            return true;
        });
    }

    public get(): Promise<UserInfo> {
        return requests.get('/session/userinfo', {session: true}).then(res => parseUserInfo(res.body));
    }
}
