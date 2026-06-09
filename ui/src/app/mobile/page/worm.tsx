import {Button, Empty, Select, Skeleton, Space} from 'antd';
import * as React from 'react';
import {AppPage} from '../components';
import {services} from '../../shared/services';
import {DEFAULT_WORM_MARKET_CATEGORY, DEFAULT_WORM_MARKET_SORT, ListWormEventsResult, WormEventItem} from '../../shared/services/worm-service';
import {WormEventCard} from './worm-shared';

const PAGE_SIZE = 20;

export const WormPage = () => {
    const [events, setEvents] = React.useState<WormEventItem[]>([]);
    const [nextCursor, setNextCursor] = React.useState('');
    const [fetchedAt, setFetchedAt] = React.useState<number>();
    const [stale, setStale] = React.useState(false);
    const [loading, setLoading] = React.useState(true);
    const [loadingMore, setLoadingMore] = React.useState(false);
    const [error, setError] = React.useState<Error>();
    const requestRef = React.useRef<(Promise<ListWormEventsResult> & {abort?: () => void}) | null>(null);
    const requestIDRef = React.useRef(0);

    const load = React.useCallback((cursor = '', append = false) => {
        const requestID = ++requestIDRef.current;
        requestRef.current?.abort?.();
        append ? setLoadingMore(true) : setLoading(true);
        setError(undefined);
        const request = services.worm.listEvents({
            limit: PAGE_SIZE,
            cursor,
            sortOption: DEFAULT_WORM_MARKET_SORT,
            categorySlug: DEFAULT_WORM_MARKET_CATEGORY
        });
        requestRef.current = request;
        request.then(
            result => {
                if (requestID !== requestIDRef.current) {
                    return;
                }
                setEvents(current => {
                    if (!append) {
                        return result.items;
                    }
                    const seen = new Set(current.map(item => item.conditionId));
                    return [...current, ...result.items.filter(item => !seen.has(item.conditionId))];
                });
                setNextCursor(result.nextCursor || '');
                setFetchedAt(result.fetchedAt);
                setStale(Boolean(result.stale));
                setLoading(false);
                setLoadingMore(false);
            },
            reason => {
                if (requestID !== requestIDRef.current) {
                    return;
                }
                setError(reason instanceof Error ? reason : new Error(String(reason?.message || reason)));
                setLoading(false);
                setLoadingMore(false);
            }
        );
    }, []);

    React.useEffect(() => {
        load();
        return () => {
            requestIDRef.current++;
            requestRef.current?.abort?.();
        };
    }, [load]);

    return (
        <AppPage
            title='Worm'
            subtitle={`Events grouped from Worm markets${fetchedAt ? ` · Fetched ${fetchedAt}` : ''}${stale ? ' (stale)' : ''}`}
            loading={loading && events.length === 0}
            error={error}
            onRefresh={() => load()}
            filters={
                <Space wrap={true}>
                    <Select
                        disabled={true}
                        value={DEFAULT_WORM_MARKET_SORT}
                        style={{width: 160}}
                        options={['new', 'trending', 'ending_soon', 'leverage'].map(value => ({value, label: value}))}
                    />
                    <Select
                        disabled={true}
                        value={DEFAULT_WORM_MARKET_CATEGORY}
                        style={{width: 150}}
                        options={['all', 'politics', 'sports', 'crypto', 'tech', 'finance', 'wtf'].map(value => ({value, label: value}))}
                    />
                </Space>
            }>
            {loading && events.length === 0 ? (
                <Skeleton active={true} />
            ) : events.length === 0 ? (
                <Empty description='No Worm events' />
            ) : (
                <>
                    <div className='worm-event-grid'>
                        {events.map(item => (
                            <WormEventCard key={item.conditionId} item={item} />
                        ))}
                    </div>
                    {nextCursor && (
                        <div className='worm-event-load-more'>
                            <Button loading={loadingMore} onClick={() => load(nextCursor, true)}>
                                Load more
                            </Button>
                        </div>
                    )}
                </>
            )}
        </AppPage>
    );
};
