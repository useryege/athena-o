import {
    ArrowDownOutlined,
    ArrowLeftOutlined,
    ArrowRightOutlined,
    ArrowUpOutlined,
    CloseOutlined,
    FileSearchOutlined,
    ReloadOutlined,
    SafetyCertificateOutlined,
    SearchOutlined,
    WarningOutlined
} from '@ant-design/icons';
import {Alert, Avatar, Button, Card, Checkbox, Drawer, Empty, Input, Modal, Progress, Space, Steps, Tag, Tooltip, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {Navigate, useNavigate, useParams, useSearchParams} from 'react-router-dom';
import {AppPage, ResourceTable, useAsyncData} from '../../components';
import {AccountDataModule} from '../../shared/access-modules';
import {Context, useAuthorization} from '../../shared/context';
import {formatBeijingUnixSeconds} from '../../shared/format';
import {memberServices as services} from '../services';
import {
    AbortableWormTradingPromise,
    ListWormTradingWalletConnectionsResult,
    WormExecutionPlan,
    WormExecutionPlanItem,
    WormExecutionPlanStep,
    WormExecutionPlanWallet,
    WormExecutionPreflightChecks,
    WormMarketCombination,
    WormTradingWalletConnectionItem,
    WormTradingWalletSummary
} from '../../shared/services/worm-trading-service';
import {requestErrorDetails, requestErrorMessage} from '../../shared/services/requests';
import {
    createDefaultWormExecutionPreflightChecks,
    disabledWormExecutionPreflightChecks,
    wormExecutionMandatoryGuardDefinitions,
    wormExecutionPreflightCheckDefinitions
} from './worm-execution-preflight';
import {short} from '../../shared/pages/shared';

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
    'MARKET_POSITION_EXISTS',
    'WALLET_REQUEST_IN_FLIGHT',
    'MARKET_UNAVAILABLE',
    'ESTIMATE_REJECTED',
    'LIQUIDITY_INSUFFICIENT',
    'INSUFFICIENT_USDC',
    'SKIPPED_AFTER_INSUFFICIENT_USDC'
];

const preflightChecksEqual = (left: WormExecutionPreflightChecks, right: WormExecutionPreflightChecks) => left.requireFullLiquidity === right.requireFullLiquidity;

interface ExecutionPlanIntent {
    accountId: string;
    id: string;
    combinationId: string;
    combinationRevision: number;
    walletIds: number[];
    preflightChecks: WormExecutionPreflightChecks;
}

const executionPlanIntent = (plan: WormExecutionPlan, accountId: string): ExecutionPlanIntent => ({
    accountId,
    id: plan.id,
    combinationId: plan.combinationId,
    combinationRevision: plan.combinationRevision,
    walletIds: plan.wallets.map(wallet => wallet.wallet.walletId),
    preflightChecks: {...plan.preflightChecks}
});

const executionPlanMatchesIntent = (plan: WormExecutionPlan, intent: ExecutionPlanIntent) => {
    const walletIds = plan.wallets.map(wallet => wallet.wallet.walletId);
    return (
        plan.id === intent.id &&
        plan.combinationId === intent.combinationId &&
        plan.combinationRevision === intent.combinationRevision &&
        walletIds.length === intent.walletIds.length &&
        walletIds.every((walletId, index) => walletId === intent.walletIds[index]) &&
        preflightChecksEqual(plan.preflightChecks, intent.preflightChecks)
    );
};

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
    onSelectedIDsChange: (ids: number[]) => void;
    onBack: () => void;
    onContinue: () => void;
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
                <Button type='primary' icon={<ArrowRightOutlined />} disabled={props.selectedIDs.length === 0 || props.loading || Boolean(props.error)} onClick={props.onContinue}>
                    Continue to checks
                </Button>
            </div>
            <div className='worm-preview-mobile-order'>
                <Button type='text' icon={<ArrowLeftOutlined />} aria-label='Back to combination' onClick={props.onBack} />
                <span>
                    <strong>{props.selectedIDs.length}</strong>
                    <small>Wallets selected</small>
                </span>
                <Button onClick={() => setOrderOpen(true)}>Review order</Button>
                <Button type='primary' disabled={props.selectedIDs.length === 0 || props.loading || Boolean(props.error)} onClick={props.onContinue}>
                    Checks
                </Button>
            </div>
            <Drawer rootClassName='worm-preview-order-drawer' title='Wallet execution order' width={440} open={orderOpen} onClose={() => setOrderOpen(false)}>
                {selectedSummary}
            </Drawer>
        </>
    );
};

