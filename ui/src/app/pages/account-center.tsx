import {
    BgColorsOutlined,
    CheckCircleOutlined,
    CheckOutlined,
    ClockCircleOutlined,
    CopyOutlined,
    DeleteOutlined,
    KeyOutlined,
    LoadingOutlined,
    LogoutOutlined,
    PlusOutlined,
    ReloadOutlined,
    RobotOutlined,
    SafetyCertificateOutlined,
    UploadOutlined,
    UserOutlined,
    WarningOutlined
} from '@ant-design/icons';
import {Alert, Avatar, Button, Form, Input, Modal, Select, Space, Tag, Typography, Upload} from 'antd';
import * as React from 'react';
import {useBlocker, useNavigate} from 'react-router-dom';
import {AppPage, ChoiceGroup, KeyValueGrid, ResourceTable, Section, StatusTag, useAsyncData} from '../components';
import {moduleAccessSummary} from '../shared/account-access';
import {accountDataAccessLabel, accountDataModules} from '../shared/access-modules';
import {
    AIConnectionDetails,
    AIConnectionVerification,
    buildAIConnectionDetails,
    createAIConnectionID,
    verifyAIConnectionCredential
} from '../shared/ai-connection';
import {Context, useAuthorization} from '../shared/context';
import {accountStatusForAccess, AccountIdentity, AccountIdentityProvider, AccountProfile, AccountStatus, AccountThemeMode, AccountTier, Token} from '../shared/models';
import {services, ThemeMode, ViewPreferences} from '../shared/services';
import requests, {requestErrorDetails, requestErrorMessage} from '../shared/services/requests';
import {boolTag} from './shared';

export type AccountCenterSection = 'profile' | 'appearance' | 'security' | 'access';

const accountSections: Array<{key: AccountCenterSection; label: string; description: string; icon: React.ReactNode; requiresAPIKey?: boolean}> = [
    {key: 'profile', label: 'Profile', description: 'Name and avatar', icon: <UserOutlined />},
    {key: 'appearance', label: 'Appearance', description: 'Theme across devices', icon: <BgColorsOutlined />},
    {key: 'security', label: 'Security', description: 'API keys and AI connections', icon: <SafetyCertificateOutlined />, requiresAPIKey: true},
    {key: 'access', label: 'Access & session', description: 'Permissions and versions', icon: <KeyOutlined />}
];

export const accountTierLabel = (tier: AccountTier) => (tier === AccountTier.Pro ? 'Pro' : 'Standard');

export const accountThemeLabel = (theme: AccountThemeMode | ThemeMode) => {
    if (theme === AccountThemeMode.Dark || theme === 'dark') {
        return 'Dark';
    }
    if (theme === AccountThemeMode.Light || theme === 'light') {
        return 'Light';
    }
    return 'System';
};

export const serverThemeMode = (theme: ThemeMode): AccountThemeMode => {
    if (theme === 'dark') {
        return AccountThemeMode.Dark;
    }
    if (theme === 'light') {
        return AccountThemeMode.Light;
    }
    return AccountThemeMode.System;
};

export const localThemeMode = (theme: AccountThemeMode): ThemeMode => {
    if (theme === AccountThemeMode.Dark) {
        return 'dark';
    }
    if (theme === AccountThemeMode.Light) {
        return 'light';
    }
    return 'system';
};

const accountInitials = (displayName: string, username: string) => (displayName || username || 'A').trim().slice(0, 1).toUpperCase();
export const unicodeCharacterCount = (value: string) => Array.from(value).length;
export const hasControlCharacters = (value: string) => /\p{Cc}/u.test(value);

const avatarURL = (url: string) => {
    if (!url) {
        return undefined;
    }
    return url.startsWith('/api/') ? requests.toAbsURL(url) : url;
};

export const AccountAvatar = (props: {profile: AccountProfile; username: string; size?: number; className?: string}) => (
    <Avatar className={props.className} size={props.size || 40} src={avatarURL(props.profile.avatarUrl)}>
        {accountInitials(props.profile.displayName, props.username)}
    </Avatar>
);

