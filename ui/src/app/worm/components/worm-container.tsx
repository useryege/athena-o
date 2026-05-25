import {MockupList, Page, SlidingPanel} from 'argo-ui';
import * as React from 'react';

import {services} from '../../shared/services';
import {WormMarketCategorySlug, WormMarketItem, WormMarketSortOption} from '../../shared/services/worm-service';
import {WormMarketDetailPanel} from './worm-market-detail-panel';

require('./worm-container.scss');

const PAGE_SIZE = 20;
const POLL_INTERVAL_MS = 1000;

const SORT_OPTIONS: Array<{label: string; value: WormMarketSortOption}> = [
    {label: 'New', value: 'new'},
    {label: 'Trending', value: 'trending'},
    {label: 'Ending Soon', value: 'ending_soon'},
    {label: 'Leverage', value: 'leverage'}
];

const CATEGORY_OPTIONS: Array<{label: string; value: WormMarketCategorySlug}> = [
    {label: 'All', value: 'all'},
    {label: 'Politics', value: 'politics'},
    {label: 'Sports', value: 'sports'},
    {label: 'Crypto', value: 'crypto'},
    {label: 'Tech', value: 'tech'},
    {label: 'Finance', value: 'finance'},
    {label: 'WTF', value: 'wtf'}
];

const isAbortedError = (err: unknown) =>
    String((err as any)?.message || '')
        .toLowerCase()
        .includes('abort');

const renderCreated = (created?: number) => (created ? new Date(created * 1000).toLocaleString() : '-');

const renderPrice = (price?: string) => price || '-';

const getMarketKey = (market: WormMarketItem, index: number) => market.conditionId || `${market.title}-${index}`;

const MarketLogo = ({market}: {market: WormMarketItem}) => {
    const logo = market.logo || market.eventLogo;
    const [failed, setFailed] = React.useState(false);

    React.useEffect(() => setFailed(false), [logo]);

    return <span className='worm-markets__logo'>{logo && !failed ? <img src={logo} alt='' onError={() => setFailed(true)} /> : <span>W</span>}</span>;
};

