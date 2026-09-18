import {CheckOutlined, CopyOutlined, DeleteOutlined, PlusOutlined} from '@ant-design/icons';
import {Alert, Button, Form, Input, Modal, Select, Space} from 'antd';
import * as React from 'react';
import {ResourceTable, Section, useAsyncData} from '../../components';
import {memberServices as services} from '../services';
import {Context, useAuthorization} from '../../shared/context';
import {Token} from '../../shared/models';
import {requestErrorMessage} from '../../shared/services/requests';
import {AccountCenterLayout} from '../../shared/pages/account-center';

const tokenTime = (value: number) => (value > 0 ? new Date(value * 1000).toLocaleString() : 'Never');

interface IssuedCredential {
    accountId: string;
    secret: string;
}

interface TokenListSnapshot {
    accountId: string;
    items: Token[];
}

const SecurityPage = () => {
    const authorization = useAuthorization();
    const ctx = React.useContext(Context);
    const [tokenForm] = Form.useForm();
    const [createDialogOpen, setCreateDialogOpen] = React.useState(false);
    const [creatingToken, setCreatingToken] = React.useState(false);
    const [deletingToken, setDeletingToken] = React.useState('');
    const [issuedCredential, setIssuedCredential] = React.useState<IssuedCredential | null>(null);
    const accountId = authorization.user.accountId;
    const activeAccountIdRef = React.useRef(accountId);
    const tokenCreationGenerationRef = React.useRef(0);
    const tokenCreationInFlightRef = React.useRef(false);
    const securityMountedRef = React.useRef(true);
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
    React.useLayoutEffect(() => {
        if (activeAccountIdRef.current === accountId) {
            return;
        }
        activeAccountIdRef.current = accountId;
        tokenCreationGenerationRef.current += 1;
        tokenCreationInFlightRef.current = false;
        revokeModalRef.current?.destroy();
        revokeModalRef.current = null;
        setCreateDialogOpen(false);
        setCreatingToken(false);
        setDeletingToken('');
        setIssuedCredential(null);
        if (createDialogOpen) {
            tokenForm.resetFields();
        }
    }, [accountId, createDialogOpen, tokenForm]);

    React.useLayoutEffect(() => {
        securityMountedRef.current = true;
        return () => {
            securityMountedRef.current = false;
            tokenCreationGenerationRef.current += 1;
            tokenCreationInFlightRef.current = false;
            revokeModalRef.current?.destroy();
            revokeModalRef.current = null;
        };
    }, []);

    React.useLayoutEffect(() => {
        if (!createDialogOpen) {
            return;
        }
        tokenForm.resetFields();
        tokenForm.setFieldsValue({id: '', expiresIn: 7_776_000});
    }, [createDialogOpen, tokenForm]);

    const openTokenCreation = () => {
        setCreateDialogOpen(true);
    };

    const closeTokenCreation = () => {
        if (creatingToken) {
            return;
        }
        setCreateDialogOpen(false);
        tokenForm.resetFields();
    };

    const createToken = async (values: {id: string; expiresIn: number}) => {
        if (!createDialogOpen || tokenCreationInFlightRef.current) {
            return;
        }
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
            setIssuedCredential({accountId: requestedAccountId, secret: nextSecret});
            setCreateDialogOpen(false);
            tokenForm.resetFields();
            tokens.reload();
            ctx.notifications.success('API key created');
        } catch (err) {
            if (generation === tokenCreationGenerationRef.current && requestedAccountId === activeAccountIdRef.current) {
                ctx.notifications.error('Could not create API key', requestErrorMessage(err));
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
        setIssuedCredential(null);
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
            <Section
                title='API keys'
                extra={
                    mayUseAPIKeys ? (
                        <Space wrap>
                            <Button aria-label='Refresh API keys' onClick={tokens.reload} loading={tokens.loading}>
                                Refresh
                            </Button>
                            <Button type='primary' icon={<PlusOutlined aria-hidden={true} />} onClick={openTokenCreation}>
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
            <Modal title='Create API key' open={createDialogOpen} footer={null} forceRender={true} onCancel={closeTokenCreation}>
                <Form form={tokenForm} layout='vertical' initialValues={{expiresIn: 7_776_000}} onFinish={createToken}>
                    <Form.Item
                        name='id'
                        label='Key ID'
                        rules={[
                            {required: true},
                            {max: 64},
                            {pattern: /^[A-Za-z0-9][A-Za-z0-9._-]*$/, message: 'Start with a letter or digit; use only letters, digits, dots, underscores, or hyphens.'}
                        ]}>
                        <Input autoComplete='off' placeholder='automation-client' />
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
                            Create key
                        </Button>
                    </div>
                </Form>
            </Modal>
            <Modal
                title='Copy your API key now'
                open={Boolean(currentIssuedCredential)}
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
        </>
    );
};
export const AccountSecurityPage = () => (
    <AccountCenterLayout active='security' showMemberSecurity={true}>
        <SecurityPage />
    </AccountCenterLayout>
);
