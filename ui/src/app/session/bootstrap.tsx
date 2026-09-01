import {App as AntApp, Button, ConfigProvider, Result, Space, theme as antTheme, Typography} from 'antd';
import * as React from 'react';
import {AppBootstrap, AppBootstrapSessionStatus} from '../shared/models';
import {sessionServices as services} from './services';
import type {ViewPreferences} from '../shared/services/view-preferences-service';
import {localThemeMode} from '../shared/theme';

const bootstrapRetryDelays = [500, 1000, 2000, 3000];
const wait = (delayMs: number) => new Promise(resolve => window.setTimeout(resolve, delayMs));

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

const usePreferences = () => {
    const [preferences, setPreferences] = React.useState<ViewPreferences>(null);
    React.useEffect(() => {
        const subscription = services.viewPreferences.getPreferences().subscribe(setPreferences);
        return () => subscription.unsubscribe();
    }, []);
    return preferences;
};

const themeTokens = (isDark: boolean) => ({
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
});

export const SessionBootstrap = (props: {
    children: (bootstrap: AppBootstrap, preferences: ViewPreferences) => React.ReactNode;
    acceptBootstrap?: (bootstrap: AppBootstrap) => boolean;
}) => {
    const preferences = usePreferences();
    const [bootstrap, setBootstrap] = React.useState<AppBootstrap>(null);
    const [bootstrapError, setBootstrapError] = React.useState<Error>(null);
    const [bootstrapRetry, setBootstrapRetry] = React.useState(0);

    React.useEffect(() => {
        let active = true;
        setBootstrapError(null);
        setBootstrap(null);
        loadAppBootstrapWithRetry(() => services.authService.bootstrap())
            .then(result => {
                if (!active) {
                    return;
                }
                if (result.session.status === AppBootstrapSessionStatus.Authenticated && result.session.userInfo) {
                    services.viewPreferences.syncServerTheme(localThemeMode(result.session.userInfo.preferences.theme));
                }
                if (props.acceptBootstrap && !props.acceptBootstrap(result)) {
                    active = false;
                    return;
                }
                setBootstrap(result);
                if (result.settings.uiCssURL && !document.querySelector(`link[data-athena-ui-css="${CSS.escape(result.settings.uiCssURL)}"]`)) {
                    const link = document.createElement('link');
                    link.href = result.settings.uiCssURL;
                    link.rel = 'stylesheet';
                    link.type = 'text/css';
                    link.dataset.athenaUiCss = result.settings.uiCssURL;
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
    }, [bootstrapRetry, props.acceptBootstrap]);

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

    if (!preferences || !bootstrap) {
        return <div className='athena-boot'>Loading Athena...</div>;
    }

    const isDark = services.viewPreferences.resolvedTheme(preferences.theme) === 'dark';
    return (
        <ConfigProvider theme={{algorithm: isDark ? antTheme.darkAlgorithm : antTheme.defaultAlgorithm, token: themeTokens(isDark)}}>
            <AntApp>{props.children(bootstrap, preferences)}</AntApp>
        </ConfigProvider>
    );
};
