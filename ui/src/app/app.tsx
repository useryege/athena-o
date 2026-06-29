import '@fortawesome/fontawesome-free/css/all.css';
import 'antd/dist/reset.css';
import './styles.css';

import {
    ApiOutlined,
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
import requests from './shared/services/requests';
import {BrandMark} from './components';
import {
    BytecodeBlacklistsPage,
    ChainCheckpointsPage,
    CollectionTasksPage,
    ContractCodeDetailPage,
    ContractCodesPage,
    HelpPage,
    LoginPage,
    NodeStatusesPage,
    NotificationsDetailPage,
    NotificationsPage,
    PolymarketHotPage,
    PolymarketMoversPage,
    PolymarketRealtimePage,
    PolymarketSportsLivePage,
    PolymarketSportsHistoryPage,
    PolymarketUMADisputedPage,
    PolymarketUMAProposedPage,
    ProjectReportsPage,
    ProjectsPage,
    SettingsPage,
    ServiceStatusPage,
    UserInfoPage,
    WalletBlacklistsPage,
    WalletsPage,
    WormPolyPage
} from './pages';

services.viewPreferences.init();

const bases = document.getElementsByTagName('base');
const base = bases.length > 0 ? bases[0].getAttribute('href') || '/' : '/';
requests.setBaseHRef(base);

const authSettingsRetryDelays = [500, 1000, 2000, 3000];

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
    polymarket: 'polymarket',
    wormPoly: 'worm-poly',
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
    projectReports: 'project-reports',
    contractCodes: 'contract-codes',
    bytecodeBlacklists: 'bytecode-blacklists',
    walletBlacklists: 'wallet-blacklists',
    nodeStatuses: 'node-statuses',
    chainCheckpoints: 'chain-checkpoints',
    collectionTasks: 'collection-tasks'
};

const permission = (resource: string, action: string, subresource = '*'): Permission => ({resource, action, subresource});
const tokenapiPermission = (subresource: string) => permission(rbacResources.tokenapi, rbacActions.get, subresource);
const serviceStatusPermission = permission(rbacResources.serviceStatus, rbacActions.get);
const permissionKey = (perm: Permission) => `${perm.resource}:${perm.action}:${perm.subresource}`;
const hasPermission = (access: AccessState, perm?: Permission) => !perm || access?.permissions[permissionKey(perm)] === true;