const PreflightChecksEditor = (props: {checks: WormExecutionPreflightChecks; onChange: (checks: WormExecutionPreflightChecks) => void; disabled?: boolean; idPrefix: string}) => {
    const disabledChecks = disabledWormExecutionPreflightChecks(props.checks);
    return (
        <div className='worm-preview-checks-editor'>
            <div className='worm-preview-checks-editor__heading'>
                <div>
                    <Typography.Text strong={true}>Execution guards</Typography.Text>
                    <Typography.Text type='secondary'>Exposure guards are mandatory. Full liquidity is the only configurable preview rule.</Typography.Text>
                </div>
            </div>
            <div className='worm-preview-guard-layout'>
                <section className='worm-preview-guard-group' aria-labelledby={`${props.idPrefix}-mandatory-heading`}>
                    <div className='worm-preview-guard-group__heading'>
                        <Typography.Text id={`${props.idPrefix}-mandatory-heading`} strong={true}>
                            Mandatory guards
                        </Typography.Text>
                        <Tag color='blue'>Always on</Tag>
                    </div>
                    <div className='worm-preview-guard-list'>
                        {wormExecutionMandatoryGuardDefinitions.map(definition => (
                            <div className='worm-preview-check-card worm-preview-check-card--mandatory' key={definition.key}>
                                <SafetyCertificateOutlined aria-hidden='true' />
                                <span>
                                    <Typography.Text strong={true}>{definition.title}</Typography.Text>
                                    <Typography.Text type='secondary'>{definition.description}</Typography.Text>
                                </span>
                                <Tag color='blue'>Guarded</Tag>
                            </div>
                        ))}
                    </div>
                </section>
                <section className='worm-preview-guard-group' aria-labelledby={`${props.idPrefix}-optional-heading`}>
                    <div className='worm-preview-guard-group__heading'>
                        <Typography.Text id={`${props.idPrefix}-optional-heading`} strong={true}>
                            Optional fill rule
                        </Typography.Text>
                        <Tag>1× estimate</Tag>
                    </div>
                    <div className='worm-preview-guard-list'>
                        {wormExecutionPreflightCheckDefinitions.map(definition => {
                            const descriptionID = `${props.idPrefix}-${definition.key}-description`;
                            return (
                                <label
                                    className={`worm-preview-check-card${props.checks[definition.key] ? ' worm-preview-check-card--enabled' : ' worm-preview-check-card--disabled'}`}
                                    key={definition.key}>
                                    <Checkbox
                                        checked={props.checks[definition.key]}
                                        disabled={props.disabled}
                                        aria-describedby={descriptionID}
                                        onChange={event => props.onChange({...props.checks, [definition.key]: event.target.checked})}
                                    />
                                    <span>
                                        <Typography.Text strong={true}>{definition.title}</Typography.Text>
                                        <Typography.Text id={descriptionID} type='secondary'>
                                            {props.checks[definition.key] ? definition.description : definition.ignoredDescription}
                                        </Typography.Text>
                                    </span>
                                    <Tag color={props.checks[definition.key] ? 'green' : 'gold'}>{props.checks[definition.key] ? 'Checked' : 'Ignored'}</Tag>
                                </label>
                            );
                        })}
                    </div>
                </section>
            </div>
            {disabledChecks.length > 0 && (
                <Alert
                    type='warning'
                    showIcon={true}
                    title='Full liquidity is not required'
                    description='A partial-fill estimate becomes an ignored liquidity warning. Both mandatory exposure guards still skip deterministically.'
                />
            )}
            <Alert
                type='info'
                showIcon={true}
                title='Market intent and 1× are fixed'
                description='The selected target market, side, backend funds, and 1× leverage come from the saved combination and server policy. Wallet authority, market validity, Estimate integrity, balances, permissions, and mutation protections also remain mandatory.'
            />
        </div>
    );
};

