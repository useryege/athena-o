import {
    ArrowDownOutlined,
    ArrowLeftOutlined,
    ArrowRightOutlined,
    ArrowUpOutlined,
    CloseOutlined,
    FileSearchOutlined,
    ReloadOutlined,
    SafetyCertificateOutlined,
    SearchOutlined
} from '@ant-design/icons';
import {Alert, Avatar, Button, Card, Checkbox, Drawer, Empty, Input, Pagination, Progress, Skeleton, Space, Steps, Tag, Tooltip, Typography} from 'antd';
import * as React from 'react';
import {Navigate, useNavigate, useParams, useSearchParams} from 'react-router-dom';
import {AppPage, ChoiceGroup, useAsyncData} from '../../components';
import {AccountDataModule} from '../../shared/access-modules';
import {Context, useAuthorization} from '../../shared/context';
import {formatBeijingUnixSeconds} from '../../shared/format';
import {memberServices as services} from '../services';
import {
    AbortableWormTradingPromise,
    ListWormTradingWalletConnectionsResult,
    WormExecutionPlan,
    WormExecutionPlanItem,
    WormMarketCombination,
    WormTradingWalletConnectionItem,
    WormTradingWalletSelection,
    WormTradingWalletSummary
} from '../../shared/services/worm-trading-service';
import {MAXIMUM_WORM_TRADING_WALLETS} from '../../shared/services/worm-trading-service';
import {realmBoundResourceURL, requestErrorDetails, requestErrorMessage} from '../../shared/services/requests';
import {wormExecutionMandatoryGuardDefinitions} from './worm-execution-preflight';

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

interface ExecutionPlanIntent {
    accountId: string;
    id: string;
    combinationId: string;
    combinationRevision: number;
    walletSelectionRevision: number;
    walletIds: number[];
}

const executionPlanIntent = (plan: WormExecutionPlan, accountId: string): ExecutionPlanIntent => ({
    accountId,
    id: plan.id,
    combinationId: plan.combinationId,
    combinationRevision: plan.combinationRevision,
    walletSelectionRevision: plan.walletSelectionRevision,
    walletIds: plan.wallets.map(wallet => wallet.wallet.walletId)
});

