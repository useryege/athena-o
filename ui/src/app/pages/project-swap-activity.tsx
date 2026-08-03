import {EyeOutlined} from '@ant-design/icons';
import {Alert, Button, Card, Drawer, Empty, Pagination, Skeleton, Space, Table, Tag, Tooltip, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {ChoiceGroup, ResourceTable, Section, StatusTag} from '../components';
import {formatBeijingDateTime, formatBlockNumber} from '../shared/format';
import {services} from '../shared/services';
import {
    PagedResponse,
    TokenProjectSwapActivity,
    TokenProjectSwapBlock,
    TokenProjectSwapEvent,
    TokenProjectSwapPairActivity,
    TokenProjectSwapPairKind
} from '../shared/services/token-service';
import {
    exactDecimalFromRaw,
    formatProjectTimeText as formatTimeText,
    ProjectExactValue as ExactValue,
    ProjectExplorerValue as ExplorerIdentifier,
    ProjectRawTokenAmount as RawAmount,
    ProjectTimeValue as TimeValue
} from './project-detail-values';

const TARGET_SWAP_BLOCK_COUNT = 100;
const ACTIVITY_REFRESH_INTERVAL_MS = 30000;
const EVENT_PAGE_SIZE = 50;
const TIMELINE_WIDTH = 980;
const TIMELINE_HEIGHT = 230;
const TIMELINE_PADDING = {top: 20, right: 22, bottom: 34, left: 62};

type TimelineAxis = 'time' | 'block';
type TimelineMetric = 'events' | 'origins' | 'quote-volume' | 'price';

interface SwapBlockSelection {
    pair: TokenProjectSwapPairActivity;
    block: TokenProjectSwapBlock;
}

const pairKinds: TokenProjectSwapPairKind[] = ['weth', 'usdt'];

const hasText = (value?: string) => value !== undefined && value !== '';

const pairLabel = (kind?: TokenProjectSwapPairKind) => (kind === 'usdt' ? 'USDT Pair' : 'Wrapped-native Pair');

const expiredReasonLabel = (reason?: string) => {
    switch (reason) {
        case 'no_swap':
            return 'No Swap within 24 hours';
        case 'inactive':
            return 'Inactive for 24 hours';
        case 'max_duration':
            return 'Seven-day maximum reached';
        default:
            return reason || 'Unknown reason';
    }
};

const addRawIntegers = (...values: Array<string | undefined>) => {
    try {
        return values.reduce((total, value) => total + BigInt(value || '0'), 0n).toString();
    } catch {
        return undefined;
    }
};

const friendlyDuration = (seconds?: string) => {
    if (!hasText(seconds)) {
        return '-';
    }
    try {
        let remaining = BigInt(seconds as string);
        const days = remaining / 86400n;
        remaining %= 86400n;
        const hours = remaining / 3600n;
        remaining %= 3600n;
        const minutes = remaining / 60n;
        const rest = remaining % 60n;
        const pieces = [
            days > 0n ? `${days}d` : '',
            hours > 0n ? `${hours}h` : '',
            minutes > 0n ? `${minutes}m` : '',
            rest > 0n || (days === 0n && hours === 0n && minutes === 0n) ? `${rest}s` : ''
        ].filter(Boolean);
        return pieces.join(' ');
    } catch {
        return seconds as string;
    }
};

const countText = (value?: number) => new Intl.NumberFormat().format(value || 0);

const heatLevel = (eventCount?: number) => {
    const count = eventCount || 0;
    if (count >= 10) {
        return 4;
    }
    if (count >= 5) {
        return 3;
    }
    if (count >= 2) {
        return 2;
    }
    return 1;
};

const Heatmap = (props: {pair: TokenProjectSwapPairActivity; onSelect: (block: TokenProjectSwapBlock) => void}) => {
    const initialFocus = Math.max(0, Math.min(TARGET_SWAP_BLOCK_COUNT - 1, (props.pair.swapBlockCount || 1) - 1));
    const [focusIndex, setFocusIndex] = React.useState(initialFocus);
    const cellRefs = React.useRef<Array<HTMLButtonElement | null>>([]);
    const blocksBySample = React.useMemo(() => new Map(props.pair.blocks.map(block => [block.sampleIndex, block])), [props.pair.blocks]);

    React.useEffect(() => {
        setFocusIndex(Math.max(0, Math.min(TARGET_SWAP_BLOCK_COUNT - 1, (props.pair.swapBlockCount || 1) - 1)));
    }, [props.pair.pairKind, props.pair.swapBlockCount]);

    const moveFocus = (next: number) => {
        const bounded = Math.max(0, Math.min(TARGET_SWAP_BLOCK_COUNT - 1, next));
        setFocusIndex(bounded);
        cellRefs.current[bounded]?.focus();
    };

    const onKeyDown = (event: React.KeyboardEvent<HTMLButtonElement>, index: number, block?: TokenProjectSwapBlock) => {
        let next: number | undefined;
        switch (event.key) {
            case 'ArrowLeft':
                next = index - 1;
                break;
            case 'ArrowRight':
                next = index + 1;
                break;
            case 'ArrowUp':
                next = index - 10;
                break;
            case 'ArrowDown':
                next = index + 10;
                break;
            case 'Home':
                next = Math.floor(index / 10) * 10;
                break;
            case 'End':
                next = Math.floor(index / 10) * 10 + 9;
                break;
            case 'Enter':
            case ' ':
                if (block) {
                    event.preventDefault();
                    props.onSelect(block);
                }
                return;
            default:
                return;
        }
        event.preventDefault();
        moveFocus(next);
    };

    return (
        <div className='swap-heatmap' role='grid' aria-label={`${pairLabel(props.pair.pairKind)} sampled Swap blocks`} aria-rowcount={10} aria-colcount={10}>
            {Array.from({length: 10}, (_, rowIndex) => (
                <div className='swap-heatmap__row' role='row' aria-rowindex={rowIndex + 1} key={rowIndex + 1}>
                    {Array.from({length: 10}, (_, columnIndex) => {
                        const index = rowIndex * 10 + columnIndex;
                        const sampleIndex = index + 1;
                        const block = blocksBySample.get(sampleIndex);
                        const emptyState = props.pair.status === 'collecting' ? 'pending' : 'unavailable';
                        const className = block
                            ? `swap-heatmap__cell swap-heatmap__cell--level-${heatLevel(block.eventCount)}`
                            : `swap-heatmap__cell swap-heatmap__cell--${emptyState}`;
                        const label = block
                            ? `Sample ${sampleIndex}, block ${block.blockNumber}, ${countText(block.eventCount)} Swap events, ${countText(
                                  block.transactionOriginCount
                              )} transaction origins, ${formatTimeText(block.blockTime) || block.blockTime || 'unknown time'}`
                            : props.pair.status === 'collecting'
                              ? `Sample ${sampleIndex}, waiting for a Swap block`
                              : `Sample ${sampleIndex}, unavailable because collection ${props.pair.status}`;
                        return (
                            <button
                                key={sampleIndex}
                                ref={element => {
                                    cellRefs.current[index] = element;
                                }}
                                type='button'
                                role='gridcell'
                                aria-colindex={columnIndex + 1}
                                aria-disabled={!block}
                                aria-label={label}
                                title={label}
                                tabIndex={focusIndex === index ? 0 : -1}
                                className={className}
                                onFocus={() => setFocusIndex(index)}
                                onClick={() => block && props.onSelect(block)}
                                onKeyDown={event => onKeyDown(event, index, block)}>
                                <span>{sampleIndex}</span>
                            </button>
                        );
                    })}
                </div>
            ))}
        </div>
    );
};

const PairCard = (props: {
    pair?: TokenProjectSwapPairActivity;
    kind: TokenProjectSwapPairKind;
    chainID?: number;
    onSelect: (pair: TokenProjectSwapPairActivity, block: TokenProjectSwapBlock) => void;
}) => {
    if (!props.pair) {
        return (
            <Card className='swap-pair-card' size='small' title={pairLabel(props.kind)}>
                <Alert type='error' title={`${pairLabel(props.kind)} activity is missing`} showIcon={true} />
            </Card>
        );
    }
    const pair = props.pair;
    const target = pair.targetSwapBlockCount || TARGET_SWAP_BLOCK_COUNT;
    const quoteVolumeRaw = addRawIntegers(pair.buyQuoteAmountRaw, pair.sellQuoteAmountRaw);
    const terminalTime = pair.status === 'completed' ? pair.completedBlockTime : pair.status === 'expired' ? pair.expiredBlockTime : undefined;
    const terminalBlock = pair.status === 'completed' ? pair.completedBlockNumber : pair.status === 'expired' ? pair.expiredBlockNumber : undefined;
    return (
        <Card
            className={`swap-pair-card swap-pair-card--${pair.status || 'unknown'}`}
            size='small'
            title={
                <span className='swap-pair-card__title'>
                    <span>{pairLabel(pair.pairKind)}</span>
                    <Typography.Text type='secondary'>
                        {pair.swapBlockCount || 0} / {target}
                    </Typography.Text>
                </span>
            }
            extra={<StatusTag value={pair.status || 'Unknown'} positive={pair.status === 'completed'} negative={pair.status === 'expired'} />}>
            <div className='swap-pair-card__address'>
                <Typography.Text type='secondary'>Pair contract</Typography.Text>
                <ExplorerIdentifier chainID={props.chainID} kind='address' value={pair.pairAddress} />
            </div>
            <div className='swap-pair-card__metrics' aria-label={`${pairLabel(pair.pairKind)} totals`}>
                {[
                    ['Swap events', countText(pair.eventCount)],
                    ['Transactions', countText(pair.transactionCount)],
                    ['Tx origins', countText(pair.transactionOriginCount)],
                    ['Buy / Sell / Complex', `${pair.buyEventCount || 0} / ${pair.sellEventCount || 0} / ${pair.complexEventCount || 0}`]
                ].map(([label, value]) => (
                    <span key={label}>
                        <Typography.Text type='secondary'>{label}</Typography.Text>
                        <strong>{value}</strong>
                    </span>
                ))}
                <span className='swap-pair-card__metric-wide'>
                    <Typography.Text type='secondary'>Simple-trade quote volume</Typography.Text>
                    <strong>
                        <RawAmount raw={quoteVolumeRaw} decimals={pair.quoteAsset?.decimals} symbol={pair.quoteAsset?.symbol} />
                    </strong>
                </span>
            </div>
            <div className='swap-pair-card__lifecycle'>
                <span>
                    <Typography.Text type='secondary'>Started</Typography.Text>
                    <strong>
                        Block <ExactValue value={pair.startBlockNumber} />
                    </strong>
                    <TimeValue value={pair.startBlockTime} />
                </span>
                <span>
                    <Typography.Text type='secondary'>First Swap</Typography.Text>
                    {pair.firstSwapBlockNumber ? (
                        <>
                            <strong>
                                Block <ExactValue value={pair.firstSwapBlockNumber} />
                            </strong>
                            <TimeValue value={pair.firstSwapBlockTime} />
                        </>
                    ) : (
                        <Typography.Text type='secondary'>Not observed</Typography.Text>
                    )}
                </span>
                <span>
                    <Typography.Text type='secondary'>Last Swap</Typography.Text>
                    {pair.lastSwapBlockNumber ? (
                        <>
                            <strong>
                                Block <ExactValue value={pair.lastSwapBlockNumber} />
                            </strong>
                            <TimeValue value={pair.lastSwapBlockTime} />
                        </>
                    ) : (
                        <Typography.Text type='secondary'>Not observed</Typography.Text>
                    )}
                </span>
                <span>
                    <Typography.Text type='secondary'>{pair.status === 'collecting' ? 'Next expiry' : 'Terminal state'}</Typography.Text>
                    {pair.status === 'collecting' ? (
                        <>
                            <strong>
                                <TimeValue value={pair.nextExpiryBlockTime} />
                            </strong>
                            <Typography.Text type='secondary'>
                                Absolute limit <TimeValue value={pair.absoluteExpiryBlockTime} />
                            </Typography.Text>
                        </>
                    ) : (
                        <>
                            <strong>{pair.status === 'expired' ? expiredReasonLabel(pair.expiredReason) : '100 Swap blocks collected'}</strong>
                            <Typography.Text type='secondary'>
                                Block <ExactValue value={terminalBlock} /> · <TimeValue value={terminalTime} />
                            </Typography.Text>
                        </>
                    )}
                </span>
            </div>
            {pair.swapBlockCount === 0 && (
                <Alert
                    type={pair.status === 'expired' ? 'warning' : 'info'}
                    showIcon={true}
                    title={
                        pair.status === 'expired'
                            ? pair.expiredReason === 'no_swap'
                                ? 'Expired before the first Swap'
                                : `Collection expired: ${expiredReasonLabel(pair.expiredReason)}`
                            : 'Waiting for the first Swap block'
                    }
                />
            )}
            <div className='swap-pair-card__heatmap-heading'>
                <div>
                    <Typography.Text strong={true}>100-block activity map</Typography.Text>
                    <Typography.Paragraph type='secondary'>Each filled cell is one distinct block that emitted at least one matching Swap.</Typography.Paragraph>
                </div>
                <span className='swap-heatmap-legend' aria-label='Event intensity legend'>
                    <i className='swap-heatmap__cell--level-1' /> 1
                    <i className='swap-heatmap__cell--level-2' /> 2–4
                    <i className='swap-heatmap__cell--level-3' /> 5–9
                    <i className='swap-heatmap__cell--level-4' /> 10+
                </span>
            </div>
            <Heatmap pair={pair} onSelect={block => props.onSelect(pair, block)} />
        </Card>
    );
};

const approximateNumber = (value?: string) => {
    if (!hasText(value)) {
        return undefined;
    }
    const numeric = Number(value);
    return Number.isFinite(numeric) ? numeric : undefined;
};

const approximateRawAmount = (raw?: string, decimals = 0) => approximateNumber(exactDecimalFromRaw(raw, decimals));

const formatChartNumber = (value: number) =>
    new Intl.NumberFormat(undefined, {
        notation: Math.abs(value) >= 100000 ? 'compact' : 'standard',
        maximumFractionDigits: Math.abs(value) < 1 ? 6 : 2
    }).format(value);

const timelineXValue = (block: TokenProjectSwapBlock, axis: TimelineAxis) => {
    if (axis === 'block') {
        return approximateNumber(block.blockNumber);
    }
    const parsed = block.blockTime ? new Date(block.blockTime).getTime() : Number.NaN;
    return Number.isFinite(parsed) ? parsed : undefined;
};

const TimelinePanel = (props: {
    pair: TokenProjectSwapPairActivity;
    axis: TimelineAxis;
    metric: TimelineMetric;
    xMinimum: number;
    xMaximum: number;
    onSelect: (block: TokenProjectSwapBlock) => void;
}) => {
    const titleID = React.useId();
    const descriptionID = React.useId();
    const quoteDecimals = props.pair.quoteAsset?.decimals || 0;
    const chartWidth = TIMELINE_WIDTH - TIMELINE_PADDING.left - TIMELINE_PADDING.right;
    const chartHeight = TIMELINE_HEIGHT - TIMELINE_PADDING.top - TIMELINE_PADDING.bottom;
    const points = props.pair.blocks
        .map(block => ({block, x: timelineXValue(block, props.axis)}))
        .filter((point): point is {block: TokenProjectSwapBlock; x: number} => point.x !== undefined)
        .sort((left, right) => left.x - right.x);
    const xSpread = props.xMaximum - props.xMinimum || 1;
    const xAt = (value: number) => TIMELINE_PADDING.left + ((value - props.xMinimum) / xSpread) * chartWidth;
    const barWidth = Math.max(3, Math.min(12, chartWidth / Math.max(points.length, 100) - 1));

    const values = points.flatMap(({block}) => {
        if (props.metric === 'events') {
            return [(block.buyEventCount || 0) + (block.sellEventCount || 0) + (block.complexEventCount || 0)];
        }
        if (props.metric === 'origins') {
            return [block.transactionOriginCount || 0];
        }
        if (props.metric === 'quote-volume') {
            return [(approximateRawAmount(block.buyQuoteAmountRaw, quoteDecimals) || 0) + (approximateRawAmount(block.sellQuoteAmountRaw, quoteDecimals) || 0)];
        }
        return [block.openPrice, block.highPrice, block.lowPrice, block.closePrice].map(approximateNumber).filter((value): value is number => value !== undefined);
    });
    const hasMetricData = values.length > 0 && (props.metric === 'price' || values.some(value => value > 0));
    const minimum = props.metric === 'price' && values.length > 0 ? Math.min(...values) : 0;
    const maximum = values.length > 0 ? Math.max(...values) : 1;
    const ySpread = maximum - minimum || Math.abs(maximum) * 0.04 || 1;
    const yAt = (value: number) => TIMELINE_PADDING.top + ((maximum + ySpread * 0.08 - value) / (ySpread * 1.16)) * chartHeight;
    const baselineY = TIMELINE_PADDING.top + chartHeight;

    const axisLabel = (value: number) => (props.axis === 'time' ? formatBeijingDateTime(new Date(value)) || '-' : formatBlockNumber(value));

    const pointTitle = (block: TokenProjectSwapBlock) => {
        const prefix = `Sample ${block.sampleIndex}, block ${block.blockNumber}, ${formatTimeText(block.blockTime) || block.blockTime || 'unknown time'}`;
        if (props.metric === 'events') {
            return `${prefix}. Buy ${block.buyEventCount || 0}, sell ${block.sellEventCount || 0}, complex ${block.complexEventCount || 0}.`;
        }
        if (props.metric === 'origins') {
            return `${prefix}. ${block.transactionOriginCount || 0} transaction origins.`;
        }
        if (props.metric === 'quote-volume') {
            const buy = exactDecimalFromRaw(block.buyQuoteAmountRaw, quoteDecimals);
            const sell = exactDecimalFromRaw(block.sellQuoteAmountRaw, quoteDecimals);
            return `${prefix}. Buy quote ${buy || '0'}, sell quote ${sell || '0'} ${props.pair.quoteAsset?.symbol || 'quote'}.`;
        }
        return `${prefix}. Open ${block.openPrice || '-'}, high ${block.highPrice || '-'}, low ${block.lowPrice || '-'}, close ${block.closePrice || '-'}, VWAP ${
            block.vwap || '-'
        } ${props.pair.quoteAsset?.symbol || 'quote'} per ${props.pair.baseAsset?.symbol || 'base'}.`;
    };

    return (
        <div className='swap-timeline-panel'>
            <div className='swap-timeline-panel__heading'>
                <strong>{pairLabel(props.pair.pairKind)}</strong>
                <span>
                    {props.metric === 'price' || props.metric === 'quote-volume' ? props.pair.quoteAsset?.symbol || 'Quote' : props.metric === 'origins' ? 'Addresses' : 'Events'}
                </span>
            </div>
            {points.length === 0 ? (
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='No sampled Swap blocks' />
            ) : (
                <svg viewBox={`0 0 ${TIMELINE_WIDTH} ${TIMELINE_HEIGHT}`} role='img' aria-labelledby={titleID} aria-describedby={descriptionID}>
                    <title id={titleID}>
                        {pairLabel(props.pair.pairKind)} {props.metric} by {props.axis}
                    </title>
                    <desc id={descriptionID}>Sparse observations are shown independently and are not connected across blocks.</desc>
                    {[0.25, 0.5, 0.75].map(ratio => (
                        <line
                            key={ratio}
                            className='swap-timeline__grid'
                            x1={TIMELINE_PADDING.left}
                            x2={TIMELINE_WIDTH - TIMELINE_PADDING.right}
                            y1={TIMELINE_PADDING.top + chartHeight * ratio}
                            y2={TIMELINE_PADDING.top + chartHeight * ratio}
                        />
                    ))}
                    <text className='swap-timeline__axis-label' x={TIMELINE_PADDING.left - 8} y={TIMELINE_PADDING.top + 6} textAnchor='end'>
                        {formatChartNumber(maximum)}
                    </text>
                    <text className='swap-timeline__axis-label' x={TIMELINE_PADDING.left - 8} y={baselineY} textAnchor='end'>
                        {formatChartNumber(minimum)}
                    </text>
                    {hasMetricData ? (
                        points.map(({block, x}) => {
                            const centerX = xAt(x);
                            const activate = () => props.onSelect(block);
                            if (props.metric === 'price') {
                                const open = approximateNumber(block.openPrice);
                                const high = approximateNumber(block.highPrice);
                                const low = approximateNumber(block.lowPrice);
                                const close = approximateNumber(block.closePrice);
                                if (open === undefined || high === undefined || low === undefined || close === undefined) {
                                    return null;
                                }
                                const tone = close >= open ? 'buy' : 'sell';
                                return (
                                    <g
                                        key={`${block.sampleIndex}-${block.blockNumber}`}
                                        className='swap-timeline__mark'
                                        role='button'
                                        tabIndex={0}
                                        aria-label={pointTitle(block)}
                                        onClick={activate}
                                        onKeyDown={event => {
                                            if (event.key === 'Enter' || event.key === ' ') {
                                                event.preventDefault();
                                                activate();
                                            }
                                        }}>
                                        <title>{pointTitle(block)}</title>
                                        <line className={`swap-timeline__ohlc swap-timeline__ohlc--${tone}`} x1={centerX} x2={centerX} y1={yAt(high)} y2={yAt(low)} />
                                        <line className={`swap-timeline__ohlc swap-timeline__ohlc--${tone}`} x1={centerX - 5} x2={centerX} y1={yAt(open)} y2={yAt(open)} />
                                        <line className={`swap-timeline__ohlc swap-timeline__ohlc--${tone}`} x1={centerX} x2={centerX + 5} y1={yAt(close)} y2={yAt(close)} />
                                    </g>
                                );
                            }
                            const segments =
                                props.metric === 'events'
                                    ? [
                                          {value: block.buyEventCount || 0, tone: 'buy'},
                                          {value: block.sellEventCount || 0, tone: 'sell'},
                                          {value: block.complexEventCount || 0, tone: 'complex'}
                                      ]
                                    : props.metric === 'quote-volume'
                                      ? [
                                            {value: approximateRawAmount(block.buyQuoteAmountRaw, quoteDecimals) || 0, tone: 'buy'},
                                            {value: approximateRawAmount(block.sellQuoteAmountRaw, quoteDecimals) || 0, tone: 'sell'}
                                        ]
                                      : [{value: block.transactionOriginCount || 0, tone: 'origin'}];
                            let accumulated = 0;
                            return (
                                <g
                                    key={`${block.sampleIndex}-${block.blockNumber}`}
                                    className='swap-timeline__mark'
                                    role='button'
                                    tabIndex={0}
                                    aria-label={pointTitle(block)}
                                    onClick={activate}
                                    onKeyDown={event => {
                                        if (event.key === 'Enter' || event.key === ' ') {
                                            event.preventDefault();
                                            activate();
                                        }
                                    }}>
                                    <title>{pointTitle(block)}</title>
                                    {segments.map(segment => {
                                        const bottom = accumulated;
                                        accumulated += segment.value;
                                        const topY = yAt(accumulated);
                                        const bottomY = bottom === 0 ? baselineY : yAt(bottom);
                                        return (
                                            <rect
                                                key={segment.tone}
                                                className={`swap-timeline__bar swap-timeline__bar--${segment.tone}`}
                                                x={centerX - barWidth / 2}
                                                y={topY}
                                                width={barWidth}
                                                height={Math.max(1, bottomY - topY)}
                                            />
                                        );
                                    })}
                                </g>
                            );
                        })
                    ) : (
                        <text className='swap-timeline__empty-label' x={TIMELINE_WIDTH / 2} y={TIMELINE_HEIGHT / 2} textAnchor='middle'>
                            {props.metric === 'price' ? 'No simple-trade execution prices' : 'No values for this metric'}
                        </text>
                    )}
                    <text className='swap-timeline__axis-label' x={TIMELINE_PADDING.left} y={TIMELINE_HEIGHT - 10}>
                        {axisLabel(props.xMinimum)}
                    </text>
                    <text className='swap-timeline__axis-label' x={TIMELINE_WIDTH - TIMELINE_PADDING.right} y={TIMELINE_HEIGHT - 10} textAnchor='end'>
                        {axisLabel(props.xMaximum)}
                    </text>
                </svg>
            )}
        </div>
    );
};

const SwapTimeline = (props: {pairs: TokenProjectSwapPairActivity[]; onSelect: (pair: TokenProjectSwapPairActivity, block: TokenProjectSwapBlock) => void}) => {
    const [axis, setAxis] = React.useState<TimelineAxis>('time');
    const [metric, setMetric] = React.useState<TimelineMetric>('events');
    const xValues = props.pairs.flatMap(pair => pair.blocks.map(block => timelineXValue(block, axis))).filter((value): value is number => value !== undefined);
    const xMinimum = xValues.length > 0 ? Math.min(...xValues) : 0;
    const xMaximum = xValues.length > 0 ? Math.max(...xValues) : 1;
    return (
        <Section
            title='Sparse Swap timeline'
            extra={
                <Space wrap={true}>
                    <ChoiceGroup<TimelineAxis>
                        ariaLabel='Timeline horizontal axis'
                        size='small'
                        value={axis}
                        options={[
                            {label: 'Time', value: 'time'},
                            {label: 'Block', value: 'block'}
                        ]}
                        onChange={setAxis}
                    />
                    <ChoiceGroup<TimelineMetric>
                        ariaLabel='Timeline metric'
                        size='small'
                        value={metric}
                        options={[
                            {label: 'Swap Events', value: 'events'},
                            {label: 'Transaction Origins', value: 'origins'},
                            {label: 'Quote Volume', value: 'quote-volume'},
                            {label: 'Execution Price', value: 'price'}
                        ]}
                        onChange={setMetric}
                    />
                </Space>
            }>
            <Typography.Paragraph type='secondary' className='swap-timeline__description'>
                WETH/WBNB and USDT use the same horizontal range but independent vertical scales. Sparse Swap blocks are never connected. Quote volume excludes complex events.
            </Typography.Paragraph>
            <div className='swap-timeline'>
                {pairKinds.map(kind => {
                    const pair = props.pairs.find(candidate => candidate.pairKind === kind);
                    return pair ? (
                        <TimelinePanel key={kind} pair={pair} axis={axis} metric={metric} xMinimum={xMinimum} xMaximum={xMaximum} onSelect={block => props.onSelect(pair, block)} />
                    ) : null;
                })}
            </div>
            <div className='swap-timeline__legend' aria-label='Timeline legend'>
                <span>
                    <i className='swap-timeline__legend-swatch swap-timeline__legend-swatch--buy' /> Buy
                </span>
                <span>
                    <i className='swap-timeline__legend-swatch swap-timeline__legend-swatch--sell' /> Sell
                </span>
                <span>
                    <i className='swap-timeline__legend-swatch swap-timeline__legend-swatch--complex' /> Complex
                </span>
                <Typography.Text type='secondary'>Pair-accounting values, not user receipts or reserve spot prices.</Typography.Text>
            </div>
        </Section>
    );
};

const DirectionCounts = (props: {block: TokenProjectSwapBlock}) => (
    <Space size={4} wrap={false}>
        <Tag color='green'>B {props.block.buyEventCount || 0}</Tag>
        <Tag color='red'>S {props.block.sellEventCount || 0}</Tag>
        <Tag>C {props.block.complexEventCount || 0}</Tag>
    </Space>
);

const SwapBlockTable = (props: {pairs: TokenProjectSwapPairActivity[]; onSelect: (pair: TokenProjectSwapPairActivity, block: TokenProjectSwapBlock) => void}) => {
    const availableKinds = props.pairs.map(pair => pair.pairKind).filter((kind): kind is TokenProjectSwapPairKind => Boolean(kind));
    const [selectedKind, setSelectedKind] = React.useState<TokenProjectSwapPairKind>('weth');
    const [page, setPage] = React.useState(1);
    const pair = props.pairs.find(candidate => candidate.pairKind === selectedKind) || props.pairs[0];

    React.useEffect(() => {
        if (!availableKinds.includes(selectedKind) && availableKinds[0]) {
            setSelectedKind(availableKinds[0]);
        }
    }, [availableKinds.join(','), selectedKind]);

    const blocks = React.useMemo(() => [...(pair?.blocks || [])].sort((left, right) => (right.sampleIndex || 0) - (left.sampleIndex || 0)), [pair]);
    const pageItems = blocks.slice((page - 1) * EVENT_PAGE_SIZE, page * EVENT_PAGE_SIZE);
    const quoteVolume = (block: TokenProjectSwapBlock) => addRawIntegers(block.buyQuoteAmountRaw, block.sellQuoteAmountRaw);
    const columns: ColumnsType<TokenProjectSwapBlock> = [
        {title: 'Sample', width: 78, render: item => <ExactValue value={String(item.sampleIndex || '')} />},
        {title: 'Block', width: 130, render: item => <ExactValue value={item.blockNumber} />},
        {title: 'Time', width: 180, render: item => <TimeValue value={item.blockTime} />},
        {title: 'Events', width: 82, render: item => countText(item.eventCount)},
        {title: 'Transactions', width: 110, render: item => countText(item.transactionCount)},
        {title: 'Origins', width: 82, render: item => countText(item.transactionOriginCount)},
        {title: 'Buy / Sell / Complex', width: 210, render: item => <DirectionCounts block={item} />},
        {
            title: 'Gap',
            width: 180,
            render: item =>
                item.sampleIndex === 1 ? (
                    <Typography.Text type='secondary'>First sample</Typography.Text>
                ) : (
                    <Tooltip title={`${item.previousBlockGap || '0'} blocks · ${item.previousTimeGapSeconds || '0'} seconds`}>
                        <span>
                            <ExactValue value={item.previousBlockGap} suffix='blocks' /> · {friendlyDuration(item.previousTimeGapSeconds)}
                        </span>
                    </Tooltip>
                )
        },
        {
            title: 'Quote volume',
            width: 180,
            render: item => <RawAmount raw={quoteVolume(item)} decimals={pair?.quoteAsset?.decimals} symbol={pair?.quoteAsset?.symbol} />
        },
        {
            title: 'Close price',
            width: 190,
            render: item => (
                <ExactValue value={item.closePrice} suffix={item.closePrice ? `${pair?.quoteAsset?.symbol || 'Quote'} / ${pair?.baseAsset?.symbol || 'Base'}` : undefined} />
            )
        },
        {
            title: '',
            fixed: 'right',
            width: 72,
            render: item => (
                <Button size='small' icon={<EyeOutlined />} aria-label={`View events in block ${item.blockNumber}`} onClick={() => pair && props.onSelect(pair, item)} />
            )
        }
    ];

    React.useEffect(() => {
        setPage(1);
    }, [selectedKind]);

    return (
        <Section
            title='Sampled Swap blocks'
            extra={
                <ChoiceGroup<TokenProjectSwapPairKind>
                    ariaLabel='Swap block Pair'
                    size='small'
                    value={selectedKind}
                    options={pairKinds.map(kind => ({label: pairLabel(kind), value: kind, disabled: !availableKinds.includes(kind)}))}
                    onChange={setSelectedKind}
                />
            }>
            <ResourceTable
                label={`${pairLabel(pair?.pairKind)} sampled Swap blocks`}
                rowKey={item => `${item.sampleIndex}-${item.blockNumber}`}
                items={pageItems}
                columns={columns}
                total={blocks.length}
                page={page}
                pageSize={EVENT_PAGE_SIZE}
                pageSizeOptions={[EVENT_PAGE_SIZE]}
                onPageChange={nextPage => setPage(nextPage)}
                onItemClick={item => pair && props.onSelect(pair, item)}
                scrollX={1510}
            />
        </Section>
    );
};

const EventFlow = (props: {event: TokenProjectSwapEvent; pair?: TokenProjectSwapPairActivity; asset: 'base' | 'quote'}) => {
    const asset = props.asset === 'base' ? props.pair?.baseAsset : props.pair?.quoteAsset;
    const input = props.asset === 'base' ? props.event.baseAmountInRaw : props.event.quoteAmountInRaw;
    const output = props.asset === 'base' ? props.event.baseAmountOutRaw : props.event.quoteAmountOutRaw;
    return (
        <span className='swap-event-flow'>
            <span>
                <Typography.Text type='secondary'>In</Typography.Text>
                <RawAmount raw={input} decimals={asset?.decimals} symbol={asset?.symbol} />
            </span>
            <span>
                <Typography.Text type='secondary'>Out</Typography.Text>
                <RawAmount raw={output} decimals={asset?.decimals} symbol={asset?.symbol} />
            </span>
        </span>
    );
};

const EventDrawer = (props: {projectID: number; chainID?: number; selection?: SwapBlockSelection; onClose: () => void}) => {
    const [page, setPage] = React.useState(1);
    const [data, setData] = React.useState<PagedResponse<TokenProjectSwapEvent>>();
    const [loading, setLoading] = React.useState(false);
    const [error, setError] = React.useState<Error>();

    React.useEffect(() => {
        setPage(1);
        setData(undefined);
        setError(undefined);
    }, [props.selection?.pair.pairKind, props.selection?.block.blockNumber]);

    React.useEffect(() => {
        const selection = props.selection;
        if (!selection?.pair.pairKind || !selection.block.blockNumber) {
            return undefined;
        }
        let active = true;
        const request = services.tokenapi.listProjectSwapEvents(props.projectID, selection.pair.pairKind, selection.block.blockNumber, {page, pageSize: EVENT_PAGE_SIZE});
        setLoading(true);
        setError(undefined);
        request.then(
            next => {
                if (active) {
                    setData(next);
                    setLoading(false);
                }
            },
            reason => {
                if (active) {
                    setError(reason instanceof Error ? reason : new Error(String(reason?.message || reason)));
                    setLoading(false);
                }
            }
        );
        return () => {
            active = false;
            request.abort?.();
        };
    }, [page, props.projectID, props.selection?.pair.pairKind, props.selection?.block.blockNumber]);

    const pair = props.selection?.pair;
    const columns: ColumnsType<TokenProjectSwapEvent> = [
        {
            title: 'Transaction',
            width: 220,
            render: item => <ExplorerIdentifier chainID={props.chainID} kind='tx' value={item.transactionHash} />
        },
        {
            title: 'Position',
            width: 120,
            render: item => (
                <span className='swap-event-position'>
                    tx <ExactValue value={item.transactionIndex} /> · log <ExactValue value={item.logIndex} />
                </span>
            )
        },
        {
            title: 'Direction',
            width: 100,
            render: item => <StatusTag value={item.direction || 'Unknown'} positive={item.direction === 'buy'} negative={item.direction === 'sell'} />
        },
        {
            title: 'Transaction origin',
            width: 210,
            render: item => <ExplorerIdentifier chainID={props.chainID} kind='address' value={item.txFrom} />
        },
        {title: 'Base flow', width: 210, render: item => <EventFlow event={item} pair={pair} asset='base' />},
        {title: 'Quote flow', width: 210, render: item => <EventFlow event={item} pair={pair} asset='quote' />},
        {
            title: 'Effective price',
            width: 210,
            render: item => (
                <ExactValue
                    value={item.effectivePrice}
                    suffix={item.effectivePrice ? `${pair?.quoteAsset?.symbol || 'Quote'} / ${pair?.baseAsset?.symbol || 'Base'}` : undefined}
                />
            )
        }
    ];
    const rowKey = (item: TokenProjectSwapEvent) => `${item.transactionHash}-${item.logIndex}`;
    return (
        <Drawer
            className='swap-event-drawer'
            title={
                props.selection
                    ? `${pairLabel(props.selection.pair.pairKind)} · Block ${props.selection.block.blockNumber} · Sample ${props.selection.block.sampleIndex}`
                    : 'Swap events'
            }
            size='min(1120px, 90vw)'
            open={Boolean(props.selection)}
            onClose={props.onClose}
            destroyOnHidden={true}>
            {error && <Alert type='error' title='Swap events unavailable' description={error.message} showIcon={true} />}
            <Typography.Paragraph type='secondary'>
                Transaction origin is the enclosing transaction&apos;s <code>tx.from</code>. Sender and recipient below are the Pair event fields.
            </Typography.Paragraph>
            <Table<TokenProjectSwapEvent>
                className='swap-event-table'
                size='small'
                rowKey={rowKey}
                loading={loading}
                columns={columns}
                dataSource={data?.items || []}
                pagination={false}
                scroll={{x: 1280}}
                expandable={{
                    expandedRowRender: item => (
                        <div className='swap-event-raw'>
                            <span>
                                <Typography.Text type='secondary'>Event sender</Typography.Text>
                                <ExplorerIdentifier chainID={props.chainID} kind='address' value={item.sender} />
                            </span>
                            <span>
                                <Typography.Text type='secondary'>Event recipient</Typography.Text>
                                <ExplorerIdentifier chainID={props.chainID} kind='address' value={item.toAddress} />
                            </span>
                            {[
                                ['amount0In', item.amount0In],
                                ['amount1In', item.amount1In],
                                ['amount0Out', item.amount0Out],
                                ['amount1Out', item.amount1Out]
                            ].map(([label, value]) => (
                                <span key={label}>
                                    <Typography.Text type='secondary'>{label}</Typography.Text>
                                    <ExactValue value={value} tooltip={`${value || '-'} base units`} />
                                </span>
                            ))}
                        </div>
                    )
                }}
            />
            {!loading && !error && (data?.items.length || 0) === 0 && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='No Swap events in this block' />}
            {(data?.total || 0) > EVENT_PAGE_SIZE && (
                <div className='swap-event-drawer__pagination'>
                    <Typography.Text type='secondary'>{data?.total || 0} events</Typography.Text>
                    <Pagination current={page} pageSize={EVENT_PAGE_SIZE} total={data?.total || 0} showSizeChanger={false} onChange={setPage} />
                </div>
            )}
        </Drawer>
    );
};

export const ProjectSwapActivityTab = (props: {projectID: number; active: boolean; refreshVersion: number}) => {
    const [activity, setActivity] = React.useState<TokenProjectSwapActivity>();
    const [loading, setLoading] = React.useState(false);
    const [error, setError] = React.useState<Error>();
    const [selection, setSelection] = React.useState<SwapBlockSelection>();
    const requestRef = React.useRef<(Promise<TokenProjectSwapActivity | undefined> & {abort?: () => void}) | undefined>();
    const requestVersionRef = React.useRef(0);

    const load = React.useCallback(() => {
        if (requestRef.current) {
            return;
        }
        const requestVersion = ++requestVersionRef.current;
        const request = services.tokenapi.getProjectSwapActivity(props.projectID);
        requestRef.current = request;
        setLoading(true);
        setError(undefined);
        request.then(
            next => {
                if (requestVersionRef.current === requestVersion) {
                    setActivity(next);
                    setLoading(false);
                    requestRef.current = undefined;
                }
            },
            reason => {
                if (requestVersionRef.current === requestVersion) {
                    setError(reason instanceof Error ? reason : new Error(String(reason?.message || reason)));
                    setLoading(false);
                    requestRef.current = undefined;
                }
            }
        );
    }, [props.projectID]);

    React.useEffect(() => {
        requestVersionRef.current += 1;
        requestRef.current?.abort?.();
        requestRef.current = undefined;
        setActivity(undefined);
        setError(undefined);
        setSelection(undefined);
    }, [props.projectID]);

    React.useEffect(() => {
        if (props.active) {
            load();
        }
    }, [load, props.active, props.refreshVersion]);

    const collecting = activity?.pairs.some(pair => pair.status === 'collecting') || false;
    const shouldPoll = props.active && (!activity || collecting);
    React.useEffect(() => {
        if (!shouldPoll) {
            return undefined;
        }
        const refreshWhenVisible = () => {
            if (document.visibilityState === 'visible') {
                load();
            }
        };
        const timer = window.setInterval(refreshWhenVisible, ACTIVITY_REFRESH_INTERVAL_MS);
        document.addEventListener('visibilitychange', refreshWhenVisible);
        return () => {
            window.clearInterval(timer);
            document.removeEventListener('visibilitychange', refreshWhenVisible);
        };
    }, [load, shouldPoll]);

    React.useEffect(
        () => () => {
            requestVersionRef.current += 1;
            requestRef.current?.abort?.();
            requestRef.current = undefined;
        },
        []
    );

    const openBlock = (pair: TokenProjectSwapPairActivity, block: TokenProjectSwapBlock) => setSelection({pair, block});
    const orderedPairs = pairKinds.map(kind => activity?.pairs.find(pair => pair.pairKind === kind)).filter((pair): pair is TokenProjectSwapPairActivity => Boolean(pair));

    if (loading && !activity) {
        return (
            <div className='project-detail-tab swap-activity' aria-busy='true'>
                <Skeleton active={true} paragraph={{rows: 12}} />
            </div>
        );
    }

    return (
        <div className='project-detail-tab swap-activity' aria-busy={loading || undefined}>
            {error && <Alert type='error' title={activity ? 'Could not refresh Swap activity' : 'Swap activity unavailable'} description={error.message} showIcon={true} />}
            {!activity ? (
                !loading && <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='Swap activity is not available for this project' />
            ) : (
                <>
                    <div className='swap-activity__meta'>
                        <Typography.Text type='secondary'>
                            Updated <TimeValue value={activity.generatedAt} />
                        </Typography.Text>
                        {loading && <Typography.Text type='secondary'>Refreshing…</Typography.Text>}
                    </div>
                    <div className='swap-pair-grid'>
                        {pairKinds.map(kind => (
                            <PairCard key={kind} kind={kind} pair={activity.pairs.find(pair => pair.pairKind === kind)} chainID={activity.chainID} onSelect={openBlock} />
                        ))}
                    </div>
                    <SwapTimeline pairs={orderedPairs} onSelect={openBlock} />
                    <SwapBlockTable pairs={orderedPairs} onSelect={openBlock} />
                    <EventDrawer projectID={props.projectID} chainID={activity.chainID} selection={selection} onClose={() => setSelection(undefined)} />
                </>
            )}
        </div>
    );
};
