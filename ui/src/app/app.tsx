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
    ProjectOutlined,
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
import {AuthSettings} from './shared/models';
import {services, ViewPreferences} from './shared/services';
import requests from './shared/services/requests';
import {BrandMark} from './mobile/components';
import {
    BytecodeBlacklistPage,
    BytecodeDetailPage,
    BytecodesPage,
    HelpPage,
    LoginPage,
    NotificationsDetailPage,
    NotificationsPage,
    PolymarketHotPage,
    PolymarketMoversPage,
    PolymarketRealtimePage,
    PolymarketSportsLivePage,
    ProjectDetailPage,
    ProjectsPage,
    SettingsPage,
    SourceQualityPromptsPage,
    UserInfoPage,
    WalletBlacklistPage,
    WalletsPage,
    WormPage
} from './mobile/pages';

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
}

const navItems: NavItem[] = [
    {
        key: 'application',
        label: 'Application',
        icon: <DashboardOutlined />,
        children: [
            {key: '/projects', label: 'Projects', path: '/projects', icon: <ProjectOutlined />},
            {key: '/wallet-blacklist', label: 'Wallet Blacklist', path: '/wallet-blacklist', icon: <ApiOutlined />}
        ]
    },
    {
        key: 'solidity',
        label: 'Solidity',
        icon: <CodeOutlined />,
        children: [
            {key: '/solidity/bytecodes', label: 'Bytecodes', path: '/solidity/bytecodes', icon: <CodeOutlined />},
            {key: '/solidity/bytecode-blacklist', label: 'Bytecode Blacklist', path: '/solidity/bytecode-blacklist', icon: <ApiOutlined />},
            {key: '/solidity/source-quality/prompts', label: 'Quality Prompts', path: '/solidity/source-quality/prompts', icon: <FileTextOutlined />}
        ]
    },
    {key: '/wallet', label: 'Wallets', path: '/wallet', icon: <WalletOutlined />},
    {key: '/worm', label: 'Worm', path: '/worm', icon: <ApiOutlined />},
    {
        key: 'polymarket',
        label: 'Polymarket',
        icon: <DashboardOutlined />,
        children: [
            {key: '/polymarket', label: 'Hot Markets', path: '/polymarket', icon: <DashboardOutlined />},
            {key: '/polymarket/realtime', label: 'Realtime', path: '/polymarket/realtime', icon: <DashboardOutlined />},
            {key: '/polymarket/movers', label: 'Movers', path: '/polymarket/movers', icon: <DashboardOutlined />},
            {key: '/polymarket/sports-live', label: 'Sports Live', path: '/polymarket/sports-live', icon: <DashboardOutlined />}
        ]
    },
    {key: '/notifications', label: 'Notifications', path: '/notifications', icon: <BellOutlined />},
    {key: '/settings', label: 'Settings', path: '/settings', icon: <SettingOutlined />},
    {key: '/user-info', label: 'User Info', path: '/user-info', icon: <UserOutlined />},
    {key: '/help', label: 'Help', path: '/help', icon: <QuestionCircleOutlined />}
];

const flattenNav = (items: NavItem[]): NavItem[] => items.flatMap(item => [item, ...(item.children ? flattenNav(item.children) : [])]);

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

async function isExpiredSSO() {
    try {
        const {iss} = await services.users.get();
        const authSettings = await services.authService.settings();
        if (iss && iss !== 'athena') {
            return ((authSettings.dexConfig && authSettings.dexConfig.connectors) || []).length > 0 || authSettings.oidcConfig;
        }
    } catch {
        return false;
    }
    return false;
}

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

