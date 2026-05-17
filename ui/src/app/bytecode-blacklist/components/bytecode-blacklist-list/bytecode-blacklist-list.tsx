import {MockupList, Page} from 'argo-ui';
import * as React from 'react';
import {services} from '../../../shared/services';
import {BytecodeBlacklistContract} from '../../../shared/services/athena-application-service';

require('./bytecode-blacklist-list.scss');

export const BytecodeBlacklistList = () => {
    const [items, setItems] = React.useState<BytecodeBlacklistContract[]>([]);
    const [loading, setLoading] = React.useState(true);
    const [refreshing, setRefreshing] = React.useState(false);
    const [submitting, setSubmitting] = React.useState(false);
    const [error, setError] = React.useState<Error | null>(null);
    const [newContract, setNewContract] = React.useState('');
    const [newNote, setNewNote] = React.useState('');
    const [editingContract, setEditingContract] = React.useState('');
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
            const req = services.athenaApplication.listBytecodeBlacklistContracts();
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
        const contract = newContract.trim();
        if (!contract || submitting) {
            return;
        }

        setSubmitting(true);
        setError(null);
        try {
            await services.athenaApplication.addBytecodeBlacklistContract(contract, newNote);
            setNewContract('');
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

    const handleDelete = async (contract: string) => {
        if (!contract || submitting) {
            return;
        }

        setSubmitting(true);
        setError(null);
        try {
            await services.athenaApplication.deleteBytecodeBlacklistContract(contract);
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

    const startEdit = (item: BytecodeBlacklistContract) => {
        setEditingContract(item.contract || '');
        setEditingNote(item.note || '');
    };

    const cancelEdit = () => {
        setEditingContract('');
        setEditingNote('');
    };

    const handleSaveNote = async () => {
        if (!editingContract || submitting) {
            return;
        }

        setSubmitting(true);
        setError(null);
        try {
            await services.athenaApplication.updateBytecodeBlacklistContractNote(editingContract, editingNote);
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
        <Page title='Bytecode Blacklist' toolbar={{breadcrumbs: [{title: 'Bytecode Blacklist'}]}}>
            <div className='bytecode-blacklist-list'>
                {error && (
                    <div className='bytecode-blacklist-list__error'>
                        <i className='fa fa-exclamation-triangle' /> Failed to process request: {error.message}
                    </div>
                )}
                {loading && items.length === 0 ? (
                    <MockupList height={50} marginTop={30} />
                ) : (
                    <div className='argo-container'>
                        <div className='white-box bytecode-blacklist-list__box'>
                            <div className='bytecode-blacklist-list__controls'>
                                <form className='bytecode-blacklist-list__add-form' onSubmit={handleAdd}>
                                    <input
                                        type='text'
                                        className='argo-field'
                                        placeholder='Contract address (0x...)'
                                        value={newContract}
                                        onChange={e => setNewContract(e.target.value)}
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
                                    <button type='submit' className='argo-button argo-button--base' disabled={!newContract.trim() || submitting}>
                                        {submitting ? 'Adding...' : 'Add Contract'}
                                    </button>
                                </form>
                                <div className='bytecode-blacklist-list__actions'>
                                    <button type='button' className='argo-button argo-button--base' disabled={refreshing || submitting} onClick={handleRefresh}>
                                        {refreshing ? 'Refreshing...' : 'Refresh'}
                                    </button>
                                </div>
                            </div>

                            <div className='argo-table-list bytecode-blacklist-list__table'>
                                <div className='argo-table-list__head'>
                                    <div className='bytecode-blacklist-list__row'>
                                        <div>Contract</div>
                                        <div>Code Hash</div>
                                        <div>Note</div>
                                        <div>Created At</div>
                                        <div className='actions'>Actions</div>
                                    </div>
                                </div>
                                {items.length === 0 ? (
                                    <div className='argo-table-list__row'>
                                        <div className='row'>
                                            <div className='columns small-12 text-center'>No blacklisted contracts found</div>
                                        </div>
                                    </div>
                                ) : (
                                    items.map((item, index) => {
                                        const contract = item.contract || '';
                                        const editing = editingContract === contract;
                                        return (
                                            <div className='argo-table-list__row' key={contract || index}>
                                                <div className='bytecode-blacklist-list__row'>
                                                    <div className='mono'>{contract || '-'}</div>
                                                    <div className='mono'>{item.codeHash || '-'}</div>
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
                                                                <button type='button' className='argo-button argo-button--base-o' disabled={submitting || !contract} onClick={() => startEdit(item)}>
                                                                    Edit Note
                                                                </button>
                                                                <button
                                                                    type='button'
                                                                    className='argo-button argo-button--base-o'
                                                                    disabled={submitting || !contract}
                                                                    onClick={() => contract && handleDelete(contract)}>
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