const executionPlanMatchesIntent = (plan: WormExecutionPlan, intent: ExecutionPlanIntent) => {
    const walletIds = plan.wallets.map(wallet => wallet.wallet.walletId);
    return (
        plan.id === intent.id &&
        plan.combinationId === intent.combinationId &&
        plan.combinationRevision === intent.combinationRevision &&
        plan.walletSelectionRevision === intent.walletSelectionRevision &&
        walletIds.length === intent.walletIds.length &&
        walletIds.every((walletId, index) => walletId === intent.walletIds[index])
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

const ExecutionWalletAvatar = (props: {wallet: WormTradingWalletSummary; size?: number}) => {
    const glyph = walletPresetGlyphs[props.wallet.avatarPresetId];
    const uploaded = props.wallet.avatarKind.toLowerCase() === 'upload' && realmBoundResourceURL(props.wallet.avatarUrl);
    return (
        <Avatar
            aria-hidden='true'
            className={`wallet-avatar ${glyph ? `wallet-avatar--${props.wallet.avatarPresetId}` : 'wallet-avatar--generated'}`}
            size={props.size || 44}
            src={uploaded || undefined}
            style={{background: 'var(--athena-panel-elevated)', color: 'var(--athena-muted)', border: '1px solid var(--athena-border-strong)'}}>
            {glyph || 'S'}
        </Avatar>
    );
};

const WalletIdentity = (props: {wallet: WormTradingWalletSummary; compact?: boolean}) => (
    <div className='worm-preview-wallet-identity'>
        <ExecutionWalletAvatar wallet={props.wallet} size={props.compact ? 36 : 44} />
        <div>
            <Typography.Text strong={true}>{props.wallet.remark || 'Solana wallet'}</Typography.Text>
            <Tooltip title={props.wallet.address}>
                <code>{props.wallet.address}</code>
            </Tooltip>
        </div>
    </div>
);

interface ConfiguredConnectionInventory extends ListWormTradingWalletConnectionsResult {
    configured: boolean;
    selectedWalletCount: number;
    unavailableSelectedWalletCount: number;
    selection: WormTradingWalletSelection;
}

const emptyWalletSelection = (): WormTradingWalletSelection => ({
    configured: false,
    revision: 0,
    selectedItems: [],
    retirements: [],
    updatedAt: 0,
    maximumWallets: MAXIMUM_WORM_TRADING_WALLETS
});

const loadConnectionInventory = (): AbortableWormTradingPromise<ConfiguredConnectionInventory> => {
    let selectionRequest: AbortableWormTradingPromise<WormTradingWalletSelection> | undefined;
    let currentRequest: AbortableWormTradingPromise<ListWormTradingWalletConnectionsResult> | undefined;
    let aborted = false;
    const promise = (async (): Promise<ConfiguredConnectionInventory> => {
        selectionRequest = services.wormTrading.getWalletSelection();
        const selection = await selectionRequest;
        if (aborted) {
            throw new DOMException('The request was aborted.', 'AbortError');
        }
        if (!selection.configured || selection.selectedItems.length === 0) {
            return {
                items: [],
                total: 0,
                page: 1,
                pageSize: connectionPageSize,
                fetchedAt: 0,
                selectionConfigured: selection.configured,
                selectionRevision: selection.revision,
                selectionUpdatedAt: selection.updatedAt,
                maximumWallets: selection.maximumWallets,
                configured: selection.configured,
                selectedWalletCount: selection.selectedItems.length,
                unavailableSelectedWalletCount: 0,
                selection
            };
        }
        const items: WormTradingWalletConnectionItem[] = [];
        const seenWalletIDs = new Set<number>();
        const seenWalletAddresses = new Set<string>();
        let page = 1;
        let expectedTotal: number | undefined;
        let fetchedAt = 0;
        while (!aborted) {
            currentRequest = services.wormTrading.listWalletConnections(page, connectionPageSize);
            const response = await currentRequest;
            if (
                response.selectionConfigured !== selection.configured ||
                response.selectionRevision !== selection.revision ||
                response.selectionUpdatedAt !== selection.updatedAt ||
                response.maximumWallets !== selection.maximumWallets
            ) {
                throw new Error('The saved Wallet selection changed while the connection inventory was loading. Refresh and try again.');
            }
            if (expectedTotal === undefined) {
                expectedTotal = response.total;
                fetchedAt = response.fetchedAt;
            } else if (response.total !== expectedTotal) {
                throw new Error('The Wallet inventory changed while it was loading. Refresh and try again.');
            }
            for (const item of response.items) {
                if (seenWalletIDs.has(item.wallet.walletId) || seenWalletAddresses.has(item.wallet.address)) {
                    throw new Error('Worm Trading returned the same Wallet more than once. Refresh and try again.');
                }
                seenWalletIDs.add(item.wallet.walletId);
                seenWalletAddresses.add(item.wallet.address);
                items.push(item);
            }
            if (items.length >= response.total) {
                if (items.length !== response.total) {
                    throw new Error('Worm Trading returned an invalid Wallet inventory.');
                }
                const inventoryByWalletID = new Map(items.map(item => [item.wallet.walletId, item]));
                const projectedSelectedItems = items.filter(item => item.selected).sort((left, right) => left.selectionOrdinal - right.selectionOrdinal);
                if (
                    projectedSelectedItems.length !== selection.selectedItems.length ||
                    projectedSelectedItems.some((item, index) => {
                        const selected = selection.selectedItems[index];
                        return item.wallet.walletId !== selected.walletId || item.wallet.address !== selected.address || item.selectionOrdinal !== selected.ordinal;
                    })
                ) {
                    throw new Error('Worm Trading returned a Wallet inventory that does not match the saved selection. Refresh and try again.');
                }
                const selectedItems = selection.selectedItems.flatMap(selected => {
                    const item = inventoryByWalletID.get(selected.walletId);
                    return item?.wallet.address === selected.address && item.selected && item.selectionOrdinal === selected.ordinal ? [item] : [];
                });
                const eligibleItems = selectedItems.filter(item => item.connection.state === 'CONNECTED');
                return {
                    items: eligibleItems,
                    total: eligibleItems.length,
                    page: 1,
                    pageSize: connectionPageSize,
                    fetchedAt,
                    selectionConfigured: selection.configured,
                    selectionRevision: selection.revision,
                    selectionUpdatedAt: selection.updatedAt,
                    maximumWallets: selection.maximumWallets,
                    configured: selection.configured,
                    selectedWalletCount: selection.selectedItems.length,
                    unavailableSelectedWalletCount: selection.selectedItems.length - eligibleItems.length,
                    selection
                };
            }
            if (response.items.length === 0) {
                throw new Error('Worm Trading returned an incomplete Wallet inventory.');
            }
            page += 1;
        }
        throw new DOMException('The request was aborted.', 'AbortError');
    })() as AbortableWormTradingPromise<ConfiguredConnectionInventory>;
    promise.abort = () => {
        aborted = true;
        selectionRequest?.abort?.();
        currentRequest?.abort?.();
    };
    return promise;
};

const emptyConnectionInventory = (): AbortableWormTradingPromise<ConfiguredConnectionInventory> => {
    const selection = emptyWalletSelection();
    const promise = Promise.resolve({
        items: [],
        total: 0,
        page: 1,
        pageSize: connectionPageSize,
        fetchedAt: 0,
        selectionConfigured: false,
        selectionRevision: 0,
        selectionUpdatedAt: 0,
        maximumWallets: selection.maximumWallets,
        configured: false,
        selectedWalletCount: 0,
        unavailableSelectedWalletCount: 0,
        selection
    }) as AbortableWormTradingPromise<ConfiguredConnectionInventory>;
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
                    <Tag className='worm-outcome'>{item.side}</Tag>
                </div>
            ))}
        </div>
        <div className='worm-preview-step-actions'>
            <Button type='primary' icon={<ArrowRightOutlined aria-hidden='true' />} onClick={props.onContinue}>
                Choose Wallets
            </Button>
        </div>
    </section>
);

