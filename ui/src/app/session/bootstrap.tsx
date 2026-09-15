import {Button, Result, Space, Typography} from 'antd';
import * as React from 'react';
import {AppBootstrap} from '../shared/models';
import {sessionServices as services} from './services';
import type {ViewPreferences} from '../shared/services/view-preferences-service';

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
                    title={<Typography.Title level={1}>API 服务暂不可用</Typography.Title>}
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

    return <>{props.children(bootstrap, preferences)}</>;
};
