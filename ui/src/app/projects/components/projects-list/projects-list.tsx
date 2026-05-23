import {MockupList, Page} from 'argo-ui';
import * as React from 'react';
import {history} from '../../../app';
import {services} from '../../../shared/services';
import {ProjectListItem, ProjectOptions} from '../../../shared/services/athena-application-service';
import {ProjectListRow} from '../project-list-row/project-list-row';
import {buildProjectsListSearch, parseProjectsListSearch} from './projects-list-query';

require('./projects-list.scss');

const AUTO_REFRESH_INTERVAL_MS = 3000;
const PAGE_SIZE = 10;

interface ProjectsListCacheEntry {
    projects: ProjectListItem[];
    page: number;
    total: number;
    lastUpdatedAt: Date | null;
}

const projectsListCache = new Map<string, ProjectsListCacheEntry>();

const renderLastUpdatedAt = (value: Date | null) => (value ? value.toLocaleTimeString() : 'Never');
const isAbortedError = (err: unknown) =>
    String((err as any)?.message || '')
        .toLowerCase()
        .includes('abort');

const getProjectRowKey = (project: ProjectListItem, index: number) => {
    if (project.contract) {
        return project.contract;
    }
    if (project.blockNumber !== undefined && project.txIndex !== undefined) {
        return `${project.blockNumber}-${project.txIndex}`;
    }
    return `project-${index}`;
};

export const ProjectsList = () => {
    const initialQueryState = React.useMemo(() => parseProjectsListSearch(history.location.search), []);
    const initialCache = React.useMemo(
        () => projectsListCache.get(buildProjectsListSearch(initialQueryState.page)),
        [initialQueryState.page]
    );
    const [projects, setProjects] = React.useState<ProjectListItem[]>(initialCache?.projects || []);
    const [loading, setLoading] = React.useState(!initialCache);
    const [refreshing, setRefreshing] = React.useState(false);
    const [autoRefresh, setAutoRefresh] = React.useState(false);
    const [page, setPage] = React.useState(initialCache?.page || initialQueryState.page);
    const [total, setTotal] = React.useState(initialCache?.total || 0);
    const [lastUpdatedAt, setLastUpdatedAt] = React.useState<Date | null>(initialCache?.lastUpdatedAt || null);
    const [error, setError] = React.useState<Error | null>(null);
    const [projectOptions, setProjectOptions] = React.useState<ProjectOptions | null>(null);
    const requestRef = React.useRef<{abort?: () => void} | null>(null);
    const optionsRequestRef = React.useRef<{abort?: () => void} | null>(null);
    const intervalRef = React.useRef<number | undefined>(undefined);
    const isMountedRef = React.useRef(false);
    const currentPageRef = React.useRef(initialQueryState.page);

    const syncUrlState = React.useCallback((nextPage: number) => {
        const nextSearch = buildProjectsListSearch(nextPage);
        if (history.location.search !== nextSearch) {
            history.replace({
                pathname: history.location.pathname,
                search: nextSearch
            });
        }
    }, []);

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
            if (isMountedRef.current && !isAbortedError(err)) {
                setError(err as Error);
            }
        } finally {
            optionsRequestRef.current = null;
        }
    }, [projectOptions]);

    const loadProjects = React.useCallback(
        async (targetPage?: number, syncSearch = false) => {
            if (requestRef.current) {
                return;
            }
            const pageToLoad = targetPage ?? currentPageRef.current;
            if (isMountedRef.current) {
                setRefreshing(true);
            }

            try {
                const req = services.athenaApplication.listProjects(pageToLoad, PAGE_SIZE);
                requestRef.current = req;
                const data = await req;
                if (isMountedRef.current) {
                    const updatedAt = new Date();
                    setProjects(data.items);
                    setTotal(data.total);
                    setPage(data.page);
                    currentPageRef.current = data.page;
                    if (syncSearch) {
                        syncUrlState(data.page);
                    }
                    setLastUpdatedAt(updatedAt);
                    setError(null);
                    projectsListCache.set(buildProjectsListSearch(data.page), {
                        projects: data.items,
                        total: data.total,
                        page: data.page,
                        lastUpdatedAt: updatedAt
                    });
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
        },
        [syncUrlState]
    );

    React.useEffect(() => {
        isMountedRef.current = true;
        loadProjects(initialQueryState.page, true);
        loadProjectOptions();

        return () => {
            isMountedRef.current = false;
            cleanupRequests();
        };
    }, [cleanupRequests, initialQueryState.page, loadProjectOptions, loadProjects]);

    const handleStart = React.useCallback(() => {
        if (intervalRef.current !== undefined) {
            return;
        }
        setAutoRefresh(true);
        loadProjects(currentPageRef.current);
        intervalRef.current = window.setInterval(() => {
            loadProjects(currentPageRef.current);
        }, AUTO_REFRESH_INTERVAL_MS);
    }, [loadProjects]);

    const handleStop = React.useCallback(() => {
        cleanupRequests();
        setAutoRefresh(false);
    }, [cleanupRequests]);

    const handleRefresh = React.useCallback(() => {
        if (autoRefresh) {
            return;
        }
        loadProjects(currentPageRef.current);
    }, [autoRefresh, loadProjects]);

    const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

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
                                    <span>Total: {total}</span>
                                    <span>
                                        Page: {page}/{totalPages}
                                    </span>
                                    <span>Last updated: {renderLastUpdatedAt(lastUpdatedAt)}</span>
                                    {refreshing && <span>Refreshing...</span>}
                                </div>
                            </div>
                            <div className='argo-table-list projects-list__table'>
                                <div className='argo-table-list__head'>
                                    <div className='projects-list__row'>
                                        <div>#</div>
                                        <div>Name</div>
                                        <div>Contract</div>
                                        <div>Mint Risk</div>
                                        <div title='Is Open Source'>Open Src</div>
                                        <div title='WETH Quote + Remove Liquidity'>WETH Pair</div>
                                        <div title='USDT Quote + Remove Liquidity'>USDT Pair</div>
                                        <div title='Creator total asset in USDT'>Creator Asset</div>
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
                                            index={(page - 1) * PAGE_SIZE + index}
                                            usdtDecimals={projectOptions?.usdtDecimals}
                                            to={project.contract ? `/projects/${project.contract}${buildProjectsListSearch(currentPageRef.current)}` : undefined}
                                        />
                                    ))
                                )}
                            </div>
                            <div className='projects-list__controls' style={{marginTop: 12}}>
                                <div className='projects-list__actions'>
                                    <button
                                        type='button'
                                        className='argo-button argo-button--base-o'
                                        disabled={page <= 1 || refreshing}
                                        onClick={() => loadProjects(page - 1, true)}>
                                        Prev
                                    </button>
                                    <button
                                        type='button'
                                        className='argo-button argo-button--base-o'
                                        disabled={page >= totalPages || refreshing}
                                        onClick={() => loadProjects(page + 1, true)}>
                                        Next
                                    </button>
                                </div>
                            </div>
                        </div>
                    </div>
                )}
            </div>
        </Page>
    );
};
