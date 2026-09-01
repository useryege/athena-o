import {
    ArrowDownOutlined,
    ArrowUpOutlined,
    CheckOutlined,
    CloseOutlined,
    DeleteOutlined,
    EditOutlined,
    EyeOutlined,
    FileSearchOutlined,
    PlusOutlined,
    ReloadOutlined,
    SaveOutlined
} from '@ant-design/icons';
import {Alert, Button, Card, Drawer, Empty, Input, Modal, Space, Tag, Tooltip, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {Navigate, useBlocker, useNavigate, useParams} from 'react-router-dom';
import {AppPage, ResourceTable, useAsyncData} from '../../components';
import {AccountDataModule} from '../../shared/access-modules';
import {Context, useAuthorization} from '../../shared/context';
import {formatBeijingUnixSeconds} from '../../shared/format';
import {memberServices as services} from '../services';
import {
    WormMarketCombination,
    WormMarketCombinationItem,
    WormMarketOutcomeSide,
    WormTradingEvent,
    WormTradingEventMarket,
    WormTradingEventOutcome
} from '../../shared/services/worm-trading-service';
import {requestErrorDetails, requestErrorMessage} from '../../shared/services/requests';
import {short, usePagedParams} from '../../shared/pages/shared';

const combinationPageSizes = [20, 50, 100];
const eventConditionIDPattern = /^[1-9A-HJ-NP-Za-km-z]{32,44}$/;
const hasControlCharacters = (value: string) =>
    Array.from(value).some(character => {
        const codePoint = character.codePointAt(0) || 0;
        return codePoint <= 31 || (codePoint >= 127 && codePoint <= 159);
    });

const parseEventConditionID = (value: string): string => {
    const input = value.trim();
    if (!input) {
        throw new Error('Enter a Worm market URL or Event Condition ID.');
    }
    let conditionID = input;
    if (/^https?:\/\//i.test(input)) {
        let parsed: URL;
        try {
            parsed = new URL(input);
        } catch {
            throw new Error('Enter a valid Worm market URL.');
        }
        const hostname = parsed.hostname.toLowerCase();
        if (parsed.protocol !== 'https:' || (hostname !== 'www.worm.wtf' && hostname !== 'worm.wtf') || parsed.username || parsed.password) {
            throw new Error('Use an HTTPS market URL from worm.wtf.');
        }
        const parts = parsed.pathname.split('/').filter(Boolean);
        if (parts.length !== 2 || parts[0].toLowerCase() !== 'market') {
            throw new Error('The Worm URL must use /market/{eventConditionId}.');
        }
        try {
            conditionID = decodeURIComponent(parts[1]);
        } catch {
            throw new Error('The Worm URL contains an invalid Event Condition ID.');
        }
    }
    if (!eventConditionIDPattern.test(conditionID)) {
        throw new Error('Enter a valid Base58 Event Condition ID.');
    }
    return conditionID;
};

const unavailableLabel = (code: string) =>
    code
        .trim()
        .toLowerCase()
        .split('_')
        .filter(Boolean)
        .map(part => part.charAt(0).toUpperCase() + part.slice(1))
        .join(' ') || 'Unavailable';

const orderedSelections = (items: WormMarketCombinationItem[]): WormMarketCombinationItem[] => items.map((item, index) => ({...item, ordinal: index + 1}));

const selectionFingerprint = (name: string, items: WormMarketCombinationItem[]) =>
    JSON.stringify({
        name,
        items: items.map(item => ({eventConditionId: item.eventConditionId, marketConditionId: item.marketConditionId, side: item.side}))
    });

const combinationNameValidationMessage = (name: string) => {
    const normalized = name.trim();
    const length = Array.from(normalized).length;
    if (length === 0) {
        return 'Enter a combination name.';
    }
    if (length > 80 || hasControlCharacters(normalized)) {
        return 'Use 1–80 characters without control characters.';
    }
    return '';
};

const validCombinationName = (name: string) => !combinationNameValidationMessage(name);

const selectionFromOutcome = (event: WormTradingEvent, market: WormTradingEventMarket, outcome: WormTradingEventOutcome, ordinal: number): WormMarketCombinationItem => ({
    ordinal,
    eventConditionId: event.eventConditionId,
    eventTitle: event.title,
    eventLogo: event.logo,
    marketConditionId: market.marketConditionId,
    marketTitle: market.title,
    marketLogo: market.logo,
    side: outcome.side,
    outcomeLabel: outcome.label
});

const refreshedSelections = (items: WormMarketCombinationItem[], events: WormTradingEvent[]) =>
    orderedSelections(
        items.map(item => {
            const event = events.find(candidate => candidate.eventConditionId === item.eventConditionId);
            const market = event?.markets.find(candidate => candidate.marketConditionId === item.marketConditionId);
            const outcome = market?.outcomes.find(candidate => candidate.side === item.side);
            return event && market && outcome ? selectionFromOutcome(event, market, outcome, item.ordinal) : item;
        })
    );

const combinationItemStatus = (item: WormMarketCombinationItem, events: WormTradingEvent[]) => {
    const event = events.find(candidate => candidate.eventConditionId === item.eventConditionId);
    if (!event) {
        return {known: false, selectable: false, code: ''};
    }
    const market = event.markets.find(candidate => candidate.marketConditionId === item.marketConditionId);
    if (!market) {
        return {known: true, selectable: false, code: 'MARKET_NOT_FOUND'};
    }
    const outcome = market.outcomes.find(candidate => candidate.side === item.side);
    if (!outcome) {
        return {known: true, selectable: false, code: 'OUTCOME_NOT_FOUND'};
    }
    return {known: true, selectable: outcome.selectable, code: outcome.unavailableCode || market.unavailableCode};
};

const exactDecimalParts = (value: string): {integer: bigint; scale: number} | undefined => {
    if (!/^\d+(?:\.\d+)?$/.test(value)) {
        return undefined;
    }
    const [whole, fraction = ''] = value.split('.');
    return {integer: BigInt(`${whole}${fraction}`), scale: fraction.length};
};

const lastTradeTenthsOfCent = (value: string): bigint | undefined => {
    const parsed = exactDecimalParts(value);
    if (!parsed) {
        return undefined;
    }
    const denominator = 10n ** BigInt(parsed.scale);
    const scaled = parsed.integer * 1000n;
    const quotient = scaled / denominator;
    return quotient + ((scaled % denominator) * 2n >= denominator ? 1n : 0n);
};

const formatTenthsOfCent = (tenths: bigint | undefined): string => {
    if (tenths === undefined) {
        return '—';
    }
    const wholeCents = tenths / 10n;
    const fractionalCent = tenths % 10n;
    return fractionalCent === 0n ? `${wholeCents}¢` : `${wholeCents}.${fractionalCent}¢`;
};

const marketLastTradeCents = (market: WormTradingEventMarket): Record<WormMarketOutcomeSide, string> => {
    const yes = market.outcomes.find(outcome => outcome.side === 'YES')?.lastTradePrice || '';
    const no = market.outcomes.find(outcome => outcome.side === 'NO')?.lastTradePrice || '';
    const yesTenths = lastTradeTenthsOfCent(yes);
    if (!yes || !no || yesTenths === undefined || yesTenths < 0n || yesTenths > 1000n) {
        return {YES: '—', NO: '—'};
    }
    return {
        YES: formatTenthsOfCent(yesTenths),
        NO: formatTenthsOfCent(1000n - yesTenths)
    };
};

const lastTradeTooltip = (value: string) =>
    value
        ? `Last trade: ${value} USDC/share. This is historical, not a current buy quote or guaranteed execution price.`
        : 'Last trade price is unavailable. This does not prevent selecting an otherwise available outcome.';

const lastTradeAccessiblePrice = (value: string, displayPrice: string) => (value ? `${displayPrice} last trade` : 'last trade price unavailable');

const selectionMarket = (item: WormMarketCombinationItem, events: WormTradingEvent[]) =>
    events.find(event => event.eventConditionId === item.eventConditionId)?.markets.find(market => market.marketConditionId === item.marketConditionId);

const selectionOutcome = (item: WormMarketCombinationItem, events: WormTradingEvent[]) => selectionMarket(item, events)?.outcomes.find(outcome => outcome.side === item.side);

const eventSnapshotPresentation = (fetchedAt: number) => {
    const full = formatBeijingUnixSeconds(fetchedAt);
    const parts = full?.split(' ') || [];
    return {
        full: full || 'Snapshot time unavailable',
        time: parts[parts.length - 1] || '—'
    };
};

const CombinationListCard = (props: {item: WormMarketCombination; canWrite: boolean; deleting: boolean; onOpen: () => void; onPreview: () => void; onDelete: () => void}) => (
    <Card className='worm-combination-list-card' size='small'>
        <div className='worm-combination-list-card__heading'>
            <div>
                <Typography.Title level={3}>{props.item.name}</Typography.Title>
                <Typography.Text type='secondary'>Revision {props.item.revision}</Typography.Text>
            </div>
            <Tag>
                {props.item.items.length} {props.item.items.length === 1 ? 'market' : 'markets'}
            </Tag>
        </div>
        <div className='worm-combination-list-card__preview'>
            {props.item.items.slice(0, 3).map(item => (
                <span key={item.marketConditionId}>
                    <Tag color={item.side === 'YES' ? 'green' : 'blue'}>{item.side}</Tag>
                    <Typography.Text ellipsis={true}>{item.marketTitle}</Typography.Text>
                </span>
            ))}
            {props.item.items.length > 3 && <Typography.Text type='secondary'>+{props.item.items.length - 3} more</Typography.Text>}
        </div>
        <Typography.Text className='worm-combination-list-card__updated' type='secondary'>
            Updated {formatBeijingUnixSeconds(props.item.updatedAt) || '-'}
        </Typography.Text>
        <Space className='worm-combination-list-card__actions' wrap={true}>
            <Button icon={props.canWrite ? <EditOutlined /> : <EyeOutlined />} onClick={props.onOpen}>
                {props.canWrite ? 'Edit' : 'View'}
            </Button>
            {props.canWrite && (
                <Button icon={<FileSearchOutlined />} onClick={props.onPreview}>
                    Preview execution
                </Button>
            )}
            {props.canWrite && (
                <Button danger={true} icon={<DeleteOutlined />} loading={props.deleting} onClick={props.onDelete}>
                    Delete
                </Button>
            )}
        </Space>
    </Card>
);

export const WormTradingCombinationsPage = () => {
    const ctx = React.useContext(Context);
    const authorization = useAuthorization();
    const canWrite = authorization.canWrite(AccountDataModule.WormTrading);
    const canWriteRef = React.useRef(canWrite);
    canWriteRef.current = canWrite;
    const accountIDRef = React.useRef(authorization.user.accountId);
    accountIDRef.current = authorization.user.accountId;
    const navigate = useNavigate();
    const {page, pageSize, setPage} = usePagedParams(20, combinationPageSizes);
    const [deletingIDs, setDeletingIDs] = React.useState<Set<string>>(() => new Set());
    const data = useAsyncData(() => services.wormTrading.listMarketCombinations(page, pageSize), [authorization.user.accountId, page, pageSize]);

    React.useEffect(() => {
        if (!canWrite) {
            setDeletingIDs(new Set());
        }
    }, [canWrite]);

    const deleteCombination = (item: WormMarketCombination) => {
        if (!canWrite) {
            return;
        }
        ctx.modal.confirm({
            title: `Delete ${item.name}?`,
            content: `This permanently removes the saved combination and its ${item.items.length} selected ${item.items.length === 1 ? 'market' : 'markets'}.`,
            okText: 'Delete combination',
            onOk: async () => {
                const operationAccountID = accountIDRef.current;
                setDeletingIDs(current => new Set(current).add(item.id));
                try {
                    await services.wormTrading.deleteMarketCombination(item.id, item.revision);
                    if (!canWriteRef.current || accountIDRef.current !== operationAccountID) {
                        return;
                    }
                    ctx.notifications.success('Combination deleted', item.name);
                    if ((data.data?.items.length || 0) === 1 && page > 1) {
                        setPage(page - 1, pageSize);
                    } else {
                        data.reload();
                    }
                } catch (error) {
                    if (canWriteRef.current && accountIDRef.current === operationAccountID) {
                        const details = requestErrorDetails(error);
                        if (details.status === 409) {
                            ctx.notifications.error('Combination changed', 'Reload the latest revision before deleting this combination.');
                        } else {
                            ctx.notifications.error('Could not delete combination', requestErrorMessage(error, 'The saved combination was not deleted.'));
                        }
                    }
                } finally {
                    setDeletingIDs(current => {
                        const next = new Set(current);
                        next.delete(item.id);
                        return next;
                    });
                }
            }
        });
    };

    const items = data.data?.items || [];
    const columns: ColumnsType<WormMarketCombination> = [
        {
            title: 'Name',
            render: item => (
                <Button type='link' className='worm-combination-name-link' onClick={() => navigate(`/worm-trading/combinations/${encodeURIComponent(item.id)}/edit`)}>
                    {item.name}
                </Button>
            )
        },
        {title: 'Markets', width: 110, render: item => item.items.length},
        {title: 'Revision', width: 100, dataIndex: 'revision'},
        {title: 'Updated', width: 190, render: item => formatBeijingUnixSeconds(item.updatedAt) || '-'},
        {
            title: 'Actions',
            width: canWrite ? 370 : 100,
            render: item => (
                <Space>
                    <Button
                        size='small'
                        icon={canWrite ? <EditOutlined /> : <EyeOutlined />}
                        onClick={() => navigate(`/worm-trading/combinations/${encodeURIComponent(item.id)}/edit`)}>
                        {canWrite ? 'Edit' : 'View'}
                    </Button>
                    {canWrite && (
                        <Button size='small' icon={<FileSearchOutlined />} onClick={() => navigate(`/worm-trading/combinations/${encodeURIComponent(item.id)}/execute`)}>
                            Preview execution
                        </Button>
                    )}
                    {canWrite && (
                        <Button size='small' danger={true} icon={<DeleteOutlined />} loading={deletingIDs.has(item.id)} onClick={() => deleteCombination(item)}>
                            Delete
                        </Button>
                    )}
                </Space>
            )
        }
    ];

    const empty = !data.loading && !data.error && (data.data?.total || 0) === 0;
    return (
        <AppPage
            title='Worm Trading Combinations'
            subtitle={
                canWrite
                    ? 'Build reusable combinations by selecting one YES or NO outcome from each Worm child market.'
                    : 'Review the saved market combinations available to this account.'
            }
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}
            extra={
                canWrite ? (
                    <Button type='primary' icon={<PlusOutlined />} onClick={() => navigate('/worm-trading/combinations/new')}>
                        New combination
                    </Button>
                ) : null
            }>
            {empty ? (
                <div className='worm-combination-empty'>
                    <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='No saved combinations'>
                        {canWrite && (
                            <Button type='primary' icon={<PlusOutlined />} onClick={() => navigate('/worm-trading/combinations/new')}>
                                Create your first combination
                            </Button>
                        )}
                    </Empty>
                </div>
            ) : (
                <ResourceTable<WormMarketCombination>
                    rowKey='id'
                    label='Saved Worm market combinations'
                    items={items}
                    columns={columns}
                    loading={data.loading}
                    total={data.data?.total}
                    page={page}
                    pageSize={pageSize}
                    pageSizeOptions={combinationPageSizes}
                    onPageChange={setPage}
                    compactEmptyDescription='No saved combinations on this page'
                    compactRender={item => (
                        <CombinationListCard
                            item={item}
                            canWrite={canWrite}
                            deleting={deletingIDs.has(item.id)}
                            onOpen={() => navigate(`/worm-trading/combinations/${encodeURIComponent(item.id)}/edit`)}
                            onPreview={() => navigate(`/worm-trading/combinations/${encodeURIComponent(item.id)}/execute`)}
                            onDelete={() => deleteCombination(item)}
                        />
                    )}
                />
            )}
        </AppPage>
    );
};

