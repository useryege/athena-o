import {DeleteOutlined, EditOutlined, PlusOutlined} from '@ant-design/icons';
import {Button, Form, Input, Modal, Space} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {AppPage, ChoiceGroup, ResourceTable, TruncatedText, useAsyncData} from '../../components';
import {Context, ModalHandle, useAuthorization} from '../../shared/context';
import {AccountDataModule} from '../../shared/access-modules';
import {formatBeijingDateTime} from '../../shared/format';
import {memberServices as services} from '../services';
import {TokenContractCodeBlocklistEntry} from '../../shared/services/token-service';
import {ChainBadge, chainLabel} from './token-shared';

export const ContractCodeBlocklistPage = () => {
    const ctx = React.useContext(Context);
    const authorization = useAuthorization();
    const canWrite = authorization.canWrite(AccountDataModule.Token);
    const [form] = Form.useForm();
    const [editing, setEditing] = React.useState<TokenContractCodeBlocklistEntry>(null);
    const deleteConfirmRef = React.useRef<ModalHandle>();
    const data = useAsyncData(() => services.tokenapi.listContractCodeBlocklistEntries(), []);
    const options = useAsyncData(() => services.tokenapi.getRuntimeConfiguration(), []);
    const chainOptions = React.useMemo(
        () =>
            (options.data?.chains || [])
                .filter((item): item is {chainID: number; chainName?: string} => item.chainID !== undefined)
                .map(item => ({
                    value: item.chainID,
                    label: chainLabel(item.chainID)
                })),
        [options.data]
    );
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
    const refresh = React.useCallback(() => {
        options.reload();
        data.reload();
    }, [data, options]);
    const add = async (values: {note?: string; sourceChainID?: number; sourceContract?: string}) => {
        if (!canWrite) {
            return;
        }
        await services.tokenapi.createContractCodeBlocklistEntry(values);
        ctx.notifications.success('Contract code blocklist entry created');
        form.resetFields();
        data.reload();
    };
    const saveNote = async (values: {note?: string}) => {
        if (!canWrite || !editing?.codeHash) {
            return;
        }
        await services.tokenapi.updateContractCodeBlocklistEntry(editing.codeHash, values.note || '');
        setEditing(null);
        data.reload();
    };
    const remove = (item: TokenContractCodeBlocklistEntry) => {
        if (!canWrite) {
            return;
        }
        deleteConfirmRef.current = ctx.modal.confirm({
            title: 'Delete contract code blocklist entry?',
            content: item.codeHash,
            onOk: async () => {
                try {
                    await services.tokenapi.deleteContractCodeBlocklistEntry(item.codeHash || '');
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
    const columns: ColumnsType<TokenContractCodeBlocklistEntry> = [
        {title: 'Code Hash', render: item => <TruncatedText value={item.codeHash} copyable={true} />},
        {title: 'Note', dataIndex: 'note'},
        {title: 'Source Chain', render: item => <ChainBadge chainID={item.sourceChainID} />},
        {title: 'Source Contract', render: item => <TruncatedText value={item.sourceContract} copyable={true} />},
        {title: 'Created', render: item => formatBeijingDateTime(item.createdAt) || '-'}
    ];
    if (canWrite) {
        columns.push({
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
        });
    }
    return (
        <AppPage
            title='Contract Code Blocklist'
            loading={data.loading || options.loading}
            error={data.error || options.error}
            onRefresh={refresh}
            filters={
                canWrite ? (
                    <Form form={form} layout='inline' onFinish={add}>
                        <Form.Item name='sourceChainID' label='Source chain' rules={[{required: true}]}>
                            <ChoiceGroup<number> ariaLabel='Source chain' className='choice-group--form' options={chainOptions} />
                        </Form.Item>
                        <Form.Item name='sourceContract' rules={[{required: true}]}>
                            <Input aria-label='Source contract' placeholder='Source contract' />
                        </Form.Item>
                        <Form.Item name='note'>
                            <Input aria-label='Contract code note' placeholder='Note' />
                        </Form.Item>
                        <Button type='primary' htmlType='submit' icon={<PlusOutlined />}>
                            Add
                        </Button>
                    </Form>
                ) : undefined
            }>
            <ResourceTable rowKey={item => item.codeHash || Math.random()} items={data.data || []} columns={columns} loading={data.loading} />
            <Modal open={canWrite && !!editing} title='Edit Contract Code Note' footer={null} onCancel={() => setEditing(null)}>
                <Form key={editing?.codeHash || 'contract-code-note'} layout='vertical' initialValues={editing || {}} onFinish={saveNote}>
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
