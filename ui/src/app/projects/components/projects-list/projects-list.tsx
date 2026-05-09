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

export const ProjectsList = () => {
    const [projects, setProjects] = React.useState<ProjectView[]>([]);
    const [loading, setLoading] = React.useState(true);
    const [error, setError] = React.useState<Error | null>(null);
    const requestInFlight = React.useRef(false);

    React.useEffect(() => {
        let isMounted = true;

        const loadProjects = async () => {
            if (requestInFlight.current) {
                return;
            }
            requestInFlight.current = true;

            try {
                const data = await services.athenaApplication.listProjects();
                if (isMounted) {
                    setProjects(data);
                    setError(null);
                }
            } catch (err) {
                if (isMounted) {
                    setError(err as Error);
                }
            } finally {
                if (isMounted) {
                    setLoading(false);
                }
                requestInFlight.current = false;
            }
        };

        // Initial load
        loadProjects();

        // Poll every 3 seconds
        const timer = setInterval(loadProjects, 3000);

        return () => {
            isMounted = false;
            clearInterval(timer);
        };
    }, []);

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
                        <div className='white-box'>
                            <div className='argo-table-list projects-list__table'>
                                <div className='argo-table-list__head'>
                                    <div className='projects-list__row'>
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
                                    projects.map(p => (
                                        <div className='argo-table-list__row' key={p.meta?.projectID || `${p.meta?.blockNumber}-${p.meta?.txIndex}`}>
                                            <div className='projects-list__row'>
                                                <div className='projects-list__cell' title={p.initState?.name || ''}>
                                                    {renderValue(p.initState?.name)}
                                                </div>
                                                <div className='projects-list__cell'>{renderValue(p.initState?.symbol)}</div>
                                                <div className='projects-list__cell'>{renderValue(p.initState?.decimals)}</div>
                                                <div className='projects-list__cell' title={p.initState?.totalSupply || ''}>
                                                    {renderShortValue(p.initState?.totalSupply)}
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
