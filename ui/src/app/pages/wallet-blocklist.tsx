import {DeleteOutlined, EditOutlined, PlusOutlined} from '@ant-design/icons';
import {Button, Form, Input, Modal, Space} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {AppPage, ResourceTable, TruncatedText, useAsyncData} from '../components';
import {Context, ModalHandle, useAuthorization} from '../shared/context';
import {AccountDataModule} from '../shared/access-modules';
import {formatBeijingDateTime} from '../shared/format';
import {services} from '../shared/services';
import {TokenWalletBlocklistEntry} from '../shared/services/token-service';

export const WalletBlocklistPage = () => {
    const ctx = React.useContext(Context);
    const authorization = useAuthorization();
    const canWrite = authorization.canWrite(AccountDataModule.Token);
    const [form] = Form.useForm();
    const [editing, setEditing] = React.useState<TokenWalletBlocklistEntry>(null);
    const deleteConfirmRef = React.useRef<ModalHandle>();
    const data = useAsyncData(() => services.tokenapi.listWalletBlocklistEntries(), []);
    React.useEffect(() => {
        if (!canWrite) {
            setEditing(null);
            form.resetFields();
            deleteConfirmRef.current?.destroy();
            deleteConfirmRef.current = undefined;
        }
    }, [canWrite, form]);
    React.useEffect(
        () => () => {
            deleteConfirmRef.current?.destroy();
        },
        []
    );
    const add = async (values: {wallet: string; note?: string}) => {
        if (!canWrite) {
            return;
        }
        await services.tokenapi.createWalletBlocklistEntry(values.wallet, values.note || '');
        ctx.notifications.success('Wallet blocklist entry created');
        form.resetFields();
        data.reload();
    };
    const saveNote = async (values: {note?: string}) => {
        if (!canWrite || !editing?.wallet) {
            return;
        }
        await services.tokenapi.updateWalletBlocklistEntry(editing.wallet, values.note || '');
        setEditing(null);
        data.reload();
    };
    const remove = (item: TokenWalletBlocklistEntry) => {
        if (!canWrite) {
            return;
        }
        deleteConfirmRef.current = ctx.modal.confirm({
            title: 'Delete wallet blocklist entry?',
            content: item.wallet,
            onOk: async () => {
                try {
                    await services.tokenapi.deleteWalletBlocklistEntry(item.wallet || '');
                    data.reload();
                } finally {
                    deleteConfirmRef.current = undefined;
                }
            },
            onCancel: () => {
                deleteConfirmRef.current = undefined;
            }
        });
    };
    const columns: ColumnsType<TokenWalletBlocklistEntry> = [
        {title: 'Wallet', render: item => <TruncatedText value={item.wallet} copyable={true} />},
        {title: 'Note', dataIndex: 'note'},
        {title: 'Created', render: item => formatBeijingDateTime(item.createdAt) || '-'}
    ];
    if (canWrite) {
        columns.push({
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
        });
    }
    return (
        <AppPage
            title='Wallet Blocklist'
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}
            filters={
                canWrite ? (
                    <Form form={form} layout='inline' onFinish={add}>
                        <Form.Item name='wallet' rules={[{required: true}]}>
                            <Input aria-label='Wallet address' placeholder='Wallet' />
                        </Form.Item>
                        <Form.Item name='note'>
                            <Input aria-label='Wallet note' placeholder='Note' />
                        </Form.Item>
                        <Button type='primary' htmlType='submit' icon={<PlusOutlined />}>
                            Add
                        </Button>
                    </Form>
                ) : undefined
            }>
            <ResourceTable rowKey={item => item.wallet || Math.random()} items={data.data || []} columns={columns} loading={data.loading} />
            <Modal open={canWrite && !!editing} title='Edit Wallet Note' footer={null} onCancel={() => setEditing(null)}>
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
