import {MockupList, Page} from 'argo-ui';
import * as React from 'react';

import {services} from '../../shared/services';
import {PolymarketSportsLiveMarketItem} from '../../shared/services/polymarket-service';

require('./polymarket-container.scss');

const POLL_INTERVAL_MS = 2000;
const DEFAULT_LIMIT = 200;

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

const MarketLogo = ({market}: {market: PolymarketSportsLiveMarketItem}) => {
    const [failed, setFailed] = React.useState(false);

    React.useEffect(() => setFailed(false), [market.image]);

    return <span className='polymarket-live__logo'>{market.image && !failed ? <img src={market.image} alt='' onError={() => setFailed(true)} /> : <span>P</span>}</span>;
};

export const PolymarketContainer = () => {
    const [markets, setMarkets] = React.useState<PolymarketSportsLiveMarketItem[]>([]);
    const [loading, setLoading] = React.useState(true);
    const [refreshing, setRefreshing] = React.useState(false);
    const [error, setError] = React.useState<Error | null>(null);
    const [stale, setStale] = React.useState(false);
    const [fetchedAt, setFetchedAt] = React.useState<number | undefined>(undefined);
    const mountedRef = React.useRef(false);
    const requestRef = React.useRef<{abort?: () => void} | null>(null);

    const loadMarkets = React.useCallback(async () => {
        if (requestRef.current?.abort) {
            requestRef.current.abort();
        }
        if (mountedRef.current) {
            setRefreshing(true);
        }

        const req = services.polymarket.listSportsLiveMarkets(DEFAULT_LIMIT);
        requestRef.current = req;
        try {
            const data = await req;
            if (mountedRef.current && requestRef.current === req) {
                setMarkets(data.items || []);
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
        loadMarkets();

        return () => {
            mountedRef.current = false;
            if (requestRef.current?.abort) {
                requestRef.current.abort();
            }
        };
    }, [loadMarkets]);

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
                interval = window.setInterval(() => loadMarkets(), POLL_INTERVAL_MS);
            }
        };
        const onVisibilityChange = () => {
            if (document.hidden) {
                stopPolling();
                abortCurrentRequest();
                return;
            }
            loadMarkets();
            startPolling();
        };

        startPolling();
        document.addEventListener('visibilitychange', onVisibilityChange);
        return () => {
            document.removeEventListener('visibilitychange', onVisibilityChange);
            stopPolling();
            abortCurrentRequest();
        };
    }, [loadMarkets]);

    return (
        <Page title='Polymarket' toolbar={{breadcrumbs: [{title: 'Polymarket'}]}}>
            <div className='polymarket-live'>
                {error && (
                    <div className='polymarket-live__error'>
                        <i className='fa fa-exclamation-triangle' /> Failed to load Polymarket Sports live markets: {error.message}
                    </div>
                )}

                {loading && markets.length === 0 ? (
                    <MockupList height={50} marginTop={30} />
                ) : (
                    <div className='argo-container'>
                        <div className='white-box polymarket-live__box'>
                            <div className='polymarket-live__status'>
                                <span>Rows: {markets.length}</span>
                                <span>Fetched At: {formatFetchedAt(fetchedAt)}</span>
                                <span>Refresh: {refreshing ? 'Updating' : 'Idle'}</span>
                                <span>Snapshot: {stale ? 'Stale' : 'Fresh'}</span>
                            </div>

                            <div className='polymarket-live__list'>
                                {markets.length === 0 ? (
                                    <div className='polymarket-live__empty'>No Sports Live markets found</div>
                                ) : (
                                    markets.map(market => (
                                        <div key={market.conditionId || market.marketSlug} className='polymarket-live__item'>
                                            <MarketLogo market={market} />
                                            <div className='polymarket-live__main'>
                                                <div className='polymarket-live__title' title={market.title}>
                                                    {market.title || '-'}
                                                </div>
                                                <div className='polymarket-live__slug' title={market.marketSlug}>
                                                    {market.marketSlug || '-'}
                                                </div>
                                                <div className='polymarket-live__slug' title={market.eventSlug}>
                                                    event: {market.eventSlug || '-'}
                                                </div>
                                            </div>
                                            <div className='polymarket-live__meta'>
                                                <div className='polymarket-live__metric'>
                                                    <span>Score</span>
                                                    <strong>{market.score || '-'}</strong>
                                                </div>
                                                <div className='polymarket-live__metric'>
                                                    <span>Period</span>
                                                    <strong>{market.period || '-'}</strong>
                                                </div>
                                                <div className='polymarket-live__metric'>
                                                    <span>Elapsed</span>
                                                    <strong>{market.elapsed || '-'}</strong>
                                                </div>
                                                <div className='polymarket-live__metric'>
                                                    <span>Last Update</span>
                                                    <strong>{formatLastUpdate(market.lastUpdate)}</strong>
                                                </div>
                                                <div className='polymarket-live__metric'>
                                                    <span>Liquidity</span>
                                                    <strong>{formatNumber(market.liquidityNum)}</strong>
                                                </div>
                                                <div className='polymarket-live__metric'>
                                                    <span>Volume</span>
                                                    <strong>{formatNumber(market.volumeNum)}</strong>
                                                </div>
                                            </div>
                                        </div>
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
