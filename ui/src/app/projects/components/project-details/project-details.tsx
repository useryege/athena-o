import {ErrorNotification, FormField, MockupList, NotificationType, Page} from 'argo-ui';
import * as React from 'react';
import {Text} from 'react-form';
import {RouteComponentProps} from 'react-router';
import {Context} from '../../../shared/context';
import {services} from '../../../shared/services';
import {
    AveDetail,
    PairV2State,
    ProjectAveState,
    ProjectEventLog,
    ProjectMeta,
    ProjectOptions,
    ProjectView
} from '../../../shared/services/athena-application-service';
import {ContractSourceInfo} from '../../../shared/services/athena-solidity-service';
import {formatUsdtValue} from '../pair-metrics-cell/pair-metrics-cell';
import {GenesisWalletRankList} from './genesis-wallet-rank-list';

require('./project-details.scss');

const PROJECT_AUTO_REFRESH_INTERVAL_MS = 1000;
const EVENT_LOG_AUTO_REFRESH_INTERVAL_MS = 3000;

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

const renderOrigin = (value?: string) => {
    switch (value) {
        case 'third_party_api':
            return '第三方 API';
        case 'reuse':
            return '复用';
        default:
            return '-';
    }
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

const parseTimelineTime = (value?: string) => {
    if (!value) {
        return null;
    }
    const time = Date.parse(value);
    return isNaN(time) ? null : time;
};

const formatTimelineDuration = (milliseconds: number) => {
    const sign = milliseconds < 0 ? '-' : '+';
    let remainingSeconds = Math.floor(Math.abs(milliseconds) / 1000);
    const days = Math.floor(remainingSeconds / 86400);
    remainingSeconds %= 86400;
    const hours = Math.floor(remainingSeconds / 3600);
    remainingSeconds %= 3600;
    const minutes = Math.floor(remainingSeconds / 60);
    const seconds = remainingSeconds % 60;
    const parts: string[] = [];

    if (days > 0) {
        parts.push(`${days}d`);
    }
    if (hours > 0 || parts.length > 0) {
        parts.push(`${hours}h`);
    }
    if (minutes > 0 || parts.length > 0) {
        parts.push(`${minutes}m`);
    }
    if (parts.length === 0 || seconds > 0) {
        parts.push(`${seconds}s`);
    }

    return `${sign}${parts.join(' ')}`;
};

const renderFetchTimeline = (meta?: ProjectMeta, aveDetail?: AveDetail, sourceInfo?: ContractSourceInfo | null) => {
    const baseTime = parseTimelineTime(meta?.fetchAt);
    const items = [
        {label: 'Project Discovered', value: meta?.fetchAt},
        {label: 'Source Code Fetched', value: sourceInfo?.sourceCodeFetchedAt},
        {label: 'Ave Detail Fetched', value: aveDetail?.fetchedAt},
        {label: 'Source Quality Report Fetched', value: sourceInfo?.sourceQualityReportFetchedAt},
        {label: 'Genesis Wallets Fetched', value: meta?.genesisWalletsFetchedAt},
        {label: 'Creator Historical Projects Fetched', value: meta?.creatorHistoricalProjectsFetchedAt}
    ].map((item, index) => ({...item, index, time: parseTimelineTime(item.value)}));
    const sortedItems = items
        .filter(item => item.time !== null)
        .sort((left, right) => (left.time as number) - (right.time as number))
        .concat(items.filter(item => item.time === null));

    return (
        <div className='white-box project-details__box'>
            <div className='project-details__section-title'>Fetch Timeline</div>
            <div className='project-details__timeline'>
                {sortedItems.map(item => {
                    const isCompleted = item.time !== null;
                    const duration = isCompleted && baseTime !== null ? formatTimelineDuration((item.time as number) - baseTime) : '';
                    return (
                        <div key={`${item.label}-${item.index}`} className={`project-details__timeline-item ${isCompleted ? 'project-details__timeline-item--done' : ''}`}>
                            <div className='project-details__timeline-marker' />
                            <div className='project-details__timeline-content'>
                                <div className='project-details__timeline-main'>
                                    <span className='project-details__timeline-label'>{item.label}</span>
                                    <span className={`project-details__timeline-status ${isCompleted ? 'project-details__timeline-status--done' : ''}`}>
                                        {isCompleted ? 'Done' : 'Pending'}
                                    </span>
                                </div>
                                <div className='project-details__timeline-meta'>
                                    <span>{isCompleted ? renderValue(item.value) : 'Pending'}</span>
                                    {duration && <span className='project-details__timeline-duration'>{duration}</span>}
                                </div>
                            </div>
                        </div>
                    );
                })}
            </div>
        </div>
    );
};

const renderQualityReportMarkdown = (markdown: string) => {
    const lines = markdown.split(/\r?\n/);
    const nodes: React.ReactNode[] = [];
    let index = 0;

    while (index < lines.length) {
        const line = lines[index];
        const trimmed = line.trim();
        if (!trimmed) {
            index++;
            continue;
        }

        if (trimmed.startsWith('```')) {
            const codeLines: string[] = [];
            index++;
            while (index < lines.length && !lines[index].trim().startsWith('```')) {
                codeLines.push(lines[index]);
                index++;
            }
            if (index < lines.length) {
                index++;
            }
            nodes.push(
                <pre key={`code-${index}`} className='project-details__quality-code'>
                    <code>{codeLines.join('\n')}</code>
                </pre>
            );
            continue;
        }

        const heading = trimmed.match(/^(#{1,4})\s+(.+)$/);
        if (heading) {
            const level = heading[1].length;
            nodes.push(
                <div key={`heading-${index}`} className={`project-details__quality-heading project-details__quality-heading--${level}`}>
                    {heading[2]}
                </div>
            );
            index++;
            continue;
        }

        if (/^[-*]\s+/.test(trimmed)) {
            const items: string[] = [];
            while (index < lines.length && /^[-*]\s+/.test(lines[index].trim())) {
                items.push(lines[index].trim().replace(/^[-*]\s+/, ''));
                index++;
            }
            nodes.push(
                <ul key={`list-${index}`} className='project-details__quality-list'>
                    {items.map((item, itemIndex) => (
                        <li key={`${item}-${itemIndex}`}>{item}</li>
                    ))}
                </ul>
            );
            continue;
        }

        nodes.push(
            <p key={`paragraph-${index}`} className='project-details__quality-paragraph'>
                {trimmed}
            </p>
        );
        index++;
    }

    return nodes;
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

type ProjectComponentKey = 'base' | 'chainState' | 'simulation' | 'ave' | 'genesisWallets' | 'creatorHistory';
type ComponentErrors = Partial<Record<ProjectComponentKey, Error>>;

const projectInitial = (meta?: ProjectMeta) => (meta?.token?.symbol || meta?.token?.name || '?').trim().slice(0, 1).toUpperCase() || '?';

const renderComponentError = (error?: Error) =>
    error ? (
        <div className='project-details__section-error'>
            <i className='fa fa-exclamation-triangle' /> {error.message}
        </div>
    ) : null;

export const ProjectDetails = (props: RouteComponentProps<RouteParams>) => {
    const ctx = React.useContext(Context);
    const contract = props.match.params.contract;
    const [project, setProject] = React.useState<ProjectView | null>(null);
    const [aveState, setAveState] = React.useState<ProjectAveState | null>(null);
    const [sourceInfo, setSourceInfo] = React.useState<ContractSourceInfo | null>(null);
    const [eventLogs, setEventLogs] = React.useState<ProjectEventLog[]>([]);
    const [loading, setLoading] = React.useState(true);
    const [isBlacklistChecking, setIsBlacklistChecking] = React.useState(true);
    const [isBlacklisted, setIsBlacklisted] = React.useState(false);
    const [addingToBlacklist, setAddingToBlacklist] = React.useState(false);
    const [lastUpdatedAt, setLastUpdatedAt] = React.useState<Date | null>(null);
    const [projectOptions, setProjectOptions] = React.useState<ProjectOptions | null>(null);
    const [error, setError] = React.useState<Error | null>(null);
    const [componentErrors, setComponentErrors] = React.useState<ComponentErrors>({});
    const [logoFailed, setLogoFailed] = React.useState(false);
    const [refreshingAve, setRefreshingAve] = React.useState(false);

    const requestRef = React.useRef<{abort?: () => void} | null>(null);
    const eventRequestRef = React.useRef<{abort?: () => void} | null>(null);
    const aveRefreshRequestRef = React.useRef<{abort?: () => void} | null>(null);
    const sourceRequestRef = React.useRef<{abort?: () => void} | null>(null);
    const optionsRequestRef = React.useRef<{abort?: () => void} | null>(null);
    const projectIntervalRef = React.useRef<number | undefined>(undefined);
    const eventIntervalRef = React.useRef<number | undefined>(undefined);
    const isMountedRef = React.useRef(false);

    const cleanupRequests = React.useCallback(() => {
        if (projectIntervalRef.current !== undefined) {
            window.clearInterval(projectIntervalRef.current);
            projectIntervalRef.current = undefined;
        }
        if (eventIntervalRef.current !== undefined) {
            window.clearInterval(eventIntervalRef.current);
            eventIntervalRef.current = undefined;
        }
        if (requestRef.current?.abort) {
            requestRef.current.abort();
            requestRef.current = null;
        }
        if (eventRequestRef.current?.abort) {
            eventRequestRef.current.abort();
            eventRequestRef.current = null;
        }
        if (aveRefreshRequestRef.current?.abort) {
            aveRefreshRequestRef.current.abort();
            aveRefreshRequestRef.current = null;
        }
        if (sourceRequestRef.current?.abort) {
            sourceRequestRef.current.abort();
            sourceRequestRef.current = null;
        }
        if (optionsRequestRef.current?.abort) {
            optionsRequestRef.current.abort();
            optionsRequestRef.current = null;
        }
    }, []);

    const mergeProjectAveState = React.useCallback((state?: ProjectAveState) => {
        setAveState(state || null);
        setProject(current => ({
            meta: current?.meta || {},
            aveDetail: state?.detail
        }));
    }, []);

    const loadProject = React.useCallback(async () => {
        if (requestRef.current) {
            return;
        }

        if (isMountedRef.current) {
            setComponentErrors({});
        }

        const projectReq = services.athenaApplication.getProject(contract);
        const aveReq = services.athenaApplication.getProjectAveState(contract);
        const requests = [projectReq, aveReq];
        requestRef.current = {
            abort: () => requests.forEach(req => req.abort?.())
        };

        let successCount = 0;
        const markSuccess = () => {
            successCount++;
            if (isMountedRef.current) {
                setLastUpdatedAt(new Date());
            }
        };
        const markError = (key: ProjectComponentKey, err: unknown) => {
            if (isMountedRef.current) {
                const nextError = err as Error;
                if (key === 'base') {
                    setComponentErrors(current => ({
                        ...current,
                        base: nextError,
                        chainState: nextError,
                        simulation: nextError,
                        genesisWallets: nextError,
                        creatorHistory: nextError
                    }));
                    return;
                }
                setComponentErrors(current => ({...current, [key]: nextError}));
            }
        };
        const runPart = async <T,>(key: ProjectComponentKey, req: Promise<T>, apply: (value: T) => void) => {
            try {
                const value = await req;
                if (isMountedRef.current) {
                    apply(value);
                    markSuccess();
                }
            } catch (err) {
                markError(key, err);
            }
        };

        await Promise.all([
            runPart<ProjectView>('base', projectReq, item => {
                setProject(item || {meta: {}});
                setComponentErrors(current => {
                    const next = {...current};
                    delete next.base;
                    delete next.chainState;
                    delete next.simulation;
                    delete next.genesisWallets;
                    delete next.creatorHistory;
                    return next;
                });
            }),
            runPart<ProjectAveState | undefined>('ave', aveReq, state => {
                mergeProjectAveState(state);
            })
        ]);

        if (isMountedRef.current) {
            setLoading(false);
            setError(successCount === 0 ? new Error('All project detail sections failed to load') : null);
        }
        requestRef.current = null;
    }, [contract, mergeProjectAveState]);

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

    const loadContractSourceInfo = React.useCallback(async () => {
        if (sourceRequestRef.current) {
            return;
        }

        if (isMountedRef.current) {
            setIsBlacklistChecking(true);
        }

        try {
            const req = services.athenaSolidity.getContractSourceInfo(contract);
            sourceRequestRef.current = req;
            const info = await req;
            if (isMountedRef.current) {
                setSourceInfo(info);
                setIsBlacklisted(!!info?.isBytecodeBlacklisted);
            }
        } catch (err) {
            if (isMountedRef.current) {
                setError(err as Error);
            }
        } finally {
            if (isMountedRef.current) {
                setIsBlacklistChecking(false);
            }
            sourceRequestRef.current = null;
        }
    }, [contract]);

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

    React.useEffect(() => {
        isMountedRef.current = true;
        setIsBlacklisted(false);
        setIsBlacklistChecking(true);
        setSourceInfo(null);
        setAveState(null);
        setComponentErrors({});
        loadProject();
        loadProjectEventLogs();
        loadContractSourceInfo();
        loadProjectOptions();
        projectIntervalRef.current = window.setInterval(() => {
            loadProject();
        }, PROJECT_AUTO_REFRESH_INTERVAL_MS);
        eventIntervalRef.current = window.setInterval(() => {
            loadProjectEventLogs();
        }, EVENT_LOG_AUTO_REFRESH_INTERVAL_MS);

        return () => {
            isMountedRef.current = false;
            cleanupRequests();
        };
    }, [cleanupRequests, loadContractSourceInfo, loadProject, loadProjectEventLogs, loadProjectOptions]);

    const breadcrumbs = [{title: 'Projects', path: `/projects${props.location.search || ''}`}, {title: contract}];
    const sourceCode = sourceInfo?.sourceCode || '';
    const isOpenSource = sourceInfo?.isOpenSource ?? sourceCode.trim().length > 0;
    const sourceQualityReport = sourceInfo?.sourceQualityReport || '';
    const aveLogo = project?.aveDetail?.token?.logoUrl;
    const showLogo = !!aveLogo && !logoFailed;

    React.useEffect(() => {
        setLogoFailed(false);
    }, [aveLogo]);

    const handleAddToBlacklist = React.useCallback(async () => {
        if (addingToBlacklist || isBlacklistChecking) {
            return;
        }

        if (isBlacklisted) {
            ctx.notifications.show({
                content: 'This contract is already in the BIN blacklist',
                type: NotificationType.Warning
            });
            return;
        }

        await ctx.popup.prompt(
            'Add to BIN Blacklist',
            api => (
                <div>
                    <div className='argo-form-row'>
                        <FormField formApi={api} label='Note' field='note' component={Text} />
                    </div>
                </div>
            ),
            {
                validate: vals => {
                    const note = String(vals.note || '').trim();
                    return {
                        note: !note && 'Note is required'
                    };
                },
                submit: async (vals, _, close) => {
                    const note = String(vals.note || '').trim();
                    if (!note) {
                        return;
                    }

                    setAddingToBlacklist(true);
                    try {
                        await services.athenaSolidity.addBytecodeBlacklistEntry(contract, note);
                        if (isMountedRef.current) {
                            setIsBlacklisted(true);
                            setSourceInfo(current => (current ? {...current, isBytecodeBlacklisted: true} : current));
                            setError(null);
                        }
                        close();
                        ctx.notifications.show({
                            content: 'Added to BIN blacklist',
                            type: NotificationType.Success
                        });
                    } catch (err) {
                        ctx.notifications.show({
                            content: <ErrorNotification title='Failed to add to BIN blacklist' e={err} />,
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

    const blacklistButtonText = isBlacklistChecking ? 'Checking...' : addingToBlacklist ? 'Adding...' : isBlacklisted ? 'Already in BIN blacklist' : 'Add to BIN blacklist';
    const blacklistButtonDisabled = isBlacklistChecking || addingToBlacklist || isBlacklisted;

    const handleRefreshAve = React.useCallback(async () => {
        if (refreshingAve) {
            return;
        }
        setRefreshingAve(true);
        try {
            const req = services.athenaApplication.refreshProjectAveDetail(contract);
            aveRefreshRequestRef.current = req;
            const state = await req;
            if (isMountedRef.current) {
                mergeProjectAveState(state);
                setComponentErrors(current => {
                    const next = {...current};
                    delete next.ave;
                    return next;
                });
            }
            ctx.notifications.show({content: 'Ave refresh scheduled', type: NotificationType.Success});
        } catch (err) {
            if (isMountedRef.current) {
                setComponentErrors(current => ({...current, ave: err as Error}));
            }
            ctx.notifications.show({
                content: <ErrorNotification title='Failed to refresh Ave detail' e={err} />,
                type: NotificationType.Error
            });
        } finally {
            aveRefreshRequestRef.current = null;
            if (isMountedRef.current) {
                setRefreshingAve(false);
            }
        }
    }, [contract, ctx, mergeProjectAveState, refreshingAve]);

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
                            </div>
                        </div>

                        <div className='white-box project-details__box'>
                            <div className='project-details__section-title'>Meta</div>
                            {renderComponentError(componentErrors.base)}
                            <div className='project-details__identity'>
                                <div className='project-details__logo' aria-hidden='true'>
                                    {showLogo ? <img src={aveLogo} alt='' onError={() => setLogoFailed(true)} /> : <span>{projectInitial(project.meta)}</span>}
                                </div>
                                <div className='project-details__identity-main'>
                                    <div className='project-details__identity-title'>{renderValue(project.meta?.token?.name)}</div>
                                    <div className='project-details__identity-subtitle'>{renderValue(project.meta?.token?.symbol)}</div>
                                </div>
                            </div>
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
                                <div className='project-details__field' style={{gridColumn: '1 / -1'}}>
                                    <span className='project-details__field-label'>Creator Historical Projects</span>
                                    <div className='project-details__field-value'>
                                        {renderComponentError(componentErrors.creatorHistory)}
                                        {project.meta?.creatorHistoricalProjects && project.meta.creatorHistoricalProjects.length > 0
                                            ? project.meta.creatorHistoricalProjects.map((item, index) => <div key={`${item}-${index}`}>{item}</div>)
                                            : '-'}
                                    </div>
                                </div>
                            </div>
                        </div>

                        {renderFetchTimeline(project.meta, project.aveDetail, sourceInfo)}
                        <div className='white-box project-details__box'>
                            <div className='project-details__header'>
                                <div className='project-details__section-title'>Ave</div>
                                <div className='project-details__actions'>
                                    <button type='button' className='argo-button argo-button--base-o' disabled={refreshingAve} onClick={handleRefreshAve}>
                                        {refreshingAve ? 'Scheduling...' : 'Refresh Ave'}
                                    </button>
                                </div>
                            </div>
                            {renderComponentError(componentErrors.ave)}
                            <div className='project-details__grid'>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Status</span>
                                    <span className='project-details__field-value'>{renderValue(aveState?.status)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Detail Available</span>
                                    <span className='project-details__field-value'>{renderValue(aveState?.detailAvailable)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Stale</span>
                                    <span className='project-details__field-value'>{renderValue(aveState?.stale)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Last Attempt</span>
                                    <span className='project-details__field-value'>{renderValue(aveState?.lastAttemptAt)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Last Success</span>
                                    <span className='project-details__field-value'>{renderValue(aveState?.lastSuccessAt)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Next Run</span>
                                    <span className='project-details__field-value'>{renderValue(aveState?.nextRunAt)}</span>
                                </div>
                                {aveState?.lastError && (
                                    <div className='project-details__field' style={{gridColumn: '1 / -1'}}>
                                        <span className='project-details__field-label'>Last Error</span>
                                        <span className='project-details__field-value'>{aveState.lastError}</span>
                                    </div>
                                )}
                            </div>
                        </div>

                        <div className='white-box project-details__box'>
                            <div className='project-details__section-title'>Genesis Wallets</div>
                            {renderComponentError(componentErrors.genesisWallets)}
                            <GenesisWalletRankList
                                genesisWallets={project.meta?.genesisWallets}
                                genesisWalletAssetStates={project.meta?.genesisWalletAssetStates}
                                usdtDecimals={projectOptions?.usdtDecimals}
                            />
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
                            {renderComponentError(componentErrors.chainState)}
                            <div className='project-details__grid'>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Name</span>
                                    <span className='project-details__field-value'>{renderValue(project.meta?.token?.name)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Symbol</span>
                                    <span className='project-details__field-value'>{renderValue(project.meta?.token?.symbol)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Decimals</span>
                                    <span className='project-details__field-value'>{renderValue(project.meta?.token?.decimals)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Total Supply</span>
                                    <span className='project-details__field-value'>{renderValue(project.meta?.token?.totalSupply)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Valid ERC20</span>
                                    <span className='project-details__field-value'>
                                        {project.meta?.token?.isValidERC20 !== undefined ? (
                                            <span className={`project-details__badge project-details__badge--${project.meta.token.isValidERC20 ? 'positive' : 'negative'}`}>
                                                {project.meta.token.isValidERC20 ? 'Yes' : 'No'}
                                            </span>
                                        ) : (
                                            '-'
                                        )}
                                    </span>
                                </div>
                            </div>
                        </div>

                        <div className='white-box project-details__box'>
                            <div className='project-details__section-title'>Asset State</div>
                            {renderComponentError(componentErrors.chainState)}
                            <div className='project-details__grid'>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Token Balance</span>
                                    <span className='project-details__field-value'>{renderValue(project.meta?.assetState?.tokenBalance)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>WETH Balance</span>
                                    <span className='project-details__field-value'>{renderValue(project.meta?.assetState?.wethBalance)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>USDT Balance</span>
                                    <span className='project-details__field-value'>{renderValue(project.meta?.assetState?.usdtBalance)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Native Balance</span>
                                    <span className='project-details__field-value'>{renderValue(project.meta?.assetState?.nativeBalance)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Total Asset (USDT)</span>
                                    <span className='project-details__field-value'>{formatUsdtValue(project.meta?.assetState?.usdtValue, projectOptions?.usdtDecimals)}</span>
                                </div>
                            </div>
                        </div>

                        {isOpenSource && (
                            <div className='white-box project-details__box'>
                                <div className='project-details__section-title'>Source Code</div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Source</span>
                                    <span className='project-details__field-value'>{renderOrigin(sourceInfo?.sourceCodeOrigin)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Contract Source Code</span>
                                    <div className='project-details__code-block'>{sourceCode}</div>
                                </div>
                            </div>
                        )}

                        {isOpenSource && (
                            <div className='white-box project-details__box'>
                                <div className='project-details__section-title'>Quality Report</div>
                                {sourceInfo?.sourceQualityReportFetchedAt && (
                                    <div className='project-details__field'>
                                        <span className='project-details__field-label'>Fetched At</span>
                                        <span className='project-details__field-value'>{renderValue(sourceInfo.sourceQualityReportFetchedAt)}</span>
                                    </div>
                                )}
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Source</span>
                                    <span className='project-details__field-value'>{renderOrigin(sourceInfo?.sourceQualityReportOrigin)}</span>
                                </div>
                                {sourceQualityReport.trim() ? (
                                    <div className='project-details__quality-report'>{renderQualityReportMarkdown(sourceQualityReport)}</div>
                                ) : (
                                    <div className='project-details__field-value'>Quality report is pending</div>
                                )}
                            </div>
                        )}

                        {renderPairSection('WETH V2 Pool', project.meta?.wethPair)}
                        {renderPairSection('USDT V2 Pool', project.meta?.usdtPair)}

                        {(project.meta?.creatorResult || componentErrors.simulation) && (
                            <div className='white-box project-details__box'>
                                <div className='project-details__section-title'>Simulation</div>
                                {renderComponentError(componentErrors.simulation)}
                                {project.meta?.creatorResult && (
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
                                )}
                            </div>
                        )}
                    </div>
                ) : null}
            </div>
        </Page>
    );
};
