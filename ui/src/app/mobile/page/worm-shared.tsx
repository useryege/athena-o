import {DownOutlined} from '@ant-design/icons';
import * as React from 'react';
import {CardTitle, LiveStatusIndicator} from '../components';
import {WormEventItem} from '../../shared/services/worm-service';

const wormMarketURL = (conditionId: string) => `https://www.worm.wtf/market/${encodeURIComponent(conditionId)}`;

const MIN_WORM_PRICE_MOVE = 0.01;
const BEIJING_TIME_ZONE = 'Asia/Shanghai';

const beijingDateTimeFormatter = new Intl.DateTimeFormat('en-US', {
    timeZone: BEIJING_TIME_ZONE,
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hourCycle: 'h23'
});

const formatBeijingMatchStart = (markets: WormEventItem['markets']): string | undefined => {
    const matchStartAt = markets.reduce<number | undefined>((earliest, market) => {
        const value = Number(market.matchStartAt);
        if (!Number.isFinite(value) || value <= 0) return earliest;
        return earliest === undefined || value < earliest ? value : earliest;
    }, undefined);
    if (matchStartAt === undefined) return undefined;

    const parts = Object.fromEntries(beijingDateTimeFormatter.formatToParts(new Date(matchStartAt * 1000)).map(part => [part.type, part.value]));
    return `北京时间 ${parts.month}-${parts.day} ${parts.hour}:${parts.minute}`;
};

const formatWormPriceMove = (value?: string): {text: string; direction: 'up' | 'down'} | null => {
    const num = Number(String(value ?? '').trim());
    if (!Number.isFinite(num) || Math.abs(num) < MIN_WORM_PRICE_MOVE) return null;
    const abs = String(Math.abs(num)).replace(/(\.\d*?[1-9])0+$|\.0+$/, '$1');
    return num > 0 ? {text: `+ ${abs}`, direction: 'up'} : {text: `-${abs}`, direction: 'down'};
};

export const WormEventCard = (props: {item: WormEventItem}) => {
    const item = props.item;
    const [expanded, setExpanded] = React.useState(false);
    const matchStartText = formatBeijingMatchStart(item.markets);
    return (
        <article className={`worm-event-card${expanded ? ' worm-event-card--expanded' : ''}`}>
            <button type='button' className='worm-event-card__header' aria-expanded={expanded} onClick={() => setExpanded(v => !v)}>
                <CardTitle
                    title={item.title}
                    subtitle={matchStartText}
                    image={item.logo}
                    tags={
                        <div className='worm-event-card__tags'>
                            <LiveStatusIndicator live={item.live} />
                        </div>
                    }
                />
                <DownOutlined className='worm-event-card__toggle' />
            </button>
            {expanded && (
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
            )}
        </article>
    );
};
