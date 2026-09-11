import * as React from 'react';

export const LoginPage = React.lazy(() => import('./pages/login').then(module => ({default: module.LoginPage})));
export const RegisterPage = React.lazy(() => import('./pages/register').then(module => ({default: module.RegisterPage})));
export const AccountCenterPage = React.lazy(() => import('../shared/pages/account-center').then(module => ({default: module.AccountCenterPage})));
export const AccountSecurityPage = React.lazy(() => import('./pages/account-security').then(module => ({default: module.AccountSecurityPage})));
export const WalletsPage = React.lazy(() => import('./pages/wallets').then(module => ({default: module.WalletsPage})));
export const WormTradingPage = React.lazy(() => import('./pages/worm-trading').then(module => ({default: module.WormTradingPage})));
export const WormTradingCombinationsPage = React.lazy(() => import('./pages/worm-trading-combinations').then(module => ({default: module.WormTradingCombinationsPage})));
export const WormTradingCombinationBuilderPage = React.lazy(() =>
    import('./pages/worm-trading-combinations').then(module => ({default: module.WormTradingCombinationBuilderPage}))
);
export const WormTradingExecutionPreviewPage = React.lazy(() =>
    import('./pages/worm-trading-execution-preview').then(module => ({default: module.WormTradingExecutionPreviewPage}))
);
export const WormTradingExecutionsPage = React.lazy(() => import('./pages/worm-trading-executions').then(module => ({default: module.WormTradingExecutionsPage})));
export const WormTradingExecutionDetailPage = React.lazy(() => import('./pages/worm-trading-executions').then(module => ({default: module.WormTradingExecutionDetailPage})));
export const MarketRadarHotPage = React.lazy(() => import('./pages/market-radar').then(module => ({default: module.MarketRadarHotPage})));
export const MarketRadarRealtimePage = React.lazy(() => import('./pages/market-radar').then(module => ({default: module.MarketRadarRealtimePage})));
export const MarketRadarMoversPage = React.lazy(() => import('./pages/market-radar').then(module => ({default: module.MarketRadarMoversPage})));
export const SportsLivePage = React.lazy(() => import('./pages/sports-live').then(module => ({default: module.SportsLivePage})));
export const SportsHistoryPage = React.lazy(() => import('./pages/sports-history').then(module => ({default: module.SportsHistoryPage})));
export const WorldCupCornersPage = React.lazy(() => import('./pages/world-cup-corners').then(module => ({default: module.WorldCupCornersPage})));
export const ManagedOOProposalsPage = React.lazy(() => import('./pages/managed-oo').then(module => ({default: module.ManagedOOProposalsPage})));
export const ManagedOODisputesPage = React.lazy(() => import('./pages/managed-oo').then(module => ({default: module.ManagedOODisputesPage})));
export const NotificationsPage = React.lazy(() => import('./pages/notifications').then(module => ({default: module.NotificationsPage})));
export const ProfitSharingRoundsPage = React.lazy(() => import('./pages/profit-sharing').then(module => ({default: module.ProfitSharingRoundsPage})));
export const ProfitSharingRoundPage = React.lazy(() => import('./pages/profit-sharing').then(module => ({default: module.ProfitSharingRoundPage})));
export const HelpPage = React.lazy(() => import('../shared/pages/help').then(module => ({default: module.HelpPage})));

export const TraderSyncAddPage = React.lazy(() => import('./pages/trader-sync/add').then(module => ({default: module.TraderSyncAddPage})));

export const TraderSyncSubscriptionsPage = React.lazy(() => import('./pages/trader-sync/subscriptions').then(module => ({default: module.TraderSyncSubscriptionsPage})));
export const TraderSyncSubscriptionPage = React.lazy(() => import('./pages/trader-sync/subscription-detail').then(module => ({default: module.TraderSyncSubscriptionPage})));

export const TraderSyncHomePage = React.lazy(() => import('./pages/trader-sync/home').then(module => ({default: module.TraderSyncHomePage})));

export const TraderSyncActivityPage = React.lazy(() => import('./pages/trader-sync/activity-detail').then(module => ({default: module.TraderSyncActivityPage})));
export const TraderSyncSummaryPage = React.lazy(() => import('./pages/trader-sync/summary-detail').then(module => ({default: module.TraderSyncSummaryPage})));
