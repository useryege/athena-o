import {CopyOutlined, EyeOutlined, PlusOutlined} from '@ant-design/icons';
import {Alert, Button, Checkbox, Form, Input, Modal, Select, Space, Tooltip} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {AppPage, KeyValueGrid, ResourceTable, SearchBar, TruncatedText, useAsyncData} from '../components';
import {Context} from '../shared/context';
import {services} from '../shared/services';
import {WalletDetail, WalletItem} from '../shared/services/wallet-service';
import {useKeywordParam, usePagedParams} from './shared';

const SecretInput = (props: {label: string; value?: string; onCopy: () => void}) => (
    <Space.Compact block={true}>
        <Input.Password aria-label={props.label} readOnly={true} value={props.value || ''} />
        <Tooltip title={`Copy ${props.label.toLowerCase()}`}>
            <Button aria-label={`Copy ${props.label.toLowerCase()}`} icon={<CopyOutlined />} onClick={props.onCopy} />
        </Tooltip>
    </Space.Compact>
);

export const WalletsPage = (props: {canCreate: boolean; canReveal: boolean}) => {
    const ctx = React.useContext(Context);
    const {page, pageSize, setPage} = usePagedParams();
    const [query, setQuery] = useKeywordParam();
    const [chain, setChain] = React.useState('');
    const [createOpen, setCreateOpen] = React.useState(false);
    const [creating, setCreating] = React.useState(false);
    const [secret, setSecret] = React.useState<WalletDetail>(null);
    const [backupWallet, setBackupWallet] = React.useState<WalletDetail>(null);
    const [backupConfirmed, setBackupConfirmed] = React.useState(false);
    const data = useAsyncData(() => services.wallet.listWallets({page, pageSize, query, chain: chain || undefined}), [page, pageSize, query, chain]);
    const reveal = async (id: number) => {
        setSecret(await services.wallet.getWallet(id, true));
    };
    const create = async (values: {chain: string; alias?: string}) => {
        setCreating(true);
        try {
            const created = await services.wallet.createWallet(values.chain, values.alias || '');
            setCreateOpen(false);
            setBackupConfirmed(false);
            setBackupWallet(created);
            data.reload();
        } catch (err: any) {
            ctx.notifications.error('Wallet creation failed', err?.message || 'Could not create this wallet.');
        } finally {
            setCreating(false);
        }
    };
    const copySecret = async (label: string, value?: string) => {
        try {
            await navigator.clipboard.writeText(value || '');
            ctx.notifications.success(`${label} copied`);
        } catch {
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
    const columns: ColumnsType<WalletItem> = [
        {title: 'Alias', dataIndex: 'alias'},
        {title: 'Chain', dataIndex: 'chain'},
        {title: 'Address', render: item => <TruncatedText value={item.address} copyable={true} />},
        {title: 'Created By', dataIndex: 'createdBy'},
        {title: 'Source', dataIndex: 'source'},
        {
            title: 'Actions',
            render: item =>
                props.canReveal ? (
                    <Button icon={<EyeOutlined />} onClick={() => reveal(item.id)}>
                        Reveal
                    </Button>
                ) : null
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
                props.canCreate ? (
                    <Button type='primary' icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
                        Create
                    </Button>
                ) : null
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
            <ResourceTable
                rowKey='id'
                items={data.data?.items || []}
                columns={columns}
                loading={data.loading}
                total={data.data?.total}
                page={page}
                pageSize={pageSize}
                onPageChange={setPage}
            />
            <Modal destroyOnHidden={true} open={createOpen} title='Create Wallet' footer={null} onCancel={() => setCreateOpen(false)}>
                <Form layout='vertical' onFinish={create}>
                    <Form.Item name='chain' label='Chain' rules={[{required: true}]}>
                        <Select options={['ETH', 'BSC', 'BASE', 'SOLANA'].map(value => ({value, label: value}))} />
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
                                value: <SecretInput label='Private key' value={backupWallet?.privateKey} onCopy={() => copySecret('Private key', backupWallet?.privateKey)} />
                            },
                            {
                                label: 'Mnemonic',
                                value: <SecretInput label='Mnemonic' value={backupWallet?.mnemonic} onCopy={() => copySecret('Mnemonic', backupWallet?.mnemonic)} />
                            }
                        ]}
                    />
                    <Checkbox checked={backupConfirmed} onChange={event => setBackupConfirmed(event.target.checked)}>
                        I have securely backed up the private key and mnemonic
                    </Checkbox>
                </Space>
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
