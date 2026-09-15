import '@fortawesome/fontawesome-free/css/all.css';
import 'antd/dist/reset.css';
import '../../assets/fonts.css';
import '../styles/member.css';

import {
    ApiOutlined,
    BarChartOutlined,
    BellOutlined,
    DashboardOutlined,
    FileTextOutlined,
    HistoryOutlined,
    IdcardOutlined,
    KeyOutlined,
    LogoutOutlined,
    MenuFoldOutlined,
    MenuUnfoldOutlined,
    PieChartOutlined,
    QuestionCircleOutlined,
    SettingOutlined,
    SwapOutlined,
    TrophyOutlined,
    WalletOutlined
} from '@ant-design/icons';
import {App as AntApp, Breadcrumb, Button, Dropdown, Layout as AntLayout, Menu, Result, Space, Spin, Tag, Tooltip, Typography} from 'antd';
import type {MenuProps} from 'antd';
import * as React from 'react';
import {createBrowserRouter, Navigate, Route, RouterProvider, Routes, useLocation, useNavigate} from 'react-router-dom';
import {Subscription} from 'rxjs';
import {AuthorizationCtx, Provider} from '../shared/context';
import {AccountDataAccess, AccountDataModule, accountDataModules} from '../shared/access-modules';
import {moduleAccessLevels, moduleAccessLevelsEqual, ModuleAccessLevels} from '../shared/account-access';
import {accountStatusForAccess, AccountStatus, AppBootstrap, AppBootstrapSession, AppBootstrapSessionStatus, AuthSettings, UserInfo} from '../shared/models';
import requests, {
    isAccountDataAccessDeniedError,
    isAccountProfitSharingAccessDeniedError,
    isAccountMaintenanceError,
    requestErrorDetails,
    requestErrorMessage
} from '../shared/services/requests';
import type {ViewPreferences} from '../shared/services/view-preferences-service';
import {WALLET_REAUTH_REQUIRED} from '../shared/services/wallet-service';
import {loginPathFor, readLoginReturnTo} from '../shared/login-navigation';
import {deploymentPath, readApplicationBaseHRef, readDeploymentBaseHRef} from '../shared/runtime-base';
import {BrandMark, clearAsyncDataCache, setAsyncDataCacheSession} from '../components';
import {AccountAvatar, accountTierLabel} from '../shared/account-presentation';
import {configureMemberSessionServices, ensureMemberBusinessServices, memberServices as services} from './services';
import {clearTraderSyncState} from './pages/trader-sync/state';
import {clearTelegramBindingInstructions} from './notification-storage';
import {loadAppBootstrapWithRetry, SessionBootstrap} from '../session/bootstrap';
import {
    AccountCenterPage,
    AccountSecurityPage,
    HelpPage,
    NotificationsPage,
    TraderSyncHomePage,
    TraderSyncActivityPage,
    TraderSyncSummaryPage,
    TraderSyncAddPage,
    TraderSyncSubscriptionsPage,
    TraderSyncSubscriptionPage,
    MarketRadarHotPage,
    MarketRadarMoversPage,
    MarketRadarRealtimePage,
    SportsLivePage,
    SportsHistoryPage,
    SolanaPage,
    ManagedOODisputesPage,
    ManagedOOProposalsPage,
    WalletsPage,
    WormTradingCombinationBuilderPage,
    WormTradingCombinationsPage,
    WormTradingExecutionPreviewPage,
    WormTradingExecutionDetailPage,
    WormTradingExecutionsPage,
    WormTradingPage,
    WorldCupCornersPage,
    ProfitSharingRoundsPage,
    ProfitSharingRoundPage,
    LoginPage,
    RegisterPage
} from './routes';

configureMemberSessionServices();
services.viewPreferences.init();

const applicationBaseHRef = readApplicationBaseHRef();
const deploymentBaseHRef = readDeploymentBaseHRef();
requests.setBaseHRef(deploymentBaseHRef);
requests.configureAuthorizationRealm('member');

const authorizationFreshnessMs = 15_000;
const maintenanceLoginPath = '/login?reason=maintenance';

interface NavItem {
    key: string;
    label: string;
    icon: React.ReactNode;
    path?: string;
    children?: NavItem[];
    module?: AccountDataModule;
    availability?: 'profit-sharing';
    disabled?: boolean;
}

interface NavSection {
    key: string;
    label: string;
    children: NavItem[];
}

interface AccessState {
    user: UserInfo;
    isAdmin: boolean;
    moduleAccess: ModuleAccessLevels;
    revision: number;
}

const canAccessItem = (authorization: AccessState, item: NavItem) => {
    if (item.availability === 'profit-sharing' && !authorization.user.access.profitSharingEnabled) {
        return false;
    }
    if (item.module === AccountDataModule.TraderSync) return authorization.moduleAccess[item.module] === AccountDataAccess.ReadWrite;
    return item.module === undefined || authorization.moduleAccess[item.module] >= AccountDataAccess.Read;
};

const marketRadarNavItem: NavItem = {
    key: 'market-radar',
    label: 'Market Radar',
    icon: <DashboardOutlined />,
    module: AccountDataModule.MarketRadar,
    children: [
        {key: '/market-radar', label: 'Hot Markets', path: '/market-radar', icon: <DashboardOutlined />},
        {
            key: '/market-radar/realtime',
            label: 'Realtime',
            path: '/market-radar/realtime',
            icon: <DashboardOutlined />
        },
        {
            key: '/market-radar/movers',
            label: 'Movers',
            path: '/market-radar/movers',
            icon: <BarChartOutlined />
        }
    ]
};

const sportsNavItem: NavItem = {
    key: 'sports',
    label: 'Sports',
    icon: <TrophyOutlined />,
    children: [
        {key: '/sports-live', label: 'Sports Live', path: '/sports-live', icon: <DashboardOutlined />, module: AccountDataModule.SportsLive},
        {
            key: '/sports-history',
            label: 'Sports History',
            path: '/sports-history',
            icon: <DashboardOutlined />,
            module: AccountDataModule.SportsHistory
        }
    ]
};

const managedOONavItem: NavItem = {
    key: 'managed-oo',
    label: 'Managed OO',
    icon: <ApiOutlined />,
    module: AccountDataModule.ManagedOO,
    children: [
        {
            key: '/managed-oo/proposals',
            label: 'Proposals',
            path: '/managed-oo/proposals',
            icon: <ApiOutlined />
        },
        {
            key: '/managed-oo/disputes',
            label: 'Disputes',
            path: '/managed-oo/disputes',
            icon: <ApiOutlined />
        }
    ]
};

