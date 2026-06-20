import {CopyOutlined, LinkOutlined} from '@ant-design/icons';
import {Button, Card, Col, Empty, Input, Row, Space, Tag, Typography} from 'antd';
import * as React from 'react';
import {useSearchParams} from 'react-router-dom';
import {AppPage, CardTitle, MetricRow, TruncatedText} from '../components';
import {services} from '../../shared/services';
import {PolymarketFIFAMoneylineEventItem, PolymarketFIFAMoneylineOptionItem, PolymarketFIFAWalletBalanceItem} from '../../shared/services/polymarket-service';
import {GetWormEventResult, WormMarketItem} from '../../shared/services/worm-service';
import {boolTag, fmt, fmtNumber} from './shared';

const defaultEventRef = '351731';
const defaultWormEventID = '87UM8qJ3BwL9ZJA4HvqtcTLgMtipBxgwPU29V3LWkD89';
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
const unixTime = (value?: number) => (value ? new Date(value * 1000).toLocaleString() : '-');
const wormMarketURL = (conditionId: string) => `https://www.worm.wtf/market/${encodeURIComponent(conditionId)}`;
type FIFAInfoGridItem = {label: React.ReactNode; value: React.ReactNode; copyText?: string};

const copyText = (value?: string) => {
    if (!value || !navigator.clipboard) {
        return;
    }
    navigator.clipboard.writeText(value).catch(() => undefined);
};

const FIFAInfoValue = (props: {value: React.ReactNode; copyText?: string}) => (
    <span className='fifa-info-grid__value-wrap' title={typeof props.value === 'string' ? props.value : undefined}>
        <span className='fifa-info-grid__value-text'>{props.value ?? '-'}</span>
        {props.copyText && (
            <Button aria-label='Copy value' className='fifa-info-grid__copy' icon={<CopyOutlined />} size='small' type='text' onClick={() => copyText(props.copyText)} />
        )}
    </span>
);

