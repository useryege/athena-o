import {MockupList, Page} from 'argo-ui';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';
import {services} from '../../../shared/services';
import {ProjectView} from '../../../shared/services/athena-application-service';

require('./project-details.scss');

const AUTO_REFRESH_INTERVAL_MS = 3000;

const renderValue = (value: string | number | boolean | undefined) => {
    if (value === undefined || value === '') {
        return '-';
    }
    return String(value);
};

interface RouteParams {
    projectID: string;
}

export const ProjectDetails = (props: RouteComponentProps<RouteParams>) => {
    const projectID = props.match.params.projectID;
    const [project, setProject] = React.useState<ProjectView | null>(null);
    const [loading, setLoading] = React.useState(true);
    const [lastUpdatedAt, setLastUpdatedAt] = React.useState<Date | null>(null);
    const [error, setError] = React.useState<Error | null>(null);

    const requestRef = React.useRef<{abort?: () => void} | null>(null);
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
    }, []);

    const loadProject = React.useCallback(async () => {
        if (requestRef.current) {
            return;
        }

        try {
            const req = services.athenaApplication.getProject(projectID);
            requestRef.current = req;
            const data = await req;
            if (isMountedRef.current) {
                setProject(data);
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
            }
            requestRef.current = null;
        }
    }, [projectID]);

    React.useEffect(() => {
        isMountedRef.current = true;
        loadProject();
        intervalRef.current = window.setInterval(loadProject, AUTO_REFRESH_INTERVAL_MS);

        return () => {
            isMountedRef.current = false;
            cleanupRequests();
        };
    }, [cleanupRequests, loadProject]);

    const breadcrumbs = [{title: 'Projects', path: '/projects'}, {title: projectID}];

    return (
        <Page title='Project Details' toolbar={{breadcrumbs}}>
            <div className='project-details'>
                {error && (
                    <div className='project-details__error'>
                        <i className='fa fa-exclamation-triangle' /> Failed to load project: {error.message}
                    </div>
                )}

                {loading && !project ? (
                    <MockupList height={50} marginTop={30} />
                ) : project ? (
                    <div className='argo-container'>
                        <div style={{textAlign: 'right', fontSize: '12px', color: '#6d7f8b', marginBottom: '10px'}}>
                            Last updated: {lastUpdatedAt ? lastUpdatedAt.toLocaleTimeString() : 'Never'}
                        </div>

                        <div className='white-box project-details__box'>
                            <div className='project-details__section-title'>Meta</div>
                            <div className='project-details__grid'>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Project ID</span>
                                    <span className='project-details__field-value'>{renderValue(project.meta?.projectID)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Contract</span>
                                    <span className='project-details__field-value'>{renderValue(project.meta?.contract)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Creator</span>
                                    <span className='project-details__field-value'>{renderValue(project.meta?.creator)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Block Number</span>
                                    <span className='project-details__field-value'>{renderValue(project.meta?.blockNumber)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Tx Hash</span>
                                    <span className='project-details__field-value'>{renderValue(project.meta?.txHash)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Tx Index</span>
                                    <span className='project-details__field-value'>{renderValue(project.meta?.txIndex)}</span>
                                </div>
                            </div>
                        </div>

                        <div className='white-box project-details__box'>
                            <div className='project-details__section-title'>Token State</div>
                            <div className='project-details__grid'>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Name</span>
                                    <span className='project-details__field-value'>{renderValue(project.token?.name)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Symbol</span>
                                    <span className='project-details__field-value'>{renderValue(project.token?.symbol)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Decimals</span>
                                    <span className='project-details__field-value'>{renderValue(project.token?.decimals)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Total Supply</span>
                                    <span className='project-details__field-value'>{renderValue(project.token?.totalSupply)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Valid ERC20</span>
                                    <span className='project-details__field-value'>
                                        {project.token?.isValidERC20 !== undefined ? (
                                            <span className={`project-details__badge project-details__badge--${project.token.isValidERC20 ? 'positive' : 'negative'}`}>
                                                {project.token.isValidERC20 ? 'Yes' : 'No'}
                                            </span>
                                        ) : (
                                            '-'
                                        )}
                                    </span>
                                </div>
                            </div>
                        </div>

                        {(project.sourceCode?.sourceCode || project.token?.sourceCode) && (
                            <div className='white-box project-details__box'>
                                <div className='project-details__section-title'>Source Code</div>
                                {(project.sourceCode?.sourceCode || project.token?.sourceCode) && (
                                    <div className='project-details__field'>
                                        <span className='project-details__field-label'>Contract Source Code</span>
                                        <div className='project-details__code-block'>{project.sourceCode?.sourceCode || project.token?.sourceCode}</div>
                                    </div>
                                )}
                            </div>
                        )}

                        <div className='white-box project-details__box'>
                            <div className='project-details__section-title'>WETH V2 Pool</div>
                            <div className='project-details__grid'>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Contract Created</span>
                                    <span className='project-details__field-value'>{renderValue(project.wethV2Pool?.isContractCreated)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Contract</span>
                                    <span className='project-details__field-value'>{renderValue(project.wethV2Pool?.contract)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Token 0</span>
                                    <span className='project-details__field-value'>{renderValue(project.wethV2Pool?.token0)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Token 1</span>
                                    <span className='project-details__field-value'>{renderValue(project.wethV2Pool?.token1)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Total Supply</span>
                                    <span className='project-details__field-value'>{renderValue(project.wethV2Pool?.totalSupply)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Reserve 0</span>
                                    <span className='project-details__field-value'>{renderValue(project.wethV2Pool?.reserve0)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Reserve 1</span>
                                    <span className='project-details__field-value'>{renderValue(project.wethV2Pool?.reserve1)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Block Timestamp Last</span>
                                    <span className='project-details__field-value'>{renderValue(project.wethV2Pool?.blockTimestampLast)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Token Reserve Balance</span>
                                    <span className='project-details__field-value'>{renderValue(project.wethV2Pool?.tokenReserveBalance)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>WETH Reserve Balance</span>
                                    <span className='project-details__field-value'>{renderValue(project.wethV2Pool?.wethReserveBalance)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Locked Liquidity</span>
                                    <span className='project-details__field-value'>{renderValue(project.wethV2Pool?.lockedLiquidity)}</span>
                                </div>
                            </div>
                        </div>

                        {project.simulate?.creatorResult && (
                            <div className='white-box project-details__box'>
                                <div className='project-details__section-title'>Simulation</div>
                                <div className='project-details__grid'>
                                    <div className='project-details__field'>
                                        <span className='project-details__field-label'>Mint via Transfer</span>
                                        <span className='project-details__field-value'>
                                            <span
                                                className={`project-details__badge project-details__badge--${project.simulate.creatorResult.canMintViaTransfer ? 'negative' : 'positive'}`}>
                                                {project.simulate.creatorResult.canMintViaTransfer ? 'Yes' : 'No'}
                                            </span>
                                        </span>
                                    </div>
                                    <div className='project-details__field'>
                                        <span className='project-details__field-label'>Mint from Dead via transferFrom</span>
                                        <span className='project-details__field-value'>
                                            <span
                                                className={`project-details__badge project-details__badge--${project.simulate.creatorResult.canMintFromDeadViaTransferFrom ? 'negative' : 'positive'}`}>
                                                {project.simulate.creatorResult.canMintFromDeadViaTransferFrom ? 'Yes' : 'No'}
                                            </span>
                                        </span>
                                    </div>
                                    <div className='project-details__field'>
                                        <span className='project-details__field-label'>Mint from Zero via transferFrom</span>
                                        <span className='project-details__field-value'>
                                            <span
                                                className={`project-details__badge project-details__badge--${project.simulate.creatorResult.canMintFromZeroViaTransferFrom ? 'negative' : 'positive'}`}>
                                                {project.simulate.creatorResult.canMintFromZeroViaTransferFrom ? 'Yes' : 'No'}
                                            </span>
                                        </span>
                                    </div>
                                    <div className='project-details__field'>
                                        <span className='project-details__field-label'>Mint from Pair via transferFrom</span>
                                        <span className='project-details__field-value'>
                                            <span
                                                className={`project-details__badge project-details__badge--${project.simulate.creatorResult.canMintFromPairViaTransferFrom ? 'negative' : 'positive'}`}>
                                                {project.simulate.creatorResult.canMintFromPairViaTransferFrom ? 'Yes' : 'No'}
                                            </span>
                                        </span>
                                    </div>
                                </div>
                            </div>
                        )}

                        {project.analysis?.sourceCodeBlacklist && (
                            <div className='white-box project-details__box'>
                                <div className='project-details__section-title'>Analysis</div>
                                <div className='project-details__grid'>
                                    <div className='project-details__field'>
                                        <span className='project-details__field-label'>Has Blacklist Fields</span>
                                        <span className='project-details__field-value'>
                                            <span
                                                className={`project-details__badge project-details__badge--${project.analysis.sourceCodeBlacklist.hasBlacklistFields ? 'negative' : 'positive'}`}>
                                                {project.analysis.sourceCodeBlacklist.hasBlacklistFields ? 'Yes' : 'No'}
                                            </span>
                                        </span>
                                    </div>
                                    <div className='project-details__field' style={{gridColumn: '1 / -1'}}>
                                        <span className='project-details__field-label'>Blacklist Fields</span>
                                        <div className='project-details__chip-list'>
                                            {project.analysis.sourceCodeBlacklist.blacklistFields && project.analysis.sourceCodeBlacklist.blacklistFields.length > 0 ? (
                                                project.analysis.sourceCodeBlacklist.blacklistFields.map((field, i) => (
                                                    <span key={i} className='project-details__chip'>
                                                        {field}
                                                    </span>
                                                ))
                                            ) : (
                                                <span className='project-details__field-value'>-</span>
                                            )}
                                        </div>
                                    </div>
                                </div>
                            </div>
                        )}
                    </div>
                ) : null}
            </div>
        </Page>
    );
};
