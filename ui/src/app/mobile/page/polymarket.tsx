import {LinkOutlined} from '@ant-design/icons';
import type {ColumnsType} from 'antd/es/table';
import {Button, Empty, Space, Tag, Tooltip, Typography} from 'antd';
import * as React from 'react';
import {AppPage, CardTitle, MetricRow, ResponsiveResourceList, useAsyncData} from '../components';
import {services} from '../../shared/services';
import {
    PolymarketHotMarketItem,
    PolymarketMoverMarketItem,
    PolymarketRealtimeMarketItem,
    PolymarketSportsLiveEventCardItem,
    PolymarketSportsLiveMarketCardItem,
    PolymarketSportsLivePriceHistorySeriesItem,
    PolymarketSportsLiveTeamItem
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

const normalizedTeamKey = (value?: string) => (value || '').trim().toLowerCase();

const matchingTeam = (outcome: string, teams: PolymarketSportsLiveTeamItem[], index: number) => {
    const key = normalizedTeamKey(outcome);
    if (key) {
        const exact = teams.find(team => [team.name, team.abbreviation, team.alias].some(value => normalizedTeamKey(value) === key));
        if (exact) {
            return exact;
        }
    }
    return teams[index];
};

const moneylineOptions = (market?: PolymarketSportsLiveMarketCardItem, teams: PolymarketSportsLiveTeamItem[] = []) => {
    if (!market) {
        return [];
    }
    const outcomes = parseGammaList(market.outcomes);
    const prices = parseGammaList(market.outcomePrices);
    return outcomes.slice(0, 2).map((outcome, index) => ({
        outcome,
        price: prices[index] || '-',
        logo: matchingTeam(outcome, teams, index)?.logo
    }));
};

const isMoneylineMarket = (market: PolymarketSportsLiveMarketCardItem) => {
    const type = market.sportsMarketType.trim().toLowerCase();
    return type === 'moneyline';
};

const moneylineMarket = (item: PolymarketSportsLiveEventCardItem) => item.markets.find(isMoneylineMarket) || item.markets[0];

const moneylineMarketKeys = (items: PolymarketSportsLiveEventCardItem[] = []) => {
    const keys = new Set<string>();
    items.forEach(item => {
        const marketKey = moneylineMarket(item)?.marketKey;
        if (marketKey) {
            keys.add(marketKey);
        }
    });
    return Array.from(keys);
};

const resolvedPriceHistory = (history: PolymarketSportsLivePriceHistorySeriesItem[] = [], outcome: string, index: number) => {
    const outcomeKey = normalizedTeamKey(outcome);
    if (outcomeKey) {
        const matched = history.find(item => normalizedTeamKey(item.outcome) === outcomeKey);
        if (matched) {
            return matched;
        }
    }
    return history[index];
};

const polymarketEventURL = (item: PolymarketSportsLiveEventCardItem) => {
    const slug = item.slug.trim();
    return slug ? `https://polymarket.com/event/${encodeURIComponent(slug)}` : '';
};

const PolymarketEventLink = (props: {item: PolymarketSportsLiveEventCardItem}) => {
    const url = polymarketEventURL(props.item);
    if (!url) {
        return null;
    }
    return (
        <Tooltip title='Open Polymarket'>
            <Button
                aria-label='Open Polymarket'
                href={url}
                icon={<LinkOutlined />}
                rel='noopener noreferrer'
                size='small'
                target='_blank'
                type='text'
                onClick={event => event.stopPropagation()}
            />
        </Tooltip>
    );
};

const chartPercent = (value?: number) => {
    if (value === undefined || !Number.isFinite(value)) {
        return '-';
    }
    return `${Math.round(value * 100)}%`;
};

const latestPrice = (series?: PolymarketSportsLivePriceHistorySeriesItem) => {
    const prices = series?.prices || [];
    for (let index = prices.length - 1; index >= 0; index -= 1) {
        const value = prices[index];
        if (Number.isFinite(value)) {
            return value;
        }
    }
    return undefined;
};

const pricePoints = (series?: PolymarketSportsLivePriceHistorySeriesItem) =>
    (series?.prices || [])
        .map((price, index) => ({
            price,
            timestamp: series?.timestamps[index] || index
        }))
        .filter(point => Number.isFinite(point.price) && Number.isFinite(point.timestamp));

const MoneylineTrendChart = (props: {
    options: ReturnType<typeof moneylineOptions>;
    history?: PolymarketSportsLivePriceHistorySeriesItem[];
}) => {
    const series = props.options
        .map((option, index) => {
            const history = resolvedPriceHistory(props.history, option.outcome, index);
            const points = pricePoints(history);
            return {
                option,
                points,
                latest: latestPrice(history),
                tone: index === 0 ? 'blue' : 'gold'
            };
        })
        .filter(item => item.points.length > 1);

    const prices = series.flatMap(item => item.points.map(point => point.price));
    const timestamps = series.flatMap(item => item.points.map(point => point.timestamp));

    if (series.length === 0 || prices.length === 0 || timestamps.length === 0) {
        return (
            <div className='sports-live-chart sports-live-chart--empty'>
                <Typography.Text type='secondary'>No history</Typography.Text>
            </div>
        );
    }

    const width = 620;
    const height = 220;
    const padding = {top: 18, right: 42, bottom: 30, left: 38};
    const minPrice = Math.min(...prices);
    const maxPrice = Math.max(...prices);
    const pricePadding = Math.max((maxPrice - minPrice) * 0.18, 0.04);
    const yMin = Math.max(0, minPrice - pricePadding);
    const yMax = Math.min(1, maxPrice + pricePadding);
    const yRange = Math.max(yMax - yMin, 0.01);
    const minTs = Math.min(...timestamps);
    const maxTs = Math.max(...timestamps);
    const xRange = Math.max(maxTs - minTs, 1);
    const chartWidth = width - padding.left - padding.right;
    const chartHeight = height - padding.top - padding.bottom;
    const yTicks = [0.7, 0.6, 0.5, 0.4, 0.3].filter(value => value >= yMin && value <= yMax);
    const visibleTicks = yTicks.length > 1 ? yTicks : [yMax, (yMax + yMin) / 2, yMin];
    const xFor = (timestamp: number) => padding.left + ((timestamp - minTs) / xRange) * chartWidth;
    const yFor = (price: number) => padding.top + (1 - (price - yMin) / yRange) * chartHeight;
    const pathFor = (points: Array<{price: number; timestamp: number}>) =>
        points
            .map((point, index) => `${index === 0 ? 'M' : 'L'} ${xFor(point.timestamp).toFixed(2)} ${yFor(point.price).toFixed(2)}`)
            .join(' ');

    return (
        <div className='sports-live-chart'>
            <svg className='sports-live-chart__svg' viewBox={`0 0 ${width} ${height}`} role='img' aria-label='Moneyline price history'>
                {visibleTicks.map(tick => {
                    const y = yFor(tick);
                    return (
                        <g className='sports-live-chart__grid' key={tick.toFixed(4)}>
                            <line x1={padding.left} x2={width - padding.right} y1={y} y2={y} />
                            <text x={width - 22} y={y + 4}>
                                {chartPercent(tick)}
                            </text>
                        </g>
                    );
                })}
                {series.map(item => (
                    <path className={`sports-live-chart__line sports-live-chart__line--${item.tone}`} d={pathFor(item.points)} key={item.option.outcome} />
                ))}
            </svg>
            <div className='sports-live-chart__labels'>
                {series.map(item => (
                    <div className={`sports-live-chart__label sports-live-chart__label--${item.tone}`} key={`${item.option.outcome}-label`}>
                        <Typography.Text className='sports-live-chart__label-name'>{item.option.outcome}</Typography.Text>
                        <Typography.Text className='sports-live-chart__label-value' strong={true}>
                            {chartPercent(item.latest)}
                        </Typography.Text>
                    </div>
                ))}
            </div>
        </div>
    );
};

const MoneylinePanel = (props: {
    market?: PolymarketSportsLiveMarketCardItem;
    teams?: PolymarketSportsLiveTeamItem[];
}) => {
    const options = moneylineOptions(props.market, props.teams);
    if (options.length === 0) {
        return <Typography.Text type='secondary'>-</Typography.Text>;
    }
    return (
        <div className='sports-live-moneyline'>
            <div className='sports-live-moneyline__rows'>
                {options.map(option => (
                    <div className='sports-live-moneyline__row' key={option.outcome}>
                        <div className='sports-live-moneyline__team'>
                            {option.logo && <img className='sports-live-moneyline__logo' src={option.logo} alt='' onError={event => (event.currentTarget.style.display = 'none')} />}
                            <Typography.Text className='sports-live-moneyline__name'>{option.outcome}</Typography.Text>
                        </div>
                        <Typography.Text className='sports-live-moneyline__price' strong={true}>
                            {option.price}
                        </Typography.Text>
                    </div>
                ))}
            </div>
        </div>
    );
};

const SportsLiveEventCard = (props: {item: PolymarketSportsLiveEventCardItem; history?: PolymarketSportsLivePriceHistorySeriesItem[]}) => {
    const moneyline = moneylineMarket(props.item);
    const options = moneylineOptions(moneyline, props.item.teams);
    const scoreMeta = [props.item.period, props.item.elapsed, props.item.gameStatus].filter(Boolean).join(' · ');

    return (
        <article className='sports-live-card'>
            <div className='sports-live-card__info'>
                <CardTitle title={props.item.title} subtitle={props.item.slug} image={props.item.image} />
                {props.item.score && (
                    <div className='sports-live-scoreboard'>
                        <Typography.Text className='sports-live-scoreboard__label'>Score</Typography.Text>
                        <Typography.Text className='sports-live-scoreboard__value' strong={true}>
                            {props.item.score}
                        </Typography.Text>
                        {scoreMeta && <Typography.Text className='sports-live-scoreboard__meta'>{scoreMeta}</Typography.Text>}
                    </div>
                )}
                <div className='sports-live-card__meta'>
                    <Space wrap={true}>
                        <Tag color='green'>Live</Tag>
                        {props.item.gameStatus && <Tag>{props.item.gameStatus}</Tag>}
                        <PolymarketEventLink item={props.item} />
                    </Space>
                </div>
                <div className='sports-live-card__stats'>
                    <div className='sports-live-card__stat'>
                        <Typography.Text className='sports-live-card__stat-label'>Volume</Typography.Text>
                        <Typography.Text className='sports-live-card__stat-value' strong={true}>
                            {fmtNumber(props.item.volume)}
                        </Typography.Text>
                    </div>
                    <div className='sports-live-card__stat'>
                        <Typography.Text className='sports-live-card__stat-label'>Liquidity</Typography.Text>
                        <Typography.Text className='sports-live-card__stat-value' strong={true}>
                            {fmtNumber(props.item.liquidity)}
                        </Typography.Text>
                    </div>
                </div>
            </div>
            <MoneylineTrendChart options={options} history={props.history} />
            <MoneylinePanel market={moneyline} teams={props.item.teams} />
        </article>
    );
};

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
            onRefresh={events.reload}>
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
