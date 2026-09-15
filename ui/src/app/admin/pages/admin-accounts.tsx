import {OperationFacts} from '../components/operation-facts';
import {ArrowLeftOutlined, CheckOutlined, DeleteOutlined, SaveOutlined, UploadOutlined} from '@ant-design/icons';
import {Alert, Button, Empty, Form, Input, Pagination, Select, Space, Switch, Tabs, Tag, Typography, Upload} from 'antd';
import * as React from 'react';
import {useBlocker, useSearchParams} from 'react-router-dom';
import {AppPage, ChoiceGroup, Section, StatusTag, useAsyncData} from '../../components';
import {accountAccessEqual, cloneAccountAccess, moduleAccessLevel, moduleAccessSummary, replaceModuleAccess} from '../../shared/account-access';
import {
    allowedAccessLevels,
    AccountDataAccess,
    AccountDataModuleDefinition,
    accountAccessDisplayModules,
    accountDataAccessLabel,
    accountDataModuleGroups,
    accountDataModules
} from '../../shared/access-modules';
import {Context, useAuthorization} from '../../shared/context';
import {accountStatusForAccess, Account, AccountAccess, AccountProfile, AccountStatus, AccountTier, parseAccountStatus} from '../../shared/models';
import {AccountAvatar, accountTierLabel, identityPresentation, identityProviderLabel} from '../../shared/account-presentation';
import {adminServices as services} from '../services';
import type {AccountsPage} from '../accounts-service';
import {requestErrorDetails, requestErrorMessage} from '../../shared/services/requests';
import {useAdminReadScope} from '../read-scope';
import {hasControlCharacters, unicodeCharacterCount} from '../../shared/validation';

const effectiveAccountStatus = (account: Account) =>
    account.status === AccountStatus.Unspecified ? accountStatusForAccess(account.access, account.administrator) : account.status;

const accountStatusLabel = (status: AccountStatus) => {
    switch (status) {
        case AccountStatus.Pending:
            return 'Pending';
        case AccountStatus.Active:
            return 'Active';
        case AccountStatus.Blocked:
            return 'Blocked';
        default:
            return 'Unknown';
    }
};

const accountStatus = (account: Account) => {
    const status = effectiveAccountStatus(account);
    return <StatusTag value={accountStatusLabel(status)} positive={status === AccountStatus.Active} negative={status === AccountStatus.Blocked} />;
};

const identityTime = (value: number) => (value > 0 ? new Date(value * 1000).toLocaleString() : 'Not yet');
const identityListTime = (value: number) => (value > 0 ? new Date(value * 1000).toLocaleDateString() : 'Not yet');
const accountIdentityValue = (account: Account) => identityPresentation(account.identity).value;
const accountPrimaryLabel = (account: Account) => account.profile.displayName || accountIdentityValue(account) || account.username;

const moduleOptions = (definition: AccountDataModuleDefinition) => allowedAccessLevels(definition).map(value => ({value, label: accountDataAccessLabel(value)}));

