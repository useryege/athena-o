import {AccountsService} from './accounts-service';
import {AuthService} from './auth-service';
import {NotificationService} from './notification-service';
import {PolymarketService} from './polymarket-service';
import {ServiceStatusService} from './service-status-service';
import {TokenAPIService} from './tokenapi-service';
import {UserService} from './user-service';
import {VersionService} from './version-service';
import {ViewPreferencesService} from './view-preferences-service';
import {WalletService} from './wallet-service';
import {WormPolyService} from './wormpoly-service';
import {WormService} from './worm-service';

export interface Services {
    tokenapi: TokenAPIService;
    users: UserService;
    authService: AuthService;
    viewPreferences: ViewPreferencesService;
    version: VersionService;
    accounts: AccountsService;
    wallet: WalletService;
    worm: WormService;
    wormpoly: WormPolyService;
    polymarket: PolymarketService;
    notification: NotificationService;
    serviceStatus: ServiceStatusService;
}

export const services: Services = {
    tokenapi: new TokenAPIService(),
    authService: new AuthService(),
    users: new UserService(),
    viewPreferences: new ViewPreferencesService(),
    version: new VersionService(),
    accounts: new AccountsService(),
    wallet: new WalletService(),
    worm: new WormService(),
    wormpoly: new WormPolyService(),
    polymarket: new PolymarketService(),
    notification: new NotificationService(),
    serviceStatus: new ServiceStatusService()
};

export * from './service-status-service';
export * from './view-preferences-service';
