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
export const NotificationsDetailPage = React.lazy(() => import('./pages/notification-detail').then(module => ({default: module.NotificationsDetailPage})));
export const ProfitSharingRoundsPage = React.lazy(() => import('./pages/profit-sharing').then(module => ({default: module.ProfitSharingRoundsPage})));
export const ProfitSharingRoundPage = React.lazy(() => import('./pages/profit-sharing').then(module => ({default: module.ProfitSharingRoundPage})));
export const ProjectsPage = React.lazy(() => import('./pages/projects').then(module => ({default: module.ProjectsPage})));
export const ProjectDetailPage = React.lazy(() => import('./pages/project-detail').then(module => ({default: module.ProjectDetailPage})));
export const ContractCodesPage = React.lazy(() => import('./pages/contract-codes').then(module => ({default: module.ContractCodesPage})));
export const ContractCodeDetailPage = React.lazy(() => import('./pages/contract-code-detail').then(module => ({default: module.ContractCodeDetailPage})));
export const ContractCodeBlocklistPage = React.lazy(() => import('./pages/contract-code-blocklist').then(module => ({default: module.ContractCodeBlocklistPage})));
export const WalletBlocklistPage = React.lazy(() => import('./pages/wallet-blocklist').then(module => ({default: module.WalletBlocklistPage})));
export const NodeStatusesPage = React.lazy(() => import('./pages/node-statuses').then(module => ({default: module.NodeStatusesPage})));
export const ChainProcessingPage = React.lazy(() => import('./pages/chain-processing').then(module => ({default: module.ChainProcessingPage})));
export const CollectionTasksPage = React.lazy(() => import('./pages/collection-tasks').then(module => ({default: module.CollectionTasksPage})));
export const HelpPage = React.lazy(() => import('../shared/pages/help').then(module => ({default: module.HelpPage})));
