import {SendOutlined} from '@ant-design/icons';
import {Button, Form, Input, Modal, Select, Space} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {useNavigate} from 'react-router-dom';
import {AppPage, ResourceTable, SearchBar, useAsyncData} from '../components';
import {Context} from '../shared/context';
import {services} from '../shared/services';
import {NotificationDelivery} from '../shared/services/notification-service';
import {useKeywordParam, usePagedParams} from './shared';

export const NotificationsPage = () => {
    const ctx = React.useContext(Context);
    const navigate = useNavigate();
    const [form] = Form.useForm();
    const {page, pageSize, setPage} = usePagedParams();
    const [keyword, setKeyword] = useKeywordParam('keyword');
    const [status, setStatus] = React.useState('');
    const [testOpen, setTestOpen] = React.useState(false);
    const [testSubmitting, setTestSubmitting] = React.useState(false);
    const data = useAsyncData(() => services.notification.listNotifications({page, pageSize, keyword, status: status || undefined}), [page, pageSize, keyword, status]);
    const sendTest = async (values: {topicLabel: string}) => {
        setTestSubmitting(true);
        try {
            await services.notification.sendTestNotification(values.topicLabel.trim());
            setTestOpen(false);
            form.resetFields();
            ctx.notifications.success('Test notification queued');
            data.reload();
        } catch (err: any) {
            ctx.notifications.error('Test notification failed', err?.message || 'Could not send the test notification');
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
            extra={
                <Button icon={<SendOutlined />} onClick={() => setTestOpen(true)}>
                    Test Notification
                </Button>
            }
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
            <Modal open={testOpen} title='Test Notification' footer={null} closable={!testSubmitting} onCancel={closeTest}>
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
