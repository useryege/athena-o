import {Tag} from 'antd';
import {CardTitle} from '../components';
import {WormMarketItem} from '../../shared/services/worm-service';

export const wormMarketLogo = (item: Pick<WormMarketItem, 'logo' | 'eventLogo'>) => item.logo || item.eventLogo || '';

export const WormMarketSummary = (props: {item: WormMarketItem}) => {
    const item = props.item;
    return (
        <div className='worm-market-summary'>
            <CardTitle
                title={item.title}
                subtitle={item.eventTitle || item.category}
                image={wormMarketLogo(item)}
                tags={item.marginEnabled ? <Tag color='green'>Margin</Tag> : <Tag>{item.state}</Tag>}
            />
        </div>
    );
};
