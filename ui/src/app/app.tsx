import '@fortawesome/fontawesome-free/css/all.css';
import 'antd/dist/reset.css';
import '../assets/fonts.css';
import './styles.css';

import {
    ApiOutlined,
    BarChartOutlined,
    BellOutlined,
    CodeOutlined,
    DashboardOutlined,
    FileTextOutlined,
    HeartOutlined,
    MenuFoldOutlined,
    MenuUnfoldOutlined,
    MoonOutlined,
    QuestionCircleOutlined,
    SettingOutlined,
    SunOutlined,
    TrophyOutlined,
    UserOutlined,
    WalletOutlined
} from '@ant-design/icons';
import {App as AntApp, Breadcrumb, Button, ConfigProvider, Dropdown, Layout as AntLayout, Menu, Result, Space, theme as antTheme, Tooltip, Typography} from 'antd';
import type {MenuProps} from 'antd';
import * as React from 'react';
import {BrowserRouter, Navigate, Route, Routes, useLocation, useNavigate} from 'react-router-dom';
import {Subscription} from 'rxjs';
import {AuthSettingsCtx, Provider} from './shared/context';
import {AuthSettings, Permission, UserInfo} from './shared/models';
import {services, ViewPreferences} from './shared/services';
import requests, {isAccountMaintenanceError} from './shared/services/requests';
import {BrandMark, clearAsyncDataCache} from './components';
import {clearProjectsReturnSnapshots} from './pages/project-navigation';
import {
    ContractCodeBlocklistPage,
    ChainProcessingPage,
    CollectionTasksPage,
    ContractCodeDetailPage,
    ContractCodesPage,
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
    SettingsPage,
    ServiceStatusPage,
    UserInfoPage,
    WalletBlocklistPage,
    WalletsPage,
    WorldCupCornersPage,
    FIFAMarketDashboardPage
} from './pages';

services.viewPreferences.init();

const bases = document.getElementsByTagName('base');
const base = bases.length > 0 ? bases[0].getAttribute('href') || '/' : '/';
requests.setBaseHRef(base);

const authSettingsRetryDelays = [500, 1000, 2000, 3000];
const maintenanceLoginPath = '/login?reason=maintenance';

const wait = (delayMs: number) => new Promise(resolve => window.setTimeout(resolve, delayMs));

interface NavItem {
    key: string;
    label: string;
    icon: React.ReactNode;
    path?: string;
    children?: NavItem[];
    permission?: Permission;
}

interface NavSection {
    key: string;
    label: string;
    children: NavItem[];
}

interface AccessState {
    user: UserInfo;
    permissions: Record<string, boolean>;
}

const rbacResources = {
    notifications: 'notifications',
    marketRadar: 'market-radar',
    sportsLive: 'sports-live',
    sportsHistory: 'sports-history',
    managedOO: 'managed-oo',
    fifaMarketDashboard: 'fifa-market-dashboard',
    worldCupCorners: 'world-cup-corners',
    tokenapi: 'tokenapi',
    serviceStatus: 'service-status',
    wallets: 'wallets'
};

const rbacActions = {
    get: 'get',
    update: 'update',
    invoke: 'invoke'
};

const tokenapiSubresources = {
    projects: 'projects',
    contractCodes: 'contract-codes',
    contractCodeBlocklist: 'contract-code-blocklist',
    walletBlocklist: 'wallet-blocklist',
    nodeStatuses: 'node-statuses',
    chainCheckpoints: 'chain-checkpoints',
    collectionTasks: 'collection-tasks'
};

const permission = (resource: string, action: string, subresource = '*'): Permission => ({resource, action, subresource});
const tokenapiPermission = (subresource: string) => permission(rbacResources.tokenapi, rbacActions.get, subresource);
const serviceStatusPermission = permission(rbacResources.serviceStatus, rbacActions.get);
const serviceStatusInvokePermission = permission(rbacResources.serviceStatus, rbacActions.invoke);
const permissionKey = (perm: Permission) => `${perm.resource}:${perm.action}:${perm.subresource}`;
const hasPermission = (access: AccessState, perm?: Permission) => !perm || access?.permissions[permissionKey(perm)] === true;