export const AccountAccessEditor = (props: {
    account: Account;
    access: AccountAccess;
    editable: boolean;
    dirty: boolean;
    updating: boolean;
    onChange: (access: AccountAccess) => void;
    onSave: () => void;
    onReset: () => void;
}) => (
    <div className='account-access-editor' aria-busy={props.updating || undefined}>
        <div className='account-access-editor__entitlements'>
            {[
                {
                    key: 'login',
                    label: identityPresentation(props.account.identity).signInLabel,
                    description: 'Controls new sign-ins through this identity and all existing Athena sessions and API keys.',
                    checked: props.access.loginEnabled,
                    onChange: (loginEnabled: boolean) => props.onChange({...props.access, loginEnabled})
                },
                {
                    key: 'api-key',
                    label: 'API Key access',
                    description: 'Allows this account to create and use API keys. Turning it off pauses existing keys without deleting them.',
                    checked: props.access.apiKeyEnabled,
                    onChange: (apiKeyEnabled: boolean) => props.onChange({...props.access, apiKeyEnabled})
                },
                {
                    key: 'profit-sharing',
                    label: 'Profit Sharing access',
                    description: 'Makes this account eligible for Profit Sharing membership and member RPCs.',
                    checked: props.access.profitSharingEnabled,
                    onChange: (profitSharingEnabled: boolean) => props.onChange({...props.access, profitSharingEnabled})
                }
            ].map(item => (
                <div className='account-access-editor__entitlement' key={item.key}>
                    <div>
                        <Typography.Text strong={true}>{item.label}</Typography.Text>
                        <Typography.Paragraph type='secondary'>{item.description}</Typography.Paragraph>
                    </div>
                    {props.editable ? (
                        <div className='account-entitlement-control'>
                            <Switch aria-label={`${item.label} for @${props.account.username}`} checked={item.checked} disabled={props.updating} onChange={item.onChange} />
                            <Typography.Text type='secondary'>{item.checked ? 'Enabled' : 'Disabled'}</Typography.Text>
                        </div>
                    ) : (
                        <StatusTag value={item.checked ? 'Enabled' : 'Disabled'} positive={item.checked} negative={!item.checked} />
                    )}
                </div>
            ))}
        </div>
        <div className='account-access-editor__groups'>
            {accountDataModuleGroups.map(group => (
                <fieldset className='account-access-module-group' key={group.key} disabled={props.updating}>
                    <legend>{group.label}</legend>
                    <div className='account-access-module-group__items'>
                        {accountAccessDisplayModules
                            .filter(definition => definition.group === group.key)
                            .map(definition => {
                                const value = moduleAccessLevel(props.access, definition.module);
                                return (
                                    <div className='account-access-module' key={definition.module}>
                                        <details className='account-access-module__copy'>
                                            <summary>
                                                <Space size={6} wrap={true}>
                                                    <Typography.Text strong={true}>{definition.label}</Typography.Text>
                                                    {definition.apiOnly && <Tag>API only</Tag>}
                                                </Space>
                                            </summary>
                                            <Typography.Text type='secondary'>{definition.description}</Typography.Text>
                                        </details>
                                        {props.editable ? (
                                            <Select<AccountDataAccess>
                                                className='account-access-module__choice'
                                                aria-label={`${definition.label} data access for @${props.account.username}`}
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
                {props.dirty ? 'Unsaved access changes' : `Revision ${props.access.revision}`}
            </Typography.Text>
            {props.editable && (
                <Space wrap={true}>
                    <Button disabled={!props.dirty || props.updating} onClick={props.onReset}>
                        Reset
                    </Button>
                    <Button type='primary' icon={<SaveOutlined aria-hidden />} loading={props.updating} disabled={!props.dirty || props.updating} onClick={props.onSave}>
                        Save access
                    </Button>
                </Space>
            )}
        </div>
    </div>
);

export const AdminAccountsPage = () => {
    const scope = useAdminReadScope('account-directory');
    return scope.isCurrent() ? (
        <AdminAccountsWorkspace key={scope.key} isCurrent={scope.isCurrent} />
    ) : (
        <AppPage title='Account Administration' loading>
            <p>Checking administrator access…</p>
        </AppPage>
    );
};

const AdminAccountsWorkspace = ({isCurrent}: {isCurrent: () => boolean}) => {
    const ctx = React.useContext(Context);
    const authorization = useAuthorization();
    const confirmationHandles = React.useRef<Array<{destroy(): void}>>([]);
    React.useEffect(
        () => () => {
            for (const handle of confirmationHandles.current) handle.destroy();
        },
        []
    );
    const confirm: typeof ctx.modal.confirm = options => {
        const handle = ctx.modal.confirm({
            ...options,
            focusable: {trap: true, focusTriggerAfterClose: true},
            wrapProps: {
                onKeyDownCapture: event => {
                    if (event.key !== 'Tab') return;
                    const buttons = Array.from(event.currentTarget.querySelectorAll<HTMLButtonElement>('button:not([disabled])'));
                    const first = buttons[0],
                        last = buttons[buttons.length - 1];
                    if (first && last && ((!event.shiftKey && document.activeElement === last) || (event.shiftKey && document.activeElement === first))) {
                        event.preventDefault();
                        (event.shiftKey ? last : first).focus();
                    }
                }
            }
        });
        confirmationHandles.current.push(handle);
        return handle;
    };
    const [searchParams, setSearchParams] = useSearchParams();
    const mobileAccountId = searchParams.get('account') || '';
    const [queryDraft, setQueryDraft] = React.useState(searchParams.get('query') || '');
    const [query, setQuery] = React.useState(searchParams.get('query') || '');
    const [status, setStatus] = React.useState<AccountStatus | undefined>(searchParams.has('status') ? parseAccountStatus(searchParams.get('status')) : undefined);
    const [page, setPage] = React.useState(Number(searchParams.get('page')) || 1);
    const [pageSize, setPageSize] = React.useState(Number(searchParams.get('pageSize')) || 50);
    const updateSearch = (values: Record<string, string>) =>
        setSearchParams(previous => {
            const next = new URLSearchParams(previous);
            for (const [key, value] of Object.entries(values)) {
                if (value) next.set(key, value);
                else next.delete(key);
            }
            return next;
        });
    const accounts = useAsyncData<AccountsPage>(() => services.adminAccounts.list({query, status, page, pageSize}), [query, status, page, pageSize]);
    const [items, setItems] = React.useState<Account[]>([]);
    const [selectedId, setSelectedId] = React.useState('');
    // A request belongs to this selection, even if the directory later returns to the same account.
    const selectionRef = React.useRef({id: selectedId});
    if (selectionRef.current.id !== selectedId) selectionRef.current = {id: selectedId};
    const selection = selectionRef.current;
    const isSelectedCurrent = () => isCurrent() && selectionRef.current === selection;
    const [displayName, setDisplayName] = React.useState('');
    const [tier, setTier] = React.useState<AccountTier>(AccountTier.Standard);
    const [accessDraft, setAccessDraft] = React.useState<AccountAccess>();
    const [savingProfile, setSavingProfile] = React.useState(false);
    const [savingTier, setSavingTier] = React.useState(false);
    const [savingAccess, setSavingAccess] = React.useState(false);
    const [uploadingAvatar, setUploadingAvatar] = React.useState(false);

    React.useEffect(() => {
        if (!accounts.data) {
            return;
        }
        setItems(accounts.data.items);
        setSelectedId(current => {
            if (mobileAccountId && accounts.data?.items.some(account => account.id === mobileAccountId)) {
                return mobileAccountId;
            }
            return current && accounts.data?.items.some(account => account.id === current) ? current : accounts.data?.items[0]?.id || '';
        });
    }, [accounts.data, mobileAccountId]);

    React.useEffect(() => {
        if (accounts.data && mobileAccountId && !accounts.data.items.some(account => account.id === mobileAccountId)) {
            setSearchParams(
                previous => {
                    const next = new URLSearchParams(previous);
                    next.delete('account');
                    return next;
                },
                {replace: true}
            );
        }
    }, [accounts.data, mobileAccountId, setSearchParams]);

    const selected = items.find(account => account.id === selectedId);
    const profileEditable = Boolean(selected && selected.id !== authorization.user.accountId);
    const accessEditable = Boolean(selected && !selected.administrator);
    const displayNameLength = unicodeCharacterCount(displayName.trim());
    const displayNameValid = displayNameLength >= 1 && displayNameLength <= 80 && !hasControlCharacters(displayName.trim());
    const displayNameDirty = Boolean(selected && displayName.trim() !== selected.profile.displayName);
    const tierDirty = Boolean(selected && tier !== selected.profile.tier);
    const accessDirty = Boolean(selected && accessDraft && accessEditable && !accountAccessEqual(selected.access, accessDraft));
    const dirty = displayNameDirty || tierDirty || accessDirty;
    const blocker = useBlocker(dirty);

    const loadDraft = React.useCallback((account?: Account) => {
        setDisplayName(account?.profile.displayName || '');
        setTier(account?.profile.tier || AccountTier.Standard);
        setAccessDraft(account ? cloneAccountAccess(account.access) : undefined);
    }, []);

    React.useEffect(() => {
        loadDraft(selected);
        setSavingProfile(false);
        setSavingTier(false);
        setSavingAccess(false);
        setUploadingAvatar(false);
    }, [selected?.id]);

    const changeListScope = (change: () => void) => {
        if (!dirty) {
            change();
            return;
        }
        confirm({
            className: 'admin-operation-confirm',
            autoFocusButton: 'cancel',
            title: `Discard changes for @${selected?.username || 'account'}?`,
            content: 'Filtering or paging the account directory will discard the current drafts.',
            okText: 'Discard and continue',
            onOk: () => {
                loadDraft(selected);
                change();
            }
        });
    };

    const searchAccounts = (value: string) =>
        changeListScope(() => {
            const next = value.trim();
            setQueryDraft(next);
            setQuery(next);
            updateSearch({query: next, page: '1'});
            setPage(1);
        });

    const filterAccounts = (nextStatus?: AccountStatus) =>
        changeListScope(() => {
            setStatus(nextStatus);
            updateSearch({status: nextStatus ? String(nextStatus) : '', page: '1'});
            setPage(1);
        });

    const paginateAccounts = (nextPage: number, nextPageSize: number) =>
        changeListScope(() => {
            setPage(nextPageSize === pageSize ? nextPage : 1);
            setPageSize(nextPageSize);
            updateSearch({page: String(nextPageSize === pageSize ? nextPage : 1), pageSize: String(nextPageSize)});
        });

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
        if (blocker.state !== 'blocked') {
            return;
        }
        let resolved = false;
        const handle = confirm({
            className: 'admin-operation-confirm',
            autoFocusButton: 'cancel',
            title: 'Discard unsaved account changes?',
            content: selected ? `Changes for @${selected.username} have not been saved.` : undefined,
            okText: 'Discard and leave',
            onOk: () => {
                resolved = true;
                loadDraft(selected);
                blocker.proceed();
            },
            onCancel: () => {
                resolved = true;
                blocker.reset();
            }
        });
        return () => {
            if (!resolved) {
                handle.destroy();
            }
        };
    }, [blocker.state, ctx.modal, loadDraft, selected]);

    const chooseAccount = (id: string, openMobileDetail = false) => {
        const choose = () => {
            setSelectedId(id);
            if (openMobileDetail) {
                updateSearch({account: id});
            }
        };
        if (!dirty) {
            choose();
            return;
        }
        confirm({
            className: 'admin-operation-confirm',
            autoFocusButton: 'cancel',
            title: `Discard changes for @${selected?.username || 'account'}?`,
            content: 'Open the selected account after discarding the current drafts.',
            okText: 'Discard and switch',
            onOk: choose
        });
    };

    const replaceProfile = (id: string, profile: AccountProfile) => {
        setItems(current => current.map(account => (account.id === id ? {...account, profile} : account)));
    };

    const conflict = async (message: string) => {
        if (!selected) {
            return undefined;
        }
        const latest = await services.adminAccounts.get(selected.id);
        if (!isSelectedCurrent()) return undefined;
        setItems(current => current.map(account => (account.id === latest.id ? latest : account)));
        ctx.notifications.warning('Account changed elsewhere', message);
        return latest;
    };

    const saveDisplayName = async () => {
        if (!selected || !profileEditable || !displayNameDirty || !displayNameValid) {
            return;
        }
        setSavingProfile(true);
        try {
            const profile = await services.adminAccounts.updateProfile(selected.id, displayName.trim(), selected.profile.revision);
            if (!isSelectedCurrent()) return;
            replaceProfile(selected.id, profile);
            setDisplayName(profile.displayName);
            ctx.notifications.success('Account profile updated', `@${selected.username}`);
        } catch (err) {
            if (!isSelectedCurrent()) return;
            if (requestErrorDetails(err).status === 409) {
                await conflict('The latest profile was reloaded. Your display-name draft was kept; review it before saving again.');
            } else {
                ctx.notifications.error('Could not update account profile', requestErrorMessage(err));
            }
        } finally {
            if (isSelectedCurrent()) setSavingProfile(false);
        }
    };

    const saveTier = async () => {
        if (!selected || !profileEditable || !tierDirty) {
            return;
        }
        setSavingTier(true);
        try {
            const profile = await services.adminAccounts.updateTier(selected.id, tier, selected.profile.revision);
            if (!isSelectedCurrent()) return;
            replaceProfile(selected.id, profile);
            setTier(profile.tier);
            ctx.notifications.success('Account tier updated', `@${selected.username} is now ${accountTierLabel(profile.tier)}.`);
        } catch (err) {
            if (!isSelectedCurrent()) return;
            if (requestErrorDetails(err).status === 409) {
                const latest = await conflict('The latest profile was reloaded. Choose the tier again before saving.');
                if (latest) {
                    setTier(latest.profile.tier);
                }
            } else {
                ctx.notifications.error('Could not update account tier', requestErrorMessage(err));
            }
        } finally {
            if (isSelectedCurrent()) setSavingTier(false);
        }
    };

    const changeAvatar = async (file?: File) => {
        if (!selected || !profileEditable || uploadingAvatar) {
            return;
        }
        if (file && file.size > 2 * 1024 * 1024) {
            ctx.notifications.error('Avatar is too large', 'Choose a JPEG, PNG, or WebP image up to 2 MiB.');
            return;
        }
        setUploadingAvatar(true);
        try {
            const profile = file
                ? await services.adminAccounts.uploadAvatar(selected.id, selected.username, file, selected.profile.revision)
                : await services.adminAccounts.deleteAvatar(selected.id, selected.username, selected.profile.revision);
            if (!isSelectedCurrent()) return;
            replaceProfile(selected.id, profile);
            ctx.notifications.success(file ? 'Account avatar updated' : 'Account avatar removed', `@${selected.username}`);
        } catch (err) {
            if (!isSelectedCurrent()) return;
            if (requestErrorDetails(err).status === 409) {
                await conflict('The latest profile is being reloaded. Try the avatar change again.');
            } else {
                ctx.notifications.error(file ? 'Could not upload avatar' : 'Could not remove avatar', requestErrorMessage(err));
            }
        } finally {
            if (isSelectedCurrent()) setUploadingAvatar(false);
        }
    };

    const commitAccess = async () => {
        if (!selected || !accessDraft || !accessDirty) {
            return;
        }
        setSavingAccess(true);
        try {
            const updated = await services.adminAccounts.updateAccess(selected.id, accessDraft);
            if (!isSelectedCurrent()) return;
            setItems(current => current.map(account => (account.id === updated.id ? updated : account)));
            setAccessDraft(cloneAccountAccess(updated.access));
            ctx.notifications.success('Account access updated', `@${updated.username}: ${moduleAccessSummary(updated.access)}.`);
            accounts.reload();
        } catch (err) {
            if (!isSelectedCurrent()) return;
            if (requestErrorDetails(err).status === 409) {
                const latest = await conflict('The authoritative access was reloaded and the stale access draft was discarded.');
                if (latest) {
                    setAccessDraft(cloneAccountAccess(latest.access));
                }
            } else {
                ctx.notifications.error('Could not update account access', requestErrorMessage(err, 'Your access draft has been kept.'));
            }
        } finally {
            if (isSelectedCurrent()) setSavingAccess(false);
        }
    };

    const saveAccess = () => {
        if (!selected || !accessDraft || !accessDirty) {
            return;
        }
        const loginDisabled = selected.access.loginEnabled && !accessDraft.loginEnabled;
        const apiKeyDisabled = selected.access.apiKeyEnabled && !accessDraft.apiKeyEnabled;
        const profitSharingDisabled = selected.access.profitSharingEnabled && !accessDraft.profitSharingEnabled;
        const reduced = accountDataModules.filter(definition => moduleAccessLevel(accessDraft, definition.module) < moduleAccessLevel(selected.access, definition.module));
        const writeGranted = accountDataModules.filter(
            definition =>
                moduleAccessLevel(selected.access, definition.module) < AccountDataAccess.ReadWrite &&
                moduleAccessLevel(accessDraft, definition.module) === AccountDataAccess.ReadWrite
        );
        if (!loginDisabled && !apiKeyDisabled && !profitSharingDisabled && reduced.length === 0 && writeGranted.length === 0) {
            void commitAccess();
            return;
        }
        confirm({
            className: 'admin-operation-confirm',
            autoFocusButton: 'cancel',
            title: `Confirm access changes for @${selected.username}`,
            content: (
                <ul className='account-access-confirmation'>
                    {loginDisabled && <li>Disable sign-in and suspend existing Athena sessions and API keys.</li>}
                    {apiKeyDisabled && <li>Pause every existing API key. Keys are not deleted and resume if API Key access is enabled again.</li>}
                    {profitSharingDisabled && <li>Remove Profit Sharing eligibility and block member requests immediately.</li>}
                    {reduced.length > 0 && <li>Reduce access: {reduced.map(item => item.label).join(', ')}.</li>}
                    {writeGranted.length > 0 && <li>Grant sensitive write access: {writeGranted.map(item => item.label).join(', ')}.</li>}
                </ul>
            ),
            okText: 'Apply changes',
            onOk: commitAccess
        });
    };

    return (
        <AppPage
            title='Account Administration'
            subtitle='Search registered Google or Phantom identities and grant sign-in, API Key, Profit Sharing, and module access.'
            loading={accounts.loading}
            error={accounts.error}
            stale={Boolean(accounts.error && accounts.data)}
            onRefresh={accounts.reload}>
            <div className={mobileAccountId ? 'admin-account-directory-toolbar admin-account-directory-toolbar--hidden' : 'admin-account-directory-toolbar'}>
                <Input.Search
                    allowClear={true}
                    aria-label='Search accounts'
                    value={queryDraft}
                    placeholder='Search username, email, wallet address, display name, or account ID'
                    enterButton={<span>Search</span>}
                    onChange={event => setQueryDraft(event.target.value)}
                    onSearch={searchAccounts}
                />
                <Select
                    aria-label='Account status filter'
                    value={status || 'all'}
                    options={[
                        {value: 'all', label: 'All'},
                        {value: AccountStatus.Pending, label: 'Pending'},
                        {value: AccountStatus.Active, label: 'Active'},
                        {value: AccountStatus.Blocked, label: 'Blocked'}
                    ]}
                    onChange={value => filterAccounts(value === 'all' ? undefined : (value as AccountStatus))}
                />
                <Pagination
                    current={page}
                    pageSize={pageSize}
                    total={accounts.data?.totalSize || 0}
                    showLessItems={true}
                    showSizeChanger={true}
                    pageSizeOptions={[25, 50, 100]}
                    onChange={paginateAccounts}
                />
            </div>
            <div className={mobileAccountId ? 'admin-accounts-mobile-list admin-accounts-mobile-list--hidden' : 'admin-accounts-mobile-list'}>
                <div className='admin-accounts-mobile-list__heading'>
                    <Typography.Title level={2}>Accounts</Typography.Title>
                    <Typography.Text type='secondary'>
                        {accounts.data
                            ? `${accounts.data.totalSize || 0} registered ${accounts.data.totalSize === 1 ? 'identity' : 'identities'}`
                            : accounts.loading
                              ? 'Loading accounts'
                              : 'Accounts unavailable'}
                    </Typography.Text>
                </div>
                <div className='admin-accounts-mobile-list__items'>
                    {items.length > 0 ? (
                        items.map(account => (
                            <button key={account.id} type='button' className='admin-accounts-mobile-card' onClick={() => chooseAccount(account.id, true)}>
                                <AccountAvatar profile={account.profile} username={account.username} size={44} />
                                <span>
                                    <strong>{accountPrimaryLabel(account)}</strong>
                                    <small>{accountIdentityValue(account) || 'Identity unavailable'}</small>
                                    <small>@{account.username}</small>
                                    <small>
                                        Created {identityListTime(account.identity.createdAt)} · Last login {identityListTime(account.identity.lastLoginAt)}
                                    </small>
                                </span>
                                <span className='admin-accounts-mobile-card__meta'>
                                    <Tag>{identityProviderLabel(account.identity.provider)}</Tag>
                                    {accountStatus(account)}
                                </span>
                            </button>
                        ))
                    ) : accounts.data && !accounts.loading ? (
                        <Empty description='No accounts match these filters' />
                    ) : null}
                </div>
            </div>
            <div className={mobileAccountId ? 'admin-accounts-layout admin-accounts-layout--mobile-detail' : 'admin-accounts-layout'}>
                <aside className='admin-account-list' aria-label='Athena accounts'>
                    <div className='admin-account-list__heading'>
                        <strong>Accounts</strong>
                        <span>{accounts.data ? accounts.data.totalSize || 0 : '—'}</span>
                    </div>
                    <div className='admin-account-list__items'>
                        {items.length > 0 ? (
                            items.map(account => (
                                <button
                                    key={account.id}
                                    type='button'
                                    className={account.id === selectedId ? 'admin-account-list__item admin-account-list__item--active' : 'admin-account-list__item'}
                                    aria-current={account.id === selectedId ? 'true' : undefined}
                                    onClick={() => chooseAccount(account.id, Boolean(mobileAccountId))}>
                                    <AccountAvatar profile={account.profile} username={account.username} size={38} />
                                    <span>
                                        <strong>{accountPrimaryLabel(account)}</strong>
                                        <small>{accountIdentityValue(account) || 'Identity unavailable'}</small>
                                        <small>@{account.username}</small>
                                        <small>
                                            {identityProviderLabel(account.identity.provider)} · Created {identityListTime(account.identity.createdAt)}
                                        </small>
                                        <small>Last login {identityListTime(account.identity.lastLoginAt)}</small>
                                    </span>
                                    {accountStatus(account)}
                                </button>
                            ))
                        ) : accounts.data && !accounts.loading ? (
                            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='No accounts' />
                        ) : null}
                    </div>
                </aside>
                <div className='admin-account-detail'>
                    <Button className='admin-account-detail__mobile-back' type='text' icon={<ArrowLeftOutlined aria-hidden />} onClick={() => updateSearch({account: ''})}>
                        Back to accounts
                    </Button>
                    {!selected ? (
                        <Alert type='info' showIcon={true} title='Select an account to manage' />
                    ) : (
                        <>
                            <div className='admin-account-detail__heading'>
                                <AccountAvatar profile={selected.profile} username={selected.username} size={56} />
                                <div>
                                    <Typography.Title level={2}>{accountPrimaryLabel(selected)}</Typography.Title>
                                    <Space size={6} wrap={true}>
                                        <Typography.Text type='secondary'>@{selected.username}</Typography.Text>
                                        {accountIdentityValue(selected) && (
                                            <Typography.Text className='account-identity-value' type='secondary'>
                                                {accountIdentityValue(selected)}
                                            </Typography.Text>
                                        )}
                                        {accountStatus(selected)}
                                        {selected.administrator && <Tag color='warning'>Administrator</Tag>}
                                    </Space>
                                </div>
                            </div>
                            {!profileEditable && (
                                <Alert className='admin-account-self-notice' type='info' showIcon={true} title='Manage your own profile and security in Account Center' />
                            )}
                            <Tabs
                                defaultActiveKey='access'
                                items={[
                                    {
                                        key: 'access',
                                        label: 'Access',
                                        forceRender: true,
                                        children: (
                                            <Section title='Account access'>
                                                <AccountAccessEditor
                                                    account={selected}
                                                    access={accessDraft || selected.access}
                                                    editable={accessEditable}
                                                    dirty={accessDirty}
                                                    updating={savingAccess}
                                                    onChange={setAccessDraft}
                                                    onSave={saveAccess}
                                                    onReset={() => setAccessDraft(cloneAccountAccess(selected.access))}
                                                />
                                            </Section>
                                        )
                                    },
                                    {
                                        key: 'profile',
                                        label: 'Profile',
                                        forceRender: true,
                                        children: (
                                            <Section title='Profile & tier'>
                                                <div className='admin-account-profile-grid'>
                                                    <div className='admin-account-avatar-editor'>
                                                        <AccountAvatar profile={selected.profile} username={selected.username} size={80} />
                                                        <Space wrap={true}>
                                                            <Upload
                                                                accept='image/jpeg,image/png,image/webp'
                                                                disabled={!profileEditable || uploadingAvatar}
                                                                showUploadList={false}
                                                                beforeUpload={file => {
                                                                    void changeAvatar(file as File);
                                                                    return Upload.LIST_IGNORE;
                                                                }}>
                                                                <Button
                                                                    icon={<UploadOutlined aria-hidden />}
                                                                    loading={uploadingAvatar}
                                                                    disabled={!profileEditable || uploadingAvatar}>
                                                                    Upload
                                                                </Button>
                                                            </Upload>
                                                            {selected.profile.avatarUrl && (
                                                                <Button
                                                                    danger={true}
                                                                    icon={<DeleteOutlined aria-hidden />}
                                                                    loading={uploadingAvatar}
                                                                    disabled={!profileEditable || uploadingAvatar}
                                                                    onClick={() => void changeAvatar()}>
                                                                    Remove
                                                                </Button>
                                                            )}
                                                        </Space>
                                                    </div>
                                                    <Form layout='vertical'>
                                                        <Form.Item label='Username' htmlFor='admin-account-username'>
                                                            <Input id='admin-account-username' value={`@${selected.username}`} readOnly={true} />
                                                        </Form.Item>
                                                        <Form.Item
                                                            label='Display name'
                                                            htmlFor='admin-account-display-name'
                                                            validateStatus={!displayNameValid ? 'error' : undefined}>
                                                            <Input
                                                                id='admin-account-display-name'
                                                                value={displayName}
                                                                showCount={{formatter: info => `${unicodeCharacterCount(info.value)}/80`}}
                                                                disabled={!profileEditable}
                                                                onChange={event => setDisplayName(event.target.value)}
                                                            />
                                                        </Form.Item>
                                                        <div className='admin-account-field-actions'>
                                                            <Typography.Text type={displayNameDirty ? 'warning' : 'secondary'}>
                                                                Revision {selected.profile.revision}
                                                            </Typography.Text>
                                                            <Button
                                                                type='primary'
                                                                icon={<SaveOutlined aria-hidden />}
                                                                loading={savingProfile}
                                                                disabled={!profileEditable || !displayNameDirty || !displayNameValid}
                                                                onClick={() => void saveDisplayName()}>
                                                                Save name
                                                            </Button>
                                                        </div>
                                                        <Form.Item label='Presentation tier'>
                                                            <ChoiceGroup<AccountTier>
                                                                ariaLabel={`Presentation tier for @${selected.username}`}
                                                                value={tier}
                                                                disabled={!profileEditable || savingTier}
                                                                options={[
                                                                    {value: AccountTier.Standard, label: 'Standard'},
                                                                    {value: AccountTier.Pro, label: 'Pro'}
                                                                ]}
                                                                onChange={setTier}
                                                            />
                                                        </Form.Item>
                                                        <div className='admin-account-field-actions'>
                                                            <Typography.Text type='secondary'>Display only; does not grant access.</Typography.Text>
                                                            <Button
                                                                icon={<CheckOutlined aria-hidden />}
                                                                loading={savingTier}
                                                                disabled={!profileEditable || !tierDirty || displayNameDirty}
                                                                onClick={() => void saveTier()}>
                                                                Apply tier
                                                            </Button>
                                                        </div>
                                                    </Form>
                                                </div>
                                            </Section>
                                        )
                                    },
                                    {
                                        key: 'identity',
                                        label: 'Identity',
                                        forceRender: true,
                                        children: (
                                            <Section title='Identity'>
                                                <OperationFacts
                                                    items={[
                                                        {label: 'Provider', value: identityProviderLabel(selected.identity.provider)},
                                                        {
                                                            label: identityPresentation(selected.identity).label,
                                                            value: (
                                                                <Typography.Text className='account-identity-value' copyable={Boolean(accountIdentityValue(selected))}>
                                                                    {accountIdentityValue(selected) || 'Unavailable'}
                                                                </Typography.Text>
                                                            )
                                                        },
                                                        {label: 'Technical account ID', value: <Typography.Text copyable={true}>{selected.id}</Typography.Text>},
                                                        {label: 'Status', value: accountStatus(selected)},
                                                        {label: 'Created', value: identityTime(selected.identity.createdAt)},
                                                        {label: 'Last login', value: identityTime(selected.identity.lastLoginAt)}
                                                    ]}
                                                />
                                            </Section>
                                        )
                                    }
                                ]}
                            />
                        </>
                    )}
                </div>
            </div>
        </AppPage>
    );
};
