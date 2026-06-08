import {DeleteOutlined, EditOutlined, EyeOutlined, LoginOutlined, PlusOutlined, SendOutlined} from '@ant-design/icons';
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
import {
    TokenAPIBytecodeBlacklist,
    TokenAPIChainIngestCheckpoint,
    TokenAPIContractCode,
    TokenAPIProject,
    TokenAPIProjectDataCollectionTask,
    TokenAPIWalletBlacklist
} from '../shared/services/tokenapi-service';
import {WalletDetail, WalletItem} from '../shared/services/wallet-service';
import {WormMarketDetail, WormMarketItem} from '../shared/services/worm-service';
import bscIcon from '../../assets/images/bsc.png';
import ethIcon from '../../assets/images/eth.png';
import solanaIcon from '../../assets/images/solana.png';

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

const chainIconAssets = {
    eth: ethIcon,
    bsc: bscIcon,
    solana: solanaIcon
};

const projectChainDisplayByID: Record<number, {label: string; icon?: string}> = {
    1: {label: 'ETH', icon: chainIconAssets.eth},
    56: {label: 'BSC', icon: chainIconAssets.bsc}
};

const chainLabel = (chainID?: number) => {
    if (chainID === undefined) {
        return '-';
    }
    return projectChainDisplayByID[chainID]?.label || String(chainID);
};

const ChainBadge = (props: {chainID?: number}) => {
    const display = props.chainID === undefined ? undefined : projectChainDisplayByID[props.chainID];
    return (
        <Tag className='chain-badge'>
            {display?.icon && <img src={display.icon} alt='' />}
            <span>{chainLabel(props.chainID)}</span>
        </Tag>
    );
};

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
    notifications: 'notifications',
    tokenapi: 'tokenapi',
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

export const ContractCodesPage = () => {
    const navigate = useNavigate();
    const {page, pageSize, setPage} = usePagedParams();
    const [codeHash, setCodeHash] = useKeywordParam('codeHash');
    const data = useAsyncData(() => services.tokenapi.listContractCodes({page, pageSize, codeHash: codeHash || undefined}), [page, pageSize, codeHash]);
    const columns: ColumnsType<TokenAPIContractCode> = [
        {title: 'Code Hash', render: item => <Link to={`/token/contract-codes/${encodeURIComponent(item.codeHash || '')}`}>{short(item.codeHash)}</Link>},
        {title: 'Deployments', dataIndex: 'deploymentCount'},
        {title: 'Source Hash', render: item => <TruncatedText value={item.sourceCodeHash} copyable={true} />},
        {title: 'Fetched', dataIndex: 'sourceCodeFetchedAt'},
        {title: 'Created', dataIndex: 'createdAt'}
    ];
    return (
        <AppPage
            title='Contract Codes'
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
                    <div onClick={() => navigate(`/token/contract-codes/${encodeURIComponent(item.codeHash || '')}`)}>
                        <CardTitle title={short(item.codeHash)} subtitle={<TruncatedText value={item.codeHash} copyable={true} />} />
                        <MetricRow
                            items={[
                                {label: 'Deployments', value: item.deploymentCount},
                                {label: 'Fetched', value: item.sourceCodeFetchedAt},
                                {label: 'Created', value: item.createdAt}
                            ]}
                        />
                    </div>
                )}
            />
        </AppPage>
    );
};

