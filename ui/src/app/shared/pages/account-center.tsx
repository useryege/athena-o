import {ClockCircleOutlined, DeleteOutlined, KeyOutlined, LogoutOutlined, ReloadOutlined, SafetyCertificateOutlined, UploadOutlined, UserOutlined} from '@ant-design/icons';
import {Button, Form, Input, Select, Space, Tag, Typography, Upload} from 'antd';
import * as React from 'react';
import {useBlocker, useNavigate} from 'react-router-dom';
import {AppPage, Section, StatusTag, useAsyncData} from '../../components';
import {moduleAccessSummary} from '../account-access';
import {accountAccessDisplayModules, accountDataAccessLabel} from '../access-modules';
import {Context, useAuthorization} from '../context';
import {accountStatusForAccess, AccountStatus} from '../models';
import {AccountAvatar, accountTierLabel, identityPresentation, identityProviderLabel} from '../account-presentation';
import {selfAccountServices as services} from '../../session/services';
import {requestErrorDetails, requestErrorMessage} from '../services/requests';
import {hasControlCharacters, unicodeCharacterCount} from '../validation';
import {boolTag} from './shared';

export type AccountCenterSection = 'profile' | 'security' | 'access';

const accountSections: Array<{key: AccountCenterSection; label: string; description: string; icon: React.ReactNode}> = [
    {key: 'profile', label: 'Profile', description: 'Name and avatar', icon: <UserOutlined aria-hidden={true} />},
    {key: 'access', label: 'Access & session', description: 'Permissions and versions', icon: <KeyOutlined aria-hidden={true} />}
];
const memberSecuritySection = {key: 'security' as const, label: 'Security', description: 'API keys and AI connections', icon: <SafetyCertificateOutlined aria-hidden={true} />};

const useUnsavedChanges = (dirty: boolean, reset: () => void, savedName: string, draftName: string) => {
    const ctx = React.useContext(Context);
    const blocker = useBlocker(dirty);

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
            title: 'Discard unsaved profile changes?',
            className: 'account-profile-leave-confirm',
            focusable: {trap: true, focusTriggerAfterClose: true},
            wrapProps: {
                onKeyDownCapture: event => {
                    if (event.key !== 'Tab') return;
                    const buttons = Array.from(event.currentTarget.querySelectorAll<HTMLButtonElement>('button:not([disabled])'));
                    const first = buttons[0];
                    const last = buttons[buttons.length - 1];
                    if (first && last && ((!event.shiftKey && document.activeElement === last) || (event.shiftKey && document.activeElement === first))) {
                        event.preventDefault();
                        (event.shiftKey ? last : first).focus();
                    }
                }
            },
            content: (
                <>
                    <p>Your current changes have not been saved.</p>
                    <dl className='account-draft-comparison'>
                        <div>
                            <dt>Saved display name</dt>
                            <dd>{savedName}</dd>
                        </div>
                        <div>
                            <dt>Your draft</dt>
                            <dd>{draftName}</dd>
                        </div>
                    </dl>
                </>
            ),
            cancelText: 'Keep editing',
            autoFocusButton: 'cancel',
            okButtonProps: {danger: true},
            okText: 'Discard and leave',
            onOk: () => {
                resolved = true;
                reset();
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
    }, [blocker.state, ctx.modal, savedName, draftName, reset]);
};

