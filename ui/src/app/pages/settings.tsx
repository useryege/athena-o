import {DownOutlined, RightOutlined} from '@ant-design/icons';
import {Button, Card, Space, Switch, Tag, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {useBlocker} from 'react-router-dom';
import {AppPage, ChoiceGroup, ResourceTable, Section, StatusTag, useAsyncData} from '../components';
import {accountAccessEqual, cloneAccountAccess, moduleAccessLevel, moduleAccessSummary, replaceModuleAccess} from '../shared/account-access';
import {AccountDataAccess, AccountDataModuleDefinition, accountDataAccessLabel, accountDataModuleGroups, accountDataModules} from '../shared/access-modules';
import {Context, useAuthorization} from '../shared/context';
import {Account, AccountAccess} from '../shared/models';
import {services} from '../shared/services';
import {requestErrorDetails, requestErrorMessage} from '../shared/services/requests';

const narrowAccountsQuery = '(max-width: 900px)';

const useNarrowAccounts = () => {
    const matches = () => typeof window !== 'undefined' && typeof window.matchMedia === 'function' && window.matchMedia(narrowAccountsQuery).matches;
    const [narrow, setNarrow] = React.useState(matches);
    React.useEffect(() => {
        if (typeof window.matchMedia !== 'function') {
            return;
        }
        const query = window.matchMedia(narrowAccountsQuery);
        const update = () => setNarrow(query.matches);
        update();
        query.addEventListener('change', update);
        return () => query.removeEventListener('change', update);
    }, []);
    return narrow;
};

const accountStatus = (account: Account) =>
    account.administrator ? (
        <Typography.Text type='secondary'>Always enabled</Typography.Text>
    ) : (
        <StatusTag value={account.access.loginEnabled ? 'Allowed' : 'Blocked'} positive={account.access.loginEnabled} negative={!account.access.loginEnabled} />
    );

const moduleOptions = (definition: AccountDataModuleDefinition) => [
    {value: AccountDataAccess.None, label: 'No access'},
    {value: AccountDataAccess.Read, label: 'Read only'},
    ...(definition.maxAccess === AccountDataAccess.ReadWrite ? [{value: AccountDataAccess.ReadWrite, label: 'Read & write'}] : [])
];

const AccountAccessEditor = (props: {
    account: Account;
    access: AccountAccess;
    editable: boolean;
    dirty: boolean;
    updating: boolean;
    onChange: (access: AccountAccess) => void;
    onSave: () => void;
    onClose: () => void;
}) => (
    <div className='account-access-editor' aria-busy={props.updating || undefined}>
        <div className='account-access-editor__login'>
            <div>
                <Typography.Text strong={true}>Login access</Typography.Text>
                <Typography.Paragraph type='secondary'>Controls web sessions, Bearer tokens, and API keys for this account.</Typography.Paragraph>
            </div>
            {props.editable ? (
                <Switch
                    aria-label={`Allow ${props.account.name} to log in`}
                    checked={props.access.loginEnabled}
                    disabled={props.updating}
                    checkedChildren='Allowed'
                    unCheckedChildren='Blocked'
                    onChange={loginEnabled => props.onChange({...props.access, loginEnabled})}
                />
            ) : (
                accountStatus(props.account)
            )}
        </div>

        <div className='account-access-editor__groups'>
            {accountDataModuleGroups.map(group => (
                <fieldset className='account-access-module-group' key={group.key} disabled={props.updating}>
                    <legend>{group.label}</legend>
                    <div className='account-access-module-group__items'>
                        {accountDataModules
                            .filter(definition => definition.group === group.key)
                            .map(definition => {
                                const value = moduleAccessLevel(props.access, definition.module);
                                return (
                                    <div className='account-access-module' key={definition.module}>
                                        <div className='account-access-module__copy'>
                                            <Space size={6} wrap={true}>
                                                <Typography.Text strong={true}>{definition.label}</Typography.Text>
                                                {definition.apiOnly && <Tag>API only</Tag>}
                                            </Space>
                                            <Typography.Text type='secondary'>{definition.description}</Typography.Text>
                                        </div>
                                        {props.editable ? (
                                            <ChoiceGroup<AccountDataAccess>
                                                className='account-access-module__choice'
                                                size='small'
                                                ariaLabel={`${definition.label} data access for ${props.account.name}`}
                                                value={value}
                                                options={moduleOptions(definition)}
                                                disabled={props.updating}
                                                onChange={dataAccess => props.onChange(replaceModuleAccess(props.access, definition.module, dataAccess))}
                                            />
                                        ) : (
                                            <Typography.Text className='account-access-module__readonly' type='secondary'>
                                                {accountDataAccessLabel(value)}
                                            </Typography.Text>
                                        )}
                                    </div>
                                );
                            })}
                    </div>
                </fieldset>
            ))}
        </div>

        <div className='account-access-editor__footer'>
            <Typography.Text className='account-access-editor__state' type={props.dirty ? 'warning' : 'secondary'} aria-live='polite'>
                {props.editable ? (props.dirty ? 'Unsaved access changes' : `Revision ${props.access.revision}`) : `Revision ${props.access.revision}`}
            </Typography.Text>
            <Space wrap={true}>
                <Button disabled={props.updating} onClick={props.onClose}>
                    {props.editable ? 'Cancel' : 'Close'}
                </Button>
                {props.editable && (
                    <Button type='primary' loading={props.updating} disabled={!props.dirty || props.updating} onClick={props.onSave}>
                        Save changes
                    </Button>
                )}
            </Space>
        </div>
    </div>
);

export const SettingsPage = () => {
    const ctx = React.useContext(Context);
    const authorization = useAuthorization();
    const narrow = useNarrowAccounts();
    const accounts = useAsyncData<Account[]>(
        () =>
            (authorization.isAdmin ? services.accounts.list() : services.accounts.get(authorization.user.username).then(account => [account])) as Promise<Account[]> & {
                abort?: () => void;
            },
        [authorization.isAdmin, authorization.user.username]
    );
    const [accountItems, setAccountItems] = React.useState<Account[]>([]);
    const [expandedAccountName, setExpandedAccountName] = React.useState('');
    const [draft, setDraft] = React.useState<AccountAccess>();
    const [updatingAccountName, setUpdatingAccountName] = React.useState('');
    const conflictReloadAccountRef = React.useRef('');

    React.useEffect(() => {
        if (!accounts.data) {
            return;
        }
        setAccountItems(accounts.data);
        const conflictAccount = conflictReloadAccountRef.current;
        if (conflictAccount) {
            conflictReloadAccountRef.current = '';
            const latest = accounts.data.find(account => account.name === conflictAccount);
            if (latest && expandedAccountName === conflictAccount) {
                setDraft(cloneAccountAccess(latest.access));
            }
        }
        if (expandedAccountName && !accounts.data.some(account => account.name === expandedAccountName)) {
            setExpandedAccountName('');
            setDraft(undefined);
        }
    }, [accounts.data, expandedAccountName]);

    const expandedAccount = accountItems.find(account => account.name === expandedAccountName);
    const editorIsWritable = Boolean(authorization.isAdmin && expandedAccount && !expandedAccount.administrator);
    const dirty = Boolean(editorIsWritable && expandedAccount && draft && !accountAccessEqual(expandedAccount.access, draft));
    const navigationBlocker = useBlocker(dirty);

    React.useEffect(() => {
        if (!dirty) {
            return;
        }
        const beforeUnload = (event: BeforeUnloadEvent) => {
            event.preventDefault();
            event.returnValue = '';
        };
        window.addEventListener('beforeunload', beforeUnload);
        return () => window.removeEventListener('beforeunload', beforeUnload);
    }, [dirty]);

    React.useEffect(() => {
        if (navigationBlocker.state !== 'blocked') {
            return;
        }
        let resolved = false;
        const handle = ctx.modal.confirm({
            title: 'Discard unsaved account access changes?',
            content: `Changes for ${expandedAccountName} have not been saved.`,
            okText: 'Discard and leave',
            onOk: () => {
                resolved = true;
                setDraft(expandedAccount ? cloneAccountAccess(expandedAccount.access) : undefined);
                navigationBlocker.proceed();
            },
            onCancel: () => {
                resolved = true;
                navigationBlocker.reset();
            }
        });
        return () => {
            if (!resolved) {
                handle.destroy();
            }
        };
    }, [ctx.modal, expandedAccount, expandedAccountName, navigationBlocker.state]);

    const openAccount = (account: Account) => {
        if (account.administrator) {
            return;
        }
        setExpandedAccountName(account.name);
        setDraft(authorization.isAdmin ? cloneAccountAccess(account.access) : undefined);
    };

    const requestAccount = (account: Account) => {
        if (updatingAccountName) {
            if (updatingAccountName !== account.name) {
                ctx.notifications.info('Account access update in progress', `Wait for ${updatingAccountName} to finish before opening another account.`);
            }
            return;
        }
        if (expandedAccountName === account.name) {
            if (!dirty) {
                setExpandedAccountName('');
                setDraft(undefined);
                return;
            }
            ctx.modal.confirm({
                title: `Discard changes for ${account.name}?`,
                content: 'The unsaved login and module access changes will be lost.',
                okText: 'Discard changes',
                onOk: () => {
                    setExpandedAccountName('');
                    setDraft(undefined);
                }
            });
            return;
        }
        if (!dirty) {
            openAccount(account);
            return;
        }
        ctx.modal.confirm({
            title: `Discard changes for ${expandedAccountName}?`,
            content: `Open ${account.name} after discarding the current unsaved changes.`,
            okText: 'Discard and switch',
            onOk: () => openAccount(account)
        });
    };

    const updateAccountAccess = async (account: Account, nextAccess: AccountAccess) => {
        setUpdatingAccountName(account.name);
        try {
            const updated = await services.accounts.updateAccess(account.name, nextAccess);
            setAccountItems(current => current.map(item => (item.name === account.name ? updated : item)));
            setDraft(cloneAccountAccess(updated.access));
            ctx.notifications.success(
                'Account access updated',
                `${account.name}: login ${updated.access.loginEnabled ? 'allowed' : 'blocked'}, ${moduleAccessSummary(updated.access)}.`
            );
        } catch (err) {
            if (requestErrorDetails(err).status === 409) {
                conflictReloadAccountRef.current = account.name;
                setDraft(undefined);
                accounts.reload();
                ctx.notifications.warning(
                    'Account access changed',
                    'A newer setting was saved elsewhere. Your draft was discarded and the authoritative access is being reloaded.'
                );
            } else {
                ctx.notifications.error('Could not update account access', requestErrorMessage(err, 'Your draft has been kept.'));
            }
        } finally {
            setUpdatingAccountName('');
        }
    };

    const saveDraft = () => {
        if (!expandedAccount || !draft || expandedAccount.administrator || updatingAccountName) {
            return;
        }
        const loginDisabled = expandedAccount.access.loginEnabled && !draft.loginEnabled;
        const downgraded = accountDataModules.filter(definition => moduleAccessLevel(draft, definition.module) < moduleAccessLevel(expandedAccount.access, definition.module));
        const writeGranted = accountDataModules.filter(
            definition =>
                moduleAccessLevel(expandedAccount.access, definition.module) < AccountDataAccess.ReadWrite &&
                moduleAccessLevel(draft, definition.module) === AccountDataAccess.ReadWrite
        );
        if (!loginDisabled && downgraded.length === 0 && writeGranted.length === 0) {
            void updateAccountAccess(expandedAccount, draft);
            return;
        }
        ctx.modal.confirm({
            title: `Confirm access changes for ${expandedAccount.name}`,
            content: (
                <div className='account-access-confirmation'>
                    <Typography.Paragraph>These changes take effect on the account's next request.</Typography.Paragraph>
                    <ul>
                        {loginDisabled && <li>Disable login, existing web sessions, Bearer tokens, and API keys.</li>}
                        {downgraded.length > 0 && <li>Reduce access: {downgraded.map(definition => definition.label).join(', ')}.</li>}
                        {writeGranted.length > 0 && <li>Grant sensitive write access: {writeGranted.map(definition => definition.label).join(', ')}.</li>}
                    </ul>
                </div>
            ),
            okText: 'Apply changes',
            onOk: () => updateAccountAccess(expandedAccount, draft)
        });
    };

    const renderEditor = (account: Account) => {
        if (account.name !== expandedAccountName || account.administrator) {
            return null;
        }
        const editable = authorization.isAdmin;
        const access = editable ? draft || cloneAccountAccess(account.access) : account.access;
        return (
            <AccountAccessEditor
                account={account}
                access={access}
                editable={editable}
                dirty={editable && !accountAccessEqual(account.access, access)}
                updating={updatingAccountName === account.name}
                onChange={setDraft}
                onSave={saveDraft}
                onClose={() => requestAccount(account)}
            />
        );
    };

    const accountAction = (account: Account) => {
        if (account.administrator) {
            return <Typography.Text type='secondary'>Fixed</Typography.Text>;
        }
        const expanded = expandedAccountName === account.name;
        return (
            <Button
                type='text'
                className='account-access-expand-button'
                icon={expanded ? <DownOutlined /> : <RightOutlined />}
                disabled={updatingAccountName === account.name}
                aria-expanded={expanded}
                onClick={() => requestAccount(account)}>
                {expanded ? 'Close' : authorization.isAdmin ? 'Edit access' : 'View access'}
            </Button>
        );
    };

    const columns: ColumnsType<Account> = [
        {title: 'Name', dataIndex: 'name'},
        {title: 'Login access', render: accountStatus},
        {title: 'Module access', render: account => moduleAccessSummary(account.access, account.administrator)},
        {title: 'Capabilities', render: item => (item.capabilities || []).join(', ') || '-'},
        {title: '', width: 140, align: 'right', render: accountAction}
    ];

    const compactAccount = (account: Account) => {
        const expanded = expandedAccountName === account.name;
        return (
            <Card
                className={`account-compact-card${updatingAccountName === account.name ? ' account-compact-card--updating' : ''}`}
                size='small'
                title={account.name}
                extra={accountStatus(account)}>
                <div className='account-compact-card__details'>
                    <div>
                        <Typography.Text type='secondary'>Module access</Typography.Text>
                        <span>{moduleAccessSummary(account.access, account.administrator)}</span>
                    </div>
                    <div>
                        <Typography.Text type='secondary'>Capabilities</Typography.Text>
                        <span>{(account.capabilities || []).join(', ') || '-'}</span>
                    </div>
                </div>
                {!account.administrator && <div className='account-compact-card__action'>{accountAction(account)}</div>}
                {narrow && expanded && <div className='account-compact-card__expanded'>{renderEditor(account)}</div>}
            </Card>
        );
    };

    return (
        <AppPage
            title='Settings'
            loading={accounts.loading}
            error={accounts.error}
            onRefresh={() => {
                void authorization.refresh();
                accounts.reload();
            }}>
            <Section title='Accounts'>
                <ResourceTable
                    rowKey='name'
                    label={authorization.isAdmin ? 'Account login and module access' : 'Your account access'}
                    items={accountItems}
                    columns={columns}
                    loading={accounts.loading}
                    rowClassName={account => (updatingAccountName === account.name ? 'account-access-row--updating' : '')}
                    compactRender={compactAccount}
                    compactEmptyDescription='No account available'
                    expandable={{
                        showExpandColumn: false,
                        expandedRowKeys: !narrow && expandedAccountName ? [expandedAccountName] : [],
                        expandedRowRender: account => (!narrow ? renderEditor(account) : null),
                        rowExpandable: account => !account.administrator
                    }}
                />
            </Section>
        </AppPage>
    );
};
