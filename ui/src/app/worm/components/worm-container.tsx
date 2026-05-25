import {MockupList, Page} from 'argo-ui';
import * as React from 'react';

import {services} from '../../shared/services';
import {WormMarketItem} from '../../shared/services/worm-service';

require('./worm-container.scss');

const PAGE_SIZE = 20;

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
    const [loading, setLoading] = React.useState(true);
    const [refreshing, setRefreshing] = React.useState(false);
    const [error, setError] = React.useState<Error | null>(null);
    const [cursor, setCursor] = React.useState('');
    const [nextCursor, setNextCursor] = React.useState('');
    const [cursorStack, setCursorStack] = React.useState<string[]>([]);
    const [lastUpdatedAt, setLastUpdatedAt] = React.useState<Date | null>(null);
    const requestRef = React.useRef<{abort?: () => void} | null>(null);
    const mountedRef = React.useRef(false);

    const loadMarkets = React.useCallback(async (targetCursor = '', nextStack?: string[]) => {
        if (requestRef.current) {
            return;
        }
        if (mountedRef.current) {
            setRefreshing(true);
        }
        try {
            const req = services.worm.listMarkets(PAGE_SIZE, targetCursor);
            requestRef.current = req;
            const data = await req;
            if (mountedRef.current) {
                setMarkets(data.items);
                setCursor(targetCursor);
                setNextCursor(data.nextCursor || '');
                if (nextStack) {
                    setCursorStack(nextStack);
                }
                setLastUpdatedAt(new Date());
                setError(null);
            }
        } catch (err) {
            if (mountedRef.current && !isAbortedError(err)) {
                setError(err as Error);
            }
        } finally {
            requestRef.current = null;
            if (mountedRef.current) {
                setLoading(false);
                setRefreshing(false);
            }
        }
    }, []);

    React.useEffect(() => {
        mountedRef.current = true;
        loadMarkets('');
        return () => {
            mountedRef.current = false;
            if (requestRef.current?.abort) {
                requestRef.current.abort();
            }
        };
    }, [loadMarkets]);

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

    const page = cursorStack.length + 1;

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
                            <div className='worm-markets__controls'>
                                <div className='worm-markets__actions'>
                                    <button type='button' className='argo-button argo-button--base' disabled={refreshing} onClick={handleRefresh}>
                                        <i className='fa fa-refresh' /> {refreshing ? 'Refreshing...' : 'Refresh'}
                                    </button>
                                </div>
                                <div className='worm-markets__status'>
                                    <span>Category: sports</span>
                                    <span>Sort: leverage</span>
                                    <span>Page: {page}</span>
                                    <span>Last updated: {lastUpdatedAt ? lastUpdatedAt.toLocaleTimeString() : 'Never'}</span>
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
                                        <div className='worm-markets__item' key={getMarketKey(market, index)}>
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
            </div>
        </Page>
    );
};
