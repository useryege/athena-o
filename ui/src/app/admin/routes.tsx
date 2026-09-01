import * as React from 'react';

export const AdminLoginPage = React.lazy(() => import('./login').then(module => ({default: module.AdminLoginPage})));
export const AccountCenterPage = React.lazy(() => import('../shared/pages/account-center').then(module => ({default: module.AccountCenterPage})));
export const AdminAccountsPage = React.lazy(() => import('./pages/admin-accounts').then(module => ({default: module.AdminAccountsPage})));
export const ProfitSharingAdminRoundsPage = React.lazy(() => import('./pages/profit-sharing-admin').then(module => ({default: module.ProfitSharingAdminRoundsPage})));
export const ProfitSharingAdminRoundPage = React.lazy(() => import('./pages/profit-sharing-admin').then(module => ({default: module.ProfitSharingAdminRoundPage})));
export const ServiceStatusPage = React.lazy(() => import('./pages/service-status').then(module => ({default: module.ServiceStatusPage})));
export const EtherscanGatewaysPage = React.lazy(() => import('./pages/etherscan-gateways').then(module => ({default: module.EtherscanGatewaysPage})));
export const HelpPage = React.lazy(() => import('../shared/pages/help').then(module => ({default: module.HelpPage})));
