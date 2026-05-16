import {MockupList, Page} from 'argo-ui';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';
import {history} from '../../../app';
import {services} from '../../../shared/services';
import {PairV2State, ProjectView} from '../../../shared/services/athena-application-service';

require('./project-details.scss');

const AUTO_REFRESH_INTERVAL_MS = 3000;

const renderValue = (value: string | number | boolean | undefined) => {
    if (value === undefined || value === '') {
        return '-';
    }
    return String(value);
};

const renderPercent = (value?: string) => {
    if (value === undefined || value === null || value === '') {
        return '-';
    }
    return `${value}%`;
};

const renderPairSection = (title: string, pair?: PairV2State) => (
    <div className='white-box project-details__box'>
        <div className='project-details__section-title'>{title}</div>
        <div className='project-details__grid'>
            <div className='project-details__field'>
                <span className='project-details__field-label'>Contract Created</span>
                <span className='project-details__field-value'>{renderValue(pair?.isCreated)}</span>
            </div>
            <div className='project-details__field'>
                <span className='project-details__field-label'>Contract</span>
                <span className='project-details__field-value'>{renderValue(pair?.contract)}</span>
            </div>
            <div className='project-details__field'>
                <span className='project-details__field-label'>Token 0</span>
                <span className='project-details__field-value'>{renderValue(pair?.token0)}</span>
            </div>
            <div className='project-details__field'>
                <span className='project-details__field-label'>Token 1</span>
                <span className='project-details__field-value'>{renderValue(pair?.token1)}</span>
            </div>
            <div className='project-details__field'>
                <span className='project-details__field-label'>Total Supply</span>
                <span className='project-details__field-value'>{renderValue(pair?.totalSupply)}</span>
            </div>
            <div className='project-details__field'>
                <span className='project-details__field-label'>Reserve 0</span>
                <span className='project-details__field-value'>{renderValue(pair?.reserve0)}</span>
            </div>
            <div className='project-details__field'>
                <span className='project-details__field-label'>Reserve 1</span>
                <span className='project-details__field-value'>{renderValue(pair?.reserve1)}</span>
            </div>
            <div className='project-details__field'>
                <span className='project-details__field-label'>Block Timestamp Last</span>
                <span className='project-details__field-value'>{renderValue(pair?.blockTimestampLast)}</span>
            </div>
            <div className='project-details__field'>
                <span className='project-details__field-label'>Base Balance</span>
                <span className='project-details__field-value'>{renderValue(pair?.baseBalance)}</span>
            </div>
            <div className='project-details__field'>
                <span className='project-details__field-label'>Quote Balance</span>
                <span className='project-details__field-value'>{renderValue(pair?.quoteBalance)}</span>
            </div>
            <div className='project-details__field'>
                <span className='project-details__field-label'>Quote USDT Value</span>
                <span className='project-details__field-value'>{renderValue(pair?.quoteUsdtValue)}</span>
            </div>
            <div className='project-details__field'>
                <span className='project-details__field-label'>Locked Liquidity</span>
                <span className='project-details__field-value'>{renderValue(pair?.lockedLiquidity)}</span>
            </div>
            <div className='project-details__field'>
                <span className='project-details__field-label'>Fee Address Hold Liquidity Balance</span>
                <span className='project-details__field-value'>{renderValue(pair?.feeAddressHoldLiquidityBalance)}</span>
            </div>
            <div className='project-details__field'>
                <span className='project-details__field-label'>Is Remove Liquidity</span>
                <span className='project-details__field-value'>{renderValue(pair?.isRemoveLiquidity)}</span>
            </div>
            <div className='project-details__field'>
                <span className='project-details__field-label'>Fee Address Hold Liquidity Ratio</span>
                <span className='project-details__field-value'>{renderPercent(pair?.feeAddressHoldLiquidityRatio)}</span>
            </div>
        </div>
    </div>
);

interface RouteParams {
    contract: string;
}