export const ProjectsPage = () => {
    const {page, pageSize, setPage} = usePagedParams();
    const [chainID, setChainID] = React.useState<number>();
    const [contract, setContract] = useKeywordParam('contract');
    const [codeHash, setCodeHash] = useKeywordParam('codeHash');
    const options = useAsyncData(() => services.tokenapi.getOptions(), []);
    const data = useAsyncData(
        () =>
            services.tokenapi.listProjects({
                page,
                pageSize,
                chainID,
                contract: contract || undefined,
                codeHash: codeHash || undefined
            }),
        [page, pageSize, chainID, contract, codeHash]
    );
    const columns: ColumnsType<TokenAPIProject> = [
        {title: 'ID', dataIndex: 'projectID'},
        {title: 'Chain', render: item => <ChainBadge chainID={item.chainID} />},
        {
            title: 'Token',
            render: item => (
                <Space direction='vertical' size={0}>
                    <Typography.Text strong={true}>{item.symbol || '-'}</Typography.Text>
                    <Typography.Text type='secondary'>{item.name || '-'}</Typography.Text>
                </Space>
            )
        },
        {title: 'Contract', render: item => <TruncatedText value={item.contract} copyable={true} />},
        {title: 'Creator', render: item => <TruncatedText value={item.creator} copyable={true} />},
        {title: 'Block', render: item => fmtNumber(item.blockNumber)},
        {title: 'Tx Index', dataIndex: 'txIndex'},
        {title: 'Code Hash', render: item => <TruncatedText value={item.codeHash} copyable={true} />},
        {title: 'Created', dataIndex: 'createdAt'}
    ];
    const chainOptions = (options.data?.chains || []).map(item => ({
        value: item.chainID,
        label: `${item.chainName || item.chainID} (${item.chainID})`
    }));
    return (
        <AppPage
            title='Projects'
            loading={data.loading || options.loading}
            error={data.error || options.error}
            onRefresh={() => {
                data.reload();
                options.reload();
            }}
            filters={
                <Space wrap={true}>
                    <Select
                        allowClear={true}
                        value={chainID}
                        placeholder='Chain'
                        style={{width: 220}}
                        options={chainOptions}
                        onChange={value => {
                            setChainID(value);
                            setPage(1, pageSize);
                        }}
                    />
                    <SearchBar value={contract} onChange={setContract} placeholder='Contract' />
                    <SearchBar value={codeHash} onChange={setCodeHash} placeholder='Code hash' />
                </Space>
            }>
            <ResponsiveResourceList
                rowKey={item => item.projectID || `${item.chainID}-${item.contract}`}
                items={data.data?.items || []}
                columns={columns}
                loading={data.loading}
                total={data.data?.total}
                page={page}
                pageSize={pageSize}
                onPageChange={setPage}
                card={item => (
                    <>
                        <CardTitle
                            title={`${item.symbol || '-'} #${item.projectID || '-'}`}
                            subtitle={<TruncatedText value={item.contract} copyable={true} />}
                            tags={<ChainBadge chainID={item.chainID} />}
                        />
                        <MetricRow
                            items={[
                                {label: 'Block', value: fmtNumber(item.blockNumber)},
                                {label: 'Tx Index', value: item.txIndex},
                                {label: 'Created', value: item.createdAt}
                            ]}
                        />
                        <KeyValueGrid
                            columns={1}
                            items={[
                                {label: 'Name', value: item.name},
                                {label: 'Creator', value: <TruncatedText value={item.creator} copyable={true} />},
                                {label: 'Tx Hash', value: <TruncatedText value={item.txHash} copyable={true} />},
                                {label: 'Code Hash', value: <TruncatedText value={item.codeHash} copyable={true} />}
                            ]}
                        />
                    </>
                )}
            />
        </AppPage>
    );
};

