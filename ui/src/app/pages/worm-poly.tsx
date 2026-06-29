import {CopyOutlined, DownOutlined, LinkOutlined} from '@ant-design/icons';
import {Button, Card, Col, Collapse, Empty, Input, Row, Tag, Typography} from 'antd';
import * as React from 'react';
import {useSearchParams} from 'react-router-dom';
import {AppPage, CardTitle, MetricRow, TruncatedText} from '../components';
import {Context} from '../shared/context';
import {services} from '../shared/services';
import {
    PolymarketFIFAMoneylineDirectionItem,
    PolymarketFIFAMoneylineEventItem,
    PolymarketFIFAMoneylineOptionItem,
    PolymarketFIFAWalletBalanceItem,
    PolymarketFIFAWalletHoldingItem,
    WormPolyFIFADashboard,
    WormPolyFIFAEventConfig
} from '../shared/services/wormpoly-service';
import {GetWormEventResult, WormMarketItem} from '../shared/services/worm-service';
import {boolTag, fmt} from './shared';

const dashboardRefreshIntervalMs = 1000;

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
const money = (value?: number) => (value === undefined || !Number.isFinite(value) ? '-' : value.toFixed(2));
const percent = (value?: number) => (value === undefined || !Number.isFinite(value) ? '-' : `${(value * 100).toFixed(2)}%`);
const balanceValue = (item: PolymarketFIFAWalletBalanceItem) =>
    item.ok && !item.errorMessage && item.amount && item.amount !== '-' ? `${item.amount} ${item.tokenSymbol || ''}`.trim() : '-';
const unixTime = (value?: number) => (value ? new Date(value * 1000).toLocaleString() : '-');
const wormMarketURL = (conditionId: string) => `https://www.worm.wtf/market/${encodeURIComponent(conditionId)}`;
type FIFAInfoGridItem = {label: React.ReactNode; value: React.ReactNode; copyText?: string};
type FIFAOutcomeKey = 'home' | 'draw' | 'away';

const fifaOutcomeKeys: FIFAOutcomeKey[] = ['home', 'draw', 'away'];

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

const numberValue = (value?: string | number) => {
    if (typeof value === 'number') {
        return Number.isFinite(value) ? value : undefined;
    }
    const normalized = String(value || '')
        .replace(/,/g, '')
        .trim();
    if (!normalized) {
        return undefined;
    }
    const parsed = Number(normalized);
    return Number.isFinite(parsed) ? parsed : undefined;
};

const fixedTokenAmountValue = (amount?: string, symbol?: string) => {
    const value = numberValue(amount);
    return value === undefined ? '-' : `${value.toFixed(2)} ${symbol || ''}`.trim();
};