const marketRadarNavItem: NavItem = {
    key: 'market-radar',
    label: 'Market Radar',
    icon: <DashboardOutlined />,
    children: [
        {key: '/market-radar', label: 'Hot Markets', path: '/market-radar', icon: <DashboardOutlined />, permission: permission(rbacResources.marketRadar, rbacActions.get)},
        {
            key: '/market-radar/realtime',
            label: 'Realtime',
            path: '/market-radar/realtime',
            icon: <DashboardOutlined />,
            permission: permission(rbacResources.marketRadar, rbacActions.get)
        },
        {
            key: '/market-radar/movers',
            label: 'Movers',
            path: '/market-radar/movers',
            icon: <BarChartOutlined />,
            permission: permission(rbacResources.marketRadar, rbacActions.get)
        }
    ]
};

const sportsNavItem: NavItem = {
    key: 'sports',
    label: 'Sports',
    icon: <TrophyOutlined />,
    children: [
        {key: '/sports-live', label: 'Sports Live', path: '/sports-live', icon: <DashboardOutlined />, permission: permission(rbacResources.sportsLive, rbacActions.get)},
        {
            key: '/sports-history',
            label: 'Sports History',
            path: '/sports-history',
            icon: <DashboardOutlined />,
            permission: permission(rbacResources.sportsHistory, rbacActions.get)
        }
    ]
};

const managedOONavItem: NavItem = {
    key: 'managed-oo',
    label: 'Managed OO',
    icon: <ApiOutlined />,
    children: [
        {
            key: '/managed-oo/proposals',
            label: 'Proposals',
            path: '/managed-oo/proposals',
            icon: <ApiOutlined />,
            permission: permission(rbacResources.managedOO, rbacActions.get)
        },
        {
            key: '/managed-oo/disputes',
            label: 'Disputes',
            path: '/managed-oo/disputes',
            icon: <ApiOutlined />,
            permission: permission(rbacResources.managedOO, rbacActions.get)
        }
    ]
};

const tokenNavItem: NavItem = {
    key: 'token',
    label: 'Token',
    icon: <DashboardOutlined />,
    children: [
        {key: '/token/projects', label: 'Projects', path: '/token/projects', icon: <FileTextOutlined />, permission: tokenapiPermission(tokenapiSubresources.projects)},
        {
            key: '/token/contract-codes',
            label: 'Contract Codes',
            path: '/token/contract-codes',
            icon: <CodeOutlined />,
            permission: tokenapiPermission(tokenapiSubresources.contractCodes)
        },
        {
            key: '/token/contract-code-blocklist',
            label: 'Contract Code Blocklist',
            path: '/token/contract-code-blocklist',
            icon: <ApiOutlined />,
            permission: tokenapiPermission(tokenapiSubresources.contractCodeBlocklist)
        },
        {
            key: '/token/wallet-blocklist',
            label: 'Wallet Blocklist',
            path: '/token/wallet-blocklist',
            icon: <WalletOutlined />,
            permission: tokenapiPermission(tokenapiSubresources.walletBlocklist)
        },
        {
            key: '/token/node-statuses',
            label: 'Node Status',
            path: '/token/node-statuses',
            icon: <ApiOutlined />,
            permission: tokenapiPermission(tokenapiSubresources.nodeStatuses)
        },
        {
            key: '/token/chain-processing',
            label: 'Chain Processing',
            path: '/token/chain-processing',
            icon: <ApiOutlined />,
            permission: tokenapiPermission(tokenapiSubresources.chainCheckpoints)
        },
        {
            key: '/token/collection-tasks',
            label: 'Collection Tasks',
            path: '/token/collection-tasks',
            icon: <FileTextOutlined />,
            permission: tokenapiPermission(tokenapiSubresources.collectionTasks)
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
                permission: permission(rbacResources.fifaMarketDashboard, rbacActions.get)
            },
            {
                key: '/world-cup-corners',
                label: 'World Cup Corners',
                path: '/world-cup-corners',
                icon: <BarChartOutlined />,
                permission: permission(rbacResources.worldCupCorners, rbacActions.get)
            }
        ]
    },
    {
        key: 'token-risk',
        label: 'Token & Risk',
        children: [tokenNavItem, {key: '/wallet', label: 'Wallets', path: '/wallet', icon: <WalletOutlined />, permission: permission(rbacResources.wallets, rbacActions.get)}]
    },
    {
        key: 'operations',
        label: 'Operations',
        children: [
            {key: '/notifications', label: 'Notifications', path: '/notifications', icon: <BellOutlined />, permission: permission(rbacResources.notifications, rbacActions.get)},
            {key: '/service-status', label: 'Service Status', path: '/service-status', icon: <HeartOutlined />, permission: serviceStatusPermission},
            {key: '/etherscan-gateways', label: 'Etherscan Gateways', path: '/etherscan-gateways', icon: <ApiOutlined />, permission: serviceStatusPermission}
        ]
    },
    {
        key: 'system',
        label: 'System',
        children: [
            {key: '/settings', label: 'Settings', path: '/settings', icon: <SettingOutlined />},
            {key: '/user-info', label: 'User Info', path: '/user-info', icon: <UserOutlined />},
            {key: '/help', label: 'Help', path: '/help', icon: <QuestionCircleOutlined />}
        ]
    }
];