const FIFAInfoGrid = (props: {items: FIFAInfoGridItem[]; columns?: 1 | 2 | 3 | 4}) => (
    <div className={`fifa-info-grid fifa-info-grid--cols-${props.columns || 2}`}>
        {props.items.map(item => (
            <div className='fifa-info-grid__item' key={String(item.label)}>
                <span className='fifa-info-grid__label'>{item.label}</span>
                <span className='fifa-info-grid__value'>
                    <FIFAInfoValue value={item.value} copyText={item.copyText} />
                </span>
            </div>
        ))}
    </div>
);

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
            <FIFAInfoGrid
                columns={1}
                items={[
                    {label: 'Market', value: option.marketSlug, copyText: option.marketSlug},
                    {label: 'Condition', value: option.conditionId, copyText: option.conditionId},
                    {label: 'Yes Token', value: option.yesTokenId, copyText: option.yesTokenId},
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
            <FIFAInfoGrid
                columns={1}
                items={[
                    {label: 'Wallet', value: item.walletAddress, copyText: item.walletAddress},
                    {label: 'Token', value: item.tokenAddress, copyText: item.tokenAddress},
                    {label: 'Raw', value: item.rawAmount, copyText: item.rawAmount}
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

const WormMarketCard = (props: {item: WormMarketItem}) => {
    const item = props.item;
    return (
        <Card className='fifa-worm-market' size='small'>
            <div className='fifa-worm-market__header'>
                <Typography.Title level={5}>{item.title || '-'}</Typography.Title>
                <Tag color={item.state === 'open' ? 'green' : 'default'}>{item.state || '-'}</Tag>
            </div>
            <MetricRow
                items={[
                    {label: 'Price', value: item.lastTradePrice || '-'},
                    {label: 'Margin', value: item.marginEnabled ? 'Yes' : 'No'},
                    {label: 'Created', value: unixTime(item.created)}
                ]}
            />
            <FIFAInfoGrid
                columns={1}
                items={[
                    {label: 'Condition', value: item.conditionId, copyText: item.conditionId},
                    {label: 'Category', value: fmt(item.category)},
                    {label: 'Event', value: item.eventConditionId, copyText: item.eventConditionId}
                ]}
            />
            {item.description && <Typography.Paragraph className='fifa-worm-market__description'>{item.description}</Typography.Paragraph>}
            <Button href={wormMarketURL(item.conditionId)} target='_blank' rel='noreferrer' icon={<LinkOutlined />}>
                Worm
            </Button>
        </Card>
    );
};

const WormEventPanel = (props: {data?: GetWormEventResult; loading?: boolean; error?: Error}) => {
    const item = props.data?.item;
    return (
        <section className='fifa-panel fifa-panel--worm'>
            <div className='fifa-panel__title'>
                <Typography.Text strong={true}>Worm Event</Typography.Text>
                <Typography.Text type='secondary'>Fetched {unixTime(props.data?.fetchedAt)}</Typography.Text>
            </div>
            {props.error && <Typography.Text type='danger'>{props.error.message}</Typography.Text>}
            {!item && !props.loading && <Empty description='No Worm event loaded' />}
            {item && (
                <>
                    <section className='fifa-event-summary'>
                        <div className='fifa-event-summary__title'>
                            <CardTitle title={item.title} subtitle={<TruncatedText value={item.conditionId} copyable={true} />} image={item.logo} />
                        </div>
                        <FIFAInfoGrid
                            items={[
                                {label: 'Event ID', value: item.conditionId, copyText: item.conditionId},
                                {label: 'Category', value: fmt(item.category)},
                                {label: 'Created', value: unixTime(item.created)},
                                {label: 'Markets', value: fmt(item.marketCount || item.markets.length)}
                            ]}
                        />
                        {item.description && <Typography.Paragraph className='fifa-worm-event-description'>{item.description}</Typography.Paragraph>}
                    </section>
                    <Row className='fifa-worm-markets' gutter={[12, 12]}>
                        {item.markets.map(market => (
                            <Col key={market.conditionId} xs={24} xxl={12}>
                                <WormMarketCard item={market} />
                            </Col>
                        ))}
                    </Row>
                </>
            )}
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
            <FIFAInfoGrid
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
    const initialWormEventID = params.get('worm_event_id') || '';
    const [eventRef, setEventRef] = React.useState(initialRef);
    const [wormEventID, setWormEventID] = React.useState(initialWormEventID);
    const [loading, setLoading] = React.useState(false);
    const [error, setError] = React.useState<Error>(null);
    const [data, setData] = React.useState<{item?: PolymarketFIFAMoneylineEventItem; fetchedAt?: number}>(null);
    const [wormLoading, setWormLoading] = React.useState(false);
    const [wormError, setWormError] = React.useState<Error>(null);
    const [wormData, setWormData] = React.useState<GetWormEventResult>(null);
    const [balancesLoading, setBalancesLoading] = React.useState(false);
    const [balancesError, setBalancesError] = React.useState<Error>(null);
    const [balancesData, setBalancesData] = React.useState<{items?: PolymarketFIFAWalletBalanceItem[]; fetchedAt?: number}>(null);
    const wormRequestRef = React.useRef<(Promise<GetWormEventResult> & {abort?: () => void}) | null>(null);
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

    const loadWorm = React.useCallback(
        (nextID = wormEventID) => {
            const normalized = nextID.trim();
            wormRequestRef.current?.abort?.();
            if (!normalized) {
                setWormData(null);
                setWormError(null);
                setWormLoading(false);
                return;
            }
            const nextParams = new URLSearchParams(params);
            nextParams.set('worm_event_id', normalized);
            setParams(nextParams);
            setWormLoading(true);
            setWormError(null);
            const req = services.worm.getEvent(normalized);
            wormRequestRef.current = req;
            req.then(nextData => {
                if (wormRequestRef.current === req) {
                    setWormData(nextData);
                }
            })
                .catch(err => {
                    if (wormRequestRef.current === req) {
                        setWormError(err instanceof Error ? err : new Error(String(err)));
                    }
                })
                .finally(() => {
                    if (wormRequestRef.current === req) {
                        wormRequestRef.current = null;
                        setWormLoading(false);
                    }
                });
        },
        [wormEventID, params, setParams]
    );

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
        if (initialWormEventID) {
            loadWorm(initialWormEventID);
        }
        return () => {
            wormRequestRef.current?.abort?.();
        };
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
        <Space className='fifa-query-stack' orientation='vertical' size={10}>
            <Space.Compact className='fifa-query'>
                <Input
                    allowClear={true}
                    value={wormEventID}
                    placeholder={`Worm Event ID, e.g. ${defaultWormEventID}`}
                    onChange={event => setWormEventID(event.target.value)}
                    onPressEnter={() => loadWorm()}
                />
                <Button type='primary' onClick={() => loadWorm()} loading={wormLoading}>
                    Query
                </Button>
            </Space.Compact>
            <Space.Compact className='fifa-query'>
                <Input
                    allowClear={true}
                    value={eventRef}
                    placeholder={`Polymarket Event ID / Slug, e.g. ${defaultEventRef}`}
                    onChange={event => setEventRef(event.target.value)}
                    onPressEnter={() => load()}
                />
                <Button onClick={() => load()} loading={loading}>
                    Query
                </Button>
            </Space.Compact>
        </Space>
    );

    const item = data?.item;
    const refresh =
        wormData?.item || item
            ? () => {
                  if (wormEventID.trim()) {
                      loadWorm();
                  }
                  if (eventRef.trim()) {
                      load();
                  }
                  loadBalances();
              }
            : undefined;
    return (
        <AppPage
            title='FIFA'
            subtitle={`Worm ${unixTime(wormData?.fetchedAt)} · Polymarket ${unixTime(data?.fetchedAt)}`}
            filters={filters}
            loading={loading || wormLoading}
            error={error}
            onRefresh={refresh}>
            <div className='fifa-page'>
                <div className='fifa-dashboard-grid'>
                    <WormEventPanel data={wormData} loading={wormLoading} error={wormError} />
                    <section className='fifa-panel fifa-panel--polymarket'>
                        <div className='fifa-panel__title'>
                            <Typography.Text strong={true}>Polymarket</Typography.Text>
                            <Typography.Text type='secondary'>Fetched {unixTime(data?.fetchedAt)}</Typography.Text>
                        </div>
                        <FIFAWalletBalancesBar items={balancesData?.items} fetchedAt={balancesData?.fetchedAt} loading={balancesLoading} error={balancesError} />
                        {!item && !loading && <Empty description='No Polymarket event loaded' />}
                        {item && (
                            <>
                                <FIFAEventSummary item={item} />
                                <Row className='fifa-moneyline-options' gutter={[12, 12]}>
                                    {item.options.map(option => (
                                        <Col key={option.outcomeKey} xs={24} xl={12} xxl={8}>
                                            <FIFAMoneylineOptionCard option={option} />
                                        </Col>
                                    ))}
                                </Row>
                            </>
                        )}
                    </section>
                </div>
            </div>
        </AppPage>
    );
};
