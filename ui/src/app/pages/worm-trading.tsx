import {ApiOutlined, CopyOutlined, DisconnectOutlined, LinkOutlined, SyncOutlined, WalletOutlined} from '@ant-design/icons';
import {Alert, Avatar, Button, Card, Empty, Pagination, Progress, Result, Skeleton, Space, Tag, Tooltip, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {useLocation, useNavigate} from 'react-router-dom';
import {AppPage, ResourceTable, useCachedAsyncData} from '../components';
import {AccountDataModule} from '../shared/access-modules';
import {Context, useAuthorization} from '../shared/context';
import {formatBeijingUnixSeconds} from '../shared/format';
import {AccountIdentityProvider} from '../shared/models';
import {SensitiveWriteScope, useSensitiveWriteLease} from '../shared/sensitive-write-scope';
import {
    services,
    WormActivityStreamState,
    WormInFlightRequest,
    WormOpenPosition,
    WormTradingAssetBalance,
    WormTradingStatus,
    WormTradingTokenAssetBalance,
    WormTradingWalletActivityItem,
    WormTradingWalletBalanceItem,
    WormTradingWalletConnectionItem,
    WormTradingWalletSummary,
    WormWalletConnection,
    WormWalletConnectionState
} from '../shared/services';
import {WORM_TRADING_LOGIN_SESSION_REQUIRED, WORM_TRADING_REAUTH_REQUIRED, WORM_TRADING_REAUTH_UNAVAILABLE} from '../shared/services/worm-trading-service';
import {requestErrorDetails, requestErrorMessage} from '../shared/services/requests';
import {usePagedParams} from './shared';

const wormTradingPageSize = 20;
const wormTradingPageSizes = [wormTradingPageSize];
const wormConnectionInventoryPageSize = 100;
const wormConnectionStartIntervalMS = 12_000;
const pendingConnectionActionKey = 'athena.worm-trading.pending-connection-action';
const connectOutcomeUnknownWarning = 'CONNECT_OUTCOME_UNKNOWN';

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

const defaultAvatarPalettes = [
    ['#5b21b6', '#a78bfa'],
    ['#1d4ed8', '#60a5fa'],
    ['#0f766e', '#2dd4bf'],
    ['#166534', '#4ade80'],
    ['#a16207', '#fbbf24'],
    ['#c2410c', '#fb923c'],
    ['#be123c', '#fb7185'],
    ['#3730a3', '#818cf8']
];

const hashWalletAddress = (value: string) => {
    let hash = 2166136261;
    for (const character of value) {
        hash ^= character.codePointAt(0) || 0;
        hash = Math.imul(hash, 16777619);
    }
    return hash >>> 0;
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

const shortAddress = (value?: string, head = 8, tail = 8) => (value && value.length > head + tail ? `${value.slice(0, head)}…${value.slice(-tail)}` : value || '-');

const titleCase = (value?: string) =>
    (value || '')
        .toLowerCase()
        .split(/[_-]+/)
        .filter(Boolean)
        .map(part => `${part.slice(0, 1).toUpperCase()}${part.slice(1)}`)
        .join(' ');

const optionalValue = (value: string, suffix = '') => (value ? `${value}${suffix}` : '-');
const liquidationPriceValue = (position: WormOpenPosition) =>
    position.liquidationPrice || (Number(position.leverage) === 1 ? 'No liquidation (1×)' : '-');
const assetAvailable = (asset?: WormTradingAssetBalance) => asset?.availability === 'AVAILABLE' || asset?.availability === 'BALANCE_AVAILABILITY_AVAILABLE';
const streamAvailable = (stream: WormActivityStreamState) => stream.availability === 'AVAILABLE';
const connectionWasQueried = (state: WormWalletConnectionState) => state === 'CONNECTED' || state === 'RECONNECT_REQUIRED';

const WormTradingWalletAvatar = (props: {wallet: WormTradingWalletSummary; size?: number}) => {
    const presetGlyph = walletPresetGlyphs[props.wallet.avatarPresetId];
    const palette = defaultAvatarPalettes[hashWalletAddress(props.wallet.address) % defaultAvatarPalettes.length];
    const uploaded = props.wallet.avatarKind.toLowerCase() === 'upload' && props.wallet.avatarUrl;
    const className = ['wallet-avatar', presetGlyph ? `wallet-avatar--${props.wallet.avatarPresetId}` : 'wallet-avatar--generated'].join(' ');
    const style = presetGlyph ? undefined : {background: `linear-gradient(145deg, ${palette[0]}, ${palette[1]})`};
    return (
        <Avatar aria-hidden='true' className={className} size={props.size || 46} src={uploaded || undefined} style={style}>
            {presetGlyph || 'S'}
        </Avatar>
    );
};

const BalanceStatusTag = (props: {status: WormTradingWalletBalanceItem['status']}) => {
    const color = props.status === 'COMPLETE' ? 'green' : props.status === 'PARTIAL' ? 'gold' : 'red';
    return <Tag color={color}>{titleCase(props.status) || 'Unavailable'}</Tag>;
};

const ActivityStatusTag = (props: {status: WormTradingWalletActivityItem['status']}) => {
    const color = props.status === 'COMPLETE' ? 'green' : props.status === 'PARTIAL' ? 'gold' : 'red';
    return <Tag color={color}>Activity {titleCase(props.status) || 'Unavailable'}</Tag>;
};

const ConnectionStatusTag = (props: {state: WormWalletConnectionState}) => {
    const color =
        props.state === 'CONNECTED' ? 'green' : props.state === 'NOT_CONNECTED' ? 'default' : props.state === 'CONNECTING' || props.state === 'DISCONNECTING' ? 'blue' : 'gold';
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
                <strong>{available ? `${optionalValue(props.asset.amount)} ${props.symbol}` : 'Unavailable'}</strong>
            </Tooltip>
            <small>{detail}</small>
        </div>
    );
};

const WalletIdentity = (props: {wallet: WormTradingWalletSummary; onCopy: () => void; compact?: boolean}) => (
    <div className={props.compact ? 'worm-trading-wallet worm-trading-wallet--compact' : 'worm-trading-wallet'}>
        <WormTradingWalletAvatar wallet={props.wallet} size={props.compact ? 42 : 46} />
        <div className='worm-trading-wallet__main'>
            <Typography.Text strong={true} ellipsis={{tooltip: props.wallet.remark || 'Solana wallet'}}>
                {props.wallet.remark || 'Solana wallet'}
            </Typography.Text>
            <span className='worm-trading-wallet__address'>
                <Tooltip title={props.wallet.address}>
                    <code>{shortAddress(props.wallet.address)}</code>
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
    const stateColor = ready ? 'green' : !status?.started || runtimeState === 'configuration_error' ? 'red' : 'gold';
    const wormState = titleCase(status?.wormAPIStatus) || (status?.credentialStoreReady ? 'Not observed' : 'Unavailable');
    const wormRuntimeState = (status?.wormAPIStatus || '').toLowerCase();
    const wormColor = wormRuntimeState === 'running' ? 'green' : wormRuntimeState === 'configuration_error' || !status?.credentialStoreReady ? 'red' : 'gold';
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
type PendingConnectionIntent = {kind: 'auto-connect'} | {kind: ManagedConnectionAction; walletId: number};

const readPendingConnectionIntent = (): PendingConnectionIntent | undefined => {
    const raw = window.sessionStorage.getItem(pendingConnectionActionKey);
    if (!raw) {
        return undefined;
    }
    window.sessionStorage.removeItem(pendingConnectionActionKey);
    try {
        const value = JSON.parse(raw) as Partial<PendingConnectionIntent>;
        if (value.kind === 'auto-connect') {
            return {kind: 'auto-connect'};
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
                ? 'Connecting wallets to Worm'
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
                {setup.processed > 0 ? 'Authorize and continue' : 'Authorize and connect'}
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
                    {setup.total > 0 && (
                        <div className='worm-trading-connection-setup__progress'>
                            <Progress percent={percent} status={progressStatus} showInfo={false} />
                            <span>
                                {setup.processed}/{setup.total} processed · {setup.succeeded} connected · {setup.failed} failed · {setup.remaining} remaining
                            </span>
                        </div>
                    )}
                    {setup.currentWallet && (
                        <Typography.Text type='secondary'>Current wallet: {setup.currentWallet.remark || shortAddress(setup.currentWallet.address)}</Typography.Text>
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

const ConnectionManagement = (props: {onReload: () => void; children: (manage: ConnectionManager) => React.ReactNode}) => {
    const ctx = React.useContext(Context);
    const authorization = useAuthorization();
    const location = useLocation();
    const lease = useSensitiveWriteLease();
    const [busyWalletId, setBusyWalletId] = React.useState<number>();
    const [stage, setStage] = React.useState('');
    const [operationBusy, setOperationBusy] = React.useState(false);
    const [setup, setSetup] = React.useState<ConnectionSetupState>({...hiddenConnectionSetupState, phase: 'discovering', message: 'Checking every Solana wallet in this account.'});
    const [connectionItems, setConnectionItems] = React.useState<WormTradingWalletConnectionItem[]>([]);
    const [walletProgress, setWalletProgress] = React.useState<Map<number, string>>(() => new Map());
    const resumedRef = React.useRef(false);
    const operationEpochRef = React.useRef(0);
    const runningRef = React.useRef(false);
    const cancelDelayRef = React.useRef<(() => void) | undefined>();
    const lastAutoStartAtRef = React.useRef(0);
    const targetWalletIDsRef = React.useRef(new Set<number>());
    const attemptedWalletIDsRef = React.useRef(new Set<number>());
    const succeededWalletIDsRef = React.useRef(new Set<number>());
    const failedWalletIDsRef = React.useRef(new Set<number>());
    const blockedWalletIDsRef = React.useRef(new Set<number>());
    const onReloadRef = React.useRef(props.onReload);
    onReloadRef.current = props.onReload;

    const isCurrent = React.useCallback((epoch: number) => operationEpochRef.current === epoch, []);

    const progressState = React.useCallback(
        (phase: ConnectionSetupPhase, message: string, options?: {currentWallet?: WormTradingWalletSummary; retryable?: boolean}): ConnectionSetupState => {
            const total = targetWalletIDsRef.current.size;
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

    const fetchConnectionInventory = React.useCallback(
        async (epoch: number): Promise<WormTradingWalletConnectionItem[] | undefined> => {
            const items: WormTradingWalletConnectionItem[] = [];
            const seenWalletIDs = new Set<number>();
            let page = 1;
            let expectedTotal: number | undefined;
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
                } else if (result.value.total !== expectedTotal) {
                    throw new Error('The Solana wallet inventory changed while Worm connections were being checked. Refresh before continuing.');
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

    const runAutoConnectionQueue = React.useCallback(
        async (epoch: number, inventory: WormTradingWalletConnectionItem[]) => {
            const candidates = inventory.filter(
                item =>
                    item.connection.state === 'NOT_CONNECTED' &&
                    item.connection.warningCode !== connectOutcomeUnknownWarning &&
                    !attemptedWalletIDsRef.current.has(item.wallet.walletId) &&
                    !blockedWalletIDsRef.current.has(item.wallet.walletId)
            );
            if (candidates.length === 0) {
                const unknown = inventory.find(item => item.connection.warningCode === connectOutcomeUnknownWarning);
                if (unknown) {
                    setSetup(
                        progressState(
                            'blocked',
                            `${unknown.wallet.remark || shortAddress(unknown.wallet.address)} has an unknown connection outcome. Automatic retry is blocked until the state is reviewed.`
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
                          : 'Confirm your identity once to connect every eligible Solana wallet. No transaction or network fee is involved.';
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
            const unknown = refreshedInventory?.find(item => item.connection.warningCode === connectOutcomeUnknownWarning);
            if (unknown) {
                blockedWalletIDsRef.current.add(unknown.wallet.walletId);
                setWalletProgress(current => new Map(current).set(unknown.wallet.walletId, 'Connection outcome is unknown; automatic retry is blocked.'));
                setSetup(
                    progressState(
                        'blocked',
                        `${unknown.wallet.remark || shortAddress(unknown.wallet.address)} has an unknown connection outcome. Automatic retry is blocked until the state is reviewed.`,
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
                    connectedCount === 1 ? 'Worm wallet connected' : `${connectedCount} Worm wallets connected`,
                    'The authoritative connection and activity state has been refreshed.'
                );
            }
            targetWalletIDsRef.current.clear();
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
            setSetup(current => ({...current, phase: 'discovering', message: 'Checking every Solana wallet in this account.', currentWallet: undefined}));
            const inventory = await fetchConnectionInventory(epoch);
            if (!inventory || !isCurrent(epoch)) {
                return;
            }
            publishInventory(inventory);
            if (resetBatch) {
                targetWalletIDsRef.current.clear();
                attemptedWalletIDsRef.current.clear();
                succeededWalletIDsRef.current.clear();
                failedWalletIDsRef.current.clear();
                blockedWalletIDsRef.current.clear();
                setWalletProgress(new Map());
            }
            for (const item of inventory) {
                if (item.connection.warningCode === connectOutcomeUnknownWarning) {
                    blockedWalletIDsRef.current.add(item.wallet.walletId);
                } else if (item.connection.state === 'NOT_CONNECTED' && !attemptedWalletIDsRef.current.has(item.wallet.walletId)) {
                    targetWalletIDsRef.current.add(item.wallet.walletId);
                }
            }
            const unknown = inventory.find(item => item.connection.warningCode === connectOutcomeUnknownWarning);
            if (unknown) {
                setSetup(
                    progressState('blocked', `${unknown.wallet.remark || shortAddress(unknown.wallet.address)} has an unknown connection outcome. Automatic retry remains blocked.`)
                );
                return;
            }
            if (runConnections) {
                await runAutoConnectionQueue(epoch, inventory);
                return;
            }
            if (failedWalletIDsRef.current.size > 0) {
                setSetup(progressState('partial', 'Connection state refreshed. Failed wallets were not retried automatically.', {retryable: true}));
            } else {
                setSetup(hiddenConnectionSetupState);
            }
        },
        [fetchConnectionInventory, isCurrent, progressState, publishInventory, runAutoConnectionQueue]
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
                    beginGoogleReauthentication({kind: 'auto-connect'});
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
        if (pending && pending.kind !== 'auto-connect') {
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

    const refreshInventory = React.useCallback(() => void startDiscovery(false, true), [startDiscovery]);
    const retry = React.useCallback(() => void startDiscovery(true, true), [startDiscovery]);
    const connections = React.useMemo(() => new Map(connectionItems.map(item => [item.wallet.walletId, item.connection])), [connectionItems]);
    const manager = React.useMemo<ConnectionManager>(
        () => ({busyWalletId, stage, operationBusy, setup, connections, walletProgress, authorize: () => void authorize(), retry, refreshInventory, confirm}),
        [authorize, busyWalletId, confirm, connections, operationBusy, refreshInventory, retry, setup, stage, walletProgress]
    );
    return <>{props.children(manager)}</>;
};

interface ConnectionManager {
    busyWalletId?: number;
    stage: string;
    operationBusy: boolean;
    setup: ConnectionSetupState;
    connections: Map<number, WormWalletConnection>;
    walletProgress: Map<number, string>;
    authorize(): void;
    retry(): void;
    refreshInventory(): void;
    confirm(action: ManagedConnectionAction, walletId: number, walletLabel: string): void;
}

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
                {props.connection.connectedAt > 0 && state === 'CONNECTED' && <small>Since {formatBeijingUnixSeconds(props.connection.connectedAt)}</small>}
                {props.manager?.walletProgress.get(props.wallet.walletId) && <small>{props.manager.walletProgress.get(props.wallet.walletId)}</small>}
            </div>
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
            {busy && props.manager?.stage && <small className='worm-trading-connection__stage'>{props.manager.stage}</small>}
        </div>
    );
};

const WalletBalanceCard = (props: {
    item: WormTradingWalletBalanceItem;
    connection?: WormWalletConnection;
    activityStatus?: WormTradingWalletActivityItem['status'];
    manager?: ConnectionManager;
    activityLoading?: boolean;
    onCopy: () => void;
}) => (
    <Card className='worm-trading-balance-card' size='small'>
        <div className='worm-trading-balance-card__header'>
            <WalletIdentity wallet={props.item.wallet} compact={true} onCopy={props.onCopy} />
            <BalanceStatusTag status={props.item.status} />
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
            <Typography.Text strong={true} ellipsis={{tooltip: props.market.title || props.market.conditionId}}>
                {props.market.title || 'Untitled market'}
            </Typography.Text>
            <small>
                {props.market.eventTitle || shortAddress(props.market.conditionId, 7, 7)}
                {props.market.lastTradePrice ? ` · Latest ${props.market.lastTradePrice}` : ''}
            </small>
        </div>
    </div>
);

const SideTag = (props: {side: string}) => <Tag color={props.side === 'YES' ? 'green' : props.side === 'NO' ? 'red' : 'default'}>{props.side || 'Unknown'}</Tag>;

const PositionCard = (props: {row: PositionRow; onCopy: () => void}) => {
    const position = props.row.position;
    return (
        <Card className='worm-trading-activity-card' size='small'>
            <WalletIdentity wallet={props.row.wallet} compact={true} onCopy={props.onCopy} />
            <MarketIdentity market={position.market} />
            <div className='worm-trading-activity-card__tags'>
                <SideTag side={position.side} />
                <Tag>{optionalValue(position.leverage, '×')}</Tag>
                {position.isLiquidated && <Tag color='red'>Liquidated</Tag>}
                {position.isClaimed && <Tag color='blue'>Claimed</Tag>}
            </div>
            <dl className='worm-trading-activity-card__facts'>
                <div>
                    <dt>Shares</dt>
                    <dd>{optionalValue(position.totalShares)}</dd>
                </div>
                <div>
                    <dt>Entry</dt>
                    <dd>{optionalValue(position.averageEntryPrice)}</dd>
                </div>
                <div>
                    <dt>Liquidity</dt>
                    <dd>{position.userLiquidity && position.totalLiquidity ? `${position.userLiquidity} / ${position.totalLiquidity}` : '-'}</dd>
                </div>
                <div>
                    <dt>Liquidation</dt>
                    <dd>{liquidationPriceValue(position)}</dd>
                </div>
                <div>
                    <dt>Unrealized P&amp;L</dt>
                    <dd>{optionalValue(position.unrealizedPnL)}</dd>
                </div>
                <div>
                    <dt>Realized P&amp;L</dt>
                    <dd>{optionalValue(position.realizedPnL)}</dd>
                </div>
            </dl>
            <div className='worm-trading-activity-card__footer'>
                <code title={position.pubkey}>{shortAddress(position.pubkey, 7, 7)}</code>
                <span>{formatBeijingUnixSeconds(position.createdAt) || '-'}</span>
            </div>
        </Card>
    );
};

const RequestCard = (props: {row: RequestRow; onCopy: () => void}) => {
    const request = props.row.request;
    return (
        <Card className='worm-trading-activity-card' size='small'>
            <WalletIdentity wallet={props.row.wallet} compact={true} onCopy={props.onCopy} />
            <MarketIdentity market={request.market} />
            <div className='worm-trading-activity-card__tags'>
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
            <div className='worm-trading-activity-card__footer'>
                <code title={request.pubkey}>{shortAddress(request.pubkey, 7, 7)}</code>
                <span>{formatBeijingUnixSeconds(request.createdAt) || '-'}</span>
            </div>
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

export const WormTradingPage = () => {
    const ctx = React.useContext(Context);
    const authorization = useAuthorization();
    const {page, pageSize, setPage} = usePagedParams(wormTradingPageSize, wormTradingPageSizes);
    const currentPage = Number.isSafeInteger(page) && page > 0 ? page : 1;
    const accountID = authorization.user.accountId;
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

    const renderContent = (manager?: ConnectionManager) => {
        const connectionFor = (walletId: number) => manager?.connections.get(walletId) || activityByWallet.get(walletId)?.connection;
        const balanceColumns: ColumnsType<WormTradingWalletBalanceItem> = [
            {
                title: 'Wallet',
                key: 'wallet',
                width: 330,
                render: (_, item) => <WalletIdentity wallet={item.wallet} onCopy={() => void copyAddress(item.wallet)} />
            },
            {title: 'SOL', key: 'sol', width: 185, render: (_, item) => <AssetValue asset={item.sol} symbol='SOL' />},
            {title: 'USDC', key: 'usdc', width: 210, render: (_, item) => <AssetValue asset={item.usdc} symbol='USDC' token={item.usdc} />},
            {title: 'Balance', key: 'status', width: 125, render: (_, item) => <BalanceStatusTag status={item.status} />},
            {
                title: 'Worm access',
                key: 'connection',
                width: 230,
                render: (_, item) => (
                    <ConnectionCell
                        wallet={item.wallet}
                        connection={connectionFor(item.wallet.walletId)}
                        activityStatus={activityByWallet.get(item.wallet.walletId)?.status}
                        manager={manager}
                        loading={activity.loading && !activity.data}
                    />
                )
            }
        ];
        const positionColumns: ColumnsType<PositionRow> = [
            {title: 'Wallet', key: 'wallet', width: 245, render: (_, row) => <WalletIdentity wallet={row.wallet} onCopy={() => void copyAddress(row.wallet)} />},
            {title: 'Market', key: 'market', width: 330, render: (_, row) => <MarketIdentity market={row.position.market} />},
            {
                title: 'Side / leverage',
                key: 'side',
                width: 145,
                render: (_, row) => (
                    <Space size={4}>
                        <SideTag side={row.position.side} />
                        <Tag>{optionalValue(row.position.leverage, '×')}</Tag>
                    </Space>
                )
            },
            {
                title: 'Shares / entry',
                key: 'entry',
                width: 165,
                render: (_, row) => (
                    <div className='worm-trading-data-pair'>
                        <strong>{optionalValue(row.position.totalShares)}</strong>
                        <small>Entry {optionalValue(row.position.averageEntryPrice)}</small>
                    </div>
                )
            },
            {
                title: 'Liquidity',
                key: 'liquidity',
                width: 160,
                render: (_, row) => (
                    <div className='worm-trading-data-pair'>
                        <strong>{optionalValue(row.position.userLiquidity)}</strong>
                        <small>Total {optionalValue(row.position.totalLiquidity)}</small>
                    </div>
                )
            },
            {
                title: 'Risk / P&L',
                key: 'pnl',
                width: 180,
                render: (_, row) => (
                    <div className='worm-trading-data-pair'>
                        <strong>UPnL {optionalValue(row.position.unrealizedPnL)}</strong>
                        <small>
                            Liq {liquidationPriceValue(row.position)} · Realized {optionalValue(row.position.realizedPnL)}
                        </small>
                    </div>
                )
            },
            {
                title: 'Position',
                key: 'position',
                width: 170,
                render: (_, row) => (
                    <div className='worm-trading-data-pair'>
                        <code title={row.position.pubkey}>{shortAddress(row.position.pubkey, 7, 7)}</code>
                        <small>{formatBeijingUnixSeconds(row.position.createdAt) || '-'}</small>
                    </div>
                )
            }
        ];
        const requestColumns: ColumnsType<RequestRow> = [
            {title: 'Wallet', key: 'wallet', width: 245, render: (_, row) => <WalletIdentity wallet={row.wallet} onCopy={() => void copyAddress(row.wallet)} />},
            {title: 'Market', key: 'market', width: 330, render: (_, row) => <MarketIdentity market={row.request.market} />},
            {
                title: 'Request state',
                key: 'state',
                width: 185,
                render: (_, row) => (
                    <div className='worm-trading-data-pair'>
                        <strong>{titleCase(row.request.state) || '-'}</strong>
                        <small>
                            {titleCase(row.request.type) || 'Request'}
                            {row.request.orderState ? ` · ${titleCase(row.request.orderState)}` : ''}
                        </small>
                    </div>
                )
            },
            {
                title: 'Order',
                key: 'order',
                width: 150,
                render: (_, row) => (
                    <Space size={4}>
                        <SideTag side={row.request.side} />
                        <Tag>{optionalValue(row.request.leverage, '×')}</Tag>
                    </Space>
                )
            },
            {
                title: 'Funds / price',
                key: 'funds',
                width: 165,
                render: (_, row) => (
                    <div className='worm-trading-data-pair'>
                        <strong>{optionalValue(row.request.funds)}</strong>
                        <small>
                            Price {optionalValue(row.request.price)} · Shares {optionalValue(row.request.shares)}
                        </small>
                    </div>
                )
            },
            {
                title: 'Request',
                key: 'request',
                width: 170,
                render: (_, row) => (
                    <div className='worm-trading-data-pair'>
                        <code title={row.request.pubkey}>{shortAddress(row.request.pubkey, 7, 7)}</code>
                        <small>{formatBeijingUnixSeconds(row.request.createdAt) || '-'}</small>
                    </div>
                )
            }
        ];

        return (
            <>
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
                    <EmptyWalletBalances />
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
                        <ResourceTable<WormTradingWalletBalanceItem>
                            rowKey={item => item.wallet.walletId || item.wallet.address}
                            label='Worm Trading wallet balances and connections'
                            items={balanceItems}
                            loading={balances.loading && !balances.data}
                            columns={balanceColumns}
                            scrollX={1080}
                            compactRender={item => (
                                <WalletBalanceCard
                                    item={item}
                                    connection={connectionFor(item.wallet.walletId)}
                                    activityStatus={activityByWallet.get(item.wallet.walletId)?.status}
                                    manager={manager}
                                    activityLoading={activity.loading && !activity.data}
                                    onCopy={() => void copyAddress(item.wallet)}
                                />
                            )}
                            compactEmptyDescription='No Solana wallets are available.'
                        />
                    </section>
                )}

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

                <section className='worm-trading-activity' aria-labelledby='worm-trading-positions-heading'>
                    <div className='worm-trading-balances__heading'>
                        <div>
                            <Typography.Title id='worm-trading-positions-heading' level={2}>
                                Open positions
                            </Typography.Title>
                            <Typography.Text type='secondary'>
                                {activity.data
                                    ? `${activity.data.openPositionCount} visible ${activity.data.openPositionCount === 1 ? 'position' : 'positions'}`
                                    : 'Current Worm positions by wallet'}
                            </Typography.Text>
                        </div>
                        {activity.data && <ActivityStatusTag status={activity.data.status} />}
                    </div>
                    {activity.data && (
                        <>
                            <StreamNotice label='Position streams' streams={queriedActivityItems.map(item => item.positions)} />
                            <ResourceTable<PositionRow>
                                rowKey={row => `${row.wallet.walletId}:${row.position.pubkey}`}
                                label='Open Worm positions'
                                items={positionRows}
                                loading={activity.loading && !activity.data}
                                columns={positionColumns}
                                scrollX={1395}
                                compactRender={row => <PositionCard row={row} onCopy={() => void copyAddress(row.wallet)} />}
                                compactEmptyDescription='No open positions are available for the connected wallets on this page.'
                            />
                        </>
                    )}
                </section>

                <section className='worm-trading-activity' aria-labelledby='worm-trading-requests-heading'>
                    <div className='worm-trading-balances__heading'>
                        <div>
                            <Typography.Title id='worm-trading-requests-heading' level={2}>
                                In-flight requests
                            </Typography.Title>
                            <Typography.Text type='secondary'>
                                {activity.data
                                    ? `${activity.data.inFlightRequestCount} visible ${activity.data.inFlightRequestCount === 1 ? 'request' : 'requests'}`
                                    : 'Open requests that have not reached a terminal state'}
                                {activity.data?.fetchedAt ? ` · Fetched ${formatBeijingUnixSeconds(activity.data.fetchedAt)}` : ''}
                            </Typography.Text>
                        </div>
                        {activity.refreshing && (
                            <Typography.Text className='worm-trading-balances__refreshing' role='status' aria-live='polite'>
                                Refreshing activity…
                            </Typography.Text>
                        )}
                    </div>
                    {activity.data && (
                        <>
                            <StreamNotice label='Request streams' streams={queriedActivityItems.map(item => item.requests)} />
                            <ResourceTable<RequestRow>
                                rowKey={row => `${row.wallet.walletId}:${row.request.pubkey}`}
                                label='In-flight Worm position requests'
                                items={requestRows}
                                loading={false}
                                columns={requestColumns}
                                scrollX={1245}
                                compactRender={row => <RequestCard row={row} onCopy={() => void copyAddress(row.wallet)} />}
                                compactEmptyDescription='No in-flight position requests are available for the connected wallets on this page.'
                            />
                        </>
                    )}
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

    const renderPage = (manager?: ConnectionManager) => (
        <AppPage
            title='Worm Trading Assets'
            subtitle='Review confirmed wallet balances and official Worm position activity. Balances are not Worm collateral or available-to-order limits.'
            loading={loading || manager?.operationBusy}
            onRefresh={() => {
                if (manager?.operationBusy) {
                    return;
                }
                refresh();
                manager?.refreshInventory();
            }}>
            {renderContent(manager)}
        </AppPage>
    );

    return canManageConnections ? (
        <SensitiveWriteScope module={AccountDataModule.WormTrading}>
            <ConnectionManagement onReload={reloadConnections}>{manager => renderPage(manager)}</ConnectionManagement>
        </SensitiveWriteScope>
    ) : (
        renderPage()
    );
};
