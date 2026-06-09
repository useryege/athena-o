import {Tag} from 'antd';
import {CardTitle, LiveStatusIndicator} from '../components';
import {WormEventItem} from '../../shared/services/worm-service';

const wormMarketURL = (conditionId: string) => `https://www.worm.wtf/market/${encodeURIComponent(conditionId)}`;

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
                            <Tag>{item.marketCount} Markets</Tag>
                        </div>
                    }
                />
            </div>
            <div className='worm-event-card__markets'>
                {item.markets.map(market => (
                    <a key={market.conditionId} className='worm-event-market' href={wormMarketURL(market.conditionId)} target='_blank' rel='noopener noreferrer'>
                        <div className='worm-event-market__title'>
                            <span>{market.title || '-'}</span>
                            <LiveStatusIndicator live={market.liveState === 'live'} />
                        </div>
                        <div className='worm-event-market__metrics'>
                            <span>
                                Price <strong>{market.lastTradePrice || '-'}</strong>
                            </span>
                            <span>
                                Move <strong>{market.livePriceChange || '-'}</strong>
                            </span>
                        </div>
                    </a>
                ))}
            </div>
        </article>
    );
};