const useUnsavedChanges = (dirty: boolean, reset: () => void, label: string) => {
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
            title: `Discard unsaved ${label}?`,
            content: 'Your current changes have not been saved.',
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
    }, [blocker.state, ctx.modal, label, reset]);
};

const AccountCenterLayout = (props: {active: AccountCenterSection; children: React.ReactNode}) => {
    const navigate = useNavigate();
    const authorization = useAuthorization();
    const profile = authorization.user.profile;
    const visibleSections = accountSections.filter(section => !section.requiresAPIKey || (authorization.user.access.apiKeyEnabled && !authorization.isAdmin));
    return (
        <AppPage title='Account Center' subtitle='Manage your Athena identity, appearance, API keys, and current access.'>
            <div className='account-center-hero'>
                <AccountAvatar profile={profile} username={authorization.user.username} size={64} />
                <div className='account-center-hero__copy'>
                    <Typography.Title level={2}>{profile.displayName || authorization.user.username}</Typography.Title>
                    <Typography.Text type='secondary'>@{authorization.user.username}</Typography.Text>
                    <Space size={6} wrap={true}>
                        <Tag>{accountTierLabel(profile.tier)}</Tag>
                        <Tag color={authorization.isAdmin ? 'gold' : 'default'}>{authorization.isAdmin ? 'Administrator' : 'Member'}</Tag>
                    </Space>
                </div>
            </div>
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
    useUnsavedChanges(dirty, reset, 'profile changes');

    const commitProfile = async () => {
        if (!dirty || !displayNameValid || saving) {
            return;
        }
        setSaving(true);
        try {
            const updated = await services.accounts.updateProfile(accountId, normalizedDisplayName, profile.revision);
            setProfile(updated);
            setDisplayName(updated.displayName);
            await authorization.refresh();
            ctx.notifications.success('Profile updated');
        } catch (err) {
            if (requestErrorDetails(err).status === 409) {
                await authorization.refresh();
                ctx.notifications.warning('Profile changed elsewhere', 'Your display-name draft was kept. Review the latest profile before saving again.');
            } else {
                ctx.notifications.error('Could not update profile', requestErrorMessage(err));
            }
        } finally {
            setSaving(false);
        }
    };

    const changeAvatar = async (file?: File) => {
        if (uploading) {
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
            setProfile(updated);
            await authorization.refresh();
            ctx.notifications.success(file ? 'Avatar updated' : 'Avatar removed');
        } catch (err) {
            if (requestErrorDetails(err).status === 409) {
                await authorization.refresh();
                ctx.notifications.warning('Profile changed elsewhere', 'The latest profile is being reloaded. Try the avatar change again.');
            } else {
                ctx.notifications.error(file ? 'Could not upload avatar' : 'Could not remove avatar', requestErrorMessage(err));
            }
        } finally {
            setUploading(false);
        }
    };

    return (
        <Section title='Profile'>
            <div className='account-profile-form'>
                <div className='account-profile-avatar'>
                    <AccountAvatar profile={profile} username={username} size={88} />
                    <div>
                        <Space wrap={true}>
                            <Upload
                                accept='image/jpeg,image/png,image/webp'
                                showUploadList={false}
                                beforeUpload={file => {
                                    void changeAvatar(file as File);
                                    return Upload.LIST_IGNORE;
                                }}>
                                <Button icon={<UploadOutlined />} loading={uploading} disabled={uploading}>
                                    Upload image
                                </Button>
                            </Upload>
                            {profile.avatarUrl && (
                                <Button danger={true} icon={<DeleteOutlined />} loading={uploading} disabled={uploading} onClick={() => void changeAvatar()}>
                                    Remove
                                </Button>
                            )}
                        </Space>
                        <Typography.Paragraph type='secondary'>JPEG, PNG, or WebP. Maximum 2 MiB. The original image is shown with a circular crop.</Typography.Paragraph>
                    </div>
                </div>
                <Form layout='vertical' onFinish={() => void commitProfile()}>
                    <Form.Item label='Username'>
                        <Input value={`@${username}`} readOnly={true} />
                    </Form.Item>
                    <Form.Item
                        label='Display name'
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
                            value={displayName}
                            showCount={{formatter: info => `${unicodeCharacterCount(info.value)}/80`}}
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
        </Section>
    );
};

const AppearancePage = (props: {preferences: ViewPreferences; changing: boolean; onThemeChange: (theme: ThemeMode) => Promise<void>}) => {
    const authorization = useAuthorization();
    return (
        <Section title='Appearance'>
            <div className='account-theme-options'>
                <div>
                    <Typography.Text strong={true}>Color theme</Typography.Text>
                    <Typography.Paragraph type='secondary'>
                        This preference follows you across signed-in devices. System tracks your operating-system appearance.
                    </Typography.Paragraph>
                </div>
                <ChoiceGroup<ThemeMode>
                    className='account-theme-choice'
                    ariaLabel='Athena color theme'
                    value={props.preferences.theme}
                    disabled={props.changing}
                    options={[
                        {value: 'system', label: 'System'},
                        {value: 'light', label: 'Light'},
                        {value: 'dark', label: 'Dark'}
                    ]}
                    onChange={theme => void props.onThemeChange(theme)}
                />
            </div>
            <Alert
                type='info'
                showIcon={true}
                title={`${accountThemeLabel(props.preferences.theme)} theme selected`}
                description={`Server preference revision ${authorization.user.preferences.revision}. Sidebar, table, and sorting preferences remain local to this browser.`}
            />
        </Section>
    );
};

const tokenTime = (value: number) => (value > 0 ? new Date(value * 1000).toLocaleString() : 'Never');

type TokenCreationPurpose = 'apiKey' | 'ai';

interface IssuedCredential {
    purpose: TokenCreationPurpose;
    accountId: string;
    secret: string;
}

interface TokenListSnapshot {
    accountId: string;
    items: Token[];
}

type AIConnectionVerificationState = AIConnectionVerification | {status: 'idle' | 'checking'; message: string};

const idleAIConnectionVerification: AIConnectionVerificationState = {status: 'idle', message: ''};

const SecurityPage = () => {
    const authorization = useAuthorization();
    const ctx = React.useContext(Context);
    const [tokenForm] = Form.useForm();
    const [createPurpose, setCreatePurpose] = React.useState<TokenCreationPurpose | null>(null);
    const [creatingToken, setCreatingToken] = React.useState(false);
    const [deletingToken, setDeletingToken] = React.useState('');
    const [issuedCredential, setIssuedCredential] = React.useState<IssuedCredential | null>(null);
    const [verificationAttempt, setVerificationAttempt] = React.useState(0);
    const [connectionVerification, setConnectionVerification] = React.useState<AIConnectionVerificationState>(idleAIConnectionVerification);
    const accountId = authorization.user.accountId;
    const activeAccountIdRef = React.useRef(accountId);
    const tokenCreationGenerationRef = React.useRef(0);
    const tokenCreationInFlightRef = React.useRef(false);
    const securityMountedRef = React.useRef(true);
    const verificationControllerRef = React.useRef<AbortController | null>(null);
    const revokeModalRef = React.useRef<{destroy(): void} | null>(null);
    const mayUseAPIKeys = authorization.user.access.apiKeyEnabled && !authorization.isAdmin;
    const tokens = useAsyncData<TokenListSnapshot>(
        () =>
            (mayUseAPIKeys
                ? services.accounts.listTokens().then(items => ({accountId, items}))
                : Promise.resolve({accountId, items: []})) as Promise<TokenListSnapshot> & {abort?: () => void},
        [accountId, mayUseAPIKeys]
    );
    const visibleTokens = tokens.data?.accountId === accountId ? tokens.data.items : [];
    const currentIssuedCredential = issuedCredential?.accountId === accountId ? issuedCredential : null;
    const aiConnection = React.useMemo<AIConnectionDetails | null>(
        () =>
            currentIssuedCredential?.purpose === 'ai'
                ? buildAIConnectionDetails({baseURI: document.baseURI, accountId: currentIssuedCredential.accountId, secret: currentIssuedCredential.secret})
                : null,
        [currentIssuedCredential]
    );

    React.useLayoutEffect(() => {
        if (activeAccountIdRef.current === accountId) {
            return;
        }
        activeAccountIdRef.current = accountId;
        tokenCreationGenerationRef.current += 1;
        tokenCreationInFlightRef.current = false;
        verificationControllerRef.current?.abort();
        verificationControllerRef.current = null;
        revokeModalRef.current?.destroy();
        revokeModalRef.current = null;
        setCreatePurpose(null);
        setCreatingToken(false);
        setDeletingToken('');
        setIssuedCredential(null);
        setVerificationAttempt(0);
        setConnectionVerification(idleAIConnectionVerification);
        if (createPurpose) {
            tokenForm.resetFields();
        }
    }, [accountId, createPurpose, tokenForm]);

    React.useLayoutEffect(
        () => {
            securityMountedRef.current = true;
            return () => {
                securityMountedRef.current = false;
                tokenCreationGenerationRef.current += 1;
                tokenCreationInFlightRef.current = false;
                verificationControllerRef.current?.abort();
                verificationControllerRef.current = null;
                revokeModalRef.current?.destroy();
                revokeModalRef.current = null;
            };
        },
        []
    );

    React.useLayoutEffect(() => {
        if (!createPurpose) {
            return;
        }
        tokenForm.resetFields();
        tokenForm.setFieldsValue({id: createPurpose === 'ai' ? createAIConnectionID() : '', expiresIn: 7_776_000});
    }, [createPurpose, tokenForm]);

    React.useEffect(() => {
        if (!aiConnection) {
            setConnectionVerification(idleAIConnectionVerification);
            return;
        }
        let current = true;
        const controller = new AbortController();
        const credential = {
            verifyUrl: aiConnection.verifyUrl,
            expectedAccountId: aiConnection.expectedAccountId,
            authorizationHeader: aiConnection.authorizationHeader
        };
        setConnectionVerification({status: 'checking', message: 'Confirming that Athena accepts the newly issued credential.'});
        verificationControllerRef.current = controller;
        void verifyAIConnectionCredential(credential, controller.signal)
            .then(result => {
                if (current && !controller.signal.aborted) {
                    setConnectionVerification(result);
                }
            })
            .finally(() => {
                if (verificationControllerRef.current === controller) {
                    verificationControllerRef.current = null;
                }
            });
        return () => {
            current = false;
            controller.abort();
            if (verificationControllerRef.current === controller) {
                verificationControllerRef.current = null;
            }
        };
    }, [aiConnection, verificationAttempt]);

    const openTokenCreation = (purpose: TokenCreationPurpose) => {
        setCreatePurpose(purpose);
    };

    const closeTokenCreation = () => {
        if (creatingToken) {
            return;
        }
        setCreatePurpose(null);
        tokenForm.resetFields();
    };

    const createToken = async (values: {id: string; expiresIn: number}) => {
        if (!createPurpose || tokenCreationInFlightRef.current) {
            return;
        }
        const purpose = createPurpose;
        const id = values.id.trim();
        const requestedAccountId = accountId;
        const generation = tokenCreationGenerationRef.current + 1;
        tokenCreationGenerationRef.current = generation;
        tokenCreationInFlightRef.current = true;
        setCreatingToken(true);
        try {
            const nextSecret = await services.accounts.createToken(id, values.expiresIn);
            if (generation !== tokenCreationGenerationRef.current || requestedAccountId !== activeAccountIdRef.current) {
                if (securityMountedRef.current) {
                    ctx.notifications.warning(
                        'Credential result discarded',
                        'The account or Security session changed before creation finished. Return to the account that requested the key to review or revoke its metadata.'
                    );
                }
                return;
            }
            if (purpose === 'ai') {
                setConnectionVerification({status: 'checking', message: 'Confirming that Athena accepts the newly issued credential.'});
            }
            setIssuedCredential({purpose, accountId: requestedAccountId, secret: nextSecret});
            setVerificationAttempt(0);
            setCreatePurpose(null);
            tokenForm.resetFields();
            tokens.reload();
            ctx.notifications.success(purpose === 'ai' ? 'Connection instructions ready' : 'API key created');
        } catch (err) {
            if (generation === tokenCreationGenerationRef.current && requestedAccountId === activeAccountIdRef.current) {
                ctx.notifications.error(purpose === 'ai' ? 'Could not create AI connection' : 'Could not create API key', requestErrorMessage(err));
            }
        } finally {
            if (generation === tokenCreationGenerationRef.current) {
                tokenCreationInFlightRef.current = false;
                setCreatingToken(false);
            }
        }
    };

    const copyValue = async (value: string, successMessage: string, failureMessage: string) => {
        try {
            if (!navigator.clipboard?.writeText) {
                throw new Error('Clipboard access is unavailable');
            }
            await navigator.clipboard.writeText(value);
            ctx.notifications.success(successMessage);
        } catch (err) {
            ctx.notifications.error(failureMessage, requestErrorMessage(err));
        }
    };

    const clearIssuedCredential = () => {
        verificationControllerRef.current?.abort();
        verificationControllerRef.current = null;
        setIssuedCredential(null);
        setVerificationAttempt(0);
        setConnectionVerification(idleAIConnectionVerification);
    };

    const deleteToken = (item: Token) => {
        const requestedAccountId = accountId;
        const remainsCurrentSecurityAccount = () => securityMountedRef.current && requestedAccountId === activeAccountIdRef.current;
        const handle = ctx.modal.confirm({
            title: `Revoke ${item.id}?`,
            content: 'Requests using this API key will fail immediately.',
            okText: 'Revoke key',
            onOk: async () => {
                if (!remainsCurrentSecurityAccount()) {
                    return;
                }
                setDeletingToken(item.id);
                try {
                    await services.accounts.deleteToken(item.id);
                    if (remainsCurrentSecurityAccount()) {
                        tokens.reload();
                        ctx.notifications.success('API key revoked');
                    }
                } catch (err) {
                    if (remainsCurrentSecurityAccount()) {
                        ctx.notifications.error('Could not revoke API key', requestErrorMessage(err));
                    }
                } finally {
                    if (remainsCurrentSecurityAccount()) {
                        setDeletingToken('');
                    }
                    if (revokeModalRef.current === handle) {
                        revokeModalRef.current = null;
                    }
                }
            },
            onCancel: () => {
                if (revokeModalRef.current === handle) {
                    revokeModalRef.current = null;
                }
            }
        });
        revokeModalRef.current = handle;
    };

    return (
        <>
            {mayUseAPIKeys && (
                <Section
                    title='Connect AI'
                    extra={
                        <Button className='account-ai-connect-action' type='primary' icon={<RobotOutlined />} onClick={() => openTokenCreation('ai')}>
                            Connect AI
                        </Button>
                    }>
                    <div className='account-ai-connect'>
                        <div className='account-ai-connect__icon' aria-hidden='true'>
                            <RobotOutlined />
                        </div>
                        <div className='account-ai-connect__copy'>
                            <Typography.Text strong={true}>Create one complete connection instruction for your AI.</Typography.Text>
                            <Typography.Paragraph type='secondary'>
                                Athena combines the service address, discovery documents, complete Bearer credential, and verification steps. Paste the result into an AI that supports HTTP or custom API tools.
                            </Typography.Paragraph>
                            <Space size={6} wrap={true}>
                                <Tag>Full account authority</Tag>
                                <Tag>One-time display</Tag>
                                <Tag>One copy, one paste</Tag>
                            </Space>
                        </div>
                    </div>
                </Section>
            )}
            <Section
                title='API keys'
                extra={
                    mayUseAPIKeys ? (
                        <Button icon={<PlusOutlined />} onClick={() => openTokenCreation('apiKey')}>
                            Create API key
                        </Button>
                    ) : undefined
                }>
                {!mayUseAPIKeys ? (
                    <Alert type='info' showIcon={true} title='API keys are not enabled for this account' />
                ) : (
                    <>
                        {tokens.error && (
                            <Alert className='account-security-warning' type='error' showIcon={true} title='Could not load API keys' description={tokens.error.message} />
                        )}
                        <ResourceTable<Token>
                            rowKey='id'
                            label='Your API keys'
                            items={visibleTokens}
                            loading={tokens.loading}
                            columns={[
                                {title: 'ID', dataIndex: 'id'},
                                {title: 'Issued', render: item => tokenTime(item.issuedAt)},
                                {title: 'Expires', render: item => tokenTime(item.expiresAt)},
                                {
                                    title: '',
                                    width: 120,
                                    align: 'right',
                                    render: item => (
                                        <Button danger={true} type='text' icon={<DeleteOutlined />} loading={deletingToken === item.id} onClick={() => deleteToken(item)}>
                                            Revoke
                                        </Button>
                                    )
                                }
                            ]}
                            compactRender={item => (
                                <div className='account-token-card'>
                                    <div>
                                        <strong>{item.id}</strong>
                                        <Typography.Text type='secondary'>Issued {tokenTime(item.issuedAt)}</Typography.Text>
                                        <Typography.Text type='secondary'>Expires {tokenTime(item.expiresAt)}</Typography.Text>
                                    </div>
                                    <Button danger={true} icon={<DeleteOutlined />} loading={deletingToken === item.id} onClick={() => deleteToken(item)}>
                                        Revoke
                                    </Button>
                                </div>
                            )}
                            compactEmptyDescription='No API keys'
                        />
                    </>
                )}
            </Section>
            <Modal
                title={createPurpose === 'ai' ? 'Connect AI' : 'Create API key'}
                open={Boolean(createPurpose)}
                footer={null}
                forceRender={true}
                onCancel={closeTokenCreation}>
                {createPurpose === 'ai' && (
                    <Alert
                        className='account-ai-authority-alert'
                        type='info'
                        showIcon={true}
                        title='Full account authority'
                        description='The AI can perform every HTTP API operation currently allowed for your ordinary account.'
                    />
                )}
                <Form form={tokenForm} layout='vertical' initialValues={{expiresIn: 7_776_000}} onFinish={createToken}>
                    <Form.Item
                        name='id'
                        label={createPurpose === 'ai' ? 'Connection name' : 'Key ID'}
                        rules={[
                            {required: true},
                            {max: 64},
                            {pattern: /^[A-Za-z0-9][A-Za-z0-9._-]*$/, message: 'Start with a letter or digit; use only letters, digits, dots, underscores, or hyphens.'}
                        ]}>
                        <Input autoComplete='off' placeholder={createPurpose === 'ai' ? 'ai-connection' : 'automation-client'} />
                    </Form.Item>
                    <Form.Item name='expiresIn' label='Expiration' rules={[{required: true}]}>
                        <Select
                            options={[
                                {value: 2_592_000, label: '30 days'},
                                {value: 7_776_000, label: '90 days'},
                                {value: 31_536_000, label: '1 year'},
                                {value: 0, label: 'No expiration'}
                            ]}
                        />
                    </Form.Item>
                    <div className='account-modal-actions'>
                        <Button disabled={creatingToken} onClick={closeTokenCreation}>
                            Cancel
                        </Button>
                        <Button type='primary' htmlType='submit' loading={creatingToken}>
                            {createPurpose === 'ai' ? 'Create connection' : 'Create key'}
                        </Button>
                    </div>
                </Form>
            </Modal>
            <Modal
                title='Copy your API key now'
                open={currentIssuedCredential?.purpose === 'apiKey'}
                closable={false}
                maskClosable={false}
                keyboard={false}
                footer={
                    <Button type='primary' icon={<CheckOutlined />} onClick={clearIssuedCredential}>
                        Done
                    </Button>
                }>
                <Alert type='warning' showIcon={true} title='This secret is shown only once' description='Store it in a secure secret manager before closing this dialog.' />
                <Input.TextArea className='account-secret-value' value={currentIssuedCredential?.secret || ''} readOnly={true} autoSize={{minRows: 4, maxRows: 8}} />
                <Button
                    icon={<CopyOutlined />}
                    onClick={() => void copyValue(currentIssuedCredential?.secret || '', 'API key copied', 'Could not copy API key')}>
                    Copy API key
                </Button>
            </Modal>
            <Modal
                className='account-ai-result-modal'
                title='AI connection instructions ready'
                width={760}
                open={Boolean(aiConnection)}
                closable={false}
                maskClosable={false}
                keyboard={false}
                footer={null}>
                <div className={`account-ai-verification account-ai-verification--${connectionVerification.status}`} role='status' aria-live='polite' aria-atomic='true'>
                    <span className='account-ai-verification__icon' aria-hidden='true'>
                        {connectionVerification.status === 'checking' && <LoadingOutlined spin={true} />}
                        {connectionVerification.status === 'ready' && <CheckCircleOutlined />}
                        {connectionVerification.status === 'failed' && <WarningOutlined />}
                    </span>
                    <span className='account-ai-verification__copy'>
                        <strong>
                            {connectionVerification.status === 'checking' && 'Checking credential'}
                            {connectionVerification.status === 'ready' && 'Credential ready'}
                            {connectionVerification.status === 'failed' && 'Could not verify credential'}
                        </strong>
                        <small>{connectionVerification.message}</small>
                    </span>
                    {connectionVerification.status === 'failed' && (
                        <Button size='small' icon={<ReloadOutlined />} onClick={() => setVerificationAttempt(attempt => attempt + 1)}>
                            Retry
                        </Button>
                    )}
                </div>
                <Alert
                    className='account-ai-result-alert'
                    type='warning'
                    showIcon={true}
                    title='This connection is shown only once'
                    description='Copy the instructions before closing. Athena cannot rebuild them from the API key list.'
                />
                <Alert
                    className='account-ai-result-alert'
                    type='info'
                    showIcon={true}
                    title='This verifies the credential, not the external AI'
                    description='The selected AI still needs to receive these instructions and support HTTP or custom API tools.'
                />
                <Input.TextArea
                    className='account-ai-instructions'
                    aria-label='AI connection instructions'
                    value={aiConnection?.instructions || ''}
                    readOnly={true}
                    autoSize={{minRows: 12, maxRows: 18}}
                />
                <div className='account-ai-result-actions'>
                    <Space wrap={true}>
                        <Button
                            type='primary'
                            icon={<CopyOutlined />}
                            autoFocus={true}
                            onClick={() => void copyValue(aiConnection?.instructions || '', 'AI connection instructions copied', 'Could not copy connection instructions')}>
                            Copy connection instructions
                        </Button>
                        <Button
                            icon={<KeyOutlined />}
                            onClick={() => void copyValue(currentIssuedCredential?.secret || '', 'API key copied', 'Could not copy API key')}>
                            Copy API key only
                        </Button>
                    </Space>
                    <Button icon={<CheckOutlined />} onClick={clearIssuedCredential}>
                        Done
                    </Button>
                </div>
            </Modal>
        </>
    );
};

export const identityProviderLabel = (provider: AccountIdentityProvider) => {
    switch (provider) {
        case AccountIdentityProvider.Google:
            return 'Google';
        case AccountIdentityProvider.SolanaWallet:
            return 'Phantom';
        case AccountIdentityProvider.Development:
            return 'Development';
        default:
            return 'Not available';
    }
};

export const identityPresentation = (identity: AccountIdentity) => {
    switch (identity.provider) {
        case AccountIdentityProvider.Google:
            return {
                label: 'Verified email',
                value: identity.verifiedEmail,
                pendingTitle: 'Your Google identity is verified',
                signInLabel: 'Google sign-in'
            };
        case AccountIdentityProvider.SolanaWallet:
            return {
                label: 'Solana address',
                value: identity.solanaAddress,
                pendingTitle: 'Your Phantom wallet ownership is verified',
                signInLabel: 'Phantom sign-in'
            };
        default:
            return {
                label: 'Identity',
                value: identity.verifiedEmail || identity.solanaAddress,
                pendingTitle: 'Your identity is verified',
                signInLabel: 'Sign-in'
            };
    }
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
                    <ClockCircleOutlined />
                </div>
                <div className='account-access-pending__copy'>
                    <Typography.Title level={2}>{identity.pendingTitle}</Typography.Title>
                    <Typography.Paragraph>
                        Your Athena account is ready, but an administrator has not granted business access yet. You can update your profile and appearance while you wait.
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
                        <Button type='primary' icon={<ReloadOutlined />} loading={refreshing} disabled={refreshing || props.loggingOut} onClick={() => void refresh()}>
                            Refresh permissions
                        </Button>
                        <Button danger={true} icon={<LogoutOutlined />} loading={props.loggingOut} disabled={refreshing || props.loggingOut} onClick={props.onLogout}>
                            Log out
                        </Button>
                    </Space>
                </div>
            </div>
        </Section>
    );
};