const compactWalletAddress = (value?: string) => {
    const normalized = String(value || '').trim();
    if (!normalized) {
        return '-';
    }
    return normalized.length > 8 ? `${normalized.slice(0, 4)}...${normalized.slice(-4)}` : normalized;
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

const FIFAMoneylineDirectionPanel = (props: {label: 'YES' | 'NO'; item: PolymarketFIFAMoneylineDirectionItem; selected?: boolean}) => (
    <section className={`fifa-moneyline-direction${props.selected ? ' fifa-moneyline-direction--selected' : ''}`}>
        <div className='fifa-moneyline-direction__header'>
            <Tag color={props.label === 'YES' ? 'blue' : 'green'}>{props.label}</Tag>
        </div>
        <MetricRow
            items={[
                {label: 'Mid', value: price(props.item?.midPrice)},
                {label: 'Bid', value: price(props.item?.bestBid)},
                {label: 'Ask', value: price(props.item?.bestAsk)},
                {label: 'Spread', value: price(props.item?.spread)}
            ]}
        />
        <div className='fifa-moneyline-direction__token'>
            <span className='fifa-moneyline-direction__token-label'>Token</span>
            <span className='fifa-moneyline-direction__token-value'>
                <FIFAInfoValue value={props.item?.tokenId || '-'} copyText={props.item?.tokenId} />
            </span>
        </div>
    </section>
);

const FIFAMoneylineOptionCard = (props: {option: PolymarketFIFAMoneylineOptionItem; selectedNo?: boolean}) => {
    const option = props.option;
    return (
        <Card className='fifa-moneyline-option' size='small'>
            <div className='fifa-moneyline-option__header'>
                <Typography.Title level={5}>{optionTitle(option)}</Typography.Title>
                <Tag color={outcomeTone(option)}>{option.enableOrderBook && option.acceptingOrders ? 'Open' : 'Unavailable'}</Tag>
            </div>
            <div className='fifa-moneyline-option__directions'>
                <FIFAMoneylineDirectionPanel label='YES' item={option.yes} />
                <FIFAMoneylineDirectionPanel label='NO' item={option.no} selected={props.selectedNo} />
            </div>
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

const FIFAWalletHoldingCard = (props: {item: PolymarketFIFAWalletHoldingItem}) => {
    const item = props.item;
    const title = item.alias || `Wallet #${item.walletId || '-'}`;
    return (
        <Card className='fifa-wallet-holding' size='small'>
            <div className='fifa-wallet-holding__row'>
                <Typography.Title className='fifa-wallet-holding__alias' level={5} title={title}>
                    {title}
                </Typography.Title>
                <div className='fifa-wallet-holding__address'>
                    <FIFAInfoValue value={compactWalletAddress(item.walletAddress)} copyText={item.walletAddress} />
                </div>
                <span className='fifa-wallet-holding__amount fifa-wallet-holding__amount--sol'>{fixedTokenAmountValue(item.solAmount, 'SOL')}</span>
                <span className='fifa-wallet-holding__amount fifa-wallet-holding__amount--usdc'>{fixedTokenAmountValue(item.usdcAmount, 'USDC')}</span>
            </div>
            {item.errorMessage && <Typography.Text type='danger'>{item.errorMessage}</Typography.Text>}
        </Card>
    );
};

const FIFAWalletHoldingsPanel = (props: {items?: PolymarketFIFAWalletHoldingItem[]; fetchedAt?: number; loading?: boolean; error?: Error}) => {
    const items = props.items || [];
    const warmingUp = !props.fetchedAt && !items.length && !props.error;
    return (
        <section className='fifa-panel fifa-wallet-holdings'>
            <div className='fifa-panel__title'>
                <Typography.Text strong={true}>Position Management</Typography.Text>
                <Typography.Text type='secondary'>Fetched {unixTime(props.fetchedAt)}</Typography.Text>
            </div>
            {props.error && <Typography.Text type='danger'>{props.error.message}</Typography.Text>}
            {warmingUp && !props.loading && <Empty description='Wallet holdings cache is warming up' />}
            {!warmingUp && !items.length && !props.loading && <Empty description='No Worm position wallets loaded' />}
            <Row className='fifa-wallet-holdings__items' gutter={[12, 12]}>
                {items.map(item => (
                    <Col key={item.walletId || item.walletAddress} span={24}>
                        <FIFAWalletHoldingCard item={item} />
                    </Col>
                ))}
            </Row>
        </section>
    );
};

const leverageValue = (value?: string) => (value ? `${value}x` : '-');

const isInteractiveCardTarget = (target: EventTarget | null, currentTarget: EventTarget) => {
    if (!(target instanceof HTMLElement) || !(currentTarget instanceof HTMLElement)) {
        return false;
    }
    const interactive = target.closest('a, button, input, textarea, select, [role="button"]');
    return Boolean(interactive && interactive !== currentTarget);
};

const WormMarketCard = (props: {item: WormMarketItem; selected?: boolean; onSelect?: () => void}) => {
    const item = props.item;
    const estimate = item.estimate;
    const selectable = Boolean(props.onSelect);
    const onClick = (event: React.MouseEvent) => {
        if (!selectable || isInteractiveCardTarget(event.target, event.currentTarget)) {
            return;
        }
        props.onSelect();
    };
    const onKeyDown = (event: React.KeyboardEvent) => {
        if (!selectable || isInteractiveCardTarget(event.target, event.currentTarget)) {
            return;
        }
        if (event.key === 'Enter' || event.key === ' ') {
            event.preventDefault();
            props.onSelect();
        }
    };
    return (
        <Card
            aria-pressed={selectable ? props.selected : undefined}
            className={`fifa-worm-market${selectable ? ' fifa-worm-market--selectable' : ''}${props.selected ? ' fifa-worm-market--selected' : ''}`}
            role={selectable ? 'button' : undefined}
            size='small'
            tabIndex={selectable ? 0 : undefined}
            onClick={onClick}
            onKeyDown={onKeyDown}>
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
                        key: 'details',
                        label: 'Details',
                        children: (
                            <>
                                <section className='fifa-worm-market__section'>
                                    <Typography.Text className='fifa-worm-market__section-title' strong={true}>
                                        Trading Config
                                    </Typography.Text>
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
                                </section>
                                <section className='fifa-worm-market__section'>
                                    <Typography.Text className='fifa-worm-market__section-title' strong={true}>
                                        IDs
                                    </Typography.Text>
                                    <FIFAInfoGrid
                                        columns={1}
                                        items={[
                                            {label: 'Condition', value: item.conditionId, copyText: item.conditionId},
                                            {label: 'Category', value: fmt(item.category)},
                                            {label: 'Event', value: item.eventConditionId, copyText: item.eventConditionId}
                                        ]}
                                    />
                                </section>
                                {item.tradingDataError && <Typography.Text type='danger'>{item.tradingDataError}</Typography.Text>}
                                {item.description && <Typography.Paragraph className='fifa-worm-market__description'>{item.description}</Typography.Paragraph>}
                                <Button href={wormMarketURL(item.conditionId)} target='_blank' rel='noreferrer' icon={<LinkOutlined />}>
                                    Worm
                                </Button>
                            </>
                        )
                    }
                ]}
            />
        </Card>
    );
};