const navItems = navSections.flatMap(section => section.children);

const flattenNav = (items: NavItem[]): NavItem[] => items.flatMap(item => [item, ...(item.children ? flattenNav(item.children) : [])]);

const filterNavItems = (items: NavItem[], access: AccessState): NavItem[] =>
    items
        .map(item => {
            const children = item.children ? filterNavItems(item.children, access) : undefined;
            if (children) {
                return children.length > 0 && hasPermission(access, item.permission) ? {...item, children} : null;
            }
            return hasPermission(access, item.permission) ? item : null;
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
    return exact?.key || '/user-info';
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
    return [section?.label, ...trail.map(item => item.label)].filter(Boolean).map(title => ({title}));
};

const usePreferences = () => {
    const [pref, setPref] = React.useState<ViewPreferences>(null);
    React.useEffect(() => {
        const sub = services.viewPreferences.getPreferences().subscribe(setPref);
        return () => sub.unsubscribe();
    }, []);
    return pref;
};

export async function loadAuthSettingsWithRetry(
    load: () => Promise<AuthSettings>,
    delays: number[] = authSettingsRetryDelays,
    sleep: (delayMs: number) => Promise<unknown> = wait
) {
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
    permissions: Object.fromEntries((user.permissions || []).map(perm => [permissionKey(perm), true]))
});

const ForbiddenPage = () => <Result status='403' title='403' subTitle='You do not have permission to access this page.' />;

const RequirePermission = (props: {access: AccessState; permission: Permission; children: React.ReactElement}) =>
    hasPermission(props.access, props.permission) ? props.children : <ForbiddenPage />;

const narrowShellQuery = '(max-width: 900px)';

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

