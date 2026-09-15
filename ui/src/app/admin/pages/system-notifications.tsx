import {SendOutlined} from '@ant-design/icons';
import {Button, Form, Input, Modal, Select, Space, Tag} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {useNavigate, useLocation} from 'react-router-dom';
import {AppPage, ResourceTable, SearchBar, useAsyncData} from '../../components';
import {useAdminReadScope} from '../read-scope';
import {Context} from '../../shared/context';
import {formatBeijingDateTime} from '../../shared/format';
import {useKeywordParam, usePagedParams} from '../../shared/pages/shared';
import {requestErrorMessage} from '../../shared/services/requests';
import type {NotificationDelivery} from '../notification-service';
import {adminServices as services} from '../services';

export const statusColor = (value: string) => {
    switch (value.toLowerCase()) {
        case 'sent':
            return 'success';
        case 'failed':
            return 'error';
        case 'unknown':
            return 'warning';
        case 'sending':
            return 'processing';
        case 'cancelled':
            return 'default';
        case 'pending':
            return 'processing';
        default:
            return 'default';
    }
};

export const severityColor = (value: string) => {
    switch (value.toLowerCase()) {
        case 'critical':
        case 'error':
            return 'error';
        case 'warning':
            return 'warning';
        case 'info':
            return 'processing';
        default:
            return 'default';
    }
};

export const SystemNotificationsPage = () => {
    const scope = useAdminReadScope('system-notifications');
    return scope.isCurrent() ? (
        <SystemNotificationsWorkspace key={scope.key} />
    ) : (
        <AppPage title='System Notifications' loading>
            <p>Checking administrator access…</p>
        </AppPage>
    );
};
const SystemNotificationsWorkspace = () => {
    const ctx = React.useContext(Context);
    const navigate = useNavigate();
    const location = useLocation();
    const [form] = Form.useForm();
    const {page, pageSize, setPage, params, setParams} = usePagedParams();
    const [keyword, setKeyword] = useKeywordParam('keyword');
    const status = params.get('status') || '';
    const telegramChat = params.get('chat') || '';
    const setFilter = (key: string, value: string) => {
        const next = new URLSearchParams(params);
        value === 'all' ? next.delete(key) : next.set(key, value);
        next.set('page', '1');
        setParams(next);
    };
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
            title: 'Title / Topic',
            render: item => (
                <div className='system-notification-title'>
                    <Button type='link' onClick={() => navigate(`/notifications/${item.id}${location.search}`)}>
                        {item.title || item.topicLabel}
                    </Button>
                    <span>{item.topicLabel || '—'}</span>
                </div>
            )
        },
        {title: 'Severity', dataIndex: 'severity', render: value => <Tag color={severityColor(String(value || ''))}>{value || '-'}</Tag>},
        {
            title: 'Chat / Channel',
            render: item => (
                <div className='system-notification-title'>
                    <span>{item.telegramChat || '—'}</span>
                    <span>{item.channel || '—'}</span>
                </div>
            )
        },
        {title: 'Status', dataIndex: 'status', render: value => <Tag color={statusColor(String(value || ''))}>{value || '-'}</Tag>},
        {title: 'Created', render: item => formatBeijingDateTime(item.createdAt) || '-'}
    ];

    const compactNotification = (item: NotificationDelivery) => (
        <article className='system-notification-card'>
            <div className='system-notification-card__heading'>
                <Button type='link' onClick={() => navigate(`/notifications/${item.id}${location.search}`)}>
                    {item.title || item.topicLabel || `Notification ${item.id}`}
                </Button>
            </div>
            <p className='admin-source-note'>{item.topicLabel || '—'}</p>
            <dl>
                <div>
                    <dt>Severity</dt>
                    <dd>
                        <Tag color={severityColor(item.severity)}>{item.severity || '-'}</Tag>
                    </dd>
                </div>
                <div>
                    <dt>Delivery</dt>
                    <dd>
                        <Tag color={statusColor(item.status)}>{item.status || '—'}</Tag>
                    </dd>
                </div>
                <div>
                    <dt>Telegram chat</dt>
                    <dd>
                        {item.telegramChat || '-'}
                        <br />
                        {item.channel || '-'}
                    </dd>
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
            stale={Boolean(data.error && data.data)}
            onRefresh={data.reload}
            extra={
                <Button type='primary' icon={<SendOutlined aria-hidden />} onClick={() => setTestOpen(true)}>
                    Test Notification
                </Button>
            }
            filters={
                <div className='system-notification-filters'>
                    <label>
                        Keyword
                        <SearchBar value={keyword} onChange={setKeyword} placeholder='Search notifications' />
                    </label>
                    <label>
                        Delivery status
                        <Select
                            aria-label='Filter by notification status'
                            value={status || 'all'}
                            options={[
                                {label: 'All statuses', value: 'all'},
                                ...['pending', 'sending', 'sent', 'failed', 'unknown', 'cancelled'].map(value => ({value, label: value}))
                            ]}
                            onChange={value => setFilter('status', value)}
                        />
                    </label>
                    <label>
                        Telegram chat
                        <Select
                            aria-label='Filter by Telegram chat'
                            value={telegramChat || 'all'}
                            options={[{label: 'All chats', value: 'all'}, ...['test', 'prod'].map(value => ({value, label: value}))]}
                            onChange={value => setFilter('chat', value)}
                        />
                    </label>
                </div>
            }>
            <ResourceTable
                rowKey='id'
                label='System notification deliveries'
                items={data.data?.items || []}
                columns={columns}
                compactRender={compactNotification}
                compactEmptyDescription='No system notification deliveries'
                loading={data.loading}
                hasData={data.data !== undefined}
                total={data.data?.total}
                page={page}
                pageSize={pageSize}
                onPageChange={setPage}
                scrollX={760}
                stickyHeader={true}
            />
            <Modal
                className='admin-operation-modal'
                keyboard={!testSubmitting}
                open={testOpen}
                title='Test Notification'
                footer={null}
                closable={!testSubmitting}
                maskClosable={!testSubmitting}
                onCancel={closeTest}>
                <p>Sends to the configured test chat. Queued does not mean delivered.</p>
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
                        <Button type='primary' htmlType='submit' icon={<SendOutlined aria-hidden />} loading={testSubmitting}>
                            Send Test Notification
                        </Button>
                    </Space>
                </Form>
            </Modal>
        </AppPage>
    );
};
