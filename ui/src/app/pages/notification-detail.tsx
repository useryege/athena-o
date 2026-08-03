import {useParams} from 'react-router-dom';
import {AppPage, KeyValueGrid, Section, useAsyncData} from '../components';
import {formatBeijingDateTime} from '../shared/format';
import {services} from '../shared/services';
import {fmt} from './shared';

const timeFields = new Set(['createdAt', 'sentAt']);

export const NotificationsDetailPage = () => {
    const {id = ''} = useParams();
    const data = useAsyncData(() => services.notification.getNotification(id), [id]);
    return (
        <AppPage title='Notification Detail' loading={data.loading} error={data.error} onRefresh={data.reload}>
            <Section title='Delivery'>
                <KeyValueGrid
                    items={Object.entries(data.data || {}).map(([label, value]) => ({
                        label,
                        value:
                            label === 'link' ? (
                                <a href={String(value)} target='_blank' rel='noreferrer'>
                                    {String(value)}
                                </a>
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