const AppRoutes = (props: {access: AccessState; onSessionEnded: () => void}) => {
    const visibleTokenDefault = filterNavItems(navItems, props.access)
        .find(item => item.key === 'token')
        ?.children?.find(item => item.path)?.path;
    const withPermission = (perm: Permission, element: React.ReactElement) => (
        <RequirePermission access={props.access} permission={perm}>
            {element}
        </RequirePermission>
    );
    return (
        <Routes>
            <Route path='/' element={<Navigate replace={true} to='/user-info' />} />
            <Route
                path='/wallet'
                element={withPermission(
                    permission(rbacResources.wallets, rbacActions.get),
                    <WalletsPage
                        canCreate={hasPermission(props.access, permission(rbacResources.wallets, rbacActions.update))}
                        canReveal={hasPermission(props.access, permission(rbacResources.wallets, rbacActions.invoke))}
                    />
                )}
            />
            <Route path='/market-radar' element={withPermission(permission(rbacResources.marketRadar, rbacActions.get), <MarketRadarHotPage />)} />
            <Route path='/market-radar/realtime' element={withPermission(permission(rbacResources.marketRadar, rbacActions.get), <MarketRadarRealtimePage />)} />
            <Route path='/market-radar/movers' element={withPermission(permission(rbacResources.marketRadar, rbacActions.get), <MarketRadarMoversPage />)} />
            <Route path='/sports-live' element={withPermission(permission(rbacResources.sportsLive, rbacActions.get), <SportsLivePage />)} />
            <Route
                path='/sports-history'
                element={withPermission(
                    permission(rbacResources.sportsHistory, rbacActions.get),
                    <SportsHistoryPage canRefresh={hasPermission(props.access, permission(rbacResources.sportsHistory, rbacActions.invoke))} />
                )}
            />
            <Route path='/world-cup-corners' element={withPermission(permission(rbacResources.worldCupCorners, rbacActions.get), <WorldCupCornersPage />)} />
            <Route
                path='/managed-oo/proposals'
                element={withPermission(
                    permission(rbacResources.managedOO, rbacActions.get),
                    <ManagedOOProposalsPage canScan={hasPermission(props.access, permission(rbacResources.managedOO, rbacActions.invoke))} />
                )}
            />
            <Route
                path='/managed-oo/disputes'
                element={withPermission(
                    permission(rbacResources.managedOO, rbacActions.get),
                    <ManagedOODisputesPage canScan={hasPermission(props.access, permission(rbacResources.managedOO, rbacActions.invoke))} />
                )}
            />
            <Route
                path='/fifa-market-dashboard'
                element={withPermission(
                    permission(rbacResources.fifaMarketDashboard, rbacActions.get),
                    <FIFAMarketDashboardPage canEdit={hasPermission(props.access, permission(rbacResources.fifaMarketDashboard, rbacActions.update))} />
                )}
            />
            <Route path='/notifications' element={withPermission(permission(rbacResources.notifications, rbacActions.get), <NotificationsPage />)} />
            <Route path='/notifications/:id' element={withPermission(permission(rbacResources.notifications, rbacActions.get), <NotificationsDetailPage />)} />
            <Route path='/settings/*' element={<SettingsPage />} />
            <Route path='/service-status' element={withPermission(serviceStatusPermission, <ServiceStatusPage />)} />
            <Route
                path='/etherscan-gateways'
                element={withPermission(serviceStatusPermission, <EtherscanGatewaysPage canRunProbe={hasPermission(props.access, serviceStatusInvokePermission)} />)}
            />
            <Route path='/user-info' element={<UserInfoPage onSessionEnded={props.onSessionEnded} />} />
            <Route path='/help' element={<HelpPage />} />
            <Route path='/token' element={visibleTokenDefault ? <Navigate replace={true} to={visibleTokenDefault} /> : <ForbiddenPage />} />
            <Route path='/token/projects' element={withPermission(tokenapiPermission(tokenapiSubresources.projects), <ProjectsPage />)} />
            <Route path='/token/projects/:projectID' element={withPermission(tokenapiPermission(tokenapiSubresources.projects), <ProjectDetailPage />)} />
            <Route path='/token/contract-codes' element={withPermission(tokenapiPermission(tokenapiSubresources.contractCodes), <ContractCodesPage />)} />
            <Route path='/token/contract-codes/:codeHash' element={withPermission(tokenapiPermission(tokenapiSubresources.contractCodes), <ContractCodeDetailPage />)} />
            <Route path='/token/contract-code-blocklist' element={withPermission(tokenapiPermission(tokenapiSubresources.contractCodeBlocklist), <ContractCodeBlocklistPage />)} />
            <Route path='/token/wallet-blocklist' element={withPermission(tokenapiPermission(tokenapiSubresources.walletBlocklist), <WalletBlocklistPage />)} />
            <Route path='/token/node-statuses' element={withPermission(tokenapiPermission(tokenapiSubresources.nodeStatuses), <NodeStatusesPage />)} />
            <Route path='/token/chain-processing' element={withPermission(tokenapiPermission(tokenapiSubresources.chainCheckpoints), <ChainProcessingPage />)} />
            <Route path='/token/collection-tasks' element={withPermission(tokenapiPermission(tokenapiSubresources.collectionTasks), <CollectionTasksPage />)} />
            <Route path='*' element={<Navigate replace={true} to='/user-info' />} />
        </Routes>
    );
};

