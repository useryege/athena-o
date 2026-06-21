import {DeleteOutlined, EditOutlined, PlusOutlined} from '@ant-design/icons';
import {Button, Form, Input, Modal, Space} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {AppPage, ResourceTable, TruncatedText, useAsyncData} from '../components';
import {Context} from '../shared/context';
import {services} from '../shared/services';
import {TokenAPIWalletBlacklist} from '../shared/services/tokenapi-service';

export const WalletBlacklistsPage = () => {
    const ctx = React.useContext(Context);
    const [form] = Form.useForm();
    const [editing, setEditing] = React.useState<TokenAPIWalletBlacklist>(null);
    const data = useAsyncData(() => services.tokenapi.listWalletBlacklists(), []);
    const add = async (values: {wallet: string; note?: string}) => {
        await services.tokenapi.createWalletBlacklist(values.wallet, values.note || '');
        ctx.notifications.success('Wallet blacklisted');
        form.resetFields();
        data.reload();
    };
    const saveNote = async (values: {note?: string}) => {
        if (!editing?.wallet) {
            return;
        }
        await services.tokenapi.updateWalletBlacklist(editing.wallet, values.note || '');
        setEditing(null);
        data.reload();
    };
    const remove = (item: TokenAPIWalletBlacklist) => {
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
                    <Button icon={<EditOutlined />} disabled={!item.wallet} onClick={() => setEditing(item)}>
                        Edit Note
                    </Button>
                    <Button danger={true} icon={<DeleteOutlined />} disabled={!item.wallet} onClick={() => remove(item)}>
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
                    <Button type='primary' htmlType='submit' icon={<PlusOutlined />}>
                        Add
                    </Button>
                </Form>
            }>
            <ResourceTable rowKey={item => item.wallet || Math.random()} items={data.data || []} columns={columns} loading={data.loading} />
            <Modal open={!!editing} title='Edit Wallet Note' footer={null} onCancel={() => setEditing(null)}>
                <Form key={editing?.wallet || 'wallet-note'} layout='vertical' initialValues={editing || {}} onFinish={saveNote}>
                    <Form.Item label='Wallet'>
                        <TruncatedText value={editing?.wallet} copyable={true} />
                    </Form.Item>
                    <Form.Item name='note' label='Note'>
                        <Input.TextArea rows={4} />
                    </Form.Item>
                    <Button type='primary' htmlType='submit'>
                        Save
                    </Button>
                </Form>
            </Modal>
        </AppPage>
    );
};
