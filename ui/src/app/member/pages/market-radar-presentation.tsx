import * as React from 'react';
import {TruncatedText} from '../../components';
import {formatBeijingDateTime, formatBeijingUnixSeconds} from '../../shared/format';
import {MarketRadarHotMarketItem, MarketRadarRealtimeMarketItem, MarketRadarMoverMarketItem, MarketRadarRealtimeTokenItem} from '../../shared/services/market-radar-service';

export const MarketVolume = ({value}: {value: number | undefined}) => <span className='athena-number'>{value === undefined ? 'Unavailable' : value.toLocaleString('en-US')}</span>;
const number = (value?: number) => (value === undefined ? 'Unavailable' : String(value));
const price = (value?: number) => (value === undefined ? 'Unavailable' : `${Number((value * 100).toPrecision(15))}%`);
type Market = MarketRadarHotMarketItem | MarketRadarRealtimeMarketItem | MarketRadarMoverMarketItem;
const Metric = ({label, children, numeric = false}: {label: string; children: React.ReactNode; numeric?: boolean}) => (
    <div className={`radar-metric${numeric ? ' athena-numeric-column' : ''}`}>
        <dt>{label}</dt>
        <dd className='athena-number'>{children}</dd>
    </div>
);
export const MarketHeading = ({item}: {item: Market}) => (
    <div className='radar-heading'>
        {item.image && (
            <img
                src={item.image}
                alt=''
                loading='lazy'
                decoding='async'
                onError={event => {
                    event.currentTarget.style.display = 'none';
                }}
            />
        )}
        <strong>{item.question || 'Unknown market'}</strong>
        <span>{item.eventSlug || 'Unknown event'}</span>
    </div>
);
export const MarketFacts = ({item}: {item: Market}) => (
    <details className='radar-facts'>
        <summary>Market facts &amp; identifiers</summary>
        <dl>
            <Metric label='Condition ID'>
                <TruncatedText value={item.conditionId} copyable />
            </Metric>
            <Metric label='Market slug'>
                <TruncatedText value={item.marketSlug} copyable />
            </Metric>
            <Metric label='Volume total'>
                <MarketVolume value={item.volumeNum} />
            </Metric>
            <Metric label='Event slug'>
                <TruncatedText value={item.eventSlug || 'Unknown'} copyable />
            </Metric>
            {'bestBid' in item && (
                <>
                    <Metric label='Best bid · 0–1 price'>{number(item.bestBid)}</Metric>
                    <Metric label='Best ask · 0–1 price'>{number(item.bestAsk)}</Metric>
                    <Metric label='Last trade · 0–1 price'>{number(item.lastTradePrice)}</Metric>
                </>
            )}
            {item.tokens.map(token => (
                <React.Fragment key={token.tokenId}>
                    <Metric label={`${token.outcome || 'Unknown outcome'} token`}>
                        <TruncatedText value={token.tokenId || 'Unknown'} copyable />
                    </Metric>
                    <Metric label={`${token.outcome} price · 0–1`}>{number(token.price)}</Metric>
                    {'windows' in token && (
                        <>
                            <Metric label='Best bid'>{number(token.bestBid)}</Metric>
                            <Metric label='Best ask'>{number(token.bestAsk)}</Metric>
                            <Metric label='Spread'>{number(token.spread)}</Metric>
                            <Metric label='Last trade price'>{number(token.lastTradePrice)}</Metric>
                            <Metric label='Last trade size'>{number(token.lastTradeSize)}</Metric>
                            <Metric label='Last trade side'>{token.lastTradeSide || 'Unknown'}</Metric>
                            <Metric label='Last sampled · UTC+8'>{formatBeijingUnixSeconds(token.lastEventAt) || 'Unavailable'}</Metric>
                            <Metric label='Window state'>{token.warmup === undefined ? 'Unknown' : token.warmup ? 'Warming up' : 'Ready'}</Metric>
                            {token.windows.map(window => (
                                <React.Fragment key={window.window}>
                                    <Metric label={`${window.window} change · pp`}>{window.warmup ? 'Warmup' : number(window.priceChangePp)}</Metric>
                                    <Metric label={`${window.window} samples`}>{number(window.sampleCount)}</Metric>
                                </React.Fragment>
                            ))}
                            {'score' in token && <Metric label='Outcome score'>{number(token.score)}</Metric>}
                            {'direction' in token && <Metric label='Outcome direction'>{token.direction || 'Unknown'}</Metric>}
                        </>
                    )}
                </React.Fragment>
            ))}
        </dl>
    </details>
);
const Updated = ({item}: {item: Market}) => <span className='radar-updated'>Updated {formatBeijingDateTime(item.updatedAt) || 'Unavailable'} · UTC+8</span>;
const Activity = ({item}: {item: Market}) => (
    <dl className='radar-activity'>
        <Metric numeric label='Volume 24h'>
            <MarketVolume value={item.volume24hr} />
        </Metric>
        <Metric numeric label='Liquidity'>
            <MarketVolume value={item.liquidityNum} />
        </Metric>
    </dl>
);
export const PriceWindows = ({tokens}: {tokens: MarketRadarRealtimeTokenItem[]}) => (
    <div className='radar-windows' role='region' aria-label='Outcome price windows'>
        <table>
            <thead>
                <tr>
                    <th>Outcome</th>
                    <th>Price</th>
                    {['1m', '5m', '15m'].map(window => (
                        <th key={window}>{window} · pp</th>
                    ))}
                </tr>
            </thead>
            <tbody>
                {tokens.map(token => (
                    <tr key={token.tokenId}>
                        <th scope='row'>{token.outcome || 'Unknown'}</th>
                        <td className='athena-number'>{price(token.price)}</td>
                        {['1m', '5m', '15m'].map(window => {
                            const point = token.windows.find(item => item.window === window);
                            const value = point?.priceChangePp;
                            const warmup = point?.warmup === true;
                            const tone = warmup ? 'radar-warmup' : value !== undefined && value > 0 ? 'radar-up' : value !== undefined && value < 0 ? 'radar-down' : '';
                            return (
                                <td key={window} className={`athena-number ${tone}`} title={point?.sampleCount === undefined ? undefined : `${point.sampleCount} samples`}>
                                    {warmup ? 'Warmup' : value === undefined ? 'Unavailable' : `${value > 0 ? '+' : ''}${value} pp`}
                                </td>
                            );
                        })}
                    </tr>
                ))}
            </tbody>
        </table>
    </div>
);
export const HotMarketRecord = ({item}: {item: MarketRadarHotMarketItem}) => (
    <div className='radar-record'>
        <div className='radar-record__main'>
            <MarketHeading item={item} />
            <dl className='radar-hot-metrics'>
                <Metric numeric label='Volume 24h'>
                    <MarketVolume value={item.volume24hr} />
                </Metric>
                <Metric numeric label='Liquidity'>
                    <MarketVolume value={item.liquidityNum} />
                </Metric>
                <Metric numeric label='Spread'>
                    {number(item.spread)}
                </Metric>
            </dl>
        </div>
        <div className='radar-record__meta'>
            <div className='radar-outcomes'>
                {item.tokens.map(token => (
                    <span key={token.tokenId}>
                        {token.outcome} <b className='athena-number'>{price(token.price)}</b>
                    </span>
                ))}
            </div>
            <Updated item={item} />
        </div>
        <MarketFacts item={item} />
    </div>
);
export const RealtimeMarketRecord = ({item}: {item: MarketRadarRealtimeMarketItem}) => (
    <div className='radar-record'>
        <div className='radar-record__main'>
            <MarketHeading item={item} />
            <PriceWindows tokens={item.tokens} />
        </div>
        <div className='radar-record__meta'>
            <Activity item={item} />
            <Updated item={item} />
        </div>
        <MarketFacts item={item} />
    </div>
);
export const MoverMarketRecord = ({item}: {item: MarketRadarMoverMarketItem}) => (
    <div className='radar-record'>
        <div className='radar-record__main'>
            <MarketHeading item={item} />
            <div>
                <dl className='radar-hot-metrics'>
                    <Metric label='Leading outcome'>{item.leader?.outcome || 'Unknown'}</Metric>
                    <Metric numeric label='Mover score'>
                        {number(item.score)}
                    </Metric>
                    <Metric label='Direction'>
                        <span className={item.direction?.toLowerCase() === 'up' ? 'radar-up' : item.direction?.toLowerCase() === 'down' ? 'radar-down' : ''}>
                            {item.direction || 'Unknown'}
                        </span>
                    </Metric>
                </dl>
                {item.leader && <PriceWindows tokens={[item.leader]} />}
            </div>
        </div>
        <div className='radar-record__meta'>
            <Activity item={item} />
            <Updated item={item} />
        </div>
        <MarketFacts item={item} />
    </div>
);
