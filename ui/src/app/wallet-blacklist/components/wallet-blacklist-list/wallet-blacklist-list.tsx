import {MockupList, Page} from 'argo-ui';
import * as React from 'react';
import {services} from '../../../shared/services';
import {WalletBlacklistEntry} from '../../../shared/services/wallet-service';

require('./wallet-blacklist-list.scss');

export const WalletBlacklistList = () => {
    const [items, setItems] = React.useState<WalletBlacklistEntry[]>([]);
    const [loading, setLoading] = React.useState(true);
    const [refreshing, setRefreshing] = React.useState(false);
    const [submitting, setSubmitting] = React.useState(false);
    const [error, setError] = React.useState<Error | null>(null);
    const [newWallet, setNewWallet] = React.useState('');
    const [newNote, setNewNote] = React.useState('');
    const [editingWallet, setEditingWallet] = React.useState('');
    const [editingNote, setEditingNote] = React.useState('');
    const requestRef = React.useRef<{abort?: () => void} | null>(null);
    const isMountedRef = React.useRef(false);

    const cleanupRequests = React.useCallback(() => {
        if (requestRef.current?.abort) {
            requestRef.current.abort();
            requestRef.current = null;
        }
    }, []);

    const loadItems = React.useCallback(async () => {
        if (requestRef.current) {
            return;
        }
        if (isMountedRef.current) {
            setRefreshing(true);
        }

        try {
            const req = services.wallet.listWalletBlacklistEntries();
            requestRef.current = req;
            const data = await req;
            if (isMountedRef.current) {
                setItems(data);
                setError(null);
            }
        } catch (err) {
            if (isMountedRef.current) {
                setError(err as Error);
            }
        } finally {
            if (isMountedRef.current) {
                setLoading(false);
                setRefreshing(false);
            }
            requestRef.current = null;
        }
    }, []);

    React.useEffect(() => {
        isMountedRef.current = true;
        loadItems();

        return () => {
            isMountedRef.current = false;
            cleanupRequests();
        };
    }, [cleanupRequests, loadItems]);

    const handleRefresh = React.useCallback(() => {
        loadItems();
    }, [loadItems]);

    const handleAdd = async (e: React.FormEvent) => {
        e.preventDefault();
        const wallet = newWallet.trim();
        if (!wallet || submitting) {
            return;
        }

        setSubmitting(true);
        setError(null);
        try {
            await services.wallet.addWalletBlacklistEntry(wallet, newNote);
            setNewWallet('');
            setNewNote('');
            await loadItems();
        } catch (err) {
            if (isMountedRef.current) {
                setError(err as Error);
            }
        } finally {
            if (isMountedRef.current) {
                setSubmitting(false);
            }
        }
    };

    const handleDelete = async (wallet: string) => {
        if (!wallet || submitting) {
            return;
        }

        setSubmitting(true);
        setError(null);
        try {
            await services.wallet.deleteWalletBlacklistEntry(wallet);
            await loadItems();
        } catch (err) {
            if (isMountedRef.current) {
                setError(err as Error);
            }
        } finally {
            if (isMountedRef.current) {
                setSubmitting(false);
            }
        }
    };

    const startEdit = (item: WalletBlacklistEntry) => {
        setEditingWallet(item.wallet || '');
        setEditingNote(item.note || '');
    };

    const cancelEdit = () => {
        setEditingWallet('');
        setEditingNote('');
    };

    const handleSaveNote = async () => {
        if (!editingWallet || submitting) {
            return;
        }

        setSubmitting(true);
        setError(null);
        try {
            await services.wallet.updateWalletBlacklistEntryNote(editingWallet, editingNote);
            cancelEdit();
            await loadItems();
        } catch (err) {
            if (isMountedRef.current) {
                setError(err as Error);
            }
        } finally {
            if (isMountedRef.current) {
                setSubmitting(false);
            }
        }
    };

    return (
        <Page title='Wallet Blacklist' toolbar={{breadcrumbs: [{title: 'Wallet Blacklist'}]}}>
            <div className='wallet-blacklist-list'>
                {error && (
                    <div className='wallet-blacklist-list__error'>
                        <i className='fa fa-exclamation-triangle' /> Failed to process request: {error.message}
                    </div>
                )}
                {loading && items.length === 0 ? (
                    <MockupList height={50} marginTop={30} />
                ) : (
                    <div className='argo-container'>
                        <div className='white-box wallet-blacklist-list__box'>
                            <div className='wallet-blacklist-list__controls'>
                                <form className='wallet-blacklist-list__add-form' onSubmit={handleAdd}>
                                    <input
                                        type='text'
                                        className='argo-field'
                                        placeholder='Wallet address (0x...)'
                                        value={newWallet}
                                        onChange={e => setNewWallet(e.target.value)}
                                        disabled={submitting}
                                    />
                                    <input
                                        type='text'
                                        className='argo-field'
                                        placeholder='Optional note'
                                        value={newNote}
                                        onChange={e => setNewNote(e.target.value)}
                                        disabled={submitting}
                                    />
                                    <button type='submit' className='argo-button argo-button--base' disabled={!newWallet.trim() || submitting}>
                                        {submitting ? 'Adding...' : 'Add Wallet'}
                                    </button>
                                </form>
                                <div className='wallet-blacklist-list__actions'>
                                    <button type='button' className='argo-button argo-button--base' disabled={refreshing || submitting} onClick={handleRefresh}>
                                        {refreshing ? 'Refreshing...' : 'Refresh'}
                                    </button>
                                </div>
                            </div>

                            <div className='argo-table-list wallet-blacklist-list__table'>
                                <div className='argo-table-list__head'>
                                    <div className='wallet-blacklist-list__row'>
                                        <div>Wallet</div>
                                        <div>Note</div>
                                        <div>Created At</div>
                                        <div className='actions'>Actions</div>
                                    </div>
                                </div>
                                {items.length === 0 ? (
                                    <div className='argo-table-list__row'>
                                        <div className='row'>
                                            <div className='columns small-12 text-center'>No blacklisted wallets found</div>
                                        </div>
                                    </div>
                                ) : (
                                    items.map((item, index) => {
                                        const wallet = item.wallet || '';
                                        const editing = editingWallet === wallet;
                                        return (
                                            <div className='argo-table-list__row' key={wallet || index}>
                                                <div className='wallet-blacklist-list__row'>
                                                    <div className='mono'>{wallet || '-'}</div>
                                                    <div>
                                                        {editing ? (
                                                            <input
                                                                type='text'
                                                                className='argo-field'
                                                                value={editingNote}
                                                                onChange={e => setEditingNote(e.target.value)}
                                                                disabled={submitting}
                                                            />
                                                        ) : (
                                                            item.note || '-'
                                                        )}
                                                    </div>
                                                    <div>{item.createdAt || '-'}</div>
                                                    <div className='actions'>
                                                        {editing ? (
                                                            <React.Fragment>
                                                                <button type='button' className='argo-button argo-button--base' disabled={submitting} onClick={handleSaveNote}>
                                                                    Save
                                                                </button>
                                                                <button type='button' className='argo-button argo-button--base-o' disabled={submitting} onClick={cancelEdit}>
                                                                    Cancel
                                                                </button>
                                                            </React.Fragment>
                                                        ) : (
                                                            <React.Fragment>
                                                                <button
                                                                    type='button'
                                                                    className='argo-button argo-button--base-o'
                                                                    disabled={submitting || !wallet}
                                                                    onClick={() => startEdit(item)}>
                                                                    Edit Note
                                                                </button>
                                                                <button
                                                                    type='button'
                                                                    className='argo-button argo-button--base-o'
                                                                    disabled={submitting || !wallet}
                                                                    onClick={() => wallet && handleDelete(wallet)}>
                                                                    Delete
                                                                </button>
                                                            </React.Fragment>
                                                        )}
                                                    </div>
                                                </div>
                                            </div>
                                        );
                                    })
                                )}
                            </div>
                        </div>
                    </div>
                )}
            </div>
        </Page>
    );
};
