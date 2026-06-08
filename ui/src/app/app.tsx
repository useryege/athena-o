import '@fortawesome/fontawesome-free/css/all.css';
import 'antd/dist/reset.css';
import './mobile/styles.css';

import {
    ApiOutlined,
    BellOutlined,
    CodeOutlined,
    DashboardOutlined,
    FileTextOutlined,
    MenuFoldOutlined,
    MenuUnfoldOutlined,
    MoonOutlined,
    QuestionCircleOutlined,
    SettingOutlined,
    SunOutlined,
    UserOutlined,
    WalletOutlined
} from '@ant-design/icons';
import {App as AntApp, Button, ConfigProvider, Drawer, Dropdown, Layout as AntLayout, Menu, Result, Space, theme as antTheme, Typography} from 'antd';
import type {MenuProps} from 'antd';
import * as React from 'react';
import {BrowserRouter, Navigate, Route, Routes, useLocation, useNavigate} from 'react-router-dom';
import {Subscription} from 'rxjs';
import {AuthSettingsCtx, Provider} from './shared/context';
import {AuthSettings, Permission, UserInfo} from './shared/models';
import {services, ViewPreferences} from './shared/services';
import requests from './shared/services/requests';
import {BrandMark} from './mobile/components';
import {
    BytecodeBlacklistsPage,
    ChainCheckpointsPage,
    CollectionTasksPage,
    ContractCodeDetailPage,
    ContractCodesPage,
    HelpPage,
    LoginPage,
    NotificationsDetailPage,
    NotificationsPage,
    PolymarketHotPage,
    PolymarketMoversPage,
    PolymarketRealtimePage,
    PolymarketSportsLivePage,
    ProjectsPage,
    SettingsPage,
    UserInfoPage,
    WalletBlacklistsPage,
    WalletsPage,
    WormPage
} from './mobile/page';

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

interface AccessState {
    user: UserInfo;
    permissions: Record<string, boolean>;
}

const rbacResources = {
    notifications: 'notifications',
    polymarket: 'polymarket',
    tokenapi: 'tokenapi',
    wallets: 'wallets',
    worm: 'worm'
};

const rbacActions = {
    get: 'get'
};

const tokenapiSubresources = {
    projects: 'projects',
    contractCodes: 'contract-codes',
    bytecodeBlacklists: 'bytecode-blacklists',
    walletBlacklists: 'wallet-blacklists',
    chainCheckpoints: 'chain-checkpoints',
    collectionTasks: 'collection-tasks'
};

const permission = (resource: string, action: string, subresource = '*'): Permission => ({resource, action, subresource});
const tokenapiPermission = (subresource: string) => permission(rbacResources.tokenapi, rbacActions.get, subresource);
const permissionKey = (perm: Permission) => `${perm.resource}:${perm.action}:${perm.subresource}`;
const hasPermission = (access: AccessState, perm?: Permission) => !perm || access?.permissions[permissionKey(perm)] === true;

const navItems: NavItem[] = [
    {
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
    },
    {
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
            }
        ]
    },
    {key: '/worm', label: 'Worm', path: '/worm', icon: <ApiOutlined />, permission: permission(rbacResources.worm, rbacActions.get)},
    {key: '/notifications', label: 'Notifications', path: '/notifications', icon: <BellOutlined />, permission: permission(rbacResources.notifications, rbacActions.get)},
    {key: '/wallet', label: 'Wallets', path: '/wallet', icon: <WalletOutlined />, permission: permission(rbacResources.wallets, rbacActions.get)},
    {key: '/settings', label: 'Settings', path: '/settings', icon: <SettingOutlined />},
    {key: '/user-info', label: 'User Info', path: '/user-info', icon: <UserOutlined />},
    {key: '/help', label: 'Help', path: '/help', icon: <QuestionCircleOutlined />}
];

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

const selectedKey = (pathname: string) => {
    const exact = flattenNav(navItems)
        .filter(item => item.path)
        .sort((a, b) => (b.path || '').length - (a.path || '').length)
        .find(item => pathname === item.path || pathname.startsWith(`${item.path}/`));
    return exact?.key || '/user-info';
};

const openKeys = (pathname: string) =>
    navItems.filter(item => (item.children || []).some(child => pathname === child.path || pathname.startsWith(`${child.path}/`))).map(item => item.key);