export const ContractCodeDetailPage = () => {
    const {codeHash = ''} = useParams();
    const decoded = decodeURIComponent(codeHash);
    const detail = useAsyncData(() => services.tokenapi.getContractCode(decoded), [decoded]);
    return (
        <AppPage title='Contract Code Detail' subtitle={<TruncatedText value={decoded} copyable={true} />} loading={detail.loading} error={detail.error} onRefresh={detail.reload}>
            <Section title='Summary'>
                <KeyValueGrid
                    items={[
                        {label: 'Code Hash', value: <TruncatedText value={decoded} copyable={true} />},
                        {label: 'Found', value: boolTag(!!detail.data)},
                        {label: 'Deployments', value: fmtNumber(detail.data?.deploymentCount)},
                        {label: 'Source Hash', value: <TruncatedText value={detail.data?.sourceCodeHash} copyable={true} />},
                        {label: 'Fetched', value: detail.data?.sourceCodeFetchedAt},
                        {label: 'Created', value: detail.data?.createdAt}
                    ]}
                />
            </Section>
            <Tabs items={[{key: 'source', label: 'Source', children: <pre className='code-block'>{detail.data?.sourceCode || 'No source available'}</pre>}]} />
        </AppPage>
    );
};

export const BytecodeBlacklistsPage = () => {
    const ctx = React.useContext(Context);
    const [form] = Form.useForm();
    const [editing, setEditing] = React.useState<TokenAPIBytecodeBlacklist>(null);
    const data = useAsyncData(() => services.tokenapi.listBytecodeBlacklists(), []);
    const options = useAsyncData(() => services.tokenapi.getOptions(), []);
    const canUpdate = useCanI(rbacResources.tokenapi, rbacActions.update);
    const canModify = canUpdate.data === true;
    const chainOptions = React.useMemo(
        () =>
            (options.data?.chains || [])
                .filter(item => item.chainID !== undefined)
                .map(item => ({
                    value: item.chainID,
                    label: item.chainName || item.chainID
                })),
        [options.data]
    );
    const chainNameByID = React.useMemo(() => {
        const names = new Map<number, string>();
        (options.data?.chains || []).forEach(item => {
            if (item.chainID !== undefined && item.chainName) {
                names.set(item.chainID, item.chainName);
            }
        });
        return names;
    }, [options.data]);
    const chainLabel = React.useCallback((chainID?: number) => (chainID === undefined ? '-' : chainNameByID.get(chainID) || chainID), [chainNameByID]);
    const refresh = React.useCallback(() => {
        options.reload();
        data.reload();
    }, [data, options]);
    const add = async (values: {note?: string; sourceChainID?: number; sourceContract?: string}) => {
        if (!canModify) {
            return;
        }
        await services.tokenapi.createBytecodeBlacklist(values);
        ctx.notifications.success('Bytecode blacklisted');
        form.resetFields();
        data.reload();
    };
    const saveNote = async (values: {note?: string}) => {
        if (!canModify || !editing?.codeHash) {
            return;
        }
        await services.tokenapi.updateBytecodeBlacklist(editing.codeHash, values.note || '');
        setEditing(null);
        data.reload();
    };
    const remove = (item: TokenAPIBytecodeBlacklist) => {
        if (!canModify) {
            return;
        }
        ctx.modal.confirm({
            title: 'Delete bytecode blacklist entry?',
            content: item.codeHash,
            onOk: async () => {
                await services.tokenapi.deleteBytecodeBlacklist(item.codeHash || '');
                data.reload();
            }
        });
    };
    const columns: ColumnsType<TokenAPIBytecodeBlacklist> = [
        {title: 'Code Hash', render: item => <TruncatedText value={item.codeHash} copyable={true} />},
        {title: 'Note', dataIndex: 'note'},
        {title: 'Source Chain', render: item => chainLabel(item.sourceChainID)},
        {title: 'Source Contract', render: item => <TruncatedText value={item.sourceContract} copyable={true} />},
        {title: 'Created', dataIndex: 'createdAt'},
        {
            title: 'Actions',
            render: item => (
                <Space>
                    <Button icon={<EditOutlined />} disabled={!canModify || !item.codeHash} onClick={() => setEditing(item)}>
                        Edit Note
                    </Button>
                    <Button danger={true} icon={<DeleteOutlined />} disabled={!canModify || !item.codeHash} onClick={() => remove(item)}>
                        Delete
                    </Button>
                </Space>
            )
        }
    ];
    return (
        <AppPage
            title='Bytecode Blacklists'
            loading={data.loading || options.loading}
            error={data.error || options.error}
            onRefresh={refresh}
            filters={
                <Form form={form} layout='inline' onFinish={add}>
                    <Form.Item name='sourceChainID' rules={[{required: true}]}>
                        <Select placeholder='Source chain' options={chainOptions} style={{minWidth: 180}} />
                    </Form.Item>
                    <Form.Item name='sourceContract' rules={[{required: true}]}>
                        <Input placeholder='Source contract' />
                    </Form.Item>
                    <Form.Item name='note'>
                        <Input placeholder='Note' />
                    </Form.Item>
                    <Button type='primary' htmlType='submit' icon={<PlusOutlined />} disabled={!canModify}>
                        Add
                    </Button>
                </Form>
            }>
            <ResponsiveResourceList
                rowKey={item => item.codeHash || Math.random()}
                items={data.data || []}
                columns={columns}
                loading={data.loading}
                card={item => (
                    <>
                        <CardTitle title={<TruncatedText value={item.codeHash} copyable={true} />} subtitle={item.note} />
                        <MetricRow
                            items={[
                                {label: 'Source Chain', value: chainLabel(item.sourceChainID)},
                                {label: 'Created', value: item.createdAt}
                            ]}
                        />
                        <InlineActions>
                            <Button size='small' icon={<EditOutlined />} disabled={!canModify || !item.codeHash} onClick={() => setEditing(item)}>
                                Edit Note
                            </Button>
                            <Button size='small' danger={true} icon={<DeleteOutlined />} disabled={!canModify || !item.codeHash} onClick={() => remove(item)}>
                                Delete
                            </Button>
                        </InlineActions>
                    </>
                )}
            />
            <Modal open={!!editing} title='Edit Bytecode Note' footer={null} onCancel={() => setEditing(null)}>
                <Form key={editing?.codeHash || 'bytecode-note'} layout='vertical' initialValues={editing || {}} onFinish={saveNote}>
                    <Form.Item label='Code Hash'>
                        <TruncatedText value={editing?.codeHash} copyable={true} />
                    </Form.Item>
                    <Form.Item name='note' label='Note'>
                        <Input.TextArea rows={4} />
                    </Form.Item>
                    <Button type='primary' htmlType='submit' disabled={!canModify}>
                        Save
                    </Button>
                </Form>
            </Modal>
        </AppPage>
    );
};

