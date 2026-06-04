import {ApiOutlined, CheckCircleOutlined, DeleteOutlined, EyeOutlined, LoginOutlined, PlusOutlined, SaveOutlined, SendOutlined, StopOutlined} from '@ant-design/icons';
import {Alert, Button, Card, Collapse, Dropdown, Form, Input, InputNumber, Modal, Select, Space, Tabs, Tag, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {Link, useNavigate, useParams, useSearchParams} from 'react-router-dom';
import {
    AppPage,
    BrandMark,
    CardTitle,
    InlineActions,
    KeyValueGrid,
    MetricRow,
    ResponsiveResourceList,
    SearchBar,
    Section,
    StatusTag,
    TruncatedText,
    useAsyncData,
    useBreakpoint
} from './components';
import {Context} from '../shared/context';
import {BytecodeBlacklistEntry, BytecodeDeployment, BytecodeListItem, SourceQualityPrompt} from '../shared/services/athena-application-service';
import {Account, UserInfo, VersionMessage} from '../shared/models';
import {NotificationDelivery} from '../shared/services/notification-service';
import {
    PolymarketHotMarketItem,
    PolymarketMoverMarketItem,
    PolymarketRealtimeMarketItem,
    PolymarketSportsLiveEventItem,
    PolymarketSportsLiveMarketItem
} from '../shared/services/polymarket-service';
import {services} from '../shared/services';
import {WalletBlacklistEntry, WalletDetail, WalletItem} from '../shared/services/wallet-service';
import {WormMarketDetail, WormMarketItem} from '../shared/services/worm-service';

const fmt = (value: unknown) => {
    if (value === undefined || value === null || value === '') {
        return '-';
    }
    if (typeof value === 'boolean') {
        return value ? 'Yes' : 'No';
    }
    return String(value);
};

const fmtNumber = (value?: number) => (value === undefined ? '-' : new Intl.NumberFormat().format(value));
const short = (value?: string, head = 10, tail = 8) => (value && value.length > head + tail ? `${value.slice(0, head)}...${value.slice(-tail)}` : value || '-');
const boolTag = (value?: boolean) => <StatusTag value={fmt(value)} positive={value === true} negative={value === false} />;
const dangerTag = (value?: boolean) => <StatusTag value={fmt(value)} positive={value === false} negative={value === true} />;

const wormMarketLogo = (item: WormMarketItem) => item.logo || item.eventLogo || '';

export const WormMarketSummary = (props: {item: WormMarketItem; onOpen?: (item: WormMarketItem) => void}) => {
    const item = props.item;
    const title = props.onOpen ? (
        <Button className='worm-market-summary__button' type='link' onClick={() => props.onOpen?.(item)}>
            {item.title}
        </Button>
    ) : (
        item.title
    );
    return (
        <div className='worm-market-summary'>
            <CardTitle
                title={title}
                subtitle={item.eventTitle || item.category}
                image={wormMarketLogo(item)}
                tags={item.marginEnabled ? <Tag color='green'>Margin</Tag> : <Tag>{item.state}</Tag>}
            />
        </div>
    );
};

export const visibleAccountsForUser = (accounts: Account[], user?: UserInfo): Account[] => {
    if (user?.username === 'admin') {
        return accounts;
    }
    return accounts.filter(account => account.name === 'admin' || account.name === user?.username);
};

export const notificationTestTopics = [
    {topic: 'token', label: '[TOKEN] 代币通知'},
    {topic: 'poly-mover', label: '[POLY] 市场异动'},
    {topic: 'poly-kickoff', label: '[POLY] 开赛通知'}
];

const rbacResources = {
    application: 'application',
    applicationDiscovery: 'application-discovery',
    notifications: 'notifications',
    wallets: 'wallets'
};

const rbacActions = {
    update: 'update',
    invoke: 'invoke'
};

const useCanI = (resource: string, action: string, subresource = '*') =>
    useAsyncData<boolean>(() => services.accounts.canI(resource, action, subresource) as any, [resource, action, subresource]);

const usePagedParams = (defaultPageSize = 20) => {
    const [params, setParams] = useSearchParams();
    const page = Number(params.get('page') || 1) || 1;
    const pageSize = Number(params.get('pageSize') || params.get('page_size') || defaultPageSize) || defaultPageSize;
    const setPage = (nextPage: number, nextPageSize: number) => {
        const next = new URLSearchParams(params);
        next.set('page', String(nextPage));
        next.set('pageSize', String(nextPageSize));
        setParams(next);
    };
    return {params, setParams, page, pageSize, setPage};
};

const useKeywordParam = (key = 'q') => {
    const [params, setParams] = useSearchParams();
    const value = params.get(key) || '';
    const setValue = (nextValue: string) => {
        const next = new URLSearchParams(params);
        if (nextValue) {
            next.set(key, nextValue);
        } else {
            next.delete(key);
        }
        next.set('page', '1');
        setParams(next);
    };
    return [value, setValue] as const;
};

const postLoginPath = '/settings';

export const LoginPage = () => {
    const navigate = useNavigate();
    const [form] = Form.useForm();
    const [loading, setLoading] = React.useState(false);
    const [error, setError] = React.useState('');

    React.useEffect(() => {
        let active = true;
        services.users
            .get()
            .then(user => {
                if (active && user.loggedIn) {
                    navigate(postLoginPath, {replace: true});
                }
            })
            .catch(() => undefined);
        return () => {
            active = false;
        };
    }, [navigate]);

    const submit = async (values: {username: string; password: string}) => {
        setLoading(true);
        setError('');
        try {
            await services.users.login(values.username, values.password);
            navigate(postLoginPath, {replace: true});
        } catch (err: any) {
            setError(err?.message || 'Login failed');
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className='login-screen'>
            <Card className='login-panel'>
                <div className='login-panel__brand'>
                    <BrandMark size='large' />
                    <Typography.Title level={3}>Athena</Typography.Title>
                </div>
                {error && <Alert type='error' title={error} showIcon={true} />}
                <Form form={form} layout='vertical' onFinish={submit}>
                    <Form.Item name='username' label='Username' rules={[{required: true}]}>
                        <Input autoComplete='username' />
                    </Form.Item>
                    <Form.Item name='password' label='Password' rules={[{required: true}]}>
                        <Input.Password autoComplete='current-password' />
                    </Form.Item>
                    <Button block={true} type='primary' htmlType='submit' loading={loading} icon={<LoginOutlined />}>
                        Log in
                    </Button>
                </Form>
            </Card>
        </div>
    );
};

export const UserInfoPage = () => {
    const ctx = React.useContext(Context);
    const navigate = useNavigate();
    const [loggingOut, setLoggingOut] = React.useState(false);
    const user = useAsyncData<UserInfo>(() => services.users.get() as any, []);
    const version = useAsyncData<VersionMessage & {version?: string}>(() => services.version.version() as any, []);
    const uiVersion = typeof SYSTEM_INFO === 'undefined' ? 'latest' : SYSTEM_INFO.version;
    const logout = async () => {
        setLoggingOut(true);
        ctx.notifications.info('Logging out');
        try {
            await services.users.logout();
            navigate('/login', {replace: true});
        } catch (err: any) {
            setLoggingOut(false);
            ctx.notifications.error('Logout failed', err?.message || 'Could not log out');
        }
    };
    return (
        <AppPage
            title='User Info'
            subtitle='Session, version, and account context'
            loading={user.loading || version.loading}
            error={user.error || version.error}
            onRefresh={() => {
                user.reload();
                version.reload();
            }}>
            <Section title='Current Session'>
                <KeyValueGrid
                    items={[
                        {label: 'Username', value: user.data?.username},
                        {label: 'Logged In', value: boolTag(user.data?.loggedIn)},
                        {label: 'Issuer', value: user.data?.iss || 'athena'},
                        {label: 'UI Version', value: uiVersion || '-'},
                        {label: 'Version', value: version.data?.Version || version.data?.version || '-'}
                    ]}
                />
            </Section>
            <Section title='Session'>
                <Button danger={true} loading={loggingOut} onClick={logout}>
                    Log out
                </Button>
            </Section>
        </AppPage>
    );
};

export const ProjectsPage = () => <AppPage title='Projects'>{null}</AppPage>;

export const WalletsPage = () => {
    const ctx = React.useContext(Context);
    const {page, pageSize, setPage} = usePagedParams();
    const [query, setQuery] = useKeywordParam();
    const [chain, setChain] = React.useState('');
    const [createOpen, setCreateOpen] = React.useState(false);
    const [secret, setSecret] = React.useState<WalletDetail>(null);
    const data = useAsyncData(() => services.wallet.listWallets({page, pageSize, query, chain: chain || undefined}), [page, pageSize, query, chain]);
    const canUpdateWallets = useCanI(rbacResources.wallets, rbacActions.update);
    const canRevealWallets = useCanI(rbacResources.wallets, rbacActions.invoke);
    const canCreateWallet = canUpdateWallets.data === true;
    const canRevealWallet = canRevealWallets.data === true;
    const reveal = async (id: number) => {
        if (!canRevealWallet) {
            return;
        }
        setSecret(await services.wallet.getWallet(id, true));
    };
    const create = async (values: {chain: string; alias?: string}) => {
        if (!canCreateWallet) {
            return;
        }
        await services.wallet.createWallet(values.chain, values.alias || '');
        setCreateOpen(false);
        ctx.notifications.success('Wallet created');
        data.reload();
    };
    const columns: ColumnsType<WalletItem> = [
        {title: 'Alias', dataIndex: 'alias'},
        {title: 'Chain', dataIndex: 'chain'},
        {title: 'Address', render: item => <TruncatedText value={item.address} copyable={true} />},
        {title: 'Source', dataIndex: 'source'},
        {
            title: 'Actions',
            render: item => (
                <Button icon={<EyeOutlined />} disabled={!canRevealWallet} onClick={() => reveal(item.id)}>
                    Reveal
                </Button>
            )
        }
    ];
    return (
        <AppPage
            title='Wallets'
            subtitle='Private key inventory and operational wallet lookup'
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}
            extra={
                <Button type='primary' icon={<PlusOutlined />} disabled={!canCreateWallet} onClick={() => setCreateOpen(true)}>
                    Create
                </Button>
            }
            filters={
                <Space wrap={true}>
                    <SearchBar value={query} onChange={setQuery} placeholder='Address or alias' />
                    <Select
                        allowClear={true}
                        value={chain || undefined}
                        placeholder='Chain'
                        style={{width: 150}}
                        onChange={value => setChain(value || '')}
                        options={['ETH', 'BSC', 'BASE', 'SOLANA'].map(value => ({value, label: value}))}
                    />
                </Space>
            }>
            <ResponsiveResourceList
                rowKey='id'
                items={data.data?.items || []}
                columns={columns}
                loading={data.loading}
                total={data.data?.total}
                page={page}
                pageSize={pageSize}
                onPageChange={setPage}
                card={item => (
                    <>
                        <CardTitle title={item.alias || item.address} subtitle={<TruncatedText value={item.address} copyable={true} />} tags={<Tag>{item.chain}</Tag>} />
                        <MetricRow
                            items={[
                                {label: 'Source', value: item.source},
                                {label: 'Created', value: item.createdAt}
                            ]}
                        />
                        <InlineActions>
                            <Button size='small' icon={<EyeOutlined />} disabled={!canRevealWallet} onClick={() => reveal(item.id)}>
                                Reveal
                            </Button>
                        </InlineActions>
                    </>
                )}
            />
            <Modal open={createOpen} title='Create Wallet' footer={null} onCancel={() => setCreateOpen(false)}>
                <Form layout='vertical' onFinish={create}>
                    <Form.Item name='chain' label='Chain' rules={[{required: true}]}>
                        <Select options={['ETH', 'BSC', 'BASE', 'SOLANA'].map(value => ({value, label: value}))} />
                    </Form.Item>
                    <Form.Item name='alias' label='Alias'>
                        <Input />
                    </Form.Item>
                    <Button type='primary' htmlType='submit' disabled={!canCreateWallet}>
                        Create
                    </Button>
                </Form>
            </Modal>
            <Modal open={!!secret} title='Wallet Secret' onCancel={() => setSecret(null)} footer={<Button onClick={() => setSecret(null)}>Close</Button>}>
                <KeyValueGrid
                    items={[
                        {label: 'Address', value: <TruncatedText value={secret?.address} copyable={true} />},
                        {label: 'Private Key', value: <TruncatedText value={secret?.privateKey} copyable={true} />},
                        {label: 'Mnemonic', value: <TruncatedText value={secret?.mnemonic} copyable={true} />}
                    ]}
                />
            </Modal>
        </AppPage>
    );
};

export const WalletBlacklistPage = () => {
    const ctx = React.useContext(Context);
    const data = useAsyncData(() => services.wallet.listWalletBlacklistEntries(), []);
    const canUpdate = useCanI(rbacResources.wallets, rbacActions.update);
    const canModify = canUpdate.data === true;
    const add = async (values: {wallet: string; note?: string}) => {
        if (!canModify) {
            return;
        }
        await services.wallet.addWalletBlacklistEntry(values.wallet, values.note || '');
        ctx.notifications.success('Wallet blacklisted');
        data.reload();
    };
    const remove = (wallet: string) => {
        if (!canModify) {
            return;
        }
        ctx.modal.confirm({
            title: 'Delete wallet blacklist entry?',
            content: wallet,
            onOk: async () => {
                await services.wallet.deleteWalletBlacklistEntry(wallet);
                data.reload();
            }
        });
    };
    return (
        <BlacklistPage
            title='Wallet Blacklist'
            items={data.data || []}
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}
            idField='wallet'
            add={add}
            remove={remove}
            canModify={canModify}
        />
    );
};

