import '@fortawesome/fontawesome-free/css/all.css';
import 'antd/dist/reset.css';
import '../../assets/fonts.css';
import '../styles/admin.css';

import {
    ApiOutlined,
    BellOutlined,
    CheckOutlined,
    DesktopOutlined,
    HeartOutlined,
    IdcardOutlined,
    KeyOutlined,
    LogoutOutlined,
    MenuFoldOutlined,
    MenuUnfoldOutlined,
    MoonOutlined,
    PieChartOutlined,
    QuestionCircleOutlined,
    SunOutlined,
    TeamOutlined
} from '@ant-design/icons';
import {App as AntApp, Breadcrumb, Button, Dropdown, Layout, Menu, Result, Space, Spin, Tag, Tooltip} from 'antd';
import type {MenuProps} from 'antd';
import * as React from 'react';
import {createBrowserRouter, Navigate, Route, RouterProvider, Routes, useLocation, useNavigate} from 'react-router-dom';
import {AccountAvatar, accountTierLabel} from '../shared/account-presentation';
import {moduleAccessLevels} from '../shared/account-access';
import {AccountDataAccess, AccountDataModule} from '../shared/access-modules';
import {AuthorizationCtx, Provider} from '../shared/context';
import {AppBootstrap, AppBootstrapSessionStatus, UserInfo} from '../shared/models';
import {readAdminLoginReturnTo} from '../shared/login-navigation';
import {deploymentPath, readApplicationBaseHRef, readDeploymentBaseHRef} from '../shared/runtime-base';
import requests, {isAccountMaintenanceError, requestErrorDetails, requestErrorMessage} from '../shared/services/requests';
import type {ThemeMode, ViewPreferences} from '../shared/services/view-preferences-service';
import {accountThemeLabel, localThemeMode, serverThemeMode} from '../shared/theme';
import {BrandMark, clearAsyncDataCache, setAsyncDataCacheSession} from '../components';
import {SessionBootstrap} from '../session/bootstrap';
import {
    AccountCenterPage,
    AdminAccountsPage,
    AdminLoginPage,
    EtherscanGatewaysPage,
    HelpPage,
    ProfitSharingAdminRoundPage,
    ProfitSharingAdminRoundsPage,
    ServiceStatusPage,
    SystemNotificationDetailPage,
    SystemNotificationsPage
} from './routes';
import {adminServices as services, configureAdminSessionServices, ensureAdminBusinessServices} from './services';

configureAdminSessionServices();
services.viewPreferences.init();
requests.setBaseHRef(readDeploymentBaseHRef());
requests.configureAuthorizationRealm('admin');

const adminSections: MenuProps['items'] = [
    {type: 'group', key: 'account-admin', label: 'Account Admin', children: [{key: '/accounts', label: 'Accounts', icon: <TeamOutlined />}]},
    {type: 'group', key: 'governance', label: 'Governance', children: [{key: '/profit-sharing', label: 'Profit Sharing', icon: <PieChartOutlined />}]},
    {
        type: 'group',
        key: 'system',
        label: 'System',
        children: [
            {key: '/service-status', label: 'Service Status', icon: <HeartOutlined />},
            {key: '/etherscan-gateways', label: 'Etherscan Gateways', icon: <ApiOutlined />},
            {key: '/notifications', label: 'Notifications', icon: <BellOutlined />}
        ]
    }
];

const routeMetadata = [
    {path: '/accounts', section: 'Account Admin', label: 'Accounts'},
    {path: '/profit-sharing', section: 'Governance', label: 'Profit Sharing'},
    {path: '/service-status', section: 'System', label: 'Service Status'},
    {path: '/etherscan-gateways', section: 'System', label: 'Etherscan Gateways'},
    {path: '/notifications', section: 'System', label: 'Notifications'},
    {path: '/account/profile', section: 'Account', label: 'Profile'},
    {path: '/account/appearance', section: 'Account', label: 'Appearance'},
    {path: '/account/access', section: 'Account', label: 'Access & session'},
    {path: '/help', section: 'Support', label: 'Help'}
];