const pageTitle = (pathname: string) => flattenNav(navItems).find(item => item.key === selectedKey(pathname))?.label || 'Athena';

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
            <Route path='/wallet' element={withPermission(permission(rbacResources.wallets, rbacActions.get), <WalletsPage />)} />
            <Route path='/worm' element={withPermission(permission(rbacResources.worm, rbacActions.get), <WormPage />)} />
            <Route path='/polymarket' element={withPermission(permission(rbacResources.polymarket, rbacActions.get), <PolymarketHotPage />)} />
            <Route path='/polymarket/realtime' element={withPermission(permission(rbacResources.polymarket, rbacActions.get), <PolymarketRealtimePage />)} />
            <Route path='/polymarket/movers' element={withPermission(permission(rbacResources.polymarket, rbacActions.get), <PolymarketMoversPage />)} />
            <Route path='/polymarket/sports-live' element={withPermission(permission(rbacResources.polymarket, rbacActions.get), <PolymarketSportsLivePage />)} />
            <Route path='/notifications' element={withPermission(permission(rbacResources.notifications, rbacActions.get), <NotificationsPage />)} />
            <Route path='/notifications/:id' element={withPermission(permission(rbacResources.notifications, rbacActions.get), <NotificationsDetailPage />)} />
            <Route path='/settings/*' element={<SettingsPage />} />
            <Route path='/user-info' element={<UserInfoPage />} />
            <Route path='/help' element={<HelpPage />} />
            <Route path='/token' element={visibleTokenDefault ? <Navigate replace={true} to={visibleTokenDefault} /> : <ForbiddenPage />} />
            <Route path='/token/projects' element={withPermission(tokenapiPermission(tokenapiSubresources.projects), <ProjectsPage />)} />
            <Route path='/token/contract-codes' element={withPermission(tokenapiPermission(tokenapiSubresources.contractCodes), <ContractCodesPage />)} />
            <Route path='/token/contract-codes/:codeHash' element={withPermission(tokenapiPermission(tokenapiSubresources.contractCodes), <ContractCodeDetailPage />)} />
            <Route path='/token/bytecode-blacklists' element={withPermission(tokenapiPermission(tokenapiSubresources.bytecodeBlacklists), <BytecodeBlacklistsPage />)} />
            <Route path='/token/wallet-blacklists' element={withPermission(tokenapiPermission(tokenapiSubresources.walletBlacklists), <WalletBlacklistsPage />)} />
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
    const [mobileNavOpen, setMobileNavOpen] = React.useState(false);
    const [desktopCollapsed, setDesktopCollapsed] = React.useState(props.pref.hideSidebar);
    const isLoginPath = location.pathname.startsWith('/login');
    const locationKey = `${location.pathname}${location.search}`;
    const [authorizedLocationKey, setAuthorizedLocationKey] = React.useState(isLoginPath ? locationKey : '');
    const [access, setAccess] = React.useState<AccessState>(null);

    React.useEffect(() => {
        setDesktopCollapsed(props.pref.hideSidebar);
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
        document.body.dataset.theme = props.pref.theme || 'light';
    }, [props.pref.theme]);

    const visibleNavItems = access ? filterNavItems(navItems, access) : [];

    const onMenuClick: MenuProps['onClick'] = item => {
        const target = flattenNav(visibleNavItems).find(navItem => navItem.key === item.key);
        if (target?.path) {
            navigate(target.path);
            setMobileNavOpen(false);
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

    const menu = (
        <Menu
            mode='inline'
            items={toMenuItems(visibleNavItems)}
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
            <AntLayout.Sider className='athena-shell__sider' collapsible={true} collapsed={desktopCollapsed} trigger={null} width={248}>
                <div className='athena-brand' onClick={() => navigate('/user-info')}>
                    <BrandMark size='small' />
                    {!desktopCollapsed && <span>Athena</span>}
                </div>
                {menu}
            </AntLayout.Sider>
            <AntLayout>
                <AntLayout.Header className='athena-shell__header'>
                    <div className='athena-shell__header-left'>
                        <Button
                            className='athena-shell__desktop-toggle'
                            type='text'
                            icon={desktopCollapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
                            onClick={() => {
                                const next = !desktopCollapsed;
                                setDesktopCollapsed(next);
                                services.viewPreferences.updatePreferences({...props.pref, hideSidebar: next});
                            }}
                        />
                        <Button className='athena-shell__mobile-menu' type='text' icon={<MenuUnfoldOutlined />} onClick={() => setMobileNavOpen(true)} />
                        <Typography.Title level={4}>{pageTitle(location.pathname)}</Typography.Title>
                    </div>
                    <div className='athena-shell__header-actions'>
                        <Dropdown
                            menu={{
                                items: themeMenu,
                                selectedKeys: [props.pref.theme || 'light'],
                                onClick: item => services.viewPreferences.updatePreferences({...props.pref, theme: item.key})
                            }}>
                            <Button type='text' icon={(props.pref.theme || 'light') === 'dark' ? <MoonOutlined /> : <SunOutlined />} />
                        </Dropdown>
                        <Button type='text' icon={<UserOutlined />} onClick={() => navigate('/user-info')} />
                    </div>
                </AntLayout.Header>
                <AntLayout.Content className='athena-shell__content'>
                    <Provider value={contextValue}>
                        <AuthSettingsCtx.Provider value={props.authSettings}>{routes}</AuthSettingsCtx.Provider>
                    </Provider>
                </AntLayout.Content>
            </AntLayout>
            <Drawer title='Athena' placement='left' open={mobileNavOpen} onClose={() => setMobileNavOpen(false)} size={312}>
                {menu}
            </Drawer>
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
                    colorPrimary: '#e05f3f',
                    colorInfo: '#2f7df6',
                    colorSuccess: '#19a974',
                    colorWarning: '#d98b18',
                    colorError: '#d14b57',
                    colorLink: '#2f7df6',
                    colorBgLayout: isDark ? '#101214' : '#f3f6f5',
                    colorBgContainer: isDark ? '#171b1d' : '#ffffff',
                    colorText: isDark ? '#edf2ef' : '#17211d',
                    colorTextSecondary: isDark ? '#9aa7a1' : '#64726c',
                    colorBorder: isDark ? 'rgba(255,255,255,0.12)' : 'rgba(28,45,37,0.12)',
                    boxShadow: isDark ? '0 18px 48px rgba(0,0,0,0.34)' : '0 18px 48px rgba(26,45,38,0.10)',
                    fontFamily: 'Inter, Heebo, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
                    fontSize: 14,
                    controlHeight: 36,
                    controlHeightSM: 30,
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