export const WalletBlacklistsPage = () => {
    const ctx = React.useContext(Context);
    const [form] = Form.useForm();
    const [editing, setEditing] = React.useState<TokenAPIWalletBlacklist>(null);
    const data = useAsyncData(() => services.tokenapi.listWalletBlacklists(), []);
    const canUpdate = useCanI(rbacResources.tokenapi, rbacActions.update);
    const canModify = canUpdate.data === true;
    const add = async (values: {wallet: string; note?: string}) => {
        if (!canModify) {
            return;
        }
        await services.tokenapi.createWalletBlacklist(values.wallet, values.note || '');
        ctx.notifications.success('Wallet blacklisted');
        form.resetFields();
        data.reload();
    };
    const saveNote = async (values: {note?: string}) => {
        if (!canModify || !editing?.wallet) {
            return;
        }
        await services.tokenapi.updateWalletBlacklist(editing.wallet, values.note || '');
        setEditing(null);
        data.reload();
    };
    const remove = (item: TokenAPIWalletBlacklist) => {
        if (!canModify) {
            return;
        }
        ctx.modal.confirm({
            title: 'Delete wallet blacklist entry?',
            content: item.wallet,
            onOk: async () => {
                await services.tokenapi.deleteWalletBlacklist(item.wallet || '');
                data.reload();
            }
        });
    };
    const columns: ColumnsType<TokenAPIWalletBlacklist> = [
        {title: 'Wallet', render: item => <TruncatedText value={item.wallet} copyable={true} />},
        {title: 'Note', dataIndex: 'note'},
        {title: 'Created', dataIndex: 'createdAt'},
        {
            title: 'Actions',
            render: item => (
                <Space>
                    <Button icon={<EditOutlined />} disabled={!canModify || !item.wallet} onClick={() => setEditing(item)}>
                        Edit Note
                    </Button>
                    <Button danger={true} icon={<DeleteOutlined />} disabled={!canModify || !item.wallet} onClick={() => remove(item)}>
                        Delete
                    </Button>
                </Space>
            )
        }
    ];
    return (
        <AppPage
            title='Wallet Blacklists'
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}
            filters={
                <Form form={form} layout='inline' onFinish={add}>
                    <Form.Item name='wallet' rules={[{required: true}]}>
                        <Input placeholder='Wallet' />
                    </Form.Item>
                    <Form.Item name='note'>
                        <Input placeholder='Note' />
                    </Form.Item>
                    <Button type='primary' htmlType='submit' icon={<PlusOutlined />} disabled={!canModify}>
                        Add
                    </Button>
                </Form>
            }>
            <ResponsiveResourceList
                rowKey={item => item.wallet || Math.random()}
                items={data.data || []}
                columns={columns}
                loading={data.loading}
                card={item => (
                    <>
                        <CardTitle title={<TruncatedText value={item.wallet} copyable={true} />} subtitle={item.note} />
                        <MetricRow items={[{label: 'Created', value: item.createdAt}]} />
                        <InlineActions>
                            <Button size='small' icon={<EditOutlined />} disabled={!canModify || !item.wallet} onClick={() => setEditing(item)}>
                                Edit Note
                            </Button>
                            <Button size='small' danger={true} icon={<DeleteOutlined />} disabled={!canModify || !item.wallet} onClick={() => remove(item)}>
                                Delete
                            </Button>
                        </InlineActions>
                    </>
                )}
            />
            <Modal open={!!editing} title='Edit Wallet Note' footer={null} onCancel={() => setEditing(null)}>
                <Form key={editing?.wallet || 'wallet-note'} layout='vertical' initialValues={editing || {}} onFinish={saveNote}>
                    <Form.Item label='Wallet'>
                        <TruncatedText value={editing?.wallet} copyable={true} />
                    </Form.Item>
                    <Form.Item name='note' label='Note'>
                        <Input.TextArea rows={4} />
                    </Form.Item>
                    <Button type='primary' htmlType='submit' disabled={!canModify}>
                        Save
                    </Button>
                </Form>
            </Modal>
        </AppPage>
    );
};

