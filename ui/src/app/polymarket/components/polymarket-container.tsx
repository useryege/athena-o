import {MockupList, Page} from 'argo-ui';
import * as React from 'react';

import {services} from '../../shared/services';
import {PolymarketSportsLiveEventItem, PolymarketSportsLiveMarketOptionItem, PolymarketSportsLiveTeamItem} from '../../shared/services/polymarket-service';

require('./polymarket-container.scss');

const POLL_INTERVAL_MS = 2000;
const DEFAULT_LIMIT = 30;
const POLYMARKET_SPORTS_LIVE_URL = 'https://polymarket.com/sports/live';
const MONEYLINE_MARKET_TYPE = 'moneyline';

const isAbortedError = (err: unknown) =>
    String((err as any)?.message || '')
        .toLowerCase()
        .includes('abort');

const formatFetchedAt = (fetchedAt?: number) => {
    if (!fetchedAt) {
        return '-';
    }
    return new Date(fetchedAt * 1000).toLocaleTimeString();
};

const formatLastUpdate = (value?: string) => {
    if (!value) {
        return '-';
    }
    const parsed = new Date(value);
    if (Number.isNaN(parsed.getTime())) {
        return value;
    }
    return parsed.toLocaleTimeString();
};

const formatNumber = (value?: number) => {
    if (value === undefined || value === null) {
        return '-';
    }
    return value.toLocaleString(undefined, {maximumFractionDigits: 2});
};

const formatPrice = (value?: string) => {
    if (!value) {
        return '-';
    }
    const numeric = Number(value);
    if (!Number.isFinite(numeric)) {
        return value;
    }
    return numeric.toFixed(2);
};

const eventURL = (eventSlug?: string) => {
    const slug = String(eventSlug || '').trim();
    if (!slug) {
        return POLYMARKET_SPORTS_LIVE_URL;
    }
    return `https://polymarket.com/event/${slug}`;
};

const marketURL = (marketSlug?: string) => {
    const slug = String(marketSlug || '').trim();
    if (!slug) {
        return POLYMARKET_SPORTS_LIVE_URL;
    }
    return `https://polymarket.com/event/${slug}`;
};

const openExternal = (url: string) => {
    try {
        const opened = window.open(url, '_blank', 'noopener,noreferrer');
        if (!opened) {
            window.open(POLYMARKET_SPORTS_LIVE_URL, '_blank', 'noopener,noreferrer');
        }
    } catch {
        window.open(POLYMARKET_SPORTS_LIVE_URL, '_blank', 'noopener,noreferrer');
    }
};

const getMoneylineGroup = (event: PolymarketSportsLiveEventItem) => (event.markets || []).find(group => String(group.type || '').toLowerCase() === MONEYLINE_MARKET_TYPE);
const sumMoneylineVolume = (event: PolymarketSportsLiveEventItem) => (getMoneylineGroup(event)?.markets || []).reduce((total, market) => total + (market.volumeNum || 0), 0);
const normalizeMatchName = (value?: string) =>
    String(value || '')
        .trim()
        .toLowerCase();
const findOutcomeTeam = (event: PolymarketSportsLiveEventItem, outcome: string, index: number) => {
    const teams = event.teams || [];
    const normalizedOutcome = normalizeMatchName(outcome);
    const exactMatch = teams.find(team => normalizeMatchName(team.name) === normalizedOutcome);
    return exactMatch || teams[index];
};

const EventLogo = ({event}: {event: PolymarketSportsLiveEventItem}) => {
    const [failed, setFailed] = React.useState(false);

    React.useEffect(() => setFailed(false), [event.image]);

    return <span className='polymarket-live__logo'>{event.image && !failed ? <img src={event.image} alt='' onError={() => setFailed(true)} /> : <span>P</span>}</span>;
};

const OutcomeFlag = ({team}: {team?: PolymarketSportsLiveTeamItem}) => {
    const [failed, setFailed] = React.useState(false);

    React.useEffect(() => setFailed(false), [team?.logo]);

    return <span className='polymarket-live__outcome-flag'>{team?.logo && !failed && <img src={team.logo} alt='' onError={() => setFailed(true)} />}</span>;
};

const MarketOutcomeButton = ({
    market,
    outcome,
    price,
    team,
    onClick
}: {
    market: PolymarketSportsLiveMarketOptionItem;
    outcome: string;
    price?: string;
    team?: PolymarketSportsLiveTeamItem;
    onClick: () => void;
}) => (
    <button type='button' className='polymarket-live__outcome' onClick={onClick} title={market.question || market.marketSlug || ''}>
        <OutcomeFlag team={team} />
        <span className='polymarket-live__outcome-label'>{outcome || market.question || market.marketSlug || '-'}</span>
        <span className='polymarket-live__outcome-price'>{formatPrice(price)}</span>
    </button>
);