const WalletChoiceCard = (props: {item: WormTradingWalletConnectionItem; selected: boolean; limitReached: boolean; onChange: (selected: boolean) => void}) => {
    const connected = props.item.connection.state === 'CONNECTED';
    const disabled = !connected || (!props.selected && props.limitReached);
    return (
        <label className={`worm-preview-wallet-choice${props.selected ? ' worm-preview-wallet-choice--selected' : ''}${disabled ? ' worm-preview-wallet-choice--disabled' : ''}`}>
            <Checkbox
                aria-label={`${props.item.wallet.remark || 'Solana wallet'} ${props.item.wallet.address}`}
                checked={props.selected}
                disabled={disabled}
                onChange={event => props.onChange(event.target.checked)}
            />
            <WalletIdentity wallet={props.item.wallet} />
            <Tag className={`worm-status worm-status--${connected ? 'success' : 'default'}`}>{connectionLabel(props.item.connection.state)}</Tag>
            {!connected && <small>Connect this Wallet on Assets before creating a preview.</small>}
            {connected && !props.selected && props.limitReached && <small>Deselect another Wallet before adding this one.</small>}
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
            <Tag className='worm-status worm-status--info'>{props.items.length}</Tag>
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
                                    icon={<ArrowUpOutlined aria-hidden='true' />}
                                    aria-label={`Move ${item.wallet.remark || 'Wallet'} earlier`}
                                    disabled={index === 0}
                                    onClick={() => props.onMove(index, -1)}
                                />
                            </Tooltip>
                            <Tooltip title='Move later'>
                                <Button
                                    type='text'
                                    size='small'
                                    icon={<ArrowDownOutlined aria-hidden='true' />}
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
                                    icon={<CloseOutlined aria-hidden='true' />}
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
    configured: boolean;
    selectedWalletCount: number;
    unavailableSelectedWalletCount: number;
    maximumWallets: number;
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
    const limitReached = props.selectedIDs.length >= props.maximumWallets;
    const normalizedQuery = query.trim().toLowerCase();
    const visibleItems = props.inventory.filter(item =>
        [item.wallet.remark, item.wallet.address, String(item.wallet.walletId)].some(value => value.toLowerCase().includes(normalizedQuery))
    );
    const toggleWallet = (walletID: number, selected: boolean) => {
        if (selected) {
            if (!selectedSet.has(walletID) && props.selectedIDs.length < props.maximumWallets) {
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
                                Selected and connected Wallets
                            </Typography.Title>
                            <Typography.Text type='secondary'>Only Wallets saved on Assets are eligible. Selection order becomes the execution order.</Typography.Text>
                        </div>
                        <Space wrap={true}>
                            <Button
                                size='small'
                                disabled={connectedItems.length === 0 || connectedItems.every(item => selectedSet.has(item.wallet.walletId))}
                                onClick={() =>
                                    props.onSelectedIDsChange([
                                        ...props.selectedIDs,
                                        ...connectedItems
                                            .filter(item => !selectedSet.has(item.wallet.walletId))
                                            .slice(0, Math.max(0, props.maximumWallets - props.selectedIDs.length))
                                            .map(item => item.wallet.walletId)
                                    ])
                                }>
                                Select all eligible
                            </Button>
                            <Button size='small' disabled={props.selectedIDs.length === 0} onClick={() => props.onSelectedIDsChange([])}>
                                Clear
                            </Button>
                        </Space>
                    </div>
                    <Input
                        aria-label='Search Wallet name or address'
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
                                <Button size='small' icon={<ReloadOutlined aria-hidden='true' />} onClick={props.onReload}>
                                    Retry
                                </Button>
                            }
                        />
                    )}
                    {!props.loading && !props.error && props.selectedWalletCount === 0 && (
                        <Alert
                            type='warning'
                            showIcon={true}
                            title={props.configured ? 'No Worm Trading Wallets selected' : 'Worm Trading Wallets are not configured'}
                            description='Choose and save at least one Wallet on Assets before creating an execution preview.'
                            action={
                                <Button size='small' onClick={props.onOpenAssets}>
                                    Manage on Assets
                                </Button>
                            }
                        />
                    )}
                    {!props.loading && !props.error && props.selectedWalletCount > 0 && connectedItems.length === 0 && (
                        <Alert
                            type='warning'
                            showIcon={true}
                            title='Selected Wallets need attention'
                            description='None of the saved Wallets is currently connected. Review connection status on Assets.'
                            action={
                                <Button size='small' onClick={props.onOpenAssets}>
                                    Review on Assets
                                </Button>
                            }
                        />
                    )}
                    {!props.loading && !props.error && connectedItems.length > 0 && props.unavailableSelectedWalletCount > 0 && (
                        <Alert
                            type='info'
                            showIcon={true}
                            title={`${props.unavailableSelectedWalletCount} selected ${props.unavailableSelectedWalletCount === 1 ? 'Wallet is' : 'Wallets are'} unavailable`}
                            description='Only selected Wallets with a confirmed connection are shown below.'
                            action={
                                <Button size='small' onClick={props.onOpenAssets}>
                                    Review on Assets
                                </Button>
                            }
                        />
                    )}
                    <div className='worm-preview-wallet-grid' aria-busy={props.loading || undefined}>
                        {props.loading ? (
                            Array.from({length: 4}, (_, index) => <Card key={index} loading={true} />)
                        ) : props.error ? null : visibleItems.length === 0 ? (
                            <Empty
                                image={Empty.PRESENTED_IMAGE_SIMPLE}
                                description={props.inventory.length === 0 ? 'No selected and connected Wallets are eligible.' : 'No Wallets match this search.'}
                            />
                        ) : (
                            visibleItems.map(item => (
                                <WalletChoiceCard
                                    key={item.wallet.walletId}
                                    item={item}
                                    selected={selectedSet.has(item.wallet.walletId)}
                                    limitReached={limitReached}
                                    onChange={selected => toggleWallet(item.wallet.walletId, selected)}
                                />
                            ))
                        )}
                    </div>
                    <Typography.Text className='worm-preview-live-status' role='status' aria-live='polite'>
                        {limitReached
                            ? `Maximum ${props.maximumWallets} Wallets selected. Deselect one before adding another.`
                            : `${props.selectedIDs.length} of ${props.maximumWallets} Wallets selected.`}
                    </Typography.Text>
                </section>
                <aside className='worm-preview-wallet-order-desktop'>{selectedSummary}</aside>
            </div>
            <div className='worm-preview-step-actions worm-preview-step-actions--split worm-preview-wallet-actions'>
                <Button icon={<ArrowLeftOutlined aria-hidden='true' />} onClick={props.onBack}>
                    Combination
                </Button>
                <Button
                    type='primary'
                    icon={<ArrowRightOutlined aria-hidden='true' />}
                    disabled={props.selectedIDs.length === 0 || props.selectedIDs.length > props.maximumWallets || props.loading || Boolean(props.error)}
                    onClick={props.onContinue}>
                    Continue to checks
                </Button>
            </div>
            <div className='worm-preview-mobile-order'>
                <Button type='text' icon={<ArrowLeftOutlined aria-hidden='true' />} aria-label='Back to combination' onClick={props.onBack} />
                <span>
                    <strong>{props.selectedIDs.length}</strong>
                    <small>Wallets selected</small>
                </span>
                <Button onClick={() => setOrderOpen(true)}>Review order</Button>
                <Button
                    type='primary'
                    disabled={props.selectedIDs.length === 0 || props.selectedIDs.length > props.maximumWallets || props.loading || Boolean(props.error)}
                    onClick={props.onContinue}>
                    Checks
                </Button>
            </div>
            <Drawer rootClassName='worm-preview-order-drawer' title='Wallet execution order' width={440} open={orderOpen} onClose={() => setOrderOpen(false)}>
                {selectedSummary}
            </Drawer>
        </>
    );
};

