import {LinkOutlined} from '@ant-design/icons';
import {Button, Card, Col, Empty, Input, Row, Space, Tag, Typography} from 'antd';
import * as React from 'react';
import {useSearchParams} from 'react-router-dom';
import {AppPage, CardTitle, KeyValueGrid, MetricRow, TruncatedText} from '../components';
import {services} from '../../shared/services';
import {PolymarketFIFAMoneylineEventItem, PolymarketFIFAMoneylineOptionItem, PolymarketFIFAWalletBalanceItem} from '../../shared/services/polymarket-service';
import {boolTag, fmt, fmtNumber} from './shared';

const defaultEventRef = '351731';
const walletRefreshIntervalMs = 10000;

const walletBalancePlaceholders: PolymarketFIFAWalletBalanceItem[] = [
    {
        chain: 'polygon',
        label: 'Polygon pUSD',
        walletAddress: '0xaff389b0c6e066057c44c25fae7277880b276ecc',
        tokenAddress: '0xc011a7e12a19f7b1f670d46f03b03f3342e82dfb',
        tokenSymbol: 'pUSD',
        rawAmount: '-',
        amount: '-',
        explorerUrl: 'https://polygonscan.com/token/0xc011a7e12a19f7b1f670d46f03b03f3342e82dfb?a=0xaff389b0c6e066057c44c25fae7277880b276ecc#transactions'
    },
    {
        chain: 'solana',
        label: 'Solana USDC',
        walletAddress: 'HhpThriqRFyYr7fA8PT5ArV4D32uitzLx7HCNCh4SXjH',
        tokenAddress: 'EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v',
        tokenSymbol: 'USDC',
        rawAmount: '-',
        amount: '-',
        explorerUrl: 'https://explorer.solana.com/address/HhpThriqRFyYr7fA8PT5ArV4D32uitzLx7HCNCh4SXjH'
    }
];

const price = (value?: number) => (value === undefined ? '-' : value.toFixed(3));
const balanceValue = (item: PolymarketFIFAWalletBalanceItem) => (item.amount && item.amount !== '-' ? `${item.amount} ${item.tokenSymbol || ''}`.trim() : '-');

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

const FIFAWalletBalanceCard = (props: {item: PolymarketFIFAWalletBalanceItem; loading?: boolean}) => {
    const item = props.item;
    const statusText = props.loading ? 'Refreshing' : item.ok ? 'Live' : item.errorMessage ? 'Error' : 'Pending';
    const statusColor = props.loading ? 'blue' : item.ok ? 'green' : item.errorMessage ? 'red' : 'default';
    return (
        <Card className='fifa-wallet-balance' size='small'>
            <div className='fifa-wallet-balance__header'>
                <Typography.Title level={5}>{item.label}</Typography.Title>
                <Tag color={statusColor}>{statusText}</Tag>
            </div>
            <div className='fifa-wallet-balance__amount'>{balanceValue(item)}</div>
            <KeyValueGrid
                columns={1}
                items={[
                    {label: 'Wallet', value: <TruncatedText value={item.walletAddress} copyable={true} />},
                    {label: 'Token', value: <TruncatedText value={item.tokenAddress} copyable={true} />},
                    {label: 'Raw', value: <TruncatedText value={item.rawAmount} copyable={true} />}
                ]}
            />
            {item.errorMessage && <Typography.Text type='danger'>{item.errorMessage}</Typography.Text>}
            <Button href={item.explorerUrl} target='_blank' rel='noreferrer' icon={<LinkOutlined />}>
                Explorer
            </Button>
        </Card>
    );
};

const FIFAWalletBalancesBar = (props: {items?: PolymarketFIFAWalletBalanceItem[]; fetchedAt?: number; loading?: boolean; error?: Error}) => {
    const itemsByChain = new Map((props.items || []).map(item => [item.chain, item]));
    const items = walletBalancePlaceholders.map(placeholder => ({...placeholder, ...(itemsByChain.get(placeholder.chain) || {})}));
    return (
        <section className='fifa-wallet-balances'>
            <div className='fifa-wallet-balances__title'>
                <Typography.Text strong={true}>Wallet Balances</Typography.Text>
                <Typography.Text type='secondary'>Fetched {fmt(props.fetchedAt)}</Typography.Text>
            </div>
            <Row gutter={[12, 12]}>
                {items.map(item => (
                    <Col key={item.chain} xs={24} md={12}>
                        <FIFAWalletBalanceCard item={item} loading={props.loading} />
                    </Col>
                ))}
            </Row>
            {props.error && <Typography.Text type='danger'>{props.error.message}</Typography.Text>}
        </section>
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
    const [balancesLoading, setBalancesLoading] = React.useState(false);
    const [balancesError, setBalancesError] = React.useState<Error>(null);
    const [balancesData, setBalancesData] = React.useState<{items?: PolymarketFIFAWalletBalanceItem[]; fetchedAt?: number}>(null);
    const balancesRequestRef = React.useRef<{abort?: () => void}>(null);
    const balancesMountedRef = React.useRef(true);

    const loadBalances = React.useCallback(() => {
        if (balancesRequestRef.current) {
            return;
        }
        setBalancesLoading(true);
        setBalancesError(null);
        const req = services.polymarket.listFIFAWalletBalances();
        balancesRequestRef.current = req;
        req.then(nextData => {
            if (balancesMountedRef.current) {
                setBalancesData(nextData);
            }
        })
            .catch(err => {
                if (balancesMountedRef.current) {
                    setBalancesError(err instanceof Error ? err : new Error(String(err)));
                }
            })
            .finally(() => {
                if (balancesRequestRef.current === req) {
                    balancesRequestRef.current = null;
                    if (balancesMountedRef.current) {
                        setBalancesLoading(false);
                    }
                }
            });
    }, []);

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

    React.useEffect(() => {
        balancesMountedRef.current = true;
        loadBalances();
        const timer = window.setInterval(loadBalances, walletRefreshIntervalMs);
        return () => {
            balancesMountedRef.current = false;
            window.clearInterval(timer);
            balancesRequestRef.current?.abort?.();
        };
    }, [loadBalances]);

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
            <div className='fifa-page'>
                <FIFAWalletBalancesBar items={balancesData?.items} fetchedAt={balancesData?.fetchedAt} loading={balancesLoading} error={balancesError} />
                {!item && !loading && <Empty description='No event loaded' />}
                {item && (
                    <>
                        <FIFAEventSummary item={item} />
                        <Row className='fifa-moneyline-options' gutter={[12, 12]}>
                            {item.options.map(option => (
                                <Col key={option.outcomeKey} xs={24} md={8}>
                                    <FIFAMoneylineOptionCard option={option} />
                                </Col>
                            ))}
                        </Row>
                    </>
                )}
            </div>
        </AppPage>
    );
};