export const ProjectDetails = (props: RouteComponentProps<RouteParams>) => {
    const contract = props.match.params.contract;
    const [project, setProject] = React.useState<ProjectView | null>(null);
    const [loading, setLoading] = React.useState(true);
    const [archiving, setArchiving] = React.useState(false);
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
            const req = services.athenaApplication.getProject(contract);
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
    }, [contract]);

    React.useEffect(() => {
        isMountedRef.current = true;
        loadProject();
        intervalRef.current = window.setInterval(loadProject, AUTO_REFRESH_INTERVAL_MS);

        return () => {
            isMountedRef.current = false;
            cleanupRequests();
        };
    }, [cleanupRequests, loadProject]);

    const breadcrumbs = [{title: 'Projects', path: '/projects'}, {title: contract}];

    const handleArchive = React.useCallback(async () => {
        setArchiving(true);
        try {
            await services.athenaApplication.archiveProject(contract);
            history.push('/projects/archived');
        } catch (err) {
            if (isMountedRef.current) {
                setError(err as Error);
            }
        } finally {
            if (isMountedRef.current) {
                setArchiving(false);
            }
        }
    }, [contract]);

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
                        <div style={{textAlign: 'right', marginBottom: '10px'}}>
                            <button type='button' className='argo-button argo-button--base' disabled={archiving} onClick={handleArchive}>
                                {archiving ? 'Archiving...' : 'Archive Project'}
                            </button>
                        </div>

                        <div className='white-box project-details__box'>
                            <div className='project-details__section-title'>Meta</div>
                            <div className='project-details__grid'>
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
                                    <span className='project-details__field-value'>{renderValue(project.chainState?.token?.name)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Symbol</span>
                                    <span className='project-details__field-value'>{renderValue(project.chainState?.token?.symbol)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Decimals</span>
                                    <span className='project-details__field-value'>{renderValue(project.chainState?.token?.decimals)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Total Supply</span>
                                    <span className='project-details__field-value'>{renderValue(project.chainState?.token?.totalSupply)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Valid ERC20</span>
                                    <span className='project-details__field-value'>
                                        {project.chainState?.token?.isValidERC20 !== undefined ? (
                                            <span className={`project-details__badge project-details__badge--${project.chainState.token.isValidERC20 ? 'positive' : 'negative'}`}>
                                                {project.chainState.token.isValidERC20 ? 'Yes' : 'No'}
                                            </span>
                                        ) : (
                                            '-'
                                        )}
                                    </span>
                                </div>
                            </div>
                        </div>

                        {project.meta?.sourceCode && (
                            <div className='white-box project-details__box'>
                                <div className='project-details__section-title'>Source Code</div>
                                {project.meta?.sourceCode && (
                                    <div className='project-details__field'>
                                        <span className='project-details__field-label'>Contract Source Code</span>
                                        <div className='project-details__code-block'>{project.meta.sourceCode}</div>
                                    </div>
                                )}
                            </div>
                        )}

                        {renderPairSection('WETH V2 Pool', project.chainState?.wethPair)}
                        {renderPairSection('USDT V2 Pool', project.chainState?.usdtPair)}

                        {project.meta?.creatorResult && (
                            <div className='white-box project-details__box'>
                                <div className='project-details__section-title'>Simulation</div>
                                <div className='project-details__grid'>
                                    <div className='project-details__field'>
                                        <span className='project-details__field-label'>Mint via Transfer to WETH Pair</span>
                                        <span className='project-details__field-value'>
                                            <span
                                                className={`project-details__badge project-details__badge--${project.meta.creatorResult.canMintViaTransferToWethPair ? 'negative' : 'positive'}`}>
                                                {project.meta.creatorResult.canMintViaTransferToWethPair ? 'Yes' : 'No'}
                                            </span>
                                        </span>
                                    </div>
                                    <div className='project-details__field'>
                                        <span className='project-details__field-label'>Mint via Transfer to USDT Pair</span>
                                        <span className='project-details__field-value'>
                                            <span
                                                className={`project-details__badge project-details__badge--${project.meta.creatorResult.canMintViaTransferToUsdtPair ? 'negative' : 'positive'}`}>
                                                {project.meta.creatorResult.canMintViaTransferToUsdtPair ? 'Yes' : 'No'}
                                            </span>
                                        </span>
                                    </div>
                                    <div className='project-details__field'>
                                        <span className='project-details__field-label'>Mint from Dead via transferFrom</span>
                                        <span className='project-details__field-value'>
                                            <span
                                                className={`project-details__badge project-details__badge--${project.meta.creatorResult.canMintFromDeadViaTransferFrom ? 'negative' : 'positive'}`}>
                                                {project.meta.creatorResult.canMintFromDeadViaTransferFrom ? 'Yes' : 'No'}
                                            </span>
                                        </span>
                                    </div>
                                    <div className='project-details__field'>
                                        <span className='project-details__field-label'>Mint from Zero via transferFrom</span>
                                        <span className='project-details__field-value'>
                                            <span
                                                className={`project-details__badge project-details__badge--${project.meta.creatorResult.canMintFromZeroViaTransferFrom ? 'negative' : 'positive'}`}>
                                                {project.meta.creatorResult.canMintFromZeroViaTransferFrom ? 'Yes' : 'No'}
                                            </span>
                                        </span>
                                    </div>
                                    <div className='project-details__field'>
                                        <span className='project-details__field-label'>Mint from WETH Pair via transferFrom</span>
                                        <span className='project-details__field-value'>
                                            <span
                                                className={`project-details__badge project-details__badge--${project.meta.creatorResult.canMintFromWethPairViaTransferFrom ? 'negative' : 'positive'}`}>
                                                {project.meta.creatorResult.canMintFromWethPairViaTransferFrom ? 'Yes' : 'No'}
                                            </span>
                                        </span>
                                    </div>
                                    <div className='project-details__field'>
                                        <span className='project-details__field-label'>Mint from USDT Pair via transferFrom</span>
                                        <span className='project-details__field-value'>
                                            <span
                                                className={`project-details__badge project-details__badge--${project.meta.creatorResult.canMintFromUsdtPairViaTransferFrom ? 'negative' : 'positive'}`}>
                                                {project.meta.creatorResult.canMintFromUsdtPairViaTransferFrom ? 'Yes' : 'No'}
                                            </span>
                                        </span>
                                    </div>
                                </div>
                            </div>
                        )}

                        {project.meta?.sourceCodeBlacklist && (
                            <div className='white-box project-details__box'>
                                <div className='project-details__section-title'>Analysis</div>
                                <div className='project-details__grid'>
                                    <div className='project-details__field'>
                                        <span className='project-details__field-label'>Has Blacklist Fields</span>
                                        <span className='project-details__field-value'>
                                            <span
                                                className={`project-details__badge project-details__badge--${project.meta.sourceCodeBlacklist.hasBlacklistFields ? 'negative' : 'positive'}`}>
                                                {project.meta.sourceCodeBlacklist.hasBlacklistFields ? 'Yes' : 'No'}
                                            </span>
                                        </span>
                                    </div>
                                    <div className='project-details__field' style={{gridColumn: '1 / -1'}}>
                                        <span className='project-details__field-label'>Blacklist Fields</span>
                                        <div className='project-details__chip-list'>
                                            {project.meta.sourceCodeBlacklist.blacklistFields && project.meta.sourceCodeBlacklist.blacklistFields.length > 0 ? (
                                                project.meta.sourceCodeBlacklist.blacklistFields.map((field, i) => (
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