const MandatoryGuards = (props: {idPrefix: string}) => (
    <div className='worm-preview-checks-editor'>
        <div className='worm-preview-checks-editor__heading'>
            <div>
                <Typography.Text id={`${props.idPrefix}-mandatory-heading`} strong={true}>
                    Mandatory execution guards
                </Typography.Text>
                <Typography.Text type='secondary'>Both guards are always enforced again immediately before Athena sends a Worm Open request.</Typography.Text>
            </div>
            <Tag className='worm-status worm-status--info'>Always on</Tag>
        </div>
        <section className='worm-preview-guard-group' aria-labelledby={`${props.idPrefix}-mandatory-heading`}>
            <div className='worm-preview-guard-list'>
                {wormExecutionMandatoryGuardDefinitions.map(definition => (
                    <div className='worm-preview-check-card worm-preview-check-card--mandatory' key={definition.key}>
                        <SafetyCertificateOutlined aria-hidden='true' />
                        <span>
                            <Typography.Text strong={true}>{definition.title}</Typography.Text>
                            <Typography.Text type='secondary'>{definition.description}</Typography.Text>
                        </span>
                        <Tag className='worm-status worm-status--info'>Always on</Tag>
                    </div>
                ))}
            </div>
        </section>
        <Alert
            type='info'
            showIcon={true}
            title='Market order and 1× leverage are fixed'
            description='The selected target market, side, funds, and 1× leverage come from the saved combination and server policy. Wallet authority, market and Estimate validity, balances, permissions, and mutation protections remain mandatory.'
        />
    </div>
);

