import {SendOutlined} from '@ant-design/icons';
import {Button, Form, Input, Modal, Space} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {useNavigate} from 'react-router-dom';
import {AppPage, ChoiceGroup, ResourceTable, SearchBar, useAsyncData} from '../components';
import {Context, useAuthorization} from '../shared/context';
import {AccountDataModule} from '../shared/access-modules';
import {formatBeijingDateTime} from '../shared/format';
import {services} from '../shared/services';
import {NotificationDelivery} from '../shared/services/notification-service';
import {useKeywordParam, usePagedParams} from './shared';

export const NotificationsPage = () => {
    const ctx = React.useContext(Context);
    const authorization = useAuthorization();
    const canWrite = authorization.canWrite(AccountDataModule.Notifications);
    const canWriteRef = React.useRef(canWrite);
    canWriteRef.current = canWrite;
    const navigate = useNavigate();
    const [form] = Form.useForm();
    const {page, pageSize, setPage} = usePagedParams();
    const [keyword, setKeyword] = useKeywordParam('keyword');
    const [status, setStatus] = React.useState('');
    const [telegramChat, setTelegramChat] = React.useState('');
    const [testOpen, setTestOpen] = React.useState(false);
    const [testSubmitting, setTestSubmitting] = React.useState(false);
    const data = useAsyncData(
        () => services.notification.listNotifications({page, pageSize, keyword, status: status || undefined, telegramChat: telegramChat || undefined}),
        [page, pageSize, keyword, status, telegramChat]
    );
    React.useEffect(() => {
        if (!canWrite) {
            setTestOpen(false);
            setTestSubmitting(false);
            form.resetFields();
        }
    }, [canWrite, form]);
    const sendTest = async (values: {topicLabel: string}) => {
        if (!canWrite) {
            return;
        }
        setTestSubmitting(true);
        try {
            await services.notification.sendTestNotification(values.topicLabel.trim());
            setTestOpen(false);
            form.resetFields();
            ctx.notifications.success('Test notification queued');
            data.reload();
        } catch (err: any) {
            if (canWriteRef.current) {
                ctx.notifications.error('Test notification failed', err?.message || 'Could not send the test notification');
            }
        } finally {
            setTestSubmitting(false);
        }
    };
    const closeTest = () => {
        if (!testSubmitting) {
            setTestOpen(false);
            form.resetFields();
        }
    };
    const columns: ColumnsType<NotificationDelivery> = [
        {
            title: 'Title',
            render: item => (
                <Button type='link' onClick={() => navigate(`/notifications/${item.id}`)}>
                    {item.title || item.topicLabel}
                </Button>
            )
        },
        {title: 'Severity', dataIndex: 'severity'},
        {title: 'Topic', dataIndex: 'topicLabel'},
        {title: 'Telegram Chat', dataIndex: 'telegramChat'},
        {title: 'Status', dataIndex: 'status'},
        {title: 'Channel', dataIndex: 'channel'},
        {title: 'Created', render: item => formatBeijingDateTime(item.createdAt) || '-'}
    ];
    return (
        <AppPage
            title='Notifications'
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}
            extra={
                canWrite ? (
                    <Button icon={<SendOutlined />} onClick={() => setTestOpen(true)}>
                        Test Notification
                    </Button>
                ) : null
            }
            filters={
                <Space wrap={true}>
                    <SearchBar value={keyword} onChange={setKeyword} placeholder='Keyword' />
                    <ChoiceGroup<string>
                        ariaLabel='Filter by notification status'
                        value={status || 'all'}
                        options={[{label: 'All', value: 'all'}, ...['pending', 'sent', 'failed'].map(value => ({value, label: value}))]}
                        onChange={value => {
                            setStatus(value === 'all' ? '' : value);
                        }}
                    />
                    <ChoiceGroup<string>
                        ariaLabel='Filter by Telegram chat'
                        value={telegramChat || 'all'}
                        options={[{label: 'All', value: 'all'}, ...['test', 'prod'].map(value => ({value, label: value}))]}
                        onChange={value => {
                            setTelegramChat(value === 'all' ? '' : value);
                        }}
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
            <Modal open={canWrite && testOpen} title='Test Notification' footer={null} closable={!testSubmitting} onCancel={closeTest}>
                <Form form={form} layout='vertical' onFinish={sendTest}>
                    <Form.Item
                        name='topicLabel'
                        label='Topic Label'
                        rules={[
                            {
                                validator: (_, value) => (typeof value === 'string' && value.trim() ? Promise.resolve() : Promise.reject(new Error('Topic Label is required')))
                            }
                        ]}>
                        <Input autoFocus={true} disabled={testSubmitting} />
                    </Form.Item>
                    <Button type='primary' htmlType='submit' icon={<SendOutlined />} loading={testSubmitting}>
                        Send Test Notification
                    </Button>
                </Form>
            </Modal>
        </AppPage>
    );
};