export const PolymarketContainer = () => {
    const [events, setEvents] = React.useState<PolymarketSportsLiveEventItem[]>([]);
    const [loading, setLoading] = React.useState(true);
    const [refreshing, setRefreshing] = React.useState(false);
    const [error, setError] = React.useState<Error | null>(null);
    const [stale, setStale] = React.useState(false);
    const [fetchedAt, setFetchedAt] = React.useState<number | undefined>(undefined);
    const mountedRef = React.useRef(false);
    const requestRef = React.useRef<{abort?: () => void} | null>(null);

    const loadSnapshot = React.useCallback(async () => {
        if (requestRef.current?.abort) {
            requestRef.current.abort();
        }
        if (mountedRef.current) {
            setRefreshing(true);
        }

        const req = services.polymarket.getSportsLiveSnapshot(DEFAULT_LIMIT);
        requestRef.current = req;
        try {
            const data = await req;
            if (mountedRef.current && requestRef.current === req) {
                setEvents(data.events || []);
                setFetchedAt(data.fetchedAt);
                setStale(Boolean(data.stale));
                setError(null);
            }
        } catch (err) {
            if (mountedRef.current && requestRef.current === req && !isAbortedError(err)) {
                setError(err as Error);
            }
        } finally {
            if (requestRef.current === req) {
                requestRef.current = null;
            }
            if (mountedRef.current && requestRef.current === null) {
                setLoading(false);
                setRefreshing(false);
            }
        }
    }, []);

    React.useEffect(() => {
        mountedRef.current = true;
        loadSnapshot();

        return () => {
            mountedRef.current = false;
            if (requestRef.current?.abort) {
                requestRef.current.abort();
            }
        };
    }, [loadSnapshot]);

    React.useEffect(() => {
        let interval: number | undefined;
        const stopPolling = () => {
            if (interval !== undefined) {
                window.clearInterval(interval);
                interval = undefined;
            }
        };
        const abortCurrentRequest = () => {
            if (requestRef.current?.abort) {
                requestRef.current.abort();
            }
        };
        const startPolling = () => {
            stopPolling();
            if (!document.hidden) {
                interval = window.setInterval(() => loadSnapshot(), POLL_INTERVAL_MS);
            }
        };
        const onVisibilityChange = () => {
            if (document.hidden) {
                stopPolling();
                abortCurrentRequest();
                return;
            }
            loadSnapshot();
            startPolling();
        };

        startPolling();
        document.addEventListener('visibilitychange', onVisibilityChange);
        return () => {
            document.removeEventListener('visibilitychange', onVisibilityChange);
            stopPolling();
            abortCurrentRequest();
        };
    }, [loadSnapshot]);

    return (
        <Page title='Polymarket Sports Live' toolbar={{breadcrumbs: [{title: 'Polymarket', path: '/polymarket'}, {title: 'Sports Live'}]}}>
            <div className='polymarket-live'>
                {error && (
                    <div className='polymarket-live__error'>
                        <i className='fa fa-exclamation-triangle' /> Failed to load Polymarket Sports live snapshot: {error.message}
                    </div>
                )}

                {loading && events.length === 0 ? (
                    <MockupList height={130} marginTop={30} />
                ) : (
                    <div className='argo-container'>
                        <div className='white-box polymarket-live__box'>
                            <div className='polymarket-live__status'>
                                <span>Events: {events.length}</span>
                                <span>Fetched At: {formatFetchedAt(fetchedAt)}</span>
                                <span>Refresh: {refreshing ? 'Updating' : 'Idle'}</span>
                                <span>Snapshot: {stale ? 'Stale' : 'Fresh'}</span>
                            </div>

                            <div className='polymarket-live__list'>
                                {events.length === 0 ? (
                                    <div className='polymarket-live__empty'>No Sports Live moneyline markets found</div>
                                ) : (
                                    events.map(event => {
                                        const moneylineGroup = getMoneylineGroup(event);
                                        return (
                                            <article key={event.eventSlug} className='polymarket-live__card'>
                                                <div className='polymarket-live__card-header'>
                                                    <div className='polymarket-live__headline'>
                                                        <span className='polymarket-live__live-dot' />
                                                        <span className='polymarket-live__live-text'>{event.period || 'LIVE'}</span>
                                                    </div>
                                                    <div className='polymarket-live__header-meta'>
                                                        <span>Vol {formatNumber(sumMoneylineVolume(event))}</span>
                                                        <span>Updated {formatLastUpdate(event.lastUpdate)}</span>
                                                        <span>Fetched {formatFetchedAt(fetchedAt)}</span>
                                                    </div>
                                                </div>

                                                <div className='polymarket-live__event-main'>
                                                    <EventLogo event={event} />
                                                    <div className='polymarket-live__event-body'>
                                                        <button type='button' className='polymarket-live__event-link' onClick={() => openExternal(eventURL(event.eventSlug))}>
                                                            {event.title || event.eventSlug || '-'}
                                                        </button>
                                                        <div className='polymarket-live__event-subtitle'>{event.gameStatus || event.elapsed || '-'}</div>
                                                    </div>
                                                    <div className='polymarket-live__score'>{event.score || '-'}</div>
                                                </div>

                                                <div className='polymarket-live__groups'>
                                                    {moneylineGroup && (
                                                        <section className='polymarket-live__group'>
                                                            <div className='polymarket-live__outcomes'>
                                                                {(moneylineGroup.markets || []).map(market => {
                                                                    const outcomes = market.outcomes || [];
                                                                    const prices = market.outcomePrices || [];
                                                                    if (outcomes.length > 0) {
                                                                        return outcomes.map((outcome, idx) => (
                                                                            <MarketOutcomeButton
                                                                                key={`${market.marketSlug}-${outcome}-${idx}`}
                                                                                market={market}
                                                                                outcome={outcome}
                                                                                price={prices[idx]}
                                                                                team={findOutcomeTeam(event, outcome, idx)}
                                                                                onClick={() => openExternal(marketURL(market.marketSlug))}
                                                                            />
                                                                        ));
                                                                    }
                                                                    return (
                                                                        <MarketOutcomeButton
                                                                            key={market.marketSlug || market.conditionId}
                                                                            market={market}
                                                                            outcome={market.question || market.marketSlug || '-'}
                                                                            team={findOutcomeTeam(event, market.question || market.marketSlug || '-', 0)}
                                                                            onClick={() => openExternal(marketURL(market.marketSlug))}
                                                                        />
                                                                    );
                                                                })}
                                                            </div>
                                                        </section>
                                                    )}
                                                </div>
                                            </article>
                                        );
                                    })
                                )}
                            </div>
                        </div>
                    </div>
                )}
            </div>
        </Page>
    );
};
