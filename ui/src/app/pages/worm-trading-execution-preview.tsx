import {ArrowDownOutlined, ArrowLeftOutlined, ArrowRightOutlined, ArrowUpOutlined, CloseOutlined, FileSearchOutlined, ReloadOutlined, SearchOutlined} from '@ant-design/icons';
import {Alert, Avatar, Button, Card, Checkbox, Drawer, Empty, Input, Progress, Space, Steps, Tag, Tooltip, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {Navigate, useNavigate, useParams, useSearchParams} from 'react-router-dom';
import {AppPage, ResourceTable, useAsyncData} from '../components';
import {AccountDataModule} from '../shared/access-modules';
import {Context, useAuthorization} from '../shared/context';
import {formatBeijingUnixSeconds} from '../shared/format';
import {services} from '../shared/services';
import {
    AbortableWormTradingPromise,
    ListWormTradingWalletConnectionsResult,
    WormExecutionPlan,
    WormExecutionPlanItem,
    WormExecutionPlanStep,
    WormExecutionPlanWallet,
    WormMarketCombination,
    WormTradingWalletConnectionItem,
    WormTradingWalletSummary
} from '../shared/services/worm-trading-service';
import {requestErrorDetails, requestErrorMessage} from '../shared/services/requests';
import {short} from './shared';

const connectionPageSize = 100;
const stepPageSizes = [20, 50, 100];
const planPollIntervalMS = 1_500;
const codeAcronyms: Record<string, string> = {
    api: 'API',
    hmac: 'HMAC',
    http: 'HTTP',
    https: 'HTTPS',
    id: 'ID',
    rpc: 'RPC',
    sol: 'SOL',
    usdc: 'USDC',
    url: 'URL'
};

const titleCase = (value: string) =>
    value
        .trim()
        .toLowerCase()
        .split(/[_-]+/)
        .filter(Boolean)
        .map(part => codeAcronyms[part] || `${part.slice(0, 1).toUpperCase()}${part.slice(1)}`)
        .join(' ');

const displayCode = (value: string, fallback = 'Unavailable') => titleCase(value) || fallback;
const displayUSDC = (value: string) => (value ? `${value} USDC` : '—');
const displayBalance = (value: string, symbol: 'SOL' | 'USDC') => (value ? `${value} ${symbol}` : 'Unavailable');
const connectionLabel = (state: string) => titleCase(state) || 'Unknown';
const marketEstimateSummary = (item?: WormExecutionPlanItem) => {
    if (!item) {
        return '';
    }
    const parts = [
        item.funds ? `${item.funds} USDC collateral` : '',
        item.leverage ? `${item.leverage}×` : '',
        item.estimate?.feeAmount ? `${item.estimate.feeAmount} USDC fee` : ''
    ];
    return parts.filter(Boolean).join(' · ');
};
const reasonPriority = [
    'OPPOSITE_SIDE_CONFLICT',
    'ALREADY_HELD',
    'REQUEST_IN_FLIGHT',
    'MARKET_UNAVAILABLE',
    'ESTIMATE_REJECTED',
    'LIQUIDITY_INSUFFICIENT',
    'INSUFFICIENT_USDC',
    'SKIPPED_AFTER_INSUFFICIENT_USDC'
];

const walletPresetGlyphs: Record<string, string> = {
    'star-violet': '★',
    'bolt-blue': 'ϟ',
    'gem-cyan': '◆',
    'leaf-green': '♧',
    'sun-amber': '☀',
    'flame-orange': '♨',
    'heart-rose': '♥',
    'moon-indigo': '☾'
};

const avatarPalettes = [
    ['#5b21b6', '#a78bfa'],
    ['#1d4ed8', '#60a5fa'],
    ['#0f766e', '#2dd4bf'],
    ['#166534', '#4ade80'],
    ['#a16207', '#fbbf24'],
    ['#c2410c', '#fb923c'],
    ['#be123c', '#fb7185'],
    ['#3730a3', '#818cf8']
];

const walletPaletteIndex = (address: string) => {
    let hash = 2166136261;
    for (const character of address) {
        hash ^= character.codePointAt(0) || 0;
        hash = Math.imul(hash, 16777619);
    }
    return (hash >>> 0) % avatarPalettes.length;
};

const ExecutionWalletAvatar = (props: {wallet: WormTradingWalletSummary; size?: number}) => {
    const glyph = walletPresetGlyphs[props.wallet.avatarPresetId];
    const palette = avatarPalettes[walletPaletteIndex(props.wallet.address)];
    const uploaded = props.wallet.avatarKind.toLowerCase() === 'upload' && props.wallet.avatarUrl;
    return (
        <Avatar
            aria-hidden='true'
            className={`wallet-avatar ${glyph ? `wallet-avatar--${props.wallet.avatarPresetId}` : 'wallet-avatar--generated'}`}
            size={props.size || 44}
            src={uploaded || undefined}
            style={glyph ? undefined : {background: `linear-gradient(145deg, ${palette[0]}, ${palette[1]})`}}>
            {glyph || 'S'}
        </Avatar>
    );
};

const WalletIdentity = (props: {wallet: WormTradingWalletSummary; compact?: boolean}) => (
    <div className='worm-preview-wallet-identity'>
        <ExecutionWalletAvatar wallet={props.wallet} size={props.compact ? 36 : 44} />
        <div>
            <Typography.Text strong={true} ellipsis={{tooltip: props.wallet.remark || 'Solana wallet'}}>
                {props.wallet.remark || 'Solana wallet'}
            </Typography.Text>
            <Tooltip title={props.wallet.address}>
                <code>{short(props.wallet.address, 8, 7)}</code>
            </Tooltip>
        </div>
    </div>
);

const loadConnectionInventory = (): AbortableWormTradingPromise<ListWormTradingWalletConnectionsResult> => {
    let currentRequest: AbortableWormTradingPromise<ListWormTradingWalletConnectionsResult> | undefined;
    let aborted = false;
    const promise = (async () => {
        const items: WormTradingWalletConnectionItem[] = [];
        let page = 1;
        let expectedTotal: number | undefined;
        let fetchedAt = 0;
        while (!aborted) {
            currentRequest = services.wormTrading.listWalletConnections(page, connectionPageSize);
            const response = await currentRequest;
            if (expectedTotal === undefined) {
                expectedTotal = response.total;
                fetchedAt = response.fetchedAt;
            } else if (response.total !== expectedTotal) {
                throw new Error('The Wallet inventory changed while it was loading. Refresh and try again.');
            }
            items.push(...response.items);
            if (items.length >= response.total) {
                if (items.length !== response.total) {
                    throw new Error('Worm Trading returned an invalid Wallet inventory.');
                }
                return {items, total: response.total, page: 1, pageSize: connectionPageSize, fetchedAt};
            }
            if (response.items.length === 0) {
                throw new Error('Worm Trading returned an incomplete Wallet inventory.');
            }
            page += 1;
        }
        throw new DOMException('The request was aborted.', 'AbortError');
    })() as AbortableWormTradingPromise<ListWormTradingWalletConnectionsResult>;
    promise.abort = () => {
        aborted = true;
        currentRequest?.abort?.();
    };
    return promise;
};

const emptyConnectionInventory = (): AbortableWormTradingPromise<ListWormTradingWalletConnectionsResult> => {
    const promise = Promise.resolve({
        items: [],
        total: 0,
        page: 1,
        pageSize: connectionPageSize,
        fetchedAt: 0
    }) as AbortableWormTradingPromise<ListWormTradingWalletConnectionsResult>;
    promise.abort = () => undefined;
    return promise;
};

const CombinationStep = (props: {combination: WormMarketCombination; onContinue: () => void}) => (
    <section className='worm-preview-combination' aria-labelledby='worm-preview-combination-heading'>
        <div className='worm-preview-section-heading'>
            <div>
                <Typography.Title id='worm-preview-combination-heading' level={2}>
                    {props.combination.name}
                </Typography.Title>
                <Typography.Text type='secondary'>Revision {props.combination.revision} · Review the frozen market order before selecting Wallets.</Typography.Text>
            </div>
            <Tag>
                {props.combination.items.length} {props.combination.items.length === 1 ? 'market' : 'markets'}
            </Tag>
        </div>
        <div className='worm-preview-combination__items'>
            {props.combination.items.map(item => (
                <div className='worm-preview-combination-item' key={item.marketConditionId}>
                    <span>{item.ordinal}</span>
                    <div>
                        <Typography.Text strong={true}>{item.marketTitle}</Typography.Text>
                        <Typography.Text type='secondary'>{item.eventTitle}</Typography.Text>
                    </div>
                    <Tag color={item.side === 'YES' ? 'green' : 'blue'}>{item.side}</Tag>
                </div>
            ))}
        </div>
        <div className='worm-preview-step-actions'>
            <Button type='primary' icon={<ArrowRightOutlined />} onClick={props.onContinue}>
                Choose Wallets
            </Button>
        </div>
    </section>
);

const WalletChoiceCard = (props: {item: WormTradingWalletConnectionItem; selected: boolean; onChange: (selected: boolean) => void}) => {
    const connected = props.item.connection.state === 'CONNECTED';
    return (
        <label className={`worm-preview-wallet-choice${props.selected ? ' worm-preview-wallet-choice--selected' : ''}${connected ? '' : ' worm-preview-wallet-choice--disabled'}`}>
            <Checkbox checked={props.selected} disabled={!connected} onChange={event => props.onChange(event.target.checked)} />
            <WalletIdentity wallet={props.item.wallet} />
            <Tag color={connected ? 'green' : 'default'}>{connectionLabel(props.item.connection.state)}</Tag>
            {!connected && <small>Connect this Wallet on Assets before creating a preview.</small>}
        </label>
    );
};

const SelectedWallets = (props: {items: WormTradingWalletConnectionItem[]; onMove: (index: number, direction: -1 | 1) => void; onRemove: (walletID: number) => void}) => (
    <div className='worm-preview-wallet-order'>
        <div className='worm-preview-wallet-order__heading'>
            <div>
                <Typography.Title level={2}>Execution order</Typography.Title>
                <Typography.Text type='secondary'>{props.items.length} selected</Typography.Text>
            </div>
            <Tag color={props.items.length > 0 ? 'processing' : 'default'}>{props.items.length}</Tag>
        </div>
        <div className='worm-preview-wallet-order__items'>
            {props.items.length === 0 ? (
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='Select at least one connected Wallet.' />
            ) : (
                props.items.map((item, index) => (
                    <Card className='worm-preview-selected-wallet' size='small' key={item.wallet.walletId}>
                        <span className='worm-preview-selected-wallet__ordinal'>{index + 1}</span>
                        <WalletIdentity wallet={item.wallet} compact={true} />
                        <div className='worm-preview-selected-wallet__actions'>
                            <Tooltip title='Move earlier'>
                                <Button
                                    type='text'
                                    size='small'
                                    icon={<ArrowUpOutlined />}
                                    aria-label={`Move ${item.wallet.remark || 'Wallet'} earlier`}
                                    disabled={index === 0}
                                    onClick={() => props.onMove(index, -1)}
                                />
                            </Tooltip>
                            <Tooltip title='Move later'>
                                <Button
                                    type='text'
                                    size='small'
                                    icon={<ArrowDownOutlined />}
                                    aria-label={`Move ${item.wallet.remark || 'Wallet'} later`}
                                    disabled={index === props.items.length - 1}
                                    onClick={() => props.onMove(index, 1)}
                                />
                            </Tooltip>
                            <Tooltip title='Remove'>
                                <Button
                                    type='text'
                                    size='small'
                                    danger={true}
                                    icon={<CloseOutlined />}
                                    aria-label={`Remove ${item.wallet.remark || 'Wallet'}`}
                                    onClick={() => props.onRemove(item.wallet.walletId)}
                                />
                            </Tooltip>
                        </div>
                    </Card>
                ))
            )}
        </div>
    </div>
);

const WalletsStep = (props: {
    inventory: WormTradingWalletConnectionItem[];
    loading: boolean;
    error?: Error;
    selectedIDs: number[];
    creating: boolean;
    onSelectedIDsChange: (ids: number[]) => void;
    onBack: () => void;
    onCreate: () => void;
    onReload: () => void;
    onOpenAssets: () => void;
}) => {
    const [query, setQuery] = React.useState('');
    const [orderOpen, setOrderOpen] = React.useState(false);
    const selectedSet = React.useMemo(() => new Set(props.selectedIDs), [props.selectedIDs]);
    const selectedItems = props.selectedIDs.flatMap(walletID => {
        const item = props.inventory.find(candidate => candidate.wallet.walletId === walletID);
        return item ? [item] : [];
    });
    const connectedItems = props.inventory.filter(item => item.connection.state === 'CONNECTED');
    const normalizedQuery = query.trim().toLowerCase();
    const visibleItems = props.inventory.filter(item =>
        [item.wallet.remark, item.wallet.address, String(item.wallet.walletId)].some(value => value.toLowerCase().includes(normalizedQuery))
    );
    const toggleWallet = (walletID: number, selected: boolean) => {
        if (selected) {
            if (!selectedSet.has(walletID)) {
                props.onSelectedIDsChange([...props.selectedIDs, walletID]);
            }
            return;
        }
        props.onSelectedIDsChange(props.selectedIDs.filter(item => item !== walletID));
    };
    const moveWallet = (index: number, direction: -1 | 1) => {
        const nextIndex = index + direction;
        if (nextIndex < 0 || nextIndex >= props.selectedIDs.length) {
            return;
        }
        const next = [...props.selectedIDs];
        [next[index], next[nextIndex]] = [next[nextIndex], next[index]];
        props.onSelectedIDsChange(next);
    };
    const selectedSummary = <SelectedWallets items={selectedItems} onMove={moveWallet} onRemove={walletID => toggleWallet(walletID, false)} />;
    return (
        <>
            <div className='worm-preview-wallet-layout'>
                <section className='worm-preview-wallet-picker' aria-labelledby='worm-preview-wallets-heading'>
                    <div className='worm-preview-section-heading'>
                        <div>
                            <Typography.Title id='worm-preview-wallets-heading' level={2}>
                                Connected Wallets
                            </Typography.Title>
                            <Typography.Text type='secondary'>Selection order becomes the Wallet execution order.</Typography.Text>
                        </div>
                        <Space wrap={true}>
                            <Button
                                size='small'
                                disabled={connectedItems.length === 0 || connectedItems.every(item => selectedSet.has(item.wallet.walletId))}
                                onClick={() =>
                                    props.onSelectedIDsChange([
                                        ...props.selectedIDs,
                                        ...connectedItems.filter(item => !selectedSet.has(item.wallet.walletId)).map(item => item.wallet.walletId)
                                    ])
                                }>
                                Select all connected
                            </Button>
                            <Button size='small' disabled={props.selectedIDs.length === 0} onClick={() => props.onSelectedIDsChange([])}>
                                Clear
                            </Button>
                        </Space>
                    </div>
                    <Input
                        allowClear={true}
                        prefix={<SearchOutlined />}
                        placeholder='Search Wallet name or address'
                        value={query}
                        onChange={event => setQuery(event.target.value)}
                    />
                    {props.error && (
                        <Alert
                            type='error'
                            showIcon={true}
                            title='Wallet inventory unavailable'
                            description={requestErrorMessage(props.error, 'Could not load the account Wallets.')}
                            action={
                                <Button size='small' icon={<ReloadOutlined />} onClick={props.onReload}>
                                    Retry
                                </Button>
                            }
                        />
                    )}
                    {!props.loading && !props.error && connectedItems.length === 0 && props.inventory.length > 0 && (
                        <Alert
                            type='warning'
                            showIcon={true}
                            title='No connected Wallets'
                            description='Connect at least one Solana Wallet on Assets before creating an execution preview.'
                            action={
                                <Button size='small' onClick={props.onOpenAssets}>
                                    Go to Assets
                                </Button>
                            }
                        />
                    )}
                    <div className='worm-preview-wallet-grid' aria-busy={props.loading || undefined}>
                        {props.loading ? (
                            Array.from({length: 4}, (_, index) => <Card key={index} loading={true} />)
                        ) : visibleItems.length === 0 ? (
                            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={props.inventory.length === 0 ? 'No Solana Wallets found.' : 'No Wallets match this search.'} />
                        ) : (
                            visibleItems.map(item => (
                                <WalletChoiceCard
                                    key={item.wallet.walletId}
                                    item={item}
                                    selected={selectedSet.has(item.wallet.walletId)}
                                    onChange={selected => toggleWallet(item.wallet.walletId, selected)}
                                />
                            ))
                        )}
                    </div>
                </section>
                <aside className='worm-preview-wallet-order-desktop'>{selectedSummary}</aside>
            </div>
            <div className='worm-preview-step-actions worm-preview-step-actions--split worm-preview-wallet-actions'>
                <Button icon={<ArrowLeftOutlined />} onClick={props.onBack}>
                    Combination
                </Button>
                <Button
                    type='primary'
                    icon={<FileSearchOutlined />}
                    loading={props.creating}
                    disabled={props.selectedIDs.length === 0 || props.loading || Boolean(props.error)}
                    onClick={props.onCreate}>
                    Build read-only preview
                </Button>
            </div>
            <div className='worm-preview-mobile-order'>
                <Button type='text' icon={<ArrowLeftOutlined />} aria-label='Back to combination' onClick={props.onBack} />
                <span>
                    <strong>{props.selectedIDs.length}</strong>
                    <small>Wallets selected</small>
                </span>
                <Button onClick={() => setOrderOpen(true)}>Review order</Button>
                <Button type='primary' loading={props.creating} disabled={props.selectedIDs.length === 0 || props.loading || Boolean(props.error)} onClick={props.onCreate}>
                    Build preview
                </Button>
            </div>
            <Drawer rootClassName='worm-preview-order-drawer' title='Wallet execution order' width={440} open={orderOpen} onClose={() => setOrderOpen(false)}>
                {selectedSummary}
            </Drawer>
        </>
    );
};

const PlanStateAlert = (props: {plan: WormExecutionPlan}) => {
    if (props.plan.state === 'BUILDING') {
        return (
            <Alert
                type='info'
                showIcon={true}
                title={`Building preview · ${displayCode(props.plan.buildStage, 'Queued')}`}
                description='Athena is refreshing market estimates, complete position/request pages, and confirmed Wallet balances. You can leave this page; no transaction is being created.'
            />
        );
    }
    if (props.plan.state === 'FAILED') {
        return (
            <Alert
                type='error'
                showIcon={true}
                title='Preview could not be completed'
                description={`${displayCode(props.plan.failureCode)}. No partial result can be used for execution.`}
            />
        );
    }
    if (props.plan.state === 'EXPIRED') {
        return (
            <Alert
                type='warning'
                showIcon={true}
                title='Preview expired'
                description='This snapshot is read-only and cannot be consumed. Create a fresh preview to obtain current balances, exposure, market state, and estimates.'
            />
        );
    }
    if (props.plan.usabilityCode) {
        return (
            <Alert
                type='warning'
                showIcon={true}
                title='Preview is not consumable'
                description={
                    props.plan.usabilityCode === 'NO_ACTIONABLE_STEPS'
                        ? 'Every step is already satisfied or deterministically skipped. There is nothing to execute.'
                        : `${displayCode(props.plan.usabilityCode)}. Review the source combination before creating a fresh preview.`
                }
            />
        );
    }
    return (
        <Alert
            type='success'
            showIcon={true}
            title='Read-only preview ready'
            description={`The snapshot expires at ${formatBeijingUnixSeconds(props.plan.expiresAt) || 'the server-defined time'}. This read-only preview cannot start an order.`}
        />
    );
};

const planProgress = (plan: WormExecutionPlan) => (plan.totalStepCount > 0 ? Math.min(100, Math.round((plan.completedStepCount / plan.totalStepCount) * 100)) : 0);
const planPresentationStatus = (plan: WormExecutionPlan) => {
    if (plan.state === 'BUILDING') {
        return {label: 'Building', color: 'blue'};
    }
    if (plan.state === 'FAILED') {
        return {label: 'Failed', color: 'red'};
    }
    if (plan.state === 'EXPIRED') {
        return {label: 'Expired', color: 'gold'};
    }
    return plan.usabilityCode ? {label: 'Preview built', color: 'gold'} : {label: 'Preview ready', color: 'green'};
};
const planLiveStatus = (plan: WormExecutionPlan) =>
    plan.state === 'BUILDING'
        ? `Building execution preview. ${plan.completedStepCount} of ${plan.totalStepCount} steps classified.`
        : plan.state === 'READY'
          ? plan.usabilityCode
              ? `Execution preview built and not consumable. ${plan.readyStepCount} actionable steps and ${plan.skippedStepCount} skipped.`
              : `Execution preview ready. ${plan.readyStepCount} actionable steps and ${plan.skippedStepCount} skipped.`
          : plan.state === 'EXPIRED'
            ? 'Execution preview expired.'
            : `Execution preview failed. ${displayCode(plan.failureCode)}.`;

const PlanSummary = (props: {plan: WormExecutionPlan}) => {
    const totalsAvailable = props.plan.state === 'READY' || props.plan.state === 'EXPIRED';
    const presentationStatus = planPresentationStatus(props.plan);
    return (
        <>
            <div className='worm-preview-plan-header'>
                <div>
                    <Typography.Title level={2}>{props.plan.combinationName}</Typography.Title>
                    <Typography.Text type='secondary'>
                        Revision {props.plan.combinationRevision} · Plan {short(props.plan.id, 10, 8)}
                    </Typography.Text>
                </div>
                <Tag color={presentationStatus.color}>{presentationStatus.label}</Tag>
            </div>
            <div className='worm-preview-live-status' role='status' aria-live='polite'>
                {planLiveStatus(props.plan)}
            </div>
            <PlanStateAlert plan={props.plan} />
            {props.plan.state === 'BUILDING' && (
                <div className='worm-preview-build-progress'>
                    <Progress percent={planProgress(props.plan)} status='active' />
                    <Typography.Text type='secondary'>
                        {props.plan.completedStepCount} of {props.plan.totalStepCount} steps classified
                    </Typography.Text>
                </div>
            )}
            <Typography.Text id='worm-preview-actionable-totals-note' type='secondary'>
                Collateral, opening fee, and USDC totals include actionable steps only.
            </Typography.Text>
            <dl className='worm-preview-metrics' aria-describedby='worm-preview-actionable-totals-note'>
                <div>
                    <dt>Wallets</dt>
                    <dd>{props.plan.walletCount}</dd>
                </div>
                <div>
                    <dt>Markets</dt>
                    <dd>{props.plan.itemCount}</dd>
                </div>
                <div>
                    <dt>Total steps</dt>
                    <dd>{props.plan.totalStepCount}</dd>
                </div>
                <div>
                    <dt>Actionable</dt>
                    <dd>{props.plan.readyStepCount}</dd>
                </div>
                <div>
                    <dt>Skipped</dt>
                    <dd>{props.plan.skippedStepCount}</dd>
                </div>
                <div>
                    <dt>Actionable collateral</dt>
                    <dd>{totalsAvailable ? displayUSDC(props.plan.maximumCollateral) : '—'}</dd>
                </div>
                <div>
                    <dt>Actionable opening fees</dt>
                    <dd>{totalsAvailable ? displayUSDC(props.plan.openingFeeEstimate) : '—'}</dd>
                </div>
                <div>
                    <dt>Actionable USDC needed</dt>
                    <dd>{totalsAvailable ? displayUSDC(props.plan.totalUSDCNeeded) : '—'}</dd>
                </div>
            </dl>
            {Object.entries(props.plan.reasonCounts).some(([reasonCode, count]) => reasonCode !== 'READY' && count > 0) && (
                <div className='worm-preview-reason-counts' aria-label='Skipped step reasons'>
                    {Object.entries(props.plan.reasonCounts)
                        .filter(([reasonCode, count]) => reasonCode !== 'READY' && count > 0)
                        .sort(([left], [right]) => {
                            const leftIndex = reasonPriority.indexOf(left);
                            const rightIndex = reasonPriority.indexOf(right);
                            return (leftIndex < 0 ? reasonPriority.length : leftIndex) - (rightIndex < 0 ? reasonPriority.length : rightIndex) || left.localeCompare(right);
                        })
                        .map(([reasonCode, count]) => (
                            <div key={reasonCode}>
                                <span>{displayCode(reasonCode)}</span>
                                <strong>{count}</strong>
                            </div>
                        ))}
                </div>
            )}
            {props.plan.wallets.length > 0 && (
                <div className='worm-preview-balance-strip'>
                    {props.plan.wallets.map(wallet => (
                        <div key={wallet.wallet.walletId}>
                            <span>{wallet.ordinal}</span>
                            <WalletIdentity wallet={wallet.wallet} compact={true} />
                            <dl>
                                <div>
                                    <dt>USDC</dt>
                                    <dd>{displayBalance(wallet.usdc.amount, 'USDC')}</dd>
                                </div>
                                <div>
                                    <dt>SOL</dt>
                                    <dd>{displayBalance(wallet.sol.amount, 'SOL')}</dd>
                                </div>
                            </dl>
                        </div>
                    ))}
                </div>
            )}
            <Alert
                type='info'
                showIcon={true}
                title='SOL is informational in Preview'
                description='Worm does not expose exact transaction fees or account rent through Estimate. Any execution must validate the authoritative SOL spending boundary before creating a draft.'
            />
        </>
    );
};

const PlanStepCard = (props: {step: WormExecutionPlanStep; wallet?: WormExecutionPlanWallet; item?: WormExecutionPlanItem}) => (
    <Card className='worm-preview-step-card' size='small'>
        <div className='worm-preview-step-card__heading'>
            <span>{props.step.ordinal}</span>
            <Tag color={props.step.disposition === 'READY' ? 'green' : 'default'}>
                {props.step.disposition === 'READY' ? 'Ready' : displayCode(props.step.reasonCode, 'Skipped')}
            </Tag>
        </div>
        {props.wallet && <WalletIdentity wallet={props.wallet.wallet} compact={true} />}
        <div className='worm-preview-step-card__market'>
            <Typography.Text strong={true}>{props.item?.marketTitle || `Market ${props.step.itemOrdinal}`}</Typography.Text>
            <span>
                {props.item && <Tag color={props.item.side === 'YES' ? 'green' : 'blue'}>{props.item.side}</Tag>}
                <Typography.Text type='secondary'>{props.item?.eventTitle || ''}</Typography.Text>
            </span>
            {props.item && <Typography.Text type='secondary'>{marketEstimateSummary(props.item)}</Typography.Text>}
        </div>
        <div className='worm-preview-step-card__balance'>
            <span>Projected USDC</span>
            <strong>
                {props.step.projectedUsdcBefore || '—'} → {props.step.projectedUsdcAfter || '—'}
            </strong>
        </div>
    </Card>
);

const PlanSteps = (props: {plan: WormExecutionPlan}) => {
    const [page, setPage] = React.useState(1);
    const [pageSize, setPageSize] = React.useState(50);
    React.useEffect(() => {
        setPage(1);
        setPageSize(50);
    }, [props.plan.id]);
    const data = useAsyncData(() => services.wormTrading.listExecutionPlanSteps(props.plan.id, page, pageSize), [props.plan.id, page, pageSize, props.plan.updatedAt]);
    const walletByOrdinal = (ordinal: number) => props.plan.wallets.find(wallet => wallet.ordinal === ordinal);
    const itemByOrdinal = (ordinal: number) => props.plan.items.find(item => item.ordinal === ordinal);
    const columns: ColumnsType<WormExecutionPlanStep> = [
        {title: '#', width: 72, dataIndex: 'ordinal'},
        {
            title: 'Wallet',
            width: 250,
            render: step => {
                const wallet = walletByOrdinal(step.walletOrdinal);
                return wallet ? <WalletIdentity wallet={wallet.wallet} compact={true} /> : `Wallet ${step.walletOrdinal}`;
            }
        },
        {
            title: 'Market and outcome',
            render: step => {
                const item = itemByOrdinal(step.itemOrdinal);
                return (
                    <div className='worm-preview-step-market'>
                        <Typography.Text strong={true}>{item?.marketTitle || `Market ${step.itemOrdinal}`}</Typography.Text>
                        <span>
                            {item && <Tag color={item.side === 'YES' ? 'green' : 'blue'}>{item.side}</Tag>}
                            <Typography.Text type='secondary'>{item?.eventTitle || ''}</Typography.Text>
                        </span>
                        {item && <Typography.Text type='secondary'>{marketEstimateSummary(item)}</Typography.Text>}
                    </div>
                );
            }
        },
        {
            title: 'Preview result',
            width: 220,
            render: step => (
                <div className='worm-preview-step-result'>
                    <Tag color={step.disposition === 'READY' ? 'green' : 'default'}>{step.disposition === 'READY' ? 'Ready' : 'Skipped'}</Tag>
                    <Typography.Text type='secondary'>{step.disposition === 'READY' ? 'Would buy at preview funds' : displayCode(step.reasonCode)}</Typography.Text>
                </div>
            )
        },
        {
            title: 'Projected USDC',
            width: 190,
            render: step => (
                <span className='worm-preview-step-usdc'>
                    {step.projectedUsdcBefore || '—'} → {step.projectedUsdcAfter || '—'}
                </span>
            )
        }
    ];
    return (
        <section className='worm-preview-steps' aria-labelledby='worm-preview-steps-heading'>
            <div className='worm-preview-section-heading'>
                <div>
                    <Typography.Title id='worm-preview-steps-heading' level={2}>
                        Wallet-major step preview
                    </Typography.Title>
                    <Typography.Text type='secondary'>Each Wallet is evaluated across every market before Athena moves to the next Wallet.</Typography.Text>
                </div>
            </div>
            {data.error && (
                <Alert
                    type='error'
                    showIcon={true}
                    title='Could not load preview steps'
                    description={requestErrorMessage(data.error)}
                    action={
                        <Button size='small' onClick={data.reload}>
                            Retry
                        </Button>
                    }
                />
            )}
            <ResourceTable<WormExecutionPlanStep>
                rowKey='ordinal'
                label='Execution preview steps'
                items={data.data?.items || []}
                columns={columns}
                loading={data.loading}
                total={data.data?.total}
                page={page}
                pageSize={pageSize}
                pageSizeOptions={stepPageSizes}
                onPageChange={(nextPage, nextPageSize) => {
                    setPage(nextPage);
                    setPageSize(nextPageSize);
                }}
                compactEmptyDescription='No classified steps are available.'
                compactRender={step => <PlanStepCard step={step} wallet={walletByOrdinal(step.walletOrdinal)} item={itemByOrdinal(step.itemOrdinal)} />}
            />
        </section>
    );
};

const ReviewStep = (props: {
    plan?: WormExecutionPlan;
    loading: boolean;
    error?: Error;
    creating: boolean;
    canChangeWallets: boolean;
    canRefresh: boolean;
    onRetryStatus: () => void;
    onBack: () => void;
    onRefresh: () => void;
}) => {
    if (!props.plan && props.loading) {
        return (
            <section className='worm-preview-review-loading' aria-live='polite'>
                <Progress type='circle' percent={0} status='active' />
                <Typography.Title level={2}>Loading execution preview</Typography.Title>
            </section>
        );
    }
    if (!props.plan) {
        return (
            <Alert
                type='error'
                showIcon={true}
                title='Execution preview unavailable'
                description={requestErrorMessage(props.error, 'The saved preview could not be loaded.')}
                action={
                    <Button size='small' onClick={props.onRetryStatus}>
                        Retry
                    </Button>
                }
            />
        );
    }
    return (
        <div className='worm-preview-review'>
            {props.error && (
                <Alert
                    type='warning'
                    showIcon={true}
                    title='Could not refresh preview status'
                    description='The last confirmed preview remains visible.'
                    action={
                        <Button size='small' onClick={props.onRetryStatus}>
                            Retry
                        </Button>
                    }
                />
            )}
            <PlanSummary plan={props.plan} />
            {props.plan.state !== 'BUILDING' && <PlanSteps plan={props.plan} />}
            <Alert
                className='worm-preview-read-only'
                type='warning'
                showIcon={true}
                title='No order can be started from this page'
                description='Execution Preview performs only Worm GET, List, and Estimate operations. It never creates a draft, asks for a Wallet signature, submits an order, or changes a position.'
            />
            <div className='worm-preview-step-actions worm-preview-step-actions--split'>
                <Button icon={<ArrowLeftOutlined />} onClick={props.onBack}>
                    {props.canChangeWallets ? 'Wallets' : 'Saved combinations'}
                </Button>
                {props.canChangeWallets && (
                    <Button icon={<ReloadOutlined />} loading={props.creating} disabled={!props.canRefresh || props.plan.state === 'BUILDING'} onClick={props.onRefresh}>
                        Refresh preview
                    </Button>
                )}
            </div>
        </div>
    );
};

export const WormTradingExecutionPreviewPage = () => {
    const ctx = React.useContext(Context);
    const authorization = useAuthorization();
    const canWrite = authorization.canWrite(AccountDataModule.WormTrading);
    const navigate = useNavigate();
    const {id = ''} = useParams();
    const [searchParams, setSearchParams] = useSearchParams();
    const planID = (searchParams.get('planId') || '').trim();
    const [workflowStep, setWorkflowStep] = React.useState(planID ? 2 : 0);
    const [selectedIDs, setSelectedIDs] = React.useState<number[]>([]);
    const [creating, setCreating] = React.useState(false);
    const [plan, setPlan] = React.useState<WormExecutionPlan>();
    const [planLoading, setPlanLoading] = React.useState(Boolean(planID));
    const [planError, setPlanError] = React.useState<Error>();
    const [planReload, setPlanReload] = React.useState(0);
    const initializedPlanIDRef = React.useRef('');
    const createRequestRef = React.useRef<ReturnType<typeof services.wormTrading.createExecutionPlan>>();
    const accountIDRef = React.useRef(authorization.user.accountId);
    const canWriteRef = React.useRef(canWrite);
    accountIDRef.current = authorization.user.accountId;
    canWriteRef.current = canWrite;
    const combination = useAsyncData(() => services.wormTrading.getMarketCombination(id), [authorization.user.accountId, id]);
    const inventory = useAsyncData(canWrite ? loadConnectionInventory : emptyConnectionInventory, [authorization.user.accountId, canWrite]);

    React.useEffect(
        () => () => {
            createRequestRef.current?.abort?.();
        },
        []
    );

    React.useEffect(() => {
        createRequestRef.current?.abort?.();
        createRequestRef.current = undefined;
        setCreating(false);
        setWorkflowStep(planID ? 2 : 0);
        setSelectedIDs([]);
        setPlan(undefined);
        setPlanError(undefined);
        initializedPlanIDRef.current = '';
        setPlanLoading(Boolean(planID));
    }, [authorization.user.accountId, id, planID]);

    React.useEffect(() => {
        if (!planID) {
            setPlanLoading(false);
            return;
        }
        let active = true;
        let timer: number | undefined;
        let request: ReturnType<typeof services.wormTrading.getExecutionPlan> | undefined;
        let continuePolling = true;
        let consecutiveFailures = 0;
        const schedulePoll = (delayMS: number) => {
            if (!active || !continuePolling) {
                return;
            }
            timer = window.setTimeout(poll, delayMS);
        };
        const scheduleExpiryPoll = (readyPlan: WormExecutionPlan, delayMS: number) => {
            if (!active || !continuePolling) {
                return;
            }
            timer = window.setTimeout(() => {
                setPlan(current =>
                    current?.id === readyPlan.id && current.state === 'READY' ? {...current, state: 'EXPIRED', usabilityCode: current.usabilityCode || 'EXPIRED'} : current
                );
                void poll();
            }, delayMS);
        };
        const poll = async () => {
            request = services.wormTrading.getExecutionPlan(planID);
            setPlanLoading(true);
            try {
                const next = await request;
                if (!active) {
                    return;
                }
                if (next.combinationId !== id) {
                    continuePolling = false;
                    throw new Error('This preview belongs to a different saved combination.');
                }
                setPlan(next);
                setPlanError(undefined);
                setPlanLoading(false);
                consecutiveFailures = 0;
                if (initializedPlanIDRef.current !== next.id && next.wallets.length > 0) {
                    initializedPlanIDRef.current = next.id;
                    setSelectedIDs(next.wallets.map(wallet => wallet.wallet.walletId));
                }
                if (next.state === 'BUILDING') {
                    schedulePoll(planPollIntervalMS);
                } else if (next.state === 'READY') {
                    const expiryDelayMS = Math.max(250, next.expiresAt * 1_000 - Date.now() + 250);
                    scheduleExpiryPoll(next, expiryDelayMS);
                } else {
                    continuePolling = false;
                }
            } catch (error) {
                if (active) {
                    setPlanError(error instanceof Error ? error : new Error(String(error)));
                    setPlanLoading(false);
                    const status = requestErrorDetails(error).status;
                    if (status >= 400 && status < 500 && status !== 429) {
                        continuePolling = false;
                    } else {
                        consecutiveFailures += 1;
                        schedulePoll(Math.min(30_000, planPollIntervalMS * 2 ** Math.min(consecutiveFailures, 4)));
                    }
                }
            }
        };
        void poll();
        return () => {
            active = false;
            if (timer !== undefined) {
                window.clearTimeout(timer);
            }
            request?.abort?.();
        };
    }, [authorization.user.accountId, id, planID, planReload]);

    React.useEffect(() => {
        if (!canWrite) {
            createRequestRef.current?.abort?.();
            createRequestRef.current = undefined;
            setCreating(false);
        }
    }, [canWrite]);

    if (!canWrite && !planID) {
        return <Navigate replace={true} to='/worm-trading/combinations' />;
    }

    const createPlan = async (walletIDs = selectedIDs) => {
        const source = combination.data;
        if (!source || creating || walletIDs.length === 0) {
            return;
        }
        setCreating(true);
        const operationAccountID = accountIDRef.current;
        const request = services.wormTrading.createExecutionPlan({
            combinationId: source.id,
            expectedCombinationRevision: source.revision,
            walletIds: walletIDs
        });
        createRequestRef.current = request;
        try {
            const created = await request;
            if (!canWriteRef.current || accountIDRef.current !== operationAccountID || createRequestRef.current !== request) {
                return;
            }
            setPlan(created);
            setPlanError(undefined);
            initializedPlanIDRef.current = created.id;
            setSelectedIDs(walletIDs);
            setWorkflowStep(2);
            const next = new URLSearchParams(searchParams);
            next.set('planId', created.id);
            setSearchParams(next, {replace: true});
        } catch (error) {
            if (!canWriteRef.current || accountIDRef.current !== operationAccountID || createRequestRef.current !== request) {
                return;
            }
            const details = requestErrorDetails(error);
            if (details.status === 409) {
                setWorkflowStep(0);
                combination.reload();
                ctx.notifications.error('Combination changed', 'Review the latest combination revision before building another preview.');
            } else {
                ctx.notifications.error('Could not create preview', requestErrorMessage(error, 'The previous preview remains unchanged.'));
            }
        } finally {
            if (createRequestRef.current === request) {
                createRequestRef.current = undefined;
                if (canWriteRef.current && accountIDRef.current === operationAccountID) {
                    setCreating(false);
                }
            }
        }
    };

    const refreshPlan = () => {
        if (!plan || !combination.data || combination.data.revision !== plan.combinationRevision) {
            setWorkflowStep(0);
            combination.reload();
            ctx.notifications.warning('Review the combination', 'The saved combination changed after this preview was created.');
            return;
        }
        const walletIDs = plan.wallets.map(wallet => wallet.wallet.walletId);
        void createPlan(walletIDs.length > 0 ? walletIDs : selectedIDs);
    };

    const combinationError = workflowStep < 2 ? combination.error : undefined;
    return (
        <AppPage
            title='Worm Trading Execution Preview'
            subtitle={
                canWrite
                    ? 'Select connected Wallets and build a read-only, wallet-major preview. No Worm order, draft, signature, or transaction is created.'
                    : 'Review this read-only, wallet-major preview. No Worm order, draft, signature, or transaction is created.'
            }
            loading={workflowStep === 0 && combination.loading}
            error={combinationError}
            onRefresh={workflowStep === 0 ? combination.reload : undefined}
            extra={
                <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/worm-trading/combinations')}>
                    Saved combinations
                </Button>
            }>
            <Steps
                className='worm-preview-workflow'
                current={workflowStep}
                responsive={true}
                items={[
                    {title: 'Combination', content: 'Confirm market order'},
                    {title: 'Wallets', content: 'Select and order'},
                    {title: 'Review', content: 'Read-only preflight'}
                ]}
            />
            {workflowStep === 0 && combination.data && <CombinationStep combination={combination.data} onContinue={() => setWorkflowStep(1)} />}
            {workflowStep === 1 && (
                <WalletsStep
                    inventory={inventory.data?.items || []}
                    loading={inventory.loading}
                    error={inventory.error}
                    selectedIDs={selectedIDs}
                    creating={creating}
                    onSelectedIDsChange={setSelectedIDs}
                    onBack={() => setWorkflowStep(0)}
                    onCreate={() => void createPlan()}
                    onReload={inventory.reload}
                    onOpenAssets={() => navigate('/worm-trading')}
                />
            )}
            {workflowStep === 2 && (
                <ReviewStep
                    plan={plan}
                    loading={planLoading}
                    error={planError}
                    creating={creating}
                    canChangeWallets={canWrite}
                    canRefresh={canWrite && Boolean(combination.data && (plan?.wallets.length || selectedIDs.length))}
                    onRetryStatus={() => setPlanReload(value => value + 1)}
                    onBack={() => (canWrite ? setWorkflowStep(1) : navigate('/worm-trading/combinations'))}
                    onRefresh={refreshPlan}
                />
            )}
        </AppPage>
    );
};