const BlacklistPage = <T extends WalletBlacklistEntry | BytecodeBlacklistEntry>(props: {
    title: string;
    items: T[];
    loading?: boolean;
    error?: Error;
    onRefresh: () => void;
    idField: 'wallet' | 'codeHash';
    add: (values: any) => Promise<void>;
    remove: (id: string) => void;
    canModify: boolean;
}) => {
    const [form] = Form.useForm();
    const idLabel = props.idField === 'wallet' ? 'Wallet' : 'Contract or Code Hash';
    const columns: ColumnsType<T> = [
        {title: idLabel, render: item => <TruncatedText value={(item as any)[props.idField]} copyable={true} />},
        {title: 'Note', dataIndex: 'note'},
        {title: 'Created', dataIndex: 'createdAt'},
        {
            title: 'Actions',
            render: item => (
                <Button danger={true} icon={<DeleteOutlined />} disabled={!props.canModify} onClick={() => props.remove((item as any)[props.idField])}>
                    Delete
                </Button>
            )
        }
    ];
    return (
        <AppPage
            title={props.title}
            loading={props.loading}
            error={props.error}
            onRefresh={props.onRefresh}
            filters={
                <Form
                    form={form}
                    layout='inline'
                    onFinish={async values => {
                        await props.add(values);
                        form.resetFields();
                    }}>
                    <Form.Item name={props.idField === 'wallet' ? 'wallet' : 'sourceContract'} rules={[{required: true}]}>
                        <Input placeholder={idLabel} />
                    </Form.Item>
                    {props.idField === 'codeHash' && (
                        <Form.Item name='sourceChainID'>
                            <InputNumber placeholder='Chain ID' />
                        </Form.Item>
                    )}
                    <Form.Item name='note'>
                        <Input placeholder='Note' />
                    </Form.Item>
                    <Button type='primary' htmlType='submit' icon={<PlusOutlined />} disabled={!props.canModify}>
                        Add
                    </Button>
                </Form>
            }>
            <ResponsiveResourceList
                rowKey={item => (item as any)[props.idField] || Math.random()}
                items={props.items}
                columns={columns}
                loading={props.loading}
                card={item => (
                    <>
                        <CardTitle title={<TruncatedText value={(item as any)[props.idField]} copyable={true} />} subtitle={(item as any).note} />
                        <MetricRow items={[{label: 'Created', value: (item as any).createdAt}]} />
                        <InlineActions>
                            <Button size='small' danger={true} icon={<DeleteOutlined />} disabled={!props.canModify} onClick={() => props.remove((item as any)[props.idField])}>
                                Delete
                            </Button>
                        </InlineActions>
                    </>
                )}
            />
        </AppPage>
    );
};

