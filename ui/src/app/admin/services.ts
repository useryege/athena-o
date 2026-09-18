import {SelfAccountService} from '../shared/services/accounts-service';
import {AuthService} from '../shared/services/auth-service';
import {ServiceStatusService} from '../shared/services/service-status-service';
import {configureServices, serviceProjection} from '../shared/services/registry';
import {UserService} from '../shared/services/user-service';
import {VersionService} from '../shared/services/version-service';
import {ViewPreferencesService} from '../shared/services/view-preferences-service';
import type {SelfAccountServices} from '../session/services';
import {AdminTraderSyncService} from './trader-sync-service';
import {AdminAccountsService} from './accounts-service';
import {AdminNotificationService} from './notification-service';
import {AdminProfitSharingService} from './profit-sharing-service';
import {OperationLogService} from './operation-log-service';

export interface AdminServices extends SelfAccountServices {
    adminTraderSync: AdminTraderSyncService;
    adminAccounts: AdminAccountsService;
    serviceStatus: ServiceStatusService;
    adminProfitSharing: AdminProfitSharingService;
    adminNotifications: AdminNotificationService;
    operationLogs: OperationLogService;
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
        adminTraderSync: new AdminTraderSyncService(),
        serviceStatus: new ServiceStatusService(),
        adminProfitSharing: new AdminProfitSharingService(),
        adminNotifications: new AdminNotificationService(),
        operationLogs: new OperationLogService()
    });
};