const routeInfo = (pathname: string) =>
    [...routeMetadata].sort((left, right) => right.path.length - left.path.length).find(item => pathname === item.path || pathname.startsWith(`${item.path}/`));

const selectedMenuKey = (pathname: string) =>
    routeMetadata.filter(item => !item.path.startsWith('/account') && item.path !== '/help').find(item => pathname === item.path || pathname.startsWith(`${item.path}/`))?.path;

const logicalAdminLocation = (pathname: string, search = '', hash = '') => `/admin${pathname === '/' ? '' : pathname}${search}${hash}`;

const adminLoginPath = (pathname: string, search = '', hash = '', maintenance = false) => {
    const query = new URLSearchParams();
    const returnTo = logicalAdminLocation(pathname, search, hash);
    if (returnTo !== '/admin') {
        query.set('returnTo', returnTo);
    }
    if (maintenance) {
        query.set('reason', 'maintenance');
    }
    const suffix = query.toString();
    return `${deploymentPath('admin/login')}${suffix ? `?${suffix}` : ''}`;
};

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

const AdminForbiddenPage = () => (
    <Result
        status='403'
        title='Administrator access required'
        subTitle='This account can use the Athena member workspace but is not allowed to open Athena Admin.'
        extra={
            <Button type='primary' onClick={() => window.location.assign(deploymentPath(''))}>
                Return to Athena
            </Button>
        }
    />
);

const AdminNotFoundPage = () => {
    const navigate = useNavigate();
    return <Result status='404' title='Page not found' extra={<Button onClick={() => navigate('/accounts')}>Return to accounts</Button>} />;
};

const AdminRoutes = (props: {
    preferences: ViewPreferences;
    settings: AppBootstrap['settings'];
    themeChanging: boolean;
    onThemeChange: (theme: ThemeMode) => Promise<void>;
    loggingOut: boolean;
    onLogout: () => void;
}) => {
    const accountProps = {
        preferences: props.preferences,
        themeChanging: props.themeChanging,
        onThemeChange: props.onThemeChange,
        loggingOut: props.loggingOut,
        onLogout: props.onLogout
    };
    return (
        <React.Suspense fallback={<div className='athena-boot'>Loading administration…</div>}>
            <Routes>
                <Route path='/' element={<Navigate replace={true} to='/accounts' />} />
                <Route path='/accounts' element={<AdminAccountsPage />} />
                <Route path='/profit-sharing' element={<ProfitSharingAdminRoundsPage />} />
                <Route path='/profit-sharing/:slug' element={<ProfitSharingAdminRoundPage />} />
                <Route path='/service-status' element={<ServiceStatusPage />} />
                <Route path='/etherscan-gateways' element={<EtherscanGatewaysPage />} />
                <Route path='/notifications' element={<SystemNotificationsPage />} />
                <Route path='/notifications/:id' element={<SystemNotificationDetailPage />} />
                <Route path='/account/profile' element={<AccountCenterPage section='profile' {...accountProps} />} />
                <Route path='/account/appearance' element={<AccountCenterPage section='appearance' {...accountProps} />} />
                <Route path='/account/access' element={<AccountCenterPage section='access' {...accountProps} />} />
                <Route path='/help' element={<HelpPage help={props.settings.help} />} />
                <Route path='*' element={<AdminNotFoundPage />} />
            </Routes>
        </React.Suspense>
    );
};