export const BytecodesPage = () => {
    const navigate = useNavigate();
    const {page, pageSize, setPage} = usePagedParams();
    const [codeHash, setCodeHash] = useKeywordParam('codeHash');
    const data = useAsyncData(() => services.athenaApplication.listBytecodes({page, pageSize, codeHash: codeHash || undefined}), [page, pageSize, codeHash]);
    const columns: ColumnsType<BytecodeListItem> = [
        {title: 'Code Hash', render: item => <Link to={`/application/bytecodes/${encodeURIComponent(item.codeHash || '')}`}>{short(item.codeHash)}</Link>},
        {title: 'Deployments', dataIndex: 'deploymentCount'},
        {title: 'Runtime Size', dataIndex: 'runtimeBytecodeSize'},
        {title: 'Open Source', render: item => boolTag(item.isOpenSource)},
        {title: 'Blacklisted', render: item => dangerTag(item.isBytecodeBlacklisted)}
    ];
    return (
        <AppPage
            title='Bytecodes'
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}
            filters={<SearchBar value={codeHash} onChange={setCodeHash} placeholder='Code hash' />}>
            <ResponsiveResourceList
                rowKey={item => item.codeHash || Math.random()}
                items={data.data?.items || []}
                columns={columns}
                loading={data.loading}
                total={data.data?.total}
                page={page}
                pageSize={pageSize}
                onPageChange={setPage}
                card={item => (
                    <div onClick={() => navigate(`/application/bytecodes/${encodeURIComponent(item.codeHash || '')}`)}>
                        <CardTitle
                            title={short(item.codeHash)}
                            subtitle={<TruncatedText value={item.codeHash} copyable={true} />}
                            tags={item.isBytecodeBlacklisted ? <Tag color='red'>Blacklisted</Tag> : <Tag>Tracked</Tag>}
                        />
                        <MetricRow
                            items={[
                                {label: 'Deployments', value: item.deploymentCount},
                                {label: 'Size', value: item.runtimeBytecodeSize},
                                {label: 'Open Source', value: fmt(item.isOpenSource)}
                            ]}
                        />
                    </div>
                )}
            />
        </AppPage>
    );
};

