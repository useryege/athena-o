import {LinkOutlined} from '@ant-design/icons';
import {Button, Card, Col, Empty, Input, Row, Space, Tag, Typography} from 'antd';
import * as React from 'react';
import {useSearchParams} from 'react-router-dom';
import {AppPage, CardTitle, KeyValueGrid, MetricRow, TruncatedText} from '../components';
import {services} from '../../shared/services';
import {PolymarketFIFAMoneylineEventItem, PolymarketFIFAMoneylineOptionItem} from '../../shared/services/polymarket-service';
import {boolTag, fmt, fmtNumber} from './shared';

const defaultEventRef = '351731';

const price = (value?: number) => (value === undefined ? '-' : value.toFixed(3));

const outcomeTone = (option: PolymarketFIFAMoneylineOptionItem) => {
    if (!option.enableOrderBook || !option.acceptingOrders) {
        return 'red';
    }
    return 'green';
};

const optionTitle = (option: PolymarketFIFAMoneylineOptionItem) => {
    if (option.outcomeKey === 'draw') {
        return 'Draw';
    }
    return option.outcomeLabel || option.outcomeKey;
};

const FIFAMoneylineOptionCard = (props: {option: PolymarketFIFAMoneylineOptionItem}) => {
    const option = props.option;
    return (
        <Card className='fifa-moneyline-option' size='small'>
            <div className='fifa-moneyline-option__header'>
                <Typography.Title level={5}>{optionTitle(option)}</Typography.Title>
                <Tag color={outcomeTone(option)}>{option.enableOrderBook && option.acceptingOrders ? 'Open' : 'Unavailable'}</Tag>
            </div>
            <MetricRow
                items={[
                    {label: 'Mid', value: price(option.midPrice)},
                    {label: 'Bid', value: price(option.bestBid)},
                    {label: 'Ask', value: price(option.bestAsk)},
                    {label: 'Spread', value: price(option.spread)}
                ]}
            />
            <KeyValueGrid
                columns={1}
                items={[
                    {label: 'Market', value: <TruncatedText value={option.marketSlug} copyable={true} />},
                    {label: 'Condition', value: <TruncatedText value={option.conditionId} copyable={true} />},
                    {label: 'Yes Token', value: <TruncatedText value={option.yesTokenId} copyable={true} />},
                    {label: 'Min Size', value: fmtNumber(option.orderMinSize)},
                    {label: 'Tick', value: price(option.tickSize)},
                    {label: 'Neg Risk', value: boolTag(option.negRisk)}
                ]}
            />
        </Card>
    );
};

const FIFAEventSummary = (props: {item: PolymarketFIFAMoneylineEventItem}) => {
    const item = props.item;
    const title = item.teams.length >= 2 ? `${item.teams[0].name} vs. ${item.teams[1].name}` : item.title;
    return (
        <section className='fifa-event-summary'>
            <div className='fifa-event-summary__title'>
                <CardTitle title={title} subtitle={item.eventSlug} image={item.image} />
                {item.polymarketUrl && (
                    <Button href={item.polymarketUrl} target='_blank' rel='noreferrer' icon={<LinkOutlined />}>
                        Polymarket
                    </Button>
                )}
            </div>
            <KeyValueGrid
                items={[
                    {label: 'Event ID', value: item.eventId},
                    {label: 'Sport', value: fmt(item.sport)},
                    {label: 'Start', value: fmt(item.startTime)},
                    {label: 'Status', value: fmt(item.gameStatus || (item.live ? 'Live' : item.ended ? 'Ended' : 'Scheduled'))},
                    {label: 'Score', value: fmt(item.score)},
                    {label: 'Updated', value: fmt(item.updatedAt)},
                    {label: 'Active', value: boolTag(item.active)},
                    {label: 'Closed', value: boolTag(item.closed)}
                ]}
            />
        </section>
    );
};

export const FIFAPage = () => {
    const [params, setParams] = useSearchParams();
    const initialRef = params.get('event_ref') || '';
    const [eventRef, setEventRef] = React.useState(initialRef);
    const [loading, setLoading] = React.useState(false);
    const [error, setError] = React.useState<Error>(null);
    const [data, setData] = React.useState<{item?: PolymarketFIFAMoneylineEventItem; fetchedAt?: number}>(null);

    const load = React.useCallback(
        (nextRef = eventRef) => {
            const normalized = nextRef.trim();
            if (!normalized) {
                setData(null);
                setError(null);
                return;
            }
            const nextParams = new URLSearchParams(params);
            nextParams.set('event_ref', normalized);
            setParams(nextParams);
            setLoading(true);
            setError(null);
            services.polymarket
                .getFIFAMoneylineEvent(normalized)
                .then(setData)
                .catch(err => setError(err instanceof Error ? err : new Error(String(err))))
                .finally(() => setLoading(false));
        },
        [eventRef, params, setParams]
    );

    React.useEffect(() => {
        if (initialRef) {
            load(initialRef);
        }
    }, []);

    const filters = (
        <Space.Compact className='fifa-query'>
            <Input
                allowClear={true}
                value={eventRef}
                placeholder={`Event ID / Slug, e.g. ${defaultEventRef}`}
                onChange={event => setEventRef(event.target.value)}
                onPressEnter={() => load()}
            />
            <Button type='primary' onClick={() => load()} loading={loading}>
                Query
            </Button>
        </Space.Compact>
    );

    const item = data?.item;
    return (
        <AppPage title='FIFA' subtitle={`Fetched ${fmt(data?.fetchedAt)}`} filters={filters} loading={loading} error={error} onRefresh={item ? () => load() : undefined}>
            {!item && !loading && <Empty description='No event loaded' />}
            {item && (
                <div className='fifa-page'>
                    <FIFAEventSummary item={item} />
                    <Row className='fifa-moneyline-options' gutter={[12, 12]}>
                        {item.options.map(option => (
                            <Col key={option.outcomeKey} xs={24} md={8}>
                                <FIFAMoneylineOptionCard option={option} />
                            </Col>
                        ))}
                    </Row>
                </div>
            )}
        </AppPage>
    );
};
