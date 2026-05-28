import {MockupList, Page} from 'argo-ui';
import * as React from 'react';
import {services} from '../../shared/services';
import {SourceQualityPrompt} from '../../shared/services/athena-solidity-service';

const renderTime = (value?: string) => (value ? new Date(value).toLocaleString() : '-');

export const SourceQualityPromptList = () => {
    const [items, setItems] = React.useState<SourceQualityPrompt[]>([]);
    const [loading, setLoading] = React.useState(true);
    const [refreshing, setRefreshing] = React.useState(false);
    const [submitting, setSubmitting] = React.useState(false);
    const [error, setError] = React.useState<Error | null>(null);
    const [editingID, setEditingID] = React.useState<number | undefined>();
    const [name, setName] = React.useState('');
    const [systemPrompt, setSystemPrompt] = React.useState('');
    const requestRef = React.useRef<{abort?: () => void} | null>(null);
    const isMountedRef = React.useRef(false);

    const loadItems = React.useCallback(async () => {
        if (requestRef.current) {
            return;
        }
        if (isMountedRef.current) {
            setRefreshing(true);
        }
        try {
            const req = services.athenaSolidity.listSourceQualityPrompts();
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
            if (requestRef.current?.abort) {
                requestRef.current.abort();
            }
        };
    }, [loadItems]);

    const resetForm = () => {
        setEditingID(undefined);
        setName('');
        setSystemPrompt('');
    };

    const startEdit = (item: SourceQualityPrompt) => {
        setEditingID(item.id);
        setName(item.name || '');
        setSystemPrompt(item.systemPrompt || '');
    };

    const handleSubmit = async (event: React.FormEvent) => {
        event.preventDefault();
        if (!name.trim() || !systemPrompt.trim() || submitting) {
            return;
        }
        setSubmitting(true);
        setError(null);
        try {
            if (editingID) {
                await services.athenaSolidity.updateSourceQualityPrompt(editingID, name, systemPrompt);
            } else {
                await services.athenaSolidity.createSourceQualityPrompt(name, systemPrompt);
            }
            resetForm();
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

    const handleActivate = async (id?: number) => {
        if (!id || submitting) {
            return;
        }
        setSubmitting(true);
        setError(null);
        try {
            await services.athenaSolidity.activateSourceQualityPrompt(id);
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

    const handleDelete = async (id?: number) => {
        if (!id || submitting) {
            return;
        }
        setSubmitting(true);
        setError(null);
        try {
            await services.athenaSolidity.deleteSourceQualityPrompt(id);
            if (editingID === id) {
                resetForm();
            }
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
        <Page title='Source Quality Prompts' toolbar={{breadcrumbs: [{title: 'Solidity', path: '/solidity/bytecodes'}, {title: 'Source Quality Prompts'}]}}>
            <div className='solidity-source-quality-prompts'>
                {error && (
                    <div className='solidity-source-quality-prompts__error'>
                        <i className='fa fa-exclamation-triangle' /> Failed to process request: {error.message}
                    </div>
                )}
                {loading && items.length === 0 ? (
                    <MockupList height={50} marginTop={30} />
                ) : (
                    <div className='argo-container'>
                        <div className='white-box solidity-source-quality-prompts__box'>
                            <form className='solidity-source-quality-prompts__form' onSubmit={handleSubmit}>
                                <input className='argo-field' placeholder='Prompt name' value={name} onChange={e => setName(e.target.value)} disabled={submitting} />
                                <textarea
                                    className='argo-field'
                                    placeholder='System prompt'
                                    value={systemPrompt}
                                    onChange={e => setSystemPrompt(e.target.value)}
                                    disabled={submitting}
                                />
                                <div className='solidity-source-quality-prompts__form-actions'>
                                    <button type='submit' className='argo-button argo-button--base' disabled={!name.trim() || !systemPrompt.trim() || submitting}>
                                        {editingID ? 'Save New Version' : 'Create Prompt'}
                                    </button>
                                    {editingID && (
                                        <button type='button' className='argo-button argo-button--base-o' disabled={submitting} onClick={resetForm}>
                                            Cancel
                                        </button>
                                    )}
                                    <button type='button' className='argo-button argo-button--base-o' disabled={refreshing || submitting} onClick={loadItems}>
                                        {refreshing ? 'Refreshing...' : 'Refresh'}
                                    </button>
                                </div>
                            </form>

                            <div className='argo-table-list solidity-source-quality-prompts__table'>
                                <div className='argo-table-list__head'>
                                    <div className='solidity-source-quality-prompts__row'>
                                        <div>Version</div>
                                        <div>Name</div>
                                        <div>Status</div>
                                        <div>Updated</div>
                                        <div>Prompt</div>
                                        <div className='actions'>Actions</div>
                                    </div>
                                </div>
                                {items.length === 0 ? (
                                    <div className='argo-table-list__row'>
                                        <div className='columns small-12 text-center'>No prompts found</div>
                                    </div>
                                ) : (
                                    items.map(item => (
                                        <div className='argo-table-list__row' key={item.id || item.version}>
                                            <div className='solidity-source-quality-prompts__row'>
                                                <div>{item.version || '-'}</div>
                                                <div>{item.name || '-'}</div>
                                                <div>
                                                    <span
                                                        className={`solidity-source-quality-prompts__badge ${item.isActive ? 'solidity-source-quality-prompts__badge--active' : ''}`}>
                                                        {item.isActive ? 'Active' : '-'}
                                                    </span>
                                                </div>
                                                <div>{renderTime(item.updatedAt)}</div>
                                                <div className='solidity-source-quality-prompts__prompt'>{item.systemPrompt || '-'}</div>
                                                <div className='actions'>
                                                    <button type='button' className='argo-button argo-button--base-o' disabled={submitting} onClick={() => startEdit(item)}>
                                                        Edit
                                                    </button>
                                                    <button
                                                        type='button'
                                                        className='argo-button argo-button--base-o'
                                                        disabled={submitting || item.isActive}
                                                        onClick={() => handleActivate(item.id)}>
                                                        Activate
                                                    </button>
                                                    <button
                                                        type='button'
                                                        className='argo-button argo-button--base-o'
                                                        disabled={submitting || item.isActive}
                                                        onClick={() => handleDelete(item.id)}>
                                                        Delete
                                                    </button>
                                                </div>
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