export const BytecodeDetailPage = () => {
    const {codeHash = ''} = useParams();
    const decoded = decodeURIComponent(codeHash);
    const detail = useAsyncData<any>(
        () =>
            Promise.all([services.athenaApplication.getBytecode(decoded), services.athenaApplication.listBytecodeDeployments(decoded, {page: 1, pageSize: 20})]).then(
                ([bytecode, deployments]) => ({bytecode, deployments})
            ) as any,
        [decoded]
    );
    const deploymentColumns: ColumnsType<BytecodeDeployment> = [
        {title: 'Chain', dataIndex: 'chainID'},
        {title: 'Contract', render: item => <TruncatedText value={item.contract} copyable={true} />},
        {title: 'First Seen', dataIndex: 'firstSeenAt'},
        {title: 'Updated', dataIndex: 'updatedAt'}
    ];
    return (
        <AppPage title='Bytecode Detail' subtitle={<TruncatedText value={decoded} copyable={true} />} loading={detail.loading} error={detail.error} onRefresh={detail.reload}>
            <Section title='Summary'>
                <KeyValueGrid
                    items={[
                        {label: 'Code Hash', value: <TruncatedText value={decoded} copyable={true} />},
                        {label: 'Runtime Size', value: detail.data?.bytecode.runtimeBytecodeSize},
                        {label: 'Open Source', value: boolTag(detail.data?.bytecode.isOpenSource)},
                        {label: 'Blacklisted', value: dangerTag(detail.data?.bytecode.isBytecodeBlacklisted)}
                    ]}
                />
            </Section>
            <Tabs
                items={[
                    {
                        key: 'deployments',
                        label: 'Deployments',
                        children: (
                            <ResponsiveResourceList
                                rowKey={item => `${item.chainID}-${item.contract}`}
                                items={detail.data?.deployments.items || []}
                                columns={deploymentColumns}
                                card={item => <CardTitle title={item.contract} subtitle={`Chain ${item.chainID}`} />}
                            />
                        )
                    },
                    {key: 'runtime', label: 'Runtime', children: <pre className='code-block'>{detail.data?.bytecode.runtimeBytecode || 'No runtime bytecode available'}</pre>},
                    {key: 'source', label: 'Source', children: <pre className='code-block'>{detail.data?.bytecode.sourceCode || 'No source available'}</pre>},
                    {key: 'report', label: 'Report', children: <pre className='code-block'>{detail.data?.bytecode.sourceQualityReport || 'No report available'}</pre>}
                ]}
            />
        </AppPage>
    );
};

