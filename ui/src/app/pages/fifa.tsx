import {CopyOutlined, DownOutlined, LinkOutlined} from '@ant-design/icons';
import {Button, Card, Col, Collapse, Empty, Input, Row, Tag, Typography} from 'antd';
import * as React from 'react';
import {useSearchParams} from 'react-router-dom';
import {AppPage, CardTitle, MetricRow, TruncatedText} from '../components';
import {Context} from '../shared/context';
import {services} from '../shared/services';
import {
    GetPolymarketFIFAMoneylineEventResult,
    PolymarketFIFAEventConfig,
    PolymarketFIFAMoneylineEventItem,
    PolymarketFIFAMoneylineOptionItem,
    PolymarketFIFAWalletBalanceItem
} from '../shared/services/polymarket-service';
import {GetWormEventResult, WormMarketItem} from '../shared/services/worm-service';
import {boolTag, fmt, fmtNumber} from './shared';

const walletRefreshIntervalMs = 3000;
const eventRefreshIntervalMs = 1000;

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

const normalizedWormMarketTitle = (value?: string) => (value || '').trim().toLowerCase();

const orderedWormMarkets = (eventTitle: string, markets: WormMarketItem[]) => {
    const match = eventTitle.trim().match(/^(.+?)\s+vs\.?\s+(.+)$/i);
    if (!match) {
        return [...markets];
    }
    const titleOrder = new Map([
        [normalizedWormMarketTitle(match[1]), 0],
        ['draw', 1],
        [normalizedWormMarketTitle(match[2]), 2]
    ]);
    return markets
        .map((market, index) => ({market, index, order: titleOrder.get(normalizedWormMarketTitle(market.title)) ?? 3}))
        .sort((left, right) => left.order - right.order || left.index - right.index)
        .map(({market}) => market);
};

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
            <Collapse
                bordered={false}
                className='fifa-card-collapse'
                items={[
                    {
                        key: 'details',
                        label: 'Details',
                        children: (
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
                        )
                    }
                ]}
            />
        </Card>
    );
};

const FIFAWalletBalanceCard = (props: {item: PolymarketFIFAWalletBalanceItem; loading?: boolean; detailsVisible?: boolean}) => {
    const item = props.item;
    const statusText = props.loading ? 'Refreshing' : item.ok ? 'Live' : item.errorMessage ? 'Error' : 'Pending';
    const statusColor = props.loading ? 'blue' : item.ok ? 'green' : item.errorMessage ? 'red' : 'default';
    return (
        <Card className='fifa-wallet-balance' size='small'>
            <div className='fifa-wallet-balance__header'>
                <Typography.Title level={5}>{item.label}</Typography.Title>
                {props.detailsVisible && <Tag color={statusColor}>{statusText}</Tag>}
            </div>
            <div className='fifa-wallet-balance__amount'>{balanceValue(item)}</div>
            {item.errorMessage && <Typography.Text type='danger'>{item.errorMessage}</Typography.Text>}
            {props.detailsVisible && (
                <>
                    <FIFAInfoGrid
                        columns={1}
                        items={[
                            {label: 'Wallet', value: item.walletAddress, copyText: item.walletAddress},
                            {label: 'Token', value: item.tokenAddress, copyText: item.tokenAddress},
                            {label: 'Raw', value: item.rawAmount, copyText: item.rawAmount}
                        ]}
                    />
                    <Button href={item.explorerUrl} target='_blank' rel='noreferrer' icon={<LinkOutlined />}>
                        Explorer
                    </Button>
                </>
            )}
        </Card>
    );
};

const FIFAWalletBalancesPanel = (props: {items?: PolymarketFIFAWalletBalanceItem[]; fetchedAt?: number; loading?: boolean; error?: Error}) => {
    const [detailsVisible, setDetailsVisible] = React.useState(false);
    const itemsByChain = new Map((props.items || []).map(item => [item.chain, item]));
    const items = walletBalancePlaceholders.map(placeholder => ({...placeholder, ...(itemsByChain.get(placeholder.chain) || {})}));
    return (
        <section className='fifa-panel fifa-wallet-balances'>
            <div className='fifa-wallet-balances__title'>
                <Typography.Text strong={true}>Wallet Balances</Typography.Text>
                <Button
                    aria-controls='fifa-wallet-balance-details'
                    aria-expanded={detailsVisible}
                    className={`fifa-wallet-balances__toggle${detailsVisible ? ' fifa-wallet-balances__toggle--expanded' : ''}`}
                    size='small'
                    type='text'
                    onClick={() => setDetailsVisible(visible => !visible)}>
                    <span>{detailsVisible ? 'Hide details' : 'Show details'}</span>
                    <DownOutlined />
                </Button>
            </div>
            <Row id='fifa-wallet-balance-details' gutter={[12, 12]}>
                {items.map(item => (
                    <Col key={item.chain} span={12}>
                        <FIFAWalletBalanceCard item={item} loading={props.loading} detailsVisible={detailsVisible} />
                    </Col>
                ))}
            </Row>
            {detailsVisible && <Typography.Text type='secondary'>Fetched {fmt(props.fetchedAt)}</Typography.Text>}
            {props.error && <Typography.Text type='danger'>{props.error.message}</Typography.Text>}
        </section>
    );
};