const wormTradingNavItem: NavItem = {
    key: 'worm-trading',
    label: 'Worm Trading',
    icon: <SwapOutlined />,
    module: AccountDataModule.WormTrading,
    children: [
        {key: '/worm-trading', label: 'Assets', path: '/worm-trading', icon: <WalletOutlined />},
        {key: '/worm-trading/combinations', label: 'Combinations', path: '/worm-trading/combinations', icon: <FileTextOutlined />},
        {key: '/worm-trading/executions', label: 'Executions', path: '/worm-trading/executions', icon: <HistoryOutlined />}
    ]
};

const tokenNavItem: NavItem = {
    key: 'token',
    label: 'Token',
    icon: <DashboardOutlined />,
    module: AccountDataModule.Token,
    disabled: true
};

const solanaNavItem: NavItem = {
    key: '/solana',
    label: 'Solana',
    path: '/solana',
    icon: <ApiOutlined />,
    module: AccountDataModule.Solana
};

const navSections: NavSection[] = [
    {
        key: 'markets',
        label: 'Markets',
        children: [
            marketRadarNavItem,
            {key: '/trader-sync', label: 'Trader Sync', path: '/trader-sync', icon: <SwapOutlined />, module: AccountDataModule.TraderSync},
            sportsNavItem,
            managedOONavItem,
            wormTradingNavItem,
            {
                key: '/world-cup-corners',
                label: 'World Cup Corners',
                path: '/world-cup-corners',
                icon: <BarChartOutlined />,
                module: AccountDataModule.WorldCupCorners
            }
        ]
    },
    {
        key: 'token-risk',
        label: 'Token & Risk',
        children: [tokenNavItem, solanaNavItem, {key: '/wallet', label: 'Wallets', path: '/wallet', icon: <WalletOutlined />, module: AccountDataModule.Wallet}]
    },
    {
        key: 'operations',
        label: 'Operations',
        children: [
            {key: '/profit-sharing', label: 'Profit Sharing', path: '/profit-sharing', icon: <PieChartOutlined />, availability: 'profit-sharing'},
            {key: '/notifications', label: 'Notifications', path: '/notifications', icon: <BellOutlined />}
        ]
    }
];

const navItems = navSections.flatMap(section => section.children);

const accountRouteMetadata = [
    {path: '/account/profile', section: 'Account', label: 'Profile'},
    {path: '/account/security', section: 'Account', label: 'Security'},
    {path: '/account/access', section: 'Account', label: 'Access & session'},
    {path: '/help', section: 'Support', label: 'Help'}
];

const flattenNav = (items: NavItem[]): NavItem[] => items.flatMap(item => [item, ...(item.children ? flattenNav(item.children) : [])]);

const filterNavItems = (items: NavItem[], access: AccessState): NavItem[] =>
    items
        .map(item => {
            const children = item.children ? filterNavItems(item.children, access) : undefined;
            if (children) {
                return children.length > 0 && canAccessItem(access, item) ? {...item, children} : null;
            }
            return canAccessItem(access, item) ? item : null;
        })
        .filter((item): item is NavItem => item !== null);

const toMenuItems = (items: NavItem[]): MenuProps['items'] =>
    items.map(item => ({
        key: item.key,
        icon: item.icon,
        label: item.label,
        disabled: item.disabled,
        children: item.children ? toMenuItems(item.children) : undefined
    }));

const filterNavSections = (sections: NavSection[], access: AccessState): NavSection[] =>
    sections.map(section => ({...section, children: filterNavItems(section.children, access)})).filter(section => section.children.length > 0);

const toSectionMenuItems = (sections: NavSection[]): MenuProps['items'] =>
    sections.map(section => ({
        key: section.key,
        type: 'group',
        label: section.label,
        children: toMenuItems(section.children)
    }));

const selectedKey = (pathname: string) => {
    const exact = flattenNav(navItems)
        .filter(item => item.path)
        .sort((a, b) => (b.path || '').length - (a.path || '').length)
        .find(item => pathname === item.path || pathname.startsWith(`${item.path}/`));
    return exact?.key || '';
};

const openKeys = (pathname: string) =>
    navItems.filter(item => (item.children || []).some(child => pathname === child.path || pathname.startsWith(`${child.path}/`))).map(item => item.key);

const navTrail = (items: NavItem[], targetKey: string): NavItem[] => {
    for (const item of items) {
        if (item.key === targetKey) {
            return [item];
        }
        const childTrail = item.children ? navTrail(item.children, targetKey) : [];
        if (childTrail.length > 0) {
            return [item, ...childTrail];
        }
    }
    return [];
};

const breadcrumbItems = (pathname: string) => {
    if (pathname.startsWith('/trader-sync/')) return [{title: 'Markets'}, {title: 'Trader Sync'}, {title: routeTitle(pathname) || 'Trader Sync'}];
    const targetKey = selectedKey(pathname);
    const section = navSections.find(candidate => navTrail(candidate.children, targetKey).length > 0);
    const trail = section ? navTrail(section.children, targetKey) : [];
    if (section) {
        const labels = [section.label, ...trail.map(item => item.label)];
        if (/^\/worm-trading\/combinations\/[^/]+\/execute\/?$/.test(pathname)) {
            labels.push('Execution Preview');
        }
        return labels.map(title => ({title}));
    }
    const accountRoute = [...accountRouteMetadata]
        .sort((left, right) => right.path.length - left.path.length)
        .find(item => pathname === item.path || pathname.startsWith(`${item.path}/`));
    return accountRoute ? [{title: accountRoute.section}, {title: accountRoute.label}] : [];
};

const routeTitle = (pathname: string) => {
    if (pathname === '/trader-sync/add') return 'Add trader';
    if (pathname.startsWith('/trader-sync/activities/')) return 'Activity';
    if (pathname.startsWith('/trader-sync/summaries/')) return 'Summary batch';
    if (pathname === '/trader-sync/subscriptions') return 'Subscriptions';
    if (pathname.startsWith('/trader-sync/subscriptions/')) return 'Subscription';
    if (/^\/worm-trading\/combinations\/[^/]+\/execute\/?$/.test(pathname)) {
        return 'Execution Preview';
    }
    const selected = selectedKey(pathname);
    const business = flattenNav(navItems).find(item => item.key === selected);
    if (business) {
        return business.label;
    }
    return [...accountRouteMetadata].sort((left, right) => right.path.length - left.path.length).find(item => pathname === item.path || pathname.startsWith(`${item.path}/`))
        ?.label;
};

export {loadAppBootstrapWithRetry};

