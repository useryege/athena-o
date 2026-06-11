import type {ColumnsType} from 'antd/es/table';
import {Space, Tag, Typography} from 'antd';
import {AppPage, CardTitle, MetricRow, ResponsiveResourceList, useAsyncData} from '../components';
import {services} from '../../shared/services';
import {
    PolymarketHotMarketItem,
    PolymarketMoverMarketItem,
    PolymarketRealtimeMarketItem,
    PolymarketSportsLiveEventCardItem,
    PolymarketSportsLiveMarketCardItem
} from '../../shared/services/polymarket-service';
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

const parseGammaList = (value?: string): string[] => {
    if (!value) {
        return [];
    }
    try {
        const parsed = JSON.parse(value);
        return Array.isArray(parsed) ? parsed.map(item => String(item)) : [];
    } catch {
        return value
            .split(',')
            .map(item => item.trim())
            .filter(Boolean);
    }
};

const moneylineSummary = (market: PolymarketSportsLiveMarketCardItem) => {
    const outcomes = parseGammaList(market.outcomes);
    const prices = parseGammaList(market.outcomePrices);
    if (outcomes.length === 0) {
        return undefined;
    }
    return outcomes.map((outcome, index) => `${outcome}: ${prices[index] || '-'}`).join(' / ');
};

const isMoneylineMarket = (market: PolymarketSportsLiveMarketCardItem) => {
    const type = market.sportsMarketType.trim().toLowerCase();
    return type === 'moneyline';
};

const SportsLiveEventCard = (props: {item: PolymarketSportsLiveEventCardItem}) => {
    const moneyline = props.item.markets.find(isMoneylineMarket);
    const moneylineText = moneyline ? moneylineSummary(moneyline) : undefined;

    return (
        <>
            <CardTitle
                title={props.item.title}
                subtitle={[props.item.score, props.item.period, props.item.elapsed].filter(Boolean).join(' · ') || props.item.slug}
                image={props.item.image}
                tags={
                    <Space wrap={true}>
                        <Tag color='green'>Live</Tag>
                        {props.item.gameStatus && <Tag>{props.item.gameStatus}</Tag>}
                    </Space>
                }
            />
            <MetricRow
                items={[
                    {label: 'Volume', value: fmtNumber(props.item.volume)},
                    {label: 'Liquidity', value: fmtNumber(props.item.liquidity)},
                    {label: 'Updated', value: fmt(props.item.updatedAt)}
                ]}
            />
            {moneylineText && (
                <Typography.Paragraph style={{margin: '8px 0 0'}}>
                    <Typography.Text strong={true}>Moneyline: </Typography.Text>
                    <Typography.Text>{moneylineText}</Typography.Text>
                </Typography.Paragraph>
            )}
        </>
    );
};

export const PolymarketSportsLivePage = () => {
    const events = useAsyncData(() => services.polymarket.listSportsLiveEvents(200), []);
    const eventColumns: ColumnsType<PolymarketSportsLiveEventCardItem> = [
        {
            title: 'Event',
            render: item => (
                <CardTitle
                    title={item.title}
                    subtitle={item.slug}
                    image={item.image}
                    tags={
                        <Space wrap={true}>
                            <Tag color='green'>Live</Tag>
                            {item.gameStatus && <Tag>{item.gameStatus}</Tag>}
                        </Space>
                    }
                />
            )
        },
        {title: 'Score', dataIndex: 'score'},
        {title: 'Volume', render: item => fmtNumber(item.volume)},
        {title: 'Liquidity', render: item => fmtNumber(item.liquidity)},
        {title: 'Markets', dataIndex: 'marketCount'},
        {title: 'Updated', dataIndex: 'updatedAt'}
    ];
    return (
        <AppPage
            title='Sports Live'
            subtitle={`Fetched ${fmt(events.data?.fetchedAt)} ${events.data?.stale ? '(stale)' : ''}`}
            loading={events.loading}
            error={events.error}
            onRefresh={events.reload}>
            <ResponsiveResourceList
                rowKey='eventKey'
                items={events.data?.items || []}
                columns={eventColumns}
                loading={events.loading}
                card={item => <SportsLiveEventCard item={item} />}
            />
        </AppPage>
    );
};