const leverageValue = (value?: string) => (value ? `${value}x` : '-');

const WormMarketCard = (props: {item: WormMarketItem}) => {
    const item = props.item;
    const estimate = item.estimate;
    return (
        <Card className='fifa-worm-market' size='small'>
            <div className='fifa-worm-market__header'>
                <Typography.Title level={5}>{item.title || '-'}</Typography.Title>
                <Tag color={item.state === 'open' ? 'green' : 'default'}>{item.state || '-'}</Tag>
            </div>
            <MetricRow
                items={[
                    {label: 'Price', value: item.lastTradePrice || '-'},
                    {label: 'YES Lev', value: leverageValue(item.maxLeverageYes)},
                    {label: 'Liq', value: estimate?.liquidationPrice || '-'}
                ]}
            />
            <section className='fifa-worm-market__section'>
                <Typography.Text className='fifa-worm-market__section-title' strong={true}>
                    Leverage Estimate
                </Typography.Text>
                <FIFAInfoGrid
                    items={[
                        {label: 'Funds', value: estimate?.funds || '-'},
                        {label: 'Leverage', value: leverageValue(estimate?.leverage || item.maxLeverageYes)},
                        {label: 'Avg', value: estimate?.averagePrice || '-'},
                        {label: 'Best Ask', value: estimate?.bestAsk || '-'},
                        {label: 'Worst Fill', value: estimate?.worstFillPrice || '-'},
                        {label: 'Shares', value: estimate?.totalShares || '-'},
                        {label: 'Cost', value: estimate?.totalCost || '-'},
                        {label: 'Fee', value: estimate?.feeAmount || '-'},
                        {label: 'Funds Needed', value: estimate?.userFundsNeeded || '-'},
                        {label: 'Fully Filled', value: estimate ? boolTag(estimate.isFullyFilled) : '-'}
                    ]}
                />
            </section>
            <Collapse
                bordered={false}
                className='fifa-card-collapse'
                items={[
                    {
                        key: 'trading-config',
                        label: 'Trading Config',
                        children: (
                            <FIFAInfoGrid
                                items={[
                                    {label: 'Kind', value: fmt(item.configKind)},
                                    {label: 'Margin', value: boolTag(item.marginEnabled)},
                                    {label: 'Max YES', value: leverageValue(item.maxLeverageYes)},
                                    {label: 'Max NO', value: leverageValue(item.maxLeverageNo)},
                                    {label: 'Opening Fee', value: fmt(item.openingFee)},
                                    {label: 'Closing Fee', value: fmt(item.closingFee)},
                                    {label: 'Annual Fee', value: fmt(item.annualFeeRate)},
                                    {label: 'Min Size', value: fmt(item.orderMinSize)},
                                    {label: 'Price Decimals', value: fmt(item.priceDecimals)},
                                    {label: 'Shares Decimals', value: fmt(item.sharesDecimals)}
                                ]}
                            />
                        )
                    }
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
            {item.tradingDataError && <Typography.Text type='danger'>{item.tradingDataError}</Typography.Text>}
            {item.description && <Typography.Paragraph className='fifa-worm-market__description'>{item.description}</Typography.Paragraph>}
            <Button href={wormMarketURL(item.conditionId)} target='_blank' rel='noreferrer' icon={<LinkOutlined />}>
                Worm
            </Button>
        </Card>
    );
};

const WormEventPanel = (props: {data?: GetWormEventResult; loading?: boolean; error?: Error}) => {
    const item = props.data?.item;
    const markets = item ? orderedWormMarkets(item.title, item.markets) : [];
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
                        {markets.map(market => (
                            <Col key={market.conditionId} span={8}>
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

export const FIFAPage = (props: {canEdit: boolean}) => {
    const ctx = React.useContext(Context);
    const [, setParams] = useSearchParams();
    const [config, setConfig] = React.useState<PolymarketFIFAEventConfig>(null);
    const [draftWormEventID, setDraftWormEventID] = React.useState('');
    const [draftEventRef, setDraftEventRef] = React.useState('');
    const [configLoading, setConfigLoading] = React.useState(false);
    const [configSaving, setConfigSaving] = React.useState(false);
    const [configError, setConfigError] = React.useState<Error>(null);
    const [loading, setLoading] = React.useState(false);
    const [error, setError] = React.useState<Error>(null);
    const [data, setData] = React.useState<{item?: PolymarketFIFAMoneylineEventItem; fetchedAt?: number}>(null);
    const [wormLoading, setWormLoading] = React.useState(false);
    const [wormError, setWormError] = React.useState<Error>(null);
    const [wormData, setWormData] = React.useState<GetWormEventResult>(null);
    const [balancesLoading, setBalancesLoading] = React.useState(false);
    const [balancesError, setBalancesError] = React.useState<Error>(null);
    const [balancesData, setBalancesData] = React.useState<{items?: PolymarketFIFAWalletBalanceItem[]; fetchedAt?: number}>(null);
    const configRequestRef = React.useRef<(Promise<PolymarketFIFAEventConfig> & {abort?: () => void}) | null>(null);
    const wormRequestRef = React.useRef<(Promise<GetWormEventResult> & {abort?: () => void}) | null>(null);
    const eventRequestRef = React.useRef<(Promise<GetPolymarketFIFAMoneylineEventResult> & {abort?: () => void}) | null>(null);
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

    const loadWorm = React.useCallback((nextID: string) => {
        const normalized = nextID.trim();
        if (wormRequestRef.current) {
            return;
        }
        if (!normalized) {
            setWormData(null);
            setWormError(null);
            setWormLoading(false);
            return;
        }
        setWormLoading(true);
        const req = services.worm.getEvent(normalized);
        wormRequestRef.current = req;
        req.then(nextData => {
            if (wormRequestRef.current === req) {
                setWormError(null);
                setWormData(nextData);
            }
        })
            .catch(err => {
                if (wormRequestRef.current === req) {
                    setWormData(null);
                    setWormError(err instanceof Error ? err : new Error(String(err)));
                }
            })
            .finally(() => {
                if (wormRequestRef.current === req) {
                    wormRequestRef.current = null;
                    setWormLoading(false);
                }
            });
    }, []);

    const load = React.useCallback((nextRef: string) => {
        const normalized = nextRef.trim();
        if (eventRequestRef.current) {
            return;
        }
        if (!normalized) {
            setData(null);
            setError(null);
            setLoading(false);
            return;
        }
        setLoading(true);
        const req = services.polymarket.getFIFAMoneylineEvent(normalized);
        eventRequestRef.current = req;
        req.then(nextData => {
            if (eventRequestRef.current === req) {
                setError(null);
                setData(nextData);
            }
        })
            .catch(err => {
                if (eventRequestRef.current === req) {
                    setData(null);
                    setError(err instanceof Error ? err : new Error(String(err)));
                }
            })
            .finally(() => {
                if (eventRequestRef.current === req) {
                    eventRequestRef.current = null;
                    setLoading(false);
                }
            });
    }, []);

    const applyConfig = React.useCallback(
        (nextConfig: PolymarketFIFAEventConfig) => {
            setConfig(nextConfig);
            setDraftWormEventID(nextConfig.wormEventId);
            setDraftEventRef(nextConfig.eventRef);
            const nextParams = new URLSearchParams();
            nextParams.set('worm_event_id', nextConfig.wormEventId);
            nextParams.set('event_ref', nextConfig.eventRef);
            if (window.location.search.replace(/^\?/, '') !== nextParams.toString()) {
                setParams(nextParams, {replace: true});
            }
        },
        [setParams]
    );

    const loadConfig = React.useCallback(() => {
        configRequestRef.current?.abort?.();
        setConfigLoading(true);
        setConfigError(null);
        const req = services.polymarket.getFIFAEventConfig();
        configRequestRef.current = req;
        req.then(nextConfig => {
            if (configRequestRef.current === req) {
                applyConfig(nextConfig);
            }
        })
            .catch(err => {
                if (configRequestRef.current === req) {
                    setConfig(null);
                    setWormData(null);
                    setData(null);
                    setConfigError(err instanceof Error ? err : new Error(String(err)));
                }
            })
            .finally(() => {
                if (configRequestRef.current === req) {
                    configRequestRef.current = null;
                    setConfigLoading(false);
                }
            });
    }, [applyConfig]);

    React.useEffect(() => {
        loadConfig();
        return () => {
            const configReq = configRequestRef.current;
            configRequestRef.current = null;
            configReq?.abort?.();
        };
    }, [loadConfig]);

    const configuredWormEventID = config?.wormEventId || '';
    const configuredEventRef = config?.eventRef || '';
    React.useEffect(() => {
        const wormReq = wormRequestRef.current;
        const eventReq = eventRequestRef.current;
        wormRequestRef.current = null;
        eventRequestRef.current = null;
        wormReq?.abort?.();
        eventReq?.abort?.();

        setWormData(null);
        setWormError(null);
        setData(null);
        setError(null);
        if (!configuredWormEventID || !configuredEventRef) {
            setWormLoading(false);
            setLoading(false);
            return undefined;
        }

        const refreshEvents = () => {
            loadWorm(configuredWormEventID);
            load(configuredEventRef);
        };
        refreshEvents();
        const timer = window.setInterval(refreshEvents, eventRefreshIntervalMs);
        return () => {
            window.clearInterval(timer);
            const currentWormReq = wormRequestRef.current;
            const currentEventReq = eventRequestRef.current;
            wormRequestRef.current = null;
            eventRequestRef.current = null;
            currentWormReq?.abort?.();
            currentEventReq?.abort?.();
        };
    }, [configuredEventRef, configuredWormEventID, load, loadWorm]);

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

    const saveConfig = async () => {
        const wormEventId = draftWormEventID.trim();
        const eventRef = draftEventRef.trim();
        if (!wormEventId || !eventRef) {
            ctx.notifications.error('Invalid FIFA event config', 'Worm Event ID and Polymarket Event Ref are required.');
            return;
        }
        setConfigSaving(true);
        setConfigError(null);
        try {
            const nextConfig = await services.polymarket.updateFIFAEventConfig({wormEventId, eventRef});
            applyConfig(nextConfig);
            ctx.notifications.success('FIFA event config saved');
        } catch (err: any) {
            const nextError = err instanceof Error ? err : new Error(String(err));
            setConfigError(nextError);
            ctx.notifications.error('FIFA event config save failed', nextError.message);
        } finally {
            setConfigSaving(false);
        }
    };

    const configPanel = (
        <section className='fifa-event-config'>
            <div className='fifa-event-config__header'>
                <Typography.Text strong={true}>Current Event</Typography.Text>
                {props.canEdit && <Tag color='blue'>Admin</Tag>}
            </div>
            {props.canEdit ? (
                <>
                    <div className='fifa-event-config__fields'>
                        <label className='fifa-event-config__field'>
                            <Typography.Text type='secondary'>Worm Event ID</Typography.Text>
                            <Input
                                allowClear={true}
                                disabled={configLoading || configSaving || !config}
                                value={draftWormEventID}
                                placeholder='Worm Event ID'
                                onChange={event => setDraftWormEventID(event.target.value)}
                                onPressEnter={saveConfig}
                            />
                        </label>
                        <label className='fifa-event-config__field'>
                            <Typography.Text type='secondary'>Polymarket Event Ref</Typography.Text>
                            <Input
                                allowClear={true}
                                disabled={configLoading || configSaving || !config}
                                value={draftEventRef}
                                placeholder='Polymarket Event ID or slug'
                                onChange={event => setDraftEventRef(event.target.value)}
                                onPressEnter={saveConfig}
                            />
                        </label>
                    </div>
                    <div className='fifa-event-config__actions'>
                        <Button type='primary' disabled={!config || configLoading} loading={configSaving} onClick={saveConfig}>
                            Save & Load
                        </Button>
                    </div>
                </>
            ) : config ? (
                <FIFAInfoGrid
                    items={[
                        {label: 'Worm Event ID', value: config.wormEventId, copyText: config.wormEventId},
                        {label: 'Polymarket Event Ref', value: config.eventRef, copyText: config.eventRef}
                    ]}
                />
            ) : (
                <Typography.Text type='secondary'>{configLoading ? 'Loading event config…' : 'Event config unavailable'}</Typography.Text>
            )}
        </section>
    );

    const item = data?.item;
    const refresh = React.useCallback(() => {
        loadConfig();
        loadBalances();
    }, [loadBalances, loadConfig]);
    return (
        <AppPage
            title='FIFA'
            subtitle={`Worm ${unixTime(wormData?.fetchedAt)} · Polymarket ${unixTime(data?.fetchedAt)}`}
            loading={configLoading || configSaving || loading || wormLoading}
            error={configError || error}
            onRefresh={refresh}>
            <div className='fifa-page'>
                {configPanel}
                <FIFAWalletBalancesPanel items={balancesData?.items} fetchedAt={balancesData?.fetchedAt} loading={balancesLoading} error={balancesError} />
                <div className='fifa-dashboard-grid'>
                    <WormEventPanel data={wormData} loading={wormLoading} error={wormError} />
                    <section className='fifa-panel fifa-panel--polymarket'>
                        <div className='fifa-panel__title'>
                            <Typography.Text strong={true}>Polymarket</Typography.Text>
                            <Typography.Text type='secondary'>Fetched {unixTime(data?.fetchedAt)}</Typography.Text>
                        </div>
                        {!item && !loading && <Empty description='No Polymarket event loaded' />}
                        {item && (
                            <>
                                <FIFAEventSummary item={item} />
                                <Row className='fifa-moneyline-options' gutter={[12, 12]}>
                                    {item.options.map(option => (
                                        <Col key={option.outcomeKey} span={8}>
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
