import {MockupList, Page} from 'argo-ui';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';

import {services} from '../../shared/services';
import {ListNotificationsResult, NotificationDelivery} from '../../shared/services/notification-service';

const PAGE_SIZE = 20;

const STATUS_OPTIONS = [
    {label: 'All Statuses', value: ''},
    {label: 'Pending', value: 'pending'},
    {label: 'Sent', value: 'sent'},
    {label: 'Failed', value: 'failed'}
];

const SEVERITY_OPTIONS = [
    {label: 'All Severities', value: ''},
    {label: 'Info', value: 'info'},
    {label: 'Warning', value: 'warning'},
    {label: 'Error', value: 'error'},
    {label: 'Critical', value: 'critical'}
];

interface NotificationsListState {
    items: NotificationDelivery[];
    total: number;
    page: number;
    status: string;
    severity: string;
    source: string;
    keyword: string;
    loading: boolean;
    refreshing: boolean;
    error: Error | null;
}

const isAbortedError = (err: unknown) =>
    String((err as any)?.message || '')
        .toLowerCase()
        .includes('abort');

const parseQuery = (search: string) => {
    const params = new URLSearchParams(search);
    return {
        page: Math.max(Number(params.get('page') || 1) || 1, 1),
        status: params.get('status') || '',
        severity: params.get('severity') || '',
        source: params.get('source') || '',
        keyword: params.get('keyword') || ''
    };
};

const buildSearch = (filters: {page: number; status: string; severity: string; source: string; keyword: string}) => {
    const params = new URLSearchParams();
    if (filters.page > 1) {
        params.set('page', String(filters.page));
    }
    if (filters.status) {
        params.set('status', filters.status);
    }
    if (filters.severity) {
        params.set('severity', filters.severity);
    }
    if (filters.source.trim()) {
        params.set('source', filters.source.trim());
    }
    if (filters.keyword.trim()) {
        params.set('keyword', filters.keyword.trim());
    }
    const value = params.toString();
    return value ? `?${value}` : '';
};

const formatDate = (value: string) => {
    if (!value) {
        return '-';
    }
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
};

const displayTitle = (item: NotificationDelivery) => item.title || item.body || `Notification #${item.id}`;

export class NotificationsList extends React.Component<RouteComponentProps<any>, NotificationsListState> {
    private request: (Promise<ListNotificationsResult> & {abort?: () => void}) | null = null;
    private mounted = false;

    constructor(props: RouteComponentProps<any>) {
        super(props);
        const query = parseQuery(props.location.search);
        this.state = {
            items: [],
            total: 0,
            page: query.page,
            status: query.status,
            severity: query.severity,
            source: query.source,
            keyword: query.keyword,
            loading: true,
            refreshing: false,
            error: null
        };
    }

    public componentDidMount() {
        this.mounted = true;
        this.load();
    }

    public componentDidUpdate(prevProps: RouteComponentProps<any>) {
        if (prevProps.location.search !== this.props.location.search) {
            const query = parseQuery(this.props.location.search);
            this.setState({...query, loading: true}, () => this.load());
        }
    }

    public componentWillUnmount() {
        this.mounted = false;
        if (this.request?.abort) {
            this.request.abort();
        }
    }

