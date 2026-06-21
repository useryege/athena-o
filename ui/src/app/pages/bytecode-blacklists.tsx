import {DeleteOutlined, EditOutlined, PlusOutlined} from '@ant-design/icons';
import {Button, Form, Input, Modal, Select, Space} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {AppPage, ResourceTable, TruncatedText, useAsyncData} from '../components';
import {Context} from '../shared/context';
import {services} from '../shared/services';
import {TokenAPIBytecodeBlacklist} from '../shared/services/tokenapi-service';
import {ChainBadge} from './token-shared';

export const BytecodeBlacklistsPage = () => {
    const ctx = React.useContext(Context);
    const [form] = Form.useForm();
    const [editing, setEditing] = React.useState<TokenAPIBytecodeBlacklist>(null);
    const data = useAsyncData(() => services.tokenapi.listBytecodeBlacklists(), []);
    const options = useAsyncData(() => services.tokenapi.getOptions(), []);
    const chainOptions = React.useMemo(
        () =>
            (options.data?.chains || [])
                .filter(item => item.chainID !== undefined)
                .map(item => ({
                    value: item.chainID,
                    label: <ChainBadge chainID={item.chainID} />
                })),
        [options.data]
    );
    const refresh = React.useCallback(() => {
        options.reload();
        data.reload();
    }, [data, options]);
    const add = async (values: {note?: string; sourceChainID?: number; sourceContract?: string}) => {
        await services.tokenapi.createBytecodeBlacklist(values);
        ctx.notifications.success('Bytecode blacklisted');
        form.resetFields();
        data.reload();
    };
    const saveNote = async (values: {note?: string}) => {
        if (!editing?.codeHash) {
            return;
        }
        await services.tokenapi.updateBytecodeBlacklist(editing.codeHash, values.note || '');
        setEditing(null);
        data.reload();
    };
    const remove = (item: TokenAPIBytecodeBlacklist) => {
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
        {title: 'Source Chain', render: item => <ChainBadge chainID={item.sourceChainID} />},
        {title: 'Source Contract', render: item => <TruncatedText value={item.sourceContract} copyable={true} />},
        {title: 'Created', dataIndex: 'createdAt'},
        {
            title: 'Actions',
            render: item => (
                <Space>
                    <Button icon={<EditOutlined />} disabled={!item.codeHash} onClick={() => setEditing(item)}>
                        Edit Note
                    </Button>
                    <Button danger={true} icon={<DeleteOutlined />} disabled={!item.codeHash} onClick={() => remove(item)}>
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
                    <Button type='primary' htmlType='submit' icon={<PlusOutlined />}>
                        Add
                    </Button>
                </Form>
            }>
            <ResourceTable rowKey={item => item.codeHash || Math.random()} items={data.data || []} columns={columns} loading={data.loading} />
            <Modal open={!!editing} title='Edit Bytecode Note' footer={null} onCancel={() => setEditing(null)}>
                <Form key={editing?.codeHash || 'bytecode-note'} layout='vertical' initialValues={editing || {}} onFinish={saveNote}>
                    <Form.Item label='Code Hash'>
                        <TruncatedText value={editing?.codeHash} copyable={true} />
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