const WormEventPanel = (props: {data?: GetWormEventResult; loading?: boolean; error?: Error; selectedConditionId?: string; onSelectMarket?: (conditionId: string) => void}) => {
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
                            <CardTitle title={item.title} subtitle={<TruncatedText value={item.conditionId} copyable={true} />} />
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
                        {markets.map((market, index) => (
                            <Col key={market.conditionId} span={8}>
                                <WormMarketCard
                                    item={market}
                                    selected={market.conditionId === props.selectedConditionId}
                                    onSelect={fifaOutcomeKeys[index] ? () => props.onSelectMarket?.(market.conditionId) : undefined}
                                />
                            </Col>
                        ))}
                    </Row>
                </>
            )}
        </section>
    );
};

const selectedWormOutcomeKey = (data?: GetWormEventResult, selectedConditionId?: string): FIFAOutcomeKey | undefined => {
    const item = data?.item;
    if (!item || !selectedConditionId) {
        return undefined;
    }
    const index = orderedWormMarkets(item.title, item.markets).findIndex(market => market.conditionId === selectedConditionId);
    return index >= 0 ? fifaOutcomeKeys[index] : undefined;
};

const selectedWormMarket = (data?: GetWormEventResult, selectedConditionId?: string) => data?.item?.markets.find(market => market.conditionId === selectedConditionId);

const HedgeCalculatorMetric = (props: {label: string; value: React.ReactNode; tone?: 'positive' | 'negative' | 'neutral'}) => (
    <div className={`fifa-hedge-calculator__metric${props.tone ? ` fifa-hedge-calculator__metric--${props.tone}` : ''}`}>
        <span className='fifa-hedge-calculator__metric-label'>{props.label}</span>
        <span className='fifa-hedge-calculator__metric-value'>{props.value}</span>
    </div>
);

const HedgeCalculatorSection = (props: {title: string; children: React.ReactNode}) => (
    <section className='fifa-hedge-calculator__section'>
        <Typography.Text className='fifa-hedge-calculator__section-title' strong={true}>
            {props.title}
        </Typography.Text>
        <div className='fifa-hedge-calculator__metrics'>{props.children}</div>
    </section>
);

