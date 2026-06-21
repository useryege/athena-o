import {EyeOutlined, PlusOutlined} from '@ant-design/icons';
import {Button, Form, Input, Modal, Select, Space} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {AppPage, KeyValueGrid, ResourceTable, SearchBar, TruncatedText, useAsyncData} from '../components';
import {Context} from '../shared/context';
import {services} from '../shared/services';
import {WalletDetail, WalletItem} from '../shared/services/wallet-service';
import {useKeywordParam, usePagedParams} from './shared';

export const WalletsPage = () => {
    const ctx = React.useContext(Context);
    const {page, pageSize, setPage} = usePagedParams();
    const [query, setQuery] = useKeywordParam();
    const [chain, setChain] = React.useState('');
    const [createOpen, setCreateOpen] = React.useState(false);
    const [secret, setSecret] = React.useState<WalletDetail>(null);
    const data = useAsyncData(() => services.wallet.listWallets({page, pageSize, query, chain: chain || undefined}), [page, pageSize, query, chain]);
    const reveal = async (id: number) => {
        setSecret(await services.wallet.getWallet(id, true));
    };
    const create = async (values: {chain: string; alias?: string}) => {
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
                <Button icon={<EyeOutlined />} onClick={() => reveal(item.id)}>
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
                <Button type='primary' icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
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
            <Modal open={createOpen} title='Create Wallet' footer={null} onCancel={() => setCreateOpen(false)}>
                <Form layout='vertical' onFinish={create}>
                    <Form.Item name='chain' label='Chain' rules={[{required: true}]}>
                        <Select options={['ETH', 'BSC', 'BASE', 'SOLANA'].map(value => ({value, label: value}))} />
                    </Form.Item>
                    <Form.Item name='alias' label='Alias'>
                        <Input />
                    </Form.Item>
                    <Button type='primary' htmlType='submit'>
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
