import {CardTitle, LiveStatusIndicator} from '../components';
import {WormEventItem} from '../../shared/services/worm-service';

const wormMarketURL = (conditionId: string) => `https://www.worm.wtf/market/${encodeURIComponent(conditionId)}`;

const MIN_WORM_PRICE_MOVE = 0.01;

const formatWormPriceMove = (value?: string): {text: string; direction: 'up' | 'down'} | null => {
    const num = Number(String(value ?? '').trim());
    if (!Number.isFinite(num) || Math.abs(num) < MIN_WORM_PRICE_MOVE) return null;
    const abs = String(Math.abs(num)).replace(/(\.\d*?[1-9])0+$|\.0+$/, '$1');
    return num > 0 ? {text: `+ ${abs}`, direction: 'up'} : {text: `-${abs}`, direction: 'down'};
};

export const WormEventCard = (props: {item: WormEventItem}) => {
    const item = props.item;
    return (
        <article className='worm-event-card'>
            <div className='worm-event-card__header'>
                <CardTitle
                    title={item.title}
                    image={item.logo}
                    tags={
                        <div className='worm-event-card__tags'>
                            <LiveStatusIndicator live={item.live} />
                        </div>
                    }
                />
            </div>
            <div className='worm-event-card__markets'>
                {item.markets.map(market => {
                    const move = formatWormPriceMove(market.livePriceChange);
                    return (
                        <a key={market.conditionId} className='worm-event-market' href={wormMarketURL(market.conditionId)} target='_blank' rel='noopener noreferrer'>
                            <div className='worm-event-market__title'>
                                <span>{market.title || '-'}</span>
                                <LiveStatusIndicator live={market.liveState === 'live'} />
                            </div>
                            <div className='worm-event-market__metrics'>
                                <span>
                                    Price <strong>{market.lastTradePrice || '-'}</strong>
                                    {move && <span className={`worm-event-market__move worm-event-market__move--${move.direction}`}>{move.text}</span>}
                                </span>
                            </div>
                        </a>
                    );
                })}
            </div>
        </article>
    );
};
