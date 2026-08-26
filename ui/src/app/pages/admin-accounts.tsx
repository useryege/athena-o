import {ArrowLeftOutlined, CheckOutlined, DeleteOutlined, SaveOutlined, UploadOutlined} from '@ant-design/icons';
import {Alert, Button, Empty, Form, Input, Pagination, Select, Space, Switch, Tag, Typography, Upload} from 'antd';
import * as React from 'react';
import {useBlocker, useSearchParams} from 'react-router-dom';
import {AppPage, ChoiceGroup, KeyValueGrid, Section, StatusTag, useAsyncData} from '../components';
import {accountAccessEqual, cloneAccountAccess, moduleAccessLevel, moduleAccessSummary, replaceModuleAccess} from '../shared/account-access';
import {AccountDataAccess, AccountDataModuleDefinition, accountDataAccessLabel, accountDataModuleGroups, accountDataModules} from '../shared/access-modules';
import {Context, useAuthorization} from '../shared/context';
import {accountStatusForAccess, Account, AccountAccess, AccountIdentityProvider, AccountProfile, AccountStatus, AccountTier} from '../shared/models';
import {services} from '../shared/services';
import {AccountsPage} from '../shared/services/accounts-service';
import {requestErrorDetails, requestErrorMessage} from '../shared/services/requests';
import {AccountAvatar, accountTierLabel, hasControlCharacters, unicodeCharacterCount} from './account-center';

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