const ChecksStep = (props: {creating: boolean; selectedWalletCount: number; onBack: () => void; onCreate: () => void}) => (
    <section className='worm-preview-checks' aria-labelledby='worm-preview-checks-heading'>
        <div className='worm-preview-section-heading'>
            <div>
                <Typography.Title id='worm-preview-checks-heading' level={2}>
                    Preview checks
                </Typography.Title>
                <Typography.Text type='secondary'>Review the two always-on exposure guards for the fixed market-order, 1× execution flow.</Typography.Text>
            </div>
            <Tag className='worm-status worm-status--info'>{props.selectedWalletCount} Wallets</Tag>
        </div>
        <MandatoryGuards idPrefix='worm-preview-checks' />
        <div className='worm-preview-step-actions worm-preview-step-actions--split worm-preview-check-actions'>
            <Button icon={<ArrowLeftOutlined aria-hidden='true' />} disabled={props.creating} onClick={props.onBack}>
                Wallets
            </Button>
            <Button type='primary' icon={<FileSearchOutlined aria-hidden='true' />} loading={props.creating} disabled={props.selectedWalletCount === 0} onClick={props.onCreate}>
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
            type='info'
            className='worm-preview-ready-state'
            showIcon={true}
            title='Read-only preview ready'
            description={`The snapshot expires at ${formatBeijingUnixSeconds(props.plan.expiresAt) || 'the server-defined time'}. This read-only preview cannot start an order.`}
        />
    );
};

const planProgress = (plan: WormExecutionPlan) => (plan.totalStepCount > 0 ? Math.min(100, Math.round((plan.completedStepCount / plan.totalStepCount) * 100)) : 0);
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

const PlanSummary = (props: {plan: WormExecutionPlan; canRerun: boolean; changing: boolean; onRerun: () => void}) => {
    const plan = props.plan;
    const totalsAvailable = plan.state === 'READY' || plan.state === 'EXPIRED';
    return (
        <div className='worm-preview-summary'>
            <div className='worm-preview-plan-header'>
                <div>
                    <Typography.Title level={2}>{plan.combinationName}</Typography.Title>
                    <Typography.Text type='secondary'>
                        Combination revision {plan.combinationRevision} · Wallet selection revision {plan.walletSelectionRevision}
                    </Typography.Text>
                </div>
                {props.canRerun && (
                    <Button icon={<ReloadOutlined aria-hidden='true' />} loading={props.changing} disabled={plan.state === 'BUILDING'} onClick={props.onRerun}>
                        Re-run preview
                    </Button>
                )}
            </div>
            {plan.state === 'BUILDING' && (
                <div className='worm-preview-build-progress'>
                    <Progress percent={planProgress(plan)} status='active' />
                    <Typography.Text type='secondary'>
                        {plan.completedStepCount} of {plan.totalStepCount} steps classified
                    </Typography.Text>
                </div>
            )}
            <dl className='worm-preview-metrics'>
                <div>
                    <dt>Wallets / markets</dt>
                    <dd>
                        {plan.walletCount} / {plan.itemCount}
                    </dd>
                </div>
                <div>
                    <dt>Steps</dt>
                    <dd>{totalsAvailable ? `${plan.readyStepCount} ready / ${plan.skippedStepCount} skipped` : `${plan.totalStepCount} awaiting classification`}</dd>
                </div>
                <div>
                    <dt>Collateral + estimated fee</dt>
                    <dd>{totalsAvailable ? `${plan.maximumCollateral || '—'} + ${plan.openingFeeEstimate || '—'} USDC` : '—'}</dd>
                </div>
                <div>
                    <dt>Total USDC needed</dt>
                    <dd>{totalsAvailable ? displayUSDC(plan.totalUSDCNeeded) : '—'}</dd>
                </div>
            </dl>
            <p className='worm-execution-note'>Market orders · 1× leverage. Collateral, opening fee, and USDC totals include actionable steps only.</p>
            <p className='worm-execution-note'>Always on: target-market position guard and wallet-wide in-flight request guard.</p>
            <p className='worm-execution-note'>
                SOL is informational here. Exact transaction fees and rent are unavailable from Estimate. Live execution requires positive SOL; Athena does not inspect the returned
                transaction or bind its actual chain spend.
            </p>
            <details className='worm-step-evidence'>
                <summary>Mandatory guards &amp; snapshot identity</summary>
                {wormExecutionMandatoryGuardDefinitions.map(definition => (
                    <p key={definition.key}>
                        <strong>{definition.title} · Always on.</strong> {definition.description}
                    </p>
                ))}
                <dl>
                    <div>
                        <dt>Plan ID</dt>
                        <dd className='athena-identifier'>{plan.id}</dd>
                    </div>
                    <div>
                        <dt>Combination ID</dt>
                        <dd className='athena-identifier'>{plan.combinationId}</dd>
                    </div>
                </dl>
                <p>
                    Requested {formatBeijingUnixSeconds(plan.requestedAt)} · Updated {formatBeijingUnixSeconds(plan.updatedAt)} UTC+8
                </p>
                <div aria-label='Skipped step reasons'>
                    {Object.entries(plan.reasonCounts)
                        .filter(([reason, count]) => reason !== 'READY' && count > 0)
                        .map(([reason, count]) => (
                            <p key={reason}>
                                {displayCode(reason)} · {count}
                            </p>
                        ))}
                </div>
            </details>
        </div>
    );
};

