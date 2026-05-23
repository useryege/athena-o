import {ErrorNotification, FormField, MockupList, NotificationType, Page} from 'argo-ui';
import * as React from 'react';
import {Text} from 'react-form';
import {RouteComponentProps} from 'react-router';
import {Context} from '../../../shared/context';
import {services} from '../../../shared/services';
import {PairV2State, ProjectComment, ProjectEventLog, ProjectOptions, ProjectView} from '../../../shared/services/athena-application-service';
import {formatUsdtValue} from '../pair-metrics-cell/pair-metrics-cell';
import {GenesisWalletRankList} from './genesis-wallet-rank-list';

require('./project-details.scss');

const AUTO_REFRESH_INTERVAL_MS = 3000;
const COMMENT_PAGE_SIZE = 5;

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

export const ProjectDetails = (props: RouteComponentProps<RouteParams>) => {
    const ctx = React.useContext(Context);
    const contract = props.match.params.contract;
    const [project, setProject] = React.useState<ProjectView | null>(null);
    const [eventLogs, setEventLogs] = React.useState<ProjectEventLog[]>([]);
    const [comments, setComments] = React.useState<ProjectComment[]>([]);
    const [commentPage, setCommentPage] = React.useState(1);
    const [commentTotal, setCommentTotal] = React.useState(0);
    const [commentInput, setCommentInput] = React.useState('');
    const [commentsLoading, setCommentsLoading] = React.useState(false);
    const [commentsError, setCommentsError] = React.useState<Error | null>(null);
    const [submittingComment, setSubmittingComment] = React.useState(false);
    const [loading, setLoading] = React.useState(true);
    const [isBlacklistChecking, setIsBlacklistChecking] = React.useState(true);
    const [isBlacklisted, setIsBlacklisted] = React.useState(false);
    const [addingToBlacklist, setAddingToBlacklist] = React.useState(false);
    const [lastUpdatedAt, setLastUpdatedAt] = React.useState<Date | null>(null);
    const [projectOptions, setProjectOptions] = React.useState<ProjectOptions | null>(null);
    const [error, setError] = React.useState<Error | null>(null);

    const requestRef = React.useRef<{abort?: () => void} | null>(null);
    const eventRequestRef = React.useRef<{abort?: () => void} | null>(null);
    const commentsRequestRef = React.useRef<{abort?: () => void} | null>(null);
    const addCommentRequestRef = React.useRef<{abort?: () => void} | null>(null);
    const blacklistRequestRef = React.useRef<{abort?: () => void} | null>(null);
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
        if (eventRequestRef.current?.abort) {
            eventRequestRef.current.abort();
            eventRequestRef.current = null;
        }
        if (commentsRequestRef.current?.abort) {
            commentsRequestRef.current.abort();
            commentsRequestRef.current = null;
        }
        if (addCommentRequestRef.current?.abort) {
            addCommentRequestRef.current.abort();
            addCommentRequestRef.current = null;
        }
        if (blacklistRequestRef.current?.abort) {
            blacklistRequestRef.current.abort();
            blacklistRequestRef.current = null;
        }
        if (optionsRequestRef.current?.abort) {
            optionsRequestRef.current.abort();
            optionsRequestRef.current = null;
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

    const loadProjectComments = React.useCallback(
        async (targetPage = 1) => {
            if (commentsRequestRef.current?.abort) {
                commentsRequestRef.current.abort();
                commentsRequestRef.current = null;
            }
            if (isMountedRef.current) {
                setCommentsLoading(true);
            }
            try {
                const req = services.athenaApplication.listProjectComments(contract, targetPage, COMMENT_PAGE_SIZE);
                commentsRequestRef.current = req;
                const data = await req;
                if (isMountedRef.current) {
                    setComments(data.items || []);
                    setCommentPage(data.page || targetPage);
                    setCommentTotal(data.total || 0);
                    setCommentsError(null);
                }
            } catch (err) {
                if (isMountedRef.current) {
                    setCommentsError(err as Error);
                }
            } finally {
                if (isMountedRef.current) {
                    setCommentsLoading(false);
                }
                commentsRequestRef.current = null;
            }
        },
        [contract]
    );

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
        setCommentPage(1);
        setCommentTotal(0);
        setCommentInput('');
        setCommentsError(null);
        loadProject();
        loadProjectEventLogs();
        loadProjectComments(1);
        loadBlacklistStatus();
        loadProjectOptions();
        intervalRef.current = window.setInterval(() => {
            loadProject();
            loadProjectEventLogs();
        }, AUTO_REFRESH_INTERVAL_MS);

        return () => {
            isMountedRef.current = false;
            cleanupRequests();
        };
    }, [cleanupRequests, loadBlacklistStatus, loadProject, loadProjectComments, loadProjectEventLogs, loadProjectOptions]);

    const breadcrumbs = [{title: 'Projects', path: `/projects${props.location.search || ''}`}, {title: contract}];
    const sourceCode = project?.meta?.sourceCode || '';
    const isOpenSource = project?.meta?.isOpenSource ?? sourceCode.trim().length > 0;
    const sourceQualityReport = project?.meta?.sourceQualityReport || '';

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
                        await services.athenaApplication.addBytecodeBlacklistContract(contract, note);
                        if (isMountedRef.current) {
                            setIsBlacklisted(true);
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
    const commentTotalPages = Math.max(1, Math.ceil(commentTotal / COMMENT_PAGE_SIZE));

    const handleSubmitComment = React.useCallback(async () => {
        const content = commentInput.trim();
        if (!content) {
            ctx.notifications.show({content: 'Comment content is required', type: NotificationType.Warning});
            return;
        }
        if (submittingComment) {
            return;
        }
        setSubmittingComment(true);
        try {
            const req = services.athenaApplication.addProjectComment(contract, content);
            addCommentRequestRef.current = req;
            await req;
            if (isMountedRef.current) {
                setCommentInput('');
            }
            await loadProjectComments(1);
            if (isMountedRef.current) {
                setError(null);
            }
            ctx.notifications.show({content: 'Comment posted', type: NotificationType.Success});
        } catch (err) {
            ctx.notifications.show({
                content: <ErrorNotification title='Failed to add comment' e={err} />,
                type: NotificationType.Error
            });
        } finally {
            addCommentRequestRef.current = null;
            if (isMountedRef.current) {
                setSubmittingComment(false);
            }
        }
    }, [commentInput, contract, ctx, loadProjectComments, submittingComment]);

    const handleCommentPageChange = React.useCallback(
        (nextPage: number) => {
            if (nextPage < 1 || nextPage > commentTotalPages || commentsLoading) {
                return;
            }
            loadProjectComments(nextPage);
        },
        [commentTotalPages, commentsLoading, loadProjectComments]
    );

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
                                    <span className='project-details__field-label'>Creator Other Projects</span>
                                    <div className='project-details__field-value'>
                                        {project.meta?.creatorOtherProjectContracts && project.meta.creatorOtherProjectContracts.length > 0
                                            ? project.meta.creatorOtherProjectContracts.map((item, index) => <div key={`${item}-${index}`}>{item}</div>)
                                            : '-'}
                                    </div>
                                </div>
                            </div>
                        </div>

                        <div className='white-box project-details__box'>
                            <div className='project-details__section-title'>Genesis Wallets</div>
                            <GenesisWalletRankList
                                genesisWallets={project.meta?.genesisWallets}
                                genesisWalletAssetStates={project.chainState?.genesisWalletAssetStates}
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
                            <div className='project-details__section-title'>Comments</div>
                            <div className='project-details__comment-form'>
                                <textarea
                                    className='project-details__comment-input'
                                    value={commentInput}
                                    rows={3}
                                    maxLength={1000}
                                    placeholder='Write a comment...'
                                    onChange={e => setCommentInput(e.currentTarget.value)}
                                />
                                <div className='project-details__comment-form-actions'>
                                    <span className='project-details__comment-counter'>{commentInput.length}/1000</span>
                                    <button type='button' className='argo-button argo-button--base' disabled={submittingComment} onClick={handleSubmitComment}>
                                        {submittingComment ? 'Posting...' : 'Post Comment'}
                                    </button>
                                </div>
                            </div>

                            {commentsLoading ? (
                                <div className='project-details__field-value'>Loading comments...</div>
                            ) : commentsError ? (
                                <div className='project-details__comment-error'>Failed to load comments: {commentsError.message}</div>
                            ) : comments.length === 0 ? (
                                <div className='project-details__field-value'>No comments yet</div>
                            ) : (
                                <div className='project-details__comment-list'>
                                    {comments.map(item => (
                                        <div key={`${item.id || 0}-${item.createdAt || ''}`} className='project-details__comment-item'>
                                            <div className='project-details__comment-meta'>
                                                <span className='project-details__comment-user'>{renderValue(item.username)}</span>
                                                <span className='project-details__comment-time'>{renderValue(item.createdAt)}</span>
                                            </div>
                                            <div className='project-details__comment-content'>{renderValue(item.content)}</div>
                                        </div>
                                    ))}
                                </div>
                            )}

                            <div className='project-details__comment-pagination'>
                                <button
                                    type='button'
                                    className='argo-button argo-button--base-o'
                                    disabled={commentsLoading || commentPage <= 1}
                                    onClick={() => handleCommentPageChange(commentPage - 1)}>
                                    Prev
                                </button>
                                <span>
                                    Page {commentPage}/{commentTotalPages} • Total {commentTotal}
                                </span>
                                <button
                                    type='button'
                                    className='argo-button argo-button--base-o'
                                    disabled={commentsLoading || commentPage >= commentTotalPages}
                                    onClick={() => handleCommentPageChange(commentPage + 1)}>
                                    Next
                                </button>
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

                        <div className='white-box project-details__box'>
                            <div className='project-details__section-title'>Asset State</div>
                            <div className='project-details__grid'>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Token Balance</span>
                                    <span className='project-details__field-value'>{renderValue(project.chainState?.assetState?.tokenBalance)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>WETH Balance</span>
                                    <span className='project-details__field-value'>{renderValue(project.chainState?.assetState?.wethBalance)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>USDT Balance</span>
                                    <span className='project-details__field-value'>{renderValue(project.chainState?.assetState?.usdtBalance)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Native Balance</span>
                                    <span className='project-details__field-value'>{renderValue(project.chainState?.assetState?.nativeBalance)}</span>
                                </div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Total Asset (USDT)</span>
                                    <span className='project-details__field-value'>{formatUsdtValue(project.chainState?.assetState?.usdtValue, projectOptions?.usdtDecimals)}</span>
                                </div>
                            </div>
                        </div>

                        {isOpenSource && (
                            <div className='white-box project-details__box'>
                                <div className='project-details__section-title'>Source Code</div>
                                <div className='project-details__field'>
                                    <span className='project-details__field-label'>Contract Source Code</span>
                                    <div className='project-details__code-block'>{sourceCode}</div>
                                </div>
                            </div>
                        )}

                        {isOpenSource && (
                            <div className='white-box project-details__box'>
                                <div className='project-details__section-title'>Quality Report</div>
                                {project.meta?.sourceQualityReportedAt && (
                                    <div className='project-details__field'>
                                        <span className='project-details__field-label'>Reported At</span>
                                        <span className='project-details__field-value'>{renderValue(project.meta.sourceQualityReportedAt)}</span>
                                    </div>
                                )}
                                {sourceQualityReport.trim() ? (
                                    <div className='project-details__quality-report'>{renderQualityReportMarkdown(sourceQualityReport)}</div>
                                ) : (
                                    <div className='project-details__field-value'>Quality report is pending</div>
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
                    </div>
                ) : null}
            </div>
        </Page>
    );
};
