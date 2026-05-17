import {MockupList, Page} from 'argo-ui';
import * as React from 'react';
import {history} from '../../../app';
import {services} from '../../../shared/services';
import {ProjectOptions, ProjectView} from '../../../shared/services/athena-application-service';
import {ProjectListRow} from '../project-list-row/project-list-row';

require('../projects-list/projects-list.scss');

const PAGE_SIZE = 20;

const getProjectRowKey = (project: ProjectView, index: number) => project.meta?.contract || `archived-project-${index}`;

export const ArchivedProjectsList = () => {
    const [projects, setProjects] = React.useState<ProjectView[]>([]);
    const [loading, setLoading] = React.useState(true);
    const [error, setError] = React.useState<Error | null>(null);
    const [page, setPage] = React.useState(1);
    const [total, setTotal] = React.useState(0);
    const [refreshing, setRefreshing] = React.useState(false);
    const [projectOptions, setProjectOptions] = React.useState<ProjectOptions | null>(null);

    const reqRef = React.useRef<{abort?: () => void} | null>(null);
    const optionsReqRef = React.useRef<{abort?: () => void} | null>(null);

    const load = React.useCallback(async (targetPage: number) => {
        if (reqRef.current) {
            return;
        }
        setRefreshing(true);
        try {
            const req = services.athenaApplication.listArchivedProjects(targetPage, PAGE_SIZE);
            reqRef.current = req;
            const data = await req;
            setProjects(data.items);
            setTotal(data.total);
            setPage(data.page);
            setError(null);
        } catch (e) {
            setError(e as Error);
        } finally {
            reqRef.current = null;
            setLoading(false);
            setRefreshing(false);
        }
    }, []);

    React.useEffect(() => {
        load(1);
        return () => {
            if (reqRef.current?.abort) {
                reqRef.current.abort();
            }
            if (optionsReqRef.current?.abort) {
                optionsReqRef.current.abort();
            }
        };
    }, [load]);

    React.useEffect(() => {
        if (projectOptions || optionsReqRef.current) {
            return;
        }
        const req = services.athenaApplication.getProjectOptions();
        optionsReqRef.current = req;
        req.then(options => setProjectOptions(options || null))
            .catch(() => undefined)
            .finally(() => {
                optionsReqRef.current = null;
            });
    }, [projectOptions]);

    const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

    return (
        <Page title='Archived Projects' toolbar={{breadcrumbs: [{title: 'Projects', path: '/projects'}, {title: 'Archived'}]}}>
            <div className='projects-list'>
                {error && (
                    <div className='projects-list__error'>
                        <i className='fa fa-exclamation-triangle' /> Failed to load archived projects: {error.message}
                    </div>
                )}
                {loading && projects.length === 0 ? (
                    <MockupList height={50} marginTop={30} />
                ) : (
                    <div className='argo-container'>
                        <div className='white-box projects-list__box'>
                            <div className='projects-list__controls'>
                                <div className='projects-list__actions'>
                                    <button type='button' className='argo-button argo-button--base-o' onClick={() => history.push('/projects')}>
                                        Back to Active
                                    </button>
                                    <button type='button' className='argo-button argo-button--base' disabled={refreshing} onClick={() => load(page)}>
                                        {refreshing ? 'Refreshing...' : 'Refresh'}
                                    </button>
                                </div>
                                <div className='projects-list__status'>
                                    <span>Total: {total}</span>
                                    <span>
                                        Page: {page}/{totalPages}
                                    </span>
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
                                        <div>Open Src</div>
                                        <div>WETH Pair</div>
                                        <div>USDT Pair</div>
                                        <div>Creator Asset (USDT)</div>
                                        <div>Block Time</div>
                                    </div>
                                </div>
                                {projects.length === 0 ? (
                                    <div className='argo-table-list__row'>
                                        <div className='row'>
                                            <div className='columns small-12 text-center'>No archived projects</div>
                                        </div>
                                    </div>
                                ) : (
                                    projects.map((project, index) => (
                                        <ProjectListRow
                                            key={getProjectRowKey(project, index)}
                                            project={project}
                                            index={(page - 1) * PAGE_SIZE + index}
                                            usdtDecimals={projectOptions?.usdtDecimals}
                                            onClick={project.meta?.contract ? () => history.push(`/projects/archived/${project.meta!.contract}`) : undefined}
                                        />
                                    ))
                                )}
                            </div>

                            <div className='projects-list__controls' style={{marginTop: 12}}>
                                <div className='projects-list__actions'>
                                    <button type='button' className='argo-button argo-button--base-o' disabled={page <= 1 || refreshing} onClick={() => load(page - 1)}>
                                        Prev
                                    </button>
                                    <button type='button' className='argo-button argo-button--base-o' disabled={page >= totalPages || refreshing} onClick={() => load(page + 1)}>
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
