import {
    ApiOutlined,
    CloseCircleOutlined,
    CopyOutlined,
    DisconnectOutlined,
    LinkOutlined,
    PauseCircleOutlined,
    PlayCircleOutlined,
    ReloadOutlined,
    SafetyCertificateOutlined,
    SearchOutlined,
    SettingOutlined,
    StopOutlined,
    SyncOutlined,
    WalletOutlined
} from '@ant-design/icons';
import {Alert, Avatar, Button, Card, Checkbox, Empty, Input, Modal, Pagination, Progress, Result, Segmented, Skeleton, Space, Tabs, Tag, Tooltip, Typography} from 'antd';
import * as React from 'react';
import {useLocation, useNavigate} from 'react-router-dom';
import {AppPage, useCachedAsyncData} from '../../components';
import {AccountDataModule} from '../../shared/access-modules';
import {Context, useAuthorization} from '../../shared/context';
import {formatBeijingUnixSeconds} from '../../shared/format';
import {AccountIdentityProvider} from '../../shared/models';
import {SensitiveWriteScope, useSensitiveWriteLease} from '../../shared/sensitive-write-scope';
import {memberServices as services} from '../services';
import type {
    WormActivityStreamState,
    WormInFlightRequest,
    WormOpenPosition,
    WormPositionCashOutAllowedAction,
    WormPositionCashOutBatch,
    WormPositionCashOutBatchAllowedAction,
    WormPositionCashOutBatchBalanceEvidence,
    WormPositionCashOutBatchItem,
    WormPositionCashOutBatchItemState,
    WormPositionCashOutOperation,
    WormPositionCashOutProjection,
    WormTradingAssetBalance,
    WormTradingStatus,
    WormTradingTokenAssetBalance,
    WormTradingWalletActivityItem,
    WormTradingWalletBalanceItem,
    WormTradingWalletConnectionItem,
    WormTradingWalletSelection,
    WormTradingWalletSummary,
    WormWalletConnection,
    WormWalletConnectionState
} from '../../shared/services/worm-trading-service';
import {
    MAXIMUM_WORM_TRADING_WALLETS,
    WORM_TRADING_LOGIN_SESSION_REQUIRED,
    WORM_TRADING_REAUTH_REQUIRED,
    WORM_TRADING_REAUTH_UNAVAILABLE
} from '../../shared/services/worm-trading-service';
import {realmBoundResourceURL, requestErrorDetails, requestErrorMessage} from '../../shared/services/requests';
import {usePagedParams} from '../../shared/pages/shared';

const wormTradingPageSize = 20;
const wormTradingPageSizes = [wormTradingPageSize];
const wormConnectionInventoryPageSize = 100;
const wormConnectionStartIntervalMS = 12_000;
const pendingConnectionActionKey = 'athena.member.worm-trading.pending-connection-action';
const pendingPositionCashOutKey = 'athena.member.worm-trading.pending-position-cash-out';
const pendingPositionCashOutBatchKey = 'athena.member.worm-trading.pending-position-cash-out-batch';
const positionCashOutReasonQuery = 'wormPositionCashOutReason';
const positionCashOutBatchReasonQuery = 'wormPositionCashOutBatchReason';
const positionCashOutPollIntervalMS = 2_000;
const positionCashOutBatchItemPageSize = 20;
const maximumSelectedPositionCashOutBatchWallets = MAXIMUM_WORM_TRADING_WALLETS;
const maximumRememberedPositionCashOuts = 100;
const canonicalPositionCashOutIDPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
const connectOutcomeUnknownWarning = 'CONNECT_OUTCOME_UNKNOWN';

const rememberedPositionCashOutBatchID = (): string => {
    const id = window.sessionStorage.getItem(pendingPositionCashOutBatchKey) || '';
    return canonicalPositionCashOutIDPattern.test(id) ? id : '';
};

const rememberPositionCashOutBatchID = (id: string) => {
    if (!canonicalPositionCashOutIDPattern.test(id)) {
        throw new Error('Athena returned an invalid Cash Out batch ID.');
    }
    window.sessionStorage.setItem(pendingPositionCashOutBatchKey, id);
};

const forgetPositionCashOutBatchID = (id?: string) => {
    if (!id || rememberedPositionCashOutBatchID() === id) {
        window.sessionStorage.removeItem(pendingPositionCashOutBatchKey);
    }
};

const rememberedPositionCashOutIDs = (): string[] => {
    try {
        const value = JSON.parse(window.sessionStorage.getItem(pendingPositionCashOutKey) || '[]');
        if (!Array.isArray(value)) {
            return [];
        }
        return Array.from(new Set(value.filter(item => typeof item === 'string' && canonicalPositionCashOutIDPattern.test(item)))).slice(-maximumRememberedPositionCashOuts);
    } catch {
        return [];
    }
};

const rememberPositionCashOutID = (id: string) => {
    if (!canonicalPositionCashOutIDPattern.test(id)) {
        throw new Error('Athena returned an invalid Cash Out operation ID.');
    }
    const ids = rememberedPositionCashOutIDs().filter(item => item !== id);
    ids.push(id);
    window.sessionStorage.setItem(pendingPositionCashOutKey, JSON.stringify(ids.slice(-maximumRememberedPositionCashOuts)));
};

