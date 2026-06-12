import {LinkOutlined} from '@ant-design/icons';
import {Button, Tooltip, Typography} from 'antd';
import {CardTitle} from '../components';
import {
    PolymarketSportsLiveEventCardItem,
    PolymarketSportsLiveMarketCardItem,
    PolymarketSportsLivePriceHistorySeriesItem,
    PolymarketSportsLiveTeamItem
} from '../../shared/services/polymarket-service';
import {fmtNumber} from './shared';

type MoneylineOption = {
    outcome: string;
    price: string;
    logo?: string;
};

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

const moneylineOptions = (market?: PolymarketSportsLiveMarketCardItem, teams: PolymarketSportsLiveTeamItem[] = []): MoneylineOption[] => {
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

export const moneylineMarket = (item: PolymarketSportsLiveEventCardItem) => item.markets.find(isMoneylineMarket) || item.markets[0];

export const moneylineMarketKeys = (items: PolymarketSportsLiveEventCardItem[] = []) => {
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

const chartOutcomeLabel = (value: string) => (value.length > 16 ? `${value.slice(0, 15)}...` : value);

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

const SportsLiveEventInfoSection = (props: {item: PolymarketSportsLiveEventCardItem}) => {
    const scoreMeta = [props.item.period, props.item.elapsed, props.item.gameStatus].filter(Boolean).join(' · ');
    const title = (
        <span className='sports-live-card__title-line'>
            <span className='sports-live-card__title-text'>{props.item.title}</span>
            <PolymarketEventLink item={props.item} />
        </span>
    );

    return (
        <div className='sports-live-card__info'>
            <CardTitle title={title} subtitle={props.item.slug} image={props.item.image} />
            {props.item.score && (
                <div className='sports-live-scoreboard'>
                    <Typography.Text className='sports-live-scoreboard__label'>Score</Typography.Text>
                    <Typography.Text className='sports-live-scoreboard__value' strong={true}>
                        {props.item.score}
                    </Typography.Text>
                    {scoreMeta && <Typography.Text className='sports-live-scoreboard__meta'>{scoreMeta}</Typography.Text>}
                </div>
            )}
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
    );
};

const SportsLiveTrendSection = (props: {
    options: MoneylineOption[];
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

    const timestamps = series.flatMap(item => item.points.map(point => point.timestamp));

    if (series.length === 0 || timestamps.length === 0) {
        return (
            <div className='sports-live-chart sports-live-chart--empty'>
                <Typography.Text type='secondary'>No history</Typography.Text>
            </div>
        );
    }

    const width = 720;
    const height = 220;
    const padding = {top: 18, right: 16, bottom: 28, left: 16};
    const axisLabelX = width - 8;
    const axisGuideEnd = width - 30;
    const endpointLabelX = width - 168;
    const plotRight = endpointLabelX - 18;
    const yTicks = [1, 0.75, 0.5, 0.25, 0];
    const minTs = Math.min(...timestamps);
    const maxTs = Math.max(...timestamps);
    const xRange = Math.max(maxTs - minTs, 1);
    const chartWidth = plotRight - padding.left;
    const chartHeight = height - padding.top - padding.bottom;
    const xFor = (timestamp: number) => padding.left + ((timestamp - minTs) / xRange) * chartWidth;
    const yFor = (price: number) => padding.top + (1 - Math.min(1, Math.max(0, price))) * chartHeight;
    const labelYFor = (price: number) => Math.min(height - padding.bottom - 30, Math.max(padding.top + 16, yFor(price)));
    const pathFor = (points: Array<{price: number; timestamp: number}>) =>
        points
            .map((point, index) => `${index === 0 ? 'M' : 'L'} ${xFor(point.timestamp).toFixed(2)} ${yFor(point.price).toFixed(2)}`)
            .join(' ');
    const endpointItems = series.map(item => {
        const lastPoint = item.points[item.points.length - 1];
        const latest = item.latest ?? lastPoint.price;
        return {
            ...item,
            lastPoint,
            latest,
            labelY: labelYFor(latest)
        };
    });
    if (endpointItems.length === 2 && Math.abs(endpointItems[0].labelY - endpointItems[1].labelY) < 48) {
        endpointItems[0].labelY = Math.max(padding.top + 16, endpointItems[0].labelY - 24);
        endpointItems[1].labelY = Math.min(height - padding.bottom - 30, endpointItems[1].labelY + 24);
    }

    return (
        <div className='sports-live-chart'>
            <svg className='sports-live-chart__svg' viewBox={`0 0 ${width} ${height}`} role='img' aria-label='Moneyline price history'>
                {yTicks.map(tick => {
                    const y = yFor(tick);
                    return (
                        <g className='sports-live-chart__grid' key={tick.toFixed(4)}>
                            <line x1={padding.left} x2={axisGuideEnd} y1={y} y2={y} />
                            <text x={axisLabelX} y={y + 4}>
                                {chartPercent(tick)}
                            </text>
                        </g>
                    );
                })}
                {endpointItems.map(item => (
                    <g className={`sports-live-chart__series sports-live-chart__series--${item.tone}`} key={item.option.outcome}>
                        <path className='sports-live-chart__line' d={pathFor(item.points)} />
                        <circle className='sports-live-chart__dot' cx={xFor(item.lastPoint.timestamp)} cy={yFor(item.lastPoint.price)} r='5' />
                        <text className='sports-live-chart__endpoint-name' x={endpointLabelX} y={item.labelY - 4}>
                            {chartOutcomeLabel(item.option.outcome)}
                        </text>
                        <text className='sports-live-chart__endpoint-value' x={endpointLabelX} y={item.labelY + 32}>
                            {chartPercent(item.latest)}
                        </text>
                    </g>
                ))}
            </svg>
        </div>
    );
};

const SportsLiveMoneylineSection = (props: {options: MoneylineOption[]}) => {
    if (props.options.length === 0) {
        return <Typography.Text type='secondary'>-</Typography.Text>;
    }
    return (
        <div className='sports-live-moneyline'>
            <div className='sports-live-moneyline__rows'>
                {props.options.map(option => (
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

export const SportsLiveEventCard = (props: {item: PolymarketSportsLiveEventCardItem; history?: PolymarketSportsLivePriceHistorySeriesItem[]}) => {
    const moneyline = moneylineMarket(props.item);
    const options = moneylineOptions(moneyline, props.item.teams);

    return (
        <article className='sports-live-card'>
            <SportsLiveEventInfoSection item={props.item} />
            <SportsLiveTrendSection options={options} history={props.history} />
            <SportsLiveMoneylineSection options={options} />
        </article>
    );
};
