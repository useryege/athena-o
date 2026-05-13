import {MockupList, Page} from 'argo-ui';
import * as React from 'react';
import {services} from '../../../shared/services';
import {SourceCodeBlacklistField} from '../../../shared/services/athena-application-service';

require('./source-code-blacklist-list.scss');

export const SourceCodeBlacklistList = () => {
    const [fields, setFields] = React.useState<SourceCodeBlacklistField[]>([]);
    const [loading, setLoading] = React.useState(true);
    const [refreshing, setRefreshing] = React.useState(false);
    const [submitting, setSubmitting] = React.useState(false);
    const [error, setError] = React.useState<Error | null>(null);
    const [newField, setNewField] = React.useState('');
    const requestRef = React.useRef<{abort?: () => void} | null>(null);
    const isMountedRef = React.useRef(false);

    const cleanupRequests = React.useCallback(() => {
        if (requestRef.current?.abort) {
            requestRef.current.abort();
            requestRef.current = null;
        }
    }, []);

    const loadFields = React.useCallback(async () => {
        if (requestRef.current) {
            return;
        }
        if (isMountedRef.current) {
            setRefreshing(true);
        }

        try {
            const req = services.athenaApplication.listSourceCodeBlacklistFields();
            requestRef.current = req;
            const data = await req;
            if (isMountedRef.current) {
                setFields(data);
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
        loadFields();

        return () => {
            isMountedRef.current = false;
            cleanupRequests();
        };
    }, [cleanupRequests, loadFields]);

    const handleRefresh = React.useCallback(() => {
        loadFields();
    }, [loadFields]);

    const handleAddField = async (e: React.FormEvent) => {
        e.preventDefault();
        const trimmedField = newField.trim();
        if (!trimmedField || submitting) {
            return;
        }

        setSubmitting(true);
        setError(null);
        try {
            await services.athenaApplication.addSourceCodeBlacklistField(trimmedField);
            setNewField('');
            await loadFields();
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

    const handleDeleteField = async (field: string) => {
        if (submitting) {
            return;
        }

        setSubmitting(true);
        setError(null);
        try {
            await services.athenaApplication.deleteSourceCodeBlacklistField(field);
            await loadFields();
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
        <Page title='Source Code Blacklist' toolbar={{breadcrumbs: [{title: 'Source Code Blacklist'}]}}>
            <div className='source-code-blacklist-list'>
                {error && (
                    <div className='source-code-blacklist-list__error'>
                        <i className='fa fa-exclamation-triangle' /> Failed to process request: {error.message}
                    </div>
                )}

                {loading && fields.length === 0 ? (
                    <MockupList height={50} marginTop={30} />
                ) : (
                    <div className='argo-container'>
                        <div className='white-box source-code-blacklist-list__box'>
                            <div className='source-code-blacklist-list__controls'>
                                <form className='source-code-blacklist-list__add-form' onSubmit={handleAddField}>
                                    <input
                                        type='text'
                                        className='argo-field'
                                        placeholder='Enter field name to blacklist'
                                        value={newField}
                                        onChange={e => setNewField(e.target.value)}
                                        disabled={submitting}
                                    />
                                    <button type='submit' className='argo-button argo-button--base' disabled={!newField.trim() || submitting}>
                                        {submitting ? 'Adding...' : 'Add Field'}
                                    </button>
                                </form>
                                <div className='source-code-blacklist-list__actions'>
                                    <button type='button' className='argo-button argo-button--base' disabled={refreshing || submitting} onClick={handleRefresh}>
                                        {refreshing ? 'Refreshing...' : 'Refresh'}
                                    </button>
                                </div>
                            </div>
                            <div className='argo-table-list source-code-blacklist-list__table'>
                                <div className='argo-table-list__head'>
                                    <div className='source-code-blacklist-list__row'>
                                        <div>ID</div>
                                        <div>Field</div>
                                        <div className='actions'>Actions</div>
                                    </div>
                                </div>
                                {fields.length === 0 ? (
                                    <div className='argo-table-list__row'>
                                        <div className='row'>
                                            <div className='columns small-12 text-center'>No blacklist fields found</div>
                                        </div>
                                    </div>
                                ) : (
                                    fields.map((item, index) => (
                                        <div className='argo-table-list__row' key={item.field || index}>
                                            <div className='source-code-blacklist-list__row'>
                                                <div>{item.id || '-'}</div>
                                                <div>{item.field}</div>
                                                <div className='actions'>
                                                    <button
                                                        type='button'
                                                        className='argo-button argo-button--base-o'
                                                        disabled={submitting}
                                                        onClick={() => item.field && handleDeleteField(item.field)}>
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
