import {ArrowLeftOutlined, CheckOutlined, DeleteOutlined, SaveOutlined, UploadOutlined} from '@ant-design/icons';
import {Alert, Button, Form, Input, Space, Switch, Tag, Typography, Upload} from 'antd';
import * as React from 'react';
import {useBlocker, useSearchParams} from 'react-router-dom';
import {AppPage, ChoiceGroup, Section, StatusTag, useAsyncData} from '../components';
import {accountAccessEqual, cloneAccountAccess, moduleAccessLevel, moduleAccessSummary, replaceModuleAccess} from '../shared/account-access';
import {AccountDataAccess, AccountDataModuleDefinition, accountDataAccessLabel, accountDataModuleGroups, accountDataModules} from '../shared/access-modules';
import {Context, useAuthorization} from '../shared/context';
import {Account, AccountAccess, AccountProfile, AccountTier} from '../shared/models';
import {services} from '../shared/services';
import {requestErrorDetails, requestErrorMessage} from '../shared/services/requests';
import {AccountAvatar, accountTierLabel, hasControlCharacters, unicodeCharacterCount} from './account-center';

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
    onReset: () => void;
}) => (
    <div className='account-access-editor' aria-busy={props.updating || undefined}>
        <div className='account-access-editor__login'>
            <div>
                <Typography.Text strong={true}>Google sign-in access</Typography.Text>
                <Typography.Paragraph type='secondary'>Controls new Google sign-ins and use of existing Athena sessions and API keys for this account.</Typography.Paragraph>
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
    const accounts = useAsyncData<Account[]>(() => services.accounts.list() as Promise<Account[]> & {abort?: () => void}, []);
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
        setItems(accounts.data);
        setSelectedName(current => {
            if (mobileAccountName && accounts.data?.some(account => account.name === mobileAccountName)) {
                return mobileAccountName;
            }
            return current && accounts.data?.some(account => account.name === current) ? current : accounts.data?.[0]?.name || '';
        });
    }, [accounts.data, mobileAccountName]);

    React.useEffect(() => {
        if (accounts.data && mobileAccountName && !accounts.data.some(account => account.name === mobileAccountName)) {
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
        const reduced = accountDataModules.filter(definition => moduleAccessLevel(accessDraft, definition.module) < moduleAccessLevel(selected.access, definition.module));
        const writeGranted = accountDataModules.filter(
            definition =>
                moduleAccessLevel(selected.access, definition.module) < AccountDataAccess.ReadWrite &&
                moduleAccessLevel(accessDraft, definition.module) === AccountDataAccess.ReadWrite
        );
        if (!loginDisabled && reduced.length === 0 && writeGranted.length === 0) {
            void commitAccess();
            return;
        }
        ctx.modal.confirm({
            title: `Confirm access changes for ${selected.name}`,
            content: (
                <ul className='account-access-confirmation'>
                    {loginDisabled && <li>Disable Google sign-in and suspend existing Athena sessions and API keys.</li>}
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
            subtitle='Manage account profiles, presentation tiers, and effective module access.'
            loading={accounts.loading}
            error={accounts.error}
            onRefresh={accounts.reload}>
            <div className={mobileAccountName ? 'admin-accounts-mobile-list admin-accounts-mobile-list--hidden' : 'admin-accounts-mobile-list'}>
                <div className='admin-accounts-mobile-list__heading'>
                    <Typography.Title level={2}>Accounts</Typography.Title>
                    <Typography.Text type='secondary'>{items.length} configured identities</Typography.Text>
                </div>
                <div className='admin-accounts-mobile-list__items'>
                    {items.map(account => (
                        <button key={account.name} type='button' className='admin-accounts-mobile-card' onClick={() => chooseAccount(account.name, true)}>
                            <AccountAvatar profile={account.profile} username={account.name} size={44} />
                            <span>
                                <strong>{account.profile.displayName || account.name}</strong>
                                <small>@{account.name}</small>
                            </span>
                            <span className='admin-accounts-mobile-card__meta'>
                                <Tag>{accountTierLabel(account.profile.tier)}</Tag>
                                {accountStatus(account)}
                            </span>
                        </button>
                    ))}
                </div>
            </div>
            <div className={mobileAccountName ? 'admin-accounts-layout admin-accounts-layout--mobile-detail' : 'admin-accounts-layout'}>
                <aside className='admin-account-list' aria-label='Athena accounts'>
                    <div className='admin-account-list__heading'>
                        <strong>Accounts</strong>
                        <span>{items.length}</span>
                    </div>
                    <div className='admin-account-list__items'>
                        {items.map(account => (
                            <button
                                key={account.name}
                                type='button'
                                className={account.name === selectedName ? 'admin-account-list__item admin-account-list__item--active' : 'admin-account-list__item'}
                                aria-current={account.name === selectedName ? 'true' : undefined}
                                onClick={() => chooseAccount(account.name, Boolean(mobileAccountName))}>
                                <AccountAvatar profile={account.profile} username={account.name} size={38} />
                                <span>
                                    <strong>{account.profile.displayName || account.name}</strong>
                                    <small>@{account.name}</small>
                                </span>
                                <Tag>{accountTierLabel(account.profile.tier)}</Tag>
                            </button>
                        ))}
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
                                    <Typography.Title level={2}>{selected.profile.displayName || selected.name}</Typography.Title>
                                    <Space size={6} wrap={true}>
                                        <Typography.Text type='secondary'>@{selected.name}</Typography.Text>
                                        {accountStatus(selected)}
                                        {selected.administrator && <Tag color='gold'>Administrator</Tag>}
                                    </Space>
                                </div>
                            </div>
                            {!profileEditable && (
                                <Alert className='admin-account-self-notice' type='info' showIcon={true} title='Manage your own profile and security in Account Center' />
                            )}
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
                            <Section title='Google sign-in & module access'>
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
