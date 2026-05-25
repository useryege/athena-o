import {MockupList} from 'argo-ui';
import * as React from 'react';

import {services} from '../../shared/services';
import {WormMarketDetail, WormMarketItem, WormMarketOrderBook, WormMarketPrice} from '../../shared/services/worm-service';

const POLL_INTERVAL_MS = 1000;

const isAbortedError = (err: unknown) =>
    String((err as any)?.message || '')
        .toLowerCase()
        .includes('abort');

const renderValue = (value?: string | number) => (value !== undefined && value !== null && String(value) !== '' ? String(value) : '-');
const renderTimestamp = (value?: number) => (value ? new Date(value * 1000).toLocaleString() : '-');
const outcomeName = (isYes: boolean, detail?: WormMarketDetail) => (isYes ? detail?.yesOutcomeLabel || 'YES' : detail?.noOutcomeLabel || 'NO');

const DetailLogo = ({market}: {market?: WormMarketItem}) => {
    const logo = market?.logo || market?.eventLogo;
    const [failed, setFailed] = React.useState(false);

    React.useEffect(() => setFailed(false), [logo]);

    return <span className='worm-market-detail__logo'>{logo && !failed ? <img src={logo} alt='' onError={() => setFailed(true)} /> : <span>W</span>}</span>;
};

const DetailMetric = ({label, value}: {label: string; value?: string | number}) => (
    <div className='worm-market-detail__metric'>
        <span>{label}</span>
        <strong title={renderValue(value)}>{renderValue(value)}</strong>
    </div>
);

const PriceMetric = ({isYes, price, detail}: {isYes: boolean; price?: WormMarketPrice; detail?: WormMarketDetail}) => (
    <DetailMetric label={`${outcomeName(isYes, detail)} Price`} value={price?.price} />
);

const OrderBookTable = ({isYes, book, detail}: {isYes: boolean; book?: WormMarketOrderBook; detail?: WormMarketDetail}) => (
    <div className='worm-market-detail__book'>
        <div className='worm-market-detail__book-title'>{outcomeName(isYes, detail)} Order Book</div>
        <div className='worm-market-detail__book-grid'>
            <div>
                <div className='worm-market-detail__book-side'>Bid</div>
                {(book?.bid || []).length === 0 ? (
                    <div className='worm-market-detail__empty-line'>-</div>
                ) : (
                    (book?.bid || []).map((level, index) => (
                        <div className='worm-market-detail__book-row' key={`bid-${index}`}>
                            <span>{renderValue(level.price)}</span>
                            <strong>{renderValue(level.totalAmount)}</strong>
                        </div>
                    ))
                )}
            </div>
            <div>
                <div className='worm-market-detail__book-side'>Ask</div>
                {(book?.ask || []).length === 0 ? (
                    <div className='worm-market-detail__empty-line'>-</div>
                ) : (
                    (book?.ask || []).map((level, index) => (
                        <div className='worm-market-detail__book-row' key={`ask-${index}`}>
                            <span>{renderValue(level.price)}</span>
                            <strong>{renderValue(level.totalAmount)}</strong>
                        </div>
                    ))
                )}
            </div>
        </div>
    </div>
);

