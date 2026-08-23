import {CopyOutlined, EyeOutlined, PlusOutlined} from '@ant-design/icons';
import {Alert, Button, Checkbox, Form, Input, Modal, Space, Tooltip} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {AppPage, AsyncState, ChoiceGroup, KeyValueGrid, ResourceTable, SearchBar, TruncatedText, useAsyncData} from '../components';
import {AccountDataModule} from '../shared/access-modules';
import {Context, useAuthorization} from '../shared/context';
import {SensitiveWriteScope, useSensitiveWriteLease} from '../shared/sensitive-write-scope';
import {services} from '../shared/services';
import {ListWalletsResult, WalletDetail, WalletItem, walletTypeLabel, walletTypeOptions} from '../shared/services/wallet-service';
import {useKeywordParam, usePagedParams} from './shared';

const walletChains = ['ETH', 'BSC', 'BASE', 'SOLANA'];
const walletChainOptions = walletChains.map(value => ({value, label: value}));

const errorMessage = (error: unknown, fallback: string) => {
    if (error && typeof error === 'object' && 'message' in error && typeof error.message === 'string') {
        return error.message;
    }
    return fallback;
};

const SecretInput = (props: {label: string; value?: string; onCopy: () => void}) => (
    <Space.Compact block={true}>
        <Input.Password aria-label={props.label} readOnly={true} value={props.value || ''} />
        <Tooltip title={`Copy ${props.label.toLowerCase()}`}>
            <Button aria-label={`Copy ${props.label.toLowerCase()}`} icon={<CopyOutlined />} onClick={props.onCopy} />
        </Tooltip>
    </Space.Compact>
);

const SecretText = (props: {label: string; value?: string; onCopy: () => void}) => (
    <Space size='small'>
        <TruncatedText value={props.value} />
        <Tooltip title={`Copy ${props.label.toLowerCase()}`}>
            <Button type='text' size='small' aria-label={`Copy ${props.label.toLowerCase()}`} icon={<CopyOutlined />} onClick={props.onCopy} />
        </Tooltip>
    </Space>
);

interface WalletWriteHandle {
    openCreate(): void;
    reveal(id: number): void;
}

