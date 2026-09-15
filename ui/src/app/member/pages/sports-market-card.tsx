import {LinkOutlined} from '@ant-design/icons';
import {Button, Tooltip, Typography} from 'antd';
import * as React from 'react';
import {TruncatedText} from '../../components';
import {formatBeijingDateTime, formatBeijingUnixSeconds} from '../../shared/format';
import {SportsLiveEventCardItem, SportsHistoryEventCardItem, SportsLiveMarketCardItem, SportsPriceHistorySeriesItem, SportsLiveTeamItem} from '../../shared/services/sports-models';
import {fmtNumber} from '../../shared/pages/shared';

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

export const sportsLiveSectionKey = (item: SportsLiveEventCardItem) => {
    const key = item.slug.trim().split('-')[0]?.trim().toLowerCase();
    return key || 'other';
};

export const isFifwcSportsLiveEvent = (item: SportsLiveEventCardItem) => sportsLiveSectionKey(item) === 'fifwc';

const isMoneylineMarket = (market: SportsLiveMarketCardItem) => {
    const type = market.sportsMarketType.trim().toLowerCase();
    return type === 'moneyline';
};

const moneylineMarkets = (item: SportsLiveEventCardItem) => {
    const markets = item.markets.filter(isMoneylineMarket);
    return markets.length > 0 ? markets : item.markets.slice(0, 1);
};

export const legacyMoneylineMarket = (item: SportsLiveEventCardItem) => moneylineMarkets(item)[0];