const AdminShell = (props: {initialUser: UserInfo; preferences: ViewPreferences; settings: AppBootstrap['settings']}) => {
    const navigate = useNavigate();
    const location = useLocation();
    const ant = AntApp.useApp();
    const narrowShell = useNarrowShell();
    const [user, setUser] = React.useState(props.initialUser);
    const [desktopSidebarCollapsed, setDesktopSidebarCollapsed] = React.useState(props.preferences.hideSidebar);
    const [mobileSidebarOpen, setMobileSidebarOpen] = React.useState(false);
    const [accountMenuOpen, setAccountMenuOpen] = React.useState(false);
    const [themeChanging, setThemeChanging] = React.useState(false);
    const [loggingOut, setLoggingOut] = React.useState(false);
    const [lastCheckedAt, setLastCheckedAt] = React.useState(Date.now());
    const moduleAccess = React.useMemo(() => moduleAccessLevels(user.access), [user.access]);
    const sidebarCollapsed = narrowShell ? !mobileSidebarOpen : desktopSidebarCollapsed;

    React.useEffect(() => {
        setMobileSidebarOpen(false);
    }, [location.pathname]);

    const notifications = React.useMemo(
        () => ({
            success: (message: string, description?: string) => ant.notification.success({title: message, description}),
            error: (message: string, description?: string) => ant.notification.error({title: message, description}),
            info: (message: string, description?: string) => ant.notification.info({title: message, description}),
            warning: (message: string, description?: string) => ant.notification.warning({title: message, description})
        }),
        [ant.notification]
    );

    const refresh = React.useCallback(async () => {
        const latest = await services.users.get();
        if (!latest.loggedIn) {
            window.location.replace(adminLoginPath(location.pathname, location.search, location.hash));
            return;
        }
        if (!latest.administrator) {
            setUser(latest);
            requests.endAuthorizationSession();
            clearAsyncDataCache();
            return;
        }
        setUser(latest);
        const sessionGeneration = requests.beginAuthorizationSession(latest.accountId);
        setAsyncDataCacheSession('admin', latest.accountId, sessionGeneration);
        setLastCheckedAt(Date.now());
        services.viewPreferences.syncServerTheme(localThemeMode(latest.preferences.theme));
    }, [location.hash, location.pathname, location.search]);

    React.useEffect(() => {
        const subscription = requests.onError.subscribe(error => {
            const details = requestErrorDetails(error);
            if (isAccountMaintenanceError(error) || details.status === 401) {
                requests.endAuthorizationSession();
                clearAsyncDataCache();
                window.location.replace(adminLoginPath(location.pathname, location.search, location.hash, isAccountMaintenanceError(error)));
                return;
            }
            if (details.status === 403 && details.reason === 'ACCOUNT_ADMIN_REQUIRED') {
                void refresh();
            }
        });
        return () => subscription.unsubscribe();
    }, [location.hash, location.pathname, location.search, refresh]);

    React.useEffect(() => {
        const info = routeInfo(location.pathname);
        document.title = info ? `${info.label} · Athena Admin` : 'Athena Admin';
    }, [location.pathname]);

    const changeTheme = async (theme: ThemeMode) => {
        if (themeChanging || localThemeMode(user.preferences.theme) === theme) {
            setAccountMenuOpen(false);
            return;
        }
        setThemeChanging(true);
        services.viewPreferences.syncServerTheme(theme);
        try {
            const preferences = await services.accounts.updatePreferences(serverThemeMode(theme), user.preferences.revision);
            setUser(current => ({...current, preferences}));
            services.viewPreferences.syncServerTheme(localThemeMode(preferences.theme));
            notifications.success('Appearance updated', `${accountThemeLabel(preferences.theme)} theme is now synced across devices.`);
        } catch (error) {
            services.viewPreferences.syncServerTheme(localThemeMode(user.preferences.theme));
            notifications.error('Could not update appearance', requestErrorMessage(error));
        } finally {
            setThemeChanging(false);
            setAccountMenuOpen(false);
        }
    };

    const logout = async () => {
        if (loggingOut) {
            return;
        }
        setLoggingOut(true);
        try {
            await services.users.logout();
            requests.endAuthorizationSession();
            clearAsyncDataCache();
            window.location.replace(deploymentPath('admin/login'));
        } catch (error) {
            notifications.error('Logout failed', requestErrorMessage(error));
            setLoggingOut(false);
        }
    };

    if (!user.administrator) {
        return <AdminForbiddenPage />;
    }

    const contextValue = {
        notifications,
        modal: ant.modal,
        navigation: {goto: navigate, replace: (path: string) => navigate(path, {replace: true})},
        baseHref: readDeploymentBaseHRef()
    };
    const authorizationValue = {
        user,
        isAdmin: true,
        access: (module: AccountDataModule) => moduleAccess[module],
        canRead: (module: AccountDataModule) => moduleAccess[module] >= AccountDataAccess.Read,
        canWrite: (module: AccountDataModule) => moduleAccess[module] >= AccountDataAccess.ReadWrite,
        revision: user.access.revision,
        lastCheckedAt,
        refresh
    };
    const info = routeInfo(location.pathname);
    const accountMenuItems: MenuProps['items'] = [
        {
            key: 'summary',
            disabled: true,
            label: (
                <div className='athena-account-menu__summary'>
                    <AccountAvatar profile={user.profile} username={user.username} size={42} />
                    <span>
                        <strong>{user.profile.displayName || user.username}</strong>
                        <small>@{user.username}</small>
                    </span>
                    <Tag>{accountTierLabel(user.profile.tier)}</Tag>
                </div>
            )
        },
        {type: 'divider'},
        {key: '/account/profile', label: 'Profile', icon: <IdcardOutlined />},
        {
            type: 'group',
            label: `Appearance · ${themeChanging ? 'Saving…' : accountThemeLabel(props.preferences.theme)}`,
            children: [
                {key: 'theme:system', label: 'System', icon: props.preferences.theme === 'system' ? <CheckOutlined /> : <DesktopOutlined />},
                {key: 'theme:light', label: 'Light', icon: props.preferences.theme === 'light' ? <CheckOutlined /> : <SunOutlined />},
                {key: 'theme:dark', label: 'Dark', icon: props.preferences.theme === 'dark' ? <CheckOutlined /> : <MoonOutlined />}
            ]
        },
        {key: '/account/access', label: 'Access', icon: <KeyOutlined />},
        {key: '/help', label: 'Help', icon: <QuestionCircleOutlined />},
        {type: 'divider'},
        {
            key: 'logout',
            label: loggingOut ? (
                <Space>
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
    ];

    const onAccountMenuClick: MenuProps['onClick'] = item => {
        if (item.key.startsWith('theme:')) {
            void changeTheme(item.key.slice('theme:'.length) as ThemeMode);
        } else if (item.key === 'logout') {
            void logout();
        } else if (item.key.startsWith('/')) {
            setAccountMenuOpen(false);
            navigate(item.key);
        }
    };

    return (
        <Provider value={contextValue}>
            <AuthorizationCtx.Provider value={authorizationValue}>
                <Layout className='athena-shell athena-admin-shell'>
                    <a className='athena-skip-link' href='#athena-main' aria-hidden={narrowShell && mobileSidebarOpen} tabIndex={narrowShell && mobileSidebarOpen ? -1 : undefined}>
                        Skip to main content
                    </a>
                    <Layout.Sider
                        className='athena-shell__sider'
                        collapsible={true}
                        collapsed={sidebarCollapsed}
                        collapsedWidth={narrowShell ? 0 : 72}
                        trigger={null}
                        width={272}
                        role={narrowShell && mobileSidebarOpen ? 'dialog' : undefined}
                        aria-modal={narrowShell && mobileSidebarOpen ? true : undefined}
                        aria-label={narrowShell && mobileSidebarOpen ? 'Administration navigation' : undefined}
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
                                        <strong>Athena Admin</strong>
                                        <small>Administration Console</small>
                                    </span>
                                )}
                            </div>
                        )}
                        <nav className='athena-sidebar-navigation' id='athena-admin-navigation' aria-label='Administration navigation'>
                            {(!narrowShell || mobileSidebarOpen) && (
                                <Menu
                                    mode='inline'
                                    items={adminSections}
                                    selectedKeys={selectedMenuKey(location.pathname) ? [selectedMenuKey(location.pathname)] : []}
                                    onClick={item => {
                                        setMobileSidebarOpen(false);
                                        navigate(item.key);
                                    }}
                                />
                            )}
                        </nav>
                        {(!narrowShell || mobileSidebarOpen) && (
                            <div className='athena-sidebar-footer'>
                                <Dropdown
                                    open={accountMenuOpen}
                                    trigger={['click']}
                                    placement='topLeft'
                                    classNames={{root: 'athena-account-menu'}}
                                    menu={{items: accountMenuItems, onClick: onAccountMenuClick, selectable: false}}
                                    onOpenChange={setAccountMenuOpen}>
                                    <button className='athena-account-trigger' type='button' aria-label='Open account menu' aria-expanded={accountMenuOpen}>
                                        <AccountAvatar profile={user.profile} username={user.username} size={38} />
                                        {!sidebarCollapsed && (
                                            <span className='athena-account-trigger__copy'>
                                                <strong>{user.profile.displayName || user.username}</strong>
                                                <small>@{user.username}</small>
                                            </span>
                                        )}
                                    </button>
                                </Dropdown>
                            </div>
                        )}
                    </Layout.Sider>
                    {narrowShell && mobileSidebarOpen && (
                        <button className='athena-shell__backdrop' type='button' aria-label='Close navigation' tabIndex={-1} onClick={() => setMobileSidebarOpen(false)} />
                    )}
                    <Layout>
                        <Layout.Header className='athena-shell__header'>
                            <div className='athena-shell__header-left'>
                                <Tooltip title={sidebarCollapsed ? 'Open navigation' : 'Close navigation'}>
                                    <Button
                                        className='athena-shell__sidebar-toggle'
                                        type='text'
                                        aria-label={sidebarCollapsed ? 'Open navigation' : 'Close navigation'}
                                        aria-controls='athena-admin-navigation'
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
                                <Breadcrumb className='athena-shell__breadcrumb' items={info ? [{title: info.section}, {title: info.label}] : []} />
                            </div>
                        </Layout.Header>
                        <Layout.Content className='athena-shell__content' id='athena-main' tabIndex={-1}>
                            <AdminRoutes
                                preferences={props.preferences}
                                settings={props.settings}
                                themeChanging={themeChanging}
                                onThemeChange={changeTheme}
                                loggingOut={loggingOut}
                                onLogout={() => void logout()}
                            />
                        </Layout.Content>
                    </Layout>
                </Layout>
            </AuthorizationCtx.Provider>
        </Provider>
    );
};