const loadAccessState = (user: UserInfo): AccessState => ({
    user,
    isAdmin: user.administrator,
    moduleAccess: moduleAccessLevels(user.access),
    revision: user.access.revision
});

const isPendingAccess = (access: AccessState) => accountStatusForAccess(access.user.access, access.isAdmin) === AccountStatus.Pending;

const moduleLandingPaths: Partial<Record<AccountDataModule, string>> = {
    [AccountDataModule.TraderSync]: '/trader-sync',
    [AccountDataModule.MarketRadar]: '/market-radar',
    [AccountDataModule.SportsLive]: '/sports-live',
    [AccountDataModule.SportsHistory]: '/sports-history',
    [AccountDataModule.ManagedOO]: '/managed-oo/proposals',
    [AccountDataModule.WormTrading]: '/worm-trading',
    [AccountDataModule.WorldCupCorners]: '/world-cup-corners',
    [AccountDataModule.Solana]: '/solana',
    [AccountDataModule.Wallet]: '/wallet'
};

const firstAuthorizedBusinessPath = (access: AccessState): string | undefined => {
    for (const definition of accountDataModules) {
        const path = moduleLandingPaths[definition.module];
        if (
            path &&
            (definition.module === AccountDataModule.TraderSync
                ? access.moduleAccess[definition.module] === AccountDataAccess.ReadWrite
                : access.moduleAccess[definition.module] >= AccountDataAccess.Read)
        ) {
            return path;
        }
    }
    return access.user.access.profitSharingEnabled ? '/profit-sharing' : undefined;
};

const mergeMonotonicAccountProjection = (previous: AccessState | null, incoming: AccessState): AccessState => {
    if (!previous || previous.user.accountId !== incoming.user.accountId || previous.user.iss !== incoming.user.iss) {
        return incoming;
    }
    const profile = incoming.user.profile.revision < previous.user.profile.revision ? previous.user.profile : incoming.user.profile;
    if (profile === incoming.user.profile) {
        return incoming;
    }
    return {...incoming, user: {...incoming.user, profile}};
};

type SessionState = {status: 'anonymous'} | {status: 'resolving'} | {status: 'authenticated'; access: AccessState} | {status: 'maintenance'} | {status: 'error'; error: Error};

const loadInitialSessionState = (session: AppBootstrapSession): SessionState => {
    switch (session.status) {
        case AppBootstrapSessionStatus.Anonymous:
            return {status: 'anonymous'};
        case AppBootstrapSessionStatus.AccountMaintenance:
            return {status: 'maintenance'};
        case AppBootstrapSessionStatus.Authenticated:
            if (!session.userInfo) {
                throw new Error('Authenticated app bootstrap session is missing user info');
            }
            return {status: 'authenticated', access: loadAccessState(session.userInfo)};
        default:
            throw new Error(`Unsupported initial app bootstrap session status: ${String(session.status)}`);
    }
};

const NotFoundPage = () => {
    const navigate = useNavigate();
    return (
        <Result
            status='404'
            title='Page not found'
            extra={
                <Button type='primary' onClick={() => navigate('/')}>
                    Return home
                </Button>
            }
        />
    );
};

const narrowShellQuery = '(max-width: 900px)';
const desktopExpandedSidebarWidth = 224;
const mobileExpandedSidebarWidth = 280;

const useNarrowShell = () => {
    const matches = () => typeof window !== 'undefined' && typeof window.matchMedia === 'function' && window.matchMedia(narrowShellQuery).matches;
    const [narrow, setNarrow] = React.useState(matches);
    React.useEffect(() => {
        if (typeof window.matchMedia !== 'function') {
            return;
        }
        const query = window.matchMedia(narrowShellQuery);
        const update = () => setNarrow(query.matches);
        update();
        query.addEventListener('change', update);
        return () => query.removeEventListener('change', update);
    }, []);
    return narrow;
};