const ChecksStep = (props: {
    checks: WormExecutionPreflightChecks;
    creating: boolean;
    selectedWalletCount: number;
    onChange: (checks: WormExecutionPreflightChecks) => void;
    onBack: () => void;
    onCreate: () => void;
}) => (
    <section className='worm-preview-checks' aria-labelledby='worm-preview-checks-heading'>
        <div className='worm-preview-section-heading'>
            <div>
                <Typography.Title id='worm-preview-checks-heading' level={2}>
                    Preview checks
                </Typography.Title>
                <Typography.Text type='secondary'>Review the always-on exposure guards and choose whether the fixed 1× order requires a full estimated fill.</Typography.Text>
            </div>
            <Tag color='processing'>{props.selectedWalletCount} Wallets</Tag>
        </div>
        <PreflightChecksEditor checks={props.checks} onChange={props.onChange} disabled={props.creating} idPrefix='worm-preview-checks' />
        <div className='worm-preview-step-actions worm-preview-step-actions--split worm-preview-check-actions'>
            <Button icon={<ArrowLeftOutlined />} disabled={props.creating} onClick={props.onBack}>
                Wallets
            </Button>
            <Button type='primary' icon={<FileSearchOutlined />} loading={props.creating} disabled={props.selectedWalletCount === 0} onClick={props.onCreate}>
                Build read-only preview
            </Button>
        </div>
    </section>
);

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
const planLiveStatus = (plan: WormExecutionPlan) => {
    const status =
        plan.state === 'BUILDING'
            ? `Building execution preview. ${plan.completedStepCount} of ${plan.totalStepCount} steps classified.`
            : plan.state === 'READY'
              ? plan.usabilityCode
                  ? `Execution preview built and not consumable. ${plan.readyStepCount} actionable steps and ${plan.skippedStepCount} skipped.`
                  : `Execution preview ready. ${plan.readyStepCount} actionable steps and ${plan.skippedStepCount} skipped.`
              : plan.state === 'EXPIRED'
                ? 'Execution preview expired.'
                : `Execution preview failed. ${displayCode(plan.failureCode)}.`;
    const advisoryEntries = Object.entries(plan.advisoryCounts).filter(([, count]) => count > 0);
    const advisoryTotal = advisoryEntries.reduce((total, [, count]) => total + count, 0);
    return advisoryTotal > 0 ? `${status} ${advisoryTotal} ignored warnings observed: ${advisoryEntries.map(([code]) => displayCode(code)).join(', ')}.` : status;
};

const PlanChecksSummary = (props: {plan: WormExecutionPlan; canChange: boolean; changing: boolean; disabled: boolean; onChange: () => void; onRerun: () => void}) => {
    const disabledChecks = disabledWormExecutionPreflightChecks(props.plan.preflightChecks);
    const advisoryEntries = Object.entries(props.plan.advisoryCounts)
        .filter(([, count]) => count > 0)
        .sort(([left], [right]) => left.localeCompare(right));
    return (
        <Card
            size='small'
            className='worm-preview-checks-summary'
            title='Guards and fill rule'
            extra={
                props.canChange ? (
                    <Space wrap={true} className='worm-preview-review-check-actions'>
                        <Button size='small' disabled={props.disabled} onClick={props.onChange}>
                            Change checks
                        </Button>
                        <Button size='small' type='primary' icon={<ReloadOutlined />} loading={props.changing} disabled={props.disabled} onClick={props.onRerun}>
                            Re-run checks
                        </Button>
                    </Space>
                ) : undefined
            }>
            <div className='worm-preview-checks-summary__body'>
                <div className='worm-preview-checks-summary__rules' aria-label='Frozen preview checks'>
                    {wormExecutionMandatoryGuardDefinitions.map(definition => (
                        <Tag key={definition.key} color='blue'>
                            Always on · {definition.title}
                        </Tag>
                    ))}
                    {wormExecutionPreflightCheckDefinitions.map(definition => (
                        <Tag key={definition.key} color={props.plan.preflightChecks[definition.key] ? 'green' : 'gold'}>
                            {props.plan.preflightChecks[definition.key] ? 'Checked' : 'Ignored'} · {definition.title}
                        </Tag>
                    ))}
                </div>
                <Typography.Text type='secondary'>
                    {disabledChecks.length === 0
                        ? 'Full estimated liquidity was required for each otherwise actionable 1× order.'
                        : 'Partial-fill estimates were allowed to continue as liquidity advisories; mandatory exposure guards still applied.'}
                </Typography.Text>
                {advisoryEntries.length > 0 && (
                    <div className='worm-preview-advisory-counts' aria-label='Ignored warning counts'>
                        {advisoryEntries.map(([code, count]) => (
                            <span key={code}>
                                <WarningOutlined aria-hidden='true' /> Ignored: {displayCode(code)} <strong>{count}</strong>
                            </span>
                        ))}
                    </div>
                )}
            </div>
        </Card>
    );
};

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
                description='Worm does not expose exact transaction fees or account rent through Estimate. Live execution refreshes the SOL balance and requires a positive balance, but Athena does not inspect Worm’s returned transaction or cryptographically bind its actual chain spend.'
            />
        </>
    );
};