export const moneylineMarketKeys = (items: SportsLiveEventCardItem[] = []) => {
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

const marketIsDraw = (eventSlug: string, market: SportsLiveMarketCardItem) => {
    const tail = marketSlugTail(eventSlug, market.slug);
    const question = normalizedTeamKey(market.question);
    return tail === 'draw' || question.includes('draw');
};

const questionTeamLabel = (question: string) => {
    const matched = question.trim().match(/^will\s+(.+?)\s+win\b/i);
    return matched?.[1]?.trim() || '';
};

const matchingTeam = (outcome: string, teams: SportsLiveTeamItem[], index: number) => {
    const key = normalizedTeamKey(outcome);
    if (key) {
        const exact = teams.find(team => [team.name, team.abbreviation, team.alias].some(value => normalizedTeamKey(value) === key));
        if (exact) {
            return exact;
        }
    }
    return teams[index];
};

const teamKeys = (team: SportsLiveTeamItem) => [team.name, team.abbreviation, team.alias].map(value => normalizedTeamKey(value)).filter(Boolean);

const matchingMoneylineTeam = (item: SportsLiveEventCardItem, market: SportsLiveMarketCardItem) => {
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

const legacyMoneylineOptions = (item: SportsLiveEventCardItem): SportsLiveDisplayOption[] => {
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

const fifwcMoneylineOutcomeOptions = (item: SportsLiveEventCardItem): SportsLiveDisplayOption[] =>
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

const resolvedPriceHistory = (history: SportsPriceHistorySeriesItem[] = [], option: SportsLiveDisplayOption) =>
    history.find(item => item.marketKey === option.marketKey && normalizedTeamKey(item.outcome) === normalizedTeamKey(option.historyOutcome));

const polymarketEventURL = (item: SportsLiveEventCardItem) => {
    const slug = item.slug.trim();
    return slug ? `https://polymarket.com/event/${encodeURIComponent(slug)}` : '';
};

const PolymarketEventLink = (props: {item: SportsLiveEventCardItem}) => {
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

const axisTime = (timestamp: number) =>
    new Intl.DateTimeFormat('en-GB', {timeZone: 'Asia/Shanghai', hour: '2-digit', minute: '2-digit', hour12: false}).format(new Date(timestamp * 1000));
const axisDate = (timestamp: number) =>
    new Intl.DateTimeFormat('en-CA', {timeZone: 'Asia/Shanghai', year: 'numeric', month: '2-digit', day: '2-digit'}).format(new Date(timestamp * 1000));

const displayOptionKey = (option: SportsLiveDisplayOption) => `${option.marketKey}:${option.historyOutcome}:${option.label}`;
const pricePoints = (series?: SportsPriceHistorySeriesItem) =>
    (series?.prices || [])
        .flatMap((price, index) => {
            const timestamp = series?.timestamps[index];
            return Number.isFinite(price) && timestamp !== undefined && Number.isFinite(timestamp) ? [{price, timestamp}] : [];
        })
        .sort((a, b) => a.timestamp - b.timestamp);

export const sportsLiveCardHistory = (item: SportsLiveEventCardItem, historyByMarketKey: Map<string, SportsPriceHistorySeriesItem[]>) => {
    const markets = isFifwcSportsLiveEvent(item) ? moneylineMarkets(item) : [legacyMoneylineMarket(item)];
    return markets.flatMap(market => (market?.marketKey ? historyByMarketKey.get(market.marketKey) || [] : []));
};

const SportsLiveEventInfoSection = (props: {item: SportsLiveEventCardItem}) => {
    const stageValue = props.item.period || props.item.gameStatus || props.item.elapsed;
    const stageMeta = [props.item.elapsed, props.item.gameStatus].filter(value => value && value !== stageValue).join(' · ');
    return (
        <div className='sports-live-card__info'>
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
    history?: SportsPriceHistorySeriesItem[];
    historyState?: 'loading' | 'failed';
    scratchMode?: boolean;
    scratchResetVersion?: number;
}) => {
    const [selected, setSelected] = React.useState<number>();
    const [revealed, setRevealed] = React.useState(0);
    const clipId = React.useId().replace(/:/g, '');
    React.useEffect(() => {
        setSelected(undefined);
        setRevealed(0);
    }, [props.scratchMode, props.scratchResetVersion]);
    const series = props.options.map((option, index) => ({option, index, points: pricePoints(resolvedPriceHistory(props.history, option))})).filter(item => item.points.length > 1);
    if (!series.length)
        return (
            <div className='sports-live-chart sports-live-chart--empty'>
                <Typography.Text type='secondary'>
                    {props.historyState === 'failed' ? 'History unavailable' : props.historyState === 'loading' ? 'Loading history…' : 'No history'}
                </Typography.Text>
            </div>
        );
    const times = series.flatMap(item => item.points.map(point => point.timestamp));
    const min = Math.min(...times),
        max = Math.max(...times);
    const width = 640,
        height = 220;
    const x = (timestamp: number) => ((timestamp - min) / Math.max(max - min, 1)) * width;
    const y = (price: number) => (1 - Math.min(1, Math.max(0, price))) * height;
    const current = selected === undefined ? max : Math.max(min, Math.min(max, selected));
    const choose = (timestamp: number) => {
        setSelected(timestamp);
        if (props.scratchMode) setRevealed(previous => Math.max(previous, (timestamp - min) / Math.max(max - min, 1)));
    };
    const scratch = props.scratchMode === true;
    return (
        <div className={`sports-live-chart ${scratch ? 'sports-live-chart--scratch' : ''}`}>
            <div className='sports-chart-heading'>
                <strong>Moneyline price history</strong>
                <span>Price · 0–100%</span>
            </div>
            <div className='sports-chart-legend'>
                {series.map(({option, index}) => (
                    <span key={displayOptionKey(option)} className={`sports-chart-tone-${index}`}>
                        <i aria-hidden='true' />
                        {option.label}
                    </span>
                ))}
            </div>
            <div className='sports-chart-plot'>
                <div className='sports-chart-axis'>
                    <span>100%</span>
                    <span>50%</span>
                    <span>0%</span>
                </div>
                <svg
                    className='sports-live-chart__svg'
                    viewBox={`0 0 ${width} ${height}`}
                    preserveAspectRatio='none'
                    role='img'
                    aria-label='Moneyline price history'
                    onPointerMove={event => {
                        if (scratch && event.buttons !== 1) return;
                        const bounds = event.currentTarget.getBoundingClientRect();
                        choose(min + Math.max(0, Math.min(1, (event.clientX - bounds.left) / bounds.width)) * (max - min));
                    }}
                    onPointerDown={event => {
                        const bounds = event.currentTarget.getBoundingClientRect();
                        choose(min + Math.max(0, Math.min(1, (event.clientX - bounds.left) / bounds.width)) * (max - min));
                    }}>
                    <defs>
                        <clipPath id={clipId}>
                            <rect width={scratch ? revealed * width : width} height={height} />
                        </clipPath>
                    </defs>
                    {[0, 0.5, 1].map(tick => (
                        <line key={tick} className='sports-chart-grid' x1={0} x2={width} y1={y(tick)} y2={y(tick)} />
                    ))}
                    <g clipPath={`url(#${clipId})`}>
                        {series.map(({option, index, points}) => (
                            <path
                                key={displayOptionKey(option)}
                                className={`sports-live-chart__line sports-chart-tone-${index}`}
                                d={points.map((point, i) => `${i ? 'L' : 'M'} ${x(point.timestamp)} ${y(point.price)}`).join(' ')}
                            />
                        ))}
                    </g>
                </svg>
            </div>
            <div className='sports-chart-times'>
                <span>
                    {axisDate(min) !== axisDate(max) ? `${axisDate(min)} ` : ''}
                    {axisTime(min)}
                </span>
                <span>
                    {axisDate(min) !== axisDate(max) ? `${axisDate(max)} ` : ''}
                    {axisTime(max)} · UTC+8
                </span>
            </div>
            <input type='range' aria-label='History time' min={min} max={max} step={1} value={current} onChange={event => choose(Number(event.target.value))} />
            <div className='sports-chart-selection'>
                <span>Selected {formatBeijingUnixSeconds(current)} · UTC+8</span>
                {scratch && !revealed ? (
                    <span>Move the time slider or drag the chart to reveal history.</span>
                ) : (
                    <div>
                        {series.map(({option, points}) => {
                            const nearest = points.reduce((a, b) => (Math.abs(a.timestamp - current) <= Math.abs(b.timestamp - current) ? a : b));
                            return (
                                <span key={displayOptionKey(option)}>
                                    {option.label} <b className='athena-number'>{(nearest.price * 100).toFixed(1)}%</b>
                                </span>
                            );
                        })}
                    </div>
                )}
            </div>
        </div>
    );
};

const SportsLiveMoneylineSection = (props: {options: SportsLiveDisplayOption[]}) => {
    if (props.options.length === 0) {
        return <Typography.Text type='secondary'>-</Typography.Text>;
    }
    return (
        <div className='sports-live-moneyline'>
            <span className='sports-live-card__stat-label'>Moneyline snapshot · Price</span>
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
    item: SportsLiveEventCardItem;
    options: SportsLiveDisplayOption[];
    history?: SportsPriceHistorySeriesItem[];
    historyState?: 'loading' | 'failed';
    scratchMode?: boolean;
    scratchResetVersion?: number;
    info?: React.ReactNode;
    className?: string;
}) => (
    <article className={`sports-live-card ${props.className || ''}`.trim()}>
        <header className='sports-event-heading'>
            <h3>{props.item.title}</h3>
            <PolymarketEventLink item={props.item} />
        </header>
        <div className='sports-live-card__info-column'>
            {props.info || <SportsLiveEventInfoSection item={props.item} />}
            <SportsLiveMoneylineSection options={props.options} />
        </div>
        <SportsLiveTrendSection
            options={props.options}
            history={props.history}
            historyState={props.historyState}
            scratchMode={props.scratchMode}
            scratchResetVersion={props.scratchResetVersion}
        />
        <details className='sports-event-sources'>
            <summary>Event identifiers &amp; source</summary>
            <dl className='market-fact-grid'>
                <div>
                    <dt>Event key</dt>
                    <dd>
                        <TruncatedText value={props.item.eventKey} copyable />
                    </dd>
                </div>
                <div>
                    <dt>Event ID</dt>
                    <dd>
                        <TruncatedText value={props.item.eventId} copyable />
                    </dd>
                </div>
                <div>
                    <dt>Event slug</dt>
                    <dd>
                        <TruncatedText value={props.item.slug} copyable />
                    </dd>
                </div>
                {props.item.markets.map(market => (
                    <React.Fragment key={market.marketKey}>
                        <div>
                            <dt>Market key</dt>
                            <dd>
                                <TruncatedText value={market.marketKey} copyable />
                            </dd>
                        </div>
                        <div>
                            <dt>Condition ID</dt>
                            <dd>
                                <TruncatedText value={market.conditionId} copyable />
                            </dd>
                        </div>
                    </React.Fragment>
                ))}
            </dl>
        </details>
    </article>
);

export const FifwcSportsLiveEventCard = (props: {item: SportsLiveEventCardItem; history?: SportsPriceHistorySeriesItem[]; historyState?: 'loading' | 'failed'}) => (
    <SportsLiveEventCardFrame item={props.item} options={fifwcMoneylineOutcomeOptions(props.item)} history={props.history} historyState={props.historyState} />
);

export const LegacySportsLiveEventCard = (props: {item: SportsLiveEventCardItem; history?: SportsPriceHistorySeriesItem[]; historyState?: 'loading' | 'failed'}) => (
    <SportsLiveEventCardFrame item={props.item} options={legacyMoneylineOptions(props.item)} history={props.history} historyState={props.historyState} />
);

export const SportsLiveEventCard = (props: {item: SportsLiveEventCardItem; history?: SportsPriceHistorySeriesItem[]; historyState?: 'loading' | 'failed'}) => {
    if (isFifwcSportsLiveEvent(props.item)) {
        return <FifwcSportsLiveEventCard item={props.item} history={props.history} historyState={props.historyState} />;
    }
    return <LegacySportsLiveEventCard item={props.item} history={props.history} historyState={props.historyState} />;
};

const sportsHistoryTime = (value?: string) => formatBeijingDateTime(value) || '-';

const SportsHistoryEventInfoSection = (props: {item: SportsHistoryEventCardItem}) => {
    return (
        <div className='sports-live-card__info'>
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
    item: SportsHistoryEventCardItem;
    history?: SportsPriceHistorySeriesItem[];
    historyState?: 'loading' | 'failed';
    scratchMode?: boolean;
    scratchResetVersion?: number;
}) => (
    <SportsLiveEventCardFrame
        className='sports-history-card'
        item={props.item}
        options={legacyMoneylineOptions(props.item)}
        history={props.history}
        historyState={props.historyState}
        scratchMode={props.scratchMode}
        scratchResetVersion={props.scratchResetVersion}
        info={<SportsHistoryEventInfoSection item={props.item} />}
    />
);
