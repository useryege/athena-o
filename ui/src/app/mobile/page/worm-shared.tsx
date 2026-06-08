import {Button, Tag} from 'antd';
import {CardTitle} from '../components';
import {WormMarketItem} from '../../shared/services/worm-service';

export const wormMarketLogo = (item: Pick<WormMarketItem, 'logo' | 'eventLogo'>) => item.logo || item.eventLogo || '';

export const WormMarketSummary = (props: {item: WormMarketItem; onOpen?: (item: WormMarketItem) => void}) => {
    const item = props.item;
    const title = props.onOpen ? (
        <Button className='worm-market-summary__button' type='link' onClick={() => props.onOpen?.(item)}>
            {item.title}
        </Button>
    ) : (
        item.title
    );
    return (
        <div className='worm-market-summary'>
            <CardTitle
                title={title}
                subtitle={item.eventTitle || item.category}
                image={wormMarketLogo(item)}
                tags={item.marginEnabled ? <Tag color='green'>Margin</Tag> : <Tag>{item.state}</Tag>}
            />
        </div>
    );
};
