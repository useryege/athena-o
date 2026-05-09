import {MockupList, Page} from 'argo-ui';
import * as React from 'react';
import {services} from '../../../shared/services';
import {ProjectView} from '../../../shared/services/athena-application-service';

require('./projects-list.scss');

const renderValue = (value: string | number | undefined) => (value === undefined || value === '' ? '-' : value);

const renderShortValue = (value: string | undefined, maxLength = 12) => {
    if (!value) {
        return '-';
    }
    return value.length > maxLength ? `${value.substring(0, maxLength)}...` : value;
};

const AUTO_REFRESH_INTERVAL_MS = 3000;

const renderLastUpdatedAt = (value: Date | null) => (value ? value.toLocaleTimeString() : 'Never');

export const ProjectsList = () => {
    const [projects, setProjects] = React.useState<ProjectView[]>([]);
    const [loading, setLoading] = React.useState(true);
    const [refreshing, setRefreshing] = React.useState(false);
    const [autoRefresh, setAutoRefresh] = React.useState(false);
    const [lastUpdatedAt, setLastUpdatedAt] = React.useState<Date | null>(null);
    const [error, setError] = React.useState<Error | null>(null);
    const requestInFlight = React.useRef(false);
    const intervalRef = React.useRef<number | undefined>(undefined);
    const isMountedRef = React.useRef(false);

    const clearRefreshTimer = React.useCallback(() => {
        if (intervalRef.current === undefined) {
            return;
        }
        window.clearInterval(intervalRef.current);
        intervalRef.current = undefined;
    }, []);

    const loadProjects = React.useCallback(async () => {
        if (requestInFlight.current) {
            return;
        }
        requestInFlight.current = true;
        if (isMountedRef.current) {
            setRefreshing(true);
        }

        try {
            const data = await services.athenaApplication.listProjects();
            if (isMountedRef.current) {
                setProjects(data);
                setLastUpdatedAt(new Date());
                setError(null);
            }
        } catch (err) {
            if (isMountedRef.current) {
                setError(err as Error);
            }
        } finally {
            if (isMountedRef.current) {
                setLoading(false);
                setRefreshing(false);
            }
            requestInFlight.current = false;
        }
    }, []);

    React.useEffect(() => {
        isMountedRef.current = true;
        loadProjects();

        return () => {
            isMountedRef.current = false;
            clearRefreshTimer();
        };
    }, [clearRefreshTimer, loadProjects]);

    const handleStart = React.useCallback(() => {
        if (intervalRef.current !== undefined) {
            return;
        }
        setAutoRefresh(true);
        loadProjects();
        intervalRef.current = window.setInterval(loadProjects, AUTO_REFRESH_INTERVAL_MS);
    }, [loadProjects]);

    const handleStop = React.useCallback(() => {
        clearRefreshTimer();
        setAutoRefresh(false);
    }, [clearRefreshTimer]);

    const handleRefresh = React.useCallback(() => {
        if (autoRefresh) {
            return;
        }
        loadProjects();
    }, [autoRefresh, loadProjects]);

    return (
        <Page title='Projects' toolbar={{breadcrumbs: [{title: 'Projects'}]}}>
            <div className='projects-list'>
                {error && (
                    <div className='projects-list__error'>
                        <i className='fa fa-exclamation-triangle' /> Failed to load projects: {error.message}
                    </div>
                )}

                {loading && projects.length === 0 ? (
                    <MockupList height={50} marginTop={30} />
                ) : (
                    <div className='argo-container'>
                        <div className='white-box projects-list__box'>
                            <div className='projects-list__controls'>
                                <div className='projects-list__actions'>
                                    <button type='button' className='argo-button argo-button--base' disabled={autoRefresh} onClick={handleStart}>
                                        Start
                                    </button>
                                    <button type='button' className='argo-button argo-button--base-o' disabled={!autoRefresh} onClick={handleStop}>
                                        Stop
                                    </button>
                                    <button type='button' className='argo-button argo-button--base' disabled={autoRefresh || refreshing} onClick={handleRefresh}>
                                        {refreshing && !autoRefresh ? 'Refreshing...' : 'Refresh'}
                                    </button>
                                </div>
                                <div className='projects-list__status'>
                                    <span>{autoRefresh ? 'Auto refresh: On' : 'Auto refresh: Off'}</span>
                                    <span>Last updated: {renderLastUpdatedAt(lastUpdatedAt)}</span>
                                    {refreshing && <span>Refreshing...</span>}
                                </div>
                            </div>
                            <div className='argo-table-list projects-list__table'>
                                <div className='argo-table-list__head'>
                                    <div className='projects-list__row'>
                                        <div>#</div>
                                        <div>Name</div>
                                        <div>Symbol</div>
                                        <div>Decimals</div>
                                        <div>Total Supply</div>
                                        <div>Contract</div>
                                        <div>Creator</div>
                                        <div>Block</div>
                                        <div>Tx Index</div>
                                        <div>Tx Hash</div>
                                    </div>
                                </div>
                                {projects.length === 0 ? (
                                    <div className='argo-table-list__row'>
                                        <div className='row'>
                                            <div className='columns small-12 text-center'>No projects found</div>
                                        </div>
                                    </div>
                                ) : (
                                    projects.map((p, index) => (
                                        <div className='argo-table-list__row' key={p.meta?.projectID || `${p.meta?.blockNumber}-${p.meta?.txIndex}`}>
                                            <div className='projects-list__row'>
                                                <div className='projects-list__cell projects-list__cell--rank'>#{index + 1}</div>
                                                <div className='projects-list__cell' title={p.token?.name || ''}>
                                                    {renderValue(p.token?.name)}
                                                </div>
                                                <div className='projects-list__cell'>{renderValue(p.token?.symbol)}</div>
                                                <div className='projects-list__cell'>{renderValue(p.token?.decimals)}</div>
                                                <div className='projects-list__cell' title={p.token?.totalSupply || ''}>
                                                    {renderShortValue(p.token?.totalSupply)}
                                                </div>
                                                <div className='projects-list__cell' title={p.meta?.contract || ''}>
                                                    {renderShortValue(p.meta?.contract)}
                                                </div>
                                                <div className='projects-list__cell' title={p.meta?.creator || ''}>
                                                    {renderShortValue(p.meta?.creator)}
                                                </div>
                                                <div className='projects-list__cell'>{renderValue(p.meta?.blockNumber)}</div>
                                                <div className='projects-list__cell'>{renderValue(p.meta?.txIndex)}</div>
                                                <div className='projects-list__cell' title={p.meta?.txHash || ''}>
                                                    {renderShortValue(p.meta?.txHash)}
                                                </div>
                                            </div>
                                        </div>
                                    ))
                                )}
                            </div>
                        </div>
                    </div>
                )}
            </div>
        </Page>
    );
};
