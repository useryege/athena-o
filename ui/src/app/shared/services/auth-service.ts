import {AppBootstrap, AppBootstrapSessionStatus, parseAppBootstrapSessionStatus, parseUserInfo} from '../models';
import requests from './requests';

export class AuthService {
    public bootstrap(): Promise<AppBootstrap> {
        return requests.get('/app/bootstrap', {session: true}).then(res => {
            const settings = res.body?.settings;
            const session = res.body?.session;
            if (!settings || !session) {
                throw new Error('App bootstrap response is incomplete');
            }
            const status = parseAppBootstrapSessionStatus(session.status);
            switch (status) {
                case AppBootstrapSessionStatus.Anonymous:
                case AppBootstrapSessionStatus.AccountMaintenance:
                    return {settings, session: {status}};
                case AppBootstrapSessionStatus.Authenticated: {
                    const userInfo = session.userInfo || session.user_info;
                    if (!userInfo) {
                        throw new Error('Authenticated app bootstrap response is missing user info');
                    }
                    return {settings, session: {status, userInfo: parseUserInfo(userInfo)}};
                }
            }
        });
    }
}