export const AccountCenterLayout = (props: {active: AccountCenterSection; showMemberSecurity?: boolean; children: React.ReactNode}) => {
    const navigate = useNavigate();
    const authorization = useAuthorization();
    const profile = authorization.user.profile;
    const visibleSections =
        props.showMemberSecurity && authorization.user.access.apiKeyEnabled && !authorization.isAdmin
            ? [...accountSections.slice(0, 1), memberSecuritySection, ...accountSections.slice(1)]
            : accountSections;
    return (
        <AppPage
            title='Account Center'
            subtitle={props.showMemberSecurity ? 'Manage your Athena identity, API keys, and current access.' : 'Manage your Athena identity and current access.'}>
            {props.active !== 'profile' && (
                <div className='account-center-hero'>
                    <AccountAvatar className='account-identity-avatar' profile={profile} username={authorization.user.username} size={36} />
                    <div className='account-center-hero__copy'>
                        <Typography.Title level={2}>{profile.displayName || authorization.user.username}</Typography.Title>
                        <Typography.Text type='secondary'>@{authorization.user.username}</Typography.Text>
                        <Space size={6} wrap={true}>
                            <Tag>{accountTierLabel(profile.tier)}</Tag>
                            <Tag color={authorization.isAdmin ? 'success' : 'default'}>{authorization.isAdmin ? 'Administrator' : 'Member'}</Tag>
                        </Space>
                    </div>
                </div>
            )}
            <div className='account-center-mobile-select'>
                <label htmlFor='account-center-section'>Account section</label>
                <Select
                    id='account-center-section'
                    value={props.active}
                    options={visibleSections.map(section => ({value: section.key, label: section.label}))}
                    onChange={section => navigate(`/account/${section}`)}
                    getPopupContainer={trigger => trigger.parentElement || document.body}
                />
            </div>
            <div className='account-center-layout'>
                <nav className='account-center-nav' aria-label='Account Center sections'>
                    {visibleSections.map(section => (
                        <button
                            key={section.key}
                            type='button'
                            className={props.active === section.key ? 'account-center-nav__item account-center-nav__item--active' : 'account-center-nav__item'}
                            aria-current={props.active === section.key ? 'page' : undefined}
                            onClick={() => navigate(`/account/${section.key}`)}>
                            <span className='account-center-nav__icon'>{section.icon}</span>
                            <span>
                                <strong>{section.label}</strong>
                                <small>{section.description}</small>
                            </span>
                        </button>
                    ))}
                </nav>
                <div className='account-center-content'>{props.children}</div>
            </div>
        </AppPage>
    );
};