const AppRoutes = (props: {access: AccessState; settings: AuthSettings; loggingOut: boolean; onLogout: () => void}) => {
    const identityKey = JSON.stringify([props.access.user.accountId, props.access.user.iss]);
    const pending = isPendingAccess(props.access);
    const moduleRoute = (module: AccountDataModule, element: React.ReactElement) =>
        props.access.moduleAccess[module] >= AccountDataAccess.Read ? element : <Navigate replace={true} to='/account/access' />;
    const traderSyncRoute = (element: React.ReactElement) =>
        props.access.moduleAccess[AccountDataModule.TraderSync] === AccountDataAccess.ReadWrite ? element : <Navigate replace={true} to='/account/access' />;
    const profitSharingRoute = (element: React.ReactElement) => (props.access.user.access.profitSharingEnabled ? element : <Navigate replace={true} to='/account/access' />);
    const accountCenterProps = {
        showMemberSecurity: props.access.user.access.apiKeyEnabled,
        loggingOut: props.loggingOut,
        onLogout: props.onLogout
    };
    return (
        <React.Suspense fallback={<div className='athena-boot'>Loading workspace…</div>}>
            <Routes>
                <Route path='/' element={<Navigate replace={true} to={pending ? '/account/access' : '/account/profile'} />} />
                <Route path='/wallet' element={moduleRoute(AccountDataModule.Wallet, <WalletsPage />)} />
                <Route path='/worm-trading' element={moduleRoute(AccountDataModule.WormTrading, <WormTradingPage key={identityKey} />)} />
                <Route path='/worm-trading/combinations' element={moduleRoute(AccountDataModule.WormTrading, <WormTradingCombinationsPage key={identityKey} />)} />
                <Route path='/worm-trading/combinations/new' element={moduleRoute(AccountDataModule.WormTrading, <WormTradingCombinationBuilderPage key={identityKey} />)} />
                <Route path='/worm-trading/combinations/:id/edit' element={moduleRoute(AccountDataModule.WormTrading, <WormTradingCombinationBuilderPage key={identityKey} />)} />
                <Route path='/worm-trading/combinations/:id/execute' element={moduleRoute(AccountDataModule.WormTrading, <WormTradingExecutionPreviewPage key={identityKey} />)} />
                <Route path='/worm-trading/executions' element={moduleRoute(AccountDataModule.WormTrading, <WormTradingExecutionsPage key={identityKey} />)} />
                <Route path='/worm-trading/executions/:id' element={moduleRoute(AccountDataModule.WormTrading, <WormTradingExecutionDetailPage key={identityKey} />)} />
                <Route path='/market-radar' element={moduleRoute(AccountDataModule.MarketRadar, <MarketRadarHotPage key={identityKey} />)} />
                <Route path='/market-radar/realtime' element={moduleRoute(AccountDataModule.MarketRadar, <MarketRadarRealtimePage key={identityKey} />)} />
                <Route path='/market-radar/movers' element={moduleRoute(AccountDataModule.MarketRadar, <MarketRadarMoversPage key={identityKey} />)} />
                <Route path='/solana' element={moduleRoute(AccountDataModule.Solana, <SolanaPage />)} />
                <Route path='/sports-live' element={moduleRoute(AccountDataModule.SportsLive, <SportsLivePage key={identityKey} />)} />
                <Route path='/sports-history' element={moduleRoute(AccountDataModule.SportsHistory, <SportsHistoryPage key={identityKey} />)} />
                <Route path='/world-cup-corners' element={moduleRoute(AccountDataModule.WorldCupCorners, <WorldCupCornersPage key={identityKey} />)} />
                <Route path='/managed-oo/proposals' element={moduleRoute(AccountDataModule.ManagedOO, <ManagedOOProposalsPage key={identityKey} />)} />
                <Route path='/managed-oo/disputes' element={moduleRoute(AccountDataModule.ManagedOO, <ManagedOODisputesPage key={identityKey} />)} />
                <Route
                    path='/trader-sync'
                    element={traderSyncRoute(
                        <TraderSyncHomePage key={JSON.stringify([props.access.user.accountId, props.access.user.iss])} ownerId={props.access.user.accountId} />
                    )}
                />
                <Route
                    path='/trader-sync/subscriptions'
                    element={traderSyncRoute(
                        <TraderSyncSubscriptionsPage key={JSON.stringify([props.access.user.accountId, props.access.user.iss])} ownerId={props.access.user.accountId} />
                    )}
                />
                <Route
                    path='/trader-sync/subscriptions/:subscriptionId'
                    element={traderSyncRoute(
                        <TraderSyncSubscriptionPage key={JSON.stringify([props.access.user.accountId, props.access.user.iss])} ownerId={props.access.user.accountId} />
                    )}
                />
                <Route
                    path='/trader-sync/add'
                    element={traderSyncRoute(
                        <TraderSyncAddPage key={JSON.stringify([props.access.user.accountId, props.access.user.iss])} ownerId={props.access.user.accountId} />
                    )}
                />
                <Route
                    path='/trader-sync/activities/:activityId'
                    element={traderSyncRoute(
                        <TraderSyncActivityPage key={JSON.stringify([props.access.user.accountId, props.access.user.iss])} ownerId={props.access.user.accountId} />
                    )}
                />
                <Route
                    path='/trader-sync/summaries/:batchId'
                    element={traderSyncRoute(
                        <TraderSyncSummaryPage key={JSON.stringify([props.access.user.accountId, props.access.user.iss])} ownerId={props.access.user.accountId} />
                    )}
                />
                <Route path='/notifications' element={<NotificationsPage key={identityKey} />} />
                <Route path='/account/profile' element={<AccountCenterPage key={identityKey} section='profile' {...accountCenterProps} />} />
                <Route
                    path='/account/security'
                    element={props.access.user.access.apiKeyEnabled ? <AccountSecurityPage key={identityKey} /> : <Navigate replace={true} to='/account/access' />}
                />
                <Route path='/account/access' element={<AccountCenterPage key={identityKey} section='access' {...accountCenterProps} />} />
                <Route path='/profit-sharing' element={profitSharingRoute(<ProfitSharingRoundsPage />)} />
                <Route path='/profit-sharing/:slug' element={profitSharingRoute(<ProfitSharingRoundPage />)} />
                <Route path='/help' element={<HelpPage help={props.settings.help} />} />
                <Route path='*' element={<NotFoundPage />} />
            </Routes>
        </React.Suspense>
    );
};

