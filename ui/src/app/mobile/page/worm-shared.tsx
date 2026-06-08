import {CardTitle} from '../components';
import {WormMarketItem} from '../../shared/services/worm-service';

export const wormMarketLogo = (item: Pick<WormMarketItem, 'logo' | 'eventLogo'>) => item.logo || item.eventLogo || '';

export const WormLiveIndicator = (props: {item: Pick<WormMarketItem, 'liveState'>}) => {
    if (props.item.liveState !== 'live') {
        return null;
    }
    return (
        <span className='worm-live-indicator'>
            <span className='worm-live-dot' />
            <span>Live</span>
        </span>
    );
};

export const WormMarketSummary = (props: {item: WormMarketItem}) => {
    const item = props.item;
    return (
        <div className='worm-market-summary'>
            <WormLiveIndicator item={item} />
            <CardTitle title={item.title} subtitle={item.eventTitle} image={wormMarketLogo(item)} />
        </div>
    );
};
