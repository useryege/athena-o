import {MockupList, Page} from 'argo-ui';
import * as React from 'react';
import {history} from '../../../app';
import {services} from '../../../shared/services';
import {ProjectOptions, ProjectView} from '../../../shared/services/athena-application-service';
import {ProjectListRow} from '../project-list-row/project-list-row';

require('./projects-list.scss');

const AUTO_REFRESH_INTERVAL_MS = 3000;

const renderLastUpdatedAt = (value: Date | null) => (value ? value.toLocaleTimeString() : 'Never');

const getProjectRowKey = (project: ProjectView, index: number) => {
    if (project.meta?.projectID) {
        return project.meta.projectID;
    }
    if (project.meta?.blockNumber !== undefined && project.meta?.txIndex !== undefined) {
        return `${project.meta.blockNumber}-${project.meta.txIndex}`;
    }
    return `project-${index}`;
};

export const ProjectsList = () => {
    const [projects, setProjects] = React.useState<ProjectView[]>([]);
    const [loading, setLoading] = React.useState(true);
    const [refreshing, setRefreshing] = React.useState(false);
    const [autoRefresh, setAutoRefresh] = React.useState(false);
    const [lastUpdatedAt, setLastUpdatedAt] = React.useState<Date | null>(null);
    const [error, setError] = React.useState<Error | null>(null);
    const [projectOptions, setProjectOptions] = React.useState<ProjectOptions | null>(null);
    const requestRef = React.useRef<{abort?: () => void} | null>(null);
    const optionsRequestRef = React.useRef<{abort?: () => void} | null>(null);
    const intervalRef = React.useRef<number | undefined>(undefined);
    const isMountedRef = React.useRef(false);

    const cleanupRequests = React.useCallback(() => {
        if (intervalRef.current !== undefined) {
            window.clearInterval(intervalRef.current);
            intervalRef.current = undefined;
        }
        if (requestRef.current?.abort) {
            requestRef.current.abort();
            requestRef.current = null;
        }
        if (optionsRequestRef.current?.abort) {
            optionsRequestRef.current.abort();
            optionsRequestRef.current = null;
        }
    }, []);

    const loadProjectOptions = React.useCallback(async () => {
        if (projectOptions || optionsRequestRef.current) {
            return;
        }
        try {
            const req = services.athenaApplication.getProjectOptions();
            optionsRequestRef.current = req;
            const options = await req;
            if (isMountedRef.current) {
                setProjectOptions(options || null);
            }
        } catch (err) {
            if (isMountedRef.current) {
                setError(err as Error);
            }
        } finally {
            optionsRequestRef.current = null;
        }
    }, [projectOptions]);

    const loadProjects = React.useCallback(async () => {
        if (requestRef.current) {
            return;
        }
        if (isMountedRef.current) {
            setRefreshing(true);
        }

        try {
            const req = services.athenaApplication.listProjects();
            requestRef.current = req;
            const data = await req;
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
            requestRef.current = null;
        }
    }, []);

    React.useEffect(() => {
        isMountedRef.current = true;
        loadProjects();
        loadProjectOptions();

        return () => {
            isMountedRef.current = false;
            cleanupRequests();
        };
    }, [cleanupRequests, loadProjectOptions, loadProjects]);

    const handleStart = React.useCallback(() => {
        if (intervalRef.current !== undefined) {
            return;
        }
        setAutoRefresh(true);
        loadProjects();
        intervalRef.current = window.setInterval(loadProjects, AUTO_REFRESH_INTERVAL_MS);
    }, [loadProjects]);

    const handleStop = React.useCallback(() => {
        cleanupRequests();
        setAutoRefresh(false);
    }, [cleanupRequests]);

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
                                    <button type='button' className='argo-button argo-button--base-o' onClick={() => history.push('/projects/archived')}>
                                        Archived
                                    </button>
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
                                        <div>Blacklist</div>
                                        <div>Mint Risk</div>
                                        <div title='Is Open Source'>Open Src</div>
                                        <div title='WETH Quote + Remove Liquidity'>WETH Pair</div>
                                        <div title='USDT Quote + Remove Liquidity'>USDT Pair</div>
                                        <div title='Block Time'>Block Time</div>
                                    </div>
                                </div>
                                {projects.length === 0 ? (
                                    <div className='argo-table-list__row'>
                                        <div className='row'>
                                            <div className='columns small-12 text-center'>No projects found</div>
                                        </div>
                                    </div>
                                ) : (
                                    projects.map((project, index) => (
                                        <ProjectListRow
                                            key={getProjectRowKey(project, index)}
                                            project={project}
                                            index={index}
                                            usdtDecimals={projectOptions?.usdtDecimals}
                                            onClick={project.meta?.projectID ? () => history.push(`/projects/${project.meta!.projectID}`) : undefined}
                                        />
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
