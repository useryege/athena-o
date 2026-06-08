import {EyeOutlined, PlusOutlined} from '@ant-design/icons';
import {Button, Form, Input, Modal, Select, Space, Tag} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {AppPage, CardTitle, InlineActions, KeyValueGrid, MetricRow, ResponsiveResourceList, SearchBar, TruncatedText, useAsyncData} from '../components';
import {Context} from '../../shared/context';
import {services} from '../../shared/services';
import {WalletDetail, WalletItem} from '../../shared/services/wallet-service';
import {rbacActions, rbacResources, useCanI, useKeywordParam, usePagedParams} from './shared';

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