const MarketOutcomeChoice = (props: {outcome: WormTradingEventOutcome; displayPrice: string; selected: boolean}) => (
    <span className='worm-combination-outcome-choice'>
        <span className='worm-combination-outcome-choice__check' aria-hidden='true'>
            {props.selected ? <CheckOutlined /> : null}
        </span>
        <strong>{props.outcome.side}</strong>
        <span className='worm-combination-outcome-choice__price'>{props.displayPrice}</span>
    </span>
);

const EventMarketCard = (props: {
    market: WormTradingEventMarket;
    selectedSide?: WormMarketOutcomeSide;
    canWrite: boolean;
    onToggle: (outcome: WormTradingEventOutcome) => void;
}) => {
    const displayPrices = marketLastTradeCents(props.market);
    const availabilityMessages = props.market.unavailableCode
        ? [`Market unavailable: ${unavailableLabel(props.market.unavailableCode)}`]
        : props.market.outcomes.filter(outcome => !outcome.selectable).map(outcome => `${outcome.side} unavailable: ${unavailableLabel(outcome.unavailableCode)}`);
    return (
        <div className={`worm-combination-market${props.selectedSide ? ' worm-combination-market--selected' : ''}`}>
            <Typography.Text className='worm-combination-market__title' strong={true} ellipsis={true} title={props.market.title}>
                {props.market.title}
            </Typography.Text>
            <div className='worm-combination-market__outcomes' role='group' aria-label={`Outcome for ${props.market.title}`}>
                {props.market.outcomes.map(outcome => {
                    const selected = props.selectedSide === outcome.side;
                    return (
                        <Tooltip
                            key={outcome.side}
                            title={`${lastTradeTooltip(outcome.lastTradePrice)}${outcome.selectable ? '' : ` ${unavailableLabel(outcome.unavailableCode)}.`}`}>
                            <span className='worm-combination-outcome-tooltip'>
                                <Button
                                    className={`worm-combination-outcome-button worm-combination-outcome-button--${outcome.side.toLowerCase()}`}
                                    disabled={!props.canWrite || !outcome.selectable}
                                    aria-label={`${props.market.title}, ${outcome.side}, ${lastTradeAccessiblePrice(outcome.lastTradePrice, displayPrices[outcome.side])}${outcome.selectable ? '' : `, unavailable: ${unavailableLabel(outcome.unavailableCode)}`}`}
                                    aria-pressed={selected}
                                    onClick={() => props.onToggle(outcome)}>
                                    <MarketOutcomeChoice outcome={outcome} displayPrice={displayPrices[outcome.side]} selected={selected} />
                                </Button>
                            </span>
                        </Tooltip>
                    );
                })}
            </div>
            {availabilityMessages.length > 0 && (
                <div className='worm-combination-market__availability'>
                    {availabilityMessages.map(message => (
                        <span key={message}>{message}</span>
                    ))}
                </div>
            )}
        </div>
    );
};