export const ChainCheckpointsPage = () => {
    const data = useAsyncData(() => services.tokenapi.listChainIngestCheckpoints(), []);
    const canUpdate = useCanI(rbacResources.tokenapi, rbacActions.update);
    const canModify = canUpdate.data === true;
    const updateStatus = async (item: TokenAPIChainIngestCheckpoint, status: string) => {
        if (!canModify || item.chainID === undefined) {
            return;
        }
        await services.tokenapi.updateChainIngestCheckpoint(item.chainID, status);
        data.reload();
    };
    const columns: ColumnsType<TokenAPIChainIngestCheckpoint> = [
        {title: 'Chain', render: item => item.chainName || item.chainID},
        {title: 'Enabled', render: item => boolTag(item.enabled)},
        {title: 'Cursor', render: item => fmtNumber(item.cursorBlockNumber)},
        {title: 'Status', render: item => <StatusTag value={item.status} positive={item.status === 'running'} />},
        {title: 'Created', dataIndex: 'createdAt'},
        {
            title: 'Actions',
            render: item => (
                <Select
                    disabled={!canModify || item.chainID === undefined}
                    value={item.status}
                    style={{width: 130}}
                    onChange={value => void updateStatus(item, value)}
                    options={['running', 'stopped'].map(value => ({value, label: value}))}
                />
            )
        }
    ];
    return (
        <AppPage title='Chain Checkpoints' loading={data.loading} error={data.error} onRefresh={data.reload}>
            <ResponsiveResourceList
                rowKey={item => item.chainID ?? Math.random()}
                items={data.data || []}
                columns={columns}
                loading={data.loading}
                card={item => (
                    <>
                        <CardTitle title={item.chainName || `Chain ${item.chainID}`} subtitle={`Cursor ${fmtNumber(item.cursorBlockNumber)}`} tags={<Tag>{item.status}</Tag>} />
                        <MetricRow
                            items={[
                                {label: 'Enabled', value: fmt(item.enabled)},
                                {label: 'Created', value: item.createdAt}
                            ]}
                        />
                        <Select
                            disabled={!canModify || item.chainID === undefined}
                            value={item.status}
                            style={{width: '100%'}}
                            onChange={value => void updateStatus(item, value)}
                            options={['running', 'stopped'].map(value => ({value, label: value}))}
                        />
                    </>
                )}
            />
        </AppPage>
    );
};