const AppRoutes = () => (
    <Routes>
        <Route path='/' element={<Navigate replace={true} to='/user-info' />} />
        <Route path='/login' element={<LoginPage />} />
        <Route path='/projects' element={<ProjectsPage />} />
        <Route path='/projects/:contract' element={<ProjectDetailPage />} />
        <Route path='/wallet' element={<WalletsPage />} />
        <Route path='/wallet-blacklist' element={<WalletBlacklistPage />} />
        <Route path='/worm' element={<WormPage />} />
        <Route path='/polymarket' element={<PolymarketHotPage />} />
        <Route path='/polymarket/realtime' element={<PolymarketRealtimePage />} />
        <Route path='/polymarket/movers' element={<PolymarketMoversPage />} />
        <Route path='/polymarket/sports-live' element={<PolymarketSportsLivePage />} />
        <Route path='/notifications' element={<NotificationsPage />} />
        <Route path='/notifications/:id' element={<NotificationsDetailPage />} />
        <Route path='/settings/*' element={<SettingsPage />} />
        <Route path='/user-info' element={<UserInfoPage />} />
        <Route path='/help' element={<HelpPage />} />
        <Route path='/solidity' element={<Navigate replace={true} to='/solidity/bytecodes' />} />
        <Route path='/solidity/bytecodes' element={<BytecodesPage />} />
        <Route path='/solidity/bytecodes/:codeHash' element={<BytecodeDetailPage />} />
        <Route path='/solidity/bytecode-blacklist' element={<BytecodeBlacklistPage />} />
        <Route path='/solidity/source-quality/prompts' element={<SourceQualityPromptsPage />} />
        <Route path='*' element={<Navigate replace={true} to='/user-info' />} />
    </Routes>
);

const Shell = (props: {pref: ViewPreferences; authSettings: AuthSettings}) => {
    const navigate = useNavigate();
    const location = useLocation();
    const ant = AntApp.useApp();
    const [mobileNavOpen, setMobileNavOpen] = React.useState(false);
    const [desktopCollapsed, setDesktopCollapsed] = React.useState(props.pref.hideSidebar);

    React.useEffect(() => {
        setDesktopCollapsed(props.pref.hideSidebar);
    }, [props.pref.hideSidebar]);

    React.useEffect(() => {
        const subscription: Subscription = requests.onError.subscribe(async err => {
            if (err.status !== 401 || location.pathname.startsWith('/login')) {
                return;
            }
            const isSSO = await isExpiredSSO();
            if (window.location.pathname.startsWith(`${base.replace(/\/$/, '')}/login`)) {
                return;
            }
            const basehref = document.querySelector('head > base')?.getAttribute('href')?.replace(/\/$/, '') || '';
            if (isSSO) {
                window.location.href = `${basehref}/auth/login?return_url=${encodeURIComponent(location.pathname + location.search)}`;
            } else {
                navigate(`/login?return_url=${encodeURIComponent(location.pathname + location.search)}`);
            }
        });
        return () => subscription?.unsubscribe();
    }, [location.pathname, location.search, navigate]);

    React.useEffect(() => {
        document.body.dataset.theme = props.pref.theme || 'light';
    }, [props.pref.theme]);

    const onMenuClick: MenuProps['onClick'] = item => {
        const target = flattenNav(navItems).find(navItem => navItem.key === item.key);
        if (target?.path) {
            navigate(target.path);
            setMobileNavOpen(false);
        }
    };

    const notifications = React.useMemo(
        () => ({
            success: (message: string, description?: string) => ant.notification.success({message, description}),
            error: (message: string, description?: string) => ant.notification.error({message, description}),
            info: (message: string, description?: string) => ant.notification.info({message, description}),
            warning: (message: string, description?: string) => ant.notification.warning({message, description})
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
        <Menu mode='inline' items={toMenuItems(navItems)} selectedKeys={[selectedKey(location.pathname)]} defaultOpenKeys={openKeys(location.pathname)} onClick={onMenuClick} />
    );

    const content = location.pathname.startsWith('/login') ? (
        <AppRoutes />
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
                        <AuthSettingsCtx.Provider value={props.authSettings}>
                            <AppRoutes />
                        </AuthSettingsCtx.Provider>
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