const EventExplorerCard = (props: {
    event: WormTradingEvent;
    selections: WormMarketCombinationItem[];
    canWrite: boolean;
    refreshing: boolean;
    refreshDisabled: boolean;
    refreshError: string;
    onToggle: (market: WormTradingEventMarket, outcome: WormTradingEventOutcome) => void;
    onRefresh: () => void;
    onRemoveEvent: () => void;
}) => {
    const snapshot = eventSnapshotPresentation(props.event.fetchedAt);
    return (
        <section className='worm-combination-event' aria-labelledby={`worm-event-${props.event.eventConditionId}`}>
            <div className='worm-combination-event__heading'>
                <div className='worm-combination-event__copy'>
                    <Typography.Title id={`worm-event-${props.event.eventConditionId}`} level={2} title={props.event.title}>
                        {props.event.title}
                    </Typography.Title>
                    <div className='worm-combination-event__meta'>
                        <span>
                            {props.event.markets.length} {props.event.markets.length === 1 ? 'market' : 'markets'}
                        </span>
                        <Tooltip title={`Athena fetched this price snapshot at ${snapshot.full} Beijing time. It is not the last trade's execution time.`}>
                            <span>Last trade · fetched {snapshot.time}</span>
                        </Tooltip>
                    </div>
                </div>
                <Button
                    className='worm-combination-event__refresh'
                    size='small'
                    icon={<ReloadOutlined />}
                    loading={props.refreshing}
                    disabled={props.refreshDisabled}
                    aria-label={`Refresh prices for ${props.event.title}`}
                    onClick={props.onRefresh}>
                    Refresh
                </Button>
                {props.canWrite && (
                    <Tooltip title='Remove this event from the builder'>
                        <Button
                            className='worm-combination-event__remove'
                            size='small'
                            aria-label={`Remove ${props.event.title}`}
                            icon={<CloseOutlined />}
                            disabled={props.refreshDisabled}
                            onClick={props.onRemoveEvent}
                        />
                    </Tooltip>
                )}
            </div>
            {props.refreshError && (
                <Alert
                    className='worm-combination-event__refresh-error'
                    type='warning'
                    showIcon={true}
                    title='Could not refresh this Event'
                    description={`${props.refreshError} The previous snapshot remains visible.`}
                />
            )}
            {props.event.markets.length === 0 ? (
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='This Event has no child markets.' />
            ) : (
                <div className='worm-combination-event__markets'>
                    {props.event.markets.map(market => (
                        <EventMarketCard
                            key={market.marketConditionId}
                            market={market}
                            selectedSide={props.selections.find(item => item.marketConditionId === market.marketConditionId)?.side}
                            canWrite={props.canWrite}
                            onToggle={outcome => props.onToggle(market, outcome)}
                        />
                    ))}
                </div>
            )}
        </section>
    );
};

