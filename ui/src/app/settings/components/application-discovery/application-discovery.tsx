import * as React from 'react';

import {Page} from '../../../shared/components';
import {services} from '../../../shared/services';
import {ProjectDiscoveryStatus} from '../../../shared/services/athena-application-service';

require('./application-discovery.scss');

type Abortable<T> = Promise<T> & {abort?: () => void};

const errorMessage = (err: any) => {
    if (!err) {
        return 'Unknown error';
    }
    if (err.response && err.response.body && err.response.body.message) {
        return err.response.body.message;
    }
    if (err.message) {
        return err.message;
    }
    return String(err);
};

export const ApplicationDiscovery = () => {
    const [status, setStatus] = React.useState<ProjectDiscoveryStatus | null>(null);
    const [loading, setLoading] = React.useState(true);
    const [permissionLoading, setPermissionLoading] = React.useState(true);
    const [canControlDiscovery, setCanControlDiscovery] = React.useState(false);
    const [submitting, setSubmitting] = React.useState(false);
    const [error, setError] = React.useState<string | null>(null);
    const mountedRef = React.useRef(false);
    const requestRef = React.useRef<Abortable<ProjectDiscoveryStatus> | null>(null);

    const loadStatus = React.useCallback(() => {
        setLoading(true);
        setError(null);
        const req = services.athenaApplication.getProjectDiscoveryStatus();
        requestRef.current = req;
        req.then(
            nextStatus => {
                if (!mountedRef.current) {
                    return;
                }
                setStatus(nextStatus);
                setLoading(false);
                requestRef.current = null;
            },
            err => {
                if (!mountedRef.current) {
                    return;
                }
                setError(errorMessage(err));
                setLoading(false);
                requestRef.current = null;
            }
        );
    }, []);

    React.useEffect(() => {
        mountedRef.current = true;
        loadStatus();
        setPermissionLoading(true);
        services.accounts.canI('application-discovery', 'update', '*').then(
            canControl => {
                if (!mountedRef.current) {
                    return;
                }
                setCanControlDiscovery(canControl);
                setPermissionLoading(false);
            },
            () => {
                if (!mountedRef.current) {
                    return;
                }
                setCanControlDiscovery(false);
                setPermissionLoading(false);
            }
        );
        return () => {
            mountedRef.current = false;
            if (requestRef.current && requestRef.current.abort) {
                requestRef.current.abort();
            }
        };
    }, [loadStatus]);

    const runAction = (action: () => Abortable<ProjectDiscoveryStatus>) => {
        setSubmitting(true);
        setError(null);
        const req = action();
        requestRef.current = req;
        req.then(
            () => {
                if (!mountedRef.current) {
                    return;
                }
                setSubmitting(false);
                requestRef.current = null;
                loadStatus();
            },
            err => {
                if (!mountedRef.current) {
                    return;
                }
                setError(errorMessage(err));
                setSubmitting(false);
                requestRef.current = null;
            }
        );
    };

    const running = !!status?.started;
    const statusText = loading && !status ? 'Loading' : status?.status || (running ? 'running' : 'stopped');
    const actionDisabled = loading || permissionLoading || submitting || !canControlDiscovery;

    return (
        <Page
            title='Application Discovery'
            toolbar={{
                breadcrumbs: [{title: 'Settings', path: '/settings'}, {title: 'Application Discovery'}]
            }}>
            <div className='application-discovery'>
                <div className='argo-container'>
                    {error && (
                        <div className='application-discovery__error'>
                            <i className='fa fa-exclamation-triangle' /> {error}
                        </div>
                    )}
                    <div className='white-box application-discovery__panel'>
                        <div className='application-discovery__header'>
                            <div>
                                <div className='application-discovery__label'>Discovery indexer</div>
                                <div className={`application-discovery__status application-discovery__status--${running ? 'running' : 'stopped'}`}>
                                    <span className='application-discovery__status-dot' />
                                    <span>{statusText}</span>
                                </div>
                            </div>
                            <div className='application-discovery__actions'>
                                <button
                                    type='button'
                                    className='argo-button argo-button--base-o'
                                    disabled={actionDisabled || running}
                                    onClick={() => !actionDisabled && !running && runAction(() => services.athenaApplication.startProjectDiscovery())}>
                                    Start
                                </button>
                                <button
                                    type='button'
                                    className='argo-button argo-button--base-o'
                                    disabled={actionDisabled || !running}
                                    onClick={() => !actionDisabled && running && runAction(() => services.athenaApplication.stopProjectDiscovery())}>
                                    Stop
                                </button>
                            </div>
                        </div>
                        {!permissionLoading && !canControlDiscovery && <div className='application-discovery__permission'>You do not have permission to control discovery.</div>}
                        {(loading || submitting) && <div className='application-discovery__activity'>{submitting ? 'Updating...' : 'Loading status...'}</div>}
                    </div>
                </div>
            </div>
        </Page>
    );
};
