import {MockupList, Page} from 'argo-ui';
import * as React from 'react';

import {services} from '../../shared/services';
import {PolymarketMoverMarketItem, PolymarketMoverTokenItem, PolymarketMoverWindowItem} from '../../shared/services/polymarket-service';

require('./polymarket-container.scss');

const POLL_INTERVAL_MS = 5000;
const FETCH_LIMIT = 500;
const PAGE_SIZE = 100;
const POLYMARKET_URL = 'https://polymarket.com';

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

const formatTimestamp = (value?: number) => {
    if (!value) {
        return '-';
    }
    return new Date(value * 1000).toLocaleTimeString();
};

const formatNumber = (value?: number) => {
    if (value === undefined || value === null) {
        return '-';
    }
    return value.toLocaleString(undefined, {maximumFractionDigits: 2});
};

const formatScore = (value?: number) => {
    if (value === undefined || value === null) {
        return '-';
    }
    return value.toFixed(2);
};

const formatPrice = (value?: number) => {
    if (value === undefined || value === null || value <= 0) {
        return '-';
    }
    return value.toFixed(3);
};

const formatPp = (value?: number, warmup?: boolean) => {
    if (warmup || value === undefined || value === null) {
        return 'warming';
    }
    const sign = value > 0 ? '+' : '';
    return `${sign}${value.toFixed(1)}pp`;
};

const marketURL = (eventSlug?: string, marketSlug?: string) => {
    const slug = String(eventSlug || marketSlug || '').trim();
    if (!slug) {
        return POLYMARKET_URL;
    }
    return `${POLYMARKET_URL}/event/${slug}`;
};

const openExternal = (url: string) => {
    try {
        const opened = window.open(url, '_blank', 'noopener,noreferrer');
        if (!opened) {
            window.open(POLYMARKET_URL, '_blank', 'noopener,noreferrer');
        }
    } catch {
        window.open(POLYMARKET_URL, '_blank', 'noopener,noreferrer');
    }
};

const windowByName = (token: PolymarketMoverTokenItem | undefined, name: string) => (token?.windows || []).find(window => window.window === name);

const MoverMarketLogo = ({market}: {market: PolymarketMoverMarketItem}) => {
    const [failed, setFailed] = React.useState(false);

    React.useEffect(() => setFailed(false), [market.image]);

    return <span className='polymarket-movers__logo'>{market.image && !failed ? <img src={market.image} alt='' onError={() => setFailed(true)} /> : <span>P</span>}</span>;
};

const WindowChip = ({window}: {window?: PolymarketMoverWindowItem}) => {
    const className = ['polymarket-movers__window'];
    if (!window || window.warmup) {
        className.push('polymarket-movers__window--warmup');
    } else if ((window.priceChangePp || 0) > 0) {
        className.push('polymarket-movers__window--up');
    } else if ((window.priceChangePp || 0) < 0) {
        className.push('polymarket-movers__window--down');
    }
    return (
        <span className={className.join(' ')}>
            <span>{window?.window || '-'}</span>
            <strong>{formatPp(window?.priceChangePp, !window || window.warmup)}</strong>
        </span>
    );
};

const MoverToken = ({token, leader}: {token: PolymarketMoverTokenItem; leader?: boolean}) => {
    const className = ['polymarket-movers__token'];
    if (leader) {
        className.push('polymarket-movers__token--leader');
    }
    if (token.direction === 'up') {
        className.push('polymarket-movers__token--up');
    } else if (token.direction === 'down') {
        className.push('polymarket-movers__token--down');
    }
    return (
        <div className={className.join(' ')} title={token.tokenId}>
            <div className='polymarket-movers__token-top'>
                <strong>{token.outcome || '-'}</strong>
                <span>{formatScore(token.score)}</span>
            </div>
            <div className='polymarket-movers__quote'>
                <span>Px {formatPrice(token.price)}</span>
                <span>Bid {formatPrice(token.bestBid)}</span>
                <span>Ask {formatPrice(token.bestAsk)}</span>
                <span>Spr {formatPrice(token.spread)}</span>
            </div>
            <div className='polymarket-movers__quote'>
                <span>Last {formatPrice(token.lastTradePrice)}</span>
                <span>{token.lastTradeSide || '-'}</span>
                <span>{formatTimestamp(token.lastEventAt)}</span>
            </div>
            <div className='polymarket-movers__windows'>{(token.windows || []).map(window => <WindowChip key={window.window} window={window} />)}</div>
        </div>
    );
};

