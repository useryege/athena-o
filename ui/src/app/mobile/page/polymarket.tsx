import type {ColumnsType} from 'antd/es/table';
import {Empty} from 'antd';
import * as React from 'react';
import {AppPage, CardTitle, MetricRow, ResponsiveResourceList, useAsyncData} from '../components';
import {services} from '../../shared/services';
import {
    PolymarketHotMarketItem,
    PolymarketMoverMarketItem,
    PolymarketRealtimeMarketItem,
    PolymarketSportsLivePriceHistorySeriesItem
} from '../../shared/services/polymarket-service';
import {SportsLiveEventCard, moneylineMarket, moneylineMarketKeys} from './polymarket-sports-live-card';
import {fmt, fmtNumber} from './shared';

const PolymarketListPage = <T extends PolymarketHotMarketItem | PolymarketRealtimeMarketItem | PolymarketMoverMarketItem>(props: {
    title: string;
    load: () => Promise<{items: T[]; fetchedAt?: number; stale?: boolean; [key: string]: any}> & {abort?: () => void};
}) => {
    const data = useAsyncData(props.load, []);
    const columns: ColumnsType<T> = [
        {title: 'Market', render: item => <CardTitle title={(item as any).question} subtitle={(item as any).eventSlug} image={(item as any).image} />},
        {title: 'Volume', render: item => fmtNumber((item as any).volume24hr || (item as any).volumeNum)},
        {title: 'Liquidity', render: item => fmtNumber((item as any).liquidityNum)},
        {title: 'Spread', render: item => fmt((item as any).spread)},
        {title: 'Updated', dataIndex: 'updatedAt'}
    ];
    return (
        <AppPage
            title={props.title}
            subtitle={`Fetched ${fmt(data.data?.fetchedAt)} ${data.data?.stale ? '(stale)' : ''}`}
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}>
            <ResponsiveResourceList
                rowKey={item => (item as any).conditionId}
                items={data.data?.items || []}
                columns={columns}
                loading={data.loading}
                card={item => (
                    <>
                        <CardTitle title={(item as any).question} subtitle={(item as any).eventSlug} image={(item as any).image} />
                        <MetricRow
                            items={[
                                {label: 'Volume', value: fmtNumber((item as any).volume24hr || (item as any).volumeNum)},
                                {label: 'Liquidity', value: fmtNumber((item as any).liquidityNum)},
                                {label: 'Tokens', value: (item as any).tokens?.length || 0}
                            ]}
                        />
                    </>
                )}
            />
        </AppPage>
    );
};

export const PolymarketHotPage = () => <PolymarketListPage title='Polymarket Hot Markets' load={() => services.polymarket.listHotMarkets(100)} />;
export const PolymarketRealtimePage = () => <PolymarketListPage title='Polymarket Realtime' load={() => services.polymarket.listRealtimeMarkets(100)} />;
export const PolymarketMoversPage = () => <PolymarketListPage title='Polymarket Movers' load={() => services.polymarket.listMovers(100)} />;

const sportsLiveRefreshIntervalMs = 3000;

export const PolymarketSportsLivePage = () => {
    const events = useAsyncData(() => services.polymarket.listSportsLiveEvents(200), []);
    const marketKeys = React.useMemo(() => moneylineMarketKeys(events.data?.items), [events.data?.items]);
    const marketKeySignature = React.useMemo(() => marketKeys.join('|'), [marketKeys]);
    const history = useAsyncData(() => {
        if (marketKeys.length === 0) {
            return Promise.resolve({items: []}) as Promise<{items: PolymarketSportsLivePriceHistorySeriesItem[]}> & {abort?: () => void};
        }
        return services.polymarket.batchGetSportsLivePriceHistory(marketKeys, 360);
    }, [marketKeySignature]);
    const eventsReloadRef = React.useRef(events.reload);
    const historyReloadRef = React.useRef(history.reload);
    const marketKeyCountRef = React.useRef(marketKeys.length);
    eventsReloadRef.current = events.reload;
    historyReloadRef.current = history.reload;
    marketKeyCountRef.current = marketKeys.length;
    const reloadAll = React.useCallback(() => {
        eventsReloadRef.current();
        if (marketKeyCountRef.current > 0) {
            historyReloadRef.current();
        }
    }, []);

    React.useEffect(() => {
        const timer = window.setInterval(reloadAll, sportsLiveRefreshIntervalMs);
        return () => window.clearInterval(timer);
    }, [reloadAll]);

    const historyByMarketKey = React.useMemo(() => {
        const out = new Map<string, PolymarketSportsLivePriceHistorySeriesItem[]>();
        if (history.error) {
            return out;
        }
        (history.data?.items || []).forEach(item => {
            const items = out.get(item.marketKey) || [];
            items.push(item);
            out.set(item.marketKey, items);
        });
        return out;
    }, [history.data?.items, history.error]);
    const items = events.data?.items || [];
    return (
        <AppPage
            title='Sports Live'
            subtitle={`Fetched ${fmt(events.data?.fetchedAt)} ${events.data?.stale ? '(stale)' : ''}`}
            loading={events.loading}
            error={events.error}
            onRefresh={reloadAll}>
            <div className='sports-live-list'>
                {!events.loading && items.length === 0 && <Empty description='No data' />}
                {items.map(item => {
                    const moneyline = moneylineMarket(item);
                    return <SportsLiveEventCard item={item} history={historyByMarketKey.get(moneyline?.marketKey || '')} key={item.eventKey} />;
                })}
            </div>
        </AppPage>
    );
};
