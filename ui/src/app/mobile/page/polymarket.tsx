import {Select, Tag} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {AppPage, CardTitle, MetricRow, ResponsiveResourceList, useAsyncData} from '../components';
import {services} from '../../shared/services';
import {PolymarketHotMarketItem, PolymarketMoverMarketItem, PolymarketRealtimeMarketItem, PolymarketSportsLiveEventItem, PolymarketSportsLiveMarketItem} from '../../shared/services/polymarket-service';
import {boolTag, fmt, fmtNumber} from './shared';

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

export const PolymarketSportsLivePage = () => {
    const [view, setView] = React.useState<'events' | 'markets'>('events');
    const events = useAsyncData(() => services.polymarket.getSportsLiveSnapshot(30), []);
    const markets = useAsyncData(() => services.polymarket.listSportsLiveMarkets(200), []);
    const eventColumns: ColumnsType<PolymarketSportsLiveEventItem> = [
        {title: 'Event', render: item => <CardTitle title={item.title} subtitle={item.gameStatus || item.period} image={item.image} />},
        {title: 'Score', dataIndex: 'score'},
        {title: 'Live', render: item => boolTag(item.live)},
        {title: 'Markets', render: item => item.markets?.reduce((sum: number, group: {markets: unknown[]}) => sum + group.markets.length, 0)}
    ];
    const marketColumns: ColumnsType<PolymarketSportsLiveMarketItem> = [
        {title: 'Market', render: item => <CardTitle title={item.title} subtitle={item.eventSlug} image={item.image} />},
        {title: 'Score', dataIndex: 'score'},
        {title: 'Volume', dataIndex: 'volumeNum'},
        {title: 'Liquidity', dataIndex: 'liquidityNum'}
    ];
    return (
        <AppPage
            title='Sports Live'
            loading={events.loading || markets.loading}
            error={events.error || markets.error}
            onRefresh={() => {
                events.reload();
                markets.reload();
            }}
            filters={
                <Select
                    value={view}
                    style={{width: 160}}
                    onChange={setView}
                    options={[
                        {value: 'events', label: 'Events'},
                        {value: 'markets', label: 'Markets'}
                    ]}
                />
            }>
            {view === 'events' ? (
                <ResponsiveResourceList
                    rowKey='eventSlug'
                    items={events.data?.events || []}
                    columns={eventColumns}
                    loading={events.loading}
                    card={item => (
                        <>
                            <CardTitle
                                title={item.title}
                                subtitle={item.score || item.gameStatus}
                                image={item.image}
                                tags={item.live ? <Tag color='red'>Live</Tag> : <Tag>{item.period}</Tag>}
                            />
                            <MetricRow
                                items={[
                                    {label: 'Elapsed', value: item.elapsed},
                                    {label: 'Markets', value: item.markets?.length || 0}
                                ]}
                            />
                        </>
                    )}
                />
            ) : (
                <ResponsiveResourceList
                    rowKey='conditionId'
                    items={markets.data?.items || []}
                    columns={marketColumns}
                    loading={markets.loading}
                    card={item => (
                        <>
                            <CardTitle title={item.title} subtitle={item.score || item.eventSlug} image={item.image} />
                            <MetricRow
                                items={[
                                    {label: 'Volume', value: fmtNumber(item.volumeNum)},
                                    {label: 'Liquidity', value: fmtNumber(item.liquidityNum)}
                                ]}
                            />
                        </>
                    )}
                />
            )}
        </AppPage>
    );
};