const ProfilePage = () => {
    const authorization = useAuthorization();
    const ctx = React.useContext(Context);
    const username = authorization.user.username;
    const accountId = authorization.user.accountId;
    const mountedRef = React.useRef(true);
    React.useLayoutEffect(() => {
        mountedRef.current = true;
        return () => {
            mountedRef.current = false;
        };
    }, []);
    const [profile, setProfile] = React.useState(authorization.user.profile);
    const [displayName, setDisplayName] = React.useState(profile.displayName);
    const [saving, setSaving] = React.useState(false);
    const [uploading, setUploading] = React.useState(false);
    const normalizedDisplayName = displayName.trim();
    const displayNameLength = unicodeCharacterCount(normalizedDisplayName);
    const displayNameHasControls = hasControlCharacters(normalizedDisplayName);
    const displayNameValid = displayNameLength >= 1 && displayNameLength <= 80 && !displayNameHasControls;
    const dirty = normalizedDisplayName !== profile.displayName;

    React.useEffect(() => {
        setProfile(authorization.user.profile);
        if (!dirty) {
            setDisplayName(authorization.user.profile.displayName);
        }
    }, [authorization.user.profile.revision]);

    const reset = React.useCallback(() => setDisplayName(profile.displayName), [profile.displayName]);
    useUnsavedChanges(dirty, reset, profile.displayName, displayName);

    const commitProfile = async () => {
        if (!dirty || !displayNameValid || saving) {
            return;
        }
        setSaving(true);
        try {
            const updated = await services.accounts.updateProfile(accountId, normalizedDisplayName, profile.revision);
            if (!mountedRef.current) return;
            setProfile(updated);
            setDisplayName(updated.displayName);
            await authorization.refresh();
            if (!mountedRef.current) return;
            ctx.notifications.success('Profile updated');
        } catch (err) {
            if (!mountedRef.current) return;
            if (requestErrorDetails(err).status === 409) {
                await authorization.refresh();
                if (!mountedRef.current) return;
                ctx.notifications.warning('Profile changed elsewhere', 'Your display-name draft was kept. Review the latest profile before saving again.');
            } else {
                ctx.notifications.error('Could not update profile', requestErrorMessage(err));
            }
        } finally {
            if (mountedRef.current) setSaving(false);
        }
    };

    const changeAvatar = async (file?: File) => {
        if (uploading) {
            return;
        }
        if (file && !['image/jpeg', 'image/png', 'image/webp'].includes(file.type)) {
            ctx.notifications.error('Unsupported avatar format', 'Choose a JPEG, PNG, or WebP image.');
            return;
        }
        if (file && file.size > 2 * 1024 * 1024) {
            ctx.notifications.error('Avatar is too large', 'Choose a JPEG, PNG, or WebP image up to 2 MiB.');
            return;
        }
        setUploading(true);
        try {
            const updated = file
                ? await services.accounts.uploadAvatar(accountId, username, file, profile.revision)
                : await services.accounts.deleteAvatar(accountId, username, profile.revision);
            if (!mountedRef.current) return;
            setProfile(updated);
            await authorization.refresh();
            if (!mountedRef.current) return;
            ctx.notifications.success(file ? 'Avatar updated' : 'Avatar removed');
        } catch (err) {
            if (!mountedRef.current) return;
            if (requestErrorDetails(err).status === 409) {
                await authorization.refresh();
                if (!mountedRef.current) return;
                ctx.notifications.warning('Profile changed elsewhere', 'The latest profile is being reloaded. Try the avatar change again.');
            } else {
                ctx.notifications.error(file ? 'Could not upload avatar' : 'Could not remove avatar', requestErrorMessage(err));
            }
        } finally {
            if (mountedRef.current) setUploading(false);
        }
    };

    return (
        <section className='section-panel account-profile-panel' aria-label='Profile'>
            <div className='account-profile-form'>
                <div className='account-profile-avatar'>
                    <AccountAvatar className='account-identity-avatar' profile={profile} username={username} size={72} />
                    <div className='account-profile-identity'>
                        <Typography.Title level={2}>{profile.displayName || username}</Typography.Title>
                        <Typography.Text type='secondary'>@{username}</Typography.Text>
                        <Space size={6} wrap={true}>
                            <Tag color={authorization.isAdmin ? 'success' : 'default'}>{authorization.isAdmin ? 'Administrator' : 'Member'}</Tag>
                            <Tag>{accountTierLabel(profile.tier)}</Tag>
                        </Space>
                        <Space wrap={true}>
                            <Upload
                                accept='image/jpeg,image/png,image/webp'
                                showUploadList={false}
                                beforeUpload={file => {
                                    void changeAvatar(file as File);
                                    return Upload.LIST_IGNORE;
                                }}>
                                <Button icon={<UploadOutlined aria-hidden={true} />} loading={uploading} disabled={uploading}>
                                    Upload image
                                </Button>
                            </Upload>
                            {profile.avatarUrl && (
                                <Button danger={true} icon={<DeleteOutlined aria-hidden={true} />} loading={uploading} disabled={uploading} onClick={() => void changeAvatar()}>
                                    Remove
                                </Button>
                            )}
                        </Space>
                        <Typography.Paragraph type='secondary'>JPEG, PNG, or WebP. Maximum 2 MiB. The original image is shown with a circular crop.</Typography.Paragraph>
                    </div>
                </div>
                <Form layout='vertical' onFinish={() => void commitProfile()}>
                    <Form.Item label='Username' htmlFor='profile-username' extra='Your permanent username cannot be changed.'>
                        <Input id='profile-username' value={`@${username}`} readOnly={true} />
                    </Form.Item>
                    <Form.Item
                        label={
                            <span className='account-profile-name-label'>
                                <span id='profile-name-label'>Display name</span>
                                <span id='profile-name-count' aria-hidden={true}>
                                    {unicodeCharacterCount(displayName)}/80
                                </span>
                            </span>
                        }
                        htmlFor='profile-display-name'
                        validateStatus={!displayNameValid ? 'error' : undefined}
                        help={
                            !normalizedDisplayName
                                ? 'Display name is required.'
                                : displayNameHasControls
                                  ? 'Control characters are not allowed.'
                                  : displayNameLength > 80
                                    ? 'Use 80 Unicode characters or fewer.'
                                    : 'Shown in the application shell and Account Center.'
                        }>
                        <Input
                            id='profile-display-name'
                            value={displayName}
                            aria-labelledby='profile-name-label'
                            aria-describedby='profile-name-count'
                            autoComplete='name'
                            onChange={event => setDisplayName(event.target.value)}
                        />
                    </Form.Item>
                    <div className='account-form-footer'>
                        <Typography.Text type={dirty ? 'warning' : 'secondary'} aria-live='polite'>
                            {dirty ? 'Unsaved profile changes' : `Profile revision ${profile.revision}`}
                        </Typography.Text>
                        <Space wrap={true}>
                            <Button disabled={!dirty || saving} onClick={reset}>
                                Reset
                            </Button>
                            <Button type='primary' htmlType='submit' loading={saving} disabled={!dirty || !displayNameValid}>
                                Save profile
                            </Button>
                        </Space>
                    </div>
                </Form>
            </div>
        </section>
    );
};

