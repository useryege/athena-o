import {SelfAccountService} from '../shared/services/accounts-service';
import {AuthService} from '../shared/services/auth-service';
import {ManagedOOService} from '../shared/services/managed-oo-service';
import {MarketRadarService} from '../shared/services/market-radar-service';
import {SportsHistoryService} from '../shared/services/sports-history-service';
import {SportsLiveService} from '../shared/services/sports-live-service';
import {configureServices, serviceProjection} from '../shared/services/registry';
import {UserService} from '../shared/services/user-service';
import {VersionService} from '../shared/services/version-service';
import {ViewPreferencesService} from '../shared/services/view-preferences-service';
import {WalletService} from '../shared/services/wallet-service';
import {WorldCupCornersService} from '../shared/services/world-cup-corners-service';
import {WormTradingService} from '../shared/services/worm-trading-service';
import type {SelfAccountServices} from '../session/services';
import {MemberNotificationService} from './notification-service';
import {MemberProfitSharingService} from './profit-sharing-service';
import {MemberSecurityService} from './security-service';
import {MemberTraderSyncService} from './trader-sync-service';

export interface MemberServices extends SelfAccountServices {
    memberSecurity: MemberSecurityService;
    traderSync: MemberTraderSyncService;
    wallet: WalletService;
    wormTrading: WormTradingService;
    marketRadar: MarketRadarService;
    sportsLive: SportsLiveService;
    sportsHistory: SportsHistoryService;
    managedOO: ManagedOOService;
    memberNotifications: MemberNotificationService;
    worldCupCorners: WorldCupCornersService;
    memberProfitSharing: MemberProfitSharingService;
}

let businessServicesConfigured = false;

export const memberServices = serviceProjection<MemberServices>();

export const configureMemberSessionServices = () =>
    configureServices('member', {
        authService: new AuthService(),
        users: new UserService(),
        viewPreferences: new ViewPreferencesService('athena.member.preferences')
    });

export const ensureMemberBusinessServices = () => {
    if (businessServicesConfigured) {
        return;
    }
    businessServicesConfigured = true;
    configureServices('member', {
        version: new VersionService(),
        accounts: new SelfAccountService(),
        memberSecurity: new MemberSecurityService(),
        traderSync: new MemberTraderSyncService(),
        wallet: new WalletService(),
        wormTrading: new WormTradingService(),
        marketRadar: new MarketRadarService(),
        sportsLive: new SportsLiveService(),
        sportsHistory: new SportsHistoryService(),
        managedOO: new ManagedOOService(),
        memberNotifications: new MemberNotificationService(),
        worldCupCorners: new WorldCupCornersService(),
        memberProfitSharing: new MemberProfitSharingService()
    });
};
