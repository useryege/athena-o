import {MockupList, Page} from 'argo-ui';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';
import {history} from '../../../app';
import {services} from '../../../shared/services';
import {ProjectView} from '../../../shared/services/athena-application-service';

require('../project-details/project-details.scss');

interface RouteParams {
    contract: string;
}

const renderValue = (value: string | number | boolean | undefined) => {
    if (value === undefined || value === '') {
        return '-';
    }
    return String(value);
};

export const ArchivedProjectDetails = (props: RouteComponentProps<RouteParams>) => {
    const contract = props.match.params.contract;
    const [project, setProject] = React.useState<ProjectView | null>(null);
    const [loading, setLoading] = React.useState(true);
    const [working, setWorking] = React.useState(false);
    const [error, setError] = React.useState<Error | null>(null);

    React.useEffect(() => {
        const req = services.athenaApplication.getArchivedProject(contract);
        req.then(data => {
            setProject(data);
            setError(null);
        })
            .catch(e => setError(e as Error))
            .finally(() => setLoading(false));
        return () => req.abort && req.abort();
    }, [contract]);

    const handleUnarchive = async () => {
        setWorking(true);
        try {
            await services.athenaApplication.unarchiveProject(contract);
            history.push(`/projects/${contract}`);
        } catch (e) {
            setError(e as Error);
        } finally {
            setWorking(false);
        }
    };

    return (
        <Page
            title='Archived Project Details'
            toolbar={{breadcrumbs: [{title: 'Projects', path: '/projects'}, {title: 'Archived', path: '/projects/archived'}, {title: contract}]}}>
            <div className='project-details'>
                {error && (
                    <div className='project-details__error'>
                        <i className='fa fa-exclamation-triangle' /> Failed to load archived project: {error.message}
                    </div>
                )}
                {loading && !project ? (
                    <MockupList height={50} marginTop={30} />
                ) : (
                    <div className='argo-container'>
                        <div className='white-box project-details__box'>
                            <div className='project-details__section-title'>Archived Meta</div>
                            <div className='project-details__grid'>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Contract</span>
                                    <span className='project-details__field-value'>{renderValue(project?.meta?.contract)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Creator</span>
                                    <span className='project-details__field-value'>{renderValue(project?.meta?.creator)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Block Number</span>
                                    <span className='project-details__field-value'>{renderValue(project?.meta?.blockNumber)}</span>
                                </div>
                            </div>
                            <div style={{marginTop: 12}}>
                                <button type='button' className='argo-button argo-button--base' disabled={working} onClick={handleUnarchive}>
                                    {working ? 'Unarchiving...' : 'Unarchive Project'}
                                </button>
                            </div>
                        </div>
                    </div>
                )}
            </div>
        </Page>
    );
};