export const BytecodeBlacklistPage = () => {
    const ctx = React.useContext(Context);
    const data = useAsyncData(() => services.athenaApplication.listBytecodeBlacklistEntries(), []);
    const canUpdate = useCanI(rbacResources.application, rbacActions.update);
    const canModify = canUpdate.data === true;
    const add = async (values: {sourceContract: string; note?: string; sourceChainID?: number}) => {
        if (!canModify) {
            return;
        }
        await services.athenaApplication.addBytecodeBlacklistEntry(values.sourceContract, values.note || '', values.sourceChainID);
        ctx.notifications.success('Bytecode blacklisted');
        data.reload();
    };
    const remove = (codeHash: string) => {
        if (!canModify) {
            return;
        }
        ctx.modal.confirm({
            title: 'Delete bytecode blacklist entry?',
            content: codeHash,
            onOk: async () => {
                await services.athenaApplication.deleteBytecodeBlacklist(codeHash);
                data.reload();
            }
        });
    };
    return (
        <BlacklistPage
            title='Bytecode Blacklist'
            items={data.data || []}
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}
            idField='codeHash'
            add={add}
            remove={remove}
            canModify={canModify}
        />
    );
};

export const SourceQualityPromptsPage = () => {
    const ctx = React.useContext(Context);
    const [editing, setEditing] = React.useState<SourceQualityPrompt>(null);
    const data = useAsyncData(() => services.athenaApplication.listSourceQualityPrompts(), []);
    const canUpdate = useCanI(rbacResources.application, rbacActions.update);
    const canModify = canUpdate.data === true;
    const save = async (values: {name: string; systemPrompt: string}) => {
        if (!canModify) {
            return;
        }
        if (editing?.id) {
            await services.athenaApplication.updateSourceQualityPrompt(editing.id, values.name, values.systemPrompt);
        } else {
            await services.athenaApplication.createSourceQualityPrompt(values.name, values.systemPrompt);
        }
        setEditing(null);
        data.reload();
    };
    return (
        <AppPage
            title='Source Quality Prompts'
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}
            extra={
                <Button type='primary' icon={<PlusOutlined />} disabled={!canModify} onClick={() => setEditing({})}>
                    New
                </Button>
            }>
            <ResponsiveResourceList
                rowKey={item => item.id || Math.random()}
                items={data.data || []}
                columns={[
                    {title: 'Version', dataIndex: 'version'},
                    {title: 'Name', dataIndex: 'name'},
                    {title: 'Active', render: item => boolTag(item.isActive)},
                    {title: 'Updated', dataIndex: 'updatedAt'},
                    {
                        title: 'Actions',
                        render: item => (
                            <Space>
                                <Button icon={<SaveOutlined />} disabled={!canModify} onClick={() => setEditing(item)}>
                                    Edit
                                </Button>
                                <Button
                                    icon={<CheckCircleOutlined />}
                                    disabled={!canModify || !item.id}
                                    onClick={async () => {
                                        await services.athenaApplication.activateSourceQualityPrompt(item.id || 0);
                                        ctx.notifications.success('Prompt activated');
                                        data.reload();
                                    }}>
                                    Activate
                                </Button>
                            </Space>
                        )
                    }
                ]}
                card={item => (
                    <>
                        <CardTitle title={item.name} subtitle={`v${item.version || '-'}`} tags={item.isActive ? <Tag color='green'>Active</Tag> : <Tag>Draft</Tag>} />
                        <InlineActions>
                            <Button size='small' disabled={!canModify} onClick={() => setEditing(item)}>
                                Edit
                            </Button>
                        </InlineActions>
                    </>
                )}
            />
            <Modal open={!!editing} title={editing?.id ? 'Edit Prompt' : 'New Prompt'} footer={null} onCancel={() => setEditing(null)} width={760}>
                <Form layout='vertical' initialValues={editing || {}} onFinish={save}>
                    <Form.Item name='name' label='Name' rules={[{required: true}]}>
                        <Input />
                    </Form.Item>
                    <Form.Item name='systemPrompt' label='System Prompt' rules={[{required: true}]}>
                        <Input.TextArea rows={10} />
                    </Form.Item>
                    <Button type='primary' htmlType='submit' disabled={!canModify}>
                        Save
                    </Button>
                </Form>
            </Modal>
        </AppPage>
    );
};

