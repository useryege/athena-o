import {MockupList, Page} from 'argo-ui';
import * as React from 'react';

import {services} from '../../shared/services';
import {WalletChain, WalletDetail, WalletItem} from '../../shared/services/wallet-service';

require('./wallet-container.scss');

const PAGE_SIZE = 20;
const CHAINS: Array<{label: string; value: WalletChain}> = [
    {label: 'ETH', value: 'ETH'},
    {label: 'BSC', value: 'BSC'},
    {label: 'Base', value: 'BASE'},
    {label: 'Solana', value: 'SOLANA'}
];

type WalletAction = 'create' | 'privateKey' | 'mnemonic';

const ACTIONS: Array<{label: string; value: WalletAction; icon: string}> = [
    {label: 'Create', value: 'create', icon: 'fa fa-plus'},
    {label: 'Private Key', value: 'privateKey', icon: 'fa fa-key'},
    {label: 'Mnemonic', value: 'mnemonic', icon: 'fa fa-list'}
];

const isAbortedError = (err: unknown) =>
    String((err as any)?.message || '')
        .toLowerCase()
        .includes('abort');

const formatDate = (value?: string) => {
    if (!value) {
        return '-';
    }
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
};

const sourceLabel = (value?: string) => {
    switch (value) {
        case 'created':
            return 'Created';
        case 'private_key':
            return 'Private Key';
        case 'mnemonic':
            return 'Mnemonic';
        default:
            return value || '-';
    }
};

