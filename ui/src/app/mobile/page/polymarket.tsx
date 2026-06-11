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

const moneylineOptions = (market?: PolymarketSportsLiveMarketCardItem) => {
    if (!market) {
        return [];
    }
    const outcomes = parseGammaList(market.outcomes);
    const prices = parseGammaList(market.outcomePrices);
    return outcomes.slice(0, 2).map((outcome, index) => ({
        outcome,
        price: prices[index] || '-'
    }));
};

const isMoneylineMarket = (market: PolymarketSportsLiveMarketCardItem) => {
    const type = market.sportsMarketType.trim().toLowerCase();
    return type === 'moneyline';
};

const moneylineMarket = (item: PolymarketSportsLiveEventCardItem) => item.markets.find(isMoneylineMarket) || item.markets[0];

const MoneylineOutcomeBlocks = (props: {market?: PolymarketSportsLiveMarketCardItem}) => {
    const options = moneylineOptions(props.market);
    if (options.length === 0) {
        return <Typography.Text type='secondary'>-</Typography.Text>;
    }
    return (
        <div className='moneyline-outcomes'>
            {options.map(option => (
                <div className='moneyline-outcome' key={option.outcome}>
                    <Typography.Text className='moneyline-outcome__name'>{option.outcome}</Typography.Text>
                    <Typography.Text className='moneyline-outcome__price' strong={true}>
                        {option.price}
                    </Typography.Text>
                </div>
            ))}
        </div>
    );
};

const SportsLiveEventCard = (props: {item: PolymarketSportsLiveEventCardItem}) => {
    const moneyline = moneylineMarket(props.item);

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
            <div className='moneyline-outcomes-wrap'>
                <Typography.Text className='moneyline-outcomes-label' strong={true}>
                    Moneyline
                </Typography.Text>
                <MoneylineOutcomeBlocks market={moneyline} />
            </div>
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
        {
            title: 'Moneyline',
            render: item => <MoneylineOutcomeBlocks market={moneylineMarket(item)} />
        },
        {title: 'Volume', render: item => fmtNumber(item.volume)},
        {title: 'Liquidity', render: item => fmtNumber(item.liquidity)},
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