export const WormMarketDetailPanel = ({conditionId, initialMarket}: {conditionId: string; initialMarket?: WormMarketItem}) => {
    const [detail, setDetail] = React.useState<WormMarketDetail | null>(null);
    const [loading, setLoading] = React.useState(true);
    const [error, setError] = React.useState<Error | null>(null);
    const requestRef = React.useRef<{abort?: () => void} | null>(null);
    const mountedRef = React.useRef(false);

    const loadDetail = React.useCallback(async () => {
        if (!conditionId || requestRef.current) {
            return;
        }
        try {
            const req = services.worm.getMarket(conditionId);
            requestRef.current = req;
            const data = await req;
            if (mountedRef.current) {
                setDetail(data);
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
            }
        }
    }, [conditionId]);

    React.useEffect(() => {
        mountedRef.current = true;
        setDetail(null);
        setError(null);
        setLoading(true);
        loadDetail();
        return () => {
            mountedRef.current = false;
            if (requestRef.current?.abort) {
                requestRef.current.abort();
            }
        };
    }, [loadDetail]);

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
                interval = window.setInterval(loadDetail, POLL_INTERVAL_MS);
            }
        };
        const handleVisibilityChange = () => {
            if (document.hidden) {
                stopPolling();
                abortCurrentRequest();
                return;
            }
            loadDetail();
            startPolling();
        };

        startPolling();
        document.addEventListener('visibilitychange', handleVisibilityChange);
        return () => {
            document.removeEventListener('visibilitychange', handleVisibilityChange);
            stopPolling();
            abortCurrentRequest();
        };
    }, [loadDetail]);

    const market = detail?.market || initialMarket;
    const yesPrice = detail?.prices.find(item => item.isYes);
    const noPrice = detail?.prices.find(item => !item.isYes);
    const yesBook = detail?.orderBooks.find(item => item.isYes);
    const noBook = detail?.orderBooks.find(item => !item.isYes);
    const config = detail?.config || {};

    return (
        <div className='worm-market-detail'>
            <div className='worm-market-detail__header'>
                <DetailLogo market={market} />
                <div className='worm-market-detail__header-main'>
                    <div className='worm-market-detail__title' title={market?.title}>
                        {renderValue(market?.title)}
                    </div>
                    <div className='worm-market-detail__event' title={market?.eventTitle || market?.eventConditionId}>
                        {renderValue(market?.eventTitle || market?.eventConditionId)}
                    </div>
                    <div className='worm-market-detail__condition' title={conditionId}>
                        {conditionId}
                    </div>
                </div>
            </div>

            {error && (
                <div className='worm-market-detail__error'>
                    <i className='fa fa-exclamation-triangle' /> Failed to load market detail: {error.message}
                </div>
            )}

            {loading && !detail ? (
                <MockupList height={70} marginTop={20} />
            ) : (
                <React.Fragment>
                    <div className='worm-market-detail__metrics'>
                        <DetailMetric label='Last Price' value={market?.lastTradePrice} />
                        <DetailMetric label='State' value={market?.state} />
                        <DetailMetric label='Margin' value={market?.marginEnabled ? 'Yes' : 'No'} />
                        <DetailMetric label='Created' value={renderTimestamp(market?.created)} />
                    </div>

                    <section className='worm-market-detail__section'>
                        <h3>Market</h3>
                        <p>{renderValue(market?.description || detail?.rules.join(' '))}</p>
                        <div className='worm-market-detail__split'>
                            <DetailMetric label='Resolution' value={renderTimestamp(detail?.resolutionDate)} />
                            <DetailMetric label='Maker Fee' value={detail?.makerFee} />
                            <DetailMetric label='Taker Fee' value={detail?.takerFee} />
                        </div>
                    </section>

                    <section className='worm-market-detail__section'>
                        <h3>Outcomes</h3>
                        <div className='worm-market-detail__outcomes'>
                            {(detail?.outcomes || []).length === 0 ? (
                                <div className='worm-market-detail__empty-line'>-</div>
                            ) : (
                                (detail?.outcomes || []).map(outcome => (
                                    <div className='worm-market-detail__outcome' key={`${outcome.isYes}-${outcome.text}`}>
                                        <span>{outcome.isYes ? 'YES' : 'NO'}</span>
                                        <strong>{renderValue(outcome.text)}</strong>
                                    </div>
                                ))
                            )}
                        </div>
                    </section>

                    <section className='worm-market-detail__section'>
                        <h3>Stats</h3>
                        <div className='worm-market-detail__metrics worm-market-detail__metrics--compact'>
                            <DetailMetric label='Volume' value={detail?.stats.totalVolume} />
                            <DetailMetric label='24H Volume' value={detail?.stats.totalVolume24H} />
                            <DetailMetric label='Market Cap' value={detail?.stats.marketCap} />
                            <DetailMetric label='Trades' value={detail?.stats.tradeCount} />
                        </div>
                    </section>

                    <section className='worm-market-detail__section'>
                        <h3>Prices</h3>
                        <div className='worm-market-detail__metrics worm-market-detail__metrics--compact'>
                            <PriceMetric isYes={true} price={yesPrice} detail={detail || undefined} />
                            <PriceMetric isYes={false} price={noPrice} detail={detail || undefined} />
                        </div>
                    </section>

                    <section className='worm-market-detail__section'>
                        <h3>Order Books</h3>
                        <div className='worm-market-detail__books'>
                            <OrderBookTable isYes={true} book={yesBook} detail={detail || undefined} />
                            <OrderBookTable isYes={false} book={noBook} detail={detail || undefined} />
                        </div>
                    </section>

                    <section className='worm-market-detail__section'>
                        <h3>Config</h3>
                        <div className='worm-market-detail__config'>
                            <DetailMetric label='Kind' value={config.kind} />
                            <DetailMetric label='Max Leverage' value={config.maxLeverage} />
                            <DetailMetric label='Order Min Size' value={config.orderMinSize} />
                            <DetailMetric label='Opening Fee' value={config.openingFee} />
                            <DetailMetric label='Closing Fee' value={config.closingFee} />
                            <DetailMetric label='Annual Fee' value={config.annualFeeRate} />
                            <DetailMetric label='Min Price' value={config.minPrice} />
                            <DetailMetric label='Max Price' value={config.maxPrice} />
                            <DetailMetric label='Min Amount' value={config.minAmount} />
                            <DetailMetric label='Max Amount' value={config.maxAmount} />
                            <DetailMetric label='Min Funds' value={config.minFunds} />
                            <DetailMetric label='Max Funds' value={config.maxFunds} />
                        </div>
                    </section>
                </React.Fragment>
            )}
        </div>
    );
};
