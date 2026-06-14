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

type SportsLiveDisplayOption = {
    marketKey: string;
    historyOutcome: string;
    label: string;
    price: string;
    logo?: string;
    kind: 'team' | 'draw' | 'market';
    sortOrder: number;
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

export const sportsLiveSectionKey = (item: PolymarketSportsLiveEventCardItem) => {
    const key = item.slug.trim().split('-')[0]?.trim().toLowerCase();
    return key || 'other';
};

export const isFifwcSportsLiveEvent = (item: PolymarketSportsLiveEventCardItem) => sportsLiveSectionKey(item) === 'fifwc';

const isMoneylineMarket = (market: PolymarketSportsLiveMarketCardItem) => {
    const type = market.sportsMarketType.trim().toLowerCase();
    return type === 'moneyline';
};

const moneylineMarkets = (item: PolymarketSportsLiveEventCardItem) => {
    const markets = item.markets.filter(isMoneylineMarket);
    return markets.length > 0 ? markets : item.markets.slice(0, 1);
};

export const legacyMoneylineMarket = (item: PolymarketSportsLiveEventCardItem) => moneylineMarkets(item)[0];

export const moneylineMarketKeys = (items: PolymarketSportsLiveEventCardItem[] = []) => {
    const keys = new Set<string>();
    items.forEach(item => {
        const markets = isFifwcSportsLiveEvent(item) ? moneylineMarkets(item) : [legacyMoneylineMarket(item)];
        markets.forEach(market => {
            if (market?.marketKey) {
                keys.add(market.marketKey);
            }
        });
    });
    return Array.from(keys);
};

const marketSlugTail = (eventSlug: string, marketSlug: string) => {
    const eventKey = normalizedTeamKey(eventSlug);
    const marketKey = normalizedTeamKey(marketSlug);
    if (!marketKey) {
        return '';
    }
    if (eventKey && marketKey.startsWith(`${eventKey}-`)) {
        return marketKey.slice(eventKey.length + 1);
    }
    const parts = marketKey.split('-').filter(Boolean);
    return parts[parts.length - 1] || '';
};

const marketIsDraw = (eventSlug: string, market: PolymarketSportsLiveMarketCardItem) => {
    const tail = marketSlugTail(eventSlug, market.slug);
    const question = normalizedTeamKey(market.question);
    return tail === 'draw' || question.includes('draw');
};

const questionTeamLabel = (question: string) => {
    const matched = question.trim().match(/^will\s+(.+?)\s+win\b/i);
    return matched?.[1]?.trim() || '';
};

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

const teamKeys = (team: PolymarketSportsLiveTeamItem) => [team.name, team.abbreviation, team.alias].map(value => normalizedTeamKey(value)).filter(Boolean);

const matchingMoneylineTeam = (item: PolymarketSportsLiveEventCardItem, market: PolymarketSportsLiveMarketCardItem) => {
    const tail = marketSlugTail(item.slug, market.slug);
    const questionTeam = normalizedTeamKey(questionTeamLabel(market.question));
    const question = normalizedTeamKey(market.question);
    for (let index = 0; index < item.teams.length; index += 1) {
        const team = item.teams[index];
        const keys = teamKeys(team);
        if (keys.some(key => key === tail || key === questionTeam)) {
            return {team, index};
        }
        const name = normalizedTeamKey(team.name);
        if (name && question.includes(name)) {
            return {team, index};
        }
    }
    return undefined;
};

const legacyMoneylineOptions = (item: PolymarketSportsLiveEventCardItem): SportsLiveDisplayOption[] => {
    const market = legacyMoneylineMarket(item);
    if (!market) {
        return [];
    }
    const outcomes = parseGammaList(market.outcomes);
    const prices = parseGammaList(market.outcomePrices);
    return outcomes.slice(0, 2).map((outcome, index) => ({
        marketKey: market.marketKey,
        historyOutcome: outcome,
        label: outcome,
        price: prices[index] || '-',
        logo: matchingTeam(outcome, item.teams, index)?.logo,
        kind: 'market',
        sortOrder: index
    }));
};

const fifwcMoneylineOutcomeOptions = (item: PolymarketSportsLiveEventCardItem): SportsLiveDisplayOption[] =>
    moneylineMarkets(item)
        .map((market, index) => {
            const outcomes = parseGammaList(market.outcomes);
            const prices = parseGammaList(market.outcomePrices);
            const yesIndex = outcomes.findIndex(outcome => normalizedTeamKey(outcome) === 'yes');
            const priceIndex = yesIndex >= 0 ? yesIndex : 0;
            const historyOutcome = outcomes[priceIndex] || 'Yes';
            const matched = matchingMoneylineTeam(item, market);
            const isDraw = marketIsDraw(item.slug, market);
            const fallbackLabel = questionTeamLabel(market.question) || historyOutcome || market.question || market.slug || 'Market';
            return {
                marketKey: market.marketKey,
                historyOutcome,
                label: isDraw ? 'DRAW' : matched?.team.name || fallbackLabel,
                price: prices[priceIndex] || '-',
                logo: isDraw ? undefined : matched?.team.logo,
                kind: isDraw ? ('draw' as const) : matched ? ('team' as const) : ('market' as const),
                sortOrder: isDraw ? 2 : matched ? matched.index : 100 + index
            };
        })
        .sort((left, right) => left.sortOrder - right.sortOrder);

const resolvedPriceHistory = (history: PolymarketSportsLivePriceHistorySeriesItem[] = [], option: SportsLiveDisplayOption, index: number) => {
    const marketKey = normalizedTeamKey(option.marketKey);
    const outcomeKey = normalizedTeamKey(option.historyOutcome);
    const matched = history.find(item => normalizedTeamKey(item.marketKey) === marketKey && normalizedTeamKey(item.outcome) === outcomeKey);
    if (matched) {
        return matched;
    }
    const marketMatched = history.find(item => normalizedTeamKey(item.marketKey) === marketKey);
    return marketMatched || history[index];
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
const chartTones = ['blue', 'gold', 'red', 'green'];
const displayOptionKey = (option: SportsLiveDisplayOption) => `${option.marketKey}:${option.historyOutcome}:${option.label}`;

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

export const sportsLiveCardHistory = (item: PolymarketSportsLiveEventCardItem, historyByMarketKey: Map<string, PolymarketSportsLivePriceHistorySeriesItem[]>) => {
    const markets = isFifwcSportsLiveEvent(item) ? moneylineMarkets(item) : [legacyMoneylineMarket(item)];
    return markets.flatMap(market => (market?.marketKey ? historyByMarketKey.get(market.marketKey) || [] : []));
};

const SportsLiveEventInfoSection = (props: {item: PolymarketSportsLiveEventCardItem}) => {
    const stageValue = props.item.period || props.item.gameStatus || props.item.elapsed;
    const stageMeta = [props.item.elapsed, props.item.gameStatus].filter(value => value && value !== stageValue).join(' · ');
    const title = (
        <span className='sports-live-card__title-line'>
            <span className='sports-live-card__title-text'>{props.item.title}</span>
            <PolymarketEventLink item={props.item} />
        </span>
    );

    return (
        <div className='sports-live-card__info'>
            <CardTitle title={title} subtitle={props.item.slug} image={props.item.image} />
            {stageValue && (
                <div className='sports-live-stage'>
                    <Typography.Text className='sports-live-stage__label'>Live Stage</Typography.Text>
                    <Typography.Text className='sports-live-stage__value' strong={true}>
                        {stageValue}
                    </Typography.Text>
                    {stageMeta && <Typography.Text className='sports-live-stage__meta'>{stageMeta}</Typography.Text>}
                </div>
            )}
            {props.item.score && (
                <div className='sports-live-scoreboard'>
                    <Typography.Text className='sports-live-scoreboard__label'>Score</Typography.Text>
                    <Typography.Text className='sports-live-scoreboard__value' strong={true}>
                        {props.item.score}
                    </Typography.Text>
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

const SportsLiveTrendSection = (props: {options: SportsLiveDisplayOption[]; history?: PolymarketSportsLivePriceHistorySeriesItem[]}) => {
    const series = props.options
        .map((option, index) => {
            const history = resolvedPriceHistory(props.history, option, index);
            const points = pricePoints(history);
            return {
                option,
                points,
                latest: latestPrice(history),
                tone: chartTones[index % chartTones.length]
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
        points.map((point, index) => `${index === 0 ? 'M' : 'L'} ${xFor(point.timestamp).toFixed(2)} ${yFor(point.price).toFixed(2)}`).join(' ');
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
    if (endpointItems.length > 1) {
        const minGap = 38;
        const minY = padding.top + 16;
        const maxY = height - padding.bottom - 30;
        const sorted = [...endpointItems].sort((left, right) => left.labelY - right.labelY);
        sorted[0].labelY = Math.max(minY, sorted[0].labelY);
        for (let index = 1; index < sorted.length; index += 1) {
            sorted[index].labelY = Math.max(sorted[index].labelY, sorted[index - 1].labelY + minGap);
        }
        const overflow = sorted[sorted.length - 1].labelY - maxY;
        if (overflow > 0) {
            sorted.forEach(item => {
                item.labelY = Math.max(minY, item.labelY - overflow);
            });
        }
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
                    <g className={`sports-live-chart__series sports-live-chart__series--${item.tone}`} key={displayOptionKey(item.option)}>
                        <path className='sports-live-chart__line' d={pathFor(item.points)} />
                        <circle className='sports-live-chart__dot' cx={xFor(item.lastPoint.timestamp)} cy={yFor(item.lastPoint.price)} r='5' />
                        <text className='sports-live-chart__endpoint-name' x={endpointLabelX} y={item.labelY - 4}>
                            {chartOutcomeLabel(item.option.label)}
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

const SportsLiveMoneylineSection = (props: {options: SportsLiveDisplayOption[]}) => {
    if (props.options.length === 0) {
        return <Typography.Text type='secondary'>-</Typography.Text>;
    }
    return (
        <div className='sports-live-moneyline'>
            <div className='sports-live-moneyline__rows'>
                {props.options.map(option => (
                    <div className='sports-live-moneyline__row' key={displayOptionKey(option)}>
                        <div className='sports-live-moneyline__team'>
                            {option.logo && (
                                <img className='sports-live-moneyline__logo' src={option.logo} alt='' onError={event => (event.currentTarget.style.display = 'none')} />
                            )}
                            {!option.logo && option.kind === 'draw' && <span className='sports-live-moneyline__badge'>D</span>}
                            <Typography.Text className='sports-live-moneyline__name'>{option.label}</Typography.Text>
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

const SportsLiveEventCardFrame = (props: {item: PolymarketSportsLiveEventCardItem; options: SportsLiveDisplayOption[]; history?: PolymarketSportsLivePriceHistorySeriesItem[]}) => (
    <article className='sports-live-card'>
        <SportsLiveEventInfoSection item={props.item} />
        <SportsLiveTrendSection options={props.options} history={props.history} />
        <SportsLiveMoneylineSection options={props.options} />
    </article>
);

export const FifwcSportsLiveEventCard = (props: {item: PolymarketSportsLiveEventCardItem; history?: PolymarketSportsLivePriceHistorySeriesItem[]}) => (
    <SportsLiveEventCardFrame item={props.item} options={fifwcMoneylineOutcomeOptions(props.item)} history={props.history} />
);

export const LegacySportsLiveEventCard = (props: {item: PolymarketSportsLiveEventCardItem; history?: PolymarketSportsLivePriceHistorySeriesItem[]}) => (
    <SportsLiveEventCardFrame item={props.item} options={legacyMoneylineOptions(props.item)} history={props.history} />
);

export const SportsLiveEventCard = (props: {item: PolymarketSportsLiveEventCardItem; history?: PolymarketSportsLivePriceHistorySeriesItem[]}) => {
    if (isFifwcSportsLiveEvent(props.item)) {
        return <FifwcSportsLiveEventCard item={props.item} history={props.history} />;
    }
    return <LegacySportsLiveEventCard item={props.item} history={props.history} />;
};
