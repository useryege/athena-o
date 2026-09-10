import {ArrowLeftOutlined} from '@ant-design/icons';
import {Alert, Button, Tag} from 'antd';
import {useNavigate, useParams} from 'react-router-dom';
import {AppPage, KeyValueGrid, Section, useAsyncData} from '../../components';
import {formatBeijingDateTime} from '../../shared/format';
import {fmt} from '../../shared/pages/shared';
import {adminServices as services} from '../services';

const timeFields = new Set(['createdAt', 'authorizedAt', 'startedAt', 'resultAt', 'sentAt']);
const fieldLabels: Record<string, string> = {createdAt: 'Created', authorizedAt: 'Send authorized', startedAt: 'HTTP started', resultAt: 'Result recorded', sentAt: 'Sent'};

const safeExternalLink = (value: unknown) => {
    const raw = String(value || '');
    try {
        const url = new URL(raw);
        if (url.protocol !== 'https:' && url.protocol !== 'http:') {
            return fmt(value);
        }
        return (
            <a href={url.toString()} target='_blank' rel='noreferrer'>
                {raw}
            </a>
        );
    } catch {
        return fmt(value);
    }
};

export const SystemNotificationDetailPage = () => {
    const {id = ''} = useParams();
    const navigate = useNavigate();
    const data = useAsyncData(() => services.adminNotifications.getNotification(id), [id]);
    return (
        <AppPage
            title='System Notification Detail'
            subtitle='Inspect the provider-facing result for one operational delivery.'
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}
            extra={
                <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/notifications')}>
                    All notifications
                </Button>
            }>
            {data.data?.status === 'unknown' && (
                <Alert type='warning' title='Delivery result unknown' description='Telegram may have received this message. It will not be resent automatically.' />
            )}
            <Section title='Delivery'>
                <KeyValueGrid
                    items={Object.entries(data.data || {}).map(([label, value]) => ({
                        label: fieldLabels[label] || label,
                        value:
                            label === 'status' ? (
                                <Tag color={value === 'sent' ? 'green' : value === 'failed' ? 'red' : value === 'unknown' ? 'orange' : value === 'sending' ? 'cyan' : 'default'}>
                                    {String(value)}
                                </Tag>
                            ) : label === 'link' ? (
                                safeExternalLink(value)
                            ) : timeFields.has(label) ? (
                                formatBeijingDateTime(String(value)) || '-'
                            ) : (
                                fmt(value)
                            )
                    }))}
                />
            </Section>
        </AppPage>
    );
};