const HedgeCalculatorPanel = (props: {wormMarket?: WormMarketItem; outcomeKey?: FIFAOutcomeKey; polymarketOption?: PolymarketFIFAMoneylineOptionItem}) => {
    const estimate = props.wormMarket?.estimate;
    const funds = numberValue(estimate?.funds);
    const leverage = numberValue(estimate?.leverage || props.wormMarket?.maxLeverageYes);
    const wormTotalShares = numberValue(estimate?.totalShares);
    const closingFeeRate = numberValue(props.wormMarket?.closingFee) ?? 0;
    const pmNoAsk = numberValue(props.polymarketOption?.no?.bestAsk);
    const missing: string[] = [];
    if (props.wormMarket && !estimate) {
        missing.push('Worm estimate');
    }
    if (estimate && funds === undefined) {
        missing.push('Worm funds');
    }
    if (estimate && (!leverage || leverage <= 0)) {
        missing.push('Worm leverage');
    }
    if (estimate && wormTotalShares === undefined) {
        missing.push('Worm shares');
    }
    if (props.wormMarket && !props.polymarketOption) {
        missing.push('Polymarket option');
    }
    if (props.polymarketOption && pmNoAsk === undefined) {
        missing.push('Polymarket NO ask');
    }

    let body: React.ReactNode;
    if (!props.wormMarket) {
        body = <Empty description='Select a Worm market card to calculate the hedge.' />;
    } else if (missing.length > 0) {
        body = <Typography.Text type='secondary'>Missing data: {missing.join(', ')}</Typography.Text>;
    } else {
        const fundsValue = funds as number;
        const leverageValue = leverage as number;
        const wormTotalSharesValue = wormTotalShares as number;
        const pmNoAskValue = pmNoAsk as number;
        const borrowed = fundsValue * (leverageValue - 1);
        const wormGrossIfYes = wormTotalSharesValue - borrowed;
        const wormNetIfYes = wormGrossIfYes * (1 - closingFeeRate);
        const hedgeShares = wormTotalSharesValue / leverageValue;
        const pmNoCost = hedgeShares * pmNoAskValue;
        const totalCost = fundsValue + pmNoCost;
        const yesProfit = wormNetIfYes - totalCost;
        const yesProfitRate = yesProfit / totalCost;
        const noSettlement = hedgeShares;
        const noProfit = noSettlement - totalCost;
        body = (
            <>
                <HedgeCalculatorSection title='Worm Input'>
                    <HedgeCalculatorMetric label='Funds' value={money(fundsValue)} />
                    <HedgeCalculatorMetric label='Leverage' value={`${price(leverageValue)}x`} />
                    <HedgeCalculatorMetric label='Total Shares' value={money(wormTotalSharesValue)} />
                    <HedgeCalculatorMetric label='Borrowed' value={money(borrowed)} />
                    <HedgeCalculatorMetric label='Closing Fee' value={percent(closingFeeRate)} />
                </HedgeCalculatorSection>
                <HedgeCalculatorSection title='Polymarket Hedge'>
                    <HedgeCalculatorMetric label='Direction' value='NO' />
                    <HedgeCalculatorMetric label='NO Ask' value={price(pmNoAskValue)} />
                    <HedgeCalculatorMetric label='Hedge Shares' value={money(hedgeShares)} />
                    <HedgeCalculatorMetric label='Hedge Cost' value={money(pmNoCost)} />
                </HedgeCalculatorSection>
                <HedgeCalculatorSection title='Outcome Summary'>
                    <HedgeCalculatorMetric label='Total Cost' value={money(totalCost)} />
                    <HedgeCalculatorMetric label='YES Settlement' value={money(wormNetIfYes)} />
                    <HedgeCalculatorMetric label='YES Profit' value={money(yesProfit)} tone={yesProfit >= 0 ? 'positive' : 'negative'} />
                    <HedgeCalculatorMetric label='YES Profit Rate' value={percent(yesProfitRate)} tone={yesProfitRate >= 0 ? 'positive' : 'negative'} />
                    <HedgeCalculatorMetric label='NO Settlement' value={money(noSettlement)} />
                    <HedgeCalculatorMetric label='NO P&L' value={money(noProfit)} tone={noProfit >= 0 ? 'positive' : 'negative'} />
                </HedgeCalculatorSection>
            </>
        );
    }

    return (
        <section className='fifa-panel fifa-hedge-calculator'>
            <div className='fifa-panel__title'>
                <Typography.Text strong={true}>Hedge Calculator</Typography.Text>
                <Typography.Text type='secondary'>
                    {props.outcomeKey ? optionTitle(props.polymarketOption || ({outcomeKey: props.outcomeKey} as PolymarketFIFAMoneylineOptionItem)) : 'No Worm selection'}
                </Typography.Text>
            </div>
            {body}
        </section>
    );
};

