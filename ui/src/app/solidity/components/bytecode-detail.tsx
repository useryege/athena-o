import {MockupList, Page} from 'argo-ui';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';

import {services} from '../../shared/services';
import {BytecodeDeployment, BytecodeDetail as BytecodeDetailModel} from '../../shared/services/athena-solidity-service';

const PAGE_SIZE = 20;

const renderValue = (value: string | number | undefined) => (value === undefined || value === '' ? '-' : value);

const renderTime = (value?: string) => (value ? new Date(value).toLocaleString() : '-');

interface BytecodeDetailRouteParams {
    codeHash: string;
}

export const BytecodeDetail = (props: RouteComponentProps<BytecodeDetailRouteParams>) => {
    const codeHash = props.match.params.codeHash;
    const [detail, setDetail] = React.useState<BytecodeDetailModel | null>(null);
    const [deployments, setDeployments] = React.useState<BytecodeDeployment[]>([]);
    const [total, setTotal] = React.useState(0);
    const [page, setPage] = React.useState(1);
    const [chainIDInput, setChainIDInput] = React.useState('');
    const [contractInput, setContractInput] = React.useState('');
    const [chainIDFilter, setChainIDFilter] = React.useState('');
    const [contractFilter, setContractFilter] = React.useState('');
    const [loadingDetail, setLoadingDetail] = React.useState(true);
    const [loadingDeployments, setLoadingDeployments] = React.useState(true);
    const [error, setError] = React.useState<Error | null>(null);
    const detailRequestRef = React.useRef<{abort?: () => void} | null>(null);
    const deploymentRequestRef = React.useRef<{abort?: () => void} | null>(null);

    const loadDetail = React.useCallback(async () => {
        if (detailRequestRef.current?.abort) {
            detailRequestRef.current.abort();
        }
        setLoadingDetail(true);
        const req = services.athenaSolidity.getBytecode(codeHash);
        detailRequestRef.current = req;
        try {
            const data = await req;
            if (detailRequestRef.current === req) {
                setDetail(data);
                setError(null);
            }
        } catch (err) {
            if (detailRequestRef.current === req) {
                setError(err as Error);
            }
        } finally {
            if (detailRequestRef.current === req) {
                detailRequestRef.current = null;
                setLoadingDetail(false);
            }
        }
    }, [codeHash]);

    const loadDeployments = React.useCallback(
        async (targetPage: number, nextChainID = chainIDFilter, nextContract = contractFilter) => {
            if (deploymentRequestRef.current?.abort) {
                deploymentRequestRef.current.abort();
            }
            setLoadingDeployments(true);
            const parsedChainID = nextChainID.trim() ? Number(nextChainID.trim()) : undefined;
            const req = services.athenaSolidity.listBytecodeDeployments(codeHash, {
                page: targetPage,
                pageSize: PAGE_SIZE,
                chainID: Number.isFinite(parsedChainID) ? parsedChainID : undefined,
                contract: nextContract.trim() || undefined
            });
            deploymentRequestRef.current = req;
            try {
                const data = await req;
                if (deploymentRequestRef.current === req) {
                    setDeployments(data.items);
                    setTotal(data.total);
                    setPage(data.page);
                    setError(null);
                }
            } catch (err) {
                if (deploymentRequestRef.current === req) {
                    setError(err as Error);
                }
            } finally {
                if (deploymentRequestRef.current === req) {
                    deploymentRequestRef.current = null;
                    setLoadingDeployments(false);
                }
            }
        },
        [chainIDFilter, codeHash, contractFilter]
    );

    React.useEffect(() => {
        loadDetail();
        loadDeployments(1);
        return () => {
            if (detailRequestRef.current?.abort) {
                detailRequestRef.current.abort();
            }
            if (deploymentRequestRef.current?.abort) {
                deploymentRequestRef.current.abort();
            }
        };
    }, [loadDeployments, loadDetail]);

    const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

    const applyDeploymentFilter = (event: React.FormEvent) => {
        event.preventDefault();
        const nextChainID = chainIDInput.trim();
        const nextContract = contractInput.trim();
        setChainIDFilter(nextChainID);
        setContractFilter(nextContract);
        setPage(1);
        loadDeployments(1, nextChainID, nextContract);
    };

    return (
        <Page
            title='Bytecode Detail'
            toolbar={{breadcrumbs: [{title: 'Solidity', path: '/solidity/bytecodes'}, {title: 'Bytecode', path: '/solidity/bytecodes'}, {title: codeHash}]}}>
            <div className='solidity-bytecode'>
                {error && (
                    <div className='solidity-bytecode__error'>
                        <i className='fa fa-exclamation-triangle' /> Failed to load bytecode: {error.message}
                    </div>
                )}
                {loadingDetail && !detail ? (
                    <MockupList height={50} marginTop={30} />
                ) : (
                    <div className='argo-container'>
                        <div className='white-box solidity-bytecode__box'>
                            <div className='solidity-bytecode__detail-grid'>
                                <div>
                                    <span className='solidity-bytecode__label'>Code Hash</span>
                                    <span className='mono'>{renderValue(detail?.codeHash)}</span>
                                </div>
                                <div>
                                    <span className='solidity-bytecode__label'>Deployments</span>
                                    <span>{renderValue(detail?.deploymentCount)}</span>
                                </div>
                                <div>
                                    <span className='solidity-bytecode__label'>Runtime Size</span>
                                    <span>{renderValue(detail?.runtimeBytecodeSize)}</span>
                                </div>
                                <div>
                                    <span className='solidity-bytecode__label'>Source</span>
                                    <span className={`solidity-bytecode__badge ${detail?.isOpenSource ? 'solidity-bytecode__badge--success' : ''}`}>
                                        {detail?.isOpenSource ? 'Open' : 'Closed'}
                                    </span>
                                </div>
                                <div>
                                    <span className='solidity-bytecode__label'>Blacklist</span>
                                    <span className={`solidity-bytecode__badge ${detail?.isBytecodeBlacklisted ? 'solidity-bytecode__badge--danger' : ''}`}>
                                        {detail?.isBytecodeBlacklisted ? 'Listed' : '-'}
                                    </span>
                                </div>
                                <div>
                                    <span className='solidity-bytecode__label'>Updated</span>
                                    <span>{renderTime(detail?.updatedAt)}</span>
                                </div>
                            </div>
                            <div className='solidity-bytecode__section'>
                                <h3>Runtime Bytecode</h3>
                                <pre className='solidity-bytecode__code'>{detail?.runtimeBytecode || '-'}</pre>
                            </div>
                            <div className='solidity-bytecode__section'>
                                <h3>Source Quality</h3>
                                <div className='solidity-bytecode__detail-grid'>
                                    <div>
                                        <span className='solidity-bytecode__label'>Source Hash</span>
                                        <span className='mono'>{renderValue(detail?.sourceCodeHash)}</span>
                                    </div>
                                    <div>
                                        <span className='solidity-bytecode__label'>Source Origin</span>
                                        <span>{renderValue(detail?.sourceCodeOrigin)}</span>
                                    </div>
                                    <div>
                                        <span className='solidity-bytecode__label'>Report Origin</span>
                                        <span>{renderValue(detail?.sourceQualityReportOrigin)}</span>
                                    </div>
                                    <div>
                                        <span className='solidity-bytecode__label'>Report Updated</span>
                                        <span>{renderTime(detail?.sourceQualityReportFetchedAt)}</span>
                                    </div>
                                    <div>
                                        <span className='solidity-bytecode__label'>Prompt Version</span>
                                        <span>{renderValue(detail?.sourceQualityPromptVersion)}</span>
                                    </div>
                                </div>
                                <pre className='solidity-bytecode__code'>{detail?.sourceQualityReport || '-'}</pre>
                            </div>
                        </div>

                        <div className='white-box solidity-bytecode__box'>
                            <div className='solidity-bytecode__controls'>
                                <form className='solidity-bytecode__filter-form' onSubmit={applyDeploymentFilter}>
                                    <input className='argo-field' placeholder='Chain ID' value={chainIDInput} onChange={e => setChainIDInput(e.target.value)} />
                                    <input className='argo-field' placeholder='Contract address (0x...)' value={contractInput} onChange={e => setContractInput(e.target.value)} />
                                    <button type='submit' className='argo-button argo-button--base'>
                                        Search
                                    </button>
                                </form>
                                <div className='solidity-bytecode__pager'>
                                    <button
                                        type='button'
                                        className='argo-button argo-button--base-o'
                                        disabled={page <= 1 || loadingDeployments}
                                        onClick={() => loadDeployments(page - 1)}>
                                        Previous
                                    </button>
                                    <span>
                                        Page {page} / {totalPages}
                                    </span>
                                    <button
                                        type='button'
                                        className='argo-button argo-button--base-o'
                                        disabled={page >= totalPages || loadingDeployments}
                                        onClick={() => loadDeployments(page + 1)}>
                                        Next
                                    </button>
                                </div>
                            </div>
                            <div className='argo-table-list solidity-bytecode__table'>
                                <div className='argo-table-list__head'>
                                    <div className='solidity-bytecode__deployment-row'>
                                        <div>Chain</div>
                                        <div>Contract</div>
                                        <div>First Seen</div>
                                        <div>Updated</div>
                                    </div>
                                </div>
                                {deployments.length === 0 ? (
                                    <div className='argo-table-list__row'>
                                        <div className='columns small-12 text-center'>No contract deployments found</div>
                                    </div>
                                ) : (
                                    deployments.map(item => (
                                        <div className='argo-table-list__row' key={`${item.chainID}-${item.contract}`}>
                                            <div className='solidity-bytecode__deployment-row'>
                                                <div>{renderValue(item.chainID)}</div>
                                                <div className='mono'>{renderValue(item.contract)}</div>
                                                <div>{renderTime(item.firstSeenAt)}</div>
                                                <div>{renderTime(item.updatedAt)}</div>
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
