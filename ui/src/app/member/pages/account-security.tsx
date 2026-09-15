import {
    CheckCircleOutlined,
    CheckOutlined,
    CopyOutlined,
    DeleteOutlined,
    KeyOutlined,
    LoadingOutlined,
    PlusOutlined,
    ReloadOutlined,
    RobotOutlined,
    WarningOutlined
} from '@ant-design/icons';
import {Alert, Button, Form, Input, Modal, Select, Space, Tag, Typography} from 'antd';
import * as React from 'react';
import {ResourceTable, Section, useAsyncData} from '../../components';
import {memberServices as services} from '../services';
import {AIConnectionDetails, AIConnectionVerification, buildAIConnectionDetails, createAIConnectionID, verifyAIConnectionCredential} from '../../shared/ai-connection';
import {Context, useAuthorization} from '../../shared/context';
import {Token} from '../../shared/models';
import {requestErrorMessage} from '../../shared/services/requests';
import {AccountCenterLayout} from '../../shared/pages/account-center';

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
            (mayUseAPIKeys ? services.memberSecurity.listTokens().then(items => ({accountId, items})) : Promise.resolve({accountId, items: []})) as Promise<TokenListSnapshot> & {
                abort?: () => void;
            },
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

    React.useLayoutEffect(() => {
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
    }, []);

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
            const nextSecret = await services.memberSecurity.createToken(id, values.expiresIn);
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
            cancelText: 'Keep key',
            autoFocusButton: 'cancel',
            okButtonProps: {danger: true},
            onOk: async () => {
                if (!remainsCurrentSecurityAccount()) {
                    return;
                }
                setDeletingToken(item.id);
                try {
                    await services.memberSecurity.deleteToken(item.id);
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
                        <Button className='account-ai-connect-action' type='primary' icon={<RobotOutlined aria-hidden={true} />} onClick={() => openTokenCreation('ai')}>
                            Connect AI
                        </Button>
                    }>
                    <div className='account-ai-connect'>
                        <div className='account-ai-connect__copy'>
                            <Typography.Text strong={true}>Create one complete connection instruction for your AI.</Typography.Text>
                            <Typography.Paragraph type='secondary'>
                                Athena combines the service address, discovery documents, complete Bearer credential, and verification steps. Paste the result into an AI that
                                supports HTTP or custom API tools.
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
                        <Space wrap>
                            <Button aria-label='Refresh API keys' onClick={tokens.reload} loading={tokens.loading}>
                                Refresh
                            </Button>
                            <Button icon={<PlusOutlined aria-hidden={true} />} onClick={() => openTokenCreation('apiKey')}>
                                Create API key
                            </Button>
                        </Space>
                    ) : undefined
                }>
                {!mayUseAPIKeys ? (
                    <Alert type='info' showIcon={true} title='API keys are not enabled for this account' />
                ) : (
                    <>
                        {tokens.error && (
                            <Alert
                                className='account-security-warning'
                                type='error'
                                showIcon={true}
                                title='Could not load API keys'
                                description={tokens.error.message}
                                action={<Button onClick={tokens.reload}>Retry</Button>}
                            />
                        )}
                        {tokens.error && tokens.data && <Alert type='warning' title='Stale API keys — showing the last successful read' />}
                        <ResourceTable<Token>
                            rowKey='id'
                            label='Your API keys'
                            items={visibleTokens}
                            loading={tokens.loading}
                            hasData={tokens.data !== undefined}
                            columns={[
                                {title: 'ID', dataIndex: 'id'},
                                {title: 'Issued', align: 'right', render: item => tokenTime(item.issuedAt)},
                                {title: 'Expires', align: 'right', render: item => tokenTime(item.expiresAt)},
                                {
                                    title: '',
                                    width: 120,
                                    align: 'right',
                                    render: item => (
                                        <Button
                                            danger={true}
                                            type='text'
                                            icon={<DeleteOutlined aria-hidden={true} />}
                                            loading={deletingToken === item.id}
                                            onClick={() => deleteToken(item)}>
                                            Revoke
                                        </Button>
                                    )
                                }
                            ]}
                            compactRender={item => (
                                <div className='account-token-card'>
                                    <div>
                                        <strong>{item.id}</strong>
                                        <dl>
                                            <div>
                                                <dt>Issued</dt>
                                                <dd>{tokenTime(item.issuedAt)}</dd>
                                            </div>
                                            <div>
                                                <dt>Expires</dt>
                                                <dd>{tokenTime(item.expiresAt)}</dd>
                                            </div>
                                        </dl>
                                    </div>
                                    <Button danger={true} icon={<DeleteOutlined aria-hidden={true} />} loading={deletingToken === item.id} onClick={() => deleteToken(item)}>
                                        Revoke
                                    </Button>
                                </div>
                            )}
                            compactEmptyDescription='No API keys'
                        />
                    </>
                )}
            </Section>
            <Modal title={createPurpose === 'ai' ? 'Connect AI' : 'Create API key'} open={Boolean(createPurpose)} footer={null} forceRender={true} onCancel={closeTokenCreation}>
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
                    <Button type='primary' icon={<CheckOutlined aria-hidden={true} />} onClick={clearIssuedCredential}>
                        Done
                    </Button>
                }>
                <Alert type='warning' showIcon={true} title='This secret is shown only once' description='Store it in a secure secret manager before closing this dialog.' />
                <Input.TextArea className='account-secret-value' value={currentIssuedCredential?.secret || ''} readOnly={true} autoSize={{minRows: 4, maxRows: 8}} />
                <Button
                    icon={<CopyOutlined aria-hidden={true} />}
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
                footer={
                    <div className='account-ai-result-actions'>
                        <Space wrap={true}>
                            <Button
                                type='primary'
                                icon={<CopyOutlined aria-hidden={true} />}
                                autoFocus={true}
                                onClick={() => void copyValue(aiConnection?.instructions || '', 'AI connection instructions copied', 'Could not copy connection instructions')}>
                                Copy connection instructions
                            </Button>
                            <Button
                                icon={<KeyOutlined aria-hidden={true} />}
                                onClick={() => void copyValue(currentIssuedCredential?.secret || '', 'API key copied', 'Could not copy API key')}>
                                Copy API key only
                            </Button>
                        </Space>
                        <Button icon={<CheckOutlined aria-hidden={true} />} onClick={clearIssuedCredential}>
                            Done
                        </Button>
                    </div>
                }>
                <div className={`account-ai-verification account-ai-verification--${connectionVerification.status}`} role='status' aria-live='polite' aria-atomic='true'>
                    <span className='account-ai-verification__icon' aria-hidden='true'>
                        {connectionVerification.status === 'checking' && <LoadingOutlined aria-hidden={true} spin={true} />}
                        {connectionVerification.status === 'ready' && <CheckCircleOutlined aria-hidden={true} />}
                        {connectionVerification.status === 'failed' && <WarningOutlined aria-hidden={true} />}
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
                        <Button size='small' icon={<ReloadOutlined aria-hidden={true} />} onClick={() => setVerificationAttempt(attempt => attempt + 1)}>
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
            </Modal>
        </>
    );
};
export const AccountSecurityPage = () => (
    <AccountCenterLayout active='security' showMemberSecurity={true}>
        <SecurityPage />
    </AccountCenterLayout>
);