const CombinationSummary = (props: {
    name: string;
    items: WormMarketCombinationItem[];
    events: WormTradingEvent[];
    canWrite: boolean;
    saving: boolean;
    refreshing: boolean;
    editing: boolean;
    onNameChange: (value: string) => void;
    onMove: (index: number, direction: -1 | 1) => void;
    onRemove: (marketConditionID: string) => void;
    onSave: () => void;
}) => {
    const nameValid = validCombinationName(props.name);
    const invalidSelections = props.items.filter(item => {
        const status = combinationItemStatus(item, props.events);
        return status.known && !status.selectable;
    });
    const saveDisabled = (props.editing && !nameValid) || props.items.length === 0 || invalidSelections.length > 0 || props.saving || props.refreshing;
    return (
        <div className='worm-combination-summary'>
            <div className='worm-combination-summary__heading'>
                <div>
                    <Typography.Title level={2}>Current combination</Typography.Title>
                    <Typography.Text type='secondary'>{props.items.length} selected</Typography.Text>
                </div>
                <Tag color={props.items.length > 0 ? 'processing' : 'default'}>{props.items.length}</Tag>
            </div>
            {props.editing && (
                <label className='worm-combination-summary__name'>
                    <span>Combination name</span>
                    {props.canWrite ? (
                        <Input
                            value={props.name}
                            status={props.name && !nameValid ? 'error' : undefined}
                            placeholder='e.g. Weekend macro signals'
                            disabled={props.saving}
                            aria-invalid={props.name && !nameValid ? true : undefined}
                            onChange={event => props.onNameChange(event.target.value)}
                        />
                    ) : (
                        <Typography.Text strong={true}>{props.name}</Typography.Text>
                    )}
                    {props.canWrite && props.name && !nameValid && <small>Use 1–80 characters without control characters.</small>}
                </label>
            )}
            {invalidSelections.length > 0 && (
                <Alert
                    type='warning'
                    showIcon={true}
                    title={`${invalidSelections.length} selected ${invalidSelections.length === 1 ? 'market is' : 'markets are'} no longer available`}
                    description='Remove or replace unavailable selections before saving.'
                />
            )}
            <div className='worm-combination-summary__items'>
                {props.items.length === 0 ? (
                    <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='Choose YES or NO from a child market.' />
                ) : (
                    props.items.map((item, index) => {
                        const status = combinationItemStatus(item, props.events);
                        const market = selectionMarket(item, props.events);
                        const outcome = selectionOutcome(item, props.events);
                        const price = outcome?.lastTradePrice || '';
                        const displayPrice = market ? marketLastTradeCents(market)[item.side] : '—';
                        return (
                            <Card className='worm-combination-selection' size='small' key={item.marketConditionId}>
                                <div className='worm-combination-selection__ordinal'>{index + 1}</div>
                                <div className='worm-combination-selection__copy'>
                                    <Typography.Text type='secondary' ellipsis={true} title={item.eventTitle}>
                                        {item.eventTitle}
                                    </Typography.Text>
                                    <Typography.Text strong={true} ellipsis={true} title={item.marketTitle}>
                                        {item.marketTitle}
                                    </Typography.Text>
                                    <span>
                                        <Tag color={item.side === 'YES' ? 'green' : 'blue'}>{item.side}</Tag>
                                        <Tooltip title={lastTradeTooltip(price)}>
                                            <Typography.Text className='worm-combination-selection__price'>· {displayPrice}</Typography.Text>
                                        </Tooltip>
                                        {status.known && !status.selectable && <Tag color='warning'>{unavailableLabel(status.code)}</Tag>}
                                    </span>
                                </div>
                                {props.canWrite && (
                                    <div className='worm-combination-selection__actions'>
                                        <Tooltip title='Move up'>
                                            <Button
                                                aria-label={`Move ${item.marketTitle} up`}
                                                size='small'
                                                icon={<ArrowUpOutlined />}
                                                disabled={index === 0 || props.saving}
                                                onClick={() => props.onMove(index, -1)}
                                            />
                                        </Tooltip>
                                        <Tooltip title='Move down'>
                                            <Button
                                                aria-label={`Move ${item.marketTitle} down`}
                                                size='small'
                                                icon={<ArrowDownOutlined />}
                                                disabled={index === props.items.length - 1 || props.saving}
                                                onClick={() => props.onMove(index, 1)}
                                            />
                                        </Tooltip>
                                        <Tooltip title='Remove'>
                                            <Button
                                                aria-label={`Remove ${item.marketTitle}`}
                                                size='small'
                                                danger={true}
                                                icon={<DeleteOutlined />}
                                                disabled={props.saving}
                                                onClick={() => props.onRemove(item.marketConditionId)}
                                            />
                                        </Tooltip>
                                    </div>
                                )}
                            </Card>
                        );
                    })
                )}
            </div>
            {props.canWrite && (
                <Button className='worm-combination-summary__save' type='primary' icon={<SaveOutlined />} loading={props.saving} disabled={saveDisabled} onClick={props.onSave}>
                    {props.editing ? 'Save changes' : 'Create combination'}
                </Button>
            )}
        </div>
    );
};

