import {useParams} from 'react-router-dom';
import {AppPage, KeyValueGrid, Section, useAsyncData} from '../components';
import {services} from '../../shared/services';
import {fmt} from './shared';

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
                            ) : (
                                fmt(value)
                            )
                    }))}
                />
            </Section>
        </AppPage>
    );
};
