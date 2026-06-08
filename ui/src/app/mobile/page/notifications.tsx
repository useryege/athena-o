import {SendOutlined} from '@ant-design/icons';
import {Button, Dropdown, Select, Space, Tag} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {useNavigate} from 'react-router-dom';
import {AppPage, CardTitle, MetricRow, ResponsiveResourceList, SearchBar, useAsyncData, useBreakpoint} from '../components';
import {services} from '../../shared/services';
import {NotificationDelivery} from '../../shared/services/notification-service';
import {useKeywordParam, usePagedParams} from './shared';
import {notificationTestTopics} from './notification-shared';

export const NotificationsPage = () => {
    const navigate = useNavigate();
    const {isMobile} = useBreakpoint();
    const {page, pageSize, setPage} = usePagedParams();
    const [keyword, setKeyword] = useKeywordParam('keyword');
    const [status, setStatus] = React.useState('');
    const data = useAsyncData(() => services.notification.listNotifications({page, pageSize, keyword, status: status || undefined}), [page, pageSize, keyword, status]);
    const sendTest = async (topic: string) => {
        await services.notification.sendTestNotification(topic);
        data.reload();
    };
    const testNotificationActions = isMobile ? (
        <Dropdown
            menu={{
                items: notificationTestTopics.map(item => ({key: item.topic, label: item.label})),
                onClick: item => void sendTest(item.key)
            }}
            trigger={['click']}>
            <Button icon={<SendOutlined />}>Test</Button>
        </Dropdown>
    ) : (
        <Space>
            {notificationTestTopics.map(item => (
                <Button key={item.topic} icon={<SendOutlined />} onClick={() => void sendTest(item.topic)}>
                    {item.label}
                </Button>
            ))}
        </Space>
    );
    const columns: ColumnsType<NotificationDelivery> = [
        {
            title: 'Title',
            render: item => (
                <Button type='link' onClick={() => navigate(`/notifications/${item.id}`)}>
                    {item.title || item.topic}
                </Button>
            )
        },
        {title: 'Severity', dataIndex: 'severity'},
        {title: 'Topic', dataIndex: 'topic'},
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
            extra={testNotificationActions}
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
                    <div onClick={() => navigate(`/notifications/${item.id}`)}>
                        <CardTitle title={item.title || item.topic} subtitle={item.body} tags={<Tag>{item.status}</Tag>} />
                        <MetricRow
                            items={[
                                {label: 'Severity', value: item.severity},
                                {label: 'Channel', value: item.channel},
                                {label: 'Created', value: item.createdAt}
                            ]}
                        />
                    </div>
                )}
            />
        </AppPage>
    );
};