const PlanSteps = (props: {plan: WormExecutionPlan}) => {
    const [page, setPage] = React.useState(1);
    const [pageSize, setPageSize] = React.useState(50);
    React.useEffect(() => {
        setPage(1);
        setPageSize(50);
    }, [props.plan.id]);
    const data = useAsyncData(() => services.wormTrading.listExecutionPlanSteps(props.plan.id, page, pageSize), [props.plan.id, page, pageSize, props.plan.updatedAt]);
    return (
        <section className='worm-preview-steps' aria-label='Wallet-major step preview'>
            {data.error && (
                <Alert
                    type='error'
                    showIcon
                    title={data.data ? 'Preview steps are stale' : 'Could not load preview steps'}
                    description={requestErrorMessage(data.error)}
                    action={<Button onClick={data.reload}>Retry</Button>}
                />
            )}
            {!data.data ? (
                data.loading ? (
                    <Skeleton active />
                ) : null
            ) : (
                <>
                    {data.data.items.length === 0 ? (
                        <Empty description='No classified steps are available.' />
                    ) : (
                        props.plan.wallets
                            .filter(wallet => data.data!.items.some(step => step.walletOrdinal === wallet.ordinal))
                            .map(wallet => {
                                const steps = data.data!.items.filter(step => step.walletOrdinal === wallet.ordinal);
                                return (
                                    <section key={wallet.ordinal} className='worm-preview-wallet-section' aria-label={`Steps for ${wallet.wallet.remark || wallet.wallet.address}`}>
                                        <div className='worm-preview-wallet-section-heading'>
                                            <div>
                                                <Typography.Title level={3}>{wallet.wallet.remark || 'Solana wallet'}</Typography.Title>
                                                <Tag className={`worm-status worm-status--${wallet.connection?.state === 'CONNECTED' ? 'success' : 'warning'}`}>
                                                    {wallet.connection ? connectionLabel(wallet.connection.state) : 'Not observed'}
                                                </Tag>
                                            </div>
                                            <span className='worm-execution-note'>
                                                USDC {wallet.usdc.amount || '—'} · SOL {wallet.sol.amount || '—'}
                                            </span>
                                        </div>
                                        <code className='athena-identifier'>{wallet.wallet.address}</code>
                                        <ol className='worm-preview-grouped-steps'>
                                            {steps.map(step => {
                                                const item = props.plan.items.find(candidate => candidate.ordinal === step.itemOrdinal);
                                                return (
                                                    <li key={step.ordinal}>
                                                        <span className='worm-execution-note'>{step.ordinal}</span>
                                                        <div>
                                                            <Typography.Text strong>{item?.marketTitle || `Market ${step.itemOrdinal}`}</Typography.Text>
                                                            <p className='worm-execution-note'>
                                                                {item?.side || 'Unavailable'} · {item?.backend || 'Unavailable'} · {item?.leverage || '—'}×
                                                            </p>
                                                        </div>
                                                        <div>
                                                            <Tag className={`worm-status worm-status--${step.disposition === 'READY' ? 'success' : 'warning'}`}>
                                                                {step.disposition === 'READY' ? 'Ready' : 'Skipped'}
                                                            </Tag>
                                                            {step.reasonCode && <p className='worm-execution-note'>{displayCode(step.reasonCode)}</p>}
                                                        </div>
                                                        <div className='worm-preview-step-funds athena-numeric'>
                                                            {step.disposition === 'READY' ? displayUSDC(item?.estimate?.userFundsNeeded || '') : 'No funds deducted'}
                                                            <p className='worm-execution-note'>{item ? marketEstimateSummary(item) : 'Estimate unavailable'}</p>
                                                        </div>
                                                    </li>
                                                );
                                            })}
                                        </ol>
                                        <details className='worm-step-evidence'>
                                            <summary>Wallet projection &amp; reasons</summary>
                                            <p className='worm-execution-note'>
                                                Current page · {steps.filter(step => step.disposition === 'READY').length} ready /{' '}
                                                {steps.filter(step => step.disposition === 'SKIPPED').length} skipped
                                            </p>
                                            {steps.map(step => {
                                                const item = props.plan.items.find(candidate => candidate.ordinal === step.itemOrdinal);
                                                return (
                                                    <div key={step.ordinal}>
                                                        <p>
                                                            Step {step.ordinal} · Projected USDC {step.projectedUsdcBefore || '—'} → {step.projectedUsdcAfter || '—'}
                                                        </p>
                                                        <p className='athena-identifier'>
                                                            Market {item?.marketConditionId || 'Unavailable'} · Event {item?.eventConditionId || 'Unavailable'}
                                                        </p>
                                                        {item?.estimate && (
                                                            <p>
                                                                Estimate fully filled: {item.estimate.isFullyFilled ? 'Yes' : 'No'} · Shares {item.estimate.totalShares} · Average
                                                                price {item.estimate.averagePrice}
                                                            </p>
                                                        )}
                                                    </div>
                                                );
                                            })}
                                        </details>
                                    </section>
                                );
                            })
                    )}
                    <div className='worm-preview-pagination'>
                        <span className='worm-execution-note'>Wallet-major order · {data.data.total} steps</span>
                        <Pagination size='small' current={page} pageSize={pageSize} total={data.data.total} showSizeChanger={false} onChange={setPage} />
                        <ChoiceGroup<number>
                            ariaLabel='Steps per page'
                            size='small'
                            value={pageSize}
                            options={stepPageSizes.map(value => ({label: String(value), value}))}
                            onChange={value => {
                                setPage(1);
                                setPageSize(value);
                            }}
                        />
                    </div>
                </>
            )}
        </section>
    );
};