export const WormPage = () => {
    const [sortOption, setSortOption] = React.useState('trending');
    const [categorySlug, setCategorySlug] = React.useState('all');
    const [detailId, setDetailId] = React.useState('');
    const data = useAsyncData(() => services.worm.listMarkets({limit: 50, sortOption: sortOption as any, categorySlug: categorySlug as any}), [sortOption, categorySlug]);
    const detail = useAsyncData<WormMarketDetail>(() => (detailId ? services.worm.getMarket(detailId) : Promise.resolve(null as WormMarketDetail)) as any, [detailId]);
    const columns: ColumnsType<WormMarketItem> = [
        {
            title: 'Market',
            render: item => <WormMarketSummary item={item} onOpen={() => setDetailId(item.conditionId)} />
        },
        {title: 'Category', dataIndex: 'category'},
        {title: 'State', dataIndex: 'state'},
        {title: 'Last Price', dataIndex: 'lastTradePrice'},
        {title: 'Margin', render: item => boolTag(item.marginEnabled)}
    ];
    return (
        <AppPage
            title='Worm'
            subtitle='Polymarket-derived markets for Worm module'
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}
            filters={
                <Space wrap={true}>
                    <Select
                        value={sortOption}
                        style={{width: 160}}
                        onChange={setSortOption}
                        options={['new', 'trending', 'ending_soon', 'leverage'].map(value => ({value, label: value}))}
                    />
                    <Select
                        value={categorySlug}
                        style={{width: 150}}
                        onChange={setCategorySlug}
                        options={['all', 'politics', 'sports', 'crypto', 'tech', 'finance', 'wtf'].map(value => ({value, label: value}))}
                    />
                </Space>
            }>
            <ResponsiveResourceList
                rowKey='conditionId'
                items={data.data?.items || []}
                columns={columns}
                loading={data.loading}
                card={item => (
                    <div onClick={() => setDetailId(item.conditionId)}>
                        <WormMarketSummary item={item} />
                        <MetricRow
                            items={[
                                {label: 'Price', value: item.lastTradePrice},
                                {label: 'State', value: item.state},
                                {label: 'Created', value: item.created}
                            ]}
                        />
                    </div>
                )}
            />
            <Modal
                className='worm-detail-modal'
                open={!!detailId}
                title='Market Detail'
                onCancel={() => setDetailId('')}
                footer={<Button onClick={() => setDetailId('')}>Close</Button>}
                width={860}>
                <WormDetail detail={detail.data} />
            </Modal>
        </AppPage>
    );
};

const WormDetail = (props: {detail?: WormMarketDetail}) => {
    const detail = props.detail;
    if (!detail) {
        return null;
    }
    const image = wormMarketLogo(detail.market);
    return (
        <Space className='worm-detail' orientation='vertical' style={{width: '100%'}}>
            <div className='worm-detail__header'>
                {image && <img src={image} alt='' />}
                <div className='worm-detail__heading'>
                    <Typography.Title level={4}>{detail.market.title || '-'}</Typography.Title>
                    <Space className='worm-detail__meta' wrap={true}>
                        {detail.market.eventTitle && <Tag>{detail.market.eventTitle}</Tag>}
                        {detail.market.category && <Tag>{detail.market.category}</Tag>}
                        {detail.market.state && <Tag>{detail.market.state}</Tag>}
                        {detail.market.marginEnabled && <Tag color='green'>Margin</Tag>}
                    </Space>
                </div>
            </div>
            <KeyValueGrid
                columns={2}
                items={[
                    {label: 'Condition', value: <TruncatedText value={detail.market.conditionId} copyable={true} />},
                    {label: 'Resolution', value: detail.resolutionDate},
                    {label: 'Maker Fee', value: detail.makerFee},
                    {label: 'Taker Fee', value: detail.takerFee},
                    {label: 'Stale', value: boolTag(detail.stale)}
                ]}
            />
            <Collapse
                items={[
                    {key: 'rules', label: 'Rules', children: <pre className='code-block'>{(detail.rules || []).join('\n\n') || 'No rules'}</pre>},
                    {key: 'config', label: 'Config', children: <pre className='code-block'>{JSON.stringify(detail.config || {}, null, 2)}</pre>}
                ]}
            />
        </Space>
    );
};

const PolymarketListPage = <T extends PolymarketHotMarketItem | PolymarketRealtimeMarketItem | PolymarketMoverMarketItem>(props: {
    title: string;
    load: () => Promise<{items: T[]; fetchedAt?: number; stale?: boolean; [key: string]: any}> & {abort?: () => void};
}) => {
    const data = useAsyncData(props.load, []);
    const columns: ColumnsType<T> = [
        {title: 'Market', render: item => <CardTitle title={(item as any).question} subtitle={(item as any).eventSlug} image={(item as any).image} />},
        {title: 'Volume', render: item => fmtNumber((item as any).volume24hr || (item as any).volumeNum)},
        {title: 'Liquidity', render: item => fmtNumber((item as any).liquidityNum)},
        {title: 'Spread', render: item => fmt((item as any).spread)},
        {title: 'Updated', dataIndex: 'updatedAt'}
    ];
    return (
        <AppPage
            title={props.title}
            subtitle={`Fetched ${fmt(data.data?.fetchedAt)} ${data.data?.stale ? '(stale)' : ''}`}
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}>
            <ResponsiveResourceList
                rowKey={item => (item as any).conditionId}
                items={data.data?.items || []}
                columns={columns}
                loading={data.loading}
                card={item => (
                    <>
                        <CardTitle title={(item as any).question} subtitle={(item as any).eventSlug} image={(item as any).image} />
                        <MetricRow
                            items={[
                                {label: 'Volume', value: fmtNumber((item as any).volume24hr || (item as any).volumeNum)},
                                {label: 'Liquidity', value: fmtNumber((item as any).liquidityNum)},
                                {label: 'Tokens', value: (item as any).tokens?.length || 0}
                            ]}
                        />
                    </>
                )}
            />
        </AppPage>
    );
};

