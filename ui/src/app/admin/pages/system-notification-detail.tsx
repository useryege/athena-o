import {OperationFacts} from '../components/operation-facts';
import {ArrowLeftOutlined} from '@ant-design/icons';
import {Alert, Button, Tag} from 'antd';
import {useNavigate, useParams, useLocation} from 'react-router-dom';
import {AppPage, Section, useAsyncData} from '../../components';
import {formatBeijingDateTime} from '../../shared/format';
import {useAdminReadScope} from '../read-scope';
import {statusColor, severityColor} from './system-notifications';
import {fmt} from '../../shared/pages/shared';
import {adminServices as services} from '../services';

const timeFields = ['createdAt', 'authorizedAt', 'startedAt', 'resultAt', 'sentAt'] as const;
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
    const scope = useAdminReadScope('system-notification-detail');
    return scope.isCurrent() ? (
        <SystemNotificationDetailWorkspace key={scope.key} />
    ) : (
        <AppPage title='System Notification Detail' loading>
            <p>Checking administrator access…</p>
        </AppPage>
    );
};
const SystemNotificationDetailWorkspace = () => {
    const {id = ''} = useParams();
    const navigate = useNavigate();
    const location = useLocation();
    const data = useAsyncData(() => services.adminNotifications.getNotification(id), [id]);
    return (
        <AppPage
            title='System Notification Detail'
            subtitle='Inspect the provider-facing result for one operational delivery.'
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}
            extra={
                <Button icon={<ArrowLeftOutlined aria-hidden />} onClick={() => navigate(`/notifications${location.search}`)}>
                    All notifications
                </Button>
            }>
            {data.data?.status === 'unknown' && (
                <Alert type='warning' title='Delivery result unknown' description='Telegram may have received this message. It will not be resent automatically.' />
            )}
            {data.data && (
                <div className='system-notification-detail-grid'>
                    <div className='system-notification-message'>
                        <Section title={data.data.title || 'Message'}>
                            <p className='admin-source-note'>Topic · {data.data.topicLabel || '—'}</p>
                            <p className='system-notification-body'>{data.data.body || '—'}</p>
                            <div className='admin-operation-divider'>
                                <p>Related link</p>
                                {safeExternalLink(data.data.link)}
                            </div>
                        </Section>
                    </div>
                    <div className='system-notification-delivery'>
                        <Section title='Delivery record'>
                            <p className='admin-source-note'>Latest recorded delivery result.</p>
                            <OperationFacts
                                items={[
                                    {label: 'Status', value: <Tag color={statusColor(data.data.status)}>{data.data.status || '—'}</Tag>},
                                    {label: 'Severity', value: <Tag color={severityColor(data.data.severity)}>{data.data.severity || '—'}</Tag>},
                                    {label: 'Telegram chat', value: data.data.telegramChat || '—'},
                                    {label: 'Channel', value: data.data.channel || '—'}
                                ]}
                            />
                            <div className='admin-operation-divider'>
                                <h3>Provider result</h3>
                                <p className='athena-identifier'>{data.data.errorMessage || '—'}</p>
                            </div>
                            <details className='admin-operation-details'>
                                <summary>Delivery identifiers</summary>
                                <OperationFacts
                                    columns={1}
                                    items={[
                                        {label: 'ID', value: <span className='athena-identifier'>{data.data.id}</span>},
                                        {label: 'Source', value: <span className='athena-identifier'>{data.data.source || '—'}</span>},
                                        {label: 'Provider message ID', value: <span className='athena-identifier'>{data.data.providerMessageId || '—'}</span>}
                                    ]}
                                />
                            </details>
                        </Section>
                    </div>
                    <div className='system-notification-timeline'>
                        <Section title='Delivery timeline'>
                            <p className='admin-source-note'>Times in UTC+8</p>
                            <dl>
                                {timeFields.map(field => (
                                    <div key={field}>
                                        <dt>{fieldLabels[field]}</dt>
                                        <dd>{formatBeijingDateTime(data.data?.[field]) || '—'}</dd>
                                    </div>
                                ))}
                            </dl>
                            <p className='admin-source-note'>A dash means no timestamp was recorded.</p>
                        </Section>
                    </div>
                </div>
            )}
        </AppPage>
    );
};