const ReviewStep = (props: {
    plan?: WormExecutionPlan;
    loading: boolean;
    error?: Error;
    creating: boolean;
    preparing: boolean;
    canWrite: boolean;
    canRerun: boolean;
    canPrepare: boolean;
    walletSelectionChanged: boolean;
    onRetryStatus: () => void;
    onBack: () => void;
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
                    description='The last confirmed preview remains visible and is stale. Refresh before preparing a Run.'
                    action={
                        <Button size='small' onClick={props.onRetryStatus}>
                            Retry
                        </Button>
                    }
                />
            )}
            {props.walletSelectionChanged && (
                <Alert
                    type='warning'
                    showIcon={true}
                    title='The saved Wallet selection changed after this preview was created'
                    description='This immutable preview remains available for review, but it cannot be prepared as a live execution. Re-run it against the current saved Wallet selection.'
                />
            )}
            <div className='worm-preview-live-status' role='status' aria-live='polite'>
                {planLiveStatus(props.plan)}
            </div>
            <PlanStateAlert plan={props.plan} />
            <div className='worm-preview-plan-panel'>
                <PlanSummary plan={props.plan} canRerun={props.canRerun} changing={props.creating || props.preparing} onRerun={props.onRerun} />
                {(props.plan.state === 'READY' || props.plan.state === 'EXPIRED') && <PlanSteps plan={props.plan} />}
            </div>
            <section className='worm-preview-prepare-panel worm-preview-read-only' aria-labelledby='worm-prepare-heading'>
                <Typography.Title id='worm-prepare-heading' level={2}>
                    Prepare live execution
                </Typography.Title>
                <p className='worm-execution-note'>
                    Create a Run to freeze this reviewed plan. You will authorize separately on the execution detail page. Creating a Run does not sign in to Worm, request a wallet
                    signature, create a position request or submit an order.
                </p>
                <div className='worm-preview-step-actions worm-preview-step-actions--split'>
                    <Button icon={<ArrowLeftOutlined aria-hidden='true' />} onClick={props.onBack}>
                        Saved combinations
                    </Button>
                    <Space wrap={true}>
                        {props.canWrite && (
                            <Button
                                type='primary'
                                icon={<SafetyCertificateOutlined aria-hidden='true' />}
                                loading={props.preparing}
                                disabled={!props.canPrepare || props.creating}
                                onClick={props.onPrepare}>
                                Prepare live execution
                            </Button>
                        )}
                    </Space>
                </div>
            </section>
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
            createRequestRef.current = undefined;
            prepareRequestRef.current = undefined;
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
        }
    }, [canWrite]);

    if (!canWrite && !planID) {
        return <Navigate replace={true} to='/worm-trading/combinations' />;
    }

    const createPlan = async (walletIDs = selectedIDs, replacedPlanID = ''): Promise<'created' | 'conflict' | 'failed'> => {
        const source = combination.data;
        const walletSelectionRevision = inventory.data?.selection.revision || 0;
        if (!source || creating || createRequestRef.current || walletIDs.length === 0 || walletSelectionRevision < 1) {
            return 'failed';
        }
        if (walletIDs.length > MAXIMUM_WORM_TRADING_WALLETS) {
            ctx.notifications.error('Too many Wallets selected', `Execution Preview supports at most ${MAXIMUM_WORM_TRADING_WALLETS} Wallets.`);
            setWorkflowStep(1);
            return 'failed';
        }
        setCreating(true);
        const operationAccountID = accountIDRef.current;
        const operationAccessRevision = accessRevisionRef.current;
        const request = services.wormTrading.createExecutionPlan({
            combinationId: source.id,
            expectedCombinationRevision: source.revision,
            expectedWalletSelectionRevision: walletSelectionRevision,
            walletIds: walletIDs
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
            if (details.status === 409 && (details.message === 'WALLET_SELECTION_CHANGED' || details.reason === 'WALLET_SELECTION_CHANGED')) {
                setWorkflowStep(1);
                inventory.reload();
                ctx.notifications.error('Worm wallet selection changed', 'Review the latest selected and connected Wallets before building another preview.');
                return 'conflict';
            } else if (details.status === 409 && details.code === 9) {
                setWorkflowStep(1);
                inventory.reload();
                ctx.notifications.error('Wallets are no longer eligible', 'Review the latest selected and connected Wallets before building another preview.');
                return 'conflict';
            } else if (details.status === 409 && details.code === 10) {
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

    const rerunPlan = async () => {
        const current = plan;
        if (!current || !combination.data || combination.data.revision !== current.combinationRevision) {
            setWorkflowStep(0);
            combination.reload();
            ctx.notifications.warning('Review the combination', 'The saved combination changed after this preview was created.');
            return;
        }
        const walletIDs = current.wallets.map(wallet => wallet.wallet.walletId);
        await createPlan(walletIDs.length > 0 ? walletIDs : selectedIDs, current.id);
    };

    const prepareLiveExecution = async () => {
        const current = plan;
        if (
            !current ||
            !canWriteRef.current ||
            preparing ||
            prepareRequestRef.current ||
            planError ||
            combination.error ||
            inventory.error ||
            combination.data?.revision !== current.combinationRevision ||
            inventory.data?.selection.revision !== current.walletSelectionRevision ||
            current.state !== 'READY' ||
            current.usabilityCode ||
            current.readyStepCount <= 0 ||
            current.expiresAt * 1_000 <= Date.now()
        ) {
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
                created.state !== 'AWAITING_AUTHORIZATION'
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
        <div className='worm-execution-theme'>
            <Button className='worm-execution-back' type='text' icon={<ArrowLeftOutlined aria-hidden='true' />} onClick={() => navigate('/worm-trading/combinations')}>
                Saved combinations
            </Button>
            <AppPage
                title='Worm Trading Execution Preview'
                subtitle={
                    canWrite
                        ? 'Select configured and connected Wallets and build a read-only, wallet-major preview. No Worm order, draft, signature, or transaction is created.'
                        : 'Review this read-only, wallet-major preview. No Worm order, draft, signature, or transaction is created.'
                }
                loading={workflowStep === 0 && combination.loading}
                error={combinationError}
                onRefresh={workflowStep === 0 ? combination.reload : undefined}>
                <Steps
                    className='worm-preview-workflow'
                    current={workflowStep}
                    responsive={true}
                    items={[{title: 'Combination'}, {title: 'Wallets'}, {title: 'Checks'}, {title: 'Review'}]}
                />
                {workflowStep === 0 && combination.data && <CombinationStep combination={combination.data} onContinue={() => setWorkflowStep(1)} />}
                {workflowStep === 1 && (
                    <WalletsStep
                        inventory={inventory.data?.items || []}
                        configured={inventory.data?.configured || false}
                        selectedWalletCount={inventory.data?.selectedWalletCount || 0}
                        unavailableSelectedWalletCount={inventory.data?.unavailableSelectedWalletCount || 0}
                        maximumWallets={inventory.data?.maximumWallets || MAXIMUM_WORM_TRADING_WALLETS}
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
                    <ChecksStep creating={creating} selectedWalletCount={selectedIDs.length} onBack={() => setWorkflowStep(1)} onCreate={() => void createPlan(selectedIDs)} />
                )}
                {workflowStep === 3 && (
                    <ReviewStep
                        plan={plan}
                        loading={planLoading}
                        error={planError}
                        creating={creating}
                        preparing={preparing}
                        canWrite={canWrite}
                        canRerun={canWrite && Boolean(combination.data && (plan?.wallets.length || selectedIDs.length))}
                        canPrepare={
                            canWrite &&
                            plan?.state === 'READY' &&
                            !planError &&
                            !combination.error &&
                            !inventory.error &&
                            combination.data?.revision === plan.combinationRevision &&
                            !plan.usabilityCode &&
                            plan.readyStepCount > 0 &&
                            plan.expiresAt * 1_000 > Date.now() &&
                            inventory.data?.selection.revision === plan.walletSelectionRevision
                        }
                        walletSelectionChanged={Boolean(
                            canWrite && plan && inventory.data?.selection.configured && inventory.data.selection.revision !== plan.walletSelectionRevision
                        )}
                        onRetryStatus={() => setPlanReload(value => value + 1)}
                        onBack={() => navigate('/worm-trading/combinations')}
                        onRerun={() => void rerunPlan()}
                        onPrepare={() => void prepareLiveExecution()}
                    />
                )}
            </AppPage>
        </div>
    );
};