export const PolymarketHotPage = () => <PolymarketListPage title='Polymarket Hot Markets' load={() => services.polymarket.listHotMarkets(100)} />;
export const PolymarketRealtimePage = () => <PolymarketListPage title='Polymarket Realtime' load={() => services.polymarket.listRealtimeMarkets(100)} />;
export const PolymarketMoversPage = () => <PolymarketListPage title='Polymarket Movers' load={() => services.polymarket.listMovers(100)} />;

export const PolymarketSportsLivePage = () => {
    const [view, setView] = React.useState<'events' | 'markets'>('events');
    const events = useAsyncData(() => services.polymarket.getSportsLiveSnapshot(30), []);
    const markets = useAsyncData(() => services.polymarket.listSportsLiveMarkets(200), []);
    const eventColumns: ColumnsType<PolymarketSportsLiveEventItem> = [
        {title: 'Event', render: item => <CardTitle title={item.title} subtitle={item.gameStatus || item.period} image={item.image} />},
        {title: 'Score', dataIndex: 'score'},
        {title: 'Live', render: item => boolTag(item.live)},
        {title: 'Markets', render: item => item.markets?.reduce((sum: number, group: {markets: unknown[]}) => sum + group.markets.length, 0)}
    ];
    const marketColumns: ColumnsType<PolymarketSportsLiveMarketItem> = [
        {title: 'Market', render: item => <CardTitle title={item.title} subtitle={item.eventSlug} image={item.image} />},
        {title: 'Score', dataIndex: 'score'},
        {title: 'Volume', dataIndex: 'volumeNum'},
        {title: 'Liquidity', dataIndex: 'liquidityNum'}
    ];
    return (
        <AppPage
            title='Sports Live'
            loading={events.loading || markets.loading}
            error={events.error || markets.error}
            onRefresh={() => {
                events.reload();
                markets.reload();
            }}
            filters={
                <Select
                    value={view}
                    style={{width: 160}}
                    onChange={setView}
                    options={[
                        {value: 'events', label: 'Events'},
                        {value: 'markets', label: 'Markets'}
                    ]}
                />
            }>
            {view === 'events' ? (
                <ResponsiveResourceList
                    rowKey='eventSlug'
                    items={events.data?.events || []}
                    columns={eventColumns}
                    loading={events.loading}
                    card={item => (
                        <>
                            <CardTitle
                                title={item.title}
                                subtitle={item.score || item.gameStatus}
                                image={item.image}
                                tags={item.live ? <Tag color='red'>Live</Tag> : <Tag>{item.period}</Tag>}
                            />
                            <MetricRow
                                items={[
                                    {label: 'Elapsed', value: item.elapsed},
                                    {label: 'Markets', value: item.markets?.length || 0}
                                ]}
                            />
                        </>
                    )}
                />
            ) : (
                <ResponsiveResourceList
                    rowKey='conditionId'
                    items={markets.data?.items || []}
                    columns={marketColumns}
                    loading={markets.loading}
                    card={item => (
                        <>
                            <CardTitle title={item.title} subtitle={item.score || item.eventSlug} image={item.image} />
                            <MetricRow
                                items={[
                                    {label: 'Volume', value: fmtNumber(item.volumeNum)},
                                    {label: 'Liquidity', value: fmtNumber(item.liquidityNum)}
                                ]}
                            />
                        </>
                    )}
                />
            )}
        </AppPage>
    );
};

