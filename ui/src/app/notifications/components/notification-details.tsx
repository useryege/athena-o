import {MockupList, Page} from 'argo-ui';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';

import {services} from '../../shared/services';
import {NotificationDelivery} from '../../shared/services/notification-service';

interface NotificationDetailsRouteParams {
    id: string;
}

interface NotificationDetailsState {
    item: NotificationDelivery | null;
    loading: boolean;
    error: Error | null;
}

const isAbortedError = (err: unknown) =>
    String((err as any)?.message || '')
        .toLowerCase()
        .includes('abort');

const formatDate = (value: string) => {
    if (!value) {
        return '-';
    }
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
};

const DetailRow = ({label, value, monospace}: {label: string; value: React.ReactNode; monospace?: boolean}) => (
    <div className='notifications-detail__row'>
        <div>{label}</div>
        <div className={monospace ? 'notifications-detail__value notifications-detail__value--mono' : 'notifications-detail__value'}>{value || '-'}</div>
    </div>
);

export class NotificationDetails extends React.Component<RouteComponentProps<NotificationDetailsRouteParams>, NotificationDetailsState> {
    private request: (Promise<NotificationDelivery> & {abort?: () => void}) | null = null;
    private mounted = false;

    constructor(props: RouteComponentProps<NotificationDetailsRouteParams>) {
        super(props);
        this.state = {item: null, loading: true, error: null};
    }

    public componentDidMount() {
        this.mounted = true;
        this.load();
    }

    public componentDidUpdate(prevProps: RouteComponentProps<NotificationDetailsRouteParams>) {
        if (prevProps.match.params.id !== this.props.match.params.id) {
            this.setState({item: null, loading: true, error: null}, () => this.load());
        }
    }

    public componentWillUnmount() {
        this.mounted = false;
        if (this.request?.abort) {
            this.request.abort();
        }
    }

    public render() {
        const item = this.state.item;
        return (
            <Page
                title='Notification Details'
                toolbar={{breadcrumbs: [{title: 'Notifications', path: '/notifications'}, {title: item?.title || `#${this.props.match.params.id}`}]}}>
                <div className='notifications-detail'>
                    {this.state.error && (
                        <div className='notifications-detail__error'>
                            <i className='fa fa-exclamation-triangle' /> Failed to load notification: {this.state.error.message}
                        </div>
                    )}

                    {this.state.loading && !item ? (
                        <MockupList height={50} marginTop={30} />
                    ) : (
                        <div className='argo-container'>
                            <div className='white-box notifications-detail__box'>
                                <div className='notifications-detail__header'>
                                    <button type='button' className='argo-button argo-button--base-o' onClick={this.backToList}>
                                        <i className='fa fa-chevron-left' /> Back
                                    </button>
                                    <span className={`notifications-detail__badge notifications-detail__badge--${item?.status || 'unknown'}`}>{item?.status || '-'}</span>
                                </div>

                                {item && (
                                    <React.Fragment>
                                        <div className='notifications-detail__grid'>
                                            <DetailRow label='ID' value={item.id} monospace={true} />
                                            <DetailRow label='Source' value={item.source} />
                                            <DetailRow
                                                label='Severity'
                                                value={
                                                    <span className={`notifications-detail__severity notifications-detail__severity--${item.severity || 'unknown'}`}>
                                                        {item.severity || '-'}
                                                    </span>
                                                }
                                            />
                                            <DetailRow label='Channel' value={item.channel} />
                                            <DetailRow label='Status' value={item.status} />
                                            <DetailRow label='Telegram Message ID' value={item.providerMessageId} monospace={true} />
                                            <DetailRow label='Created At' value={formatDate(item.createdAt)} />
                                            <DetailRow label='Sent At' value={formatDate(item.sentAt)} />
                                            <DetailRow
                                                label='Link'
                                                value={
                                                    item.link ? (
                                                        <a href={item.link} target='_blank' rel='noreferrer noopener'>
                                                            {item.link}
                                                        </a>
                                                    ) : (
                                                        '-'
                                                    )
                                                }
                                            />
                                            <DetailRow label='Error Message' value={item.errorMessage} />
                                        </div>

                                        <div className='notifications-detail__section'>
                                            <h3>Title</h3>
                                            <p>{item.title || '-'}</p>
                                        </div>
                                        <div className='notifications-detail__section'>
                                            <h3>Body</h3>
                                            <pre>{item.body || '-'}</pre>
                                        </div>
                                    </React.Fragment>
                                )}
                            </div>
                        </div>
                    )}
                </div>
            </Page>
        );
    }

    private load = async () => {
        if (this.request?.abort) {
            this.request.abort();
        }
        const req = services.notification.getNotification(this.props.match.params.id);
        this.request = req;
        try {
            const item = await req;
            if (this.mounted && this.request === req) {
                this.setState({item, error: null});
            }
        } catch (err) {
            if (this.mounted && this.request === req && !isAbortedError(err)) {
                this.setState({error: err as Error});
            }
        } finally {
            if (this.request === req) {
                this.request = null;
            }
            if (this.mounted) {
                this.setState({loading: false});
            }
        }
    };

    private backToList = () => {
        this.props.history.push(`/notifications${this.props.location.search}`);
    };
}