const CreateCombinationNameModal = (props: {
    open: boolean;
    value: string;
    error: string;
    submitting: boolean;
    onChange: (value: string) => void;
    onCancel: () => void;
    onSubmit: () => void;
}) => (
    <Modal
        className='worm-combination-name-modal'
        width={480}
        open={props.open}
        title='Create combination'
        footer={null}
        destroyOnHidden={true}
        closable={!props.submitting}
        keyboard={!props.submitting}
        mask={{closable: !props.submitting}}
        onCancel={props.onCancel}>
        <form
            className='worm-combination-name-modal__form'
            onSubmit={event => {
                event.preventDefault();
                props.onSubmit();
            }}>
            <label className='worm-combination-name-modal__field' htmlFor='worm-combination-create-name'>
                <span>Combination name</span>
                <Input
                    id='worm-combination-create-name'
                    value={props.value}
                    autoFocus={true}
                    autoComplete='off'
                    placeholder='e.g. Weekend macro signals'
                    status={props.error ? 'error' : undefined}
                    disabled={props.submitting}
                    aria-invalid={props.error ? true : undefined}
                    aria-describedby='worm-combination-create-name-help'
                    onChange={event => props.onChange(event.target.value)}
                />
                <small id='worm-combination-create-name-help' className={props.error ? 'worm-combination-name-modal__error' : undefined} role={props.error ? 'alert' : undefined}>
                    {props.error || 'Use 1–80 characters without control characters.'}
                </small>
            </label>
            <div className='worm-combination-name-modal__actions'>
                <Button htmlType='button' disabled={props.submitting} onClick={props.onCancel}>
                    Cancel
                </Button>
                <Button type='primary' htmlType='submit' loading={props.submitting} disabled={!validCombinationName(props.value)}>
                    Create combination
                </Button>
            </div>
        </form>
    </Modal>
);