    public render() {
        const totalPages = Math.max(Math.ceil(this.state.total / PAGE_SIZE), 1);
        const rangeStart = this.state.total === 0 ? 0 : (this.state.page - 1) * PAGE_SIZE + 1;
        const rangeEnd = Math.min(this.state.page * PAGE_SIZE, this.state.total);

        return (
            <Page title='Notifications' toolbar={{breadcrumbs: [{title: 'Notifications'}]}}>
                <div className='notifications-page'>
                    {this.state.error && (
                        <div className='notifications-page__error'>
                            <i className='fa fa-exclamation-triangle' /> Failed to load notifications: {this.state.error.message}
                        </div>
                    )}

                    {this.state.loading && this.state.items.length === 0 ? (
                        <MockupList height={50} marginTop={30} />
                    ) : (
                        <div className='argo-container'>
                            <div className='white-box notifications-page__box'>
                                <form className='notifications-page__filters' onSubmit={this.applyFilters}>
                                    <select className='argo-field' value={this.state.status} onChange={event => this.setState({status: event.target.value})}>
                                        {STATUS_OPTIONS.map(item => (
                                            <option key={item.value || 'all'} value={item.value}>
                                                {item.label}
                                            </option>
                                        ))}
                                    </select>
                                    <select className='argo-field' value={this.state.severity} onChange={event => this.setState({severity: event.target.value})}>
                                        {SEVERITY_OPTIONS.map(item => (
                                            <option key={item.value || 'all'} value={item.value}>
                                                {item.label}
                                            </option>
                                        ))}
                                    </select>
                                    <input
                                        className='argo-field'
                                        type='text'
                                        placeholder='Source'
                                        value={this.state.source}
                                        onChange={event => this.setState({source: event.target.value})}
                                    />
                                    <input
                                        className='argo-field'
                                        type='text'
                                        placeholder='Keyword'
                                        value={this.state.keyword}
                                        onChange={event => this.setState({keyword: event.target.value})}
                                    />
                                    <div className='notifications-page__filter-actions'>
                                        <button type='submit' className='argo-button argo-button--base'>
                                            Apply
                                        </button>
                                        <button type='button' className='argo-button argo-button--base-o' onClick={this.resetFilters}>
                                            Reset
                                        </button>
                                    </div>
                                </form>

                                <div className='notifications-page__summary'>
                                    <span>
                                        {rangeStart}-{rangeEnd} of {this.state.total}
                                    </span>
                                    <button type='button' className='argo-button argo-button--base-o' disabled={this.state.refreshing} onClick={this.refresh}>
                                        {this.state.refreshing ? 'Refreshing...' : 'Refresh'}
                                    </button>
                                </div>

                                <div className='argo-table-list argo-table-list--clickable notifications-page__table'>
                                    <div className='argo-table-list__head'>
                                        <div className='notifications-page__row'>
                                            <div>Title</div>
                                            <div>Status</div>
                                            <div>Severity</div>
                                            <div>Source</div>
                                            <div>Created At</div>
                                        </div>
                                    </div>
                                    {this.state.items.length === 0 ? (
                                        <div className='argo-table-list__row'>
                                            <div className='row'>
                                                <div className='columns small-12 text-center'>No notifications found</div>
                                            </div>
                                        </div>
                                    ) : (
                                        this.state.items.map(item => (
                                            <div
                                                className='argo-table-list__row'
                                                key={item.id}
                                                role='button'
                                                tabIndex={0}
                                                onClick={() => this.openDetails(item.id)}
                                                onKeyDown={event => this.handleRowKeyDown(event, item.id)}>
                                                <div className='notifications-page__row'>
                                                    <div className='notifications-page__title'>{displayTitle(item)}</div>
                                                    <div>
                                                        <span className={`notifications-page__badge notifications-page__badge--${item.status || 'unknown'}`}>
                                                            {item.status || '-'}
                                                        </span>
                                                    </div>
                                                    <div>
                                                        <span className={`notifications-page__severity notifications-page__severity--${item.severity || 'unknown'}`}>
                                                            {item.severity || '-'}
                                                        </span>
                                                    </div>
                                                    <div className='notifications-page__source'>{item.source || '-'}</div>
                                                    <div>{formatDate(item.createdAt)}</div>
                                                </div>
                                            </div>
                                        ))
                                    )}
                                </div>

                                <div className='notifications-page__pager'>
                                    <button
                                        type='button'
                                        className='argo-button argo-button--base-o'
                                        disabled={this.state.page <= 1 || this.state.refreshing}
                                        onClick={() => this.gotoPage(this.state.page - 1)}>
                                        <i className='fa fa-chevron-left' /> Prev
                                    </button>
                                    <span>
                                        Page {this.state.page} / {totalPages}
                                    </span>
                                    <button
                                        type='button'
                                        className='argo-button argo-button--base-o'
                                        disabled={this.state.page >= totalPages || this.state.refreshing}
                                        onClick={() => this.gotoPage(this.state.page + 1)}>
                                        Next <i className='fa fa-chevron-right' />
                                    </button>
                                </div>
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
        if (this.mounted) {
            this.setState({refreshing: true});
        }
        const req = services.notification.listNotifications({
            page: this.state.page,
            pageSize: PAGE_SIZE,
            status: this.state.status,
            severity: this.state.severity,
            source: this.state.source,
            keyword: this.state.keyword
        });
        this.request = req;
        try {
            const data = await req;
            if (this.mounted && this.request === req) {
                this.setState({items: data.items, total: data.total, page: data.page || this.state.page, error: null});
            }
        } catch (err) {
            if (this.mounted && this.request === req && !isAbortedError(err)) {
                this.setState({error: err as Error});
            }
        } finally {
            if (this.request === req) {
                this.request = null;
            }
            if (this.mounted && this.request === null) {
                this.setState({loading: false, refreshing: false});
            }
        }
    };

    private applyFilters = (event: React.FormEvent<HTMLFormElement>) => {
        event.preventDefault();
        this.gotoFilters(1);
    };

    private resetFilters = () => {
        this.props.history.push('/notifications');
    };

    private refresh = () => this.load();

    private gotoPage = (page: number) => this.gotoFilters(page);

    private gotoFilters(page: number) {
        const search = buildSearch({
            page,
            status: this.state.status,
            severity: this.state.severity,
            source: this.state.source,
            keyword: this.state.keyword
        });
        this.props.history.push(`/notifications${search}`);
    }

    private openDetails(id: number) {
        this.props.history.push(`/notifications/${id}${this.props.location.search}`);
    }

    private handleRowKeyDown(event: React.KeyboardEvent, id: number) {
        if (event.key === 'Enter' || event.key === ' ') {
            event.preventDefault();
            this.openDetails(id);
        }
    }
}