const StepAdvisories = ({codes}: {codes: string[]}) =>
    codes.length > 0 ? (
        <div className='worm-preview-step-advisories' aria-label='Ignored preview warnings'>
            {codes.map(code => (
                <Typography.Text type='warning' key={code}>
                    <WarningOutlined aria-hidden='true' /> Ignored: {displayCode(code)}
                </Typography.Text>
            ))}
        </div>
    ) : null;

const PlanStepCard = (props: {step: WormExecutionPlanStep; wallet?: WormExecutionPlanWallet; item?: WormExecutionPlanItem}) => (
    <Card className='worm-preview-step-card' size='small'>
        <div className='worm-preview-step-card__heading'>
            <span>{props.step.ordinal}</span>
            <Tag color={props.step.disposition === 'READY' ? 'green' : 'default'}>
                {props.step.disposition === 'READY' ? 'Actionable' : displayCode(props.step.reasonCode, 'Skipped')}
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
        <StepAdvisories codes={props.step.advisoryCodes} />
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
    const data = useAsyncData(
        () => services.wormTrading.listExecutionPlanSteps(props.plan.id, props.plan.preflightChecks, page, pageSize),
        [props.plan.id, props.plan.preflightChecks, page, pageSize, props.plan.updatedAt]
    );
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
                    <Tag color={step.disposition === 'READY' ? 'green' : 'default'}>{step.disposition === 'READY' ? 'Actionable' : 'Skipped'}</Tag>
                    <Typography.Text type='secondary'>{step.disposition === 'READY' ? 'Actionable at preview funds' : displayCode(step.reasonCode)}</Typography.Text>
                    <StepAdvisories codes={step.advisoryCodes} />
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
    preparing: boolean;
    canChangeChecks: boolean;
    canRerun: boolean;
    canPrepare: boolean;
    onRetryStatus: () => void;
    onBack: () => void;
    onChangeChecks: () => void;
    onRerun: () => void;
    onPrepare: () => void;
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
            <PlanChecksSummary
                plan={props.plan}
                canChange={props.canChangeChecks}
                changing={props.creating}
                disabled={!props.canRerun || props.plan.state === 'BUILDING' || props.preparing}
                onChange={props.onChangeChecks}
                onRerun={props.onRerun}
            />
            <PlanSummary plan={props.plan} />
            {props.plan.state !== 'BUILDING' && <PlanSteps plan={props.plan} />}
            <Alert
                className='worm-preview-read-only'
                type='warning'
                showIcon={true}
                title='This preview remains read-only'
                description='Preparing a live execution only freezes this reviewed plan. It does not sign in to Worm, ask a Wallet to sign, create a position request, or submit an order.'
            />
            <div className='worm-preview-step-actions worm-preview-step-actions--split'>
                <Button icon={<ArrowLeftOutlined />} onClick={props.onBack}>
                    Saved combinations
                </Button>
                <Space wrap={true}>
                    {props.canChangeChecks && (
                        <Button
                            type='primary'
                            icon={<SafetyCertificateOutlined />}
                            loading={props.preparing}
                            disabled={!props.canPrepare || props.creating}
                            onClick={props.onPrepare}>
                            Prepare live execution
                        </Button>
                    )}
                </Space>
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
    const [workflowStep, setWorkflowStep] = React.useState(planID ? 3 : 0);
    const [selectedIDs, setSelectedIDs] = React.useState<number[]>([]);
    const [preflightChecks, setPreflightChecks] = React.useState<WormExecutionPreflightChecks>(createDefaultWormExecutionPreflightChecks);
    const [rerunOpen, setRerunOpen] = React.useState(false);
    const [rerunChecks, setRerunChecks] = React.useState<WormExecutionPreflightChecks>(createDefaultWormExecutionPreflightChecks);
    const [creating, setCreating] = React.useState(false);
    const [preparing, setPreparing] = React.useState(false);
    const [plan, setPlan] = React.useState<WormExecutionPlan>();
    const [planLoading, setPlanLoading] = React.useState(Boolean(planID));
    const [planError, setPlanError] = React.useState<Error>();
    const [planReload, setPlanReload] = React.useState(0);
    const initializedPlanIDRef = React.useRef('');
    const planIntentRef = React.useRef<ExecutionPlanIntent>();
    const createRequestRef = React.useRef<ReturnType<typeof services.wormTrading.createExecutionPlan>>();
    const prepareRequestRef = React.useRef<ReturnType<typeof services.wormTrading.createExecutionRun>>();
    const accountIDRef = React.useRef(authorization.user.accountId);
    const accessRevisionRef = React.useRef(authorization.revision);
    const canWriteRef = React.useRef(canWrite);
    accountIDRef.current = authorization.user.accountId;
    accessRevisionRef.current = authorization.revision;
    canWriteRef.current = canWrite;
    const combination = useAsyncData(() => services.wormTrading.getMarketCombination(id), [authorization.user.accountId, authorization.revision, id]);
    const inventory = useAsyncData(canWrite ? loadConnectionInventory : emptyConnectionInventory, [authorization.user.accountId, authorization.revision, canWrite]);

    React.useEffect(
        () => () => {
            createRequestRef.current?.abort?.();
            prepareRequestRef.current?.abort?.();
        },
        []
    );

    React.useEffect(() => {
        createRequestRef.current?.abort?.();
        prepareRequestRef.current?.abort?.();
        createRequestRef.current = undefined;
        prepareRequestRef.current = undefined;
        setCreating(false);
        setPreparing(false);
        setWorkflowStep(planID ? 3 : 0);
        setSelectedIDs([]);
        setPreflightChecks(createDefaultWormExecutionPreflightChecks());
        setRerunChecks(createDefaultWormExecutionPreflightChecks());
        setRerunOpen(false);
        setPlan(undefined);
        setPlanError(undefined);
        initializedPlanIDRef.current = '';
        if (planIntentRef.current?.accountId !== authorization.user.accountId || planIntentRef.current?.id !== planID || planIntentRef.current.combinationId !== id) {
            planIntentRef.current = undefined;
        }
        setPlanLoading(Boolean(planID));
    }, [authorization.user.accountId, authorization.revision, id, planID]);

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
                if (planIntentRef.current) {
                    if (!executionPlanMatchesIntent(next, planIntentRef.current)) {
                        continuePolling = false;
                        throw new Error('Worm Trading changed immutable preview inputs. Existing data was not replaced.');
                    }
                } else {
                    planIntentRef.current = executionPlanIntent(next, accountIDRef.current);
                }
                setPlan(next);
                setPlanError(undefined);
                setPlanLoading(false);
                consecutiveFailures = 0;
                if (initializedPlanIDRef.current !== next.id && next.wallets.length > 0) {
                    initializedPlanIDRef.current = next.id;
                    setSelectedIDs(next.wallets.map(wallet => wallet.wallet.walletId));
                    setPreflightChecks(next.preflightChecks);
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
    }, [authorization.user.accountId, authorization.revision, id, planID, planReload]);

    React.useEffect(() => {
        if (!canWrite) {
            createRequestRef.current?.abort?.();
            prepareRequestRef.current?.abort?.();
            createRequestRef.current = undefined;
            prepareRequestRef.current = undefined;
            setCreating(false);
            setPreparing(false);
            setRerunOpen(false);
            setRerunChecks(createDefaultWormExecutionPreflightChecks());
        }
    }, [canWrite]);

    if (!canWrite && !planID) {
        return <Navigate replace={true} to='/worm-trading/combinations' />;
    }

    const createPlan = async (walletIDs = selectedIDs, checks = preflightChecks, replacedPlanID = ''): Promise<'created' | 'conflict' | 'failed'> => {
        const source = combination.data;
        if (!source || creating || createRequestRef.current || walletIDs.length === 0) {
            return 'failed';
        }
        setCreating(true);
        const operationAccountID = accountIDRef.current;
        const operationAccessRevision = accessRevisionRef.current;
        const request = services.wormTrading.createExecutionPlan({
            combinationId: source.id,
            expectedCombinationRevision: source.revision,
            walletIds: walletIDs,
            preflightChecks: checks
        });
        createRequestRef.current = request;
        try {
            const created = await request;
            if (
                !canWriteRef.current ||
                accountIDRef.current !== operationAccountID ||
                accessRevisionRef.current !== operationAccessRevision ||
                createRequestRef.current !== request
            ) {
                return 'failed';
            }
            if (replacedPlanID && created.id === replacedPlanID) {
                throw new Error('Worm Trading did not create a new immutable preview. Existing data was not replaced.');
            }
            setPlan(created);
            setPlanError(undefined);
            initializedPlanIDRef.current = created.id;
            planIntentRef.current = executionPlanIntent(created, operationAccountID);
            setSelectedIDs(walletIDs);
            setPreflightChecks(created.preflightChecks);
            setWorkflowStep(3);
            const next = new URLSearchParams(searchParams);
            next.set('planId', created.id);
            setSearchParams(next, {replace: true});
            return 'created';
        } catch (error) {
            if (
                !canWriteRef.current ||
                accountIDRef.current !== operationAccountID ||
                accessRevisionRef.current !== operationAccessRevision ||
                createRequestRef.current !== request
            ) {
                return 'failed';
            }
            const details = requestErrorDetails(error);
            if (details.status === 409) {
                setWorkflowStep(0);
                combination.reload();
                ctx.notifications.error('Combination changed', 'Review the latest combination revision before building another preview.');
                return 'conflict';
            } else {
                ctx.notifications.error('Could not create preview', requestErrorMessage(error, 'The previous preview remains unchanged.'));
                return 'failed';
            }
        } finally {
            if (createRequestRef.current === request) {
                createRequestRef.current = undefined;
                if (canWriteRef.current && accountIDRef.current === operationAccountID && accessRevisionRef.current === operationAccessRevision) {
                    setCreating(false);
                }
            }
        }
    };

    const openRerun = () => {
        if (!plan || !combination.data || combination.data.revision !== plan.combinationRevision) {
            setWorkflowStep(0);
            combination.reload();
            ctx.notifications.warning('Review the combination', 'The saved combination changed after this preview was created.');
            return;
        }
        setRerunChecks({...plan.preflightChecks});
        setRerunOpen(true);
    };

    const rerunPlan = async () => {
        const current = plan;
        if (!current || !combination.data || combination.data.revision !== current.combinationRevision) {
            setRerunOpen(false);
            setWorkflowStep(0);
            combination.reload();
            ctx.notifications.warning('Review the combination', 'The saved combination changed after this preview was created.');
            return;
        }
        const walletIDs = current.wallets.map(wallet => wallet.wallet.walletId);
        const result = await createPlan(walletIDs.length > 0 ? walletIDs : selectedIDs, rerunChecks, current.id);
        if (result !== 'failed') {
            setRerunOpen(false);
        }
    };

    const prepareLiveExecution = async () => {
        const current = plan;
        if (!current || preparing || current.state !== 'READY' || current.usabilityCode || current.readyStepCount <= 0 || current.expiresAt * 1_000 <= Date.now()) {
            return;
        }
        setPreparing(true);
        const operationAccountID = accountIDRef.current;
        const operationAccessRevision = accessRevisionRef.current;
        const request = services.wormTrading.createExecutionRun(current.id, {
            commandId: window.crypto.randomUUID(),
            expectedRevision: current.combinationRevision
        });
        prepareRequestRef.current = request;
        try {
            const created = await request;
            if (
                !canWriteRef.current ||
                accountIDRef.current !== operationAccountID ||
                accessRevisionRef.current !== operationAccessRevision ||
                prepareRequestRef.current !== request
            ) {
                return;
            }
            if (
                created.planId !== current.id ||
                created.combinationId !== current.combinationId ||
                created.combinationRevision !== current.combinationRevision ||
                created.state !== 'AWAITING_AUTHORIZATION' ||
                !preflightChecksEqual(created.preflightChecks, current.preflightChecks)
            ) {
                throw new Error('Worm Trading returned a live execution for different immutable preview inputs.');
            }
            navigate(`/worm-trading/executions/${encodeURIComponent(created.id)}`);
        } catch (error) {
            if (
                canWriteRef.current &&
                accountIDRef.current === operationAccountID &&
                accessRevisionRef.current === operationAccessRevision &&
                prepareRequestRef.current === request
            ) {
                const details = requestErrorDetails(error);
                if (details.status === 409) {
                    ctx.notifications.error('Could not freeze execution', 'The preview or combination revision changed. Refresh the preview before trying again.');
                    setPlanReload(value => value + 1);
                } else {
                    ctx.notifications.error('Could not prepare live execution', requestErrorMessage(error, 'No execution run was created.'));
                }
            }
        } finally {
            if (prepareRequestRef.current === request) {
                prepareRequestRef.current = undefined;
                if (canWriteRef.current && accountIDRef.current === operationAccountID && accessRevisionRef.current === operationAccessRevision) {
                    setPreparing(false);
                }
            }
        }
    };

    const combinationError = workflowStep < 3 ? combination.error : undefined;
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
                    {title: 'Checks', content: 'Confirm guards'},
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
                    onSelectedIDsChange={setSelectedIDs}
                    onBack={() => setWorkflowStep(0)}
                    onContinue={() => setWorkflowStep(2)}
                    onReload={inventory.reload}
                    onOpenAssets={() => navigate('/worm-trading')}
                />
            )}
            {workflowStep === 2 && (
                <ChecksStep
                    checks={preflightChecks}
                    creating={creating}
                    selectedWalletCount={selectedIDs.length}
                    onChange={setPreflightChecks}
                    onBack={() => setWorkflowStep(1)}
                    onCreate={() => void createPlan(selectedIDs, preflightChecks)}
                />
            )}
            {workflowStep === 3 && (
                <ReviewStep
                    plan={plan}
                    loading={planLoading}
                    error={planError}
                    creating={creating}
                    preparing={preparing}
                    canChangeChecks={canWrite}
                    canRerun={canWrite && Boolean(combination.data && (plan?.wallets.length || selectedIDs.length))}
                    canPrepare={canWrite && plan?.state === 'READY' && !plan.usabilityCode && plan.readyStepCount > 0 && plan.expiresAt * 1_000 > Date.now()}
                    onRetryStatus={() => setPlanReload(value => value + 1)}
                    onBack={() => navigate('/worm-trading/combinations')}
                    onChangeChecks={() => {
                        if (!plan || !combination.data || combination.data.revision !== plan.combinationRevision) {
                            setWorkflowStep(0);
                            combination.reload();
                            ctx.notifications.warning('Review the combination', 'The saved combination changed or could not be confirmed for this preview.');
                            return;
                        }
                        setPreflightChecks({...plan.preflightChecks});
                        if (plan.wallets.length > 0) {
                            setSelectedIDs(plan.wallets.map(wallet => wallet.wallet.walletId));
                        }
                        setWorkflowStep(2);
                    }}
                    onRerun={openRerun}
                    onPrepare={() => void prepareLiveExecution()}
                />
            )}
            <Modal
                className='worm-preview-rerun-modal'
                title='Re-run preview checks'
                width={820}
                open={rerunOpen}
                okText='Re-run checks'
                cancelText='Cancel'
                confirmLoading={creating}
                closable={!creating}
                keyboard={!creating}
                maskClosable={!creating}
                cancelButtonProps={{disabled: creating}}
                okButtonProps={{disabled: !plan || plan.state === 'BUILDING' || preparing}}
                onCancel={() => {
                    if (!creating) {
                        setRerunOpen(false);
                    }
                }}
                onOk={() => void rerunPlan()}>
                <Typography.Paragraph type='secondary'>
                    A new immutable preview will use the same Combination revision and Wallet order. The existing preview remains unchanged.
                </Typography.Paragraph>
                <PreflightChecksEditor checks={rerunChecks} onChange={setRerunChecks} disabled={creating} idPrefix='worm-preview-rerun-checks' />
            </Modal>
        </AppPage>
    );
};
