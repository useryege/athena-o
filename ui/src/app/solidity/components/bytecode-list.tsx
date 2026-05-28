import {MockupList, Page} from 'argo-ui';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';
import {Link} from 'react-router-dom';

import {services} from '../../shared/services';
import {BytecodeListItem} from '../../shared/services/athena-solidity-service';

const PAGE_SIZE = 20;

const renderValue = (value: string | number | undefined) => (value === undefined || value === '' ? '-' : value);

const renderTime = (value?: string) => (value ? new Date(value).toLocaleString() : '-');

export const BytecodeList = (_props: RouteComponentProps<any>) => {
    const [items, setItems] = React.useState<BytecodeListItem[]>([]);
    const [total, setTotal] = React.useState(0);
    const [page, setPage] = React.useState(1);
    const [codeHashInput, setCodeHashInput] = React.useState('');
    const [codeHashFilter, setCodeHashFilter] = React.useState('');
    const [loading, setLoading] = React.useState(true);
    const [refreshing, setRefreshing] = React.useState(false);
    const [error, setError] = React.useState<Error | null>(null);
    const requestRef = React.useRef<{abort?: () => void} | null>(null);

    const loadBytecodes = React.useCallback(
        async (targetPage: number, targetCodeHash = codeHashFilter) => {
            if (requestRef.current?.abort) {
                requestRef.current.abort();
            }
            setRefreshing(true);
            const req = services.athenaSolidity.listBytecodes({
                page: targetPage,
                pageSize: PAGE_SIZE,
                codeHash: targetCodeHash.trim() || undefined
            });
            requestRef.current = req;
            try {
                const data = await req;
                if (requestRef.current === req) {
                    setItems(data.items);
                    setTotal(data.total);
                    setPage(data.page);
                    setError(null);
                }
            } catch (err) {
                if (requestRef.current === req) {
                    setError(err as Error);
                }
            } finally {
                if (requestRef.current === req) {
                    requestRef.current = null;
                    setLoading(false);
                    setRefreshing(false);
                }
            }
        },
        [codeHashFilter]
    );

    React.useEffect(() => {
        loadBytecodes(1);
        return () => {
            if (requestRef.current?.abort) {
                requestRef.current.abort();
            }
        };
    }, [loadBytecodes]);

    const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));
    const canPrev = page > 1;
    const canNext = page < totalPages;

    const applyFilter = (event: React.FormEvent) => {
        event.preventDefault();
        const nextFilter = codeHashInput.trim();
        setCodeHashFilter(nextFilter);
        setPage(1);
        loadBytecodes(1, nextFilter);
    };

    return (
        <Page title='Bytecode' toolbar={{breadcrumbs: [{title: 'Solidity', path: '/solidity/bytecodes'}, {title: 'Bytecode'}]}}>
            <div className='solidity-bytecode'>
                {error && (
                    <div className='solidity-bytecode__error'>
                        <i className='fa fa-exclamation-triangle' /> Failed to load bytecodes: {error.message}
                    </div>
                )}
                {loading && items.length === 0 ? (
                    <MockupList height={50} marginTop={30} />
                ) : (
                    <div className='argo-container'>
                        <div className='white-box solidity-bytecode__box'>
                            <div className='solidity-bytecode__controls'>
                                <form className='solidity-bytecode__filter-form' onSubmit={applyFilter}>
                                    <input
                                        className='argo-field'
                                        placeholder='Code hash (0x...)'
                                        value={codeHashInput}
                                        onChange={e => setCodeHashInput(e.target.value)}
                                    />
                                    <button type='submit' className='argo-button argo-button--base'>
                                        Search
                                    </button>
                                </form>
                                <div className='solidity-bytecode__pager'>
                                    <button type='button' className='argo-button argo-button--base-o' disabled={!canPrev || refreshing} onClick={() => loadBytecodes(page - 1)}>
                                        Previous
                                    </button>
                                    <span>
                                        Page {page} / {totalPages}
                                    </span>
                                    <button type='button' className='argo-button argo-button--base-o' disabled={!canNext || refreshing} onClick={() => loadBytecodes(page + 1)}>
                                        Next
                                    </button>
                                </div>
                            </div>
                            <div className='argo-table-list solidity-bytecode__table'>
                                <div className='argo-table-list__head'>
                                    <div className='solidity-bytecode__row'>
                                        <div>Code Hash</div>
                                        <div>Deployments</div>
                                        <div>Size</div>
                                        <div>Source</div>
                                        <div>Blacklist</div>
                                        <div>Updated</div>
                                    </div>
                                </div>
                                {items.length === 0 ? (
                                    <div className='argo-table-list__row'>
                                        <div className='columns small-12 text-center'>No bytecodes found</div>
                                    </div>
                                ) : (
                                    items.map(item => (
                                        <div className='argo-table-list__row' key={item.codeHash}>
                                            <Link className='solidity-bytecode__row solidity-bytecode__row-link' to={`/solidity/bytecodes/${encodeURIComponent(item.codeHash || '')}`}>
                                                <div className='mono'>{renderValue(item.codeHash)}</div>
                                                <div>{renderValue(item.deploymentCount)}</div>
                                                <div>{renderValue(item.runtimeBytecodeSize)}</div>
                                                <div>
                                                    <span className={`solidity-bytecode__badge ${item.isOpenSource ? 'solidity-bytecode__badge--success' : ''}`}>
                                                        {item.isOpenSource ? 'Open' : 'Closed'}
                                                    </span>
                                                </div>
                                                <div>
                                                    <span className={`solidity-bytecode__badge ${item.isBytecodeBlacklisted ? 'solidity-bytecode__badge--danger' : ''}`}>
                                                        {item.isBytecodeBlacklisted ? 'Listed' : '-'}
                                                    </span>
                                                </div>
                                                <div>{renderTime(item.updatedAt)}</div>
                                            </Link>
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
