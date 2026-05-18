import {ErrorNotification, FormField, MockupList, NotificationType, Page} from 'argo-ui';
import * as React from 'react';
import {Text} from 'react-form';
import {RouteComponentProps} from 'react-router';
import {Context} from '../../../shared/context';
import {services} from '../../../shared/services';
import {PairV2State, ProjectEventLog, ProjectView} from '../../../shared/services/athena-application-service';

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

const renderEventType = (eventType?: number) => {
    switch (eventType) {
        case 1:
            return 'Project Created';
        case 2:
            return 'Contract Source Opened';
        default:
            return `Unknown(${eventType ?? 0})`;
    }
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
    const ctx = React.useContext(Context);
    const contract = props.match.params.contract;
    const [project, setProject] = React.useState<ProjectView | null>(null);
    const [eventLogs, setEventLogs] = React.useState<ProjectEventLog[]>([]);
    const [loading, setLoading] = React.useState(true);
    const [changingArchiveState, setChangingArchiveState] = React.useState(false);
    const [isBlacklistChecking, setIsBlacklistChecking] = React.useState(true);
    const [isBlacklisted, setIsBlacklisted] = React.useState(false);
    const [addingToBlacklist, setAddingToBlacklist] = React.useState(false);
    const [lastUpdatedAt, setLastUpdatedAt] = React.useState<Date | null>(null);
    const [error, setError] = React.useState<Error | null>(null);

    const requestRef = React.useRef<{abort?: () => void} | null>(null);
    const eventRequestRef = React.useRef<{abort?: () => void} | null>(null);
    const blacklistRequestRef = React.useRef<{abort?: () => void} | null>(null);
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
        if (eventRequestRef.current?.abort) {
            eventRequestRef.current.abort();
            eventRequestRef.current = null;
        }
        if (blacklistRequestRef.current?.abort) {
            blacklistRequestRef.current.abort();
            blacklistRequestRef.current = null;
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

    const loadProjectEventLogs = React.useCallback(async () => {
        if (eventRequestRef.current) {
            return;
        }

        try {
            const req = services.athenaApplication.listProjectEventLogs(contract);
            eventRequestRef.current = req;
            const items = await req;
            if (isMountedRef.current) {
                setEventLogs(items || []);
            }
        } catch (err) {
            if (isMountedRef.current) {
                setError(err as Error);
            }
        } finally {
            eventRequestRef.current = null;
        }
    }, [contract]);

    const loadBlacklistStatus = React.useCallback(async () => {
        if (blacklistRequestRef.current) {
            return;
        }

        if (isMountedRef.current) {
            setIsBlacklistChecking(true);
        }

        try {
            const req = services.athenaApplication.listBytecodeBlacklistContracts();
            blacklistRequestRef.current = req;
            const items = await req;
            const normalizedContract = contract.trim().toLowerCase();
            const exists = (items || []).some(item => (item.contract || '').trim().toLowerCase() === normalizedContract);
            if (isMountedRef.current) {
                setIsBlacklisted(exists);
            }
        } catch (err) {
            if (isMountedRef.current) {
                setError(err as Error);
            }
        } finally {
            if (isMountedRef.current) {
                setIsBlacklistChecking(false);
            }
            blacklistRequestRef.current = null;
        }
    }, [contract]);

    React.useEffect(() => {
        isMountedRef.current = true;
        setIsBlacklisted(false);
        setIsBlacklistChecking(true);
        loadProject();
        loadProjectEventLogs();
        loadBlacklistStatus();
        intervalRef.current = window.setInterval(() => {
            loadProject();
            loadProjectEventLogs();
        }, AUTO_REFRESH_INTERVAL_MS);

        return () => {
            isMountedRef.current = false;
            cleanupRequests();
        };
    }, [cleanupRequests, loadBlacklistStatus, loadProject, loadProjectEventLogs]);

    const breadcrumbs = [{title: 'Projects', path: '/projects'}, {title: contract}];
    const isArchived = project?.meta?.isArchived ?? false;

    const handleArchiveStateChange = React.useCallback(async () => {
        if (changingArchiveState) {
            return;
        }
        setChangingArchiveState(true);
        try {
            if (isArchived) {
                await services.athenaApplication.unarchiveProject(contract);
            } else {
                await services.athenaApplication.archiveProject(contract);
            }
            await loadProject();
        } catch (err) {
            if (isMountedRef.current) {
                setError(err as Error);
            }
        } finally {
            if (isMountedRef.current) {
                setChangingArchiveState(false);
            }
        }
    }, [changingArchiveState, contract, isArchived, loadProject]);

    const handleAddToBlacklist = React.useCallback(async () => {
        if (addingToBlacklist || isBlacklistChecking) {
            return;
        }

        if (isBlacklisted) {
            ctx.notifications.show({
                content: '该合约已在 BIN 黑名单中',
                type: NotificationType.Warning
            });
            return;
        }

        await ctx.popup.prompt(
            '加入 BIN 黑名单',
            api => (
                <div>
                    <div className='argo-form-row'>
                        <FormField formApi={api} label='备注' field='note' component={Text} />
                    </div>
                </div>
            ),
            {
                validate: vals => {
                    const note = String(vals.note || '').trim();
                    return {
                        note: !note && '备注不能为空'
                    };
                },
                submit: async (vals, _, close) => {
                    const note = String(vals.note || '').trim();
                    if (!note) {
                        return;
                    }

                    setAddingToBlacklist(true);
                    try {
                        await services.athenaApplication.addBytecodeBlacklistContract(contract, note);
                        if (isMountedRef.current) {
                            setIsBlacklisted(true);
                            setError(null);
                        }
                        close();
                        ctx.notifications.show({
                            content: '已加入 BIN 黑名单',
                            type: NotificationType.Success
                        });
                    } catch (err) {
                        ctx.notifications.show({
                            content: <ErrorNotification title='加入 BIN 黑名单失败' e={err} />,
                            type: NotificationType.Error
                        });
                    } finally {
                        if (isMountedRef.current) {
                            setAddingToBlacklist(false);
                        }
                    }
                }
            }
        );
    }, [addingToBlacklist, contract, ctx, isBlacklistChecking, isBlacklisted]);

    const blacklistButtonText = isBlacklistChecking ? 'Checking...' : addingToBlacklist ? 'Adding...' : isBlacklisted ? '已加入BIN黑名单' : '加入BIN黑名单';
    const blacklistButtonDisabled = isBlacklistChecking || addingToBlacklist || isBlacklisted;

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
                        <div className='project-details__header'>
                            <div className='project-details__last-updated'>Last updated: {lastUpdatedAt ? lastUpdatedAt.toLocaleTimeString() : 'Never'}</div>
                            <div className='project-details__actions'>
                                <button type='button' className='argo-button argo-button--base-o' disabled={blacklistButtonDisabled} onClick={handleAddToBlacklist}>
                                    {blacklistButtonText}
                                </button>
                                <button type='button' className='argo-button argo-button--base' disabled={changingArchiveState} onClick={handleArchiveStateChange}>
                                    {changingArchiveState ? (isArchived ? 'Unarchiving...' : 'Archiving...') : isArchived ? 'Unarchive Project' : 'Archive Project'}
                                </button>
                            </div>
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
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Status</span>
                                    <span className='project-details__field-value'>
                                        <span className={`project-details__badge project-details__badge--${isArchived ? 'negative' : 'positive'}`}>
                                            {isArchived ? 'Archived' : 'Active'}
                                        </span>
                                    </span>
                                </div>
                            </div>
                        </div>

                        <div className='white-box project-details__box'>
                            <div className='project-details__section-title'>Event Log</div>
                            {eventLogs.length === 0 ? (
                                <div className='project-details__field-value'>No events available</div>
                            ) : (
                                <div className='project-details__grid'>
                                    {eventLogs.map(item => (
                                        <div key={`${item.id || 0}-${item.occurredAt || ''}`} className='project-details__field' style={{gridColumn: '1 / -1'}}>
                                            <span className='project-details__field-label'>
                                                {renderEventType(item.eventType)} • {renderValue(item.occurredAt)}
                                            </span>
                                            <span className='project-details__field-value'>{renderValue(item.message)}</span>
                                        </div>
                                    ))}
                                </div>
                            )}
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

                        <div className='white-box project-details__box'>
                            <div className='project-details__section-title'>Creator State</div>
                            <div className='project-details__grid'>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Token Balance</span>
                                    <span className='project-details__field-value'>{renderValue(project.chainState?.creatorState?.tokenBalance)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>WETH Balance</span>
                                    <span className='project-details__field-value'>{renderValue(project.chainState?.creatorState?.wethBalance)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>USDT Balance</span>
                                    <span className='project-details__field-value'>{renderValue(project.chainState?.creatorState?.usdtBalance)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Native Balance</span>
                                    <span className='project-details__field-value'>{renderValue(project.chainState?.creatorState?.nativeBalance)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Total Asset (USDT)</span>
                                    <span className='project-details__field-value'>{renderValue(project.chainState?.creatorState?.usdtValue)}</span>
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
