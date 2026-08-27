import '@fortawesome/fontawesome-free/css/all.css';
import 'antd/dist/reset.css';
import '../assets/fonts.css';
import './styles.css';

import {
    ApiOutlined,
    BarChartOutlined,
    BgColorsOutlined,
    BellOutlined,
    CheckOutlined,
    CodeOutlined,
    DashboardOutlined,
    DesktopOutlined,
    FileTextOutlined,
    HeartOutlined,
    IdcardOutlined,
    KeyOutlined,
    LogoutOutlined,
    MenuFoldOutlined,
    MenuUnfoldOutlined,
    MoonOutlined,
    PieChartOutlined,
    QuestionCircleOutlined,
    SettingOutlined,
    SunOutlined,
    TrophyOutlined,
    UserOutlined,
    WalletOutlined
} from '@ant-design/icons';
import {App as AntApp, Breadcrumb, Button, ConfigProvider, Dropdown, Layout as AntLayout, Menu, Result, Space, Spin, Tag, theme as antTheme, Tooltip, Typography} from 'antd';
import type {MenuProps} from 'antd';
import * as React from 'react';
import {createBrowserRouter, Navigate, Route, RouterProvider, Routes, useLocation, useNavigate} from 'react-router-dom';
import {Subscription} from 'rxjs';
import {AuthorizationCtx, Provider} from './shared/context';
import {AccountDataAccess, AccountDataModule, accountDataModules} from './shared/access-modules';
import {moduleAccessLevels, moduleAccessLevelsEqual, ModuleAccessLevels} from './shared/account-access';
import {accountStatusForAccess, AccountStatus, AppBootstrap, AppBootstrapSession, AppBootstrapSessionStatus, AuthSettings, UserInfo} from './shared/models';
import {services, ThemeMode, ViewPreferences} from './shared/services';
import requests, {isAccountDataAccessDeniedError, isAccountMaintenanceError, requestErrorMessage} from './shared/services/requests';
import {loginPathFor, readLoginReturnTo} from './shared/login-navigation';
import {BrandMark, clearAsyncDataCache} from './components';
import {clearProjectsReturnSnapshots} from './pages/project-navigation';
import {
    ContractCodeBlocklistPage,
    ChainProcessingPage,
    CollectionTasksPage,
    ContractCodeDetailPage,
    ContractCodesPage,
    AccountAvatar,
    AccountCenterPage,
    accountThemeLabel,
    accountTierLabel,
    AdminAccountsPage,
    EtherscanGatewaysPage,
    HelpPage,
    LoginPage,
    NodeStatusesPage,
    NotificationsDetailPage,
    NotificationsPage,
    MarketRadarHotPage,
    MarketRadarMoversPage,
    MarketRadarRealtimePage,
    SportsLivePage,
    SportsHistoryPage,
    ManagedOODisputesPage,
    ManagedOOProposalsPage,
    ProjectDetailPage,
    ProjectsPage,
    ServiceStatusPage,
    WalletBlocklistPage,
    WalletsPage,
    WorldCupCornersPage,
    FIFAMarketDashboardPage,
    ProfitSharingRoundsPage,
    ProfitSharingRoundPage,
    ProfitSharingAdminRoundsPage,
    ProfitSharingAdminRoundPage,
    RegisterPage,
    localThemeMode,
    serverThemeMode
} from './pages';

services.viewPreferences.init();

const bases = document.getElementsByTagName('base');
const base = bases.length > 0 ? bases[0].getAttribute('href') || '/' : '/';
requests.setBaseHRef(base);

const bootstrapRetryDelays = [500, 1000, 2000, 3000];
const authorizationFreshnessMs = 15_000;
const maintenanceLoginPath = '/login?reason=maintenance';

const wait = (delayMs: number) => new Promise(resolve => window.setTimeout(resolve, delayMs));