const WalletWriteSurface = React.forwardRef<WalletWriteHandle, {onCreated: () => void}>(function WalletWriteSurface(props, ref) {
    const ctx = React.useContext(Context);
    const lease = useSensitiveWriteLease();
    const [createOpen, setCreateOpen] = React.useState(false);
    const [creating, setCreating] = React.useState(false);
    const [secret, setSecret] = React.useState<WalletDetail>(null);
    const [backupWallet, setBackupWallet] = React.useState<WalletDetail>(null);
    const [backupConfirmed, setBackupConfirmed] = React.useState(false);

    const reveal = React.useCallback(
        async (id: number) => {
            const result = await lease.runTask(() => services.wallet.getWallet(id, true));
            if (result.status === 'fulfilled') {
                setSecret(result.value);
            } else if (result.status === 'rejected') {
                ctx.notifications.error('Could not reveal wallet secret', errorMessage(result.error, 'The wallet secret request failed.'));
            }
        },
        [ctx.notifications, lease]
    );

    React.useImperativeHandle(
        ref,
        () => ({
            openCreate: () => setCreateOpen(true),
            reveal: id => {
                void reveal(id);
            }
        }),
        [reveal]
    );

    const create = async (values: {chain: string; type: string; alias?: string}) => {
        setCreating(true);
        const result = await lease.runTask(() => services.wallet.createWallet(values.chain, values.type, values.alias || ''));
        if (result.status === 'discarded') {
            return;
        }
        setCreating(false);
        if (result.status === 'rejected') {
            ctx.notifications.error('Wallet creation failed', errorMessage(result.error, 'Could not create this wallet.'));
            return;
        }
        setCreateOpen(false);
        setBackupConfirmed(false);
        setBackupWallet(result.value);
        props.onCreated();
    };

    const copySecret = async (label: string, value?: string) => {
        const result = await lease.runTask(() => navigator.clipboard.writeText(value || ''));
        if (result.status === 'fulfilled') {
            ctx.notifications.success(`${label} copied`);
        } else if (result.status === 'rejected') {
            ctx.notifications.error(`Could not copy ${label.toLowerCase()}`);
        }
    };

    const completeBackup = () => {
        if (!backupConfirmed) {
            return;
        }
        setBackupWallet(null);
        setBackupConfirmed(false);
    };

    return (
        <>
            <Modal destroyOnHidden={true} open={createOpen} title='Create Wallet' footer={null} onCancel={() => setCreateOpen(false)}>
                <Form layout='vertical' onFinish={create}>
                    <Form.Item name='chain' label='Chain' rules={[{required: true}]}>
                        <ChoiceGroup<string> ariaLabel='Wallet chain' className='choice-group--form' options={walletChainOptions} />
                    </Form.Item>
                    <Form.Item name='type' label='Type' rules={[{required: true}]}>
                        <ChoiceGroup<string> ariaLabel='Wallet type' className='choice-group--form' options={walletTypeOptions} />
                    </Form.Item>
                    <Form.Item name='alias' label='Alias'>
                        <Input />
                    </Form.Item>
                    <Button type='primary' htmlType='submit' loading={creating}>
                        Create
                    </Button>
                </Form>
            </Modal>
            <Modal
                destroyOnHidden={true}
                open={!!backupWallet}
                title='Back Up Wallet'
                width={680}
                closable={false}
                keyboard={false}
                maskClosable={false}
                footer={
                    <Button type='primary' disabled={!backupConfirmed} onClick={completeBackup}>
                        Done
                    </Button>
                }>
                <Space direction='vertical' size='middle' style={{width: '100%'}}>
                    <Alert
                        showIcon={true}
                        type='warning'
                        message='Back up this wallet now'
                        description='Store the private key and mnemonic securely. Never share them with anyone.'
                    />
                    <KeyValueGrid
                        columns={1}
                        items={[
                            {label: 'Address', value: <TruncatedText value={backupWallet?.address} copyable={true} />},
                            {
                                label: 'Private Key',
                                value: <SecretInput label='Private key' value={backupWallet?.privateKey} onCopy={() => void copySecret('Private key', backupWallet?.privateKey)} />
                            },
                            {
                                label: 'Mnemonic',
                                value: <SecretInput label='Mnemonic' value={backupWallet?.mnemonic} onCopy={() => void copySecret('Mnemonic', backupWallet?.mnemonic)} />
                            }
                        ]}
                    />
                    <Checkbox checked={backupConfirmed} onChange={event => setBackupConfirmed(event.target.checked)}>
                        I have securely backed up the private key and mnemonic
                    </Checkbox>
                </Space>
            </Modal>
            <Modal
                destroyOnHidden={true}
                open={!!secret}
                title='Wallet Secret'
                width={680}
                onCancel={() => setSecret(null)}
                footer={<Button onClick={() => setSecret(null)}>Close</Button>}>
                <KeyValueGrid
                    columns={1}
                    items={[
                        {label: 'Address', value: <TruncatedText value={secret?.address} copyable={true} />},
                        {
                            label: 'Private Key',
                            value: <SecretText label='Private key' value={secret?.privateKey} onCopy={() => void copySecret('Private key', secret?.privateKey)} />
                        },
                        {
                            label: 'Mnemonic',
                            value: <SecretText label='Mnemonic' value={secret?.mnemonic} onCopy={() => void copySecret('Mnemonic', secret?.mnemonic)} />
                        }
                    ]}
                />
            </Modal>
        </>
    );
});