const identityProviderLabel = (provider: AccountIdentityProvider) => (provider === AccountIdentityProvider.Google ? 'Google' : 'Unavailable');
const identityTime = (value: number) => (value > 0 ? new Date(value * 1000).toLocaleString() : 'Not yet');
const identityListTime = (value: number) => (value > 0 ? new Date(value * 1000).toLocaleDateString() : 'Not yet');

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
    onReset: () => void;
}) => (
    <div className='account-access-editor' aria-busy={props.updating || undefined}>
        <div className='account-access-editor__entitlements'>
            {[
                {
                    key: 'login',
                    label: 'Google sign-in',
                    description: 'Controls new Google sign-ins and all existing Athena sessions and API keys.',
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
                        <Switch
                            aria-label={`${item.label} for ${props.account.name}`}
                            checked={item.checked}
                            disabled={props.updating}
                            checkedChildren='Enabled'
                            unCheckedChildren='Disabled'
                            onChange={item.onChange}
                        />
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
                                                {props.account.administrator ? accountDataAccessLabel(definition.maxAccess) : accountDataAccessLabel(value)}
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
                    <Button type='primary' icon={<SaveOutlined />} loading={props.updating} disabled={!props.dirty || props.updating} onClick={props.onSave}>
                        Save access
                    </Button>
                </Space>
            )}
        </div>
    </div>
);

export const AdminAccountsPage = () => {
    const ctx = React.useContext(Context);
    const authorization = useAuthorization();
    const [searchParams, setSearchParams] = useSearchParams();
    const mobileAccountName = searchParams.get('account') || '';
    const [queryDraft, setQueryDraft] = React.useState('');
    const [query, setQuery] = React.useState('');
    const [status, setStatus] = React.useState<AccountStatus | undefined>();
    const [page, setPage] = React.useState(1);
    const [pageSize, setPageSize] = React.useState(50);
    const accounts = useAsyncData<AccountsPage>(
        () => services.accounts.list({query, status, page, pageSize}) as Promise<AccountsPage> & {abort?: () => void},
        [query, status, page, pageSize]
    );
    const [items, setItems] = React.useState<Account[]>([]);
    const [selectedName, setSelectedName] = React.useState('');
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
        setSelectedName(current => {
            if (mobileAccountName && accounts.data?.items.some(account => account.name === mobileAccountName)) {
                return mobileAccountName;
            }
            return current && accounts.data?.items.some(account => account.name === current) ? current : accounts.data?.items[0]?.name || '';
        });
    }, [accounts.data, mobileAccountName]);

    React.useEffect(() => {
        if (accounts.data && mobileAccountName && !accounts.data.items.some(account => account.name === mobileAccountName)) {
            setSearchParams({}, {replace: true});
        }
    }, [accounts.data, mobileAccountName, setSearchParams]);

    const selected = items.find(account => account.name === selectedName);
    const profileEditable = Boolean(selected && selected.name !== authorization.user.username);
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
    }, [selected?.name]);

    const changeListScope = (change: () => void) => {
        if (!dirty) {
            change();
            return;
        }
        ctx.modal.confirm({
            title: `Discard changes for ${selectedName}?`,
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
            setPage(1);
        });

    const filterAccounts = (nextStatus?: AccountStatus) =>
        changeListScope(() => {
            setStatus(nextStatus);
            setPage(1);
        });

    const paginateAccounts = (nextPage: number, nextPageSize: number) =>
        changeListScope(() => {
            setPage(nextPageSize === pageSize ? nextPage : 1);
            setPageSize(nextPageSize);
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
        const handle = ctx.modal.confirm({
            title: 'Discard unsaved account changes?',
            content: selected ? `Changes for ${selected.name} have not been saved.` : undefined,
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

    const chooseAccount = (name: string, openMobileDetail = false) => {
        const choose = () => {
            setSelectedName(name);
            if (openMobileDetail) {
                setSearchParams({account: name});
            }
        };
        if (!dirty) {
            choose();
            return;
        }
        ctx.modal.confirm({
            title: `Discard changes for ${selectedName}?`,
            content: `Open ${name} after discarding the current drafts.`,
            okText: 'Discard and switch',
            onOk: choose
        });
    };

    const replaceProfile = (name: string, profile: AccountProfile) => {
        setItems(current => current.map(account => (account.name === name ? {...account, profile} : account)));
    };

    const conflict = async (message: string) => {
        if (!selected) {
            return undefined;
        }
        const latest = await services.accounts.get(selected.name);
        setItems(current => current.map(account => (account.name === latest.name ? latest : account)));
        ctx.notifications.warning('Account changed elsewhere', message);
        return latest;
    };

    const saveDisplayName = async () => {
        if (!selected || !profileEditable || !displayNameDirty || !displayNameValid) {
            return;
        }
        setSavingProfile(true);
        try {
            const profile = await services.accounts.updateProfile(selected.name, displayName.trim(), selected.profile.revision);
            replaceProfile(selected.name, profile);
            setDisplayName(profile.displayName);
            ctx.notifications.success('Account profile updated', selected.name);
        } catch (err) {
            if (requestErrorDetails(err).status === 409) {
                await conflict('The latest profile was reloaded. Your display-name draft was kept; review it before saving again.');
            } else {
                ctx.notifications.error('Could not update account profile', requestErrorMessage(err));
            }
        } finally {
            setSavingProfile(false);
        }
    };

    const saveTier = async () => {
        if (!selected || !profileEditable || !tierDirty) {
            return;
        }
        setSavingTier(true);
        try {
            const profile = await services.accounts.updateTier(selected.name, tier, selected.profile.revision);
            replaceProfile(selected.name, profile);
            setTier(profile.tier);
            ctx.notifications.success('Account tier updated', `${selected.name} is now ${accountTierLabel(profile.tier)}.`);
        } catch (err) {
            if (requestErrorDetails(err).status === 409) {
                const latest = await conflict('The latest profile was reloaded. Choose the tier again before saving.');
                if (latest) {
                    setTier(latest.profile.tier);
                }
            } else {
                ctx.notifications.error('Could not update account tier', requestErrorMessage(err));
            }
        } finally {
            setSavingTier(false);
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
                ? await services.accounts.uploadAvatar(selected.name, file, selected.profile.revision)
                : await services.accounts.deleteAvatar(selected.name, selected.profile.revision);
            replaceProfile(selected.name, profile);
            ctx.notifications.success(file ? 'Account avatar updated' : 'Account avatar removed', selected.name);
        } catch (err) {
            if (requestErrorDetails(err).status === 409) {
                await conflict('The latest profile is being reloaded. Try the avatar change again.');
            } else {
                ctx.notifications.error(file ? 'Could not upload avatar' : 'Could not remove avatar', requestErrorMessage(err));
            }
        } finally {
            setUploadingAvatar(false);
        }
    };

    const commitAccess = async () => {
        if (!selected || !accessDraft || !accessDirty) {
            return;
        }
        setSavingAccess(true);
        try {
            const updated = await services.accounts.updateAccess(selected.name, accessDraft);
            setItems(current => current.map(account => (account.name === updated.name ? updated : account)));
            setAccessDraft(cloneAccountAccess(updated.access));
            ctx.notifications.success('Account access updated', `${updated.name}: ${moduleAccessSummary(updated.access)}.`);
            accounts.reload();
        } catch (err) {
            if (requestErrorDetails(err).status === 409) {
                const latest = await conflict('The authoritative access was reloaded and the stale access draft was discarded.');
                if (latest) {
                    setAccessDraft(cloneAccountAccess(latest.access));
                }
            } else {
                ctx.notifications.error('Could not update account access', requestErrorMessage(err, 'Your access draft has been kept.'));
            }
        } finally {
            setSavingAccess(false);
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
        ctx.modal.confirm({
            title: `Confirm access changes for ${selected.name}`,
            content: (
                <ul className='account-access-confirmation'>
                    {loginDisabled && <li>Disable Google sign-in and suspend existing Athena sessions and API keys.</li>}
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
            subtitle='Search registered Google identities and grant sign-in, API Key, Profit Sharing, and module access.'
            loading={accounts.loading}
            error={accounts.error}
            onRefresh={accounts.reload}>
            <div className={mobileAccountName ? 'admin-account-directory-toolbar admin-account-directory-toolbar--hidden' : 'admin-account-directory-toolbar'}>
                <Input.Search
                    allowClear={true}
                    aria-label='Search accounts'
                    value={queryDraft}
                    placeholder='Search email, display name, or account ID'
                    enterButton='Search'
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
            <div className={mobileAccountName ? 'admin-accounts-mobile-list admin-accounts-mobile-list--hidden' : 'admin-accounts-mobile-list'}>
                <div className='admin-accounts-mobile-list__heading'>
                    <Typography.Title level={2}>Accounts</Typography.Title>
                    <Typography.Text type='secondary'>
                        {accounts.data?.totalSize || 0} registered {accounts.data?.totalSize === 1 ? 'identity' : 'identities'}
                    </Typography.Text>
                </div>
                <div className='admin-accounts-mobile-list__items'>
                    {items.length > 0 ? (
                        items.map(account => (
                            <button key={account.name} type='button' className='admin-accounts-mobile-card' onClick={() => chooseAccount(account.name, true)}>
                                <AccountAvatar profile={account.profile} username={account.name} size={44} />
                                <span>
                                    <strong>{account.profile.displayName || account.identity.verifiedEmail || account.name}</strong>
                                    <small>{account.identity.verifiedEmail || 'Email unavailable'}</small>
                                    <small>@{account.name}</small>
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
                    ) : (
                        <Empty description='No accounts match these filters' />
                    )}
                </div>
            </div>
            <div className={mobileAccountName ? 'admin-accounts-layout admin-accounts-layout--mobile-detail' : 'admin-accounts-layout'}>
                <aside className='admin-account-list' aria-label='Athena accounts'>
                    <div className='admin-account-list__heading'>
                        <strong>Accounts</strong>
                        <span>{accounts.data?.totalSize || 0}</span>
                    </div>
                    <div className='admin-account-list__items'>
                        {items.length > 0 ? (
                            items.map(account => (
                                <button
                                    key={account.name}
                                    type='button'
                                    className={account.name === selectedName ? 'admin-account-list__item admin-account-list__item--active' : 'admin-account-list__item'}
                                    aria-current={account.name === selectedName ? 'true' : undefined}
                                    onClick={() => chooseAccount(account.name, Boolean(mobileAccountName))}>
                                    <AccountAvatar profile={account.profile} username={account.name} size={38} />
                                    <span>
                                        <strong>{account.profile.displayName || account.identity.verifiedEmail || account.name}</strong>
                                        <small>{account.identity.verifiedEmail || 'Email unavailable'}</small>
                                        <small>@{account.name}</small>
                                        <small>
                                            {identityProviderLabel(account.identity.provider)} · Created {identityListTime(account.identity.createdAt)}
                                        </small>
                                        <small>Last login {identityListTime(account.identity.lastLoginAt)}</small>
                                    </span>
                                    {accountStatus(account)}
                                </button>
                            ))
                        ) : (
                            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='No accounts' />
                        )}
                    </div>
                </aside>
                <div className='admin-account-detail'>
                    <Button className='admin-account-detail__mobile-back' type='text' icon={<ArrowLeftOutlined />} onClick={() => setSearchParams({})}>
                        Back to accounts
                    </Button>
                    {!selected ? (
                        <Alert type='info' showIcon={true} title='Select an account to manage' />
                    ) : (
                        <>
                            <div className='admin-account-detail__heading'>
                                <AccountAvatar profile={selected.profile} username={selected.name} size={56} />
                                <div>
                                    <Typography.Title level={2}>{selected.profile.displayName || selected.identity.verifiedEmail || selected.name}</Typography.Title>
                                    <Space size={6} wrap={true}>
                                        <Typography.Text type='secondary'>@{selected.name}</Typography.Text>
                                        {selected.identity.verifiedEmail && <Typography.Text type='secondary'>{selected.identity.verifiedEmail}</Typography.Text>}
                                        {accountStatus(selected)}
                                        {selected.administrator && <Tag color='gold'>Administrator</Tag>}
                                    </Space>
                                </div>
                            </div>
                            {!profileEditable && (
                                <Alert className='admin-account-self-notice' type='info' showIcon={true} title='Manage your own profile and security in Account Center' />
                            )}
                            <Section title='Google identity'>
                                <KeyValueGrid
                                    items={[
                                        {label: 'Provider', value: identityProviderLabel(selected.identity.provider)},
                                        {label: 'Verified email', value: selected.identity.verifiedEmail || 'Unavailable'},
                                        {label: 'Internal account ID', value: selected.name},
                                        {label: 'Status', value: accountStatus(selected)},
                                        {label: 'Created', value: identityTime(selected.identity.createdAt)},
                                        {label: 'Last login', value: identityTime(selected.identity.lastLoginAt)}
                                    ]}
                                />
                            </Section>
                            <Section title='Profile & tier'>
                                <div className='admin-account-profile-grid'>
                                    <div className='admin-account-avatar-editor'>
                                        <AccountAvatar profile={selected.profile} username={selected.name} size={80} />
                                        <Space wrap={true}>
                                            <Upload
                                                accept='image/jpeg,image/png,image/webp'
                                                disabled={!profileEditable || uploadingAvatar}
                                                showUploadList={false}
                                                beforeUpload={file => {
                                                    void changeAvatar(file as File);
                                                    return Upload.LIST_IGNORE;
                                                }}>
                                                <Button icon={<UploadOutlined />} loading={uploadingAvatar} disabled={!profileEditable || uploadingAvatar}>
                                                    Upload
                                                </Button>
                                            </Upload>
                                            {selected.profile.avatarUrl && (
                                                <Button
                                                    danger={true}
                                                    icon={<DeleteOutlined />}
                                                    loading={uploadingAvatar}
                                                    disabled={!profileEditable || uploadingAvatar}
                                                    onClick={() => void changeAvatar()}>
                                                    Remove
                                                </Button>
                                            )}
                                        </Space>
                                    </div>
                                    <Form layout='vertical'>
                                        <Form.Item label='Username'>
                                            <Input value={selected.name} disabled={true} />
                                        </Form.Item>
                                        <Form.Item label='Display name' validateStatus={!displayNameValid ? 'error' : undefined}>
                                            <Input
                                                value={displayName}
                                                showCount={{formatter: info => `${unicodeCharacterCount(info.value)}/80`}}
                                                disabled={!profileEditable}
                                                onChange={event => setDisplayName(event.target.value)}
                                            />
                                        </Form.Item>
                                        <div className='admin-account-field-actions'>
                                            <Typography.Text type={displayNameDirty ? 'warning' : 'secondary'}>Revision {selected.profile.revision}</Typography.Text>
                                            <Button
                                                type='primary'
                                                icon={<SaveOutlined />}
                                                loading={savingProfile}
                                                disabled={!profileEditable || !displayNameDirty || !displayNameValid}
                                                onClick={() => void saveDisplayName()}>
                                                Save name
                                            </Button>
                                        </div>
                                        <Form.Item label='Presentation tier'>
                                            <ChoiceGroup<AccountTier>
                                                ariaLabel={`Presentation tier for ${selected.name}`}
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
                                                icon={<CheckOutlined />}
                                                loading={savingTier}
                                                disabled={!profileEditable || !tierDirty || displayNameDirty}
                                                onClick={() => void saveTier()}>
                                                Apply tier
                                            </Button>
                                        </div>
                                    </Form>
                                </div>
                            </Section>
                            <Section title='Authorization & module access'>
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
                        </>
                    )}
                </div>
            </div>
        </AppPage>
    );
};