const identityTime = (value: number) => (value > 0 ? new Date(value * 1000).toLocaleString() : 'Not yet');

const PendingAccessPage = (props: {loggingOut: boolean; onLogout: () => void}) => {
    const authorization = useAuthorization();
    const ctx = React.useContext(Context);
    const identity = identityPresentation(authorization.user.identity);
    const [refreshing, setRefreshing] = React.useState(false);
    const refresh = async () => {
        if (refreshing) {
            return;
        }
        setRefreshing(true);
        try {
            await authorization.refresh();
            ctx.notifications.success('Access checked', 'Your latest Athena permissions have been loaded.');
        } catch (err) {
            ctx.notifications.error('Could not refresh access', requestErrorMessage(err));
        } finally {
            setRefreshing(false);
        }
    };
    return (
        <Section title='Access pending'>
            <div className='account-access-pending'>
                <div className='account-access-pending__icon' aria-hidden='true'>
                    <ClockCircleOutlined aria-hidden={true} />
                </div>
                <div className='account-access-pending__copy'>
                    <Typography.Title level={2}>{identity.pendingTitle}</Typography.Title>
                    <Typography.Paragraph>
                        Your Athena account is ready, but an administrator has not granted business access yet. You can update your profile while you wait.
                    </Typography.Paragraph>
                    <div className='account-access-pending__identity'>
                        <Typography.Text type='secondary'>{identity.label}</Typography.Text>
                        <Typography.Text className='account-identity-value' copyable={Boolean(identity.value)}>
                            {identity.value || 'Unavailable'}
                        </Typography.Text>
                    </div>
                    <Typography.Text className='account-access-pending__checked' type='secondary' aria-live='polite'>
                        Permissions are checked every 15 seconds and when this window regains focus. Last checked{' '}
                        {authorization.lastCheckedAt > 0 ? new Date(authorization.lastCheckedAt).toLocaleTimeString() : 'not yet'}.
                    </Typography.Text>
                    <Space wrap={true}>
                        <Button
                            type='primary'
                            icon={<ReloadOutlined aria-hidden={true} />}
                            loading={refreshing}
                            disabled={refreshing || props.loggingOut}
                            onClick={() => void refresh()}>
                            Refresh permissions
                        </Button>
                        <Button
                            danger={true}
                            icon={<LogoutOutlined aria-hidden={true} />}
                            loading={props.loggingOut}
                            disabled={refreshing || props.loggingOut}
                            onClick={props.onLogout}>
                            Log out
                        </Button>
                    </Space>
                </div>
            </div>
        </Section>
    );
};

const AccountSessionFacts = (props: {items: Array<{label: string; value: React.ReactNode}>}) => (
    <dl className='account-session-facts'>
        {props.items.map(item => (
            <div key={item.label}>
                <dt>{item.label}</dt>
                <dd>{item.value}</dd>
            </div>
        ))}
    </dl>
);