const ActiveAccessPage = () => {
    const authorization = useAuthorization();
    const identity = identityPresentation(authorization.user.identity);
    const version = useAsyncData<any>(() => services.version.version() as any, []);
    const uiVersion = typeof SYSTEM_INFO === 'undefined' ? 'latest' : SYSTEM_INFO.version;
    return (
        <>
            <Section title='Current session'>
                <KeyValueGrid
                    items={[
                        {label: 'Username', value: `@${authorization.user.username}`},
                        {label: 'Identity provider', value: identityProviderLabel(authorization.user.identity.provider)},
                        {
                            label: identity.label,
                            value: (
                                <Typography.Text className='account-identity-value' copyable={Boolean(identity.value)}>
                                    {identity.value || '-'}
                                </Typography.Text>
                            )
                        },
                        {label: 'Account created', value: identityTime(authorization.user.identity.createdAt)},
                        {label: 'Last sign-in', value: identityTime(authorization.user.identity.lastLoginAt)},
                        {label: 'Logged in', value: boolTag(authorization.user.loggedIn)},
                        {label: 'Role', value: authorization.isAdmin ? <StatusTag value='Administrator' positive={true} /> : 'Member'},
                        {label: 'Tier', value: accountTierLabel(authorization.user.profile.tier)},
                        {label: 'Module access', value: moduleAccessSummary(authorization.user.access, authorization.isAdmin)},
                        {label: 'API key access', value: boolTag(authorization.user.access.apiKeyEnabled)},
                        {label: 'Profit Sharing access', value: boolTag(authorization.user.access.profitSharingEnabled)},
                        {label: 'Access revision', value: authorization.revision},
                        {label: 'Issuer', value: authorization.user.iss || 'athena'},
                        {label: 'UI version', value: uiVersion || '-'},
                        {label: 'API version', value: version.data?.Version || version.data?.version || '-'}
                    ]}
                />
            </Section>
            <Section title='Module access'>
                <div className='account-module-summary'>
                    {accountDataModules.map(definition => (
                        <div key={definition.module}>
                            <span>
                                <strong>{definition.label}</strong>
                                <small>{definition.description}</small>
                            </span>
                            <Tag>{authorization.isAdmin ? accountDataAccessLabel(definition.maxAccess) : accountDataAccessLabel(authorization.access(definition.module))}</Tag>
                        </div>
                    ))}
                </div>
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

export const AccountCenterPage = (props: {
    section: AccountCenterSection;
    preferences: ViewPreferences;
    themeChanging: boolean;
    onThemeChange: (theme: ThemeMode) => Promise<void>;
    loggingOut: boolean;
    onLogout: () => void;
}) => (
    <AccountCenterLayout active={props.section}>
        {props.section === 'profile' && <ProfilePage />}
        {props.section === 'appearance' && <AppearancePage preferences={props.preferences} changing={props.themeChanging} onThemeChange={props.onThemeChange} />}
        {props.section === 'security' && <SecurityPage />}
        {props.section === 'access' && <AccessPage loggingOut={props.loggingOut} onLogout={props.onLogout} />}
    </AccountCenterLayout>
);