export const WalletContainer = () => {
    const [chain, setChain] = React.useState<WalletChain>('ETH');
    const [query, setQuery] = React.useState('');
    const [items, setItems] = React.useState<WalletItem[]>([]);
    const [total, setTotal] = React.useState(0);
    const [page, setPage] = React.useState(1);
    const [loading, setLoading] = React.useState(true);
    const [refreshing, setRefreshing] = React.useState(false);
    const [submitting, setSubmitting] = React.useState(false);
    const [revealing, setRevealing] = React.useState(false);
    const [savingAlias, setSavingAlias] = React.useState(false);
    const [error, setError] = React.useState<Error | null>(null);
    const [notice, setNotice] = React.useState('');
    const [action, setAction] = React.useState<WalletAction>('create');
    const [alias, setAlias] = React.useState('');
    const [privateKey, setPrivateKey] = React.useState('');
    const [mnemonic, setMnemonic] = React.useState('');
    const [selected, setSelected] = React.useState<WalletDetail | null>(null);
    const [aliasDraft, setAliasDraft] = React.useState('');
    const requestRef = React.useRef<{abort?: () => void} | null>(null);
    const mountedRef = React.useRef(false);

    const loadWallets = React.useCallback(
        async (targetPage = page) => {
            if (requestRef.current?.abort) {
                requestRef.current.abort();
            }
            if (mountedRef.current) {
                setRefreshing(true);
            }
            const req = services.wallet.listWallets({chain, query: query.trim(), page: targetPage, pageSize: PAGE_SIZE});
            requestRef.current = req;
            try {
                const result = await req;
                if (mountedRef.current && requestRef.current === req) {
                    setItems(result.items);
                    setTotal(result.total);
                    setPage(result.page || targetPage);
                    setError(null);
                }
            } catch (err) {
                if (mountedRef.current && requestRef.current === req && !isAbortedError(err)) {
                    setError(err as Error);
                }
            } finally {
                if (requestRef.current === req) {
                    requestRef.current = null;
                }
                if (mountedRef.current) {
                    setLoading(false);
                    setRefreshing(false);
                }
            }
        },
        [chain, query]
    );

    React.useEffect(() => {
        mountedRef.current = true;
        setLoading(true);
        loadWallets(1);
        return () => {
            mountedRef.current = false;
            if (requestRef.current?.abort) {
                requestRef.current.abort();
            }
        };
    }, [chain]);

    React.useEffect(() => {
        setPage(1);
        setSelected(null);
    }, [chain]);

    const totalPages = Math.max(Math.ceil(total / PAGE_SIZE), 1);
    const rangeStart = total === 0 ? 0 : (page - 1) * PAGE_SIZE + 1;
    const rangeEnd = Math.min(page * PAGE_SIZE, total);

    const selectWallet = async (item: WalletItem) => {
        setError(null);
        setNotice('');
        try {
            const detail = await services.wallet.getWallet(item.id, false);
            if (mountedRef.current) {
                setSelected(detail);
                setAliasDraft(detail.alias || '');
            }
        } catch (err) {
            if (mountedRef.current) {
                setError(err as Error);
            }
        }
    };

    const revealSecrets = async () => {
        if (!selected || revealing) {
            return;
        }
        setRevealing(true);
        setError(null);
        try {
            const detail = await services.wallet.getWallet(selected.id, true);
            if (mountedRef.current) {
                setSelected(detail);
                setAliasDraft(detail.alias || '');
            }
        } catch (err) {
            if (mountedRef.current) {
                setError(err as Error);
            }
        } finally {
            if (mountedRef.current) {
                setRevealing(false);
            }
        }
    };

    const saveAlias = async () => {
        if (!selected || savingAlias) {
            return;
        }
        setSavingAlias(true);
        setError(null);
        try {
            const item = await services.wallet.updateAlias(selected.id, aliasDraft);
            if (mountedRef.current) {
                setSelected({...selected, alias: item.alias || '', updatedAt: item.updatedAt});
                setNotice('Alias saved');
                await loadWallets(page);
            }
        } catch (err) {
            if (mountedRef.current) {
                setError(err as Error);
            }
        } finally {
            if (mountedRef.current) {
                setSavingAlias(false);
            }
        }
    };

    const submitWallet = async (event: React.FormEvent) => {
        event.preventDefault();
        if (submitting) {
            return;
        }
        setSubmitting(true);
        setError(null);
        setNotice('');
        try {
            let item: WalletDetail;
            if (action === 'create') {
                item = await services.wallet.createWallet(chain, alias);
            } else if (action === 'privateKey') {
                item = await services.wallet.importPrivateKey(chain, privateKey, alias);
            } else {
                item = await services.wallet.importMnemonic(chain, mnemonic, alias);
            }
            if (mountedRef.current) {
                setAlias('');
                setPrivateKey('');
                setMnemonic('');
                setSelected(item);
                setAliasDraft(item.alias || '');
                setNotice(action === 'create' ? 'Wallet created' : 'Wallet imported');
                await loadWallets(1);
            }
        } catch (err) {
            if (mountedRef.current) {
                setError(err as Error);
            }
        } finally {
            if (mountedRef.current) {
                setSubmitting(false);
            }
        }
    };

    const canSubmit = action === 'create' || (action === 'privateKey' && privateKey.trim()) || (action === 'mnemonic' && mnemonic.trim());

    return (
        <Page title='Wallets' toolbar={{breadcrumbs: [{title: 'Wallets'}]}}>
            <div className='wallet-page'>
                {error && (
                    <div className='wallet-page__error'>
                        <i className='fa fa-exclamation-triangle' /> {error.message}
                    </div>
                )}
                {notice && (
                    <div className='wallet-page__notice'>
                        <i className='fa fa-check-circle' /> {notice}
                    </div>
                )}
                {loading && items.length === 0 ? (
                    <MockupList height={50} marginTop={30} />
                ) : (
                    <div className='argo-container'>
                        <div className='wallet-page__layout'>
                            <div className='white-box wallet-page__main'>
                                <div className='wallet-page__tabs'>
                                    {CHAINS.map(item => (
                                        <button
                                            type='button'
                                            key={item.value}
                                            className={`wallet-page__tab ${chain === item.value ? 'wallet-page__tab--active' : ''}`}
                                            onClick={() => setChain(item.value)}>
                                            {item.label}
                                        </button>
                                    ))}
                                </div>

                                <form className='wallet-page__form' onSubmit={submitWallet}>
                                    <div className='wallet-page__actions'>
                                        {ACTIONS.map(item => (
                                            <button
                                                type='button'
                                                key={item.value}
                                                className={`wallet-page__action ${action === item.value ? 'wallet-page__action--active' : ''}`}
                                                onClick={() => setAction(item.value)}
                                                title={item.label}>
                                                <i className={item.icon} /> <span>{item.label}</span>
                                            </button>
                                        ))}
                                    </div>
                                    <input className='argo-field' type='text' placeholder='Alias' value={alias} onChange={event => setAlias(event.target.value)} disabled={submitting} />
                                    {action === 'privateKey' && (
                                        <input
                                            className='argo-field wallet-page__secret-input'
                                            type='password'
                                            placeholder='Private key'
                                            value={privateKey}
                                            onChange={event => setPrivateKey(event.target.value)}
                                            disabled={submitting}
                                        />
                                    )}
                                    {action === 'mnemonic' && (
                                        <textarea
                                            className='argo-field wallet-page__mnemonic-input'
                                            placeholder='Mnemonic'
                                            value={mnemonic}
                                            onChange={event => setMnemonic(event.target.value)}
                                            disabled={submitting}
                                        />
                                    )}
                                    <button type='submit' className='argo-button argo-button--base' disabled={!canSubmit || submitting}>
                                        <i className='fa fa-save' /> {submitting ? 'Saving...' : 'Save'}
                                    </button>
                                </form>

                                <form
                                    className='wallet-page__filters'
                                    onSubmit={event => {
                                        event.preventDefault();
                                        loadWallets(1);
                                    }}>
                                    <input className='argo-field' type='text' placeholder='Search address or alias' value={query} onChange={event => setQuery(event.target.value)} />
                                    <button type='submit' className='argo-button argo-button--base-o' disabled={refreshing}>
                                        <i className='fa fa-search' /> Search
                                    </button>
                                    <button type='button' className='argo-button argo-button--base-o' disabled={refreshing} onClick={() => loadWallets(page)}>
                                        <i className='fa fa-sync' /> {refreshing ? 'Refreshing...' : 'Refresh'}
                                    </button>
                                </form>

                                <div className='wallet-page__summary'>
                                    <span>
                                        {rangeStart}-{rangeEnd} of {total}
                                    </span>
                                </div>

                                <div className='argo-table-list argo-table-list--clickable wallet-page__table'>
                                    <div className='argo-table-list__head'>
                                        <div className='wallet-page__row'>
                                            <div>Address</div>
                                            <div>Alias</div>
                                            <div>Source</div>
                                            <div>Created At</div>
                                        </div>
                                    </div>
                                    {items.length === 0 ? (
                                        <div className='argo-table-list__row'>
                                            <div className='row'>
                                                <div className='columns small-12 text-center'>No wallets found</div>
                                            </div>
                                        </div>
                                    ) : (
                                        items.map(item => (
                                            <div
                                                className={`argo-table-list__row ${selected?.id === item.id ? 'wallet-page__selected-row' : ''}`}
                                                key={item.id}
                                                role='button'
                                                tabIndex={0}
                                                onClick={() => selectWallet(item)}>
                                                <div className='wallet-page__row'>
                                                    <div className='wallet-page__address'>{item.address || '-'}</div>
                                                    <div>{item.alias || '-'}</div>
                                                    <div>{sourceLabel(item.source)}</div>
                                                    <div>{formatDate(item.createdAt)}</div>
                                                </div>
                                            </div>
                                        ))
                                    )}
                                </div>

                                <div className='wallet-page__pager'>
                                    <button type='button' className='argo-button argo-button--base-o' disabled={page <= 1 || refreshing} onClick={() => loadWallets(page - 1)}>
                                        <i className='fa fa-chevron-left' /> Prev
                                    </button>
                                    <span>
                                        Page {page} / {totalPages}
                                    </span>
                                    <button type='button' className='argo-button argo-button--base-o' disabled={page >= totalPages || refreshing} onClick={() => loadWallets(page + 1)}>
                                        Next <i className='fa fa-chevron-right' />
                                    </button>
                                </div>
                            </div>

                            <div className='white-box wallet-page__detail'>
                                {selected ? (
                                    <React.Fragment>
                                        <div className='wallet-page__detail-head'>
                                            <div>
                                                <div className='wallet-page__detail-title'>{selected.alias || selected.address}</div>
                                                <div className='wallet-page__detail-subtitle'>{selected.chain}</div>
                                            </div>
                                            <button type='button' className='argo-button argo-button--base-o' disabled={revealing} onClick={revealSecrets}>
                                                <i className='fa fa-eye' /> {revealing ? 'Revealing...' : 'Reveal'}
                                            </button>
                                        </div>
                                        <label className='wallet-page__label'>Alias</label>
                                        <div className='wallet-page__alias-edit'>
                                            <input className='argo-field' type='text' value={aliasDraft} onChange={event => setAliasDraft(event.target.value)} disabled={savingAlias} />
                                            <button type='button' className='argo-button argo-button--base' disabled={savingAlias} onClick={saveAlias}>
                                                <i className='fa fa-save' /> {savingAlias ? 'Saving...' : 'Save'}
                                            </button>
                                        </div>
                                        <div className='wallet-page__detail-grid'>
                                            <span>Address</span>
                                            <strong className='wallet-page__secret'>{selected.address || '-'}</strong>
                                            <span>Source</span>
                                            <strong>{sourceLabel(selected.source)}</strong>
                                            <span>Path</span>
                                            <strong>{selected.derivationPath || '-'}</strong>
                                            <span>Updated</span>
                                            <strong>{formatDate(selected.updatedAt)}</strong>
                                        </div>
                                        <label className='wallet-page__label'>Private Key</label>
                                        <pre className='wallet-page__secret-box'>{selected.privateKey || 'Hidden'}</pre>
                                        <label className='wallet-page__label'>Mnemonic</label>
                                        <pre className='wallet-page__secret-box'>{selected.mnemonic || 'Hidden'}</pre>
                                    </React.Fragment>
                                ) : (
                                    <div className='wallet-page__empty-detail'>Select a wallet</div>
                                )}
                            </div>
                        </div>
                    </div>
                )}
            </div>
        </Page>
    );
};