const ActiveAccessPage = () => {
    const authorization = useAuthorization();
    const identity = identityPresentation(authorization.user.identity);
    const version = useAsyncData<any>(() => services.version.version() as any, []);
    const uiVersion = typeof SYSTEM_INFO === 'undefined' ? 'latest' : SYSTEM_INFO.version;
    return (
        <>
            <Section title='Module access' extra={<Typography.Text type='secondary'>{moduleAccessSummary(authorization.user.access)}</Typography.Text>}>
                <Typography.Paragraph type='secondary'>
                    {authorization.isAdmin
                        ? 'Administrator access is active. Member modules, API keys, and member Profit Sharing are unavailable for this identity.'
                        : 'Your current permissions in Athena. Access is managed by an administrator.'}
                </Typography.Paragraph>
                <div className='account-module-summary'>
                    {accountAccessDisplayModules.map(definition => (
                        <div key={definition.module}>
                            <span>
                                <strong>{definition.label}</strong>
                                <small>{definition.description}</small>
                            </span>
                            <Tag color={authorization.access(definition.module) === 2 ? 'success' : undefined}>
                                {accountDataAccessLabel(authorization.access(definition.module))}
                            </Tag>
                        </div>
                    ))}
                </div>
                <div className='account-access-flags'>
                    <span>
                        API key access <StatusTag value={authorization.user.access.apiKeyEnabled ? 'Yes' : 'No'} positive={authorization.user.access.apiKeyEnabled} />
                    </span>
                    <span>
                        Profit Sharing access{' '}
                        <StatusTag value={authorization.user.access.profitSharingEnabled ? 'Yes' : 'No'} positive={authorization.user.access.profitSharingEnabled} />
                    </span>
                </div>
            </Section>
            <Section title='Current session' extra={<span>Logged in {boolTag(authorization.user.loggedIn)}</span>}>
                <AccountSessionFacts
                    items={[
                        {
                            label: identity.label,
                            value: (
                                <Typography.Text className='account-identity-value' copyable={Boolean(identity.value)}>
                                    {identity.value || '-'}
                                </Typography.Text>
                            )
                        },
                        {label: 'Identity provider', value: identityProviderLabel(authorization.user.identity.provider)},
                        {label: 'Username', value: `@${authorization.user.username}`},
                        {label: 'Role', value: authorization.isAdmin ? <StatusTag value='Administrator' positive={true} /> : 'Member'},
                        {label: 'Tier', value: accountTierLabel(authorization.user.profile.tier)}
                    ]}
                />
                <details className='account-session-details'>
                    <summary>Times, revisions &amp; versions</summary>
                    <AccountSessionFacts
                        items={[
                            {label: 'Account created', value: identityTime(authorization.user.identity.createdAt)},
                            {label: 'Last sign-in', value: identityTime(authorization.user.identity.lastLoginAt)},
                            {label: 'Access revision', value: authorization.revision},
                            {label: 'Issuer', value: authorization.user.iss || 'athena'},
                            {label: 'UI version', value: uiVersion || '-'},
                            {label: 'API version', value: version.data?.Version || version.data?.version || '-'}
                        ]}
                    />
                </details>
            </Section>
        </>
    );
};

const AccessPage = (props: {loggingOut: boolean; onLogout: () => void}) => {
    const authorization = useAuthorization();
    return accountStatusForAccess(authorization.user.access, authorization.isAdmin) === AccountStatus.Pending ? (
        <PendingAccessPage loggingOut={props.loggingOut} onLogout={props.onLogout} />
    ) : (
        <ActiveAccessPage />
    );
};

export const AccountCenterPage = (props: {section: Exclude<AccountCenterSection, 'security'>; showMemberSecurity?: boolean; loggingOut: boolean; onLogout: () => void}) => (
    <AccountCenterLayout active={props.section} showMemberSecurity={props.showMemberSecurity}>
        {props.section === 'profile' && <ProfilePage />}
        {props.section === 'access' && <AccessPage loggingOut={props.loggingOut} onLogout={props.onLogout} />}
    </AccountCenterLayout>
);
