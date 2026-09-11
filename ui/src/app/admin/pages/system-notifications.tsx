import {SendOutlined} from '@ant-design/icons';
import {Button, Form, Input, Modal, Space, Tag} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {useNavigate} from 'react-router-dom';
import {AppPage, ChoiceGroup, ResourceTable, SearchBar, useAsyncData} from '../../components';
import {Context} from '../../shared/context';
import {formatBeijingDateTime} from '../../shared/format';
import {useKeywordParam, usePagedParams} from '../../shared/pages/shared';
import {requestErrorMessage} from '../../shared/services/requests';
import type {NotificationDelivery} from '../notification-service';
import {adminServices as services} from '../services';

const statusColor = (value: string) => {
    switch (value.toLowerCase()) {
        case 'sent':
            return 'green';
        case 'failed':
            return 'red';
        case 'unknown':
            return 'orange';
        case 'sending':
            return 'cyan';
        case 'cancelled':
            return 'default';
        case 'pending':
            return 'blue';
        default:
            return 'default';
    }
};

const severityColor = (value: string) => {
    switch (value.toLowerCase()) {
        case 'critical':
        case 'error':
            return 'red';
        case 'warning':
            return 'orange';
        case 'info':
            return 'blue';
        default:
            return 'default';
    }
};

export const SystemNotificationsPage = () => {
    const ctx = React.useContext(Context);
    const navigate = useNavigate();
    const [form] = Form.useForm();
    const {page, pageSize, setPage} = usePagedParams();
    const [keyword, setKeyword] = useKeywordParam('keyword');
    const [status, setStatus] = React.useState('');
    const [telegramChat, setTelegramChat] = React.useState('');
    const [testOpen, setTestOpen] = React.useState(false);
    const [testSubmitting, setTestSubmitting] = React.useState(false);
    const testRequestRef = React.useRef<ReturnType<typeof services.adminNotifications.sendTestNotification>>();
    const mountedRef = React.useRef(true);
    const data = useAsyncData(
        () => services.adminNotifications.listNotifications({page, pageSize, keyword, status: status || undefined, telegramChat: telegramChat || undefined}),
        [page, pageSize, keyword, status, telegramChat]
    );

    React.useEffect(() => {
        mountedRef.current = true;
        return () => {
            mountedRef.current = false;
            testRequestRef.current?.abort?.();
        };
    }, []);

    const sendTest = async (values: {topicLabel: string}) => {
        if (testSubmitting || testRequestRef.current) {
            return;
        }
        setTestSubmitting(true);
        const request = services.adminNotifications.sendTestNotification(values.topicLabel.trim());
        testRequestRef.current = request;
        try {
            await request;
            if (!mountedRef.current) {
                return;
            }
            setTestOpen(false);
            form.resetFields();
            ctx.notifications.success('Test notification queued');
            data.reload();
        } catch (error) {
            if (mountedRef.current) {
                ctx.notifications.error('Test notification failed', requestErrorMessage(error, 'Could not send the test notification'));
            }
        } finally {
            if (testRequestRef.current === request) {
                testRequestRef.current = undefined;
            }
            if (mountedRef.current) {
                setTestSubmitting(false);
            }
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
        {title: 'Severity', dataIndex: 'severity', render: value => <Tag color={severityColor(String(value || ''))}>{value || '-'}</Tag>},
        {title: 'Topic', dataIndex: 'topicLabel'},
        {title: 'Telegram Chat', dataIndex: 'telegramChat'},
        {title: 'Status', dataIndex: 'status', render: value => <Tag color={statusColor(String(value || ''))}>{value || '-'}</Tag>},
        {title: 'Channel', dataIndex: 'channel'},
        {title: 'Created', render: item => formatBeijingDateTime(item.createdAt) || '-'}
    ];

    const compactNotification = (item: NotificationDelivery) => (
        <article className='system-notification-card'>
            <div className='system-notification-card__heading'>
                <Button type='link' onClick={() => navigate(`/notifications/${item.id}`)}>
                    {item.title || item.topicLabel || `Notification ${item.id}`}
                </Button>
                <Tag color={statusColor(item.status)}>{item.status || '-'}</Tag>
            </div>
            <dl>
                <div>
                    <dt>Severity</dt>
                    <dd>
                        <Tag color={severityColor(item.severity)}>{item.severity || '-'}</Tag>
                    </dd>
                </div>
                <div>
                    <dt>Topic</dt>
                    <dd>{item.topicLabel || '-'}</dd>
                </div>
                <div>
                    <dt>Telegram chat</dt>
                    <dd>{item.telegramChat || '-'}</dd>
                </div>
                <div>
                    <dt>Created</dt>
                    <dd>{formatBeijingDateTime(item.createdAt) || '-'}</dd>
                </div>
            </dl>
        </article>
    );

    return (
        <AppPage
            title='System Notifications'
            subtitle='Inspect Telegram delivery records and send an operational connectivity test.'
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
                    <ChoiceGroup<string>
                        ariaLabel='Filter by notification status'
                        value={status || 'all'}
                        options={[{label: 'All', value: 'all'}, ...['pending', 'sending', 'sent', 'failed', 'unknown', 'cancelled'].map(value => ({value, label: value}))]}
                        onChange={value => {
                            setStatus(value === 'all' ? '' : value);
                            setPage(1, pageSize);
                        }}
                    />
                    <ChoiceGroup<string>
                        ariaLabel='Filter by Telegram chat'
                        value={telegramChat || 'all'}
                        options={[{label: 'All', value: 'all'}, ...['test', 'prod'].map(value => ({value, label: value}))]}
                        onChange={value => {
                            setTelegramChat(value === 'all' ? '' : value);
                            setPage(1, pageSize);
                        }}
                    />
                </Space>
            }>
            <ResourceTable
                rowKey='id'
                label='System notification deliveries'
                items={data.data?.items || []}
                columns={columns}
                compactRender={compactNotification}
                compactEmptyDescription='No system notification deliveries'
                loading={data.loading}
                total={data.data?.total}
                page={page}
                pageSize={pageSize}
                onPageChange={setPage}
                scrollX={1_080}
                stickyHeader={true}
            />
            <Modal open={testOpen} title='Test Notification' footer={null} closable={!testSubmitting} maskClosable={!testSubmitting} onCancel={closeTest}>
                <Form form={form} layout='vertical' onFinish={sendTest}>
                    <Form.Item
                        name='topicLabel'
                        label='Topic Label'
                        rules={[
                            {
                                validator: (_, value) => (typeof value === 'string' && value.trim() ? Promise.resolve() : Promise.reject(new Error('Topic Label is required')))
                            }
                        ]}>
                        <Input autoFocus={true} autoComplete='off' disabled={testSubmitting} />
                    </Form.Item>
                    <Space className='system-notification-test-actions' wrap={true}>
                        <Button disabled={testSubmitting} onClick={closeTest}>
                            Cancel
                        </Button>
                        <Button type='primary' htmlType='submit' icon={<SendOutlined />} loading={testSubmitting}>
                            Send Test Notification
                        </Button>
                    </Space>
                </Form>
            </Modal>
        </AppPage>
    );
};
