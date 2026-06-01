import {AccountsService} from './accounts-service';
import {AthenaApplicationService} from './athena-application-service';
import {AthenaSolidityService} from './athena-solidity-service';
import {AuthService} from './auth-service';
import {NotificationService} from './notification-service';
import {PolymarketService} from './polymarket-service';
import {UserService} from './user-service';
import {VersionService} from './version-service';
import {ViewPreferencesService} from './view-preferences-service';
import {WalletService} from './wallet-service';
import {WormService} from './worm-service';

export interface Services {
    athenaApplication: AthenaApplicationService;
    athenaSolidity: AthenaSolidityService;
    users: UserService;
    authService: AuthService;
    viewPreferences: ViewPreferencesService;
    version: VersionService;
    accounts: AccountsService;
    wallet: WalletService;
    worm: WormService;
    polymarket: PolymarketService;
    notification: NotificationService;
}

export const services: Services = {
    athenaApplication: new AthenaApplicationService(),
    athenaSolidity: new AthenaSolidityService(),
    authService: new AuthService(),
    users: new UserService(),
    viewPreferences: new ViewPreferencesService(),
    version: new VersionService(),
    accounts: new AccountsService(),
    wallet: new WalletService(),
    worm: new WormService(),
    polymarket: new PolymarketService(),
    notification: new NotificationService()
};

export * from './view-preferences-service';
