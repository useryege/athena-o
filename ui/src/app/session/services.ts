import type {SelfAccountService} from '../shared/services/accounts-service';
import type {AuthService} from '../shared/services/auth-service';
import {serviceProjection} from '../shared/services/registry';
import type {UserService} from '../shared/services/user-service';
import type {VersionService} from '../shared/services/version-service';
import type {ViewPreferencesService} from '../shared/services/view-preferences-service';

export interface SessionServices {
    users: UserService;
    authService: AuthService;
    viewPreferences: ViewPreferencesService;
}

export interface SelfAccountServices extends SessionServices {
    version: VersionService;
    accounts: SelfAccountService;
}

export const sessionServices = serviceProjection<SessionServices>();
export const selfAccountServices = serviceProjection<SelfAccountServices>();