const Shell = (props: {pref: ViewPreferences; authSettings: AuthSettings}) => {
    const navigate = useNavigate();
    const location = useLocation();
    const ant = AntApp.useApp();
    const narrowShell = useNarrowShell();
    const [desktopSidebarCollapsed, setDesktopSidebarCollapsed] = React.useState(props.pref.hideSidebar);
    const [mobileSidebarOpen, setMobileSidebarOpen] = React.useState(false);
    const sidebarRef = React.useRef<HTMLDivElement>(null);
    const shellBackgroundRef = React.useRef<HTMLElement>(null);
    const mobileSidebarToggleRef = React.useRef<HTMLButtonElement>(null);
    const accessGenerationRef = React.useRef(0);
    const sidebarCollapsed = narrowShell ? !mobileSidebarOpen : desktopSidebarCollapsed;
    const isLoginPath = location.pathname.startsWith('/login');
    const [access, setAccess] = React.useState<AccessState>(null);
    const [accessError, setAccessError] = React.useState<Error>(null);
    const [accessRetry, setAccessRetry] = React.useState(0);

    const endSession = React.useCallback(() => {
        accessGenerationRef.current += 1;
        requests.invalidatePendingRequestErrors();
        setAccess(null);
        setAccessError(null);
        clearAsyncDataCache();
        clearProjectsReturnSnapshots();
    }, []);

    React.useEffect(() => {
        setDesktopSidebarCollapsed(props.pref.hideSidebar);
    }, [props.pref.hideSidebar]);

    React.useEffect(() => {
        if (narrowShell) {
            setMobileSidebarOpen(false);
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
        if (isLoginPath) {
            endSession();
            return;
        }

        if (access) {
            return;
        }

        let active = true;
        const generation = accessGenerationRef.current;
        setAccessError(null);
        services.users
            .get()
            .then(user => {
                if (!active || generation !== accessGenerationRef.current) {
                    return;
                }
                if (!user.loggedIn) {
                    endSession();
                    navigate('/login', {replace: true});
                    return;
                }
                setAccess(loadAccessState(user));
            })
            .catch(err => {
                if (!active || generation !== accessGenerationRef.current) {
                    return;
                }
                if (isAccountMaintenanceError(err)) {
                    endSession();
                    navigate(maintenanceLoginPath, {replace: true});
                    return;
                }
                if (err?.status === 401) {
                    endSession();
                    navigate('/login', {replace: true});
                    return;
                }
                setAccessError(err instanceof Error ? err : new Error(err?.message || String(err)));
            });
        return () => {
            active = false;
        };
    }, [access, accessRetry, endSession, isLoginPath, navigate]);

    React.useEffect(() => {
        const subscription: Subscription = requests.onError.subscribe(err => {
            if (isLoginPath) {
                return;
            }
            const maintenance = isAccountMaintenanceError(err);
            if (!maintenance && err.status !== 401) {
                return;
            }
            if (window.location.pathname.startsWith(`${base.replace(/\/$/, '')}/login`)) {
                return;
            }
            endSession();
            navigate(maintenance ? maintenanceLoginPath : '/login', {replace: true});
        });
        return () => subscription?.unsubscribe();
    }, [endSession, isLoginPath, navigate]);

    React.useEffect(() => {
        const current = flattenNav(navItems).find(item => item.key === selectedKey(location.pathname));
        document.title = current ? `${current.label} · Athena` : 'Athena';
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

    const isDark = props.pref.theme === 'dark';
    const nextTheme: ViewPreferences['theme'] = isDark ? 'light' : 'dark';

    const userMenu: MenuProps['items'] = [
        {key: '/user-info', label: 'User Info', icon: <UserOutlined />},
        {key: '/settings', label: 'Settings', icon: <SettingOutlined />},
        {key: '/help', label: 'Help', icon: <QuestionCircleOutlined />}
    ];

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
    if (isLoginPath) {
        routes = (
            <Routes>
                <Route path='/login' element={<LoginPage />} />
                <Route path='*' element={<Navigate replace={true} to='/login' />} />
            </Routes>
        );
    } else if (accessError) {
        routes = (
            <div className='athena-recoverable'>
                <Result
                    status='warning'
                    title='Unable to load session'
                    subTitle='Athena could not load your account and permissions. Retry when the service is available.'
                    extra={
                        <Space orientation='vertical' size={12}>
                            <Button
                                type='primary'
                                onClick={() => {
                                    setAccessError(null);
                                    setAccessRetry(value => value + 1);
                                }}>
                                Retry
                            </Button>
                            <Typography.Text type='secondary'>{accessError.message}</Typography.Text>
                        </Space>
                    }
                />
            </div>
        );
    } else {
        routes = access ? <AppRoutes access={access} onSessionEnded={endSession} /> : <div className='athena-boot'>Loading Athena...</div>;
    }
    const content = isLoginPath ? (
        routes
    ) : (
        <AntLayout className='athena-shell'>
            <a className='athena-skip-link' href='#athena-main' aria-hidden={narrowShell && mobileSidebarOpen} tabIndex={narrowShell && mobileSidebarOpen ? -1 : undefined}>
                Skip to main content
            </a>
            <AntLayout.Sider
                className='athena-shell__sider'
                collapsible={true}
                collapsed={sidebarCollapsed}
                collapsedWidth={narrowShell ? 0 : 72}
                trigger={null}
                width={248}
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
                    <button
                        className='athena-brand'
                        type='button'
                        aria-label='Open Athena user information'
                        onClick={() => {
                            if (narrowShell) {
                                setMobileSidebarOpen(false);
                            }
                            navigate('/user-info');
                        }}>
                        <BrandMark size='small' />
                        {!sidebarCollapsed && (
                            <span className='athena-brand__copy'>
                                <strong>Athena</strong>
                                <small>Operations Console</small>
                            </span>
                        )}
                    </button>
                )}
                <nav id='athena-primary-navigation' aria-label='Primary navigation'>
                    {(!narrowShell || mobileSidebarOpen) && menu}
                </nav>
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
                    <div className='athena-shell__header-actions'>
                        <Tooltip title={`Switch to ${nextTheme} theme`}>
                            <Button
                                className='athena-shell__icon-button'
                                type='text'
                                aria-label={`Switch to ${nextTheme} theme`}
                                icon={isDark ? <SunOutlined /> : <MoonOutlined />}
                                onClick={() => services.viewPreferences.updatePreferences({theme: nextTheme})}
                            />
                        </Tooltip>
                        <Tooltip title='User menu'>
                            <Dropdown menu={{items: userMenu, onClick: item => navigate(item.key)}}>
                                <Button className='athena-shell__icon-button' type='text' aria-label='Open user menu' icon={<UserOutlined />} />
                            </Dropdown>
                        </Tooltip>
                    </div>
                </AntLayout.Header>
                <AntLayout.Content className='athena-shell__content' id='athena-main' tabIndex={-1}>
                    <Provider value={contextValue}>
                        <AuthSettingsCtx.Provider value={props.authSettings}>{routes}</AuthSettingsCtx.Provider>
                    </Provider>
                </AntLayout.Content>
            </AntLayout>
        </AntLayout>
    );

    return (
        <Provider value={contextValue}>
            <AuthSettingsCtx.Provider value={props.authSettings}>{content}</AuthSettingsCtx.Provider>
        </Provider>
    );
};

const Bootstrap = () => {
    const pref = usePreferences();
    const [authSettings, setAuthSettings] = React.useState<AuthSettings>(null);
    const [settingsError, setSettingsError] = React.useState<Error>(null);
    const [settingsRetry, setSettingsRetry] = React.useState(0);

    React.useEffect(() => {
        let active = true;
        setSettingsError(null);
        setAuthSettings(null);
        loadAuthSettingsWithRetry(() => services.authService.settings())
            .then(settings => {
                if (!active) {
                    return;
                }
                setAuthSettings(settings);
                if (settings.uiCssURL) {
                    const link = document.createElement('link');
                    link.href = settings.uiCssURL;
                    link.rel = 'stylesheet';
                    link.type = 'text/css';
                    document.head.appendChild(link);
                }
            })
            .catch(err => {
                if (active) {
                    setSettingsError(err instanceof Error ? err : new Error(String(err)));
                }
            });
        return () => {
            active = false;
        };
    }, [settingsRetry]);

    if (settingsError) {
        return (
            <div className='athena-recoverable'>
                <Result
                    status='warning'
                    title='API 服务暂不可用'
                    subTitle='Athena 后端网关还没有准备好，或正在重启。请稍后重试。'
                    extra={
                        <Space orientation='vertical' size={12}>
                            <Button type='primary' onClick={() => setSettingsRetry(value => value + 1)}>
                                重试
                            </Button>
                            <Typography.Text type='secondary'>{settingsError.message}</Typography.Text>
                        </Space>
                    }
                />
            </div>
        );
    }

    if (!pref || !authSettings) {
        return <div className='athena-boot'>Loading Athena...</div>;
    }

    const isDark = pref.theme === 'dark';
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
                <BrowserRouter basename={base} future={{v7_startTransition: true, v7_relativeSplatPath: true}}>
                    <Shell pref={pref} authSettings={authSettings} />
                </BrowserRouter>
            </AntApp>
        </ConfigProvider>
    );
};

export const App = () => <Bootstrap />;