const FIFAEventSummary = (props: {item: PolymarketFIFAMoneylineEventItem}) => {
    const item = props.item;
    const title = item.teams.length >= 2 ? `${item.teams[0].name} vs. ${item.teams[1].name}` : item.title;
    return (
        <section className='fifa-event-summary'>
            <div className='fifa-event-summary__title'>
                <CardTitle title={title} subtitle={item.eventSlug} />
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

export const WormPolyPage = (props: {canEdit: boolean}) => {
    const ctx = React.useContext(Context);
    const [, setParams] = useSearchParams();
    const [dashboard, setDashboard] = React.useState<WormPolyFIFADashboard>(null);
    const [draftWormEventID, setDraftWormEventID] = React.useState('');
    const [draftEventRef, setDraftEventRef] = React.useState('');
    const [saving, setSaving] = React.useState(false);
    const [configError, setConfigError] = React.useState<Error>(null);
    const [loading, setLoading] = React.useState(false);
    const [error, setError] = React.useState<Error>(null);
    const [selectedWormConditionId, setSelectedWormConditionId] = React.useState('');
    const dashboardRequestRef = React.useRef<(Promise<WormPolyFIFADashboard> & {abort?: () => void}) | null>(null);

    const applyConfig = React.useCallback(
        (nextConfig: WormPolyFIFAEventConfig) => {
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

    const applyDashboard = React.useCallback(
        (nextDashboard: WormPolyFIFADashboard) => {
            setDashboard(nextDashboard);
            if (nextDashboard.config) {
                applyConfig(nextDashboard.config);
            }
        },
        [applyConfig]
    );

    const loadDashboard = React.useCallback(() => {
        if (dashboardRequestRef.current) {
            return;
        }
        setLoading(true);
        setConfigError(null);
        setError(null);
        const req = services.wormpoly.getFIFADashboard();
        dashboardRequestRef.current = req;
        req.then(nextDashboard => {
            if (dashboardRequestRef.current === req) {
                applyDashboard(nextDashboard);
            }
        })
            .catch(err => {
                if (dashboardRequestRef.current === req) {
                    setDashboard(null);
                    setError(err instanceof Error ? err : new Error(String(err)));
                }
            })
            .finally(() => {
                if (dashboardRequestRef.current === req) {
                    dashboardRequestRef.current = null;
                    setLoading(false);
                }
            });
    }, [applyDashboard]);

    React.useEffect(() => {
        loadDashboard();
        const timer = window.setInterval(loadDashboard, dashboardRefreshIntervalMs);
        return () => {
            window.clearInterval(timer);
            const req = dashboardRequestRef.current;
            dashboardRequestRef.current = null;
            req?.abort?.();
        };
    }, [loadDashboard]);

    React.useEffect(() => {
        if (!selectedWormConditionId) {
            return;
        }
        const item = dashboard?.wormEvent;
        const selectedStillExists = Boolean(item?.markets.some(market => market.conditionId === selectedWormConditionId));
        if (!selectedStillExists) {
            setSelectedWormConditionId('');
        }
    }, [dashboard?.wormEvent, selectedWormConditionId]);

    const saveConfig = async () => {
        const wormEventId = draftWormEventID.trim();
        const eventRef = draftEventRef.trim();
        if (!wormEventId || !eventRef) {
            ctx.notifications.error('Invalid Worm Poly event config', 'Worm Event ID and Polymarket Event Ref are required.');
            return;
        }
        setSaving(true);
        setConfigError(null);
        try {
            const nextConfig = await services.wormpoly.updateFIFAEventConfig({wormEventId, eventRef});
            applyConfig(nextConfig);
            setDashboard(current => ({...(current || {walletBalances: [], walletHoldings: []}), config: nextConfig}));
            dashboardRequestRef.current?.abort?.();
            dashboardRequestRef.current = null;
            loadDashboard();
            ctx.notifications.success('Worm Poly event config saved');
        } catch (err: any) {
            const nextError = err instanceof Error ? err : new Error(String(err));
            setConfigError(nextError);
            ctx.notifications.error('Worm Poly event config save failed', nextError.message);
        } finally {
            setSaving(false);
        }
    };

    const config = dashboard?.config;
    const wormData: GetWormEventResult = dashboard?.wormEvent || dashboard?.wormFetchedAt ? {item: dashboard?.wormEvent, fetchedAt: dashboard?.wormFetchedAt} : null;
    const data: {item?: PolymarketFIFAMoneylineEventItem; fetchedAt?: number} =
        dashboard?.polymarketEvent || dashboard?.polymarketFetchedAt ? {item: dashboard?.polymarketEvent, fetchedAt: dashboard?.polymarketFetchedAt} : null;
    const balancesData: {items?: PolymarketFIFAWalletBalanceItem[]; fetchedAt?: number} = dashboard
        ? {items: dashboard.walletBalances, fetchedAt: dashboard.walletBalancesFetchedAt}
        : null;
    const holdingsData: {items?: PolymarketFIFAWalletHoldingItem[]; fetchedAt?: number} = dashboard
        ? {items: dashboard.walletHoldings, fetchedAt: dashboard.walletHoldingsFetchedAt}
        : null;
    const wormError = dashboard?.wormError ? new Error(dashboard.wormError) : null;
    const balancesError = dashboard?.walletBalancesError ? new Error(dashboard.walletBalancesError) : null;
    const holdingsError = dashboard?.walletHoldingsError ? new Error(dashboard.walletHoldingsError) : null;
    const polymarketError = dashboard?.polymarketError ? new Error(dashboard.polymarketError) : null;

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
                                disabled={loading || saving || !config}
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
                                disabled={loading || saving || !config}
                                value={draftEventRef}
                                placeholder='Polymarket Event ID or slug'
                                onChange={event => setDraftEventRef(event.target.value)}
                                onPressEnter={saveConfig}
                            />
                        </label>
                    </div>
                    <div className='fifa-event-config__actions'>
                        <Button type='primary' disabled={!config || loading} loading={saving} onClick={saveConfig}>
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
                <Typography.Text type='secondary'>{loading ? 'Loading event config…' : 'Event config unavailable'}</Typography.Text>
            )}
        </section>
    );

    const item = data?.item;
    const selectedOutcomeKey = selectedWormOutcomeKey(wormData, selectedWormConditionId);
    const selectedMarket = selectedWormMarket(wormData, selectedWormConditionId);
    const selectedPolymarketOption = item?.options.find(option => option.outcomeKey === selectedOutcomeKey);
    const selectWormMarket = React.useCallback((conditionId: string) => {
        setSelectedWormConditionId(current => (current === conditionId ? '' : conditionId));
    }, []);
    return (
        <AppPage
            title='Worm Poly'
            subtitle={`Worm ${unixTime(wormData?.fetchedAt)} · Polymarket ${unixTime(data?.fetchedAt)}`}
            loading={loading || saving}
            error={configError || error}
            onRefresh={loadDashboard}>
            <div className='fifa-page'>
                {configPanel}
                <FIFAWalletBalancesPanel items={balancesData?.items} fetchedAt={balancesData?.fetchedAt} loading={loading} error={balancesError} />
                <HedgeCalculatorPanel wormMarket={selectedMarket} outcomeKey={selectedOutcomeKey} polymarketOption={selectedPolymarketOption} />
                <div className='fifa-dashboard-grid'>
                    <div className='fifa-dashboard-column fifa-dashboard-column--left'>
                        <WormEventPanel data={wormData} loading={loading} error={wormError} selectedConditionId={selectedWormConditionId} onSelectMarket={selectWormMarket} />
                        <FIFAWalletHoldingsPanel items={holdingsData?.items} fetchedAt={holdingsData?.fetchedAt} loading={loading} error={holdingsError} />
                    </div>
                    <section className='fifa-panel fifa-panel--polymarket'>
                        <div className='fifa-panel__title'>
                            <Typography.Text strong={true}>Polymarket</Typography.Text>
                            <Typography.Text type='secondary'>Fetched {unixTime(data?.fetchedAt)}</Typography.Text>
                        </div>
                        {polymarketError && <Typography.Text type='danger'>{polymarketError.message}</Typography.Text>}
                        {!item && !loading && <Empty description='No Polymarket event loaded' />}
                        {item && (
                            <>
                                <FIFAEventSummary item={item} />
                                <Row className='fifa-moneyline-options' gutter={[12, 12]}>
                                    {item.options.map(option => (
                                        <Col key={option.outcomeKey} span={8}>
                                            <FIFAMoneylineOptionCard option={option} selectedNo={selectedOutcomeKey === option.outcomeKey} />
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
