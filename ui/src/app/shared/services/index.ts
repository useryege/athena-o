import {AccountsService} from './accounts-service';
import {AuthService} from './auth-service';
import {NotificationService} from './notification-service';
import {ManagedOOService} from './managed-oo-service';
import {MarketRadarService} from './market-radar-service';
import {ServiceStatusService} from './service-status-service';
import {TokenService} from './token-service';
import {UserService} from './user-service';
import {VersionService} from './version-service';
import {ViewPreferencesService} from './view-preferences-service';
import {WalletService} from './wallet-service';
import {WormTradingService} from './worm-trading-service';
import {SportsHistoryService} from './sports-history-service';
import {SportsLiveService} from './sports-live-service';
import {WorldCupCornersService} from './world-cup-corners-service';
import {ProfitSharingService} from './profit-sharing-service';

export interface Services {
    tokenapi: TokenService;
    users: UserService;
    authService: AuthService;
    viewPreferences: ViewPreferencesService;
    version: VersionService;
    accounts: AccountsService;
    wallet: WalletService;
    wormTrading: WormTradingService;
    marketRadar: MarketRadarService;
    sportsLive: SportsLiveService;
    sportsHistory: SportsHistoryService;
    managedOO: ManagedOOService;
    notification: NotificationService;
    serviceStatus: ServiceStatusService;
    worldCupCorners: WorldCupCornersService;
    profitSharing: ProfitSharingService;
}

export const services: Services = {
    tokenapi: new TokenService(),
    authService: new AuthService(),
    users: new UserService(),
    viewPreferences: new ViewPreferencesService(),
    version: new VersionService(),
    accounts: new AccountsService(),
    wallet: new WalletService(),
    wormTrading: new WormTradingService(),
    marketRadar: new MarketRadarService(),
    sportsLive: new SportsLiveService(),
    sportsHistory: new SportsHistoryService(),
    managedOO: new ManagedOOService(),
    notification: new NotificationService(),
    serviceStatus: new ServiceStatusService(),
    worldCupCorners: new WorldCupCornersService(),
    profitSharing: new ProfitSharingService()
};

export * from './service-status-service';
export * from './view-preferences-service';
export * from './world-cup-corners-service';
export * from './profit-sharing-service';
export * from './worm-trading-service';
