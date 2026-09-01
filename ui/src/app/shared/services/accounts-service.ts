import {AccountPreferences, AccountProfile, AccountThemeMode, parseAccountPreferences, parseAccountProfile} from '../models';
import requests from './requests';

const selfAccountWriteScope = {feature: 'self-account' as const, mode: 'write' as const};

const avatarProfile = (value: any, username: string): AccountProfile => parseAccountProfile(value?.profile || value, username);
const accountThemeAPIValue = (theme: AccountThemeMode) => {
    if (theme === AccountThemeMode.Dark) {
        return 3;
    }
    if (theme === AccountThemeMode.Light) {
        return 2;
    }
    return 1;
};

export class SelfAccountService {
    public updateProfile(id: string, displayName: string, expectedRevision: number): Promise<AccountProfile> {
        return requests
            .put(`/account/${encodeURIComponent(id)}/profile`, selfAccountWriteScope)
            .send({displayName, expectedRevision})
            .then(res => parseAccountProfile(res.body));
    }

    public updatePreferences(theme: AccountThemeMode, expectedRevision: number): Promise<AccountPreferences> {
        return requests
            .put('/account/preferences', selfAccountWriteScope)
            .send({theme: accountThemeAPIValue(theme), expectedRevision})
            .then(res => parseAccountPreferences(res.body));
    }

    public uploadAvatar(id: string, username: string, file: File, expectedRevision: number): Promise<AccountProfile> {
        return requests
            .rawPut(`/api/v1/account/${encodeURIComponent(id)}/avatar`, selfAccountWriteScope)
            .field('expectedRevision', String(expectedRevision))
            .attach('file', file as any)
            .then(res => avatarProfile(res.body, username));
    }

    public deleteAvatar(id: string, username: string, expectedRevision: number): Promise<AccountProfile> {
        return requests
            .rawDelete(`/api/v1/account/${encodeURIComponent(id)}/avatar`, selfAccountWriteScope)
            .query({expectedRevision})
            .then(res => avatarProfile(res.body, username));
    }
}