export const CollectionTasksPage = () => {
    const {page, pageSize, setPage} = usePagedParams();
    const [projectID, setProjectID] = React.useState<number>();
    const [dataType, setDataType] = React.useState('');
    const [status, setStatus] = React.useState('');
    const data = useAsyncData(
        () =>
            services.tokenapi.listProjectDataCollectionTasks({
                page,
                pageSize,
                projectID,
                dataType: dataType || undefined,
                status: status || undefined
            }),
        [page, pageSize, projectID, dataType, status]
    );
    const columns: ColumnsType<TokenAPIProjectDataCollectionTask> = [
        {title: 'Project', dataIndex: 'projectID'},
        {title: 'Data Type', dataIndex: 'dataType'},
        {title: 'Status', dataIndex: 'status'},
        {title: 'Attempts', dataIndex: 'attempts'},
        {title: 'Next Attempt', dataIndex: 'nextAttemptAt'},
        {title: 'Last Error', render: item => <TruncatedText value={item.lastError} />},
        {title: 'Created', dataIndex: 'createdAt'}
    ];
    return (
        <AppPage
            title='Collection Tasks'
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}
            filters={
                <Space wrap={true}>
                    <InputNumber value={projectID} placeholder='Project ID' onChange={value => setProjectID(typeof value === 'number' ? value : undefined)} />
                    <Input value={dataType} placeholder='Data type' onChange={event => setDataType(event.target.value)} />
                    <Select
                        allowClear={true}
                        value={status || undefined}
                        placeholder='Status'
                        style={{width: 150}}
                        onChange={value => setStatus(value || '')}
                        options={['pending', 'succeeded', 'failed'].map(value => ({value, label: value}))}
                    />
                </Space>
            }>
            <ResponsiveResourceList
                rowKey={item => `${item.projectID}-${item.dataType}`}
                items={data.data?.items || []}
                columns={columns}
                loading={data.loading}
                total={data.data?.total}
                page={page}
                pageSize={pageSize}
                onPageChange={setPage}
                card={item => (
                    <>
                        <CardTitle title={`${item.projectID} / ${item.dataType}`} subtitle={item.lastError} tags={<Tag>{item.status}</Tag>} />
                        <MetricRow
                            items={[
                                {label: 'Attempts', value: item.attempts},
                                {label: 'Next', value: item.nextAttemptAt},
                                {label: 'Created', value: item.createdAt}
                            ]}
                        />
                    </>
                )}
            />
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
    const visibleAccounts = visibleAccountsForUser(accounts.data || [], user.data);
    return (
        <AppPage
            title='Settings'
            loading={user.loading || accounts.loading}
            error={user.error || accounts.error}
            onRefresh={() => {
                user.reload();
                accounts.reload();
            }}>
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
