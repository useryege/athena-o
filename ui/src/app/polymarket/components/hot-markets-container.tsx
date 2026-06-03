import {MockupList, Page} from 'argo-ui';
import * as React from 'react';

import {services} from '../../shared/services';
import {PolymarketHotMarketItem, PolymarketHotMarketTokenItem} from '../../shared/services/polymarket-service';

require('./polymarket-container.scss');

const POLL_INTERVAL_MS = 10000;
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

const formatTimestamp = (value?: string) => {
    if (!value) {
        return '-';
    }
    const parsed = new Date(value);
    if (Number.isNaN(parsed.getTime())) {
        return value;
    }
    return parsed.toLocaleString();
};

const formatNumber = (value?: number) => {
    if (value === undefined || value === null) {
        return '-';
    }
    return value.toLocaleString(undefined, {maximumFractionDigits: 2});
};

const formatPrice = (value?: number) => {
    if (value === undefined || value === null) {
        return '-';
    }
    return value.toFixed(2);
};

const marketURL = (marketSlug?: string) => {
    const slug = String(marketSlug || '').trim();
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

const HotMarketLogo = ({market}: {market: PolymarketHotMarketItem}) => {
    const [failed, setFailed] = React.useState(false);

    React.useEffect(() => setFailed(false), [market.image]);

    return <span className='polymarket-hot__logo'>{market.image && !failed ? <img src={market.image} alt='' onError={() => setFailed(true)} /> : <span>P</span>}</span>;
};

const HotMarketToken = ({token}: {token: PolymarketHotMarketTokenItem}) => (
    <div className='polymarket-hot__token' title={token.tokenId}>
        <span className='polymarket-hot__token-outcome'>{token.outcome || '-'}</span>
        <span className='polymarket-hot__token-price'>{formatPrice(token.price)}</span>
    </div>
);

const HotMarketMetric = ({label, value}: {label: string; value: string}) => (
    <div className='polymarket-hot__metric'>
        <span>{label}</span>
        <strong>{value}</strong>
    </div>
);

export const HotMarketsContainer = () => {
    const [markets, setMarkets] = React.useState<PolymarketHotMarketItem[]>([]);
    const [currentPage, setCurrentPage] = React.useState(1);
    const [loading, setLoading] = React.useState(true);
    const [refreshing, setRefreshing] = React.useState(false);
    const [error, setError] = React.useState<Error | null>(null);
    const [stale, setStale] = React.useState(false);
    const [fetchedAt, setFetchedAt] = React.useState<number | undefined>(undefined);
    const [monitoredMarkets, setMonitoredMarkets] = React.useState<number | undefined>(undefined);
    const [monitoredTokens, setMonitoredTokens] = React.useState<number | undefined>(undefined);
    const [candidateCount, setCandidateCount] = React.useState<number | undefined>(undefined);
    const mountedRef = React.useRef(false);
    const requestRef = React.useRef<{abort?: () => void} | null>(null);

    const loadHotMarkets = React.useCallback(async () => {
        if (requestRef.current?.abort) {
            requestRef.current.abort();
        }
        if (mountedRef.current) {
            setRefreshing(true);
        }

        const req = services.polymarket.listHotMarkets(FETCH_LIMIT);
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
                setStale(Boolean(data.stale));
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
        loadHotMarkets();

        return () => {
            mountedRef.current = false;
            if (requestRef.current?.abort) {
                requestRef.current.abort();
            }
        };
    }, [loadHotMarkets]);

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
                interval = window.setInterval(() => loadHotMarkets(), POLL_INTERVAL_MS);
            }
        };
        const onVisibilityChange = () => {
            if (document.hidden) {
                stopPolling();
                abortCurrentRequest();
                return;
            }
            loadHotMarkets();
            startPolling();
        };

        startPolling();
        document.addEventListener('visibilitychange', onVisibilityChange);
        return () => {
            document.removeEventListener('visibilitychange', onVisibilityChange);
            stopPolling();
            abortCurrentRequest();
        };
    }, [loadHotMarkets]);

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
        <Page title='Polymarket Hot Markets' toolbar={{breadcrumbs: [{title: 'Polymarket', path: '/polymarket'}, {title: 'Hot Markets'}]}}>
            <div className='polymarket-hot'>
                {error && (
                    <div className='polymarket-hot__error'>
                        <i className='fa fa-exclamation-triangle' /> Failed to load Polymarket hot markets: {error.message}
                    </div>
                )}

                {loading && markets.length === 0 ? (
                    <MockupList height={130} marginTop={30} />
                ) : (
                    <div className='argo-container'>
                        <div className='white-box polymarket-hot__box'>
                            <div className='polymarket-hot__status'>
                                <span>Shown: {shownLabel}</span>
                                <span>Monitored: {formatNumber(monitoredMarkets)}</span>
                                <span>Tokens: {formatNumber(monitoredTokens)}</span>
                                <span>Candidates: {formatNumber(candidateCount)}</span>
                                <span>Fetched At: {formatFetchedAt(fetchedAt)}</span>
                                <span>Refresh: {refreshing ? 'Updating' : 'Idle'}</span>
                                <span>Snapshot: {stale ? 'Stale' : 'Fresh'}</span>
                            </div>

                            <div className='polymarket-hot__pagination' aria-label='Hot markets pagination'>
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

                            <div className='polymarket-hot__list'>
                                {markets.length === 0 ? (
                                    <div className='polymarket-hot__empty'>No hot markets found</div>
                                ) : (
                                    visibleMarkets.map((market, index) => (
                                        <article key={market.conditionId || `${market.marketSlug}-${index}`} className='polymarket-hot__row'>
                                            <div className='polymarket-hot__rank'>{startIndex + index + 1}</div>
                                            <HotMarketLogo market={market} />
                                            <div className='polymarket-hot__main'>
                                                <button type='button' className='polymarket-hot__title' onClick={() => openExternal(marketURL(market.marketSlug))}>
                                                    {market.question || market.marketSlug || '-'}
                                                </button>
                                                <div className='polymarket-hot__subtitle'>{market.conditionId || '-'}</div>
                                                <div className='polymarket-hot__tokens'>{(market.tokens || []).map(token => <HotMarketToken key={token.tokenId} token={token} />)}</div>
                                            </div>
                                            <div className='polymarket-hot__metrics'>
                                                <HotMarketMetric label='24h Vol' value={formatNumber(market.volume24hr)} />
                                                <HotMarketMetric label='Liquidity' value={formatNumber(market.liquidityNum)} />
                                                <HotMarketMetric label='Last' value={formatPrice(market.lastTradePrice)} />
                                                <HotMarketMetric label='Spread' value={formatPrice(market.spread)} />
                                                <HotMarketMetric label='Bid / Ask' value={`${formatPrice(market.bestBid)} / ${formatPrice(market.bestAsk)}`} />
                                                <HotMarketMetric label='Updated' value={formatTimestamp(market.updatedAt)} />
                                            </div>
                                        </article>
                                    ))
                                )}
                            </div>
                        </div>
                    </div>
                )}
            </div>
        </Page>
    );
};