export const WormTradingCombinationBuilderPage = () => {
    const {id = ''} = useParams<{id: string}>();
    const editing = Boolean(id);
    const ctx = React.useContext(Context);
    const authorization = useAuthorization();
    const canWrite = authorization.canWrite(AccountDataModule.WormTrading);
    const canWriteRef = React.useRef(canWrite);
    canWriteRef.current = canWrite;
    const accountIDRef = React.useRef(authorization.user.accountId);
    accountIDRef.current = authorization.user.accountId;
    const navigate = useNavigate();
    const [name, setName] = React.useState('');
    const [selections, setSelections] = React.useState<WormMarketCombinationItem[]>([]);
    const [events, setEvents] = React.useState<WormTradingEvent[]>([]);
    const [baseline, setBaseline] = React.useState(() => selectionFingerprint('', []));
    const [initialized, setInitialized] = React.useState(!editing);
    const [eventInput, setEventInput] = React.useState('');
    const [eventInputError, setEventInputError] = React.useState('');
    const [eventLoadErrors, setEventLoadErrors] = React.useState<string[]>([]);
    const [eventRefreshErrors, setEventRefreshErrors] = React.useState<Record<string, string>>({});
    const [addingEvent, setAddingEvent] = React.useState(false);
    const [hydratingEvents, setHydratingEvents] = React.useState(false);
    const [refreshingEventID, setRefreshingEventID] = React.useState('');
    const [saving, setSaving] = React.useState(false);
    const [reviewOpen, setReviewOpen] = React.useState(false);
    const [createNameOpen, setCreateNameOpen] = React.useState(false);
    const [createName, setCreateName] = React.useState('');
    const [createNameError, setCreateNameError] = React.useState('');
    const addEventRequestRef = React.useRef<ReturnType<typeof services.wormTrading.getEvent>>();
    const refreshEventRequestRef = React.useRef<ReturnType<typeof services.wormTrading.getEvent>>();
    const allowNavigationRef = React.useRef(false);
    const currentDraft = selectionFingerprint(name, selections);
    const dirty = canWrite && initialized && currentDraft !== baseline;
    const combination = useAsyncData<WormMarketCombination | undefined>(
        () => (editing ? services.wormTrading.getMarketCombination(id) : Promise.resolve(undefined)),
        [authorization.user.accountId, editing, id]
    );
    const blocker = useBlocker(() => dirty && !allowNavigationRef.current);

    React.useEffect(
        () => () => {
            addEventRequestRef.current?.abort?.();
            refreshEventRequestRef.current?.abort?.();
        },
        []
    );

    React.useEffect(() => {
        refreshEventRequestRef.current?.abort?.();
        refreshEventRequestRef.current = undefined;
        setRefreshingEventID('');
        setEventRefreshErrors({});
    }, [authorization.user.accountId]);

    React.useEffect(() => {
        setCreateNameOpen(false);
        setCreateName('');
        setCreateNameError('');
    }, [authorization.user.accountId, editing]);

    React.useEffect(() => {
        if (!dirty) {
            return;
        }
        const beforeUnload = (event: BeforeUnloadEvent) => {
            event.preventDefault();
            event.returnValue = '';
        };
        window.addEventListener('beforeunload', beforeUnload);
        return () => window.removeEventListener('beforeunload', beforeUnload);
    }, [dirty]);

    React.useEffect(() => {
        if (blocker.state !== 'blocked') {
            return;
        }
        let resolved = false;
        const handle = ctx.modal.confirm({
            title: 'Discard unsaved combination changes?',
            content: editing ? 'Your name, market choices, and ordering changes have not been saved.' : 'Your market choices and ordering changes have not been saved.',
            okText: 'Discard and leave',
            onOk: () => {
                resolved = true;
                blocker.proceed();
            },
            onCancel: () => {
                resolved = true;
                blocker.reset();
            }
        });
        return () => {
            if (!resolved) {
                handle.destroy();
            }
        };
    }, [blocker.state, ctx.modal]);

    React.useEffect(() => {
        const source = combination.data;
        if (!editing || !source) {
            return;
        }
        const nextSelections = orderedSelections(source.items);
        setName(source.name);
        setSelections(nextSelections);
        setBaseline(selectionFingerprint(source.name, nextSelections));
        setInitialized(true);
        setEventLoadErrors([]);
        setEventRefreshErrors({});
        setEvents([]);
        const eventIDs = Array.from(new Set(source.items.map(item => item.eventConditionId)));
        const requests = eventIDs.map(eventConditionID => services.wormTrading.getEvent(eventConditionID));
        let active = true;
        setHydratingEvents(requests.length > 0);
        Promise.all(
            requests.map((request, index) =>
                request.then(
                    event => ({event, eventConditionID: eventIDs[index]}),
                    error => ({error, eventConditionID: eventIDs[index]})
                )
            )
        ).then(results => {
            if (!active) {
                return;
            }
            const loaded = results.flatMap(result => ('event' in result ? [result.event] : []));
            const failed = results.flatMap(result => ('error' in result ? [result.eventConditionID] : []));
            setEvents(loaded);
            setEventLoadErrors(failed);
            setSelections(current => refreshedSelections(current, loaded));
            setHydratingEvents(false);
        });
        return () => {
            active = false;
            requests.forEach(request => request.abort?.());
        };
    }, [authorization.user.accountId, combination.data, editing]);

    React.useEffect(() => {
        if (canWrite) {
            return;
        }
        setReviewOpen(false);
        setSaving(false);
        setCreateNameOpen(false);
        setCreateName('');
        setCreateNameError('');
        const source = combination.data;
        if (source) {
            const nextSelections = orderedSelections(source.items);
            setName(source.name);
            setSelections(nextSelections);
            setBaseline(selectionFingerprint(source.name, nextSelections));
        }
    }, [canWrite, combination.data]);

    if (!editing && !canWrite) {
        return <Navigate replace={true} to='/worm-trading/combinations' />;
    }

    const addEvent = async () => {
        if (!canWrite || addingEvent || hydratingEvents || refreshingEventID || refreshEventRequestRef.current) {
            return;
        }
        let eventConditionID: string;
        try {
            eventConditionID = parseEventConditionID(eventInput);
        } catch (error) {
            setEventInputError(requestErrorMessage(error, 'Enter a valid Worm event.'));
            return;
        }
        if (events.some(event => event.eventConditionId === eventConditionID)) {
            setEventInputError('This Event is already in the builder.');
            return;
        }
        setAddingEvent(true);
        setEventInputError('');
        const operationAccountID = accountIDRef.current;
        const request = services.wormTrading.getEvent(eventConditionID);
        addEventRequestRef.current = request;
        try {
            const event = await request;
            if (!canWriteRef.current || accountIDRef.current !== operationAccountID) {
                return;
            }
            const existingMarketIDs = new Set(events.flatMap(item => item.markets.map(market => market.marketConditionId)));
            if (event.markets.some(market => existingMarketIDs.has(market.marketConditionId))) {
                throw new Error('Worm returned a child market that is already loaded under another Event.');
            }
            setEvents(current => [...current, event]);
            setEventLoadErrors(current => current.filter(item => item !== eventConditionID));
            setEventRefreshErrors(current => {
                const next = {...current};
                delete next[eventConditionID];
                return next;
            });
            setSelections(current => refreshedSelections(current, [event]));
            setEventInput('');
        } catch (error) {
            if (canWriteRef.current && accountIDRef.current === operationAccountID) {
                setEventInputError(requestErrorMessage(error, 'Could not load this Worm Event.'));
            }
        } finally {
            if (addEventRequestRef.current === request) {
                addEventRequestRef.current = undefined;
                setAddingEvent(false);
            }
        }
    };

    const refreshEvent = async (event: WormTradingEvent) => {
        if (refreshingEventID || refreshEventRequestRef.current || addEventRequestRef.current || addingEvent || hydratingEvents || saving) {
            return;
        }
        const eventConditionID = event.eventConditionId;
        const operationAccountID = accountIDRef.current;
        const otherMarketIDs = new Set(
            events.filter(candidate => candidate.eventConditionId !== eventConditionID).flatMap(candidate => candidate.markets.map(market => market.marketConditionId))
        );
        setRefreshingEventID(eventConditionID);
        setEventRefreshErrors(current => {
            const next = {...current};
            delete next[eventConditionID];
            return next;
        });
        const request = services.wormTrading.getEvent(eventConditionID);
        refreshEventRequestRef.current = request;
        try {
            const refreshed = await request;
            if (accountIDRef.current !== operationAccountID || refreshEventRequestRef.current !== request) {
                return;
            }
            if (refreshed.markets.some(market => otherMarketIDs.has(market.marketConditionId))) {
                throw new Error('Worm returned a child market that is already loaded under another Event.');
            }
            setEvents(current => current.map(candidate => (candidate.eventConditionId === eventConditionID ? refreshed : candidate)));
            setSelections(current => refreshedSelections(current, [refreshed]));
        } catch (error) {
            if (accountIDRef.current === operationAccountID && refreshEventRequestRef.current === request) {
                setEventRefreshErrors(current => ({...current, [eventConditionID]: requestErrorMessage(error, 'Could not load the latest Worm Event snapshot.')}));
            }
        } finally {
            if (refreshEventRequestRef.current === request) {
                refreshEventRequestRef.current = undefined;
                setRefreshingEventID('');
            }
        }
    };

    const toggleOutcome = (event: WormTradingEvent, market: WormTradingEventMarket, outcome: WormTradingEventOutcome) => {
        if (!canWrite || !outcome.selectable) {
            return;
        }
        setSelections(current => {
            const existingIndex = current.findIndex(item => item.marketConditionId === market.marketConditionId);
            if (existingIndex < 0) {
                return orderedSelections([...current, selectionFromOutcome(event, market, outcome, current.length + 1)]);
            }
            if (current[existingIndex].side === outcome.side) {
                return orderedSelections(current.filter((_, index) => index !== existingIndex));
            }
            const next = [...current];
            next[existingIndex] = selectionFromOutcome(event, market, outcome, existingIndex + 1);
            return orderedSelections(next);
        });
    };

    const removeSelection = (marketConditionID: string) => {
        if (canWrite) {
            setSelections(current => orderedSelections(current.filter(item => item.marketConditionId !== marketConditionID)));
        }
    };

    const removeEvent = (event: WormTradingEvent) => {
        if (!canWrite) {
            return;
        }
        const selectedMarketIDs = new Set(event.markets.map(market => market.marketConditionId));
        const selectedCount = selections.filter(item => selectedMarketIDs.has(item.marketConditionId)).length;
        const remove = () => {
            setEvents(current => current.filter(item => item.eventConditionId !== event.eventConditionId));
            setSelections(current => orderedSelections(current.filter(item => !selectedMarketIDs.has(item.marketConditionId))));
            setEventRefreshErrors(current => {
                const next = {...current};
                delete next[event.eventConditionId];
                return next;
            });
        };
        if (selectedCount === 0) {
            remove();
            return;
        }
        ctx.modal.confirm({
            title: `Remove ${event.title}?`,
            content: `This also removes ${selectedCount} selected ${selectedCount === 1 ? 'market' : 'markets'} from the current combination.`,
            okText: 'Remove event',
            onOk: remove
        });
    };

    const moveSelection = (index: number, direction: -1 | 1) => {
        if (!canWrite) {
            return;
        }
        setSelections(current => {
            const target = index + direction;
            if (index < 0 || index >= current.length || target < 0 || target >= current.length) {
                return current;
            }
            const next = [...current];
            [next[index], next[target]] = [next[target], next[index]];
            return orderedSelections(next);
        });
    };

    const save = async (submittedName: string) => {
        if (!canWrite || saving || refreshingEventID || !validCombinationName(submittedName) || selections.length === 0) {
            return;
        }
        const invalidSelection = selections.find(item => {
            const status = combinationItemStatus(item, events);
            return status.known && !status.selectable;
        });
        if (invalidSelection) {
            ctx.notifications.warning('Review unavailable markets', 'Remove or replace unavailable selections before saving.');
            return;
        }
        const input = {
            name: submittedName.trim(),
            items: selections.map(item => ({eventConditionId: item.eventConditionId, marketConditionId: item.marketConditionId, side: item.side}))
        };
        setSaving(true);
        const operationAccountID = accountIDRef.current;
        try {
            const saved = editing
                ? await services.wormTrading.updateMarketCombination(id, {...input, expectedRevision: combination.data?.revision || 0})
                : await services.wormTrading.createMarketCombination(input);
            if (!canWriteRef.current || accountIDRef.current !== operationAccountID) {
                return;
            }
            setBaseline(selectionFingerprint(saved.name, saved.items));
            allowNavigationRef.current = true;
            ctx.notifications.success(editing ? 'Combination updated' : 'Combination created', saved.name);
            navigate('/worm-trading/combinations', {replace: true});
        } catch (error) {
            if (!canWriteRef.current || accountIDRef.current !== operationAccountID) {
                return;
            }
            const details = requestErrorDetails(error);
            if (details.status === 409) {
                if (details.message === 'market combination name already exists') {
                    if (editing) {
                        ctx.notifications.error('Combination name already exists', 'Choose a different name for this account.');
                    } else {
                        setCreateNameError('A combination with this name already exists in this account.');
                    }
                } else {
                    ctx.notifications.error('Combination changed', 'Reload the latest revision before applying your changes again.');
                }
            } else {
                ctx.notifications.error(editing ? 'Could not update combination' : 'Could not create combination', requestErrorMessage(error));
            }
        } finally {
            setSaving(false);
        }
    };

    const openCreateNameModal = () => {
        if (!canWrite || editing || saving || refreshingEventID || selections.length === 0) {
            return;
        }
        const invalidSelection = selections.find(item => {
            const status = combinationItemStatus(item, events);
            return status.known && !status.selectable;
        });
        if (invalidSelection) {
            ctx.notifications.warning('Review unavailable markets', 'Remove or replace unavailable selections before saving.');
            return;
        }
        setCreateName('');
        setCreateNameError('');
        setCreateNameOpen(true);
    };

    const closeCreateNameModal = () => {
        if (saving) {
            return;
        }
        setCreateNameOpen(false);
        setCreateName('');
        setCreateNameError('');
    };

    const changeCreateName = (value: string) => {
        setCreateName(value);
        setCreateNameError(value ? combinationNameValidationMessage(value) : '');
    };

    const submitCreateName = () => {
        const validationError = combinationNameValidationMessage(createName);
        if (validationError) {
            setCreateNameError(validationError);
            return;
        }
        void save(createName);
    };

    const summary = (
        <CombinationSummary
            name={name}
            items={selections}
            events={events}
            canWrite={canWrite}
            saving={saving}
            refreshing={Boolean(refreshingEventID)}
            editing={editing}
            onNameChange={setName}
            onMove={moveSelection}
            onRemove={removeSelection}
            onSave={editing ? () => void save(name) : openCreateNameModal}
        />
    );

    const loading = editing && (combination.loading || !initialized);
    return (
        <AppPage
            title={editing ? (canWrite ? 'Edit Worm Trading Combination' : 'Worm Trading Combination') : 'New Worm Trading Combination'}
            subtitle={
                canWrite
                    ? 'Add one or more Worm Events, choose exactly one direction per child market, and arrange the saved order.'
                    : 'This is a read-only view of the saved market choices and their order.'
            }
            loading={loading}
            error={combination.error}>
            {!loading && !combination.error && (
                <>
                    {canWrite && (
                        <Card className='worm-combination-event-input' size='small'>
                            <div>
                                <Typography.Title level={2}>Add a Worm Event</Typography.Title>
                                <Typography.Text type='secondary'>Paste a worm.wtf market URL or its Event Condition ID.</Typography.Text>
                            </div>
                            <div className='worm-combination-event-input__control'>
                                <Input
                                    value={eventInput}
                                    placeholder='https://www.worm.wtf/market/... or Event Condition ID'
                                    status={eventInputError ? 'error' : undefined}
                                    aria-invalid={eventInputError ? true : undefined}
                                    disabled={addingEvent || hydratingEvents || Boolean(refreshingEventID) || saving}
                                    onChange={event => {
                                        setEventInput(event.target.value);
                                        if (eventInputError) {
                                            setEventInputError('');
                                        }
                                    }}
                                    onPressEnter={() => void addEvent()}
                                />
                                <Button
                                    type='primary'
                                    icon={<PlusOutlined />}
                                    loading={addingEvent}
                                    disabled={!eventInput.trim() || hydratingEvents || Boolean(refreshingEventID) || saving}
                                    onClick={() => void addEvent()}>
                                    Add Event
                                </Button>
                            </div>
                            {eventInputError && (
                                <Typography.Text className='worm-combination-event-input__error' type='danger' role='alert'>
                                    {eventInputError}
                                </Typography.Text>
                            )}
                        </Card>
                    )}
                    {eventLoadErrors.length > 0 && (
                        <Alert
                            className='worm-combination-load-alert'
                            type='warning'
                            showIcon={true}
                            title='Some saved Events could not be refreshed'
                            description={`${eventLoadErrors.map(item => short(item, 8, 6)).join(', ')}. Existing saved selections remain visible in the review panel.`}
                        />
                    )}
                    {hydratingEvents && <Alert className='worm-combination-load-alert' type='info' showIcon={true} title='Refreshing current child markets…' />}
                    <div className='worm-combination-builder'>
                        <main className='worm-combination-builder__events'>
                            {events.length === 0 && !hydratingEvents ? (
                                <div className='worm-combination-builder__empty'>
                                    <Empty
                                        image={Empty.PRESENTED_IMAGE_SIMPLE}
                                        description={canWrite ? 'Add a Worm Event to explore its child markets.' : 'Current Event details are unavailable.'}
                                    />
                                </div>
                            ) : (
                                events.map(event => (
                                    <EventExplorerCard
                                        key={event.eventConditionId}
                                        event={event}
                                        selections={selections}
                                        canWrite={canWrite}
                                        refreshing={refreshingEventID === event.eventConditionId}
                                        refreshDisabled={Boolean(refreshingEventID) || addingEvent || hydratingEvents || saving}
                                        refreshError={eventRefreshErrors[event.eventConditionId] || ''}
                                        onToggle={(market, outcome) => toggleOutcome(event, market, outcome)}
                                        onRefresh={() => void refreshEvent(event)}
                                        onRemoveEvent={() => removeEvent(event)}
                                    />
                                ))
                            )}
                        </main>
                        <aside className='worm-combination-builder__summary'>{summary}</aside>
                    </div>
                    <div className='worm-combination-mobile-review'>
                        <span>
                            <strong>{selections.length}</strong>
                            <small>{selections.length === 1 ? 'market selected' : 'markets selected'}</small>
                        </span>
                        <Button type='primary' onClick={() => setReviewOpen(true)}>
                            Review combination
                        </Button>
                    </div>
                    <Drawer
                        rootClassName='worm-combination-review-drawer'
                        title='Review combination'
                        placement='right'
                        width={480}
                        open={reviewOpen}
                        onClose={() => setReviewOpen(false)}>
                        {summary}
                    </Drawer>
                    <CreateCombinationNameModal
                        open={!editing && createNameOpen}
                        value={createName}
                        error={createNameError}
                        submitting={saving}
                        onChange={changeCreateName}
                        onCancel={closeCreateNameModal}
                        onSubmit={submitCreateName}
                    />
                </>
            )}
        </AppPage>
    );
};
