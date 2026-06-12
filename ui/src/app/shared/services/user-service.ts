import {UserInfo} from '../models';
import requests from './requests';

export interface CaptchaChallenge {
    captchaId: string;
    imageDataUrl: string;
    expiresIn: number;
}

export class UserService {
    public getCaptcha(): Promise<CaptchaChallenge> {
        return requests.get('/session/captcha').then(res => ({
            captchaId: res.body.captcha_id,
            imageDataUrl: res.body.image_data_url,
            expiresIn: res.body.expires_in
        }));
    }

    public login(username: string, password: string, captchaId: string, captchaAnswer: string): Promise<{token: string}> {
        return requests
            .post('/session')
            .send({username, password, captcha_id: captchaId, captcha_answer: captchaAnswer})
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