const AdminEntry = () => {
    const location = useLocation();
    return (
        <SessionBootstrap>
            {(bootstrap, preferences) => {
                if (bootstrap.session.status !== AppBootstrapSessionStatus.Authenticated || !bootstrap.session.userInfo) {
                    if (location.pathname !== '/login') {
                        window.location.replace(
                            adminLoginPath(location.pathname, location.search, location.hash, bootstrap.session.status === AppBootstrapSessionStatus.AccountMaintenance)
                        );
                        return null;
                    }
                    return (
                        <React.Suspense fallback={<div className='athena-boot'>Loading sign in…</div>}>
                            <AdminLoginPage />
                        </React.Suspense>
                    );
                }
                if (!bootstrap.session.userInfo.administrator) {
                    return <AdminForbiddenPage />;
                }
                ensureAdminBusinessServices();
                const sessionGeneration = requests.beginAuthorizationSession(bootstrap.session.userInfo.accountId);
                setAsyncDataCacheSession('admin', bootstrap.session.userInfo.accountId, sessionGeneration);
                if (location.pathname === '/login') {
                    const returnTo = readAdminLoginReturnTo(location.search);
                    window.location.replace(deploymentPath(returnTo));
                    return null;
                }
                return <AdminShell initialUser={bootstrap.session.userInfo} preferences={preferences} settings={bootstrap.settings} />;
            }}
        </SessionBootstrap>
    );
};

export const AdminApp = () => {
    const [router] = React.useState(() =>
        createBrowserRouter([{path: '*', element: <AdminEntry />}], {
            basename: readApplicationBaseHRef(),
            future: {v7_relativeSplatPath: true}
        })
    );
    return <RouterProvider router={router} future={{v7_startTransition: true}} />;
};