export const WormContainer = () => {
    const [markets, setMarkets] = React.useState<WormMarketItem[]>([]);
    const [sortOption, setSortOption] = React.useState<WormMarketSortOption>('new');
    const [categorySlug, setCategorySlug] = React.useState<WormMarketCategorySlug>('all');
    const [loading, setLoading] = React.useState(true);
    const [refreshing, setRefreshing] = React.useState(false);
    const [error, setError] = React.useState<Error | null>(null);
    const [cursor, setCursor] = React.useState('');
    const [nextCursor, setNextCursor] = React.useState('');
    const [cursorStack, setCursorStack] = React.useState<string[]>([]);
    const [lastUpdatedAt, setLastUpdatedAt] = React.useState<Date | null>(null);
    const [stale, setStale] = React.useState(false);
    const [selectedConditionId, setSelectedConditionId] = React.useState('');
    const requestRef = React.useRef<{abort?: () => void} | null>(null);
    const mountedRef = React.useRef(false);

    const loadMarkets = React.useCallback(
        async (targetCursor = '', nextStack?: string[]) => {
            if (requestRef.current?.abort) {
                requestRef.current.abort();
            }
            if (mountedRef.current) {
                setRefreshing(true);
            }
            const req = services.worm.listMarkets({
                limit: PAGE_SIZE,
                cursor: targetCursor,
                sortOption,
                categorySlug
            });
            requestRef.current = req;
            try {
                const data = await req;
                if (mountedRef.current && requestRef.current === req) {
                    setMarkets(data.items);
                    setCursor(targetCursor);
                    setNextCursor(data.nextCursor || '');
                    if (nextStack) {
                        setCursorStack(nextStack);
                    }
                    setLastUpdatedAt(data.fetchedAt ? new Date(data.fetchedAt * 1000) : new Date());
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
        },
        [categorySlug, sortOption]
    );

    React.useEffect(() => {
        mountedRef.current = true;
        setLoading(true);
        setCursor('');
        setNextCursor('');
        setCursorStack([]);
        setSelectedConditionId('');
        loadMarkets('', []);
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
                interval = window.setInterval(() => loadMarkets(cursor), POLL_INTERVAL_MS);
            }
        };
        const handleVisibilityChange = () => {
            if (document.hidden) {
                stopPolling();
                abortCurrentRequest();
                return;
            }
            loadMarkets(cursor);
            startPolling();
        };

        startPolling();
        document.addEventListener('visibilitychange', handleVisibilityChange);
        return () => {
            document.removeEventListener('visibilitychange', handleVisibilityChange);
            stopPolling();
            abortCurrentRequest();
        };
    }, [cursor, loadMarkets]);

    const handleRefresh = React.useCallback(() => loadMarkets(cursor), [cursor, loadMarkets]);
    const handleNext = React.useCallback(() => {
        if (nextCursor) {
            loadMarkets(nextCursor, [...cursorStack, cursor]);
        }
    }, [cursor, cursorStack, loadMarkets, nextCursor]);
    const handlePrev = React.useCallback(() => {
        if (cursorStack.length === 0) {
            return;
        }
        const nextStack = cursorStack.slice(0, -1);
        loadMarkets(cursorStack[cursorStack.length - 1], nextStack);
    }, [cursorStack, loadMarkets]);
    const handleMarketKeyDown = React.useCallback((event: React.KeyboardEvent, conditionId: string) => {
        if (event.key === 'Enter' || event.key === ' ') {
            event.preventDefault();
            setSelectedConditionId(conditionId);
        }
    }, []);

    const page = cursorStack.length + 1;
    const selectedMarket = markets.find(market => market.conditionId === selectedConditionId);
    const activeSortLabel = SORT_OPTIONS.find(item => item.value === sortOption)?.label || 'New';
    const activeCategoryLabel = CATEGORY_OPTIONS.find(item => item.value === categorySlug)?.label || 'All';

    return (
        <Page title='Worm' toolbar={{breadcrumbs: [{title: 'Worm'}]}}>
            <div className='worm-markets'>
                {error && (
                    <div className='worm-markets__error'>
                        <i className='fa fa-exclamation-triangle' /> Failed to load Worm markets: {error.message}
                    </div>
                )}

                {loading && markets.length === 0 ? (
                    <MockupList height={50} marginTop={30} />
                ) : (
                    <div className='argo-container'>
                        <div className='white-box worm-markets__box'>
                            <div className='worm-markets__filters'>
                                <div className='worm-markets__filter-group'>
                                    {SORT_OPTIONS.map(item => (
                                        <button
                                            type='button'
                                            key={item.value}
                                            className={`worm-markets__filter ${sortOption === item.value ? 'worm-markets__filter--active' : ''}`}
                                            onClick={() => setSortOption(item.value)}>
                                            {item.label}
                                        </button>
                                    ))}
                                </div>
                                <div className='worm-markets__filter-group'>
                                    {CATEGORY_OPTIONS.map(item => (
                                        <button
                                            type='button'
                                            key={item.value}
                                            className={`worm-markets__filter ${categorySlug === item.value ? 'worm-markets__filter--active' : ''}`}
                                            onClick={() => setCategorySlug(item.value)}>
                                            {item.label}
                                        </button>
                                    ))}
                                </div>
                            </div>
                            <div className='worm-markets__controls'>
                                <div className='worm-markets__actions'>
                                    <button type='button' className='argo-button argo-button--base' disabled={refreshing} onClick={handleRefresh}>
                                        <i className='fa fa-refresh' /> {refreshing ? 'Refreshing...' : 'Refresh'}
                                    </button>
                                </div>
                                <div className='worm-markets__status'>
                                    <span>Section: {activeSortLabel}</span>
                                    <span>Category: {activeCategoryLabel}</span>
                                    <span>Page: {page}</span>
                                    <span>
                                        Last updated: {lastUpdatedAt ? lastUpdatedAt.toLocaleTimeString() : 'Never'}
                                        {stale ? ' (stale)' : ''}
                                    </span>
                                </div>
                            </div>

                            <div className='worm-markets__list'>
                                {markets.length === 0 ? (
                                    <div className='worm-markets__empty'>
                                        <div className='row'>
                                            <div className='columns small-12 text-center'>No Worm markets found</div>
                                        </div>
                                    </div>
                                ) : (
                                    markets.map((market, index) => (
                                        <div
                                            className={`worm-markets__item ${market.conditionId === selectedConditionId ? 'worm-markets__item--selected' : ''}`}
                                            key={getMarketKey(market, index)}
                                            role='button'
                                            tabIndex={0}
                                            onClick={() => setSelectedConditionId(market.conditionId)}
                                            onKeyDown={event => handleMarketKeyDown(event, market.conditionId)}>
                                            <MarketLogo market={market} />
                                            <div className='worm-markets__main'>
                                                <div className='worm-markets__title' title={market.title}>
                                                    {market.title || '-'}
                                                </div>
                                                <div className='worm-markets__event' title={market.eventTitle || market.eventConditionId}>
                                                    {market.eventTitle || '-'}
                                                </div>
                                                <div className='worm-markets__condition' title={market.conditionId}>
                                                    {market.conditionId}
                                                </div>
                                            </div>
                                            <div className='worm-markets__meta'>
                                                <div className='worm-markets__metric'>
                                                    <span>Price</span>
                                                    <strong>{renderPrice(market.lastTradePrice)}</strong>
                                                </div>
                                                <div className='worm-markets__metric'>
                                                    <span>State</span>
                                                    <strong className={`worm-markets__badge worm-markets__badge--${market.state || 'unknown'}`}>{market.state || '-'}</strong>
                                                </div>
                                                <div className='worm-markets__metric'>
                                                    <span>Margin</span>
                                                    <strong>{market.marginEnabled ? 'Yes' : 'No'}</strong>
                                                </div>
                                                <div className='worm-markets__metric worm-markets__metric--created'>
                                                    <span>Created</span>
                                                    <strong>{renderCreated(market.created)}</strong>
                                                </div>
                                            </div>
                                        </div>
                                    ))
                                )}
                            </div>

                            <div className='worm-markets__controls worm-markets__controls--footer'>
                                <div className='worm-markets__actions'>
                                    <button type='button' className='argo-button argo-button--base-o' disabled={cursorStack.length === 0 || refreshing} onClick={handlePrev}>
                                        <i className='fa fa-chevron-left' /> Prev
                                    </button>
                                    <button type='button' className='argo-button argo-button--base-o' disabled={!nextCursor || refreshing} onClick={handleNext}>
                                        Next <i className='fa fa-chevron-right' />
                                    </button>
                                </div>
                            </div>
                        </div>
                    </div>
                )}
                <SlidingPanel
                    header={<div className='worm-market-detail__panel-title'>Market Detail</div>}
                    isShown={!!selectedConditionId}
                    onClose={() => setSelectedConditionId('')}>
                    {selectedConditionId && <WormMarketDetailPanel key={selectedConditionId} conditionId={selectedConditionId} initialMarket={selectedMarket} />}
                </SlidingPanel>
            </div>
        </Page>
    );
};
