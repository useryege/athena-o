import {SelfAccountService} from '../shared/services/accounts-service';
import {AuthService} from '../shared/services/auth-service';
import {ServiceStatusService} from '../shared/services/service-status-service';
import {configureServices, serviceProjection} from '../shared/services/registry';
import {UserService} from '../shared/services/user-service';
import {VersionService} from '../shared/services/version-service';
import {ViewPreferencesService} from '../shared/services/view-preferences-service';
import type {SelfAccountServices} from '../session/services';
import {AdminAccountsService} from './accounts-service';
import {AdminProfitSharingService} from './profit-sharing-service';

export interface AdminServices extends SelfAccountServices {
    adminAccounts: AdminAccountsService;
    serviceStatus: ServiceStatusService;
    adminProfitSharing: AdminProfitSharingService;
}

let businessServicesConfigured = false;

export const adminServices = serviceProjection<AdminServices>();

export const configureAdminSessionServices = () =>
    configureServices('admin', {
        authService: new AuthService(),
        users: new UserService(),
        viewPreferences: new ViewPreferencesService('athena.admin.preferences')
    });

export const ensureAdminBusinessServices = () => {
    if (businessServicesConfigured) {
        return;
    }
    businessServicesConfigured = true;
    configureServices('admin', {
        version: new VersionService(),
        accounts: new SelfAccountService(),
        adminAccounts: new AdminAccountsService(),
        serviceStatus: new ServiceStatusService(),
        adminProfitSharing: new AdminProfitSharingService()
    });
};