interface WalletsViewProps {
    data: AsyncState<ListWalletsResult>;
    page: number;
    pageSize: number;
    setPage: (page: number, pageSize: number) => void;
    query: string;
    setQuery: (query: string) => void;
    chain: string;
    setChain: (chain: string) => void;
    walletType: string;
    setWalletType: (walletType: string) => void;
    canWrite: boolean;
    onOpenCreate: () => void;
    onReveal: (id: number) => void;
    writeSurface: React.ReactNode;
}

const WalletsView = (props: WalletsViewProps) => {
    const columns: ColumnsType<WalletItem> = [
        {title: 'Alias', dataIndex: 'alias'},
        {title: 'Type', render: item => walletTypeLabel(item.type)},
        {title: 'Chain', dataIndex: 'chain'},
        {title: 'Address', render: item => <TruncatedText value={item.address} copyable={true} />},
        {title: 'Created By', dataIndex: 'createdBy'},
        {title: 'Source', dataIndex: 'source'}
    ];
    if (props.canWrite) {
        columns.push({
            title: 'Actions',
            render: item => (
                <Button icon={<EyeOutlined />} onClick={() => props.onReveal(item.id)}>
                    Reveal
                </Button>
            )
        });
    }

    return (
        <AppPage
            title='Wallets'
            subtitle='Private key inventory and operational wallet lookup'
            loading={props.data.loading}
            error={props.data.error}
            onRefresh={props.data.reload}
            extra={
                props.canWrite ? (
                    <Button type='primary' icon={<PlusOutlined />} onClick={props.onOpenCreate}>
                        Create
                    </Button>
                ) : null
            }
            filters={
                <Space wrap={true}>
                    <SearchBar value={props.query} onChange={props.setQuery} placeholder='Address or alias' />
                    <ChoiceGroup<string>
                        ariaLabel='Filter by chain'
                        value={props.chain || 'all'}
                        options={[{label: 'All', value: 'all'}, ...walletChainOptions]}
                        onChange={value => props.setChain(value === 'all' ? '' : value)}
                    />
                    <ChoiceGroup<string>
                        ariaLabel='Filter by type'
                        value={props.walletType || 'all'}
                        options={[{label: 'All', value: 'all'}, ...walletTypeOptions]}
                        onChange={value => props.setWalletType(value === 'all' ? '' : value)}
                    />
                </Space>
            }>
            <ResourceTable
                rowKey='id'
                items={props.data.data?.items || []}
                columns={columns}
                loading={props.data.loading}
                total={props.data.data?.total}
                page={props.page}
                pageSize={props.pageSize}
                onPageChange={props.setPage}
            />
            {props.writeSurface}
        </AppPage>
    );
};

export const WalletsPage = () => {
    const authorization = useAuthorization();
    const canWrite = authorization.canWrite(AccountDataModule.Wallet);
    const writeHandle = React.useRef<WalletWriteHandle>(null);
    const {page, pageSize, setPage} = usePagedParams();
    const [query, setQuery] = useKeywordParam();
    const [chain, setChain] = React.useState('');
    const [walletType, setWalletType] = React.useState('');
    const data = useAsyncData(
        () => services.wallet.listWallets({page, pageSize, query, chain: chain || undefined, type: walletType || undefined}),
        [page, pageSize, query, chain, walletType]
    );

    return (
        <WalletsView
            data={data}
            page={page}
            pageSize={pageSize}
            setPage={setPage}
            query={query}
            setQuery={setQuery}
            chain={chain}
            setChain={setChain}
            walletType={walletType}
            setWalletType={setWalletType}
            canWrite={canWrite}
            onOpenCreate={() => writeHandle.current?.openCreate()}
            onReveal={id => writeHandle.current?.reveal(id)}
            writeSurface={
                <SensitiveWriteScope module={AccountDataModule.Wallet}>
                    <WalletWriteSurface ref={writeHandle} onCreated={data.reload} />
                </SensitiveWriteScope>
            }
        />
    );
};