interface NavItem {
    key: string;
    label: string;
    icon: React.ReactNode;
    path?: string;
    children?: NavItem[];
    access?: 'admin';
    module?: AccountDataModule;
    availability?: 'profit-sharing';
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
    if (item.availability === 'profit-sharing' && !authorization.isAdmin && !authorization.user.access.profitSharingEnabled) {
        return false;
    }
    if (item.access === 'admin') {
        return authorization.isAdmin;
    }
    return item.module === undefined || authorization.isAdmin || authorization.moduleAccess[item.module] >= AccountDataAccess.Read;
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

const tokenNavItem: NavItem = {
    key: 'token',
    label: 'Token',
    icon: <DashboardOutlined />,
    module: AccountDataModule.Token,
    children: [
        {key: '/token/projects', label: 'Projects', path: '/token/projects', icon: <FileTextOutlined />},
        {
            key: '/token/contract-codes',
            label: 'Contract Codes',
            path: '/token/contract-codes',
            icon: <CodeOutlined />
        },
        {
            key: '/token/contract-code-blocklist',
            label: 'Contract Code Blocklist',
            path: '/token/contract-code-blocklist',
            icon: <ApiOutlined />
        },
        {
            key: '/token/wallet-blocklist',
            label: 'Wallet Blocklist',
            path: '/token/wallet-blocklist',
            icon: <WalletOutlined />
        },
        {
            key: '/token/node-statuses',
            label: 'Node Status',
            path: '/token/node-statuses',
            icon: <ApiOutlined />
        },
        {
            key: '/token/chain-processing',
            label: 'Chain Processing',
            path: '/token/chain-processing',
            icon: <ApiOutlined />
        },
        {
            key: '/token/collection-tasks',
            label: 'Collection Tasks',
            path: '/token/collection-tasks',
            icon: <FileTextOutlined />
        }
    ]
};

const navSections: NavSection[] = [
    {
        key: 'markets',
        label: 'Markets',
        children: [
            marketRadarNavItem,
            sportsNavItem,
            managedOONavItem,
            {
                key: '/fifa-market-dashboard',
                label: 'FIFA Market Dashboard',
                path: '/fifa-market-dashboard',
                icon: <TrophyOutlined />,
                module: AccountDataModule.FIFAMarketDashboard
            },
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
        children: [tokenNavItem, {key: '/wallet', label: 'Wallets', path: '/wallet', icon: <WalletOutlined />, module: AccountDataModule.Wallet}]
    },
    {
        key: 'operations',
        label: 'Operations',
        children: [
            {key: '/profit-sharing', label: 'Profit Sharing', path: '/profit-sharing', icon: <PieChartOutlined />, availability: 'profit-sharing'},
            {key: '/admin/profit-sharing', label: 'Profit Sharing Admin', path: '/admin/profit-sharing', icon: <SettingOutlined />, access: 'admin'},
            {key: '/notifications', label: 'Notifications', path: '/notifications', icon: <BellOutlined />, module: AccountDataModule.Notifications},
            {key: '/service-status', label: 'Service Status', path: '/service-status', icon: <HeartOutlined />, access: 'admin'},
            {key: '/etherscan-gateways', label: 'Etherscan Gateways', path: '/etherscan-gateways', icon: <ApiOutlined />, access: 'admin'}
        ]
    }
];

const navItems = navSections.flatMap(section => section.children);

const accountRouteMetadata = [
    {path: '/account/profile', section: 'Account', label: 'Profile'},
    {path: '/account/appearance', section: 'Account', label: 'Appearance'},
    {path: '/account/security', section: 'Account', label: 'Security'},
    {path: '/account/access', section: 'Account', label: 'Access & session'},
    {path: '/admin/accounts', section: 'Administration', label: 'Accounts'},
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
    const targetKey = selectedKey(pathname);
    const section = navSections.find(candidate => navTrail(candidate.children, targetKey).length > 0);
    const trail = section ? navTrail(section.children, targetKey) : [];
    if (section) {
        return [section.label, ...trail.map(item => item.label)].map(title => ({title}));
    }
    const accountRoute = [...accountRouteMetadata]
        .sort((left, right) => right.path.length - left.path.length)
        .find(item => pathname === item.path || pathname.startsWith(`${item.path}/`));
    return accountRoute ? [{title: accountRoute.section}, {title: accountRoute.label}] : [];
};

const routeTitle = (pathname: string) => {
    const selected = selectedKey(pathname);
    const business = flattenNav(navItems).find(item => item.key === selected);
    if (business) {
        return business.label;
    }
    return [...accountRouteMetadata].sort((left, right) => right.path.length - left.path.length).find(item => pathname === item.path || pathname.startsWith(`${item.path}/`))
        ?.label;
};

const usePreferences = () => {
    const [pref, setPref] = React.useState<ViewPreferences>(null);
    React.useEffect(() => {
        const sub = services.viewPreferences.getPreferences().subscribe(setPref);
        return () => sub.unsubscribe();
    }, []);
    return pref;
};

export async function loadAppBootstrapWithRetry(load: () => Promise<AppBootstrap>, delays: number[] = bootstrapRetryDelays, sleep: (delayMs: number) => Promise<unknown> = wait) {
    let lastError: Error = null;
    for (let attempt = 0; attempt <= delays.length; attempt++) {
        try {
            return await load();
        } catch (err) {
            lastError = err instanceof Error ? err : new Error(String(err));
            if (attempt < delays.length) {
                await sleep(delays[attempt]);
            }
        }
    }
    throw lastError;
}

const loadAccessState = (user: UserInfo): AccessState => ({
    user,
    isAdmin: user.administrator,
    moduleAccess: moduleAccessLevels(user.access, user.administrator),
    revision: user.access.revision
});

const isPendingAccess = (access: AccessState) => accountStatusForAccess(access.user.access, access.isAdmin) === AccountStatus.Pending;

const moduleLandingPaths: Partial<Record<AccountDataModule, string>> = {
    [AccountDataModule.MarketRadar]: '/market-radar',
    [AccountDataModule.SportsLive]: '/sports-live',
    [AccountDataModule.SportsHistory]: '/sports-history',
    [AccountDataModule.ManagedOO]: '/managed-oo/proposals',
    [AccountDataModule.FIFAMarketDashboard]: '/fifa-market-dashboard',
    [AccountDataModule.WorldCupCorners]: '/world-cup-corners',
    [AccountDataModule.Token]: '/token/projects',
    [AccountDataModule.Wallet]: '/wallet',
    [AccountDataModule.Notifications]: '/notifications'
};

const firstAuthorizedBusinessPath = (access: AccessState): string | undefined => {
    for (const definition of accountDataModules) {
        const path = moduleLandingPaths[definition.module];
        if (path && (access.isAdmin || access.moduleAccess[definition.module] >= AccountDataAccess.Read)) {
            return path;
        }
    }
    return access.isAdmin || access.user.access.profitSharingEnabled ? '/profit-sharing' : undefined;
};

const mergeMonotonicAccountProjection = (previous: AccessState | null, incoming: AccessState): AccessState => {
    if (!previous || previous.user.accountId !== incoming.user.accountId || previous.user.iss !== incoming.user.iss) {
        return incoming;
    }
    const profile = incoming.user.profile.revision < previous.user.profile.revision ? previous.user.profile : incoming.user.profile;
    const preferences = incoming.user.preferences.revision < previous.user.preferences.revision ? previous.user.preferences : incoming.user.preferences;
    if (profile === incoming.user.profile && preferences === incoming.user.preferences) {
        return incoming;
    }
    return {...incoming, user: {...incoming.user, profile, preferences}};
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

const ForbiddenPage = () => <Result status='403' title='403' subTitle='You do not have permission to access this page.' />;

const narrowShellQuery = '(max-width: 900px)';
const desktopExpandedSidebarWidth = 272;
const mobileExpandedSidebarWidth = 248;

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

const AppRoutes = (props: {
    access: AccessState;
    preferences: ViewPreferences;
    settings: AuthSettings;
    themeChanging: boolean;
    onThemeChange: (theme: ThemeMode) => Promise<void>;
    loggingOut: boolean;
    onLogout: () => void;
}) => {
    const pending = isPendingAccess(props.access);
    const moduleRoute = (module: AccountDataModule, element: React.ReactElement) =>
        props.access.isAdmin || props.access.moduleAccess[module] >= AccountDataAccess.Read ? element : <Navigate replace={true} to='/account/access' />;
    const profitSharingRoute = (element: React.ReactElement) =>
        props.access.isAdmin || props.access.user.access.profitSharingEnabled ? element : <Navigate replace={true} to='/account/access' />;
    const adminRoute = (element: React.ReactElement) => (props.access.isAdmin ? element : pending ? <Navigate replace={true} to='/account/access' /> : <ForbiddenPage />);
    const accountCenterProps = {
        preferences: props.preferences,
        themeChanging: props.themeChanging,
        onThemeChange: props.onThemeChange,
        loggingOut: props.loggingOut,
        onLogout: props.onLogout
    };
    return (
        <Routes>
            <Route path='/' element={<Navigate replace={true} to={pending ? '/account/access' : '/account/profile'} />} />
            <Route path='/wallet' element={moduleRoute(AccountDataModule.Wallet, <WalletsPage />)} />
            <Route path='/market-radar' element={moduleRoute(AccountDataModule.MarketRadar, <MarketRadarHotPage />)} />
            <Route path='/market-radar/realtime' element={moduleRoute(AccountDataModule.MarketRadar, <MarketRadarRealtimePage />)} />
            <Route path='/market-radar/movers' element={moduleRoute(AccountDataModule.MarketRadar, <MarketRadarMoversPage />)} />
            <Route path='/sports-live' element={moduleRoute(AccountDataModule.SportsLive, <SportsLivePage />)} />
            <Route path='/sports-history' element={moduleRoute(AccountDataModule.SportsHistory, <SportsHistoryPage />)} />
            <Route path='/world-cup-corners' element={moduleRoute(AccountDataModule.WorldCupCorners, <WorldCupCornersPage />)} />
            <Route path='/managed-oo/proposals' element={moduleRoute(AccountDataModule.ManagedOO, <ManagedOOProposalsPage />)} />
            <Route path='/managed-oo/disputes' element={moduleRoute(AccountDataModule.ManagedOO, <ManagedOODisputesPage />)} />
            <Route path='/fifa-market-dashboard' element={moduleRoute(AccountDataModule.FIFAMarketDashboard, <FIFAMarketDashboardPage />)} />
            <Route path='/notifications' element={moduleRoute(AccountDataModule.Notifications, <NotificationsPage />)} />
            <Route path='/notifications/:id' element={moduleRoute(AccountDataModule.Notifications, <NotificationsDetailPage />)} />
            <Route path='/account/profile' element={<AccountCenterPage section='profile' {...accountCenterProps} />} />
            <Route path='/account/appearance' element={<AccountCenterPage section='appearance' {...accountCenterProps} />} />
            <Route
                path='/account/security'
                element={
                    props.access.user.access.apiKeyEnabled && !props.access.isAdmin ? (
                        <AccountCenterPage section='security' {...accountCenterProps} />
                    ) : (
                        <Navigate replace={true} to='/account/access' />
                    )
                }
            />
            <Route path='/account/access' element={<AccountCenterPage section='access' {...accountCenterProps} />} />
            <Route path='/admin/accounts' element={adminRoute(<AdminAccountsPage />)} />
            <Route path='/profit-sharing' element={profitSharingRoute(<ProfitSharingRoundsPage />)} />
            <Route path='/profit-sharing/:slug' element={profitSharingRoute(<ProfitSharingRoundPage />)} />
            <Route path='/admin/profit-sharing' element={adminRoute(<ProfitSharingAdminRoundsPage />)} />
            <Route path='/admin/profit-sharing/:slug' element={adminRoute(<ProfitSharingAdminRoundPage />)} />
            <Route path='/service-status' element={adminRoute(<ServiceStatusPage />)} />
            <Route path='/etherscan-gateways' element={adminRoute(<EtherscanGatewaysPage />)} />
            <Route path='/help' element={<HelpPage help={props.settings.help} />} />
            <Route path='/token' element={moduleRoute(AccountDataModule.Token, <Navigate replace={true} to='/token/projects' />)} />
            <Route path='/token/projects' element={moduleRoute(AccountDataModule.Token, <ProjectsPage />)} />
            <Route path='/token/projects/:projectID' element={moduleRoute(AccountDataModule.Token, <ProjectDetailPage />)} />
            <Route path='/token/contract-codes' element={moduleRoute(AccountDataModule.Token, <ContractCodesPage />)} />
            <Route path='/token/contract-codes/:codeHash' element={moduleRoute(AccountDataModule.Token, <ContractCodeDetailPage />)} />
            <Route path='/token/contract-code-blocklist' element={moduleRoute(AccountDataModule.Token, <ContractCodeBlocklistPage />)} />
            <Route path='/token/wallet-blocklist' element={moduleRoute(AccountDataModule.Token, <WalletBlocklistPage />)} />
            <Route path='/token/node-statuses' element={moduleRoute(AccountDataModule.Token, <NodeStatusesPage />)} />
            <Route path='/token/chain-processing' element={moduleRoute(AccountDataModule.Token, <ChainProcessingPage />)} />
            <Route path='/token/collection-tasks' element={moduleRoute(AccountDataModule.Token, <CollectionTasksPage />)} />
            <Route path='*' element={<Navigate replace={true} to={pending ? '/account/access' : '/account/profile'} />} />
        </Routes>
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
    const [themeChanging, setThemeChanging] = React.useState(false);
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
        accessGenerationRef.current += 1;
        accessRefreshAllowedRef.current = false;
        accessRef.current = null;
        accessRefreshRef.current = null;
        accessDeniedRefreshRef.current = null;
        accessRefreshedAtRef.current = 0;
        pendingAccessRef.current = false;
        setAccessRefreshedAt(0);
        requests.invalidatePendingRequestErrors();
        requests.abortAuthorizationRequests();
        setAccountMenuOpen(false);
        setThemeChanging(false);
        setLoggingOut(false);
        setSession(status === 'maintenance' ? {status: 'maintenance'} : {status: 'anonymous'});
        clearAsyncDataCache();
        clearProjectsReturnSnapshots();
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
                const previous = accessRef.current;
                const next = mergeMonotonicAccountProjection(previous, loadAccessState(user));
                services.viewPreferences.syncServerTheme(localThemeMode(next.user.preferences.theme));
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
                        requests.invalidatePendingRequestErrors();
                        requests.abortAuthorizationRequests();
                        clearAsyncDataCache();
                        clearProjectsReturnSnapshots();
                    } else {
                        if (priorAccess.user.access.apiKeyEnabled && !next.user.access.apiKeyEnabled) {
                            requests.abortAuthorizationFeatureRequests('api-key');
                        }
                        if (priorAccess.user.access.profitSharingEnabled && !next.user.access.profitSharingEnabled) {
                            requests.abortAuthorizationFeatureRequests('profit-sharing');
                        }
                        accountDataModules.forEach(definition => {
                            const prior = priorAccess.moduleAccess[definition.module];
                            const current = next.moduleAccess[definition.module];
                            if (prior >= AccountDataAccess.Read && current < AccountDataAccess.Read) {
                                requests.abortAuthorizationRequests(definition.module);
                                clearAsyncDataCache(definition.module);
                                if (definition.module === AccountDataModule.Token) {
                                    clearProjectsReturnSnapshots();
                                }
                            } else if (prior >= AccountDataAccess.ReadWrite && current < AccountDataAccess.ReadWrite) {
                                requests.abortAuthorizationRequests(definition.module, 'write');
                            }
                        });
                        if (!isPendingAccess(priorAccess) && isPendingAccess(next)) {
                            clearAsyncDataCache();
                            clearProjectsReturnSnapshots();
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
        if (access) {
            services.viewPreferences.syncServerTheme(localThemeMode(access.user.preferences.theme));
        }
    }, [access?.user.preferences.revision, access?.user.accountId]);

    React.useEffect(() => {
        if (narrowShell) {
            setMobileSidebarOpen(false);
            setAccountMenuOpen(false);
        }
    }, [location.pathname, narrowShell]);

    React.useLayoutEffect(() => {
        if (!narrowShell || !mobileSidebarOpen) {
            return;
        }
        const sidebar = sidebarRef.current;
        const background = shellBackgroundRef.current;
        sidebar?.querySelector<HTMLElement>('.athena-shell__mobile-close, .athena-brand, [role="menuitem"]')?.focus();
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
            background?.removeAttribute('inert');
            background?.removeAttribute('aria-hidden');
            document.removeEventListener('keydown', handleDialogKeyboard, true);
            window.requestAnimationFrame(() => mobileSidebarToggleRef.current?.focus());
        };
    }, [mobileSidebarOpen, narrowShell]);

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
            navigate(readLoginReturnTo(location.search), {replace: true});
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
            if (isAccountDataAccessDeniedError(err)) {
                void refreshAfterAccessDenied();
                return;
            }
            const maintenance = isAccountMaintenanceError(err);
            if (!maintenance && err.status !== 401) {
                return;
            }
            if (window.location.pathname.startsWith(`${base.replace(/\/$/, '')}/login`)) {
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
        if (target?.path) {
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
            baseHref: base
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

    const changeTheme = React.useCallback(
        async (theme: ThemeMode) => {
            const current = accessRef.current;
            if (!current || themeChanging || localThemeMode(current.user.preferences.theme) === theme) {
                setAccountMenuOpen(false);
                return;
            }
            const generation = accessGenerationRef.current;
            const isCurrentAccountSession = () => {
                const latest = accessRef.current;
                return (
                    generation === accessGenerationRef.current &&
                    accessRefreshAllowedRef.current &&
                    latest !== null &&
                    latest.user.accountId === current.user.accountId &&
                    latest.user.iss === current.user.iss
                );
            };
            setThemeChanging(true);
            services.viewPreferences.syncServerTheme(theme);
            try {
                const preferences = await services.accounts.updatePreferences(serverThemeMode(theme), current.user.preferences.revision);
                const latest = accessRef.current;
                if (isCurrentAccountSession() && latest) {
                    const next = mergeMonotonicAccountProjection(latest, {...latest, user: {...latest.user, preferences}});
                    services.viewPreferences.syncServerTheme(localThemeMode(next.user.preferences.theme));
                    accessRef.current = next;
                    setSession({status: 'authenticated', access: next});
                    notifications.success('Appearance updated', `${accountThemeLabel(next.user.preferences.theme)} theme is now synced across devices.`);
                }
            } catch (err) {
                const latest = accessRef.current;
                if (isCurrentAccountSession() && latest) {
                    services.viewPreferences.syncServerTheme(localThemeMode(latest.user.preferences.theme));
                    try {
                        await refreshAccess(true);
                    } catch {
                        // Keep the last authoritative projection if a refresh is unavailable.
                    }
                    const refreshed = accessRef.current;
                    if (isCurrentAccountSession() && refreshed) {
                        services.viewPreferences.syncServerTheme(localThemeMode(refreshed.user.preferences.theme));
                        notifications.error('Could not update appearance', requestErrorMessage(err));
                    }
                }
            } finally {
                if (generation === accessGenerationRef.current) {
                    setThemeChanging(false);
                    setAccountMenuOpen(false);
                }
            }
        },
        [notifications, refreshAccess, themeChanging]
    );

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
            navigate('/login', {replace: true});
        } catch (err) {
            notifications.error('Logout failed', requestErrorMessage(err, 'Could not log out. Your session is still active.'));
            setLoggingOut(false);
        }
    }, [endSession, loggingOut, navigate, notifications]);

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
              {
                  type: 'group',
                  label: (
                      <span className='athena-account-menu__label'>
                          <span>
                              <BgColorsOutlined /> Appearance
                          </span>
                          <small>{themeChanging ? 'Saving…' : accountThemeLabel(props.pref.theme)}</small>
                      </span>
                  ),
                  children: [
                      {key: 'theme:system', label: 'System', disabled: themeChanging, icon: props.pref.theme === 'system' ? <CheckOutlined /> : <DesktopOutlined />},
                      {key: 'theme:light', label: 'Light', disabled: themeChanging, icon: props.pref.theme === 'light' ? <CheckOutlined /> : <SunOutlined />},
                      {key: 'theme:dark', label: 'Dark', disabled: themeChanging, icon: props.pref.theme === 'dark' ? <CheckOutlined /> : <MoonOutlined />}
                  ]
              },
              {key: '/account/access', label: 'Access', icon: <KeyOutlined />},
              ...(access.user.access.apiKeyEnabled && !access.isAdmin ? [{key: '/account/security', label: 'Security', icon: <SettingOutlined />}] : []),
              ...(access.isAdmin ? [{key: '/admin/accounts', label: 'Manage accounts', icon: <UserOutlined />}] : []),
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
        if (item.key.startsWith('theme:')) {
            void changeTheme(item.key.slice('theme:'.length) as ThemeMode);
            return;
        }
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
                <Route path='/login' element={<LoginPage />} />
                <Route path='*' element={<Navigate replace={true} to='/login' />} />
            </Routes>
        );
    } else if (access && !isLoginPath) {
        routes = (
            <AppRoutes
                access={access}
                preferences={props.pref}
                settings={props.settings}
                themeChanging={themeChanging}
                onThemeChange={changeTheme}
                loggingOut={loggingOut}
                onLogout={() => void logout()}
            />
        );
    } else {
        routes = <div className='athena-boot'>Loading Athena...</div>;
    }
    const content = isLoginPath ? (
        routes
    ) : (
        <AntLayout className='athena-shell' style={shellStyle}>
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
                {access && (!narrowShell || mobileSidebarOpen) && (
                    <div className='athena-sidebar-footer'>
                        <Dropdown
                            open={accountMenuOpen}
                            trigger={['click']}
                            placement='topLeft'
                            overlayClassName='athena-account-menu'
                            destroyOnHidden={true}
                            menu={{items: accountMenuItems, onClick: onAccountMenuClick, selectable: false}}
                            getPopupContainer={trigger => (trigger.closest('.athena-sidebar-footer') as HTMLElement) || sidebarRef.current || document.body}
                            onOpenChange={setAccountMenuOpen}>
                            <button
                                className='athena-account-trigger'
                                type='button'
                                aria-label='Open account menu'
                                aria-haspopup='menu'
                                aria-expanded={accountMenuOpen}
                                title={sidebarCollapsed ? access.user.profile.displayName || access.user.username : undefined}>
                                <AccountAvatar profile={access.user.profile} username={access.user.username} size={38} />
                                {!sidebarCollapsed && (
                                    <span className='athena-account-trigger__copy'>
                                        <strong>{access.user.profile.displayName || access.user.username}</strong>
                                        <small>@{access.user.username}</small>
                                    </span>
                                )}
                                {!sidebarCollapsed && (
                                    <span className='athena-account-trigger__more' aria-hidden='true'>
                                        •••
                                    </span>
                                )}
                            </button>
                        </Dropdown>
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
                        <Tooltip title={sidebarCollapsed ? 'Open navigation' : 'Close navigation'}>
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

const Bootstrap = () => {
    const pref = usePreferences();
    const [appBootstrap, setAppBootstrap] = React.useState<AppBootstrap>(null);
    const [bootstrapError, setBootstrapError] = React.useState<Error>(null);
    const [bootstrapRetry, setBootstrapRetry] = React.useState(0);

    React.useEffect(() => {
        let active = true;
        setBootstrapError(null);
        setAppBootstrap(null);
        loadAppBootstrapWithRetry(() => services.authService.bootstrap())
            .then(result => {
                if (!active) {
                    return;
                }
                if (result.session.status === AppBootstrapSessionStatus.Authenticated && result.session.userInfo) {
                    services.viewPreferences.syncServerTheme(localThemeMode(result.session.userInfo.preferences.theme));
                }
                setAppBootstrap(result);
                if (result.settings.uiCssURL) {
                    const link = document.createElement('link');
                    link.href = result.settings.uiCssURL;
                    link.rel = 'stylesheet';
                    link.type = 'text/css';
                    document.head.appendChild(link);
                }
            })
            .catch(err => {
                if (active) {
                    setBootstrapError(err instanceof Error ? err : new Error(String(err)));
                }
            });
        return () => {
            active = false;
        };
    }, [bootstrapRetry]);

    if (bootstrapError) {
        return (
            <div className='athena-recoverable'>
                <Result
                    status='warning'
                    title='API 服务暂不可用'
                    subTitle='Athena 后端网关还没有准备好，或正在重启。请稍后重试。'
                    extra={
                        <Space orientation='vertical' size={12}>
                            <Button type='primary' onClick={() => setBootstrapRetry(value => value + 1)}>
                                重试
                            </Button>
                            <Typography.Text type='secondary'>{bootstrapError.message}</Typography.Text>
                        </Space>
                    }
                />
            </div>
        );
    }

    if (!pref || !appBootstrap) {
        return <div className='athena-boot'>Loading Athena...</div>;
    }

    const isDark = services.viewPreferences.resolvedTheme(pref.theme) === 'dark';
    return (
        <ConfigProvider
            theme={{
                algorithm: isDark ? antTheme.darkAlgorithm : antTheme.defaultAlgorithm,
                token: {
                    borderRadius: 8,
                    colorPrimary: isDark ? '#e76f51' : '#c94f2d',
                    colorInfo: isDark ? '#3b82f6' : '#2563eb',
                    colorSuccess: isDark ? '#22c55e' : '#168a45',
                    colorWarning: isDark ? '#f59e0b' : '#b86a00',
                    colorError: isDark ? '#ef4444' : '#c9363e',
                    colorLink: isDark ? '#60a5fa' : '#2563eb',
                    colorBgBase: isDark ? '#0b0f14' : '#f5f6f8',
                    colorBgLayout: isDark ? '#0b0f14' : '#f5f6f8',
                    colorBgContainer: isDark ? '#111820' : '#ffffff',
                    colorBgElevated: isDark ? '#16202a' : '#f8fafc',
                    colorText: isDark ? '#eef3f8' : '#17202b',
                    colorTextSecondary: isDark ? '#94a3b8' : '#5f6b7a',
                    colorTextLightSolid: isDark ? '#101820' : '#ffffff',
                    colorBorder: isDark ? '#263341' : '#dfe4ea',
                    colorBorderSecondary: isDark ? '#1f2a35' : '#e8ecf0',
                    colorSplit: isDark ? '#263341' : '#dfe4ea',
                    colorFillSecondary: isDark ? '#1b2632' : '#eef1f4',
                    controlOutline: isDark ? 'rgba(96,165,250,0.45)' : 'rgba(37,99,235,0.35)',
                    boxShadow: isDark ? '0 18px 48px rgba(0,0,0,0.34)' : '0 18px 48px rgba(23,32,43,0.10)',
                    boxShadowSecondary: isDark ? '0 12px 32px rgba(0,0,0,0.28)' : '0 8px 24px rgba(23,32,43,0.08)',
                    fontFamily: 'Heebo, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
                    fontSize: 14,
                    controlHeight: 36,
                    controlHeightSM: 30,
                    motionDurationFast: '0.15s',
                    motionDurationMid: '0.2s',
                    wireframe: false
                }
            }}>
            <AntApp>
                <Shell pref={pref} initialSession={appBootstrap.session} settings={appBootstrap.settings} />
            </AntApp>
        </ConfigProvider>
    );
};

export const App = () => {
    const [router] = React.useState(() =>
        createBrowserRouter([{path: '*', element: <AppEntry />}], {
            basename: base,
            future: {v7_relativeSplatPath: true}
        })
    );
    return <RouterProvider router={router} future={{v7_startTransition: true}} />;
};

const RegistrationBootstrap = () => (
    <ConfigProvider>
        <AntApp>
            <RegisterPage />
        </AntApp>
    </ConfigProvider>
);

const AppEntry = () => {
    const location = useLocation();
    return location.pathname === '/register' ? <RegistrationBootstrap /> : <Bootstrap />;
};