const Shell = (props: {pref: ViewPreferences; initialSession: AppBootstrapSession; settings: AuthSettings}) => {
    const navigate = useNavigate();
    const location = useLocation();
    const ant = AntApp.useApp();
    const narrowShell = useNarrowShell();
    const [session, setSession] = React.useState<SessionState>(() => loadInitialSessionState(props.initialSession));
    const initialAccess = session.status === 'authenticated' ? session.access : null;
    const [desktopSidebarCollapsed, setDesktopSidebarCollapsed] = React.useState(props.pref.hideSidebar);
    const [mobileSidebarOpen, setMobileSidebarOpen] = React.useState(false);
    const [accessRefreshedAt, setAccessRefreshedAt] = React.useState(initialAccess ? Date.now() : 0);
    const [accountMenuOpen, setAccountMenuOpen] = React.useState(false);
    const [loggingOut, setLoggingOut] = React.useState(false);
    const sidebarRef = React.useRef<HTMLDivElement>(null);
    const shellBackgroundRef = React.useRef<HTMLElement>(null);
    const mobileSidebarToggleRef = React.useRef<HTMLButtonElement>(null);
    const accessGenerationRef = React.useRef(0);
    const accessRefreshAllowedRef = React.useRef(Boolean(initialAccess));
    const accessRef = React.useRef<AccessState>(initialAccess);
    const accessRefreshRef = React.useRef<Promise<boolean>>(null);
    const accessDeniedRefreshRef = React.useRef<Promise<void>>(null);
    const accessRefreshedAtRef = React.useRef(initialAccess ? Date.now() : 0);
    const pendingAccessRef = React.useRef(Boolean(initialAccess && isPendingAccess(initialAccess)));
    const sidebarCollapsed = narrowShell ? !mobileSidebarOpen : desktopSidebarCollapsed;
    const expandedSidebarWidth = narrowShell ? mobileExpandedSidebarWidth : desktopExpandedSidebarWidth;
    const shellStyle = {'--athena-sidebar-width': `${expandedSidebarWidth}px`} as React.CSSProperties;
    const isLoginPath = location.pathname.startsWith('/login');
    const access = session.status === 'authenticated' ? session.access : null;

    const endSession = React.useCallback((status: 'anonymous' | 'maintenance' = 'anonymous') => {
        clearTraderSyncState();
        accessGenerationRef.current += 1;
        accessRefreshAllowedRef.current = false;
        accessRef.current = null;
        accessRefreshRef.current = null;
        accessDeniedRefreshRef.current = null;
        accessRefreshedAtRef.current = 0;
        pendingAccessRef.current = false;
        setAccessRefreshedAt(0);
        requests.invalidatePendingRequestErrors();
        requests.endAuthorizationSession();
        setAccountMenuOpen(false);
        setLoggingOut(false);
        setSession(status === 'maintenance' ? {status: 'maintenance'} : {status: 'anonymous'});
        clearAsyncDataCache();
        clearTelegramBindingInstructions();
    }, []);

    const startAccessRefresh = React.useCallback((): Promise<boolean> => {
        if (!accessRefreshAllowedRef.current) {
            return Promise.resolve(false);
        }
        if (!accessRef.current) {
            setSession({status: 'resolving'});
        }
        const generation = accessGenerationRef.current;
        const request = (async () => {
            try {
                const user = await services.users.get();
                if (generation !== accessGenerationRef.current) {
                    return false;
                }
                if (!user.loggedIn) {
                    endSession('anonymous');
                    navigate(loginPathFor(location.pathname, location.search, location.hash), {replace: true});
                    return false;
                }
                if (user.administrator) {
                    endSession('anonymous');
                    window.location.replace(deploymentPath('admin'));
                    return false;
                }
                const sessionGeneration = requests.beginAuthorizationSession(user.accountId);
                setAsyncDataCacheSession('member', user.accountId, sessionGeneration);
                const previous = accessRef.current;
                const next = mergeMonotonicAccountProjection(previous, loadAccessState(user));
                const authorizationChanged =
                    Boolean(previous) &&
                    (previous.user.accountId !== next.user.accountId ||
                        previous.user.iss !== next.user.iss ||
                        previous.revision !== next.revision ||
                        previous.isAdmin !== next.isAdmin ||
                        !moduleAccessLevelsEqual(previous.moduleAccess, next.moduleAccess));
                if (authorizationChanged) {
                    const priorAccess = previous as AccessState;
                    const identityChanged = priorAccess.user.accountId !== next.user.accountId || priorAccess.user.iss !== next.user.iss || priorAccess.isAdmin !== next.isAdmin;
                    if (identityChanged) {
                        clearTraderSyncState();
                        requests.invalidatePendingRequestErrors();
                        requests.abortAuthorizationRequests();
                        clearAsyncDataCache();
                        clearTelegramBindingInstructions();
                    } else {
                        if (priorAccess.user.access.apiKeyEnabled && !next.user.access.apiKeyEnabled) {
                            requests.abortAuthorizationFeatureRequests('api-key');
                        }
                        if (priorAccess.user.access.profitSharingEnabled && !next.user.access.profitSharingEnabled) {
                            requests.abortAuthorizationFeatureRequests('profit-sharing');
                        }
                        revokeLostModuleAccess(priorAccess.moduleAccess, next.moduleAccess);
                        if (!isPendingAccess(priorAccess) && isPendingAccess(next)) {
                            clearAsyncDataCache();
                        }
                    }
                }
                accessRefreshedAtRef.current = Date.now();
                setAccessRefreshedAt(accessRefreshedAtRef.current);
                accessRef.current = next;
                setSession({status: 'authenticated', access: next});
                return true;
            } catch (err: any) {
                if (generation !== accessGenerationRef.current) {
                    return false;
                }
                if (isAccountMaintenanceError(err)) {
                    endSession('maintenance');
                    navigate(maintenanceLoginPath, {replace: true});
                    return false;
                }
                if (err?.status === 401) {
                    endSession('anonymous');
                    navigate(loginPathFor(location.pathname, location.search, location.hash), {replace: true});
                    return false;
                }
                if (!accessRef.current) {
                    setSession({status: 'error', error: err instanceof Error ? err : new Error(err?.message || String(err))});
                }
                throw err;
            }
        })();
        accessRefreshRef.current = request;
        const clearPendingRefresh = () => {
            if (accessRefreshRef.current === request) {
                accessRefreshRef.current = null;
            }
        };
        void request.then(clearPendingRefresh, clearPendingRefresh);
        return request;
    }, [endSession, location.hash, location.pathname, location.search, navigate]);

    const refreshAccess = React.useCallback(
        (force = false): Promise<boolean> => {
            if (!force && accessRef.current && Date.now() - accessRefreshedAtRef.current < authorizationFreshnessMs) {
                return Promise.resolve(true);
            }
            const pending = accessRefreshRef.current;
            if (!pending) {
                return startAccessRefresh();
            }
            if (!force) {
                return pending;
            }

            // A mutation-triggered refresh must observe state after any request
            // that may have started before the mutation. Coalesce concurrent
            // forced callers into the one follow-up request.
            const generation = accessGenerationRef.current;
            const refreshAfterPending = (): Promise<boolean> => {
                if (generation !== accessGenerationRef.current) {
                    return Promise.resolve(false);
                }
                return accessRefreshRef.current || startAccessRefresh();
            };
            return pending.then(refreshAfterPending, refreshAfterPending);
        },
        [startAccessRefresh]
    );

    const establishAuthenticatedSession = React.useCallback(async () => {
        accessRefreshAllowedRef.current = true;
        try {
            return await refreshAccess(true);
        } catch {
            return false;
        }
    }, [refreshAccess]);

    const refreshAfterAccessDenied = React.useCallback(() => {
        if (accessDeniedRefreshRef.current) {
            return accessDeniedRefreshRef.current;
        }
        const request = refreshAccess(true).then(
            () => undefined,
            () => undefined
        );
        accessDeniedRefreshRef.current = request;
        void request.finally(() => {
            window.setTimeout(() => {
                if (accessDeniedRefreshRef.current === request) {
                    accessDeniedRefreshRef.current = null;
                }
            }, 250);
        });
        return request;
    }, [refreshAccess]);

    React.useEffect(() => {
        if (!access) {
            pendingAccessRef.current = false;
            return;
        }
        const pending = isPendingAccess(access);
        const wasPending = pendingAccessRef.current;
        pendingAccessRef.current = pending;
        if (!wasPending || pending) {
            return;
        }
        const destination = firstAuthorizedBusinessPath(access);
        if (destination && (location.pathname.startsWith('/account/') || location.pathname === '/help')) {
            navigate(destination, {replace: true});
        }
    }, [access?.revision, access?.user.access.profitSharingEnabled, access?.user.accountId, location.pathname, navigate]);

    React.useEffect(() => {
        setDesktopSidebarCollapsed(props.pref.hideSidebar);
    }, [props.pref.hideSidebar]);

    React.useEffect(() => {
        if (narrowShell) {
            setMobileSidebarOpen(false);
            setAccountMenuOpen(false);
        }
    }, [location.pathname, narrowShell]);

    React.useEffect(() => {
        if (!accountMenuOpen) return;
        const closeOnEscape = (event: KeyboardEvent) => {
            if (event.key === 'Escape') setAccountMenuOpen(false);
        };
        document.addEventListener('keydown', closeOnEscape, true);
        return () => document.removeEventListener('keydown', closeOnEscape, true);
    }, [accountMenuOpen]);

    React.useLayoutEffect(() => {
        if (!narrowShell || !mobileSidebarOpen) {
            return;
        }
        const sidebar = sidebarRef.current;
        const background = shellBackgroundRef.current;
        const focusInitialControl = () => sidebar?.querySelector<HTMLElement>('.athena-shell__mobile-close, .athena-brand, [role="menuitem"]')?.focus();
        let focusFrame = 0;
        const focusWhenVisible = () => {
            if (sidebar && getComputedStyle(sidebar).visibility !== 'hidden' && sidebar.getClientRects().length > 0) {
                focusInitialControl();
                return;
            }
            focusFrame = window.requestAnimationFrame(focusWhenVisible);
        };
        focusFrame = window.requestAnimationFrame(focusWhenVisible);
        background?.setAttribute('inert', '');
        background?.setAttribute('aria-hidden', 'true');
        const previousBodyOverflow = document.body.style.overflow;
        document.body.style.overflow = 'hidden';
        const handleDialogKeyboard = (event: KeyboardEvent) => {
            if (event.key === 'Escape') {
                event.preventDefault();
                setMobileSidebarOpen(false);
                return;
            }
            if (event.key !== 'Tab' || !sidebar) {
                return;
            }
            const focusable = Array.from(
                sidebar.querySelectorAll<HTMLElement>(
                    'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [role="menuitem"], [tabindex]:not([tabindex="-1"])'
                )
            ).filter(item => item.getClientRects().length > 0 && item.getAttribute('aria-hidden') !== 'true');
            if (focusable.length === 0) {
                event.preventDefault();
                return;
            }
            const activeIndex = focusable.findIndex(item => item === document.activeElement);
            let nextIndex = activeIndex + 1;
            if (event.shiftKey) {
                nextIndex = activeIndex <= 0 ? focusable.length - 1 : activeIndex - 1;
            } else if (activeIndex < 0 || activeIndex === focusable.length - 1) {
                nextIndex = 0;
            }
            event.preventDefault();
            focusable[nextIndex].focus();
        };
        document.addEventListener('keydown', handleDialogKeyboard, true);
        return () => {
            document.body.style.overflow = previousBodyOverflow;
            window.cancelAnimationFrame(focusFrame);
            background?.removeAttribute('inert');
            background?.removeAttribute('aria-hidden');
            document.removeEventListener('keydown', handleDialogKeyboard, true);
            window.requestAnimationFrame(() => mobileSidebarToggleRef.current?.focus());
        };
    }, [mobileSidebarOpen, narrowShell]);

    React.useLayoutEffect(() => {
        const navigation = sidebarRef.current?.querySelector('#athena-primary-navigation');
        navigation?.querySelectorAll('[aria-current="page"]').forEach(item => item.removeAttribute('aria-current'));
        navigation?.querySelector('.ant-menu-item-selected')?.setAttribute('aria-current', 'page');
    }, [location.pathname, mobileSidebarOpen, narrowShell, access?.revision]);

    React.useEffect(() => {
        if (session.status === 'anonymous' && !isLoginPath) {
            navigate(loginPathFor(location.pathname, location.search, location.hash), {replace: true});
            return;
        }
        if (session.status === 'maintenance' && (!isLoginPath || new URLSearchParams(location.search).get('reason') !== 'maintenance')) {
            navigate(maintenanceLoginPath, {replace: true});
            return;
        }
        if (session.status === 'authenticated' && isLoginPath) {
            const returnTo = readLoginReturnTo(location.search);
            if (returnTo === '/admin' || returnTo.startsWith('/admin/')) {
                window.location.replace(deploymentPath(returnTo));
                return;
            }
            navigate(returnTo, {replace: true});
        }
    }, [isLoginPath, location.hash, location.pathname, location.search, navigate, session.status]);

    React.useEffect(() => {
        if (isLoginPath || !access) {
            return;
        }
        const refreshVisibleAccess = (force: boolean) => {
            if (document.visibilityState === 'visible') {
                void refreshAccess(force).catch(() => undefined);
            }
        };
        const refreshOnInterval = () => refreshVisibleAccess(false);
        const refreshOnAttention = () => refreshVisibleAccess(true);
        const interval = window.setInterval(refreshOnInterval, authorizationFreshnessMs);
        window.addEventListener('focus', refreshOnAttention);
        document.addEventListener('visibilitychange', refreshOnAttention);
        return () => {
            window.clearInterval(interval);
            window.removeEventListener('focus', refreshOnAttention);
            document.removeEventListener('visibilitychange', refreshOnAttention);
        };
    }, [Boolean(access), isLoginPath, refreshAccess]);

    React.useEffect(() => {
        const subscription: Subscription = requests.onError.subscribe(err => {
            if (isLoginPath) {
                return;
            }
            if (isAccountDataAccessDeniedError(err) || isAccountProfitSharingAccessDeniedError(err)) {
                void refreshAfterAccessDenied();
                return;
            }
            const details = requestErrorDetails(err);
            if (details.status === 401 && details.reason === WALLET_REAUTH_REQUIRED) {
                return;
            }
            const maintenance = isAccountMaintenanceError(err);
            if (!maintenance && details.status !== 401) {
                return;
            }
            if (window.location.pathname.startsWith(`${applicationBaseHRef.replace(/\/$/, '')}/login`)) {
                return;
            }
            endSession(maintenance ? 'maintenance' : 'anonymous');
            navigate(maintenance ? maintenanceLoginPath : loginPathFor(location.pathname, location.search, location.hash), {replace: true});
        });
        return () => subscription?.unsubscribe();
    }, [endSession, isLoginPath, location.hash, location.pathname, location.search, navigate, refreshAfterAccessDenied]);

    React.useEffect(() => {
        const current = routeTitle(location.pathname);
        document.title = current ? `${current} · Athena` : 'Athena';
    }, [location.pathname]);

    const visibleNavSections = access ? filterNavSections(navSections, access) : [];
    const visibleNavItems = visibleNavSections.flatMap(section => section.children);

    const onMenuClick: MenuProps['onClick'] = item => {
        const target = flattenNav(visibleNavItems).find(navItem => navItem.key === item.key);
        if (target?.path && !target.disabled) {
            if (narrowShell) {
                setMobileSidebarOpen(false);
            }
            navigate(target.path);
        }
    };

    const notifications = React.useMemo(
        () => ({
            success: (message: string, description?: string) => ant.notification.success({title: message, description}),
            error: (message: string, description?: string) => ant.notification.error({title: message, description}),
            info: (message: string, description?: string) => ant.notification.info({title: message, description}),
            warning: (message: string, description?: string) => ant.notification.warning({title: message, description})
        }),
        [ant.notification]
    );

    const contextValue = React.useMemo(
        () => ({
            notifications,
            modal: ant.modal,
            navigation: {
                goto: navigate,
                replace: (path: string) => navigate(path, {replace: true})
            },
            baseHref: deploymentBaseHRef
        }),
        [ant.modal, navigate, notifications]
    );
    const authorizationValue = React.useMemo(() => {
        if (!access) {
            return null;
        }
        const generation = accessGenerationRef.current;
        const accountId = access.user.accountId;
        const issuer = access.user.iss;
        return {
            user: access.user,
            isAdmin: access.isAdmin,
            access: (module: AccountDataModule) => access.moduleAccess[module],
            canRead: (module: AccountDataModule) => access.moduleAccess[module] >= AccountDataAccess.Read,
            canWrite: (module: AccountDataModule) => access.moduleAccess[module] >= AccountDataAccess.ReadWrite,
            revision: access.revision,
            lastCheckedAt: accessRefreshedAt,
            refresh: async () => {
                const current = accessRef.current;
                if (
                    generation !== accessGenerationRef.current ||
                    !accessRefreshAllowedRef.current ||
                    !current ||
                    current.user.accountId !== accountId ||
                    current.user.iss !== issuer
                ) {
                    return;
                }
                await refreshAccess(true);
            }
        };
    }, [access, accessRefreshedAt, refreshAccess]);

    const logout = React.useCallback(async () => {
        if (loggingOut) {
            return;
        }
        setLoggingOut(true);
        notifications.info('Logging out');
        try {
            await services.users.logout();
            setAccountMenuOpen(false);
            endSession();
            window.location.replace(deploymentPath('login'));
        } catch (err) {
            notifications.error('Logout failed', requestErrorMessage(err, 'Could not log out. Your session is still active.'));
            setLoggingOut(false);
        }
    }, [endSession, loggingOut, notifications]);

    const navigateFromAccountMenu = (path: string) => {
        setAccountMenuOpen(false);
        if (narrowShell) {
            setMobileSidebarOpen(false);
        }
        navigate(path);
    };

    const accountMenuItems: MenuProps['items'] = access
        ? [
              {
                  key: 'account-summary',
                  disabled: true,
                  label: (
                      <div className='athena-account-menu__summary'>
                          <AccountAvatar profile={access.user.profile} username={access.user.username} size={42} />
                          <span>
                              <strong>{access.user.profile.displayName || access.user.username}</strong>
                              <small>@{access.user.username}</small>
                          </span>
                          <Tag>{accountTierLabel(access.user.profile.tier)}</Tag>
                      </div>
                  )
              },
              {type: 'divider'},
              {key: '/account/profile', label: 'Profile', icon: <IdcardOutlined />},
              {key: '/account/access', label: 'Access', icon: <KeyOutlined />},
              ...(access.user.access.apiKeyEnabled ? [{key: '/account/security', label: 'Security', icon: <SettingOutlined />}] : []),
              {key: '/help', label: 'Help', icon: <QuestionCircleOutlined />},
              {type: 'divider'},
              {
                  key: 'logout',
                  label: loggingOut ? (
                      <Space size={8}>
                          <Spin size='small' />
                          Logging out…
                      </Space>
                  ) : (
                      'Log out'
                  ),
                  icon: <LogoutOutlined />,
                  danger: true,
                  disabled: loggingOut
              }
          ]
        : [];

    const onAccountMenuClick: MenuProps['onClick'] = item => {
        if (item.key === 'logout') {
            void logout();
            return;
        }
        if (item.key.startsWith('/')) {
            navigateFromAccountMenu(item.key);
        }
    };

    const menu = (
        <Menu
            mode='inline'
            items={toSectionMenuItems(visibleNavSections)}
            selectedKeys={[selectedKey(location.pathname)]}
            defaultOpenKeys={openKeys(location.pathname)}
            onClick={onMenuClick}
        />
    );

    let routes: React.ReactNode;
    if (session.status === 'error') {
        routes = (
            <div className='athena-recoverable'>
                <Result
                    status='warning'
                    title='Unable to load session'
                    subTitle='Athena could not load your account access. Retry when the service is available.'
                    extra={
                        <Space orientation='vertical' size={12}>
                            <Button type='primary' onClick={() => void establishAuthenticatedSession()}>
                                Retry
                            </Button>
                            <Typography.Text type='secondary'>{session.error.message}</Typography.Text>
                        </Space>
                    }
                />
            </div>
        );
    } else if (isLoginPath && (session.status === 'anonymous' || session.status === 'maintenance')) {
        routes = (
            <Routes>
                <Route
                    path='/login'
                    element={
                        <React.Suspense fallback={<div className='athena-boot'>Loading sign in…</div>}>
                            <LoginPage />
                        </React.Suspense>
                    }
                />
                <Route path='*' element={<Navigate replace={true} to='/login' />} />
            </Routes>
        );
    } else if (access && !isLoginPath) {
        routes = <AppRoutes access={access} settings={props.settings} loggingOut={loggingOut} onLogout={() => void logout()} />;
    } else {
        routes = <div className='athena-boot'>Loading Athena...</div>;
    }
    const content = isLoginPath ? (
        routes
    ) : (
        <AntLayout className='athena-shell athena-member-shell' style={shellStyle}>
            <a className='athena-skip-link' href='#athena-main' aria-hidden={narrowShell && mobileSidebarOpen} tabIndex={narrowShell && mobileSidebarOpen ? -1 : undefined}>
                Skip to main content
            </a>
            <AntLayout.Sider
                className='athena-shell__sider'
                collapsible={true}
                collapsed={sidebarCollapsed}
                collapsedWidth={narrowShell ? 0 : 72}
                trigger={null}
                width={expandedSidebarWidth}
                ref={sidebarRef}
                role={narrowShell && mobileSidebarOpen ? 'dialog' : undefined}
                aria-modal={narrowShell && mobileSidebarOpen ? true : undefined}
                aria-label={narrowShell && mobileSidebarOpen ? 'Primary navigation' : undefined}
                aria-hidden={narrowShell && !mobileSidebarOpen}>
                {narrowShell && mobileSidebarOpen && (
                    <Button
                        className='athena-shell__mobile-close'
                        type='text'
                        aria-label='Close navigation'
                        icon={<MenuFoldOutlined />}
                        onClick={() => setMobileSidebarOpen(false)}
                    />
                )}
                {(!narrowShell || mobileSidebarOpen) && (
                    <div className='athena-brand'>
                        <BrandMark size='small' />
                        {!sidebarCollapsed && (
                            <span className='athena-brand__copy'>
                                <strong>Athena</strong>
                                <small>Operations Console</small>
                            </span>
                        )}
                    </div>
                )}
                <nav className='athena-sidebar-navigation' id='athena-primary-navigation' aria-label='Primary navigation'>
                    {(!narrowShell || mobileSidebarOpen) && menu}
                </nav>
                {(!narrowShell || mobileSidebarOpen) && (
                    <div className='athena-sidebar-bottom'>
                        <Button
                            type='text'
                            icon={<QuestionCircleOutlined />}
                            aria-label='Help'
                            aria-current={location.pathname === '/help' ? 'page' : undefined}
                            onClick={() => {
                                setMobileSidebarOpen(false);
                                navigate('/help');
                            }}>
                            <span className='athena-sidebar-bottom__label'>Help</span>
                        </Button>
                        {!sidebarCollapsed && <p>Member workspace</p>}
                    </div>
                )}
            </AntLayout.Sider>
            {narrowShell && mobileSidebarOpen && (
                <button
                    className='athena-shell__backdrop'
                    type='button'
                    aria-label='Close navigation'
                    aria-hidden='true'
                    tabIndex={-1}
                    onClick={() => {
                        setMobileSidebarOpen(false);
                    }}
                />
            )}
            <AntLayout ref={shellBackgroundRef}>
                <AntLayout.Header className='athena-shell__header'>
                    <div className='athena-shell__header-left'>
                        <Tooltip title={narrowShell ? undefined : sidebarCollapsed ? 'Open navigation' : 'Close navigation'}>
                            <Button
                                ref={mobileSidebarToggleRef}
                                className='athena-shell__sidebar-toggle'
                                type='text'
                                aria-label={sidebarCollapsed ? 'Open navigation' : 'Close navigation'}
                                aria-controls='athena-primary-navigation'
                                aria-expanded={!sidebarCollapsed}
                                icon={sidebarCollapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
                                onClick={() => {
                                    if (narrowShell) {
                                        setMobileSidebarOpen(current => !current);
                                        return;
                                    }
                                    const next = !desktopSidebarCollapsed;
                                    setDesktopSidebarCollapsed(next);
                                    services.viewPreferences.updatePreferences({hideSidebar: next});
                                }}
                            />
                        </Tooltip>
                        <Breadcrumb className='athena-shell__breadcrumb' items={breadcrumbItems(location.pathname)} />
                    </div>
                    {access && (
                        <div className='athena-shell__header-actions'>
                            <Dropdown
                                open={accountMenuOpen}
                                trigger={['click']}
                                placement='bottomRight'
                                classNames={{root: 'athena-account-menu'}}
                                destroyOnHidden={true}
                                menu={{items: accountMenuItems, onClick: onAccountMenuClick, selectable: false}}
                                onOpenChange={setAccountMenuOpen}>
                                <button className='athena-account-trigger' type='button' aria-label='Open account menu' aria-haspopup='menu' aria-expanded={accountMenuOpen}>
                                    <AccountAvatar className='athena-account-avatar' profile={access.user.profile} username={access.user.username} size={28} />
                                    <span className='athena-account-trigger__copy'>
                                        <strong>{access.user.profile.displayName || access.user.username}</strong>
                                        <small>@{access.user.username}</small>
                                    </span>
                                    <span className='athena-account-trigger__more' aria-hidden='true'>
                                        •••
                                    </span>
                                </button>
                            </Dropdown>
                        </div>
                    )}
                </AntLayout.Header>
                <AntLayout.Content className='athena-shell__content' id='athena-main' tabIndex={-1}>
                    {routes}
                </AntLayout.Content>
            </AntLayout>
        </AntLayout>
    );

    return (
        <Provider value={contextValue}>
            <AuthorizationCtx.Provider value={authorizationValue}>{content}</AuthorizationCtx.Provider>
        </Provider>
    );
};

const acceptMemberBootstrap = (bootstrap: AppBootstrap) => {
    if (bootstrap.session.status === AppBootstrapSessionStatus.Authenticated && bootstrap.session.userInfo?.administrator) {
        window.location.replace(deploymentPath('admin'));
        return false;
    }
    return true;
};

const Bootstrap = () => (
    <SessionBootstrap acceptBootstrap={acceptMemberBootstrap}>
        {(bootstrap, preferences) => {
            if (bootstrap.session.status === AppBootstrapSessionStatus.Authenticated && bootstrap.session.userInfo && !bootstrap.session.userInfo.administrator) {
                ensureMemberBusinessServices();
                const sessionGeneration = requests.beginAuthorizationSession(bootstrap.session.userInfo.accountId);
                setAsyncDataCacheSession('member', bootstrap.session.userInfo.accountId, sessionGeneration);
            }
            return <Shell pref={preferences} initialSession={bootstrap.session} settings={bootstrap.settings} />;
        }}
    </SessionBootstrap>
);

export const MemberApp = () => {
    const [router] = React.useState(() =>
        createBrowserRouter([{path: '*', element: <AppEntry />}], {
            basename: applicationBaseHRef,
            future: {v7_relativeSplatPath: true}
        })
    );
    return <RouterProvider router={router} future={{v7_startTransition: true}} />;
};

const RegistrationBootstrap = () => (
    <React.Suspense fallback={<div className='athena-boot'>Loading registration…</div>}>
        <RegisterPage />
    </React.Suspense>
);

const AppEntry = () => {
    const location = useLocation();
    return location.pathname === '/register' ? <RegistrationBootstrap /> : <Bootstrap />;
};

// Apply the same module lifecycle to every grant, including modules whose pages
// have not yet been registered. Cache invalidation fences in-flight responses.
export const revokeLostModuleAccess = (previous: ModuleAccessLevels, next: ModuleAccessLevels) => {
    if (previous[AccountDataModule.TraderSync] >= AccountDataAccess.ReadWrite && next[AccountDataModule.TraderSync] < AccountDataAccess.ReadWrite) {
        clearTraderSyncState();
    }
    accountDataModules.forEach(definition => {
        const prior = previous[definition.module];
        const current = next[definition.module];
        if (prior >= AccountDataAccess.Read && current < AccountDataAccess.Read) {
            requests.abortAuthorizationRequests(definition.module);
            clearAsyncDataCache(definition.module);
        } else if (prior >= AccountDataAccess.ReadWrite && current < AccountDataAccess.ReadWrite) {
            requests.abortAuthorizationRequests(definition.module, 'write');
        }
    });
};
