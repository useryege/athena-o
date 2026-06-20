import {LinkOutlined} from '@ant-design/icons';
import {Button, Tooltip, Typography} from 'antd';
import * as React from 'react';
import {CardTitle} from '../components';
import {
    PolymarketSportsLiveEventCardItem,
    PolymarketSportsHistoryEventCardItem,
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
const chartSelectionLabel = (value: string) => (value.length > 20 ? `${value.slice(0, 19)}...` : value);
const chartSelectionPercent = (value: number) => `${(value * 100).toFixed(1)}%`;
const chartSelectionTime = (timestamp: number) =>
    new Intl.DateTimeFormat('en-US', {
        month: 'short',
        day: 'numeric',
        hour: 'numeric',
        minute: '2-digit'
    }).format(new Date(timestamp * 1000));
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

const SportsLiveTrendSection = (props: {
    options: SportsLiveDisplayOption[];
    history?: PolymarketSportsLivePriceHistorySeriesItem[];
    scratchMode?: boolean;
    scratchResetVersion?: number;
}) => {
    const [selectedTimestamp, setSelectedTimestamp] = React.useState<number>();
    const [scratchRevealX, setScratchRevealX] = React.useState(0);
    const chartId = React.useId().replace(/:/g, '');
    const scratchClipId = `sports-live-chart-scratch-${chartId}`;
    const scratchActive = props.scratchMode === true;
    const series = props.options
        .map((option, index) => {
            const history = resolvedPriceHistory(props.history, option, index);
            const points = pricePoints(history);
            return {
                key: displayOptionKey(option),
                option,
                points,
                latest: latestPrice(history),
                tone: chartTones[index % chartTones.length]
            };
        })
        .filter(item => item.points.length > 1);

    const timestamps = series.flatMap(item => item.points.map(point => point.timestamp));

    React.useEffect(() => {
        if (scratchActive) {
            setScratchRevealX(0);
            setSelectedTimestamp(undefined);
        }
    }, [scratchActive, props.scratchResetVersion]);

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
    const chartPointerPosition = (element: SVGSVGElement, clientX: number, clientY: number) => {
        const bounds = element.getBoundingClientRect();
        if (bounds.width <= 0 || bounds.height <= 0) {
            return undefined;
        }
        const pointerX = ((clientX - bounds.left) / bounds.width) * width;
        const pointerY = ((clientY - bounds.top) / bounds.height) * height;
        return {pointerX, pointerY};
    };
    const revealScratchAt = (element: SVGSVGElement, clientX: number, clientY: number) => {
        const position = chartPointerPosition(element, clientX, clientY);
        if (!position || position.pointerY < 0 || position.pointerY > height) {
            return;
        }
        const nextX = Math.min(width, Math.max(0, position.pointerX));
        setScratchRevealX(current => Math.max(current, nextX));
    };
    const selectNearestTime = (element: SVGSVGElement, clientX: number, clientY: number) => {
        const position = chartPointerPosition(element, clientX, clientY);
        if (!position) {
            return;
        }
        const {pointerX, pointerY} = position;
        if (pointerX < padding.left || pointerX > plotRight || pointerY < padding.top || pointerY > height - padding.bottom) {
            setSelectedTimestamp(undefined);
            return;
        }

        const pointerTimestamp = minTs + ((pointerX - padding.left) / chartWidth) * xRange;
        let nearestTimestamp = timestamps[0];
        timestamps.forEach(timestamp => {
            if (Math.abs(timestamp - pointerTimestamp) < Math.abs(nearestTimestamp - pointerTimestamp)) {
                nearestTimestamp = timestamp;
            }
        });
        setSelectedTimestamp(current => (current === nearestTimestamp ? current : nearestTimestamp));
    };
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
    const selectedX = selectedTimestamp === undefined ? 0 : xFor(selectedTimestamp);
    const scratchCanLabelLeft = scratchActive && selectedX > padding.left + 180;
    const labelsOnLeft = scratchCanLabelLeft || (!scratchActive && selectedX > plotRight - 184);
    const selectionLabelX = labelsOnLeft ? selectedX - 16 : selectedX + 16;
    const selectionTextAnchor = labelsOnLeft ? ('end' as const) : ('start' as const);
    const selectionItems =
        selectedTimestamp === undefined
            ? []
            : series.map(item => {
                  const point = item.points.reduce((nearest, candidate) =>
                      Math.abs(candidate.timestamp - selectedTimestamp) < Math.abs(nearest.timestamp - selectedTimestamp) ? candidate : nearest
                  );
                  const pointY = yFor(point.price);
                  return {
                      ...item,
                      point,
                      pointY,
                      labelY: pointY - 5
                  };
              });
    if (selectionItems.length > 0) {
        const minY = padding.top + 24;
        const maxY = height - padding.bottom - 34;
        const minGap = selectionItems.length > 1 ? Math.min(46, (maxY - minY) / (selectionItems.length - 1)) : 0;
        const sorted = [...selectionItems].sort((left, right) => left.labelY - right.labelY);
        sorted[0].labelY = Math.max(minY, sorted[0].labelY);
        for (let index = 1; index < sorted.length; index += 1) {
            sorted[index].labelY = Math.max(sorted[index].labelY, sorted[index - 1].labelY + minGap);
        }
        const overflow = sorted[sorted.length - 1].labelY - maxY;
        if (overflow > 0) {
            sorted.forEach(item => {
                item.labelY -= overflow;
            });
        }
        const underflow = minY - sorted[0].labelY;
        if (underflow > 0) {
            sorted.forEach(item => {
                item.labelY += underflow;
            });
        }
    }
    const selectionTimeX = Math.min(width - 76, Math.max(76, selectedX));
    const scratchRevealWidth = scratchActive ? Math.min(width, Math.max(0, scratchRevealX)) : width;
    const scratchCoverWidth = Math.max(0, width - scratchRevealWidth);
    const chartClassName = `sports-live-chart ${scratchActive ? 'sports-live-chart--scratch' : ''}`.trim();
    const revealAndSelect = (element: SVGSVGElement, clientX: number, clientY: number) => {
        if (scratchActive) {
            revealScratchAt(element, clientX, clientY);
        }
        selectNearestTime(element, clientX, clientY);
    };

    return (
        <div className={chartClassName}>
            <svg
                className='sports-live-chart__svg'
                viewBox={`0 0 ${width} ${height}`}
                role='img'
                aria-label='Moneyline price history'
                onPointerDown={event => {
                    if (scratchActive) {
                        event.currentTarget.setPointerCapture(event.pointerId);
                        revealAndSelect(event.currentTarget, event.clientX, event.clientY);
                    }
                }}
                onPointerMove={event => {
                    if (scratchActive || event.pointerType !== 'touch') {
                        revealAndSelect(event.currentTarget, event.clientX, event.clientY);
                    }
                }}
                onPointerLeave={event => {
                    if (event.pointerType !== 'touch') {
                        setSelectedTimestamp(undefined);
                    }
                }}
                onPointerUp={event => {
                    if (scratchActive || event.pointerType === 'touch') {
                        revealAndSelect(event.currentTarget, event.clientX, event.clientY);
                    }
                    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
                        event.currentTarget.releasePointerCapture(event.pointerId);
                    }
                }}>
                <defs>
                    <clipPath id={scratchClipId}>
                        <rect x='0' y='0' width={scratchRevealWidth} height={height} />
                    </clipPath>
                </defs>
                {scratchActive && scratchCoverWidth > 0 && (
                    <rect className='sports-live-chart__scratch-cover' x={scratchRevealWidth} y='0' width={scratchCoverWidth} height={height} />
                )}
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
                <g clipPath={scratchActive ? `url(#${scratchClipId})` : undefined}>
                    {endpointItems.map(item => (
                        <g className={`sports-live-chart__series sports-live-chart__series--${item.tone}`} key={displayOptionKey(item.option)}>
                            <path className='sports-live-chart__line' d={pathFor(item.points)} />
                            <circle className='sports-live-chart__dot' cx={xFor(item.lastPoint.timestamp)} cy={yFor(item.lastPoint.price)} r='5' />
                            {selectedTimestamp === undefined && (
                                <>
                                    <text className='sports-live-chart__endpoint-name' x={endpointLabelX} y={item.labelY - 4}>
                                        {chartOutcomeLabel(item.option.label)}
                                    </text>
                                    <text className='sports-live-chart__endpoint-value' x={endpointLabelX} y={item.labelY + 32}>
                                        {chartPercent(item.latest)}
                                    </text>
                                </>
                            )}
                        </g>
                    ))}
                    {selectedTimestamp !== undefined && (
                        <g className='sports-live-chart__selection'>
                            <line className='sports-live-chart__selection-guide' x1={selectedX} x2={selectedX} y1={padding.top} y2={height - padding.bottom} />
                            <text className='sports-live-chart__selection-time' x={selectionTimeX} y='14'>
                                {chartSelectionTime(selectedTimestamp)}
                            </text>
                            {selectionItems.map(item => (
                                <g className={`sports-live-chart__selection-item sports-live-chart__series--${item.tone}`} key={item.key}>
                                    <line
                                        className='sports-live-chart__selection-connector'
                                        x1={selectedX + (labelsOnLeft ? -7 : 7)}
                                        x2={selectionLabelX + (labelsOnLeft ? 5 : -5)}
                                        y1={item.pointY}
                                        y2={item.labelY + 7}
                                    />
                                    <circle className='sports-live-chart__selection-halo' cx={selectedX} cy={item.pointY} r='9' />
                                    <circle className='sports-live-chart__selection-dot' cx={selectedX} cy={item.pointY} r='4.5' />
                                    <text className='sports-live-chart__selection-name' textAnchor={selectionTextAnchor} x={selectionLabelX} y={item.labelY}>
                                        {chartSelectionLabel(item.option.label)}
                                    </text>
                                    <text className='sports-live-chart__selection-price' textAnchor={selectionTextAnchor} x={selectionLabelX} y={item.labelY + 27}>
                                        {chartSelectionPercent(item.point.price)}
                                    </text>
                                </g>
                            ))}
                        </g>
                    )}
                </g>
                {scratchActive && scratchRevealWidth > 0 && scratchRevealWidth < width && (
                    <line className='sports-live-chart__scratch-edge' x1={scratchRevealWidth} x2={scratchRevealWidth} y1='0' y2={height} />
                )}
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

const SportsLiveEventCardFrame = (props: {
    item: PolymarketSportsLiveEventCardItem;
    options: SportsLiveDisplayOption[];
    history?: PolymarketSportsLivePriceHistorySeriesItem[];
    scratchMode?: boolean;
    scratchResetVersion?: number;
    info?: React.ReactNode;
    className?: string;
}) => (
    <article className={`sports-live-card ${props.className || ''}`.trim()}>
        <div className='sports-live-card__info-column'>
            {props.info || <SportsLiveEventInfoSection item={props.item} />}
            <SportsLiveMoneylineSection options={props.options} />
        </div>
        <SportsLiveTrendSection options={props.options} history={props.history} scratchMode={props.scratchMode} scratchResetVersion={props.scratchResetVersion} />
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

const sportsHistoryTime = (value?: string) => {
    if (!value) {
        return '-';
    }
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
};

const SportsHistoryEventInfoSection = (props: {item: PolymarketSportsHistoryEventCardItem}) => {
    const title = (
        <span className='sports-live-card__title-line'>
            <span className='sports-live-card__title-text'>{props.item.title}</span>
            <PolymarketEventLink item={props.item} />
        </span>
    );
    return (
        <div className='sports-live-card__info'>
            <CardTitle title={title} subtitle={props.item.slug} image={props.item.image} />
            <div className='sports-history-times'>
                <div className='sports-history-time'>
                    <Typography.Text className='sports-live-card__stat-label'>Started</Typography.Text>
                    <Typography.Text className='sports-history-time__value' strong={true}>
                        {sportsHistoryTime(props.item.startTime)}
                    </Typography.Text>
                </div>
                <div className='sports-history-time'>
                    <Typography.Text className='sports-live-card__stat-label'>Finished</Typography.Text>
                    <Typography.Text className='sports-history-time__value' strong={true}>
                        {sportsHistoryTime(props.item.finishedAt)}
                    </Typography.Text>
                </div>
            </div>
            <div className='sports-live-stage sports-history-status'>
                <Typography.Text className='sports-live-stage__label'>Final Status</Typography.Text>
                <Typography.Text className='sports-live-stage__value' strong={true}>
                    {props.item.gameStatus || props.item.period || 'Finished'}
                </Typography.Text>
            </div>
            {props.item.score && (
                <div className='sports-live-scoreboard'>
                    <Typography.Text className='sports-live-scoreboard__label'>Final Score</Typography.Text>
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

export const SportsHistoryEventCard = (props: {
    item: PolymarketSportsHistoryEventCardItem;
    history?: PolymarketSportsLivePriceHistorySeriesItem[];
    scratchMode?: boolean;
    scratchResetVersion?: number;
}) => (
    <SportsLiveEventCardFrame
        className='sports-history-card'
        item={props.item}
        options={legacyMoneylineOptions(props.item)}
        history={props.history}
        scratchMode={props.scratchMode}
        scratchResetVersion={props.scratchResetVersion}
        info={<SportsHistoryEventInfoSection item={props.item} />}
    />
);