const MoverMetric = ({label, value}: {label: string; value: string}) => (
    <div className='polymarket-movers__metric'>
        <span>{label}</span>
        <strong>{value}</strong>
    </div>
);

export const MoversContainer = () => {
    const [markets, setMarkets] = React.useState<PolymarketMoverMarketItem[]>([]);
    const [currentPage, setCurrentPage] = React.useState(1);
    const [loading, setLoading] = React.useState(true);
    const [refreshing, setRefreshing] = React.useState(false);
    const [error, setError] = React.useState<Error | null>(null);
    const [stale, setStale] = React.useState(false);
    const [connected, setConnected] = React.useState(false);
    const [fetchedAt, setFetchedAt] = React.useState<number | undefined>(undefined);
    const [lastEventAt, setLastEventAt] = React.useState<number | undefined>(undefined);
    const [monitoredMarkets, setMonitoredMarkets] = React.useState<number | undefined>(undefined);
    const [monitoredTokens, setMonitoredTokens] = React.useState<number | undefined>(undefined);
    const [candidateCount, setCandidateCount] = React.useState<number | undefined>(undefined);
    const mountedRef = React.useRef(false);
    const requestRef = React.useRef<{abort?: () => void} | null>(null);

    const loadMovers = React.useCallback(async () => {
        if (requestRef.current?.abort) {
            requestRef.current.abort();
        }
        if (mountedRef.current) {
            setRefreshing(true);
        }

        const req = services.polymarket.listMovers(FETCH_LIMIT);
        requestRef.current = req;
        try {
            const data = await req;
            if (mountedRef.current && requestRef.current === req) {
                const nextItems = data.items || [];
                setMarkets(nextItems);
                setCurrentPage(page => {
                    const nextTotalPages = Math.max(1, Math.ceil(nextItems.length / PAGE_SIZE));
                    return Math.min(Math.max(1, page), nextTotalPages);
                });
                setFetchedAt(data.fetchedAt);
                setLastEventAt(data.lastEventAt);
                setStale(Boolean(data.stale));
                setConnected(Boolean(data.connected));
                setMonitoredMarkets(data.monitoredMarkets);
                setMonitoredTokens(data.monitoredTokens);
                setCandidateCount(data.candidateCount);
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
        loadMovers();

        return () => {
            mountedRef.current = false;
            if (requestRef.current?.abort) {
                requestRef.current.abort();
            }
        };
    }, [loadMovers]);

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
                interval = window.setInterval(() => loadMovers(), POLL_INTERVAL_MS);
            }
        };
        const onVisibilityChange = () => {
            if (document.hidden) {
                stopPolling();
                abortCurrentRequest();
                return;
            }
            loadMovers();
            startPolling();
        };

        startPolling();
        document.addEventListener('visibilitychange', onVisibilityChange);
        return () => {
            document.removeEventListener('visibilitychange', onVisibilityChange);
            stopPolling();
            abortCurrentRequest();
        };
    }, [loadMovers]);

    const totalPages = Math.max(1, Math.ceil(markets.length / PAGE_SIZE));
    const page = Math.min(Math.max(1, currentPage), totalPages);
    const startIndex = (page - 1) * PAGE_SIZE;
    const endIndex = Math.min(startIndex + PAGE_SIZE, markets.length);
    const visibleMarkets = markets.slice(startIndex, endIndex);
    const shownLabel = markets.length === 0 ? '0 of 0' : `${startIndex + 1}-${endIndex} of ${markets.length}`;
    const canGoPrevious = page > 1;
    const canGoNext = page < totalPages;
    const goFirst = () => setCurrentPage(1);
    const goPrevious = () => setCurrentPage(value => Math.max(1, value - 1));
    const goNext = () => setCurrentPage(value => Math.min(totalPages, value + 1));
    const goLast = () => setCurrentPage(totalPages);

    return (
        <Page title='Polymarket Movers' toolbar={{breadcrumbs: [{title: 'Polymarket', path: '/polymarket'}, {title: 'Movers'}]}}>
            <div className='polymarket-movers'>
                {error && (
                    <div className='polymarket-movers__error'>
                        <i className='fa fa-exclamation-triangle' /> Failed to load Polymarket movers: {error.message}
                    </div>
                )}

                {loading && markets.length === 0 ? (
                    <MockupList height={130} marginTop={30} />
                ) : (
                    <div className='argo-container'>
                        <div className='white-box polymarket-movers__box'>
                            <div className='polymarket-movers__status'>
                                <span>Shown: {shownLabel}</span>
                                <span>Connection: {connected ? 'Connected' : 'Waiting'}</span>
                                <span>Snapshot: {stale ? 'Stale' : 'Fresh'}</span>
                                <span>Monitored: {formatNumber(monitoredMarkets)}</span>
                                <span>Tokens: {formatNumber(monitoredTokens)}</span>
                                <span>Candidates: {formatNumber(candidateCount)}</span>
                                <span>Last Event: {formatTimestamp(lastEventAt)}</span>
                                <span>Fetched At: {formatFetchedAt(fetchedAt)}</span>
                                <span>Refresh: {refreshing ? 'Updating' : 'Idle'}</span>
                            </div>

                            <div className='polymarket-movers__pagination' aria-label='Polymarket movers pagination'>
                                <button type='button' onClick={goFirst} disabled={!canGoPrevious}>
                                    First
                                </button>
                                <button type='button' onClick={goPrevious} disabled={!canGoPrevious}>
                                    Previous
                                </button>
                                <span>
                                    Page {page} / {totalPages}
                                </span>
                                <button type='button' onClick={goNext} disabled={!canGoNext}>
                                    Next
                                </button>
                                <button type='button' onClick={goLast} disabled={!canGoNext}>
                                    Last
                                </button>
                            </div>

                            <div className='polymarket-movers__list'>
                                {markets.length === 0 ? (
                                    <div className='polymarket-movers__empty'>No movers found</div>
                                ) : (
                                    visibleMarkets.map((market, index) => {
                                        const leader = market.leader;
                                        return (
                                            <article key={market.conditionId || `${market.marketSlug}-${index}`} className='polymarket-movers__row'>
                                                <div className='polymarket-movers__rank'>{startIndex + index + 1}</div>
                                                <MoverMarketLogo market={market} />
                                                <div className='polymarket-movers__main'>
                                                    <button type='button' className='polymarket-movers__title' onClick={() => openExternal(marketURL(market.eventSlug, market.marketSlug))}>
                                                        {market.question || market.marketSlug || '-'}
                                                    </button>
                                                    <div className='polymarket-movers__subtitle'>{market.conditionId || '-'}</div>
                                                    <div className='polymarket-movers__metrics'>
                                                        <MoverMetric label='Leader' value={leader?.outcome || '-'} />
                                                        <MoverMetric label='Score' value={formatScore(market.score)} />
                                                        <MoverMetric label='Direction' value={market.direction || '-'} />
                                                        <MoverMetric label='Price' value={formatPrice(leader?.price)} />
                                                        <MoverMetric label='24h Vol' value={formatNumber(market.volume24hr)} />
                                                        <MoverMetric label='Liquidity' value={formatNumber(market.liquidityNum)} />
                                                    </div>
                                                    <div className='polymarket-movers__leader-windows'>
                                                        <WindowChip window={windowByName(leader, '1m')} />
                                                        <WindowChip window={windowByName(leader, '5m')} />
                                                        <WindowChip window={windowByName(leader, '15m')} />
                                                    </div>
                                                </div>
                                                <div className='polymarket-movers__tokens'>
                                                    {(market.tokens || []).map(token => <MoverToken key={token.tokenId} token={token} leader={leader?.tokenId === token.tokenId} />)}
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
