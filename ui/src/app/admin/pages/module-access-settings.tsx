import * as React from 'react';
import {Alert, Button, Spin} from 'antd';
import {ResourceTable, Section, StatusTag} from '../../components';
import {moduleAccessDefinitions, moduleAccessService, type ModuleKey, type ModuleAccessSetting} from '../../shared/module-access-service';
import {requestErrorMessage} from '../../shared/services/requests';
import {formatBeijingDateTime} from '../../shared/format';
import {useAdminReadScope} from '../read-scope';
import {OperationFacts} from '../components/operation-facts';

export const ModuleAccessSettings = ({active, refreshRevision = 0}: {active: boolean; refreshRevision?: number}) => {
    const scope = useAdminReadScope('module-access-settings');
    const [rows, setRows] = React.useState<ModuleAccessSetting[]>();
    const [error, setError] = React.useState('');
    const [confirmed, setConfirmed] = React.useState(false);
    const [loading, setLoading] = React.useState(false);
    const [saving, setSaving] = React.useState<Set<ModuleKey>>(new Set());
    const [saveErrors, setSaveErrors] = React.useState<Partial<Record<ModuleKey, string>>>({});
    const operations = React.useRef<{refresh(): void; save(key: ModuleKey, state: 1 | 2): void}>();
    React.useLayoutEffect(() => {
        let alive = true;
        setConfirmed(false);
        // These flags belong to this effect's requests, not to the previously active tab lifecycle.
        setSaving(new Set());
        setLoading(false);
        let sequence = 0;
        let read: ReturnType<typeof moduleAccessService.settings> | undefined;
        const writes = new Map<ModuleKey, ReturnType<typeof moduleAccessService.save>>();
        const current = () => alive && scope.isCurrent();
        const cancelRead = () => {
            sequence++;
            read?.abort?.();
            read = undefined;
        };
        const refresh = () => {
            if (!current() || !active || document.visibilityState === 'hidden' || read || writes.size) return;
            const version = ++sequence;
            setLoading(true);
            read = moduleAccessService.settings();
            void read
                .then(
                    value => {
                        if (!current() || version !== sequence) return;
                        setRows(value);
                        setConfirmed(true);
                        setError('');
                    },
                    reason => {
                        if (!current() || version !== sequence) return;
                        setConfirmed(false);
                        setError(requestErrorMessage(reason));
                    }
                )
                .finally(() => {
                    if (current() && version === sequence) {
                        read = undefined;
                        setLoading(false);
                    }
                });
        };
        const save = (key: ModuleKey, state: 1 | 2) => {
            if (!current() || writes.has(key)) return;
            cancelRead();
            setLoading(false);
            const write = moduleAccessService.save(key, state);
            writes.set(key, write);
            setSaving(new Set(writes.keys()));
            setSaveErrors(previous => ({...previous, [key]: undefined}));
            void write
                .then(
                    setting => {
                        if (!current()) return;
                        setRows(previous => previous?.map(row => (row.module_key === key ? setting : row)));
                    },
                    reason => {
                        if (!current()) return;
                        setConfirmed(false);
                        setSaveErrors(previous => ({...previous, [key]: `${requestErrorMessage(reason)}. Reloading the actual setting; the change will not be sent again.`}));
                    }
                )
                .finally(() => {
                    writes.delete(key);
                    if (current()) {
                        setSaving(new Set(writes.keys()));
                        refresh();
                    }
                });
        };
        operations.current = {refresh, save};
        const unsubscribe = scope.subscribeInvalidation?.(() => {
            cancelRead();
            writes.forEach(write => write.abort?.());
            setRows(undefined);
            setConfirmed(false);
        });
        const timer = window.setInterval(refresh, 5000);
        document.addEventListener('visibilitychange', refresh);
        window.addEventListener('focus', refresh);
        refresh();
        return () => {
            alive = false;
            cancelRead();
            writes.forEach(write => write.abort?.());
            unsubscribe?.();
            window.clearInterval(timer);
            document.removeEventListener('visibilitychange', refresh);
            window.removeEventListener('focus', refresh);
            operations.current = undefined;
        };
    }, [active, scope.key]);
    React.useEffect(() => {
        if (refreshRevision) operations.current?.refresh();
    }, [refreshRevision]);
    const action = (row: ModuleAccessSetting) => (
        <div>
            <Button
                disabled={!confirmed || saving.has(row.module_key)}
                loading={saving.has(row.module_key)}
                aria-label={`${row.state === 1 ? 'Close' : 'Open'} ${moduleAccessDefinitions.find(item => item.key === row.module_key)?.label} access`}
                onClick={() => operations.current?.save(row.module_key, row.state === 1 ? 2 : 1)}>
                {saving.has(row.module_key) ? 'Saving…' : row.state === 1 ? 'Close access' : 'Open access'}
            </Button>
            {saveErrors[row.module_key] && <p role='alert'>{saveErrors[row.module_key]}</p>}
        </div>
    );
    const label = (row: ModuleAccessSetting) => moduleAccessDefinitions.find(item => item.key === row.module_key)?.label;
    const status = (row: ModuleAccessSetting) => <StatusTag value={row.state === 1 ? 'Open' : 'Closed'} positive={row.state === 1} />;
    const modified = (row: ModuleAccessSetting) =>
        row.updated_at ? `${row.updated_by_username || row.updated_by_account_id || 'Administrator'} · ${formatBeijingDateTime(row.updated_at)} (UTC+8)` : 'Default setting';
    return (
        <Section
            title='Module Access'
            extra={
                <Button onClick={() => operations.current?.refresh()} disabled={loading || saving.size > 0}>
                    Refresh access settings
                </Button>
            }>
            <p>Closing access blocks new user requests. Background tasks and notifications continue running.</p>
            <p>Token access controls are deferred.</p>
            {loading && (
                <span role='status'>
                    <Spin size='small' /> Reading access settings…
                </span>
            )}
            {error && <Alert type='error' title='Module access settings unavailable' description={error} />}
            {!confirmed && rows && <Alert type='warning' title='Access settings could not be confirmed. Changes are disabled until a successful read.' />}
            {rows && (
                <ResourceTable<ModuleAccessSetting>
                    rowKey={row => row.module_key}
                    items={rows}
                    label='Module access settings'
                    columns={[
                        {title: 'Module', render: label},
                        {title: 'Access', render: status},
                        {title: 'Last modified', render: modified},
                        {title: 'Action', render: action}
                    ]}
                    compactRender={row => (
                        <>
                            <OperationFacts
                                columns={1}
                                items={[
                                    {label: 'Module', value: label(row)},
                                    {label: 'Access', value: status(row)},
                                    {label: 'Last modified', value: modified(row)}
                                ]}
                            />
                            {action(row)}
                        </>
                    )}
                />
            )}
        </Section>
    );
};