const forgetPositionCashOutID = (id: string) => {
    const ids = rememberedPositionCashOutIDs().filter(item => item !== id);
    if (ids.length === 0) {
        window.sessionStorage.removeItem(pendingPositionCashOutKey);
        return;
    }
    window.sessionStorage.setItem(pendingPositionCashOutKey, JSON.stringify(ids));
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

const formatIntegerString = (value?: string) => {
    if (!value) {
        return '-';
    }
    try {
        return new Intl.NumberFormat().format(BigInt(value));
    } catch {
        return value;
    }
};

const displayIdentity = (value?: string) => value || '-';

const titleCase = (value?: string) =>
    (value || '')
        .toLowerCase()
        .split(/[_-]+/)
        .filter(Boolean)
        .map(part => `${part.slice(0, 1).toUpperCase()}${part.slice(1)}`)
        .join(' ');

const optionalValue = (value: string, suffix = '') => (value ? `${value}${suffix}` : '-');
const liquidationPriceValue = (position: WormOpenPosition) => position.liquidationPrice || (Number(position.leverage) === 1 ? 'No liquidation (1×)' : '-');
const assetAvailable = (asset?: WormTradingAssetBalance) => asset?.availability === 'AVAILABLE' || asset?.availability === 'BALANCE_AVAILABILITY_AVAILABLE';
const streamAvailable = (stream: WormActivityStreamState) => stream.availability === 'AVAILABLE';
const connectionWasQueried = (state: WormWalletConnectionState) => state === 'CONNECTED' || state === 'RECONNECT_REQUIRED';

const WormTradingWalletAvatar = (props: {wallet: WormTradingWalletSummary; size?: number}) => {
    const presetGlyph = walletPresetGlyphs[props.wallet.avatarPresetId];
    const uploaded = props.wallet.avatarKind.toLowerCase() === 'upload' && realmBoundResourceURL(props.wallet.avatarUrl);
    const className = ['wallet-avatar', presetGlyph ? `wallet-avatar--${props.wallet.avatarPresetId}` : 'wallet-avatar--generated'].join(' ');
    const style = {background: 'var(--athena-panel-elevated)', color: 'var(--athena-muted)', border: '1px solid var(--athena-border-strong)'};
    return (
        <Avatar aria-hidden='true' className={className} size={props.size || 46} src={uploaded || undefined} style={style}>
            {presetGlyph || 'S'}
        </Avatar>
    );
};

const BalanceStatusTag = (props: {status: WormTradingWalletBalanceItem['status']}) => {
    const color = props.status === 'COMPLETE' ? 'success' : props.status === 'PARTIAL' ? 'warning' : 'error';
    return <Tag color={color}>{titleCase(props.status) || 'Unavailable'}</Tag>;
};

const ActivityStatusTag = (props: {status: WormTradingWalletActivityItem['status']}) => {
    const color = props.status === 'COMPLETE' ? 'success' : props.status === 'PARTIAL' ? 'warning' : 'error';
    return props.status === 'COMPLETE' ? (
        <span className='worm-activity-complete'>Activity complete</span>
    ) : (
        <Tag color={color}>Activity {titleCase(props.status) || 'Unavailable'}</Tag>
    );
};

const ConnectionStatusTag = (props: {state: WormWalletConnectionState}) => {
    const color =
        props.state === 'CONNECTED'
            ? 'success'
            : props.state === 'NOT_CONNECTED'
              ? 'default'
              : props.state === 'CONNECTING' || props.state === 'DISCONNECTING'
                ? 'processing'
                : 'warning';
    return <Tag color={color}>{titleCase(props.state)}</Tag>;
};

const AssetValue = (props: {asset: WormTradingAssetBalance; symbol: 'SOL' | 'USDC'; token?: WormTradingTokenAssetBalance}) => {
    const available = assetAvailable(props.asset);
    const detail = available
        ? [
              props.token ? `${props.token.tokenAccountCount} token ${props.token.tokenAccountCount === 1 ? 'account' : 'accounts'}` : undefined,
              `slot ${formatIntegerString(props.asset.observedSlot)}`
          ]
              .filter(Boolean)
              .join(' · ')
        : titleCase(props.asset.errorCode || props.asset.availability) || 'Balance unavailable';
    const exact = available ? (props.asset.atomicAmount ? `${props.asset.atomicAmount} atomic units · ${props.asset.decimals} decimals` : 'Atomic amount unavailable') : detail;
    return (
        <div className={available ? 'worm-trading-asset' : 'worm-trading-asset worm-trading-asset--unavailable'}>
            <Tooltip title={exact}>
                <strong className='athena-number'>{available ? optionalValue(props.asset.amount) : 'Unavailable'}</strong>
            </Tooltip>
            <small>
                {available
                    ? props.token
                        ? `Native USDC · ${props.token.tokenAccountCount} token ${props.token.tokenAccountCount === 1 ? 'account' : 'accounts'}`
                        : 'Confirmed'
                    : detail}
            </small>
            <details className='worm-balance-evidence'>
                <summary>Balance evidence</summary>
                <small>
                    {detail} · {exact}
                </small>
            </details>
        </div>
    );
};

const WalletIdentity = (props: {wallet: WormTradingWalletSummary; onCopy: () => void; compact?: boolean}) => (
    <div className={props.compact ? 'worm-trading-wallet worm-trading-wallet--compact' : 'worm-trading-wallet'}>
        <WormTradingWalletAvatar wallet={props.wallet} size={props.compact ? 42 : 46} />
        <div className='worm-trading-wallet__main'>
            <Typography.Text strong={true}>{props.wallet.remark || 'Solana wallet'}</Typography.Text>
            <span className='worm-trading-wallet__address'>
                <Tooltip title={props.wallet.address}>
                    <code>{displayIdentity(props.wallet.address)}</code>
                </Tooltip>
                <Tooltip title='Copy address'>
                    <Button type='text' size='small' aria-label={`Copy ${props.wallet.remark || 'Solana wallet'} address`} icon={<CopyOutlined />} onClick={props.onCopy} />
                </Tooltip>
            </span>
        </div>
    </div>
);

const RuntimeSummary = (props: {status?: WormTradingStatus; loading: boolean; error?: Error; fallbackNetwork?: string; fallbackCommitment?: string}) => {
    if (props.loading && !props.status) {
        return (
            <section className='worm-trading-runtime' aria-label='Loading Worm Trading runtime status'>
                <Skeleton active={true} paragraph={{rows: 3}} />
            </section>
        );
    }

    const status = props.status;
    const runtimeState = status?.status.toLowerCase() || '';
    const ready = Boolean(status?.started && runtimeState === 'running' && status.rpcReachable && status.batchSupported && status.genesisVerified && status.usdcVerified);
    const state = !status ? 'Status unavailable' : !status.started ? 'Stopped' : runtimeState === 'configuration_error' ? 'Configuration error' : ready ? 'Ready' : 'Degraded';
    const stateColor = ready ? 'success' : !status?.started || runtimeState === 'configuration_error' ? 'error' : 'warning';
    const wormState = titleCase(status?.wormAPIStatus) || (status?.credentialStoreReady ? 'Not observed' : 'Unavailable');
    const wormRuntimeState = (status?.wormAPIStatus || '').toLowerCase();
    const wormColor = wormRuntimeState === 'running' ? 'success' : wormRuntimeState === 'configuration_error' || !status?.credentialStoreReady ? 'error' : 'warning';
    return (
        <section className='worm-trading-runtime' aria-labelledby='worm-trading-runtime-heading'>
            <div className='worm-trading-runtime__services'>
                <div className='worm-trading-runtime__heading'>
                    <span>
                        <LinkOutlined aria-hidden='true' />
                        <Typography.Text id='worm-trading-runtime-heading' strong={true}>
                            Solana connection
                        </Typography.Text>
                    </span>
                    <Tag color={stateColor}>{state}</Tag>
                </div>
                <div className='worm-trading-runtime__heading'>
                    <span>
                        <ApiOutlined aria-hidden='true' />
                        <Typography.Text strong={true}>Worm position access</Typography.Text>
                    </span>
                    <Tag color={wormColor}>{wormState}</Tag>
                </div>
            </div>
            <details>
                <summary>Connection details</summary>
                <dl className='worm-trading-runtime__facts'>
                    <div>
                        <dt>Network</dt>
                        <dd>{titleCase(status?.network || props.fallbackNetwork) || '-'}</dd>
                    </div>
                    <div>
                        <dt>Commitment</dt>
                        <dd>{titleCase(status?.commitment || props.fallbackCommitment) || '-'}</dd>
                    </div>
                    <div>
                        <dt>Confirmed slot</dt>
                        <dd>{formatIntegerString(status?.latestConfirmedSlot)}</dd>
                    </div>
                    <div>
                        <dt>RPC latency</dt>
                        <dd>{status ? `${status.latencyMS} ms` : '-'}</dd>
                    </div>
                    <div>
                        <dt>Credential store</dt>
                        <dd>{status ? (status.credentialStoreReady ? 'Ready' : 'Unavailable') : '-'}</dd>
                    </div>
                </dl>
            </details>
            {(props.error || status?.lastErrorCategory || status?.wormAPILastErrorCategory) && (
                <Typography.Text className='worm-trading-runtime__error'>
                    {props.error
                        ? requestErrorMessage(props.error, 'Runtime status is unavailable.')
                        : `Latest issue: ${titleCase(status?.lastErrorCategory || status?.wormAPILastErrorCategory)}`}
                </Typography.Text>
            )}
        </section>
    );
};

type ManagedConnectionAction = 'reconnect' | 'regenerate' | 'cleanup';
type PendingConnectionIntent = {kind: 'reconcile-selection'} | {kind: ManagedConnectionAction; walletId: number};

const readPendingConnectionIntent = (): PendingConnectionIntent | undefined => {
    const raw = window.sessionStorage.getItem(pendingConnectionActionKey);
    if (!raw) {
        return undefined;
    }
    window.sessionStorage.removeItem(pendingConnectionActionKey);
    try {
        const value = JSON.parse(raw) as Partial<PendingConnectionIntent>;
        if (value.kind === 'reconcile-selection') {
            return {kind: 'reconcile-selection'};
        }
        const walletId = Number(value.walletId);
        return (value.kind === 'reconnect' || value.kind === 'regenerate' || value.kind === 'cleanup') && Number.isSafeInteger(walletId) && walletId > 0
            ? {kind: value.kind, walletId}
            : undefined;
    } catch {
        return undefined;
    }
};

interface PhantomPublicKey {
    toString(): string;
}

interface PhantomProvider {
    isPhantom?: boolean;
    publicKey?: PhantomPublicKey | null;
    connect(): Promise<{publicKey: PhantomPublicKey}>;
    signMessage(message: Uint8Array, display?: 'utf8'): Promise<{signature: Uint8Array}>;
    on?(event: 'accountChanged', listener: (publicKey: PhantomPublicKey | null) => void): void;
    off?(event: 'accountChanged', listener: (publicKey: PhantomPublicKey | null) => void): void;
    removeListener?(event: 'accountChanged', listener: (publicKey: PhantomPublicKey | null) => void): void;
}

const phantomProvider = (): PhantomProvider | undefined => {
    const provider = (window as Window & {phantom?: {solana?: PhantomProvider}}).phantom?.solana;
    return provider?.isPhantom ? provider : undefined;
};

const rawBase64URL = (bytes: Uint8Array) =>
    window
        .btoa(String.fromCharCode(...bytes))
        .replace(/\+/g, '-')
        .replace(/\//g, '_')
        .replace(/=+$/, '');

const wormConnectionErrorMessage = (error: unknown, fallback: string) => {
    const reason = requestErrorDetails(error).reason || '';
    if (reason === WORM_TRADING_LOGIN_SESSION_REQUIRED) {
        return 'This operation requires an interactive Athena login. API Keys can read connected wallets but cannot manage Worm credentials.';
    }
    if (reason === WORM_TRADING_REAUTH_UNAVAILABLE) {
        return 'Worm credential reauthentication is temporarily unavailable. No credential was created or revoked.';
    }
    if (reason === WORM_TRADING_REAUTH_REQUIRED) {
        return fallback;
    }
    if (reason === 'CONNECT_OUTCOME_UNKNOWN') {
        return 'Worm did not confirm whether it created the credential. Athena will not retry automatically; review the connection state before trying again.';
    }
    return requestErrorMessage(error, fallback);
};

type ConnectionSetupPhase = 'hidden' | 'discovering' | 'authorization-required' | 'connecting' | 'partial' | 'paused' | 'blocked';

interface ConnectionSetupState {
    phase: ConnectionSetupPhase;
    total: number;
    processed: number;
    succeeded: number;
    failed: number;
    remaining: number;
    message: string;
    currentWallet?: WormTradingWalletSummary;
    retryable?: boolean;
}

const hiddenConnectionSetupState: ConnectionSetupState = {
    phase: 'hidden',
    total: 0,
    processed: 0,
    succeeded: 0,
    failed: 0,
    remaining: 0,
    message: ''
};

const ConnectionSetupPanel = (props: {manager: ConnectionManager}) => {
    const setup = props.manager.setup;
    if (setup.phase === 'hidden') {
        return null;
    }
    const title =
        setup.phase === 'discovering'
            ? 'Checking Worm wallet access'
            : setup.phase === 'authorization-required'
              ? 'Authorize Worm wallet connections'
              : setup.phase === 'connecting'
                ? 'Applying Worm wallet selection'
                : setup.phase === 'partial'
                  ? 'Some Worm wallets are not connected'
                  : setup.phase === 'blocked'
                    ? 'Worm connection requires review'
                    : 'Automatic Worm connection is paused';
    const type = setup.phase === 'blocked' ? 'error' : setup.phase === 'partial' || setup.phase === 'paused' || setup.phase === 'authorization-required' ? 'warning' : 'info';
    const progressStatus: 'active' | 'exception' = setup.phase === 'blocked' ? 'exception' : 'active';
    const percent = setup.total > 0 ? Math.min(100, Math.round((setup.processed / setup.total) * 100)) : 0;
    const action =
        setup.phase === 'authorization-required' ? (
            <Button type='primary' icon={<LinkOutlined />} disabled={props.manager.operationBusy} onClick={props.manager.authorize}>
                {setup.processed > 0 ? 'Authorize and continue' : 'Authorize and apply'}
            </Button>
        ) : (setup.phase === 'partial' || setup.phase === 'paused') && setup.retryable ? (
            <Button type='primary' icon={<SyncOutlined />} disabled={props.manager.operationBusy} onClick={props.manager.retry}>
                Retry failed connections
            </Button>
        ) : undefined;
    return (
        <Alert
            className={`worm-trading-connection-setup worm-trading-connection-setup--${setup.phase}`}
            type={type}
            showIcon={true}
            title={title}
            description={
                <div className='worm-trading-connection-setup__body'>
                    <Typography.Paragraph>{setup.message}</Typography.Paragraph>
                    {setup.phase === 'authorization-required' && (
                        <Typography.Paragraph>
                            One identity confirmation permits Worm credential management for five minutes. It does not authorize a trade or extend automatically.
                        </Typography.Paragraph>
                    )}
                    {setup.total > 0 && (
                        <div className='worm-trading-connection-setup__progress'>
                            <Progress aria-label='Wallet connection setup progress' percent={percent} status={progressStatus} showInfo={false} />
                            <span>
                                {setup.processed}/{setup.total} processed · {setup.succeeded} applied · {setup.failed} failed · {setup.remaining} remaining
                            </span>
                        </div>
                    )}
                    {setup.currentWallet && (
                        <Typography.Text type='secondary'>Current wallet: {setup.currentWallet.remark || displayIdentity(setup.currentWallet.address)}</Typography.Text>
                    )}
                    <span className='worm-trading-connection-setup__live' role='status' aria-live='polite' aria-atomic='true'>
                        {setup.message}
                    </span>
                </div>
            }
            action={action}
        />
    );
};

type WalletSelectionFilter = 'all' | 'selected' | 'attention';

const sameWalletSelection = (left: number[], right: number[]) => {
    if (left.length !== right.length) {
        return false;
    }
    const rightSet = new Set(right);
    return left.every(walletID => rightSet.has(walletID));
};

const WalletSelectionModal = (props: {
    open: boolean;
    inventory: WormTradingWalletConnectionItem[];
    selection: WormTradingWalletSelection;
    saving: boolean;
    error?: Error;
    onCancel: () => void;
    onSave: (walletIDs: number[]) => void;
}) => {
    const [query, setQuery] = React.useState('');
    const [filter, setFilter] = React.useState<WalletSelectionFilter>('all');
    const [draftWalletIDs, setDraftWalletIDs] = React.useState<number[]>([]);
    React.useEffect(() => {
        if (!props.open) {
            return;
        }
        setQuery('');
        setFilter('all');
        setDraftWalletIDs(props.selection.selectedItems.map(item => item.walletId));
    }, [props.open, props.selection]);

    const authoritativeWalletIDs = props.selection.selectedItems.map(item => item.walletId);
    const authoritativeSet = new Set(authoritativeWalletIDs);
    const draftSet = new Set(draftWalletIDs);
    const inventoryWalletIDs = new Set(props.inventory.map(item => item.wallet.walletId));
    const normalizedDraftWalletIDs = [
        ...props.inventory.filter(item => draftSet.has(item.wallet.walletId)).map(item => item.wallet.walletId),
        ...draftWalletIDs.filter(walletID => !inventoryWalletIDs.has(walletID))
    ];
    const retirementSet = new Set(props.selection.retirements.map(item => item.walletId));
    const inventoryByWalletID = new Map(props.inventory.map(item => [item.wallet.walletId, item]));
    const attentionSet = new Set(props.inventory.filter(item => item.needsAttention).map(item => item.wallet.walletId));
    props.selection.retirements.forEach(item => attentionSet.add(item.walletId));
    const normalizedQuery = query.trim().toLowerCase();
    const visibleItems = props.inventory.filter(item => {
        const walletID = item.wallet.walletId;
        if (filter === 'selected' && !draftSet.has(walletID)) {
            return false;
        }
        if (filter === 'attention' && !attentionSet.has(walletID)) {
            return false;
        }
        return [item.wallet.remark, item.wallet.address, String(walletID)].some(value => value.toLowerCase().includes(normalizedQuery));
    });
    const additions = draftWalletIDs.filter(walletID => !authoritativeSet.has(walletID)).length;
    const removals = authoritativeWalletIDs.filter(walletID => !draftSet.has(walletID)).length;
    const selectionChanged = !props.selection.configured || !sameWalletSelection(authoritativeWalletIDs, normalizedDraftWalletIDs);
    const atLimit = draftWalletIDs.length >= props.selection.maximumWallets;
    const blockingLegacyItems = props.selection.configured
        ? []
        : props.inventory.filter(item => !draftSet.has(item.wallet.walletId) && !item.retirementPending && !item.removalAllowed);
    const toggleWallet = (walletID: number, selected: boolean) => {
        const inventoryItem = inventoryByWalletID.get(walletID);
        if (selected) {
            if (draftSet.has(walletID) || atLimit) {
                return;
            }
            setDraftWalletIDs(current => [...current, walletID]);
            return;
        }
        if (props.selection.configured && authoritativeSet.has(walletID) && inventoryItem && !inventoryItem.removalAllowed) {
            return;
        }
        setDraftWalletIDs(current => current.filter(item => item !== walletID));
    };
    const saveLabel = `Save and apply${additions > 0 || removals > 0 ? ` (+${additions} / −${removals})` : ''}`;
    return (
        <Modal
            className='worm-wallet-selection-modal'
            centered={true}
            destroyOnHidden={true}
            maskClosable={!props.saving}
            closable={!props.saving}
            open={props.open}
            width={960}
            title={
                <div className='worm-wallet-selection-modal__title'>
                    <span>Manage Worm Trading wallets</span>
                    <Tag color={atLimit ? 'warning' : 'processing'}>
                        {draftWalletIDs.length}/{props.selection.maximumWallets}
                    </Tag>
                </div>
            }
            onCancel={props.onCancel}
            footer={
                <div className='worm-wallet-selection-modal__footer'>
                    <Typography.Text type={atLimit ? 'warning' : 'secondary'} role='status' aria-live='polite'>
                        {blockingLegacyItems.length > 0
                            ? `${blockingLegacyItems.length} existing ${blockingLegacyItems.length === 1 ? 'wallet must be selected or have its blocker' : 'wallets must be selected or have their blockers'} resolved.`
                            : atLimit
                              ? `Maximum ${props.selection.maximumWallets} wallets selected. Deselect one to choose another.`
                              : `${props.selection.maximumWallets - draftWalletIDs.length} slots available.`}
                    </Typography.Text>
                    <div>
                        <Button disabled={props.saving} onClick={props.onCancel}>
                            Cancel
                        </Button>
                        <Button
                            type='primary'
                            loading={props.saving}
                            disabled={Boolean(props.error) || !selectionChanged || draftWalletIDs.length > props.selection.maximumWallets || blockingLegacyItems.length > 0}
                            onClick={() => props.onSave(normalizedDraftWalletIDs)}>
                            {saveLabel}
                        </Button>
                    </div>
                </div>
            }>
            <div className='worm-wallet-selection-modal__body'>
                <Alert
                    type='info'
                    showIcon={true}
                    title='This selection controls Worm credentials and future execution previews.'
                    description='Saving does not place or cancel orders, close positions, sign a transaction, or move funds. Athena disconnects removed wallets before connecting additions.'
                />
                {props.error && (
                    <Alert
                        type='error'
                        showIcon={true}
                        title='The complete Solana wallet inventory is unavailable'
                        description='Saving is disabled because Athena cannot safely evaluate every existing Worm wallet. Close this dialog and retry the inventory refresh.'
                    />
                )}
                {removals > 0 && (
                    <Alert
                        type='warning'
                        showIcon={true}
                        title={`${removals} ${removals === 1 ? 'wallet' : 'wallets'} will be removed`}
                        description='Credential cleanup can remain visible under Needs attention if Worm does not confirm revocation immediately.'
                    />
                )}
                {blockingLegacyItems.length > 0 && (
                    <Alert
                        type='error'
                        showIcon={true}
                        title={`${blockingLegacyItems.length} existing ${blockingLegacyItems.length === 1 ? 'wallet cannot' : 'wallets cannot'} be removed yet`}
                        description='Select each blocked wallet or resolve the reason shown on its card before saving the first Worm Trading configuration.'
                    />
                )}
                <div className='worm-wallet-selection-modal__tools'>
                    <Input
                        allowClear={true}
                        prefix={<SearchOutlined />}
                        placeholder='Search wallet name, address, or ID'
                        value={query}
                        onChange={event => setQuery(event.target.value)}
                    />
                    <Segmented<WalletSelectionFilter>
                        value={filter}
                        options={[
                            {value: 'all', label: `All (${props.inventory.length})`},
                            {value: 'selected', label: `Selected (${draftWalletIDs.length})`},
                            {value: 'attention', label: `Needs attention (${attentionSet.size})`}
                        ]}
                        onChange={setFilter}
                    />
                </div>
                <div className='worm-wallet-selection-modal__grid' aria-busy={props.saving || undefined}>
                    {visibleItems.length === 0 ? (
                        <Empty
                            image={Empty.PRESENTED_IMAGE_SIMPLE}
                            description={
                                filter === 'attention' ? 'No wallets need attention.' : normalizedQuery ? 'No wallets match this search.' : 'No Solana wallets are available.'
                            }
                        />
                    ) : (
                        visibleItems.map(item => {
                            const walletID = item.wallet.walletId;
                            const selected = draftSet.has(walletID);
                            const willAdd = selected && !authoritativeSet.has(walletID);
                            const willRemove = !selected && authoritativeSet.has(walletID);
                            const retiring = retirementSet.has(walletID) && !selected;
                            const removalLocked = props.selection.configured && authoritativeSet.has(walletID) && !item.removalAllowed;
                            const legacyRemovalBlocked = !props.selection.configured && !selected && !item.retirementPending && !item.removalAllowed;
                            const disabled = props.saving || Boolean(props.error) || removalLocked || (!selected && atLimit);
                            return (
                                <label
                                    className={`worm-wallet-selection-card${selected ? ' worm-wallet-selection-card--selected' : ''}${disabled ? ' worm-wallet-selection-card--disabled' : ''}`}
                                    key={walletID}>
                                    <Checkbox
                                        aria-label={`Select ${item.wallet.remark || 'Solana wallet'} ${item.wallet.address}`}
                                        checked={selected}
                                        disabled={disabled}
                                        onChange={event => toggleWallet(walletID, event.target.checked)}
                                    />
                                    <div className='worm-wallet-selection-card__identity'>
                                        <WormTradingWalletAvatar wallet={item.wallet} size={42} />
                                        <span>
                                            <Typography.Text strong={true} ellipsis={{tooltip: item.wallet.remark || 'Solana wallet'}}>
                                                {item.wallet.remark || 'Solana wallet'}
                                            </Typography.Text>
                                            <Tooltip title={item.wallet.address}>
                                                <code>{displayIdentity(item.wallet.address)}</code>
                                            </Tooltip>
                                        </span>
                                    </div>
                                    <div className='worm-wallet-selection-card__status'>
                                        {willAdd ? (
                                            <Tag color='processing'>Will add</Tag>
                                        ) : willRemove ? (
                                            <Tag color='warning'>Will remove</Tag>
                                        ) : retiring ? (
                                            <Tag color='warning'>Retiring</Tag>
                                        ) : null}
                                        {item.pendingState && <Tag color='processing'>{titleCase(item.pendingState)}</Tag>}
                                        <ConnectionStatusTag state={item.connection.state} />
                                    </div>
                                    {(removalLocked || legacyRemovalBlocked) && (
                                        <small>
                                            {removalLocked ? 'This selected wallet cannot be removed' : 'Select this existing wallet or resolve its removal blocker'}:{' '}
                                            {titleCase(item.removalReasonCode) || 'Worm Trading state requires attention'}.
                                        </small>
                                    )}
                                    {!removalLocked && !legacyRemovalBlocked && item.connection.warningCode === connectOutcomeUnknownWarning && (
                                        <small>Connection outcome is unknown and requires review.</small>
                                    )}
                                </label>
                            );
                        })
                    )}
                </div>
                {props.selection.retirements.some(retirement => !props.inventory.some(item => item.wallet.walletId === retirement.walletId)) && (
                    <Alert
                        type='warning'
                        showIcon={true}
                        title='Some retired wallets are no longer in the current inventory'
                        description='Their durable cleanup state remains recorded. Refresh after the wallet inventory is available again.'
                    />
                )}
            </div>
        </Modal>
    );
};

const ConnectionManagement = (props: {onReload: () => void; children: (manage: ConnectionManager) => React.ReactNode}) => {
    const ctx = React.useContext(Context);
    const authorization = useAuthorization();
    const location = useLocation();
    const lease = useSensitiveWriteLease();
    const [busyWalletId, setBusyWalletId] = React.useState<number>();
    const [stage, setStage] = React.useState('');
    const [operationBusy, setOperationBusy] = React.useState(false);
    const [setup, setSetup] = React.useState<ConnectionSetupState>({...hiddenConnectionSetupState, phase: 'discovering', message: 'Loading your Worm wallet selection.'});
    const [connectionItems, setConnectionItems] = React.useState<WormTradingWalletConnectionItem[]>([]);
    const [selection, setSelection] = React.useState<WormTradingWalletSelection>();
    const [selectionError, setSelectionError] = React.useState<Error>();
    const [selectionModalOpen, setSelectionModalOpen] = React.useState(false);
    const [walletProgress, setWalletProgress] = React.useState<Map<number, string>>(() => new Map());
    const resumedRef = React.useRef(false);
    const operationEpochRef = React.useRef(0);
    const runningRef = React.useRef(false);
    const cancelDelayRef = React.useRef<(() => void) | undefined>();
    const lastAutoStartAtRef = React.useRef(0);
    const targetWalletIDsRef = React.useRef(new Set<number>());
    const retirementWalletIDsRef = React.useRef(new Set<number>());
    const attemptedWalletIDsRef = React.useRef(new Set<number>());
    const succeededWalletIDsRef = React.useRef(new Set<number>());
    const failedWalletIDsRef = React.useRef(new Set<number>());
    const blockedWalletIDsRef = React.useRef(new Set<number>());
    const inventorySelectionRef = React.useRef<{configured: boolean; revision: number; updatedAt: number; maximumWallets: number}>();
    const onReloadRef = React.useRef(props.onReload);
    onReloadRef.current = props.onReload;

    const isCurrent = React.useCallback((epoch: number) => operationEpochRef.current === epoch, []);

    const progressState = React.useCallback(
        (phase: ConnectionSetupPhase, message: string, options?: {currentWallet?: WormTradingWalletSummary; retryable?: boolean}): ConnectionSetupState => {
            const total = targetWalletIDsRef.current.size + retirementWalletIDsRef.current.size;
            const processed = attemptedWalletIDsRef.current.size;
            return {
                phase,
                total,
                processed,
                succeeded: succeededWalletIDsRef.current.size,
                failed: failedWalletIDsRef.current.size,
                remaining: Math.max(0, total - processed),
                message,
                currentWallet: options?.currentWallet,
                retryable: options?.retryable
            };
        },
        []
    );

    const setOperationStage = React.useCallback((message: string) => {
        setStage(message);
        setSetup(current => ({...current, message}));
    }, []);

    const beginOperation = React.useCallback(() => {
        if (runningRef.current) {
            return undefined;
        }
        runningRef.current = true;
        setOperationBusy(true);
        return ++operationEpochRef.current;
    }, []);

    const finishOperation = React.useCallback(
        (epoch: number) => {
            if (!isCurrent(epoch)) {
                return;
            }
            runningRef.current = false;
            setOperationBusy(false);
            setBusyWalletId(undefined);
            setStage('');
        },
        [isCurrent]
    );

    React.useEffect(
        () => () => {
            operationEpochRef.current++;
            runningRef.current = false;
            cancelDelayRef.current?.();
        },
        []
    );

    const publishInventory = React.useCallback((items: WormTradingWalletConnectionItem[]) => {
        setConnectionItems(items);
        setWalletProgress(current => {
            const next = new Map(current);
            for (const item of items) {
                if (item.connection.warningCode === connectOutcomeUnknownWarning) {
                    next.set(item.wallet.walletId, 'Connection outcome is unknown; automatic retry is blocked.');
                } else if (item.connection.state !== 'NOT_CONNECTED') {
                    next.delete(item.wallet.walletId);
                }
            }
            return next;
        });
    }, []);

    const fetchWalletSelection = React.useCallback(
        async (epoch: number): Promise<WormTradingWalletSelection | undefined> => {
            const result = await lease.runTask(() => services.wormTrading.getWalletSelection());
            if (!isCurrent(epoch) || result.status === 'discarded') {
                return undefined;
            }
            if (result.status === 'rejected') {
                throw result.error;
            }
            setSelection(result.value);
            setSelectionError(undefined);
            return result.value;
        },
        [isCurrent, lease]
    );

    const fetchConnectionInventory = React.useCallback(
        async (epoch: number): Promise<WormTradingWalletConnectionItem[] | undefined> => {
            const items: WormTradingWalletConnectionItem[] = [];
            const seenWalletIDs = new Set<number>();
            let page = 1;
            let expectedTotal: number | undefined;
            let expectedSelection: {configured: boolean; revision: number; updatedAt: number; maximumWallets: number} | undefined;
            while (true) {
                const result = await lease.runTask(() => services.wormTrading.listWalletConnections(page, wormConnectionInventoryPageSize));
                if (!isCurrent(epoch) || result.status === 'discarded') {
                    return undefined;
                }
                if (result.status === 'rejected') {
                    throw result.error;
                }
                if (expectedTotal === undefined) {
                    expectedTotal = result.value.total;
                    expectedSelection = {
                        configured: result.value.selectionConfigured,
                        revision: result.value.selectionRevision,
                        updatedAt: result.value.selectionUpdatedAt,
                        maximumWallets: result.value.maximumWallets
                    };
                } else if (result.value.total !== expectedTotal) {
                    throw new Error('The Solana wallet inventory changed while Worm connections were being checked. Refresh before continuing.');
                }
                if (
                    !expectedSelection ||
                    result.value.selectionConfigured !== expectedSelection.configured ||
                    result.value.selectionRevision !== expectedSelection.revision ||
                    result.value.selectionUpdatedAt !== expectedSelection.updatedAt ||
                    result.value.maximumWallets !== expectedSelection.maximumWallets
                ) {
                    throw new Error('The saved Worm wallet selection changed while the connection inventory was loading. Refresh before continuing.');
                }
                for (const item of result.value.items) {
                    if (seenWalletIDs.has(item.wallet.walletId)) {
                        throw new Error('Worm Trading returned the same wallet more than once. Existing connection data was not replaced.');
                    }
                    seenWalletIDs.add(item.wallet.walletId);
                    items.push(item);
                }
                if (items.length >= expectedTotal) {
                    if (items.length !== expectedTotal) {
                        throw new Error('Worm Trading returned an inconsistent wallet total. Existing connection data was not replaced.');
                    }
                    inventorySelectionRef.current = expectedSelection;
                    return items;
                }
                if (result.value.items.length === 0) {
                    throw new Error('Worm Trading returned an incomplete wallet inventory. Existing connection data was not replaced.');
                }
                page++;
            }
        },
        [isCurrent, lease]
    );

    const establishSolanaLease = React.useCallback(async () => {
        const provider = phantomProvider();
        if (!provider) {
            ctx.notifications.error('Phantom is required', 'Install or enable Phantom in this browser to approve Worm credential management.');
            return false;
        }
        const expectedAddress = authorization.user.identity.solanaAddress;
        let connectedAddress = '';
        let accountChanged = false;
        const onAccountChanged = (publicKey: PhantomPublicKey | null) => {
            if (connectedAddress && (!publicKey || publicKey.toString() !== connectedAddress)) {
                accountChanged = true;
            }
        };
        const removeListener = () => {
            try {
                if (provider.off) {
                    provider.off('accountChanged', onAccountChanged);
                } else {
                    provider.removeListener?.('accountChanged', onAccountChanged);
                }
            } catch {
                // Listener cleanup cannot make an already completed signature trustworthy again.
            }
        };

        try {
            setOperationStage('Connecting to Phantom…');
            const connection = provider.publicKey ? {publicKey: provider.publicKey} : await provider.connect();
            connectedAddress = connection.publicKey?.toString() || '';
            if (!connectedAddress || connectedAddress !== expectedAddress) {
                throw new Error('Connect the same Phantom account that you use to sign in to Athena.');
            }
            provider.on?.('accountChanged', onAccountChanged);

            setOperationStage('Preparing a Worm credential approval message…');
            const challenge = await lease.runTask(() => services.wormTrading.createSolanaCredentialChallenge());
            if (challenge.status === 'discarded') {
                return false;
            }
            if (challenge.status === 'rejected') {
                throw challenge.error;
            }
            if (!challenge.value.message) {
                throw new Error('Athena returned an empty reauthentication message.');
            }

            setOperationStage('Approve the message in Phantom. No transaction or network fee is involved…');
            const signed = await provider.signMessage(new TextEncoder().encode(challenge.value.message), 'utf8');
            if (accountChanged || provider.publicKey?.toString() !== expectedAddress) {
                throw new Error('The connected Phantom account changed before verification completed.');
            }

            setOperationStage('Verifying the signature…');
            const verified = await lease.runTask(() => services.wormTrading.verifySolanaCredentialSignature(rawBase64URL(signed.signature)));
            if (verified.status === 'discarded') {
                return false;
            }
            if (verified.status === 'rejected') {
                throw verified.error;
            }
            return true;
        } catch (error) {
            ctx.notifications.error('Could not confirm your identity', wormConnectionErrorMessage(error, error instanceof Error ? error.message : 'Phantom verification failed.'));
            return false;
        } finally {
            removeListener();
        }
    }, [authorization.user.identity.solanaAddress, ctx.notifications, lease, setOperationStage]);

    const establishDevelopmentLease = React.useCallback(async () => {
        setOperationStage('Confirming the local development session…');
        const result = await lease.runTask(() => services.wormTrading.createDevelopmentCredentialLease());
        if (result.status === 'fulfilled') {
            return true;
        }
        if (result.status === 'rejected') {
            ctx.notifications.error('Could not confirm development session', wormConnectionErrorMessage(result.error, 'Local Worm credential access was denied.'));
        }
        return false;
    }, [ctx.notifications, lease, setOperationStage]);

    const beginGoogleReauthentication = React.useCallback(
        (intent: PendingConnectionIntent) => {
            window.sessionStorage.setItem(pendingConnectionActionKey, JSON.stringify(intent));
            setOperationStage('Opening Google for a fresh identity check…');
            window.location.assign(services.wormTrading.googleCredentialReauthenticationURL('/worm-trading'));
        },
        [setOperationStage]
    );

    const waitForAutoConnectionSlot = React.useCallback(
        async (epoch: number) => {
            const remaining = Math.max(0, lastAutoStartAtRef.current + wormConnectionStartIntervalMS - Date.now());
            if (remaining === 0) {
                return true;
            }
            setOperationStage('Waiting to respect the Worm connection rate limit…');
            await new Promise<void>(resolve => {
                const timer = window.setTimeout(resolve, remaining);
                cancelDelayRef.current = () => {
                    window.clearTimeout(timer);
                    resolve();
                };
            });
            cancelDelayRef.current = undefined;
            return isCurrent(epoch);
        },
        [isCurrent, setOperationStage]
    );

    const refreshAuthoritativeState = React.useCallback(
        async (epoch: number) => {
            onReloadRef.current();
            const items = await fetchConnectionInventory(epoch);
            if (items && isCurrent(epoch)) {
                publishInventory(items);
            }
            return items;
        },
        [fetchConnectionInventory, isCurrent, publishInventory]
    );

    const runRetirementQueue = React.useCallback(
        async (epoch: number, inventory: WormTradingWalletConnectionItem[]): Promise<WormTradingWalletConnectionItem[] | undefined> => {
            const candidates = inventory.filter(item => retirementWalletIDsRef.current.has(item.wallet.walletId));
            if (candidates.length === 0) {
                return inventory;
            }
            setWalletProgress(current => {
                const next = new Map(current);
                candidates.forEach(item => next.set(item.wallet.walletId, 'Queued for credential retirement.'));
                return next;
            });
            for (const item of candidates) {
                if (!isCurrent(epoch)) {
                    return undefined;
                }
                const walletID = item.wallet.walletId;
                setBusyWalletId(walletID);
                setSetup(progressState('connecting', 'Disconnecting removed wallets before connecting additions.', {currentWallet: item.wallet}));
                setOperationStage('Revoking the removed wallet credential…');
                setWalletProgress(current => new Map(current).set(walletID, 'Retiring Worm credential…'));
                const result = await lease.runTask(() => services.wormTrading.disconnectWallet(walletID));
                if (!isCurrent(epoch) || result.status === 'discarded') {
                    return undefined;
                }
                if (result.status === 'fulfilled') {
                    attemptedWalletIDsRef.current.add(walletID);
                    succeededWalletIDsRef.current.add(walletID);
                    setWalletProgress(current => new Map(current).set(walletID, 'Credential retirement requested; refreshing authoritative state…'));
                    continue;
                }
                const details = requestErrorDetails(result.error);
                if (details.reason === WORM_TRADING_REAUTH_REQUIRED) {
                    const callbackReason = new URLSearchParams(location.search).get('wormTradingReason') || new URLSearchParams(location.search).get('wormCredentialReason') || '';
                    setWalletProgress(current => new Map(current).set(walletID, 'Waiting for Worm credential authorization.'));
                    setSetup(
                        progressState(
                            'authorization-required',
                            callbackReason
                                ? `Google authorization did not complete (${callbackReason}). Authorize once to continue cleanup.`
                                : 'Confirm your identity once to retire removed credentials before connecting additions.'
                        )
                    );
                    return undefined;
                }
                attemptedWalletIDsRef.current.add(walletID);
                failedWalletIDsRef.current.add(walletID);
                setWalletProgress(current => new Map(current).set(walletID, 'Credential cleanup still needs attention.'));
                setSetup(
                    progressState(
                        'partial',
                        wormConnectionErrorMessage(result.error, 'A removed wallet could not be disconnected. Additions remain paused to preserve the 20-wallet limit.'),
                        {
                            retryable: details.status !== 401 && details.status !== 403
                        }
                    )
                );
                return undefined;
            }
            setBusyWalletId(undefined);
            setOperationStage('Refreshing wallet retirement state…');
            const refreshed = await refreshAuthoritativeState(epoch);
            if (!refreshed || !isCurrent(epoch)) {
                return undefined;
            }
            const refreshedSelection = await fetchWalletSelection(epoch);
            if (!refreshedSelection || !isCurrent(epoch)) {
                return undefined;
            }
            const outstandingRetirements = new Set(refreshedSelection.retirements.map(item => item.walletId));
            const incomplete = candidates.find(item => outstandingRetirements.has(item.wallet.walletId));
            if (incomplete) {
                setSetup(
                    progressState(
                        'partial',
                        `${incomplete.wallet.remark || displayIdentity(incomplete.wallet.address)} still requires credential cleanup. Additions remain paused until retirement completes.`,
                        {retryable: true}
                    )
                );
                return undefined;
            }
            return refreshed;
        },
        [fetchWalletSelection, isCurrent, lease, location.search, progressState, refreshAuthoritativeState, setOperationStage]
    );

    const runAutoConnectionQueue = React.useCallback(
        async (epoch: number, inventory: WormTradingWalletConnectionItem[]) => {
            const candidates = inventory.filter(
                item =>
                    targetWalletIDsRef.current.has(item.wallet.walletId) &&
                    item.connection.state === 'NOT_CONNECTED' &&
                    item.connection.warningCode !== connectOutcomeUnknownWarning &&
                    !attemptedWalletIDsRef.current.has(item.wallet.walletId) &&
                    !blockedWalletIDsRef.current.has(item.wallet.walletId)
            );
            if (candidates.length === 0) {
                const unknown = inventory.find(item => targetWalletIDsRef.current.has(item.wallet.walletId) && item.connection.warningCode === connectOutcomeUnknownWarning);
                if (unknown) {
                    setSetup(
                        progressState(
                            'blocked',
                            `${unknown.wallet.remark || displayIdentity(unknown.wallet.address)} has an unknown connection outcome. Automatic retry is blocked until the state is reviewed.`
                        )
                    );
                } else if (failedWalletIDsRef.current.size > 0) {
                    setSetup(
                        progressState('partial', 'No failed connection was retried automatically. Refresh the authoritative state, then retry when ready.', {retryable: true})
                    );
                } else {
                    setSetup(hiddenConnectionSetupState);
                }
                return;
            }

            setWalletProgress(current => {
                const next = new Map(current);
                candidates.forEach(item => next.set(item.wallet.walletId, 'Queued for automatic connection.'));
                return next;
            });

            let terminalPhase: ConnectionSetupPhase | undefined;
            let terminalMessage = '';
            let retryable = false;
            for (const item of candidates) {
                if (!isCurrent(epoch)) {
                    return;
                }
                const walletID = item.wallet.walletId;
                setBusyWalletId(walletID);
                setSetup(progressState('connecting', 'Waiting for the next safe Worm connection slot.', {currentWallet: item.wallet}));
                if (!(await waitForAutoConnectionSlot(epoch))) {
                    return;
                }
                setOperationStage('Creating a Worm credential…');
                setSetup(progressState('connecting', 'Creating Worm credentials one wallet at a time.', {currentWallet: item.wallet}));
                setWalletProgress(current => new Map(current).set(walletID, 'Connecting automatically…'));
                lastAutoStartAtRef.current = Date.now();
                const result = await lease.runTask(() => services.wormTrading.connectWallet(walletID));
                if (!isCurrent(epoch) || result.status === 'discarded') {
                    return;
                }
                if (result.status === 'fulfilled') {
                    attemptedWalletIDsRef.current.add(walletID);
                    if (result.value.warningCode === connectOutcomeUnknownWarning) {
                        failedWalletIDsRef.current.add(walletID);
                        blockedWalletIDsRef.current.add(walletID);
                        setWalletProgress(current => new Map(current).set(walletID, 'Connection outcome is unknown; automatic retry is blocked.'));
                        terminalPhase = 'blocked';
                        terminalMessage = 'Worm did not confirm whether it created the credential. Automatic processing stopped and this wallet will not be retried.';
                        break;
                    }
                    succeededWalletIDsRef.current.add(walletID);
                    setWalletProgress(current => new Map(current).set(walletID, 'Connected; refreshing authoritative status…'));
                    setSetup(progressState('connecting', 'The wallet connected successfully. Preparing the next wallet.', {currentWallet: item.wallet}));
                    continue;
                }

                const details = requestErrorDetails(result.error);
                if (details.reason === WORM_TRADING_REAUTH_REQUIRED) {
                    const callbackReason = new URLSearchParams(location.search).get('wormTradingReason') || new URLSearchParams(location.search).get('wormCredentialReason') || '';
                    setWalletProgress(current => {
                        const next = new Map(current);
                        candidates.forEach(candidate => {
                            if (!attemptedWalletIDsRef.current.has(candidate.wallet.walletId)) {
                                next.set(candidate.wallet.walletId, 'Waiting for Worm credential authorization.');
                            }
                        });
                        return next;
                    });
                    terminalPhase = 'authorization-required';
                    terminalMessage = callbackReason
                        ? `Google authorization did not complete (${callbackReason}). Authorize once to continue the remaining wallets.`
                        : succeededWalletIDsRef.current.size > 0
                          ? 'The Worm authorization lease expired. Authorize once to continue the remaining wallets.'
                          : 'Confirm your identity once to apply the selected Worm wallets. No transaction or network fee is involved.';
                    break;
                }

                attemptedWalletIDsRef.current.add(walletID);
                failedWalletIDsRef.current.add(walletID);
                if (details.reason === connectOutcomeUnknownWarning) {
                    blockedWalletIDsRef.current.add(walletID);
                    setWalletProgress(current => new Map(current).set(walletID, 'Connection outcome is unknown; automatic retry is blocked.'));
                    terminalPhase = 'blocked';
                    terminalMessage = wormConnectionErrorMessage(result.error, 'The connection outcome is unknown.');
                    break;
                }

                setWalletProgress(current => new Map(current).set(walletID, 'Automatic connection failed; retry from the connection setup panel.'));
                const walletLocalFailure = details.status === 400 || details.status === 404 || details.status === 409;
                if (!walletLocalFailure) {
                    terminalPhase = 'paused';
                    terminalMessage = wormConnectionErrorMessage(result.error, 'Automatic Worm connection stopped before the remaining wallets were attempted.');
                    retryable = details.status !== 401 && details.status !== 403;
                    break;
                }
                setSetup(progressState('connecting', 'One wallet failed validation. Continuing with the remaining wallets.', {currentWallet: item.wallet}));
            }

            setBusyWalletId(undefined);
            setOperationStage('Refreshing authoritative Worm connection state…');
            let refreshedInventory: WormTradingWalletConnectionItem[] | undefined;
            try {
                refreshedInventory = await refreshAuthoritativeState(epoch);
            } catch (error) {
                if (isCurrent(epoch)) {
                    setSetup(
                        progressState('paused', wormConnectionErrorMessage(error, 'Connections were processed, but the authoritative state could not be refreshed.'), {
                            retryable: true
                        })
                    );
                }
                return;
            }
            if (!isCurrent(epoch)) {
                return;
            }
            const unknown = refreshedInventory?.find(item => targetWalletIDsRef.current.has(item.wallet.walletId) && item.connection.warningCode === connectOutcomeUnknownWarning);
            if (unknown) {
                blockedWalletIDsRef.current.add(unknown.wallet.walletId);
                setWalletProgress(current => new Map(current).set(unknown.wallet.walletId, 'Connection outcome is unknown; automatic retry is blocked.'));
                setSetup(
                    progressState(
                        'blocked',
                        `${unknown.wallet.remark || displayIdentity(unknown.wallet.address)} has an unknown connection outcome. Automatic retry is blocked until the state is reviewed.`,
                        {retryable: false}
                    )
                );
                return;
            }
            if (terminalPhase) {
                setSetup(progressState(terminalPhase, terminalMessage, {retryable}));
                return;
            }
            if (failedWalletIDsRef.current.size > 0) {
                const partialState = progressState('partial', 'Eligible wallets were processed. Failed wallets were not retried automatically.', {retryable: true});
                setSetup(partialState);
                targetWalletIDsRef.current = new Set(failedWalletIDsRef.current);
                attemptedWalletIDsRef.current = new Set(failedWalletIDsRef.current);
                succeededWalletIDsRef.current.clear();
                return;
            }
            const connectedCount = succeededWalletIDsRef.current.size;
            if (connectedCount > 0) {
                ctx.notifications.success(
                    connectedCount === 1 ? 'Worm wallet selection applied' : `${connectedCount} Worm wallet changes applied`,
                    'The authoritative connection and asset state has been refreshed.'
                );
            }
            targetWalletIDsRef.current.clear();
            retirementWalletIDsRef.current.clear();
            attemptedWalletIDsRef.current.clear();
            succeededWalletIDsRef.current.clear();
            failedWalletIDsRef.current.clear();
            blockedWalletIDsRef.current.clear();
            setWalletProgress(new Map());
            setSetup(hiddenConnectionSetupState);
        },
        [ctx.notifications, isCurrent, lease, location.search, progressState, refreshAuthoritativeState, setOperationStage, waitForAutoConnectionSlot]
    );

    const discoverAndRun = React.useCallback(
        async (epoch: number, resetBatch: boolean, runConnections: boolean) => {
            setSetup(current => ({...current, phase: 'discovering', message: 'Loading your Worm wallet selection and current credentials.', currentWallet: undefined}));
            const currentSelection = await fetchWalletSelection(epoch);
            if (!currentSelection || !isCurrent(epoch)) {
                return;
            }
            const inventory = await fetchConnectionInventory(epoch);
            if (!inventory || !isCurrent(epoch)) {
                return;
            }
            const inventorySelection = inventorySelectionRef.current;
            if (
                !inventorySelection ||
                inventorySelection.configured !== currentSelection.configured ||
                inventorySelection.revision !== currentSelection.revision ||
                inventorySelection.updatedAt !== currentSelection.updatedAt ||
                inventorySelection.maximumWallets !== currentSelection.maximumWallets
            ) {
                throw new Error('The saved Worm wallet selection changed while its connection inventory was loading. Refresh before continuing.');
            }
            const selectedByWalletID = new Map(currentSelection.selectedItems.map(item => [item.walletId, item]));
            const retiringByWalletID = new Map(currentSelection.retirements.map(item => [item.walletId, item]));
            if (
                inventory.some(item => {
                    const selectedItem = selectedByWalletID.get(item.wallet.walletId);
                    const retirement = retiringByWalletID.get(item.wallet.walletId);
                    return (
                        item.selected !== Boolean(selectedItem) ||
                        item.selectionOrdinal !== (selectedItem?.ordinal || 0) ||
                        (selectedItem !== undefined && selectedItem.address !== item.wallet.address) ||
                        item.retirementPending !== Boolean(retirement) ||
                        (retirement !== undefined && retirement.address !== item.wallet.address)
                    );
                })
            ) {
                throw new Error('Worm Trading returned a connection inventory for a different saved wallet selection.');
            }
            publishInventory(inventory);
            if (resetBatch) {
                targetWalletIDsRef.current.clear();
                retirementWalletIDsRef.current.clear();
                attemptedWalletIDsRef.current.clear();
                succeededWalletIDsRef.current.clear();
                failedWalletIDsRef.current.clear();
                blockedWalletIDsRef.current.clear();
                setWalletProgress(new Map());
            }
            if (!currentSelection.configured) {
                targetWalletIDsRef.current.clear();
                retirementWalletIDsRef.current.clear();
                setSetup(hiddenConnectionSetupState);
                return;
            }
            const selectedWalletIDs = new Set(currentSelection.selectedItems.map(item => item.walletId));
            const retiredWalletIDs = new Set(currentSelection.retirements.map(item => item.walletId));
            const inventoryWalletIDs = new Set(inventory.map(item => item.wallet.walletId));
            const missingRetirement = currentSelection.retirements.find(item => !inventoryWalletIDs.has(item.walletId));
            targetWalletIDsRef.current = new Set([...targetWalletIDsRef.current].filter(walletID => selectedWalletIDs.has(walletID)));
            retirementWalletIDsRef.current = new Set([...retirementWalletIDsRef.current].filter(walletID => retiredWalletIDs.has(walletID)));
            for (const item of inventory) {
                const walletID = item.wallet.walletId;
                if (retiredWalletIDs.has(walletID)) {
                    retirementWalletIDsRef.current.add(walletID);
                }
                if (!selectedWalletIDs.has(walletID)) {
                    continue;
                }
                if (item.connection.warningCode === connectOutcomeUnknownWarning) {
                    blockedWalletIDsRef.current.add(item.wallet.walletId);
                    targetWalletIDsRef.current.add(item.wallet.walletId);
                } else if (item.connection.state === 'NOT_CONNECTED' && !attemptedWalletIDsRef.current.has(item.wallet.walletId)) {
                    targetWalletIDsRef.current.add(item.wallet.walletId);
                }
            }
            if (missingRetirement) {
                setSetup(
                    progressState(
                        'blocked',
                        `Retired wallet ${displayIdentity(missingRetirement.address)} is no longer in the current Solana wallet inventory. Credential additions remain blocked until cleanup can be resolved.`
                    )
                );
                return;
            }
            if (runConnections) {
                const afterRetirements = await runRetirementQueue(epoch, inventory);
                if (!afterRetirements || !isCurrent(epoch)) {
                    return;
                }
                await runAutoConnectionQueue(epoch, afterRetirements);
                return;
            }
            if (failedWalletIDsRef.current.size > 0) {
                setSetup(progressState('partial', 'Connection state refreshed. Failed wallets were not retried automatically.', {retryable: true}));
            } else {
                setSetup(hiddenConnectionSetupState);
            }
        },
        [fetchConnectionInventory, fetchWalletSelection, isCurrent, progressState, publishInventory, runAutoConnectionQueue, runRetirementQueue]
    );

    const startDiscovery = React.useCallback(
        async (resetBatch: boolean, runConnections: boolean) => {
            const epoch = beginOperation();
            if (epoch === undefined) {
                return;
            }
            try {
                await discoverAndRun(epoch, resetBatch, runConnections);
            } catch (error) {
                if (isCurrent(epoch)) {
                    setSelectionError(error instanceof Error ? error : new Error(String(error)));
                    setSetup(progressState('paused', wormConnectionErrorMessage(error, 'Could not check the Worm connection inventory.'), {retryable: true}));
                }
            } finally {
                finishOperation(epoch);
            }
        },
        [beginOperation, discoverAndRun, finishOperation, isCurrent, progressState]
    );

    const authorize = React.useCallback(async () => {
        const epoch = beginOperation();
        if (epoch === undefined) {
            return;
        }
        setSetup(progressState('connecting', 'Confirming your identity once for the remaining wallets.'));
        try {
            let established = false;
            switch (authorization.user.identity.provider) {
                case AccountIdentityProvider.Google:
                    beginGoogleReauthentication({kind: 'reconcile-selection'});
                    return;
                case AccountIdentityProvider.SolanaWallet:
                    established = await establishSolanaLease();
                    break;
                case AccountIdentityProvider.Development:
                    established = await establishDevelopmentLease();
                    break;
                default:
                    ctx.notifications.error('Reauthentication is unavailable', 'This login identity cannot approve Worm credential management.');
            }
            if (!established) {
                if (isCurrent(epoch)) {
                    setSetup(progressState('authorization-required', 'Identity confirmation did not complete. No wallet connection was retried.'));
                }
                return;
            }
            await discoverAndRun(epoch, false, true);
        } catch (error) {
            if (isCurrent(epoch)) {
                setSetup(progressState('paused', wormConnectionErrorMessage(error, 'Could not continue Worm wallet connections.'), {retryable: true}));
            }
        } finally {
            finishOperation(epoch);
        }
    }, [
        authorization.user.identity.provider,
        beginGoogleReauthentication,
        beginOperation,
        ctx.notifications,
        discoverAndRun,
        establishDevelopmentLease,
        establishSolanaLease,
        finishOperation,
        isCurrent,
        progressState
    ]);

    const runManagedAction = React.useCallback(
        async (action: ManagedConnectionAction, walletId: number, resumed = false) => {
            const epoch = beginOperation();
            if (epoch === undefined) {
                return;
            }
            const cleanup = action === 'cleanup';
            const regenerate = action === 'regenerate';
            setBusyWalletId(walletId);
            setSetup(
                progressState(
                    'connecting',
                    cleanup
                        ? 'Retrying Worm credential cleanup.'
                        : regenerate
                          ? 'Preparing a new Worm credential after an unknown outcome.'
                          : 'Preparing a replacement Worm credential.'
                )
            );
            setOperationStage(cleanup ? 'Requesting credential cleanup…' : regenerate ? 'Preparing a new credential…' : 'Preparing a replacement credential…');
            const request = () =>
                lease.runTask(() =>
                    action === 'reconnect'
                        ? services.wormTrading.reconnectWallet(walletId)
                        : action === 'regenerate'
                          ? services.wormTrading.regenerateWallet(walletId)
                          : services.wormTrading.disconnectWallet(walletId)
                );
            try {
                let result = await request();
                if (result.status === 'rejected' && requestErrorDetails(result.error).reason === WORM_TRADING_REAUTH_REQUIRED && !resumed) {
                    switch (authorization.user.identity.provider) {
                        case AccountIdentityProvider.Google:
                            beginGoogleReauthentication({kind: action, walletId});
                            return;
                        case AccountIdentityProvider.SolanaWallet:
                            if (await establishSolanaLease()) {
                                setOperationStage(
                                    cleanup ? 'Cleaning up the Worm credential…' : regenerate ? 'Creating the new Worm credential…' : 'Creating the replacement Worm credential…'
                                );
                                result = await request();
                            }
                            break;
                        case AccountIdentityProvider.Development:
                            if (await establishDevelopmentLease()) {
                                setOperationStage(
                                    cleanup ? 'Cleaning up the Worm credential…' : regenerate ? 'Creating the new Worm credential…' : 'Creating the replacement Worm credential…'
                                );
                                result = await request();
                            }
                            break;
                        default:
                            ctx.notifications.error('Reauthentication is unavailable', 'This login identity cannot approve Worm credential management.');
                    }
                }
                if (!isCurrent(epoch) || result.status === 'discarded') {
                    return;
                }
                if (result.status === 'fulfilled') {
                    if (!cleanup && result.value.warningCode === connectOutcomeUnknownWarning) {
                        ctx.notifications.error(
                            'Could not reconnect Worm wallet',
                            'Worm did not confirm whether it created the new credential. The unknown connection state remains and Athena will not retry automatically.'
                        );
                        onReloadRef.current();
                        const inventory = await fetchConnectionInventory(epoch);
                        if (inventory && isCurrent(epoch)) {
                            publishInventory(inventory);
                            setSetup(progressState('blocked', 'The Worm connection outcome is still unknown. Automatic retry remains blocked.'));
                        }
                        return;
                    }
                    ctx.notifications.success(cleanup ? 'Worm credential cleanup completed' : 'Worm wallet reconnected');
                    if (cleanup || regenerate) {
                        attemptedWalletIDsRef.current.delete(walletId);
                        failedWalletIDsRef.current.delete(walletId);
                        blockedWalletIDsRef.current.delete(walletId);
                        succeededWalletIDsRef.current.delete(walletId);
                        targetWalletIDsRef.current.delete(walletId);
                        retirementWalletIDsRef.current.delete(walletId);
                        setWalletProgress(current => {
                            const next = new Map(current);
                            next.delete(walletId);
                            return next;
                        });
                    }
                    onReloadRef.current();
                    await discoverAndRun(epoch, false, true);
                    return;
                }
                const callbackReason = new URLSearchParams(location.search).get('wormTradingReason') || new URLSearchParams(location.search).get('wormCredentialReason') || '';
                const fallback =
                    resumed && callbackReason
                        ? `Google reauthentication did not complete (${callbackReason}).`
                        : `Could not ${cleanup ? 'clean up' : 'reconnect'} this Worm wallet.`;
                ctx.notifications.error(cleanup ? 'Could not clean up Worm credential' : 'Could not reconnect Worm wallet', wormConnectionErrorMessage(result.error, fallback));
                onReloadRef.current();
                const inventory = await fetchConnectionInventory(epoch);
                if (inventory && isCurrent(epoch)) {
                    publishInventory(inventory);
                    const unknown = inventory.find(item => item.wallet.walletId === walletId && item.connection.warningCode === connectOutcomeUnknownWarning);
                    setSetup(
                        unknown
                            ? progressState('blocked', 'The Worm connection outcome is unknown. Automatic retry is blocked until the state is reviewed.')
                            : hiddenConnectionSetupState
                    );
                }
            } catch (error) {
                if (isCurrent(epoch)) {
                    ctx.notifications.error('Could not refresh Worm connection state', wormConnectionErrorMessage(error, 'The previous connection state remains visible.'));
                    setSetup(hiddenConnectionSetupState);
                }
            } finally {
                finishOperation(epoch);
            }
        },
        [
            authorization.user.identity.provider,
            beginGoogleReauthentication,
            beginOperation,
            ctx.notifications,
            discoverAndRun,
            establishDevelopmentLease,
            establishSolanaLease,
            fetchConnectionInventory,
            finishOperation,
            isCurrent,
            lease,
            location.search,
            progressState,
            publishInventory,
            setOperationStage
        ]
    );

    React.useEffect(() => {
        if (resumedRef.current) {
            return;
        }
        resumedRef.current = true;
        const pending = readPendingConnectionIntent();
        if (pending && pending.kind !== 'reconcile-selection') {
            void runManagedAction(pending.kind, pending.walletId, true);
            return;
        }
        void startDiscovery(true, true);
    }, [runManagedAction, startDiscovery]);

    const confirm = React.useCallback(
        (action: ManagedConnectionAction, walletId: number, walletLabel: string) => {
            const cleanup = action === 'cleanup';
            const regenerate = action === 'regenerate';
            ctx.modal.confirm({
                className: 'worm-confirm-modal',
                title: cleanup ? `Retry credential cleanup for ${walletLabel}?` : `Reconnect ${walletLabel} to Worm?`,
                content: cleanup ? (
                    <Typography.Paragraph>
                        Athena will retry revoking only the Worm API credential it created for this wallet. This does not cancel orders, close positions, or move funds.
                    </Typography.Paragraph>
                ) : regenerate ? (
                    <>
                        <Typography.Paragraph>
                            Athena will create, securely store, and use a new Worm API credential for this wallet. A credential from the earlier attempt may already exist and may
                            remain valid.
                        </Typography.Paragraph>
                        <Typography.Paragraph>
                            Athena will not list or revoke that unknown credential. This operation does not sign a transaction, place or cancel orders, close positions, or move
                            funds.
                        </Typography.Paragraph>
                    </>
                ) : (
                    <Typography.Paragraph>
                        Athena will create and securely store a replacement Worm API credential. The previous Athena credential remains recorded until Worm confirms its revocation.
                        This does not cancel orders, close positions, or move funds.
                    </Typography.Paragraph>
                ),
                okText: cleanup ? 'Retry cleanup' : regenerate ? 'Regenerate and reconnect' : 'Reconnect',
                onOk: () => runManagedAction(action, walletId)
            });
        },
        [ctx.modal, runManagedAction]
    );

    const saveWalletSelection = React.useCallback(
        async (walletIDs: number[]) => {
            const currentSelection = selection;
            if (!currentSelection) {
                return;
            }
            const epoch = beginOperation();
            if (epoch === undefined) {
                return;
            }
            setSetup(progressState('connecting', 'Saving your Worm wallet selection before applying credential changes.'));
            setOperationStage('Saving Worm wallet selection…');
            try {
                const result = await lease.runTask(() => services.wormTrading.replaceWalletSelection({expectedRevision: currentSelection.revision, walletIds: walletIDs}));
                if (!isCurrent(epoch) || result.status === 'discarded') {
                    return;
                }
                if (result.status === 'rejected') {
                    const errorDetails = requestErrorDetails(result.error);
                    const errorMessage = `${errorDetails.message || ''} ${errorDetails.reason || ''}`.toLowerCase();
                    const revisionConflict = errorDetails.status === 409 && errorDetails.code === 10 && errorMessage.includes('wallet selection revision');
                    if (revisionConflict) {
                        const authoritative = await fetchWalletSelection(epoch);
                        if (authoritative && isCurrent(epoch)) {
                            const inventory = await fetchConnectionInventory(epoch);
                            if (inventory && isCurrent(epoch)) {
                                publishInventory(inventory);
                            }
                            setSelectionModalOpen(true);
                        }
                        ctx.notifications.warning('Wallet selection changed', 'The latest saved selection is now shown. Review it before saving again.');
                    } else {
                        const message = wormConnectionErrorMessage(result.error, 'The previous wallet selection remains unchanged.');
                        ctx.notifications.error('Could not save Worm wallets', message);
                        setSetup(progressState('paused', message, {retryable: true}));
                        const inventory = await fetchConnectionInventory(epoch);
                        if (inventory && isCurrent(epoch)) {
                            publishInventory(inventory);
                        }
                    }
                    if (revisionConflict) {
                        setSetup(hiddenConnectionSetupState);
                    }
                    return;
                }
                setSelection(result.value);
                setSelectionError(undefined);
                setSelectionModalOpen(false);
                ctx.notifications.success(
                    'Worm wallet selection saved',
                    walletIDs.length === 0
                        ? 'No wallets are selected. Any retired credentials will continue cleanup.'
                        : `${walletIDs.length} ${walletIDs.length === 1 ? 'wallet is' : 'wallets are'} selected. Credential changes are now being applied.`
                );
                onReloadRef.current();
                await discoverAndRun(epoch, true, true);
            } catch (error) {
                if (isCurrent(epoch)) {
                    ctx.notifications.error(
                        'Could not apply Worm wallets',
                        wormConnectionErrorMessage(error, 'The saved selection remains authoritative. Refresh to review its status.')
                    );
                    setSetup(progressState('paused', wormConnectionErrorMessage(error, 'Could not apply the saved Worm wallet selection.'), {retryable: true}));
                }
            } finally {
                finishOperation(epoch);
            }
        },
        [
            beginOperation,
            ctx.notifications,
            discoverAndRun,
            fetchConnectionInventory,
            fetchWalletSelection,
            finishOperation,
            isCurrent,
            lease,
            progressState,
            publishInventory,
            selection,
            setOperationStage
        ]
    );

    const refreshInventory = React.useCallback(() => void startDiscovery(false, true), [startDiscovery]);
    const retry = React.useCallback(() => void startDiscovery(true, true), [startDiscovery]);
    const connections = React.useMemo(() => new Map(connectionItems.map(item => [item.wallet.walletId, item.connection])), [connectionItems]);
    const manager = React.useMemo<ConnectionManager>(
        () => ({
            busyWalletId,
            stage,
            operationBusy,
            setup,
            connections,
            inventory: connectionItems,
            selection,
            selectionError,
            walletProgress,
            authorize: () => void authorize(),
            retry,
            refreshInventory,
            openSelection: () => setSelectionModalOpen(true),
            confirm
        }),
        [authorize, busyWalletId, confirm, connectionItems, connections, operationBusy, refreshInventory, retry, selection, selectionError, setup, stage, walletProgress]
    );
    return (
        <>
            {props.children(manager)}
            {selection && (
                <WalletSelectionModal
                    open={selectionModalOpen}
                    inventory={connectionItems}
                    selection={selection}
                    saving={operationBusy}
                    error={selectionError}
                    onCancel={() => setSelectionModalOpen(false)}
                    onSave={walletIDs => void saveWalletSelection(walletIDs)}
                />
            )}
        </>
    );
};

interface ConnectionManager {
    busyWalletId?: number;
    stage: string;
    operationBusy: boolean;
    setup: ConnectionSetupState;
    connections: Map<number, WormWalletConnection>;
    inventory: WormTradingWalletConnectionItem[];
    selection?: WormTradingWalletSelection;
    selectionError?: Error;
    walletProgress: Map<number, string>;
    authorize(): void;
    retry(): void;
    refreshInventory(): void;
    openSelection(): void;
    confirm(action: ManagedConnectionAction, walletId: number, walletLabel: string): void;
}

const WalletSelectionSummary = (props: {manager: ConnectionManager}) => {
    const selection = props.manager.selection;
    if (!selection?.configured) {
        return null;
    }
    const selectedWalletIDs = new Set(selection.selectedItems.map(item => item.walletId));
    const selectedConnections = props.manager.inventory.filter(item => selectedWalletIDs.has(item.wallet.walletId));
    const connected = selectedConnections.filter(item => item.connection.state === 'CONNECTED').length;
    const waiting = selection.selectedItems.filter(selected => {
        const inventoryItem = props.manager.inventory.find(item => item.wallet.walletId === selected.walletId);
        return !inventoryItem || inventoryItem.pendingState === 'CONNECT_PENDING' || inventoryItem.connection.state !== 'CONNECTED';
    }).length;
    const retiring = selection.retirements.length;
    const attentionWalletIDs = new Set(props.manager.inventory.filter(item => item.needsAttention).map(item => item.wallet.walletId));
    selection.retirements.forEach(item => attentionWalletIDs.add(item.walletId));
    selection.selectedItems.forEach(item => {
        if (!props.manager.inventory.some(inventoryItem => inventoryItem.wallet.walletId === item.walletId)) {
            attentionWalletIDs.add(item.walletId);
        }
    });
    const attention = attentionWalletIDs.size;
    return (
        <section className='worm-wallet-selection-summary' aria-label='Worm Trading wallet selection status'>
            <div className='worm-wallet-selection-summary__heading'>
                <span>
                    <SettingOutlined aria-hidden='true' />
                    <Typography.Title level={2}>Worm wallets</Typography.Title>
                </span>
                <Button size='small' disabled={props.manager.operationBusy || Boolean(props.manager.selectionError)} onClick={props.manager.openSelection}>
                    Manage selection · {selection.selectedItems.length}/{selection.maximumWallets}
                </Button>
            </div>
            <dl>
                <div>
                    <dt>Selected</dt>
                    <dd>{selection.selectedItems.length}</dd>
                </div>
                <div>
                    <dt>Connected</dt>
                    <dd>{connected}</dd>
                </div>
                <div>
                    <dt>Waiting</dt>
                    <dd>{waiting}</dd>
                </div>
                <div className={attention > 0 ? 'worm-wallet-selection-summary__attention' : undefined}>
                    <dt>Needs attention</dt>
                    <dd>{attention}</dd>
                </div>
            </dl>
            {retiring > 0 && (
                <Typography.Text type='secondary' role='status'>
                    {retiring} removed {retiring === 1 ? 'wallet is' : 'wallets are'} retained for credential cleanup and {retiring === 1 ? 'remains' : 'remain'} available under
                    Needs attention.
                </Typography.Text>
            )}
        </section>
    );
};

const WalletSelectionGuide = (props: {manager: ConnectionManager}) => {
    if (!props.manager.selection || props.manager.selectionError) {
        return props.manager.selectionError ? (
            <Result
                status='error'
                title='Worm wallet settings are unavailable'
                subTitle={requestErrorMessage(props.manager.selectionError, 'No wallet was connected automatically.')}
                extra={
                    <Button type='primary' icon={<ReloadOutlined />} disabled={props.manager.operationBusy} onClick={props.manager.retry}>
                        Try again
                    </Button>
                }
            />
        ) : (
            <div className='worm-wallet-selection-guide-loading' aria-label='Loading Worm wallet settings'>
                <Skeleton active={true} paragraph={{rows: 4}} />
            </div>
        );
    }
    return (
        <Result
            className='worm-wallet-selection-guide'
            icon={<WalletOutlined />}
            title='Choose wallets for Worm Trading'
            subTitle={`Athena will not connect any wallet until you save a selection. Choose up to ${props.manager.selection.maximumWallets}; your selection is restored the next time you open this page.`}
            extra={
                <Button type='primary' icon={<SettingOutlined />} disabled={props.manager.operationBusy} onClick={props.manager.openSelection}>
                    Choose Worm wallets
                </Button>
            }
        />
    );
};

const ReadOnlyWalletSelectionGuide = () => (
    <Result
        className='worm-wallet-selection-guide'
        icon={<WalletOutlined />}
        title='Worm Trading wallets are not configured'
        subTitle='No wallet is connected automatically. A user with Worm Trading write access must choose and save the wallets for this account.'
    />
);

const ConnectionCell = (props: {
    wallet: WormTradingWalletSummary;
    connection?: WormWalletConnection;
    activityStatus?: WormTradingWalletActivityItem['status'];
    manager?: ConnectionManager;
    loading?: boolean;
}) => {
    if (!props.connection) {
        return (
            <div className='worm-trading-connection'>
                <Tag>{props.loading ? 'Loading access…' : 'Access unavailable'}</Tag>
            </div>
        );
    }
    const busy = props.manager?.busyWalletId === props.wallet.walletId;
    const state = props.connection.state;
    const label = props.wallet.remark || 'Solana wallet';
    const actions =
        state === 'RECONNECT_REQUIRED' && props.connection.warningCode === connectOutcomeUnknownWarning
            ? [{kind: 'regenerate' as const, label: 'Reconnect', icon: <SyncOutlined />}]
            : props.connection.warningCode === connectOutcomeUnknownWarning
              ? []
              : state === 'RECONNECT_REQUIRED'
                ? [{kind: 'reconnect' as const, label: 'Reconnect', icon: <SyncOutlined />}]
                : state === 'DISCONNECTING' || state === 'REVOCATION_REQUIRED'
                  ? [{kind: 'cleanup' as const, label: 'Retry credential cleanup', icon: <DisconnectOutlined />}]
                  : [];
    return (
        <div className='worm-trading-connection'>
            <div className='worm-trading-connection__state'>
                <ConnectionStatusTag state={state} />
                {props.activityStatus && (connectionWasQueried(state) ? <ActivityStatusTag status={props.activityStatus} /> : <Tag>Activity not queried</Tag>)}
                {props.connection.warningCode && <small>{titleCase(props.connection.warningCode)}</small>}
                {props.manager?.walletProgress.get(props.wallet.walletId) && <small>{props.manager.walletProgress.get(props.wallet.walletId)}</small>}
            </div>
            <details className='worm-connection-details'>
                <summary>{props.manager ? 'Manage connection' : 'Connection details'}</summary>
                {props.connection.connectedAt > 0 && <small>Connected since {formatBeijingUnixSeconds(props.connection.connectedAt)}</small>}
                {props.manager && actions.length > 0 && (
                    <Space size={4} wrap={true}>
                        {actions.map(action => (
                            <Button
                                key={action.kind}
                                size='small'
                                danger={action.kind === 'cleanup'}
                                icon={action.icon}
                                loading={busy}
                                disabled={props.manager.operationBusy}
                                onClick={() => props.manager?.confirm(action.kind, props.wallet.walletId, label)}>
                                {action.label}
                            </Button>
                        ))}
                    </Space>
                )}
            </details>
            {busy && props.manager?.stage && <small className='worm-trading-connection__stage'>{props.manager.stage}</small>}
        </div>
    );
};

const WalletBalanceCard = (props: {
    item: WormTradingWalletBalanceItem;
    connection?: WormWalletConnection;
    activityStatus?: WormTradingWalletActivityItem['status'];
    manager?: ConnectionManager;
    batchManager?: PositionCashOutBatchManager;
    activityLoading?: boolean;
    onCopy: () => void;
}) => {
    const walletID = props.item.wallet.walletId;
    const selected = props.batchManager?.selectedWalletIDs.has(walletID) || false;
    const selectionDisabled = props.batchManager?.selectionDisabled(walletID) || false;
    return (
        <Card className='worm-trading-balance-card' size='small'>
            {props.batchManager && (
                <div className='worm-position-cash-out-batch-wallet-select'>
                    <Checkbox
                        checked={selected}
                        disabled={selectionDisabled}
                        aria-label={`Include ${props.item.wallet.remark || displayIdentity(props.item.wallet.address)} in Cash Out batch`}
                        onChange={event => props.batchManager?.toggleWallet(walletID, event.target.checked)}>
                        Include in Cash Out batch
                    </Checkbox>
                </div>
            )}
            <div className='worm-trading-balance-card__header'>
                <WalletIdentity wallet={props.item.wallet} compact={true} onCopy={props.onCopy} />
                {props.item.status !== 'COMPLETE' && <BalanceStatusTag status={props.item.status} />}
            </div>
            <div className='worm-trading-balance-card__assets'>
                <div>
                    <span>SOL balance</span>
                    <AssetValue asset={props.item.sol} symbol='SOL' />
                </div>
                <div>
                    <span>USDC balance</span>
                    <AssetValue asset={props.item.usdc} symbol='USDC' token={props.item.usdc} />
                </div>
            </div>
            <div className='worm-trading-balance-card__connection'>
                <span>Worm access</span>
                <ConnectionCell
                    wallet={props.item.wallet}
                    connection={props.connection}
                    activityStatus={props.activityStatus}
                    manager={props.manager}
                    loading={props.activityLoading}
                />
            </div>
        </Card>
    );
};

const EmptyWalletBalances = () => {
    const authorization = useAuthorization();
    const navigate = useNavigate();
    const canReadWallets = authorization.canRead(AccountDataModule.Wallet);
    const canWriteWallets = authorization.canWrite(AccountDataModule.Wallet);
    const detail = canWriteWallets
        ? 'Create or import a Solana wallet in Wallets, then refresh this page.'
        : canReadWallets
          ? 'You can review Wallets, but creating or importing one requires Wallet write access.'
          : 'Ask an administrator for Wallet write access before creating or importing a Solana wallet.';
    const action = canWriteWallets
        ? {label: 'Add Solana wallet', path: '/wallet', primary: true}
        : canReadWallets
          ? {label: 'View Wallets', path: '/wallet', primary: false}
          : {label: 'Review my access', path: '/account/access', primary: false};
    return (
        <div className='worm-trading-empty'>
            <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='No Solana wallets are available for Worm Trading.'>
                <Typography.Paragraph type='secondary'>{detail}</Typography.Paragraph>
                <Button type={action.primary ? 'primary' : 'default'} icon={<WalletOutlined />} onClick={() => navigate(action.path)}>
                    {action.label}
                </Button>
            </Empty>
        </div>
    );
};

interface PositionRow {
    wallet: WormTradingWalletSummary;
    position: WormOpenPosition;
}

interface RequestRow {
    wallet: WormTradingWalletSummary;
    request: WormInFlightRequest;
}

const MarketIdentity = (props: {market: WormOpenPosition['market']}) => (
    <div className='worm-trading-market'>
        <Avatar shape='square' size={38} src={props.market.logo || undefined} icon={<ApiOutlined />} />
        <div>
            <Typography.Text strong={true}>{props.market.title || 'Untitled market'}</Typography.Text>
            <small>
                {props.market.eventTitle || displayIdentity(props.market.conditionId)}
                {props.market.lastTradePrice ? ` · Latest ${props.market.lastTradePrice}` : ''}
            </small>
        </div>
    </div>
);

const SideTag = (props: {side: string}) => <Tag color={props.side === 'YES' ? 'success' : props.side === 'NO' ? 'processing' : 'default'}>{props.side || 'Unknown'}</Tag>;

const PositionCard = (props: {row: PositionRow; onCopy: () => void; cashOutManager?: PositionCashOutManager}) => {
    const position = props.row.position;
    return (
        <Card className='worm-trading-activity-card' size='small'>
            <MarketIdentity market={position.market} />
            <div className='worm-trading-activity-card__tags'>
                <span>{props.row.wallet.remark || 'Solana wallet'}</span>
                <SideTag side={position.side} />
                <Tag>{optionalValue(position.leverage, '×')}</Tag>
                {position.isLiquidated && <Tag color='error'>Liquidated</Tag>}
                {position.isClaimed && <Tag color='processing'>Claimed</Tag>}
            </div>
            <dl className='worm-trading-activity-card__facts'>
                <div>
                    <dt>Shares</dt>
                    <dd>
                        {optionalValue(position.totalShares)}
                        <small>Entry {optionalValue(position.averageEntryPrice)}</small>
                    </dd>
                </div>
                <div>
                    <dt>Liquidity</dt>
                    <dd>
                        {optionalValue(position.userLiquidity)}
                        <small>Total {optionalValue(position.totalLiquidity)}</small>
                    </dd>
                </div>
                <div>
                    <dt>Unrealized P&amp;L</dt>
                    <dd
                        className={
                            position.unrealizedPnL
                                ? Number(position.unrealizedPnL) > 0
                                    ? 'worm-value-positive'
                                    : Number(position.unrealizedPnL) < 0
                                      ? 'worm-value-negative'
                                      : ''
                                : 'worm-value-unavailable'
                        }>
                        {position.unrealizedPnL ? `${Number(position.unrealizedPnL) > 0 ? '+' : ''}${position.unrealizedPnL}` : 'Unavailable'}
                        <small>Realized {optionalValue(position.realizedPnL)}</small>
                    </dd>
                </div>
                <div>
                    <dt>Liquidation</dt>
                    <dd>
                        {liquidationPriceValue(position)}
                        <small>Latest market price {optionalValue(position.market.lastTradePrice)}</small>
                    </dd>
                </div>
            </dl>
            <details className='worm-trading-activity-evidence'>
                <summary>Wallet &amp; position evidence</summary>
                <WalletIdentity wallet={props.row.wallet} compact={true} onCopy={props.onCopy} />
                <div className='worm-trading-activity-card__footer'>
                    <code title={position.pubkey}>{displayIdentity(position.pubkey)}</code>
                    <span>{formatBeijingUnixSeconds(position.createdAt) || '-'}</span>
                </div>
                <div>
                    Market <code>{position.market.conditionId}</code>
                </div>
                <div>
                    Event <code>{position.market.eventConditionId}</code>
                </div>
            </details>
            {props.cashOutManager && (
                <div className='worm-position-cash-out-card-action'>
                    <PositionCashOutButton row={props.row} manager={props.cashOutManager} compact={true} />
                </div>
            )}
        </Card>
    );
};

const RequestCard = (props: {row: RequestRow; onCopy: () => void}) => {
    const request = props.row.request;
    return (
        <Card className='worm-trading-activity-card' size='small'>
            <MarketIdentity market={request.market} />
            <div className='worm-trading-activity-card__tags'>
                <span>{props.row.wallet.remark || 'Solana wallet'}</span>
                <SideTag side={request.side} />
                <Tag>{titleCase(request.type) || 'Request'}</Tag>
                <Tag color='processing'>{titleCase(request.state)}</Tag>
            </div>
            <dl className='worm-trading-activity-card__facts'>
                <div>
                    <dt>Leverage</dt>
                    <dd>{optionalValue(request.leverage, '×')}</dd>
                </div>
                <div>
                    <dt>Funds</dt>
                    <dd>{optionalValue(request.funds)}</dd>
                </div>
                <div>
                    <dt>Price</dt>
                    <dd>{optionalValue(request.price)}</dd>
                </div>
                <div>
                    <dt>Shares</dt>
                    <dd>{optionalValue(request.shares)}</dd>
                </div>
                <div>
                    <dt>Order state</dt>
                    <dd>{titleCase(request.orderState) || '-'}</dd>
                </div>
            </dl>
            <details className='worm-trading-activity-evidence'>
                <summary>Wallet &amp; request evidence</summary>
                <WalletIdentity wallet={props.row.wallet} compact={true} onCopy={props.onCopy} />
                <div className='worm-trading-activity-card__footer'>
                    <code title={request.pubkey}>{displayIdentity(request.pubkey)}</code>
                    <span>{formatBeijingUnixSeconds(request.createdAt) || '-'}</span>
                </div>
                <div>
                    Market <code>{request.market.conditionId}</code>
                </div>
                <div>
                    Event <code>{request.market.eventConditionId}</code>
                </div>
            </details>
        </Card>
    );
};

const StreamNotice = (props: {label: string; streams: WormActivityStreamState[]}) => {
    const unavailable = props.streams.filter(stream => !streamAvailable(stream));
    const truncated = props.streams.filter(stream => stream.truncated).length;
    if (!unavailable.length && !truncated) {
        return null;
    }
    const codes = Array.from(new Set(unavailable.map(stream => titleCase(stream.errorCode || stream.availability)).filter(Boolean)));
    return (
        <Alert
            className='worm-trading-stream-notice'
            type={unavailable.length ? 'warning' : 'info'}
            showIcon={true}
            title={
                unavailable.length
                    ? `${props.label} are unavailable for ${unavailable.length} ${unavailable.length === 1 ? 'wallet' : 'wallets'}.`
                    : `${props.label} are truncated.`
            }
            description={[
                unavailable.length ? codes.join(', ') || 'The affected streams were not treated as empty.' : '',
                truncated ? `${truncated} wallet streams show only the first 100 records.` : ''
            ]
                .filter(Boolean)
                .join(' ')}
        />
    );
};

interface PositionCashOutView {
    operationId: string;
    state: WormPositionCashOutProjection['state'];
    reasonCode: string;
    allowedAction: WormPositionCashOutAllowedAction;
    batchId: string;
    batchState: WormPositionCashOutProjection['batchState'];
    batchItemState: WormPositionCashOutProjection['batchItemState'];
    batchLockReasonCode: string;
    revision: number;
    updatedAt: number;
}

interface PositionCashOutBatchManager {
    batch?: WormPositionCashOutBatch;
    selectedWalletIDs: ReadonlySet<number>;
    toggleWallet(walletID: number, selected: boolean): void;
    clearSelection(): void;
    selectionDisabled(walletID: number): boolean;
    walletLocked(walletID: number): boolean;
    itemStateFor(row: PositionRow): WormPositionCashOutBatchItemState | '';
    selectionSummary: React.ReactNode;
    panel: React.ReactNode;
}

interface PositionCashOutManager {
    busyKey: string;
    viewFor(row: PositionRow): PositionCashOutView;
    confirm(row: PositionRow): void;
    authorize(row: PositionRow, view: PositionCashOutView): void;
    reconcile(row: PositionRow, view: PositionCashOutView): void;
    alerts: React.ReactNode;
}

const positionCashOutRowKey = (row: PositionRow) => `${row.wallet.walletId}:${row.position.pubkey}`;
const cashOutPollStates = new Set<WormPositionCashOutProjection['state']>([
    'AWAITING_AUTHORIZATION',
    'QUEUED',
    'PREFLIGHTING',
    'CLOSING',
    'AWAITING_COMPLETION',
    'RECONCILIATION_REQUIRED'
]);
const cashOutExecutionPendingStates = new Set<WormPositionCashOutProjection['state']>(['QUEUED', 'PREFLIGHTING', 'CLOSING', 'AWAITING_COMPLETION']);
const terminalCashOutStates = new Set<WormPositionCashOutProjection['state']>(['COMPLETED', 'FAILED', 'EXPIRED']);

const cashOutReasonMessage = (reasonCode: string) => {
    switch (reasonCode) {
        case 'WALLET_EXECUTION_ACTIVE':
            return 'This wallet has an unfinished execution Run. Finish or terminate that Run before cashing out a position.';
        case 'WALLET_CASH_OUT_ACTIVE':
        case 'WALLET_POSITION_CASH_OUT_ACTIVE':
            return 'Another position in this wallet already has an unfinished Cash Out.';
        case 'WALLET_CASH_OUT_BATCH_ACTIVE':
        case 'WALLET_POSITION_CASH_OUT_BATCH_ACTIVE':
            return 'This wallet is locked by an unfinished batch Cash Out. Use the batch controls before starting another operation.';
        case 'POSITION_NOT_FOUND':
            return 'Worm no longer returns this exact position. Refresh positions before taking another action.';
        case 'POSITION_LIQUIDATED':
        case 'POSITION_NOT_CLOSABLE':
            return 'This position is no longer eligible for Cash Out.';
        default:
            return titleCase(reasonCode) || 'Cash Out is not available for this position.';
    }
};

const cashOutViewFromOperation = (operation: WormPositionCashOutOperation): PositionCashOutView => ({
    operationId: operation.id,
    state: operation.state,
    reasonCode: operation.reasonCode,
    allowedAction: operation.allowedActions.includes('CHECK_STATUS') ? 'CHECK_STATUS' : operation.allowedActions.includes('AUTHORIZE_CASH_OUT') ? 'AUTHORIZE_CASH_OUT' : 'NONE',
    batchId: '',
    batchState: '',
    batchItemState: '',
    batchLockReasonCode: '',
    revision: operation.revision,
    updatedAt: operation.updatedAt
});

const cashOutOperationMatchesRow = (operation: WormPositionCashOutOperation, row: PositionRow) =>
    operation.walletId === row.wallet.walletId &&
    operation.walletAddress === row.wallet.address &&
    operation.positionPubkey === row.position.pubkey &&
    operation.positionRequestPubkey === row.position.positionRequestPubkey &&
    operation.marketConditionId === row.position.market.conditionId &&
    operation.isYes === (row.position.side === 'YES') &&
    (row.position.createdAt <= 0 || operation.positionCreatedAt === row.position.createdAt);

const PositionCashOutButton = (props: {row: PositionRow; manager: PositionCashOutManager; compact?: boolean}) => {
    const view = props.manager.viewFor(props.row);
    const rowKey = positionCashOutRowKey(props.row);
    const busy = props.manager.busyKey === rowKey;
    const interactionBlocked = Boolean(props.manager.busyKey) && !busy;
    const buttonProps = props.compact ? {block: true as const} : {size: 'small' as const};
    if (view.allowedAction === 'CASH_OUT') {
        return (
            <Button
                {...buttonProps}
                danger={true}
                icon={<CloseCircleOutlined />}
                loading={busy}
                disabled={interactionBlocked}
                aria-label={`Cash out ${props.row.position.side} position in ${props.row.position.market.title}`}
                onClick={() => props.manager.confirm(props.row)}>
                Cash out
            </Button>
        );
    }
    if (view.allowedAction === 'AUTHORIZE_CASH_OUT') {
        return (
            <Button
                {...buttonProps}
                type='primary'
                danger={true}
                icon={<SafetyCertificateOutlined />}
                loading={busy}
                disabled={interactionBlocked}
                onClick={() => props.manager.authorize(props.row, view)}>
                Authorize cash out
            </Button>
        );
    }
    if (view.allowedAction === 'CHECK_STATUS') {
        return (
            <Button {...buttonProps} icon={<ReloadOutlined />} loading={busy} disabled={interactionBlocked} onClick={() => props.manager.reconcile(props.row, view)}>
                Check status
            </Button>
        );
    }
    if (view.batchId) {
        const batchLabel =
            view.batchItemState === ''
                ? 'Batch locked'
                : view.batchItemState === 'PENDING'
                  ? 'Queued'
                  : view.batchItemState === 'AWAITING_BALANCE'
                    ? 'Verifying balance…'
                    : view.batchItemState === 'COMPLETED'
                      ? 'Closed'
                      : view.batchItemState === 'NOT_EXECUTED'
                        ? 'Not executed'
                        : view.batchItemState === 'FAILED' || view.batchItemState === 'RECONCILIATION_REQUIRED'
                          ? 'Check batch'
                          : 'Closing…';
        const button = (
            <Button {...buttonProps} danger={view.batchItemState !== 'COMPLETED' && view.batchItemState !== 'NOT_EXECUTED'} loading={false} disabled={true}>
                {batchLabel}
            </Button>
        );
        return <Tooltip title={cashOutReasonMessage(view.reasonCode || 'WALLET_CASH_OUT_BATCH_ACTIVE')}>{button}</Tooltip>;
    }
    const active = cashOutPollStates.has(view.state) && view.reasonCode !== 'WALLET_CASH_OUT_ACTIVE' && view.reasonCode !== 'WALLET_POSITION_CASH_OUT_ACTIVE';
    const label = active ? (view.state === 'RECONCILIATION_REQUIRED' ? 'Checking…' : 'Closing…') : view.state === 'COMPLETED' ? 'Closed' : 'Cash out unavailable';
    const button = (
        <Button {...buttonProps} danger={active} loading={active || busy} disabled={true}>
            {label}
        </Button>
    );
    return view.reasonCode ? <Tooltip title={cashOutReasonMessage(view.reasonCode)}>{button}</Tooltip> : button;
};

const PositionCashOutManagement = (props: {
    rows: PositionRow[];
    activityFetchedAt: number;
    onRefreshAssets: () => void;
    batchManager?: PositionCashOutBatchManager;
    children: (manager: PositionCashOutManager) => React.ReactNode;
}) => {
    const ctx = React.useContext(Context);
    const authorization = useAuthorization();
    const location = useLocation();
    const lease = useSensitiveWriteLease();
    const [operations, setOperations] = React.useState<Map<string, WormPositionCashOutOperation>>(() => new Map());
    const [busyKey, setBusyKey] = React.useState('');
    const [pollError, setPollError] = React.useState('');
    const notifiedRef = React.useRef(new Set<string>());

    const runSensitive = React.useCallback(
        async <T,>(start: () => Promise<T> & {abort?: () => void}) => {
            const result = await lease.runTask(start);
            if (result.status === 'fulfilled') {
                return result.value;
            }
            if (result.status === 'rejected') {
                throw result.error;
            }
            throw new DOMException('The Cash Out request is no longer current.', 'AbortError');
        },
        [lease]
    );

    const publishOperation = React.useCallback(
        (operation: WormPositionCashOutOperation) => {
            const key = `${operation.walletId}:${operation.positionPubkey}`;
            setOperations(current => {
                const previous = current.get(key);
                if (previous && previous.id === operation.id && previous.revision > operation.revision) {
                    return current;
                }
                const next = new Map(current);
                next.set(key, operation);
                return next;
            });
            setPollError('');
            if (terminalCashOutStates.has(operation.state) && !notifiedRef.current.has(operation.id)) {
                notifiedRef.current.add(operation.id);
                forgetPositionCashOutID(operation.id);
                if (operation.state === 'COMPLETED') {
                    ctx.notifications.success('Position cashed out', 'Worm has confirmed that the exact position is closed.');
                } else {
                    ctx.notifications.error(
                        operation.state === 'EXPIRED' ? 'Cash Out authorization expired' : 'Cash Out was not completed',
                        operation.reasonCode ? cashOutReasonMessage(operation.reasonCode) : 'No additional Close request was sent.'
                    );
                }
                props.onRefreshAssets();
            }
        },
        [ctx.notifications, props.onRefreshAssets]
    );

    const viewFor = React.useCallback(
        (row: PositionRow): PositionCashOutView => {
            const projection = row.position.cashOut;
            const tracked = operations.get(positionCashOutRowKey(row));
            const current =
                !tracked || projection.batchId || (projection.operationId && (projection.operationId !== tracked.id || projection.revision >= tracked.revision))
                    ? projection
                    : cashOutViewFromOperation(tracked);
            if (!props.batchManager?.walletLocked(row.wallet.walletId)) {
                return current;
            }
            const itemState = current.batchItemState || props.batchManager.itemStateFor(row);
            return {
                ...current,
                reasonCode: current.reasonCode || current.batchLockReasonCode || 'WALLET_CASH_OUT_BATCH_ACTIVE',
                allowedAction: 'NONE',
                batchId: current.batchId || props.batchManager.batch?.id || '',
                batchState: current.batchState || props.batchManager.batch?.state || '',
                batchItemState: itemState
            };
        },
        [operations, props.batchManager]
    );

    const authorizeOperation = React.useCallback(
        async (row: PositionRow, operation: PositionCashOutView) => {
            if (!operation.operationId || operation.revision < 1 || operation.allowedAction !== 'AUTHORIZE_CASH_OUT') {
                throw new Error('Athena did not return a durable Cash Out operation. No Close request was sent.');
            }
            const command = {commandId: window.crypto.randomUUID(), expectedRevision: operation.revision};
            const identity = authorization.user.identity;
            rememberPositionCashOutID(operation.operationId);
            if (identity.provider === AccountIdentityProvider.Google) {
                const returnTo = `${location.pathname}${location.search}`;
                const authorizationURL = new URL(services.wormTrading.googlePositionCashOutAuthorizationURL(operation.operationId, command, returnTo), window.location.origin);
                if (authorizationURL.origin !== window.location.origin) {
                    throw new Error('Athena returned an invalid Cash Out authorization route.');
                }
                const form = document.createElement('form');
                form.method = 'POST';
                form.action = authorizationURL.toString();
                form.hidden = true;
                document.body.appendChild(form);
                form.submit();
                form.remove();
                return;
            }
            let authorized: WormPositionCashOutOperation;
            if (identity.provider === AccountIdentityProvider.SolanaWallet) {
                const provider = phantomProvider();
                if (!provider) {
                    throw new Error('Phantom is required to authorize this Cash Out.');
                }
                const connected = provider.publicKey ? {publicKey: provider.publicKey} : await provider.connect();
                if (connected.publicKey.toString() !== identity.solanaAddress) {
                    throw new Error('Phantom is connected to a different login address.');
                }
                const challenge = await runSensitive(() => services.wormTrading.createSolanaPositionCashOutAuthorizationChallenge(operation.operationId, command));
                const signed = await provider.signMessage(new TextEncoder().encode(challenge.message), 'utf8');
                authorized = await runSensitive(() => services.wormTrading.verifySolanaPositionCashOutAuthorization(operation.operationId, rawBase64URL(signed.signature)));
            } else if (identity.provider === AccountIdentityProvider.Development) {
                authorized = await runSensitive(() => services.wormTrading.authorizeDevelopmentPositionCashOut(operation.operationId, command));
            } else {
                throw new Error('This login identity cannot authorize a Cash Out.');
            }
            if (!cashOutOperationMatchesRow(authorized, row)) {
                throw new Error('Athena returned a Cash Out for a different position. Existing state was not replaced.');
            }
            publishOperation(authorized);
            props.onRefreshAssets();
        },
        [authorization.user.identity, location.pathname, location.search, props.onRefreshAssets, publishOperation, runSensitive]
    );

    const authorize = React.useCallback(
        async (row: PositionRow, operation: PositionCashOutView) => {
            const key = positionCashOutRowKey(row);
            if (busyKey) {
                return;
            }
            setBusyKey(key);
            try {
                await authorizeOperation(row, operation);
            } catch (reason) {
                if (!(reason instanceof DOMException && reason.name === 'AbortError')) {
                    ctx.notifications.error('Could not authorize Cash Out', requestErrorMessage(reason, 'No Close request was replayed.'));
                    try {
                        publishOperation(await runSensitive(() => services.wormTrading.getPositionCashOut(operation.operationId)));
                    } catch {
                        props.onRefreshAssets();
                    }
                }
            } finally {
                setBusyKey('');
            }
        },
        [authorizeOperation, busyKey, ctx.notifications, props.onRefreshAssets, publishOperation, runSensitive]
    );

    const createAndAuthorize = React.useCallback(
        async (row: PositionRow) => {
            const key = positionCashOutRowKey(row);
            if (busyKey) {
                return;
            }
            setBusyKey(key);
            try {
                const operation = await runSensitive(() =>
                    services.wormTrading.createPositionCashOut({
                        commandId: window.crypto.randomUUID(),
                        walletId: row.wallet.walletId,
                        positionPubkey: row.position.pubkey
                    })
                );
                if (!cashOutOperationMatchesRow(operation, row)) {
                    throw new Error('Athena returned a Cash Out for a different position. No authorization was sent.');
                }
                publishOperation(operation);
                if (terminalCashOutStates.has(operation.state)) {
                    return;
                }
                await authorizeOperation(row, cashOutViewFromOperation(operation));
            } catch (reason) {
                if (!(reason instanceof DOMException && reason.name === 'AbortError')) {
                    ctx.notifications.error('Could not prepare Cash Out', requestErrorMessage(reason, 'No Close request was replayed.'));
                    props.onRefreshAssets();
                }
            } finally {
                setBusyKey('');
            }
        },
        [authorizeOperation, busyKey, ctx.notifications, props.onRefreshAssets, publishOperation, runSensitive]
    );

    const confirm = React.useCallback(
        (row: PositionRow) => {
            ctx.modal.confirm({
                className: 'worm-confirm-modal',
                title: 'Cash out this Worm position?',
                content: (
                    <div className='worm-position-cash-out-confirmation'>
                        <dl>
                            <div>
                                <dt>Wallet</dt>
                                <dd>
                                    {row.wallet.remark || 'Solana wallet'}
                                    <br />
                                    <code>{row.wallet.address}</code>
                                </dd>
                            </div>
                            <div>
                                <dt>Market</dt>
                                <dd>{row.position.market.title || row.position.market.conditionId}</dd>
                            </div>
                            <div>
                                <dt>Position</dt>
                                <dd>
                                    {row.position.side} · {optionalValue(row.position.totalShares)} shares
                                    <br />
                                    <code>{row.position.pubkey}</code>
                                </dd>
                            </div>
                        </dl>
                        <Alert
                            type='warning'
                            showIcon={true}
                            title='Full-position market exit'
                            description='This closes the entire position at the available market price. The final execution price is not guaranteed, and partial Cash Out is not supported.'
                        />
                        <Typography.Paragraph type='secondary'>
                            You will confirm your identity next. Phantom signs only an identity message; it does not submit a transaction or charge a network fee.
                        </Typography.Paragraph>
                        <Typography.Paragraph type='secondary'>
                            Pending does not mean Closed. If the result becomes unknown, use Check status and do not submit another Cash Out.
                        </Typography.Paragraph>
                    </div>
                ),
                okText: 'Continue to authorization',
                cancelText: 'Keep position open',
                onOk: () => createAndAuthorize(row)
            });
        },
        [createAndAuthorize, ctx.modal]
    );

    const reconcileByKey = React.useCallback(
        async (key: string, operation: PositionCashOutView) => {
            if (!operation.operationId || operation.revision < 1 || busyKey) {
                return;
            }
            setBusyKey(key);
            try {
                const current = await runSensitive(() => services.wormTrading.getPositionCashOut(operation.operationId));
                publishOperation(current);
                if (current.state !== 'RECONCILIATION_REQUIRED' || !current.allowedActions.includes('CHECK_STATUS')) {
                    props.onRefreshAssets();
                    return;
                }
                const next = await runSensitive(() =>
                    services.wormTrading.reconcilePositionCashOut(current.id, {
                        commandId: window.crypto.randomUUID(),
                        expectedRevision: current.revision
                    })
                );
                publishOperation(next);
                props.onRefreshAssets();
            } catch (reason) {
                if (!(reason instanceof DOMException && reason.name === 'AbortError')) {
                    ctx.notifications.error('Could not check Cash Out status', requestErrorMessage(reason, 'Athena did not resend Close.'));
                    try {
                        publishOperation(await runSensitive(() => services.wormTrading.getPositionCashOut(operation.operationId)));
                    } catch {
                        props.onRefreshAssets();
                    }
                }
            } finally {
                setBusyKey('');
            }
        },
        [busyKey, ctx.notifications, props.onRefreshAssets, publishOperation, runSensitive]
    );

    const reconcile = React.useCallback((row: PositionRow, operation: PositionCashOutView) => reconcileByKey(positionCashOutRowKey(row), operation), [reconcileByKey]);

    React.useEffect(() => {
        const pendingIDs = rememberedPositionCashOutIDs();
        const url = new URL(window.location.href);
        const reason = url.searchParams.get(positionCashOutReasonQuery) || '';
        if (reason) {
            ctx.notifications.error('Could not authorize Cash Out', cashOutReasonMessage(reason));
            url.searchParams.delete(positionCashOutReasonQuery);
            window.history.replaceState(window.history.state, '', `${url.pathname}${url.search}${url.hash}`);
        }
        if (pendingIDs.length === 0) {
            return;
        }
        let active = true;
        void (async () => {
            for (const id of pendingIDs) {
                try {
                    const operation = await runSensitive(() => services.wormTrading.getPositionCashOut(id));
                    if (!active) {
                        return;
                    }
                    publishOperation(operation);
                    props.onRefreshAssets();
                } catch (error) {
                    if (active && !(error instanceof DOMException && error.name === 'AbortError')) {
                        const status = requestErrorDetails(error).status;
                        if (status === 403 || status === 404) {
                            forgetPositionCashOutID(id);
                        }
                        setPollError(requestErrorMessage(error, 'Cash Out status could not be restored after authorization.'));
                    }
                }
            }
        })();
        return () => {
            active = false;
        };
    }, [authorization.revision, authorization.user.accountId, ctx.notifications, props.onRefreshAssets, publishOperation, runSensitive]);

    const pollingKey = React.useMemo(() => {
        const ids = new Set<string>();
        const trackedIDs = new Set<string>();
        props.rows.forEach(row => {
            if (row.position.cashOut.operationId && cashOutPollStates.has(row.position.cashOut.state)) {
                ids.add(row.position.cashOut.operationId);
            }
        });
        operations.forEach(operation => {
            trackedIDs.add(operation.id);
            if (cashOutPollStates.has(operation.state)) {
                ids.add(operation.id);
            }
        });
        rememberedPositionCashOutIDs().forEach(id => {
            if (!trackedIDs.has(id)) {
                ids.add(id);
            }
        });
        return Array.from(ids).sort().join('|');
    }, [operations, props.rows]);

    React.useEffect(() => {
        if (!pollingKey) {
            return;
        }
        const pollingIDs = pollingKey.split('|');
        let active = true;
        let timer: number | undefined;
        const poll = async () => {
            for (const id of pollingIDs) {
                try {
                    const operation = await runSensitive(() => services.wormTrading.getPositionCashOut(id));
                    if (!active) {
                        return;
                    }
                    publishOperation(operation);
                } catch (reason) {
                    if (active && !(reason instanceof DOMException && reason.name === 'AbortError')) {
                        const status = requestErrorDetails(reason).status;
                        if (status === 403 || status === 404) {
                            forgetPositionCashOutID(id);
                        }
                        setPollError(requestErrorMessage(reason, 'Cash Out status could not be refreshed. No Close request was replayed.'));
                    }
                }
            }
            if (active) {
                timer = window.setTimeout(poll, positionCashOutPollIntervalMS);
            }
        };
        timer = window.setTimeout(poll, positionCashOutPollIntervalMS);
        return () => {
            active = false;
            if (timer !== undefined) {
                window.clearTimeout(timer);
            }
        };
    }, [pollingKey, publishOperation, runSensitive]);

    React.useEffect(() => {
        if (props.activityFetchedAt <= 0) {
            return;
        }
        const currentRows = new Map(props.rows.map(row => [positionCashOutRowKey(row), row]));
        setOperations(current => {
            let changed = false;
            const next = new Map(current);
            current.forEach((operation, key) => {
                const row = currentRows.get(key);
                const projection = row?.position.cashOut;
                if ((!row && operation.state === 'COMPLETED') || (projection?.operationId && projection.operationId !== operation.id)) {
                    next.delete(key);
                    changed = true;
                } else if (
                    projection &&
                    !projection.operationId &&
                    (operation.state === 'FAILED' || operation.state === 'EXPIRED') &&
                    props.activityFetchedAt > operation.updatedAt
                ) {
                    next.delete(key);
                    changed = true;
                }
            });
            return changed ? next : current;
        });
    }, [props.activityFetchedAt, props.rows]);

    const exactVisibleOperationIDs = new Set<string>();
    operations.forEach(operation => {
        if (props.rows.some(row => cashOutOperationMatchesRow(operation, row))) {
            exactVisibleOperationIDs.add(operation.id);
        }
    });
    const unknownRows = props.rows
        .map(row => ({row, view: viewFor(row)}))
        .filter(item => item.view.state === 'RECONCILIATION_REQUIRED' && item.view.operationId && item.view.allowedAction === 'CHECK_STATUS');
    const visibleUnknownIDs = new Set(unknownRows.map(item => item.view.operationId));
    const orphanUnknownOperations = Array.from(operations.values()).filter(operation => operation.state === 'RECONCILIATION_REQUIRED' && !visibleUnknownIDs.has(operation.id));
    const orphanPendingOperations = Array.from(operations.values()).filter(
        operation => cashOutExecutionPendingStates.has(operation.state) && !exactVisibleOperationIDs.has(operation.id)
    );
    const alerts = (
        <>
            {pollError && (
                <Alert
                    className='worm-position-cash-out-alert'
                    type='warning'
                    showIcon={true}
                    title='Cash Out status could not be refreshed'
                    description={`${pollError} Refreshing status never resends Close.`}
                />
            )}
            {orphanPendingOperations.map(operation => (
                <Alert
                    className='worm-position-cash-out-alert'
                    key={operation.id}
                    type='info'
                    showIcon={true}
                    title='Cash Out is still pending'
                    description={`Worm has not yet confirmed that wallet ${displayIdentity(operation.walletAddress)} position ${displayIdentity(operation.positionPubkey)} is closed. Its position row may disappear while Close is processing; Pending does not mean Closed, and Athena will not resend Close.`}
                />
            ))}
            {unknownRows.map(({row, view}) => (
                <Alert
                    className='worm-position-cash-out-alert'
                    key={view.operationId}
                    type='error'
                    showIcon={true}
                    title='Cash Out outcome requires attention'
                    description={`Athena cannot yet prove whether ${row.wallet.remark || displayIdentity(row.wallet.address)} position ${displayIdentity(row.position.pubkey)} is closed. Do not submit another Cash Out; Check status performs read-only reconciliation.`}
                    action={
                        view.allowedAction === 'CHECK_STATUS' ? (
                            <Button icon={<ReloadOutlined />} loading={busyKey === positionCashOutRowKey(row)} onClick={() => reconcile(row, view)}>
                                Check status
                            </Button>
                        ) : undefined
                    }
                />
            ))}
            {orphanUnknownOperations.map(operation => {
                const view = cashOutViewFromOperation(operation);
                const key = `${operation.walletId}:${operation.positionPubkey}`;
                return (
                    <Alert
                        className='worm-position-cash-out-alert'
                        key={operation.id}
                        type='error'
                        showIcon={true}
                        title='Cash Out outcome requires attention'
                        description={`Athena cannot yet prove whether wallet ${displayIdentity(operation.walletAddress)} position ${displayIdentity(operation.positionPubkey)} is closed. Do not submit another Cash Out; Check status performs read-only reconciliation.`}
                        action={
                            operation.allowedActions.includes('CHECK_STATUS') ? (
                                <Button icon={<ReloadOutlined />} loading={busyKey === key} onClick={() => void reconcileByKey(key, view)}>
                                    Check status
                                </Button>
                            ) : undefined
                        }
                    />
                );
            })}
        </>
    );
    const manager: PositionCashOutManager = {
        busyKey,
        viewFor,
        confirm,
        authorize: (row, view) => void authorize(row, view),
        reconcile: (row, view) => void reconcile(row, view),
        alerts
    };
    return <>{props.children(manager)}</>;
};

const terminalPositionCashOutBatchStates = new Set<WormPositionCashOutBatch['state']>(['TERMINATED', 'COMPLETED', 'FAILED', 'CANCELLED', 'EXPIRED']);

const positionCashOutBatchReasonMessage = (reasonCode: string) => {
    switch (reasonCode) {
        case 'BALANCE_NOT_UPDATED':
        case 'USDC_BALANCE_NOT_UPDATED':
            return 'The exact position is closed, but Athena has not observed a newer confirmed USDC balance that is strictly higher than the pre-Close baseline.';
        case 'BALANCE_UNAVAILABLE':
        case 'USDC_BALANCE_UNAVAILABLE':
            return 'Confirmed USDC balance evidence is unavailable. No later position was activated.';
        case 'POSITION_CHANGED':
        case 'POSITION_IDENTITY_CHANGED':
        case 'POSITION_NOT_FOUND':
            return 'A frozen position no longer matches the authoritative Worm snapshot. Terminate the remaining batch and build a new one.';
        case 'WALLET_EXECUTION_ACTIVE':
            return 'A selected wallet is used by an unfinished execution Run.';
        case 'WALLET_POSITION_CASH_OUT_ACTIVE':
            return 'A selected wallet already has an unfinished single-position Cash Out.';
        case 'WALLET_POSITION_CASH_OUT_BATCH_ACTIVE':
            return 'A selected wallet is already locked by another Cash Out batch.';
        default:
            return titleCase(reasonCode) || 'Athena stopped before activating another position.';
    }
};

const positionCashOutBatchStateColor = (state: WormPositionCashOutBatch['state']) => {
    if (state === 'COMPLETED') {
        return 'success';
    }
    if (state === 'FAILED' || state === 'RECONCILIATION_REQUIRED') {
        return 'error';
    }
    if (state === 'PAUSED' || state === 'PAUSE_REQUESTED' || state === 'AWAITING_AUTHORIZATION' || state === 'EXPIRED') {
        return 'warning';
    }
    if (state === 'CANCELLED' || state === 'TERMINATED') {
        return 'default';
    }
    return 'processing';
};

const positionCashOutBatchItemStateColor = (state: WormPositionCashOutBatchItemState) => {
    if (state === 'COMPLETED') {
        return 'success';
    }
    if (state === 'FAILED' || state === 'RECONCILIATION_REQUIRED') {
        return 'error';
    }
    if (state === 'AWAITING_BALANCE') {
        return 'warning';
    }
    if (state === 'NOT_EXECUTED') {
        return 'default';
    }
    return state === 'PENDING' ? 'default' : 'processing';
};

const formatUSDCAtomicAmount = (value: string) => {
    if (!/^-?\d+$/.test(value)) {
        return '-';
    }
    const negative = value.startsWith('-');
    const digits = (negative ? value.slice(1) : value).padStart(7, '0');
    const whole = digits.slice(0, -6).replace(/^0+(?=\d)/, '');
    const fraction = digits.slice(-6).replace(/0+$/, '');
    return `${negative ? '-' : ''}${whole}${fraction ? `.${fraction}` : ''} USDC`;
};

const PositionCashOutBatchEvidence = (props: {baseline?: WormPositionCashOutBatchBalanceEvidence; observed?: WormPositionCashOutBatchBalanceEvidence; delta: string}) => (
    <dl className='worm-position-cash-out-batch-evidence'>
        <div>
            <dt>Before Close</dt>
            <dd>
                {props.baseline ? formatUSDCAtomicAmount(props.baseline.atomicAmount) : 'Not captured'}
                <small>{props.baseline ? `Confirmed slot ${formatIntegerString(props.baseline.observedSlot)}` : 'The baseline is captured immediately before dispatch.'}</small>
            </dd>
        </div>
        <div>
            <dt>Latest observed</dt>
            <dd>
                {props.observed ? formatUSDCAtomicAmount(props.observed.atomicAmount) : 'Waiting for evidence'}
                <small>{props.observed ? `Confirmed slot ${formatIntegerString(props.observed.observedSlot)}` : 'No newer confirmed balance has been accepted.'}</small>
            </dd>
        </div>
        <div>
            <dt>Net increase</dt>
            <dd className={props.delta && !props.delta.startsWith('-') && props.delta !== '0' ? 'is-positive' : ''}>
                {props.delta ? formatUSDCAtomicAmount(props.delta) : '-'}
                <small>Must be strictly greater than zero.</small>
            </dd>
        </div>
    </dl>
);

const PositionCashOutBatchManagement = (props: {
    visibleWallets: WormTradingWalletBalanceItem[];
    onRefreshAssets: () => void;
    children: (manager: PositionCashOutBatchManager) => React.ReactNode;
}) => {
    const ctx = React.useContext(Context);
    const authorization = useAuthorization();
    const location = useLocation();
    const lease = useSensitiveWriteLease();
    const [selectedWalletIDs, setSelectedWalletIDs] = React.useState<Set<number>>(() => new Set());
    const [batch, setBatch] = React.useState<WormPositionCashOutBatch>();
    const [items, setItems] = React.useState<WormPositionCashOutBatchItem[]>([]);
    const [itemsTotal, setItemsTotal] = React.useState(0);
    const [itemsPage, setItemsPage] = React.useState(1);
    const [busyAction, setBusyAction] = React.useState('');
    const [statusError, setStatusError] = React.useState('');
    const [restoring, setRestoring] = React.useState(true);
    const [restoreNonce, setRestoreNonce] = React.useState(0);
    const [pendingReviewID, setPendingReviewID] = React.useState('');
    const reviewModalIDRef = React.useRef('');
    const latestBatchRef = React.useRef<WormPositionCashOutBatch>();
    const notifiedRef = React.useRef(new Set<string>());
    const completedCountRef = React.useRef(new Map<string, number>());

    const runSensitive = React.useCallback(
        async <T,>(start: () => Promise<T> & {abort?: () => void}) => {
            const result = await lease.runTask(start);
            if (result.status === 'fulfilled') {
                return result.value;
            }
            if (result.status === 'rejected') {
                throw result.error;
            }
            throw new DOMException('The Cash Out batch request is no longer current.', 'AbortError');
        },
        [lease]
    );

    const publishBatch = React.useCallback(
        (next: WormPositionCashOutBatch) => {
            const previous = latestBatchRef.current;
            if (previous?.id === next.id && previous.revision > next.revision) {
                return;
            }
            latestBatchRef.current = next;
            setBatch(next);
            setStatusError('');
            if (terminalPositionCashOutBatchStates.has(next.state)) {
                forgetPositionCashOutBatchID(next.id);
            } else {
                rememberPositionCashOutBatchID(next.id);
            }
            const previousCompletedCount = completedCountRef.current.get(next.id) || 0;
            completedCountRef.current.set(next.id, next.completedCount);
            if (next.completedCount > previousCompletedCount) {
                props.onRefreshAssets();
            }
            if (terminalPositionCashOutBatchStates.has(next.state) && !notifiedRef.current.has(`${next.id}:${next.state}`)) {
                notifiedRef.current.add(`${next.id}:${next.state}`);
                if (next.state === 'COMPLETED') {
                    ctx.notifications.success('Cash Out batch completed', `All ${next.completedCount} frozen positions passed the confirmed USDC balance gate.`);
                } else if (next.state === 'FAILED') {
                    ctx.notifications.error('Cash Out batch failed safely', positionCashOutBatchReasonMessage(next.reasonCode));
                }
                props.onRefreshAssets();
            }
        },
        [ctx.notifications, props.onRefreshAssets]
    );

    const loadItems = React.useCallback(async (batchID: string, page: number) => {
        const result = await services.wormTrading.listPositionCashOutBatchItems(batchID, page, positionCashOutBatchItemPageSize);
        if (latestBatchRef.current?.id !== batchID) {
            return;
        }
        setItems(result.items);
        setItemsTotal(result.total);
    }, []);

    const refreshBatch = React.useCallback(
        async (batchID: string, page = itemsPage) => {
            const next = await services.wormTrading.getPositionCashOutBatch(batchID);
            publishBatch(next);
            if (next.positionCount > 0) {
                await loadItems(next.id, page);
            } else {
                setItems([]);
                setItemsTotal(0);
            }
            return next;
        },
        [itemsPage, loadItems, publishBatch]
    );

    const authorizeBatch = React.useCallback(
        async (current: WormPositionCashOutBatch) => {
            if (!current.allowedActions.includes('AUTHORIZE_BATCH')) {
                throw new Error('This Cash Out batch is not awaiting authorization.');
            }
            const command = {commandId: window.crypto.randomUUID(), expectedRevision: current.revision};
            const identity = authorization.user.identity;
            rememberPositionCashOutBatchID(current.id);
            if (identity.provider === AccountIdentityProvider.Google) {
                const returnURL = new URL(window.location.href);
                returnURL.searchParams.delete(positionCashOutBatchReasonQuery);
                const returnTo = `${returnURL.pathname}${returnURL.search}${returnURL.hash}`;
                const authorizationURL = new URL(services.wormTrading.googlePositionCashOutBatchAuthorizationURL(current.id, command, returnTo), window.location.origin);
                if (authorizationURL.origin !== window.location.origin) {
                    throw new Error('Athena returned an invalid Cash Out batch authorization route.');
                }
                const form = document.createElement('form');
                form.method = 'POST';
                form.action = authorizationURL.toString();
                form.hidden = true;
                document.body.appendChild(form);
                form.submit();
                form.remove();
                return;
            }
            let authorized: WormPositionCashOutBatch;
            if (identity.provider === AccountIdentityProvider.SolanaWallet) {
                const provider = phantomProvider();
                if (!provider) {
                    throw new Error('Phantom is required to authorize this Cash Out batch.');
                }
                const connected = provider.publicKey ? {publicKey: provider.publicKey} : await provider.connect();
                if (connected.publicKey.toString() !== identity.solanaAddress) {
                    throw new Error('Phantom is connected to a different login address.');
                }
                const challenge = await runSensitive(() => services.wormTrading.createSolanaPositionCashOutBatchAuthorizationChallenge(current.id, command));
                const signed = await provider.signMessage(new TextEncoder().encode(challenge.message), 'utf8');
                authorized = await runSensitive(() => services.wormTrading.verifySolanaPositionCashOutBatchAuthorization(current.id, rawBase64URL(signed.signature)));
            } else if (identity.provider === AccountIdentityProvider.Development) {
                authorized = await runSensitive(() => services.wormTrading.authorizeDevelopmentPositionCashOutBatch(current.id, command));
            } else {
                throw new Error('This login identity cannot authorize a Cash Out batch.');
            }
            publishBatch(authorized);
        },
        [authorization.user.identity, publishBatch, runSensitive]
    );

    const applyCommand = React.useCallback(
        async (
            current: WormPositionCashOutBatch,
            action: 'cancel' | 'pause' | 'continue' | 'terminate' | 'check-status',
            requiredAction: WormPositionCashOutBatchAllowedAction
        ) => {
            if (busyAction || !current.allowedActions.includes(requiredAction)) {
                return;
            }
            setBusyAction(action);
            try {
                const next = await runSensitive(() =>
                    services.wormTrading.commandPositionCashOutBatch(current.id, action, {
                        commandId: window.crypto.randomUUID(),
                        expectedRevision: current.revision
                    })
                );
                publishBatch(next);
                if (next.positionCount > 0) {
                    await loadItems(next.id, itemsPage);
                }
            } catch (reason) {
                if (!(reason instanceof DOMException && reason.name === 'AbortError')) {
                    ctx.notifications.error('Could not update Cash Out batch', requestErrorMessage(reason, 'No Worm Close request was replayed.'));
                    try {
                        await refreshBatch(current.id);
                    } catch {
                        setStatusError('Cash Out batch status could not be refreshed after the command failed.');
                    }
                }
            } finally {
                setBusyAction('');
            }
        },
        [busyAction, ctx.notifications, itemsPage, loadItems, publishBatch, refreshBatch, runSensitive]
    );

    const cancelBatch = React.useCallback((current: WormPositionCashOutBatch) => applyCommand(current, 'cancel', 'CANCEL'), [applyCommand]);

    const openReview = React.useCallback(
        (current: WormPositionCashOutBatch) => {
            if (reviewModalIDRef.current || !current.allowedActions.includes('AUTHORIZE_BATCH')) {
                return;
            }
            reviewModalIDRef.current = current.id;
            const closeReview = () => {
                reviewModalIDRef.current = '';
            };
            const reauthorizing = current.state === 'PAUSED';
            ctx.modal.confirm({
                className: 'worm-confirm-modal',
                width: 720,
                title: reauthorizing ? 'Reauthorize paused Cash Out batch?' : 'Authorize serial Cash Out batch?',
                content: (
                    <div className='worm-position-cash-out-batch-confirmation'>
                        <div className='worm-position-cash-out-batch-confirmation__summary'>
                            <strong>
                                {current.walletCount} {current.walletCount === 1 ? 'wallet' : 'wallets'} · {current.positionCount}{' '}
                                {current.positionCount === 1 ? 'position' : 'positions'}
                            </strong>
                            <span>Wallet order is fixed below. Each wallet closes positions from newest to oldest.</span>
                        </div>
                        <ol className='worm-position-cash-out-batch-confirmation__wallets'>
                            {current.wallets.map(wallet => (
                                <li key={wallet.walletId}>
                                    <span>
                                        {wallet.ordinal}. {wallet.remark || 'Solana wallet'}
                                        <br />
                                        <code>{wallet.address}</code>
                                    </span>
                                    <strong>{wallet.positionCount} positions</strong>
                                </li>
                            ))}
                        </ol>
                        <Alert
                            type='warning'
                            showIcon={true}
                            title='Every position is a full market exit'
                            description='Final prices are not guaranteed and partial Cash Out is not supported. Athena sends at most one Close for each frozen position.'
                        />
                        <Alert
                            type='warning'
                            showIcon={true}
                            title='Strict confirmed USDC gate after every Close'
                            description='The next position starts only after confirmed USDC is newer and strictly higher than the pre-Close baseline. After two minutes without a net increase, the whole batch pauses. Check status never resumes it; you must click Continue after credit is observed.'
                        />
                        <Typography.Paragraph type='secondary'>
                            Worm does not provide a payout transaction ID. An unrelated incoming transfer can create a false positive, while an outgoing transfer can hide the
                            payout and create a false negative.
                        </Typography.Paragraph>
                        <Typography.Paragraph type='secondary'>
                            Pause and Terminate do not cancel a Close that was already dispatched. You will confirm your identity once for this exact frozen batch.
                        </Typography.Paragraph>
                    </div>
                ),
                okText: 'Continue to one-time authorization',
                cancelText: reauthorizing ? 'Keep paused' : 'Cancel batch',
                onOk: async () => {
                    try {
                        await authorizeBatch(current);
                    } catch (reason) {
                        if (!(reason instanceof DOMException && reason.name === 'AbortError')) {
                            ctx.notifications.error('Could not authorize Cash Out batch', requestErrorMessage(reason, 'No Close request was sent.'));
                            try {
                                await refreshBatch(current.id);
                            } catch {
                                setStatusError('Cash Out batch status could not be refreshed after authorization failed.');
                            }
                        }
                        throw reason;
                    } finally {
                        closeReview();
                    }
                },
                onCancel: async () => {
                    closeReview();
                    if (!reauthorizing) {
                        await cancelBatch(current);
                    }
                }
            });
        },
        [authorizeBatch, cancelBatch, ctx.modal, ctx.notifications, refreshBatch]
    );

    const createBatch = React.useCallback(async () => {
        if (busyAction || restoring || (!batch && statusError !== '') || selectedWalletIDs.size < 1 || selectedWalletIDs.size > maximumSelectedPositionCashOutBatchWallets) {
            return;
        }
        setBusyAction('create');
        try {
            const created = await runSensitive(() =>
                services.wormTrading.createPositionCashOutBatch({commandId: window.crypto.randomUUID(), walletIds: Array.from(selectedWalletIDs)})
            );
            setSelectedWalletIDs(new Set());
            setItemsPage(1);
            setItems([]);
            setItemsTotal(0);
            publishBatch(created);
            setPendingReviewID(created.id);
        } catch (reason) {
            if (!(reason instanceof DOMException && reason.name === 'AbortError')) {
                ctx.notifications.error('Could not build Cash Out batch', requestErrorMessage(reason, 'No Close request was sent.'));
            }
        } finally {
            setBusyAction('');
        }
    }, [batch, busyAction, ctx.notifications, publishBatch, restoring, runSensitive, selectedWalletIDs, statusError]);

    React.useEffect(() => {
        setSelectedWalletIDs(new Set());
        setBatch(undefined);
        setItems([]);
        setItemsTotal(0);
        setItemsPage(1);
        setStatusError('');
        setRestoring(true);
        setPendingReviewID('');
        latestBatchRef.current = undefined;
    }, [authorization.revision, authorization.user.accountId, location.pathname]);

    React.useEffect(() => {
        const url = new URL(window.location.href);
        const reason = url.searchParams.get(positionCashOutBatchReasonQuery) || '';
        if (reason) {
            ctx.notifications.error('Could not authorize Cash Out batch', positionCashOutBatchReasonMessage(reason));
            url.searchParams.delete(positionCashOutBatchReasonQuery);
            window.history.replaceState(window.history.state, '', `${url.pathname}${url.search}${url.hash}`);
        }
        let active = true;
        const pendingID = rememberedPositionCashOutBatchID();
        void (async () => {
            try {
                let restored: WormPositionCashOutBatch | undefined;
                let rememberedTerminal: WormPositionCashOutBatch | undefined;
                if (pendingID) {
                    try {
                        restored = await services.wormTrading.getPositionCashOutBatch(pendingID);
                    } catch (error) {
                        const status = requestErrorDetails(error).status;
                        if (status === 403 || status === 404) {
                            forgetPositionCashOutBatchID(pendingID);
                        } else {
                            throw error;
                        }
                    }
                }
                if (restored && terminalPositionCashOutBatchStates.has(restored.state)) {
                    rememberedTerminal = restored;
                    restored = undefined;
                }
                restored = restored || (await services.wormTrading.getActivePositionCashOutBatch());
                if (rememberedTerminal) {
                    // Keep the remembered terminal ID durable until the active
                    // lookup has also succeeded. A transient /active failure
                    // can then be retried without losing the redirect result.
                    forgetPositionCashOutBatchID(rememberedTerminal.id);
                }
                restored = restored || rememberedTerminal;
                if (!active || !restored) {
                    return;
                }
                publishBatch(restored);
                if (restored.positionCount > 0) {
                    await loadItems(restored.id, 1);
                }
            } catch (error) {
                if (active && !(error instanceof DOMException && error.name === 'AbortError')) {
                    setStatusError(requestErrorMessage(error, 'Active Cash Out batch could not be restored.'));
                }
            } finally {
                if (active) {
                    setRestoring(false);
                }
            }
        })();
        return () => {
            active = false;
        };
    }, [authorization.revision, authorization.user.accountId, ctx.notifications, loadItems, publishBatch, restoreNonce]);

    React.useEffect(() => {
        if (!batch || terminalPositionCashOutBatchStates.has(batch.state)) {
            return;
        }
        let active = true;
        let timer: number | undefined;
        const poll = async () => {
            try {
                const next = await services.wormTrading.getPositionCashOutBatch(batch.id);
                if (!active) {
                    return;
                }
                publishBatch(next);
                if (next.positionCount > 0) {
                    await loadItems(next.id, itemsPage);
                }
            } catch (error) {
                if (active && !(error instanceof DOMException && error.name === 'AbortError')) {
                    setStatusError(requestErrorMessage(error, 'Cash Out batch status could not be refreshed. No Close request was replayed.'));
                }
            }
            if (active) {
                timer = window.setTimeout(poll, positionCashOutPollIntervalMS);
            }
        };
        timer = window.setTimeout(poll, positionCashOutPollIntervalMS);
        return () => {
            active = false;
            if (timer !== undefined) {
                window.clearTimeout(timer);
            }
        };
    }, [batch?.id, batch?.state, itemsPage, loadItems, publishBatch]);

    React.useEffect(() => {
        if (!batch || batch.id !== pendingReviewID) {
            return;
        }
        if (batch.state === 'AWAITING_AUTHORIZATION' && batch.allowedActions.includes('AUTHORIZE_BATCH')) {
            setPendingReviewID('');
            openReview(batch);
        } else if (terminalPositionCashOutBatchStates.has(batch.state)) {
            setPendingReviewID('');
        }
    }, [batch, openReview, pendingReviewID]);

    const toggleWallet = React.useCallback(
        (walletID: number, selected: boolean) => {
            if (restoring || (!batch && statusError !== '') || !props.visibleWallets.some(item => item.wallet.walletId === walletID)) {
                return;
            }
            setSelectedWalletIDs(current => {
                const next = new Set(current);
                if (selected) {
                    if (next.size >= maximumSelectedPositionCashOutBatchWallets) {
                        return current;
                    }
                    next.add(walletID);
                } else {
                    next.delete(walletID);
                }
                return next;
            });
        },
        [batch, props.visibleWallets, restoring, statusError]
    );

    const activeBatch = batch && !terminalPositionCashOutBatchStates.has(batch.state) ? batch : undefined;
    const batchWalletIDs = new Set(activeBatch?.wallets.map(wallet => wallet.walletId) || []);
    const itemStateFor = React.useCallback(
        (row: PositionRow): WormPositionCashOutBatchItemState | '' => {
            const current = batch?.currentItem;
            if (current?.walletId === row.wallet.walletId && current.positionPubkey === row.position.pubkey) {
                return current.state;
            }
            return items.find(item => item.walletId === row.wallet.walletId && item.positionPubkey === row.position.pubkey)?.state || '';
        },
        [batch?.currentItem, items]
    );

    const selectionSummary = selectedWalletIDs.size > 0 && !activeBatch && (
        <div className='worm-position-cash-out-batch-selection' role='region' aria-label='Cash Out batch wallet selection'>
            <div>
                <strong>
                    {selectedWalletIDs.size}/{maximumSelectedPositionCashOutBatchWallets} wallets selected
                </strong>
                <span>
                    The backend will scan and freeze every Open Position in these wallets, including positions not visible on this page.
                    {selectedWalletIDs.size >= maximumSelectedPositionCashOutBatchWallets ? ' The Worm Trading wallet limit is reached.' : ''}
                </span>
            </div>
            <Space wrap={true}>
                <Button disabled={Boolean(busyAction)} onClick={() => setSelectedWalletIDs(new Set())}>
                    Clear
                </Button>
                <Button
                    type='primary'
                    danger={true}
                    icon={<CloseCircleOutlined />}
                    loading={busyAction === 'create'}
                    disabled={Boolean(busyAction)}
                    onClick={() => void createBatch()}>
                    Review &amp; cash out wallets
                </Button>
            </Space>
        </div>
    );

    const currentItem = batch?.currentItem;
    const completedPercent = batch && batch.positionCount > 0 ? Math.round((batch.completedCount / batch.positionCount) * 100) : 0;
    const liveMessage = batch
        ? `${titleCase(batch.state)}. ${batch.completedCount} of ${batch.positionCount} positions completed.${
              currentItem ? ` Current position ${currentItem.ordinal} in wallet ${currentItem.walletOrdinal}: ${titleCase(currentItem.state)}.` : ''
          }`
        : '';
    const panel = batch && (
        <Card className='worm-position-cash-out-batch-panel' title='Serial Cash Out batch'>
            <div className='worm-position-cash-out-batch-panel__heading'>
                <div>
                    <Space size={6} wrap={true}>
                        <Tag color={positionCashOutBatchStateColor(batch.state)}>{titleCase(batch.state)}</Tag>
                        <Typography.Text strong={true}>
                            {batch.completedCount}/{batch.positionCount} positions completed
                        </Typography.Text>
                        {batch.notExecutedCount > 0 && <Tag>{batch.notExecutedCount} not executed</Tag>}
                    </Space>
                    <Typography.Text type='secondary'>{batch.walletCount} wallets · Strictly serial · Confirmed USDC balance gate after every Close</Typography.Text>
                </div>
                <Typography.Text type='secondary'>Updated {formatBeijingUnixSeconds(batch.updatedAt)}</Typography.Text>
            </div>
            <Progress
                aria-label='Serial Cash Out completion'
                percent={completedPercent}
                status={
                    batch.state === 'FAILED' || batch.state === 'RECONCILIATION_REQUIRED'
                        ? 'exception'
                        : batch.state === 'COMPLETED'
                          ? 'success'
                          : terminalPositionCashOutBatchStates.has(batch.state)
                            ? 'normal'
                            : 'active'
                }
            />
            <div className='worm-position-cash-out-batch-live-region' role='status' aria-live='polite' aria-atomic='true'>
                {liveMessage}
            </div>
            {statusError && (
                <Alert type='warning' showIcon={true} title='Batch status could not be refreshed' description={`${statusError} Refreshing and Check status never resend Close.`} />
            )}
            {(batch.state === 'PAUSED' ||
                batch.state === 'RECONCILIATION_REQUIRED' ||
                batch.state === 'FAILED' ||
                (batch.state === 'TERMINATE_REQUESTED' && batch.reasonCode !== '')) && (
                <Alert
                    className='worm-position-cash-out-batch-panel__alert'
                    type={batch.state === 'PAUSED' ? 'warning' : 'error'}
                    showIcon={true}
                    title={
                        batch.state === 'PAUSED'
                            ? 'Later Cash Outs are paused'
                            : batch.state === 'FAILED'
                              ? 'Batch stopped safely'
                              : batch.state === 'TERMINATE_REQUESTED'
                                ? 'Termination is waiting for safe reconciliation'
                                : 'Close outcome requires read-only reconciliation'
                    }
                    description={`${positionCashOutBatchReasonMessage(batch.reasonCode)} Pending or unknown does not mean Closed. Do not create another Cash Out for a locked wallet.`}
                />
            )}
            {currentItem ? (
                <section className='worm-position-cash-out-batch-current' aria-labelledby='worm-position-cash-out-batch-current-heading'>
                    <div className='worm-position-cash-out-batch-current__heading'>
                        <div>
                            <Typography.Title id='worm-position-cash-out-batch-current-heading' level={4}>
                                Current target
                            </Typography.Title>
                            <Typography.Text type='secondary'>
                                Wallet {currentItem.walletOrdinal}/{batch.walletCount} · Position {currentItem.positionOrdinal} in wallet · Global {currentItem.ordinal}/
                                {batch.positionCount}
                            </Typography.Text>
                        </div>
                        <Tag color={positionCashOutBatchItemStateColor(currentItem.state)}>{titleCase(currentItem.state)}</Tag>
                    </div>
                    <div className='worm-position-cash-out-batch-current__target'>
                        <div>
                            <small>Wallet</small>
                            <strong>{currentItem.walletRemark || displayIdentity(currentItem.walletAddress)}</strong>
                            <code>{displayIdentity(currentItem.walletAddress)}</code>
                        </div>
                        <div>
                            <small>Market position</small>
                            <strong>
                                {currentItem.isYes ? 'YES' : 'NO'} · {currentItem.shares} shares
                            </strong>
                            <span>{currentItem.marketTitle || displayIdentity(currentItem.marketConditionId)}</span>
                        </div>
                        <div>
                            <small>Frozen position</small>
                            <code title={currentItem.positionPubkey}>{displayIdentity(currentItem.positionPubkey)}</code>
                            <span>{formatBeijingUnixSeconds(currentItem.positionCreatedAt)}</span>
                        </div>
                    </div>
                    <PositionCashOutBatchEvidence baseline={currentItem.baseline} observed={currentItem.observed} delta={currentItem.deltaAtomicAmount} />
                </section>
            ) : (
                <Alert
                    className='worm-position-cash-out-batch-panel__alert'
                    type={batch.state === 'COMPLETED' ? 'success' : 'info'}
                    showIcon={true}
                    title={
                        batch.state === 'BUILDING'
                            ? 'Building authoritative position snapshot'
                            : terminalPositionCashOutBatchStates.has(batch.state)
                              ? titleCase(batch.state)
                              : 'Waiting for the next safe boundary'
                    }
                    description={
                        batch.state === 'BUILDING' ? 'Athena is fully paging every selected wallet. Authorization remains unavailable until all targets are frozen.' : undefined
                    }
                />
            )}
            {items.length > 0 && (
                <section className='worm-position-cash-out-batch-items' aria-labelledby='worm-position-cash-out-batch-items-heading'>
                    <div className='worm-position-cash-out-batch-items__heading'>
                        <Typography.Title id='worm-position-cash-out-batch-items-heading' level={4}>
                            Frozen execution order
                        </Typography.Title>
                        <Typography.Text type='secondary'>Newest position first within each wallet</Typography.Text>
                    </div>
                    <ol start={(itemsPage - 1) * positionCashOutBatchItemPageSize + 1}>
                        {items.map(item => (
                            <li key={item.id} className={item.id === currentItem?.id ? 'is-current' : ''}>
                                <span className='worm-position-cash-out-batch-items__ordinal'>{item.ordinal}</span>
                                <div>
                                    <strong>{item.walletRemark || displayIdentity(item.walletAddress)}</strong>
                                    <span>
                                        {item.isYes ? 'YES' : 'NO'} · {item.shares} shares · {item.marketTitle || displayIdentity(item.marketConditionId)}
                                    </span>
                                </div>
                                <Tag color={positionCashOutBatchItemStateColor(item.state)}>{titleCase(item.state)}</Tag>
                            </li>
                        ))}
                    </ol>
                    {itemsTotal > positionCashOutBatchItemPageSize && (
                        <Pagination
                            size='small'
                            responsive={true}
                            current={itemsPage}
                            pageSize={positionCashOutBatchItemPageSize}
                            total={itemsTotal}
                            showSizeChanger={false}
                            onChange={page => {
                                setItemsPage(page);
                                void loadItems(batch.id, page).catch(reason => setStatusError(requestErrorMessage(reason, 'Batch items could not be loaded.')));
                            }}
                        />
                    )}
                </section>
            )}
            {batch.allowedActions.length > 0 && (
                <div className='worm-position-cash-out-batch-controls' aria-label='Cash Out batch controls'>
                    <Space wrap={true}>
                        {batch.allowedActions.includes('AUTHORIZE_BATCH') && (
                            <Button type='primary' danger={true} icon={<SafetyCertificateOutlined />} disabled={Boolean(busyAction)} onClick={() => openReview(batch)}>
                                Review &amp; authorize
                            </Button>
                        )}
                        {batch.allowedActions.includes('CANCEL') && (
                            <Button disabled={Boolean(busyAction)} loading={busyAction === 'cancel'} onClick={() => void cancelBatch(batch)}>
                                Cancel batch
                            </Button>
                        )}
                        {batch.allowedActions.includes('PAUSE') && (
                            <Button
                                icon={<PauseCircleOutlined />}
                                disabled={Boolean(busyAction)}
                                loading={busyAction === 'pause'}
                                onClick={() => void applyCommand(batch, 'pause', 'PAUSE')}>
                                Pause after current
                            </Button>
                        )}
                        {batch.allowedActions.includes('CONTINUE') && (
                            <Button
                                type='primary'
                                icon={<PlayCircleOutlined />}
                                disabled={Boolean(busyAction)}
                                loading={busyAction === 'continue'}
                                onClick={() => void applyCommand(batch, 'continue', 'CONTINUE')}>
                                Continue
                            </Button>
                        )}
                        {batch.allowedActions.includes('CHECK_STATUS') && (
                            <Button
                                icon={<ReloadOutlined />}
                                disabled={Boolean(busyAction)}
                                loading={busyAction === 'check-status'}
                                onClick={() => void applyCommand(batch, 'check-status', 'CHECK_STATUS')}>
                                Check status
                            </Button>
                        )}
                        {batch.allowedActions.includes('TERMINATE') && (
                            <Button
                                danger={true}
                                icon={<StopOutlined />}
                                disabled={Boolean(busyAction)}
                                loading={busyAction === 'terminate'}
                                onClick={() =>
                                    ctx.modal.confirm({
                                        className: 'worm-confirm-modal',
                                        title: 'Terminate remaining Cash Outs?',
                                        content:
                                            'A Close already dispatched for the current position cannot be cancelled. Athena will finish its safe reconciliation and mark every later frozen position Not executed.',
                                        okText: 'Terminate remaining',
                                        okButtonProps: {danger: true},
                                        onOk: () => applyCommand(batch, 'terminate', 'TERMINATE')
                                    })
                                }>
                                Terminate remaining
                            </Button>
                        )}
                    </Space>
                </div>
            )}
        </Card>
    );

    const recoveryPanel = !batch ? (
        restoring ? (
            <Alert
                type='info'
                showIcon={true}
                title='Checking for an active Cash Out batch'
                description='Wallet selection remains locked until authoritative batch status is known.'
            />
        ) : statusError ? (
            <Alert
                type='error'
                showIcon={true}
                title='Active Cash Out batch status is unavailable'
                description={`${statusError} Wallet selection is disabled to prevent a conflicting batch.`}
                action={
                    <Button
                        size='small'
                        icon={<ReloadOutlined />}
                        onClick={() => {
                            setStatusError('');
                            setRestoring(true);
                            setRestoreNonce(value => value + 1);
                        }}>
                        Retry
                    </Button>
                }
            />
        ) : undefined
    ) : undefined;
    const manager: PositionCashOutBatchManager = {
        batch,
        selectedWalletIDs,
        toggleWallet,
        clearSelection: () => setSelectedWalletIDs(new Set()),
        selectionDisabled: walletID =>
            restoring ||
            (!batch && statusError !== '') ||
            Boolean(activeBatch) ||
            (!selectedWalletIDs.has(walletID) && selectedWalletIDs.size >= maximumSelectedPositionCashOutBatchWallets),
        walletLocked: walletID => batchWalletIDs.has(walletID),
        itemStateFor,
        selectionSummary,
        panel: (
            <>
                {recoveryPanel}
                {panel}
            </>
        )
    };
    return <>{props.children(manager)}</>;
};

export const WormTradingPage = () => {
    const ctx = React.useContext(Context);
    const authorization = useAuthorization();
    const {page, pageSize, setPage} = usePagedParams(wormTradingPageSize, wormTradingPageSizes);
    const currentPage = Number.isSafeInteger(page) && page > 0 ? page : 1;
    const accountID = JSON.stringify([authorization.user.accountId, authorization.user.iss]);
    const [activityTab, setActivityTab] = React.useState('positions');
    const navigate = useNavigate();
    const canManageConnections = authorization.canWrite(AccountDataModule.WormTrading);
    const runtime = useCachedAsyncData(`worm-trading:status:${accountID}`, () => services.wormTrading.getStatus(), {
        staleTimeMs: 0,
        module: AccountDataModule.WormTrading
    });
    const balances = useCachedAsyncData(
        `worm-trading:wallet-balances:${accountID}:${currentPage}:${pageSize}`,
        () => services.wormTrading.listWalletBalances(currentPage, pageSize),
        {staleTimeMs: 0, module: AccountDataModule.WormTrading}
    );
    const activity = useCachedAsyncData(
        `worm-trading:wallet-activity:${accountID}:${currentPage}:${pageSize}`,
        () => services.wormTrading.listWalletActivity(currentPage, pageSize),
        {staleTimeMs: 0, module: AccountDataModule.WormTrading}
    );

    React.useEffect(() => {
        const total = Math.max(balances.data?.total || 0, activity.data?.total || 0);
        if (!balances.data && !activity.data) {
            return;
        }
        const lastPage = Math.max(1, Math.ceil(total / pageSize));
        if (currentPage > lastPage) {
            setPage(lastPage, pageSize);
        }
    }, [activity.data, balances.data, currentPage, pageSize, setPage]);

    React.useEffect(() => {
        if (!canManageConnections) {
            window.sessionStorage.removeItem(pendingConnectionActionKey);
            window.sessionStorage.removeItem(pendingPositionCashOutKey);
        }
    }, [accountID, authorization.revision, canManageConnections]);

    const copyAddress = React.useCallback(
        async (wallet: WormTradingWalletSummary) => {
            try {
                await navigator.clipboard.writeText(wallet.address);
                ctx.notifications.success('Address copied');
            } catch {
                ctx.notifications.error('Could not copy address', 'Select the address and copy it manually.');
            }
        },
        [ctx.notifications]
    );

    const refresh = React.useCallback(() => {
        runtime.reload();
        balances.reload();
        activity.reload();
    }, [activity, balances, runtime]);
    const refreshAssets = React.useCallback(() => {
        balances.reload();
        activity.reload();
    }, [activity.reload, balances.reload]);
    const reloadConnections = React.useCallback(() => {
        runtime.reload();
        balances.reload();
        activity.reload();
    }, [activity, balances, runtime]);
    const loading = runtime.loading || runtime.refreshing || balances.loading || balances.refreshing || activity.loading || activity.refreshing;
    const balanceItems = balances.data?.items || [];
    const activityItems = activity.data?.items || [];
    const queriedActivityItems = activityItems.filter(item => connectionWasQueried(item.connection.state));
    const activityByWallet = new Map(activityItems.map(item => [item.wallet.walletId, item]));
    const positionRows: PositionRow[] = activityItems
        .flatMap(item => item.openPositions.map(position => ({wallet: item.wallet, position})))
        .sort((left, right) => right.position.createdAt - left.position.createdAt);
    const requestRows: RequestRow[] = activityItems
        .flatMap(item => item.inFlightRequests.map(request => ({wallet: item.wallet, request})))
        .sort((left, right) => right.request.createdAt - left.request.createdAt);
    const total = Math.max(balances.data?.total || 0, activity.data?.total || 0);
    const walletSelectionSummary = balances.data?.walletSelection || activity.data?.walletSelection;

    const renderContent = (manager?: ConnectionManager, cashOutManager?: PositionCashOutManager, batchManager?: PositionCashOutBatchManager) => {
        if (manager && !manager.selection?.configured) {
            return <WalletSelectionGuide manager={manager} />;
        }
        if (!manager && walletSelectionSummary && !walletSelectionSummary.configured) {
            return <ReadOnlyWalletSelectionGuide />;
        }
        const connectionFor = (walletId: number) => manager?.connections.get(walletId) || activityByWallet.get(walletId)?.connection;

        return (
            <>
                <div className='worm-assets-wallet-panel'>
                    {manager ? (
                        <WalletSelectionSummary manager={manager} />
                    ) : (
                        <div className='worm-assets-readonly-heading'>
                            <Typography.Title level={2}>Worm wallets</Typography.Title>
                            <Typography.Text type='secondary'>Read-only access · {walletSelectionSummary?.selectedCount ?? 'Unknown'} selected</Typography.Text>
                        </div>
                    )}
                    {manager && <ConnectionSetupPanel manager={manager} />}
                    <RuntimeSummary
                        status={runtime.data}
                        loading={runtime.loading}
                        error={runtime.error}
                        fallbackNetwork={balances.data?.network}
                        fallbackCommitment={balances.data?.commitment}
                    />
                    {balances.error && !balances.data ? (
                        <Result
                            status='error'
                            title='Wallet balances are unavailable'
                            subTitle={requestErrorMessage(balances.error, 'The balance request failed. No wallet was treated as empty or zero.')}
                            extra={
                                <Button type='primary' disabled={manager?.operationBusy} onClick={refresh}>
                                    Try again
                                </Button>
                            }
                        />
                    ) : balances.data && balances.data.total === 0 ? (
                        (manager?.selection?.configured && manager.selection.selectedItems.length === 0) ||
                        (!manager && walletSelectionSummary?.configured && walletSelectionSummary.selectedCount === 0) ? (
                            <div className='worm-trading-empty'>
                                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='No wallets are selected for Worm Trading.'>
                                    <Typography.Paragraph type='secondary'>
                                        {manager
                                            ? `Choose up to ${manager.selection?.maximumWallets || MAXIMUM_WORM_TRADING_WALLETS} wallets to show their Assets and make them eligible for execution previews.`
                                            : 'A user with Worm Trading write access can add wallets to this saved selection.'}
                                    </Typography.Paragraph>
                                    {manager && (
                                        <Button
                                            type='primary'
                                            icon={<SettingOutlined />}
                                            disabled={manager.operationBusy || Boolean(manager.selectionError)}
                                            onClick={manager.openSelection}>
                                            Manage Worm wallets
                                        </Button>
                                    )}
                                </Empty>
                            </div>
                        ) : (
                            <EmptyWalletBalances />
                        )
                    ) : (
                        <section className='worm-trading-balances' aria-labelledby='worm-trading-balances-heading'>
                            <div className='worm-trading-balances__heading'>
                                <div>
                                    <Typography.Title id='worm-trading-balances-heading' level={2}>
                                        Wallet balances
                                    </Typography.Title>
                                    <Typography.Text type='secondary'>
                                        {balances.data ? `${balances.data.total} Solana ${balances.data.total === 1 ? 'wallet' : 'wallets'}` : 'Loading wallets'}
                                        {balances.data?.fetchedAt ? ` · Fetched ${formatBeijingUnixSeconds(balances.data.fetchedAt)}` : ''}
                                    </Typography.Text>
                                </div>
                                {(balances.refreshing || activity.refreshing) && (
                                    <Typography.Text className='worm-trading-balances__refreshing' role='status' aria-live='polite'>
                                        Refreshing wallet data…
                                    </Typography.Text>
                                )}
                            </div>
                            {balances.error && balances.data && (
                                <Alert
                                    className='worm-trading-section-alert'
                                    type='warning'
                                    showIcon={true}
                                    title='Could not refresh balances'
                                    description={requestErrorMessage(balances.error)}
                                />
                            )}
                            {balances.data ? (
                                <div className='worm-assets-wallet-list'>
                                    <div className='worm-assets-wallet-columns' aria-hidden='true'>
                                        <span>Wallet</span>
                                        <span>SOL balance</span>
                                        <span>USDC balance</span>
                                        <span>Worm access</span>
                                    </div>
                                    {balanceItems.map(item => (
                                        <WalletBalanceCard
                                            key={item.wallet.walletId}
                                            item={item}
                                            connection={connectionFor(item.wallet.walletId)}
                                            activityStatus={activityByWallet.get(item.wallet.walletId)?.status}
                                            manager={manager}
                                            batchManager={batchManager}
                                            activityLoading={activity.loading && !activity.data}
                                            onCopy={() => void copyAddress(item.wallet)}
                                        />
                                    ))}
                                </div>
                            ) : (
                                <Skeleton active={true} paragraph={{rows: 4}} />
                            )}
                            {batchManager?.selectionSummary}
                        </section>
                    )}

                    <div className='worm-assets-balance-note'>Balances are not available-to-order limits.</div>
                </div>
                {batchManager?.panel}

                {activity.error && (
                    <Alert
                        className='worm-trading-section-alert'
                        type={activity.data ? 'warning' : 'error'}
                        showIcon={true}
                        title={activity.data ? 'Could not refresh Worm activity' : 'Worm activity is unavailable'}
                        description={
                            activity.data
                                ? 'Existing connection, position, and request data remains visible.'
                                : requestErrorMessage(activity.error, 'Balances remain available; no activity stream was treated as empty.')
                        }
                        action={
                            <Button disabled={manager?.operationBusy} onClick={activity.reload}>
                                Try activity again
                            </Button>
                        }
                    />
                )}

                {cashOutManager?.alerts}

                <section className='worm-assets-activity-panel' aria-label='Worm activity'>
                    <Tabs
                        activeKey={activityTab}
                        onChange={setActivityTab}
                        items={[
                            {
                                key: 'positions',
                                label: `Open positions${activity.data ? ` · ${activity.data.openPositionCount}` : ''}`,
                                children: activity.data ? (
                                    <>
                                        <StreamNotice label='Position streams' streams={queriedActivityItems.map(item => item.positions)} />
                                        {positionRows.map(row => (
                                            <PositionCard key={positionCashOutRowKey(row)} row={row} cashOutManager={cashOutManager} onCopy={() => void copyAddress(row.wallet)} />
                                        ))}
                                        {positionRows.length === 0 && queriedActivityItems.every(item => streamAvailable(item.positions)) && (
                                            <Empty
                                                image={Empty.PRESENTED_IMAGE_SIMPLE}
                                                description={queriedActivityItems.length ? 'No open positions for the connected wallets.' : 'No connected wallets to query.'}
                                            />
                                        )}
                                    </>
                                ) : activity.loading ? (
                                    <Skeleton active={true} />
                                ) : null
                            },
                            {
                                key: 'requests',
                                label: `In-flight requests${activity.data ? ` · ${activity.data.inFlightRequestCount}` : ''}`,
                                children: activity.data ? (
                                    <>
                                        <StreamNotice label='Request streams' streams={queriedActivityItems.map(item => item.requests)} />
                                        {requestRows.map(row => (
                                            <RequestCard key={`${row.wallet.walletId}:${row.request.pubkey}`} row={row} onCopy={() => void copyAddress(row.wallet)} />
                                        ))}
                                        {requestRows.length === 0 && queriedActivityItems.every(item => streamAvailable(item.requests)) && (
                                            <Empty
                                                image={Empty.PRESENTED_IMAGE_SIMPLE}
                                                description={queriedActivityItems.length ? 'No in-flight requests for the connected wallets.' : 'No connected wallets to query.'}
                                            />
                                        )}
                                    </>
                                ) : activity.loading ? (
                                    <Skeleton active={true} />
                                ) : null
                            }
                        ]}
                    />
                </section>

                {total > pageSize && (
                    <Pagination
                        className='worm-trading-pagination'
                        responsive={true}
                        current={currentPage}
                        pageSize={pageSize}
                        total={total}
                        showSizeChanger={false}
                        showTotal={count => `${count} wallets`}
                        onChange={nextPage => setPage(nextPage, pageSize)}
                    />
                )}

                {total > 0 && balanceItems.length === 0 && !balances.loading && (
                    <Alert
                        type='warning'
                        showIcon={true}
                        title='This page contains no wallet rows.'
                        description='Move to an earlier page or refresh after your wallet inventory changes.'
                    />
                )}
            </>
        );
    };

    const renderPage = (manager?: ConnectionManager, cashOutManager?: PositionCashOutManager, batchManager?: PositionCashOutBatchManager) => (
        <div className='worm-theme-page'>
            <AppPage
                title='Worm Trading Assets'
                subtitle='Confirmed wallet balances, open positions and in-flight requests.'
                loading={loading || manager?.operationBusy}
                onRefresh={() => {
                    if (manager?.operationBusy) {
                        return;
                    }
                    refresh();
                    manager?.refreshInventory();
                }}>
                <nav className='worm-assets-navigation' aria-label='Worm Trading pages'>
                    <Button type='text' aria-current='page'>
                        Assets
                    </Button>
                    <Button type='text' onClick={() => navigate('/worm-trading/combinations')}>
                        Combinations
                    </Button>
                    <Button type='text' onClick={() => navigate('/worm-trading/executions')}>
                        Executions
                    </Button>
                </nav>
                {renderContent(manager, cashOutManager, batchManager)}
            </AppPage>
        </div>
    );

    return canManageConnections ? (
        <SensitiveWriteScope module={AccountDataModule.WormTrading}>
            <ConnectionManagement onReload={reloadConnections}>
                {manager =>
                    manager.selection?.configured ? (
                        <PositionCashOutBatchManagement visibleWallets={balanceItems} onRefreshAssets={refreshAssets}>
                            {batchManager => (
                                <PositionCashOutManagement
                                    rows={positionRows}
                                    activityFetchedAt={activity.data?.fetchedAt || 0}
                                    onRefreshAssets={refreshAssets}
                                    batchManager={batchManager}>
                                    {cashOutManager => renderPage(manager, cashOutManager, batchManager)}
                                </PositionCashOutManagement>
                            )}
                        </PositionCashOutBatchManagement>
                    ) : (
                        renderPage(manager)
                    )
                }
            </ConnectionManagement>
        </SensitiveWriteScope>
    ) : (
        renderPage()
    );
};
