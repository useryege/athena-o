import {ArrowLeftOutlined} from '@ant-design/icons';
import {Button} from 'antd';
import {useNavigate, useParams} from 'react-router-dom';
import {AppPage, KeyValueGrid, Section, useAsyncData} from '../../components';
import {formatBeijingDateTime} from '../../shared/format';
import {fmt} from '../../shared/pages/shared';
import {adminServices as services} from '../services';

const timeFields = new Set(['createdAt', 'sentAt']);

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
            <Section title='Delivery'>
                <KeyValueGrid
                    items={Object.entries(data.data || {}).map(([label, value]) => ({
                        label,
                        value: label === 'link' ? safeExternalLink(value) : timeFields.has(label) ? formatBeijingDateTime(String(value)) || '-' : fmt(value)
                    }))}
                />
            </Section>
        </AppPage>
    );
};