const polymarketNavItem: NavItem = {
    key: 'polymarket',
    label: 'Polymarket',
    icon: <DashboardOutlined />,
    children: [
        {key: '/polymarket', label: 'Hot Markets', path: '/polymarket', icon: <DashboardOutlined />, permission: permission(rbacResources.polymarket, rbacActions.get)},
        {
            key: '/polymarket/realtime',
            label: 'Realtime',
            path: '/polymarket/realtime',
            icon: <DashboardOutlined />,
            permission: permission(rbacResources.polymarket, rbacActions.get)
        },
        {
            key: '/polymarket/movers',
            label: 'Movers',
            path: '/polymarket/movers',
            icon: <DashboardOutlined />,
            permission: permission(rbacResources.polymarket, rbacActions.get)
        },
        {
            key: '/polymarket/sports-live',
            label: 'Sports Live',
            path: '/polymarket/sports-live',
            icon: <DashboardOutlined />,
            permission: permission(rbacResources.polymarket, rbacActions.get)
        },
        {
            key: '/polymarket/sports-history',
            label: 'Sports History',
            path: '/polymarket/sports-history',
            icon: <DashboardOutlined />,
            permission: permission(rbacResources.polymarket, rbacActions.get)
        },
        {
            key: '/polymarket/uma-proposed',
            label: 'UMA Proposed',
            path: '/polymarket/uma-proposed',
            icon: <ApiOutlined />,
            permission: permission(rbacResources.polymarket, rbacActions.get)
        },
        {
            key: '/polymarket/uma-disputed',
            label: 'UMA Disputed',
            path: '/polymarket/uma-disputed',
            icon: <ApiOutlined />,
            permission: permission(rbacResources.polymarket, rbacActions.get)
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
            key: '/token/project-reports',
            label: 'Project Reports',
            path: '/token/project-reports',
            icon: <FileTextOutlined />,
            permission: tokenapiPermission(tokenapiSubresources.projectReports)
        },
        {
            key: '/token/contract-codes',
            label: 'Contract Codes',
            path: '/token/contract-codes',
            icon: <CodeOutlined />,
            permission: tokenapiPermission(tokenapiSubresources.contractCodes)
        },
        {
            key: '/token/bytecode-blacklists',
            label: 'Bytecode Blacklists',
            path: '/token/bytecode-blacklists',
            icon: <ApiOutlined />,
            permission: tokenapiPermission(tokenapiSubresources.bytecodeBlacklists)
        },
        {
            key: '/token/wallet-blacklists',
            label: 'Wallet Blacklists',
            path: '/token/wallet-blacklists',
            icon: <WalletOutlined />,
            permission: tokenapiPermission(tokenapiSubresources.walletBlacklists)
        },
        {
            key: '/token/node-statuses',
            label: 'Node Status',
            path: '/token/node-statuses',
            icon: <ApiOutlined />,
            permission: tokenapiPermission(tokenapiSubresources.nodeStatuses)
        },
        {
            key: '/token/chain-checkpoints',
            label: 'Chain Checkpoints',
            path: '/token/chain-checkpoints',
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
            polymarketNavItem,
            {key: '/worm-poly', label: 'Worm Poly', path: '/worm-poly', icon: <TrophyOutlined />, permission: permission(rbacResources.wormPoly, rbacActions.get)}
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
            {key: '/service-status', label: 'Service Status', path: '/service-status', icon: <HeartOutlined />, permission: serviceStatusPermission}
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

const AppRoutes = (props: {access: AccessState}) => {
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
            <Route path='/login' element={<LoginPage />} />
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
            <Route path='/polymarket' element={withPermission(permission(rbacResources.polymarket, rbacActions.get), <PolymarketHotPage />)} />
            <Route path='/polymarket/realtime' element={withPermission(permission(rbacResources.polymarket, rbacActions.get), <PolymarketRealtimePage />)} />
            <Route path='/polymarket/movers' element={withPermission(permission(rbacResources.polymarket, rbacActions.get), <PolymarketMoversPage />)} />
            <Route path='/polymarket/sports-live' element={withPermission(permission(rbacResources.polymarket, rbacActions.get), <PolymarketSportsLivePage />)} />
            <Route
                path='/polymarket/sports-history'
                element={withPermission(
                    permission(rbacResources.polymarket, rbacActions.get),
                    <PolymarketSportsHistoryPage canRefresh={hasPermission(props.access, permission(rbacResources.polymarket, rbacActions.invoke))} />
                )}
            />
            <Route
                path='/polymarket/uma-proposed'
                element={withPermission(
                    permission(rbacResources.polymarket, rbacActions.get),
                    <PolymarketUMAProposedPage canScan={hasPermission(props.access, permission(rbacResources.polymarket, rbacActions.invoke))} />
                )}
            />
            <Route
                path='/polymarket/uma-disputed'
                element={withPermission(
                    permission(rbacResources.polymarket, rbacActions.get),
                    <PolymarketUMADisputedPage canScan={hasPermission(props.access, permission(rbacResources.polymarket, rbacActions.invoke))} />
                )}
            />
            <Route
                path='/worm-poly'
                element={withPermission(
                    permission(rbacResources.wormPoly, rbacActions.get),
                    <WormPolyPage canEdit={hasPermission(props.access, permission(rbacResources.wormPoly, rbacActions.update))} />
                )}
            />
            <Route path='/notifications' element={withPermission(permission(rbacResources.notifications, rbacActions.get), <NotificationsPage />)} />
            <Route path='/notifications/:id' element={withPermission(permission(rbacResources.notifications, rbacActions.get), <NotificationsDetailPage />)} />
            <Route path='/settings/*' element={<SettingsPage />} />
            <Route path='/service-status' element={withPermission(serviceStatusPermission, <ServiceStatusPage />)} />
            <Route path='/user-info' element={<UserInfoPage />} />
            <Route path='/help' element={<HelpPage />} />
            <Route path='/token' element={visibleTokenDefault ? <Navigate replace={true} to={visibleTokenDefault} /> : <ForbiddenPage />} />
            <Route path='/token/projects' element={withPermission(tokenapiPermission(tokenapiSubresources.projects), <ProjectsPage />)} />
            <Route path='/token/project-reports' element={withPermission(tokenapiPermission(tokenapiSubresources.projectReports), <ProjectReportsPage />)} />
            <Route path='/token/contract-codes' element={withPermission(tokenapiPermission(tokenapiSubresources.contractCodes), <ContractCodesPage />)} />
            <Route path='/token/contract-codes/:codeHash' element={withPermission(tokenapiPermission(tokenapiSubresources.contractCodes), <ContractCodeDetailPage />)} />
            <Route path='/token/bytecode-blacklists' element={withPermission(tokenapiPermission(tokenapiSubresources.bytecodeBlacklists), <BytecodeBlacklistsPage />)} />
            <Route path='/token/wallet-blacklists' element={withPermission(tokenapiPermission(tokenapiSubresources.walletBlacklists), <WalletBlacklistsPage />)} />
            <Route path='/token/node-statuses' element={withPermission(tokenapiPermission(tokenapiSubresources.nodeStatuses), <NodeStatusesPage />)} />
            <Route path='/token/chain-checkpoints' element={withPermission(tokenapiPermission(tokenapiSubresources.chainCheckpoints), <ChainCheckpointsPage />)} />
            <Route path='/token/collection-tasks' element={withPermission(tokenapiPermission(tokenapiSubresources.collectionTasks), <CollectionTasksPage />)} />
            <Route path='*' element={<Navigate replace={true} to='/user-info' />} />
        </Routes>
    );
};

const Shell = (props: {pref: ViewPreferences; authSettings: AuthSettings}) => {
    const navigate = useNavigate();
    const location = useLocation();
    const ant = AntApp.useApp();
    const [sidebarCollapsed, setSidebarCollapsed] = React.useState(props.pref.hideSidebar);
    const isLoginPath = location.pathname.startsWith('/login');
    const locationKey = `${location.pathname}${location.search}`;
    const [authorizedLocationKey, setAuthorizedLocationKey] = React.useState(isLoginPath ? locationKey : '');
    const [access, setAccess] = React.useState<AccessState>(null);

    React.useEffect(() => {
        setSidebarCollapsed(props.pref.hideSidebar);
    }, [props.pref.hideSidebar]);

    React.useEffect(() => {
        if (isLoginPath) {
            setAccess(null);
            setAuthorizedLocationKey(locationKey);
            return;
        }

        let active = true;
        setAuthorizedLocationKey('');
        setAccess(null);
        services.users
            .get()
            .then(user => {
                if (!active) {
                    return;
                }
                if (!user.loggedIn) {
                    navigate('/login', {replace: true});
                    return;
                }
                const nextAccess = loadAccessState(user);
                if (!active) {
                    return;
                }
                setAccess(nextAccess);
                setAuthorizedLocationKey(locationKey);
            })
            .catch(err => {
                if (!active) {
                    return;
                }
                if (err?.status === 401) {
                    navigate('/login', {replace: true});
                    return;
                }
                setAccess({user: {loggedIn: true, username: '', iss: '', groups: [], permissions: []}, permissions: {}});
                setAuthorizedLocationKey(locationKey);
            });
        return () => {
            active = false;
        };
    }, [isLoginPath, locationKey, navigate]);

    React.useEffect(() => {
        const subscription: Subscription = requests.onError.subscribe(err => {
            if (err.status !== 401 || isLoginPath) {
                return;
            }
            if (window.location.pathname.startsWith(`${base.replace(/\/$/, '')}/login`)) {
                return;
            }
            navigate('/login', {replace: true});
        });
        return () => subscription?.unsubscribe();
    }, [isLoginPath, navigate]);

    React.useEffect(() => {
        document.body.dataset.theme = props.pref.theme || 'dark';
    }, [props.pref.theme]);

    React.useEffect(() => {
        const current = flattenNav(navItems).find(item => item.key === selectedKey(location.pathname));
        document.title = current ? `${current.label} · Athena` : 'Athena';
    }, [location.pathname]);

    const visibleNavSections = access ? filterNavSections(navSections, access) : [];
    const visibleNavItems = visibleNavSections.flatMap(section => section.children);

    const onMenuClick: MenuProps['onClick'] = item => {
        const target = flattenNav(visibleNavItems).find(navItem => navItem.key === item.key);
        if (target?.path) {
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

    const themeMenu: MenuProps['items'] = [
        {key: 'light', label: 'Light', icon: <SunOutlined />},
        {key: 'dark', label: 'Dark', icon: <MoonOutlined />}
    ];

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

    const routes = !isLoginPath && (authorizedLocationKey !== locationKey || !access) ? <div className='athena-boot'>Loading Athena...</div> : <AppRoutes access={access} />;
    const content = isLoginPath ? (
        routes
    ) : (
        <AntLayout className='athena-shell'>
            <a className='athena-skip-link' href='#athena-main'>
                Skip to main content
            </a>
            <AntLayout.Sider className='athena-shell__sider' collapsible={true} collapsed={sidebarCollapsed} collapsedWidth={72} trigger={null} width={248}>
                <button className='athena-brand' type='button' aria-label='Open Athena user information' onClick={() => navigate('/user-info')}>
                    <BrandMark size='small' />
                    {!sidebarCollapsed && (
                        <span className='athena-brand__copy'>
                            <strong>Athena</strong>
                            <small>Operations Console</small>
                        </span>
                    )}
                </button>
                <nav aria-label='Primary navigation'>{menu}</nav>
            </AntLayout.Sider>
            <AntLayout>
                <AntLayout.Header className='athena-shell__header'>
                    <div className='athena-shell__header-left'>
                        <Tooltip title={sidebarCollapsed ? 'Expand navigation' : 'Collapse navigation'}>
                            <Button
                                className='athena-shell__sidebar-toggle'
                                type='text'
                                aria-label={sidebarCollapsed ? 'Expand navigation' : 'Collapse navigation'}
                                icon={sidebarCollapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
                                onClick={() => {
                                    const next = !sidebarCollapsed;
                                    setSidebarCollapsed(next);
                                    services.viewPreferences.updatePreferences({...props.pref, hideSidebar: next});
                                }}
                            />
                        </Tooltip>
                        <Breadcrumb className='athena-shell__breadcrumb' items={breadcrumbItems(location.pathname)} />
                    </div>
                    <div className='athena-shell__header-actions'>
                        <Tooltip title='Change theme'>
                            <Dropdown
                                menu={{
                                    items: themeMenu,
                                    selectedKeys: [props.pref.theme || 'dark'],
                                    onClick: item => services.viewPreferences.updatePreferences({...props.pref, theme: item.key as ViewPreferences['theme']})
                                }}>
                                <Button type='text' aria-label='Change color theme' icon={(props.pref.theme || 'dark') === 'dark' ? <MoonOutlined /> : <SunOutlined />} />
                            </Dropdown>
                        </Tooltip>
                        <Tooltip title='User menu'>
                            <Dropdown menu={{items: userMenu, onClick: item => navigate(item.key)}}>
                                <Button type='text' aria-label='Open user menu' icon={<UserOutlined />} />
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
                    colorPrimary: isDark ? '#e76f51' : '#d85d42',
                    colorInfo: isDark ? '#3b82f6' : '#2563eb',
                    colorSuccess: isDark ? '#22c55e' : '#168a45',
                    colorWarning: isDark ? '#f59e0b' : '#b86a00',
                    colorError: isDark ? '#ef4444' : '#c9363e',
                    colorLink: isDark ? '#60a5fa' : '#2563eb',
                    colorBgBase: isDark ? '#0b0f14' : '#f4f6f8',
                    colorBgLayout: isDark ? '#0b0f14' : '#f4f6f8',
                    colorBgContainer: isDark ? '#111820' : '#ffffff',
                    colorBgElevated: isDark ? '#16202a' : '#f8fafc',
                    colorText: isDark ? '#eef3f8' : '#18212b',
                    colorTextSecondary: isDark ? '#94a3b8' : '#637083',
                    colorBorder: isDark ? '#263341' : '#d8e0e8',
                    colorBorderSecondary: isDark ? '#1f2a35' : '#e4e9ef',
                    colorSplit: isDark ? '#263341' : '#d8e0e8',
                    colorFillSecondary: isDark ? '#1b2632' : '#edf1f5',
                    controlOutline: isDark ? 'rgba(96,165,250,0.45)' : 'rgba(37,99,235,0.35)',
                    boxShadow: isDark ? '0 18px 48px rgba(0,0,0,0.34)' : '0 18px 48px rgba(24,33,43,0.10)',
                    boxShadowSecondary: isDark ? '0 12px 32px rgba(0,0,0,0.28)' : '0 12px 32px rgba(24,33,43,0.08)',
                    fontFamily: 'Inter, Heebo, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
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