export const NotificationsPage = () => {
    const navigate = useNavigate();
    const {isMobile} = useBreakpoint();
    const {page, pageSize, setPage} = usePagedParams();
    const [keyword, setKeyword] = useKeywordParam('keyword');
    const [status, setStatus] = React.useState('');
    const data = useAsyncData(() => services.notification.listNotifications({page, pageSize, keyword, status: status || undefined}), [page, pageSize, keyword, status]);
    const canInvoke = useCanI(rbacResources.notifications, rbacActions.invoke);
    const canSendTest = canInvoke.data === true;
    const sendTest = async (topic: string) => {
        if (!canSendTest) {
            return;
        }
        await services.notification.sendTestNotification(topic);
        data.reload();
    };
    const testNotificationActions = isMobile ? (
        <Dropdown
            menu={{
                items: notificationTestTopics.map(item => ({key: item.topic, label: item.label, disabled: !canSendTest})),
                onClick: item => void sendTest(item.key)
            }}
            trigger={['click']}>
            <Button icon={<SendOutlined />} disabled={!canSendTest}>
                Test
            </Button>
        </Dropdown>
    ) : (
        <Space>
            {notificationTestTopics.map(item => (
                <Button key={item.topic} icon={<SendOutlined />} disabled={!canSendTest} onClick={() => void sendTest(item.topic)}>
                    {item.label}
                </Button>
            ))}
        </Space>
    );
    const columns: ColumnsType<NotificationDelivery> = [
        {
            title: 'Title',
            render: item => (
                <Button type='link' onClick={() => navigate(`/notifications/${item.id}`)}>
                    {item.title || item.topic}
                </Button>
            )
        },
        {title: 'Severity', dataIndex: 'severity'},
        {title: 'Topic', dataIndex: 'topic'},
        {title: 'Status', dataIndex: 'status'},
        {title: 'Channel', dataIndex: 'channel'},
        {title: 'Created', dataIndex: 'createdAt'}
    ];
    return (
        <AppPage
            title='Notifications'
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}
            extra={testNotificationActions}
            filters={
                <Space wrap={true}>
                    <SearchBar value={keyword} onChange={setKeyword} placeholder='Keyword' />
                    <Select
                        allowClear={true}
                        value={status || undefined}
                        style={{width: 150}}
                        placeholder='Status'
                        onChange={value => setStatus(value || '')}
                        options={['pending', 'sent', 'failed'].map(value => ({value, label: value}))}
                    />
                </Space>
            }>
            <ResponsiveResourceList
                rowKey='id'
                items={data.data?.items || []}
                columns={columns}
                loading={data.loading}
                total={data.data?.total}
                page={page}
                pageSize={pageSize}
                onPageChange={setPage}
                card={item => (
                    <div onClick={() => navigate(`/notifications/${item.id}`)}>
                        <CardTitle title={item.title || item.topic} subtitle={item.body} tags={<Tag>{item.status}</Tag>} />
                        <MetricRow
                            items={[
                                {label: 'Severity', value: item.severity},
                                {label: 'Channel', value: item.channel},
                                {label: 'Created', value: item.createdAt}
                            ]}
                        />
                    </div>
                )}
            />
        </AppPage>
    );
};

export const NotificationsDetailPage = () => {
    const {id = ''} = useParams();
    const data = useAsyncData(() => services.notification.getNotification(id), [id]);
    return (
        <AppPage title='Notification Detail' loading={data.loading} error={data.error} onRefresh={data.reload}>
            <Section title='Delivery'>
                <KeyValueGrid
                    items={Object.entries(data.data || {}).map(([label, value]) => ({
                        label,
                        value:
                            label === 'link' ? (
                                <a href={String(value)} target='_blank' rel='noreferrer'>
                                    {String(value)}
                                </a>
                            ) : (
                                fmt(value)
                            )
                    }))}
                />
            </Section>
        </AppPage>
    );
};

export const SettingsPage = () => {
    const user = useAsyncData<UserInfo>(() => services.users.get() as any, []);
    const accounts = useAsyncData<Account[]>(() => services.accounts.list() as any, []);
    const discovery = useAsyncData(() => services.athenaApplication.getProjectDiscoveryStatus(), []);
    const canUpdateDiscovery = useCanI(rbacResources.applicationDiscovery, rbacActions.update);
    const canToggleDiscovery = canUpdateDiscovery.data === true;
    const visibleAccounts = visibleAccountsForUser(accounts.data || [], user.data);
    const toggleDiscovery = async () => {
        if (!canToggleDiscovery) {
            return;
        }
        if (discovery.data?.started) {
            await services.athenaApplication.stopProjectDiscovery();
        } else {
            await services.athenaApplication.startProjectDiscovery();
        }
        discovery.reload();
    };
    return (
        <AppPage
            title='Settings'
            loading={user.loading || accounts.loading || discovery.loading}
            error={user.error || accounts.error || discovery.error}
            onRefresh={() => {
                user.reload();
                accounts.reload();
                discovery.reload();
            }}>
            <Section
                title='Application Discovery'
                extra={
                    <Button icon={discovery.data?.started ? <StopOutlined /> : <ApiOutlined />} disabled={!canToggleDiscovery} onClick={toggleDiscovery}>
                        {discovery.data?.started ? 'Stop' : 'Start'}
                    </Button>
                }>
                <KeyValueGrid
                    items={[
                        {label: 'Started', value: boolTag(discovery.data?.started)},
                        {label: 'Status', value: discovery.data?.status}
                    ]}
                />
            </Section>
            <Section title='Accounts'>
                <ResponsiveResourceList
                    rowKey='name'
                    items={visibleAccounts}
                    columns={[
                        {title: 'Name', dataIndex: 'name'},
                        {title: 'Enabled', render: item => boolTag(item.enabled)},
                        {title: 'Capabilities', render: item => (item.capabilities || []).join(', ')}
                    ]}
                    card={item => (
                        <CardTitle title={item.name} subtitle={(item.capabilities || []).join(', ')} tags={item.enabled ? <Tag color='green'>Enabled</Tag> : <Tag>Disabled</Tag>} />
                    )}
                />
            </Section>
        </AppPage>
    );
};

export const HelpPage = () => (
    <AppPage title='Help' subtitle='Operational reference links'>
        <Section title='Resources'>
            <Space orientation='vertical'>
                <a href='swagger-ui'>Swagger UI</a>
                <Typography.Text type='secondary'>ATHENA mobile UI is optimized for browsing, search, detail inspection, and common operations.</Typography.Text>
            </Space>
        </Section>
    </AppPage>
);
