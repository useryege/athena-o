import {ApiOutlined, CopyOutlined, DisconnectOutlined, LinkOutlined, SyncOutlined, WalletOutlined} from '@ant-design/icons';
import {Alert, Avatar, Button, Card, Empty, Pagination, Result, Skeleton, Space, Tag, Tooltip, Typography} from 'antd';
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
    WormTradingWalletSummary,
    WormWalletConnection,
    WormWalletConnectionState
} from '../shared/services';
import {WORM_TRADING_LOGIN_SESSION_REQUIRED, WORM_TRADING_REAUTH_REQUIRED, WORM_TRADING_REAUTH_UNAVAILABLE} from '../shared/services/worm-trading-service';
import {requestErrorDetails, requestErrorMessage} from '../shared/services/requests';
import {usePagedParams} from './shared';

const wormTradingPageSize = 20;
const wormTradingPageSizes = [wormTradingPageSize];
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

type ConnectionAction = 'connect' | 'reconnect' | 'disconnect';

interface PendingConnectionAction {
    action: ConnectionAction;
    walletId: number;
}

const readPendingConnectionAction = (): PendingConnectionAction | undefined => {
    const raw = window.sessionStorage.getItem(pendingConnectionActionKey);
    if (!raw) {
        return undefined;
    }
    window.sessionStorage.removeItem(pendingConnectionActionKey);
    try {
        const value = JSON.parse(raw) as Partial<PendingConnectionAction>;
        const walletId = Number(value.walletId);
        return (value.action === 'connect' || value.action === 'reconnect' || value.action === 'disconnect') && Number.isInteger(walletId) && walletId > 0
            ? {action: value.action, walletId}
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

const ConnectionManagement = (props: {onReload: () => void; children: (manage: ConnectionManager) => React.ReactNode}) => {
    const ctx = React.useContext(Context);
    const authorization = useAuthorization();
    const location = useLocation();
    const lease = useSensitiveWriteLease();
    const [busyWalletId, setBusyWalletId] = React.useState<number>();
    const [stage, setStage] = React.useState('');
    const resumedRef = React.useRef(false);

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
            setStage('Connecting to Phantom…');
            const connection = provider.publicKey ? {publicKey: provider.publicKey} : await provider.connect();
            connectedAddress = connection.publicKey?.toString() || '';
            if (!connectedAddress || connectedAddress !== expectedAddress) {
                throw new Error('Connect the same Phantom account that you use to sign in to Athena.');
            }
            provider.on?.('accountChanged', onAccountChanged);

            setStage('Preparing a Worm credential approval message…');
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

            setStage('Approve the message in Phantom. No transaction or network fee is involved…');
            const signed = await provider.signMessage(new TextEncoder().encode(challenge.value.message), 'utf8');
            if (accountChanged || provider.publicKey?.toString() !== expectedAddress) {
                throw new Error('The connected Phantom account changed before verification completed.');
            }

            setStage('Verifying the signature…');
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
    }, [authorization.user.identity.solanaAddress, ctx.notifications, lease]);

    const establishDevelopmentLease = React.useCallback(async () => {
        setStage('Confirming the local development session…');
        const result = await lease.runTask(() => services.wormTrading.createDevelopmentCredentialLease());
        if (result.status === 'fulfilled') {
            return true;
        }
        if (result.status === 'rejected') {
            ctx.notifications.error('Could not confirm development session', wormConnectionErrorMessage(result.error, 'Local Worm credential access was denied.'));
        }
        return false;
    }, [ctx.notifications, lease]);

    const beginGoogleReauthentication = React.useCallback((action: PendingConnectionAction) => {
        window.sessionStorage.setItem(pendingConnectionActionKey, JSON.stringify(action));
        setStage('Opening Google for a fresh identity check…');
        window.location.assign(services.wormTrading.googleCredentialReauthenticationURL('/worm-trading'));
    }, []);

    const run = React.useCallback(
        async (action: ConnectionAction, walletId: number, resumed = false) => {
            if (busyWalletId !== undefined) {
                return;
            }
            setBusyWalletId(walletId);
            setStage(
                action === 'disconnect' ? 'Requesting credential revocation…' : action === 'reconnect' ? 'Preparing a replacement credential…' : 'Preparing a Worm credential…'
            );
            const request = () =>
                lease.runTask(() =>
                    action === 'connect'
                        ? services.wormTrading.connectWallet(walletId)
                        : action === 'reconnect'
                          ? services.wormTrading.reconnectWallet(walletId)
                          : services.wormTrading.disconnectWallet(walletId)
                );
            let result = await request();
            if (result.status === 'rejected' && requestErrorDetails(result.error).reason === WORM_TRADING_REAUTH_REQUIRED && !resumed) {
                switch (authorization.user.identity.provider) {
                    case AccountIdentityProvider.Google:
                        beginGoogleReauthentication({action, walletId});
                        return;
                    case AccountIdentityProvider.SolanaWallet:
                        if (await establishSolanaLease()) {
                            setStage(action === 'disconnect' ? 'Revoking the Worm credential…' : 'Creating the Worm credential…');
                            result = await request();
                        } else {
                            setBusyWalletId(undefined);
                            setStage('');
                            return;
                        }
                        break;
                    case AccountIdentityProvider.Development:
                        if (await establishDevelopmentLease()) {
                            setStage(action === 'disconnect' ? 'Revoking the Worm credential…' : 'Creating the Worm credential…');
                            result = await request();
                        } else {
                            setBusyWalletId(undefined);
                            setStage('');
                            return;
                        }
                        break;
                    default:
                        ctx.notifications.error('Reauthentication is unavailable', 'This login identity cannot approve Worm credential management.');
                        setBusyWalletId(undefined);
                        setStage('');
                        return;
                }
            }

            if (result.status === 'fulfilled') {
                ctx.notifications.success(action === 'disconnect' ? 'Worm wallet disconnected' : action === 'reconnect' ? 'Worm wallet reconnected' : 'Worm wallet connected');
                props.onReload();
            } else if (result.status === 'rejected') {
                const callbackReason = new URLSearchParams(location.search).get('wormTradingReason') || new URLSearchParams(location.search).get('wormCredentialReason') || '';
                const fallback = resumed && callbackReason ? `Google reauthentication did not complete (${callbackReason}).` : `Could not ${action} this Worm wallet.`;
                ctx.notifications.error(
                    action === 'disconnect' ? 'Could not disconnect Worm wallet' : action === 'reconnect' ? 'Could not reconnect Worm wallet' : 'Could not connect Worm wallet',
                    wormConnectionErrorMessage(result.error, fallback)
                );
                // A rejected upstream operation can still persist a fail-closed
                // connection state such as CONNECT_OUTCOME_UNKNOWN or
                // REVOCATION_REQUIRED. Reload before allowing another action.
                props.onReload();
            }
            setBusyWalletId(undefined);
            setStage('');
        },
        [
            authorization.user.identity.provider,
            beginGoogleReauthentication,
            busyWalletId,
            ctx.notifications,
            establishDevelopmentLease,
            establishSolanaLease,
            lease,
            location.search,
            props
        ]
    );

    React.useEffect(() => {
        if (resumedRef.current) {
            return;
        }
        resumedRef.current = true;
        const pending = readPendingConnectionAction();
        if (pending) {
            void run(pending.action, pending.walletId, true);
        }
    }, [run]);

    const confirm = React.useCallback(
        (action: ConnectionAction, walletId: number, walletLabel: string) => {
            const disconnecting = action === 'disconnect';
            const reconnecting = action === 'reconnect';
            ctx.modal.confirm({
                title: disconnecting ? `Disconnect ${walletLabel} from Worm?` : reconnecting ? `Reconnect ${walletLabel} to Worm?` : `Connect ${walletLabel} to Worm?`,
                content: disconnecting ? (
                    <Typography.Paragraph>
                        Athena will revoke only the Worm API credential it created for this wallet. This does not cancel orders, close positions, or move funds.
                    </Typography.Paragraph>
                ) : reconnecting ? (
                    <Typography.Paragraph>
                        Athena will create and securely store a replacement Worm API credential. The previous Athena credential remains recorded until Worm confirms its revocation.
                        This does not cancel orders, close positions, or move funds.
                    </Typography.Paragraph>
                ) : (
                    <Typography.Paragraph>
                        Athena will ask the custodial wallet to sign a fixed Worm credential challenge, then securely store the official Worm API credential. No transaction or
                        network fee is involved.
                    </Typography.Paragraph>
                ),
                okText: disconnecting ? 'Disconnect' : reconnecting ? 'Reconnect' : 'Connect',
                onOk: () => run(action, walletId)
            });
        },
        [ctx.modal, run]
    );

    const manager = React.useMemo<ConnectionManager>(() => ({busyWalletId, stage, confirm}), [busyWalletId, confirm, stage]);
    return <>{props.children(manager)}</>;
};

interface ConnectionManager {
    busyWalletId?: number;
    stage: string;
    confirm(action: ConnectionAction, walletId: number, walletLabel: string): void;
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
        props.connection.warningCode === connectOutcomeUnknownWarning
            ? []
            : state === 'NOT_CONNECTED'
              ? [{kind: 'connect' as const, label: 'Connect', icon: <LinkOutlined />}]
              : state === 'RECONNECT_REQUIRED'
                ? [
                      {kind: 'reconnect' as const, label: 'Reconnect', icon: <SyncOutlined />},
                      {kind: 'disconnect' as const, label: 'Disconnect', icon: <DisconnectOutlined />}
                  ]
                : state === 'CONNECTED'
                  ? [{kind: 'disconnect' as const, label: 'Disconnect', icon: <DisconnectOutlined />}]
                  : state === 'DISCONNECTING' || state === 'REVOCATION_REQUIRED'
                    ? [{kind: 'disconnect' as const, label: 'Retry disconnect', icon: <DisconnectOutlined />}]
                    : [];
    return (
        <div className='worm-trading-connection'>
            <div className='worm-trading-connection__state'>
                <ConnectionStatusTag state={state} />
                {props.activityStatus && (connectionWasQueried(state) ? <ActivityStatusTag status={props.activityStatus} /> : <Tag>Activity not queried</Tag>)}
                {props.connection.warningCode && <small>{titleCase(props.connection.warningCode)}</small>}
                {props.connection.connectedAt > 0 && state === 'CONNECTED' && <small>Since {formatBeijingUnixSeconds(props.connection.connectedAt)}</small>}
            </div>
            {props.manager && actions.length > 0 && (
                <Space size={4} wrap={true}>
                    {actions.map(action => (
                        <Button
                            key={action.kind}
                            size='small'
                            danger={action.kind === 'disconnect'}
                            icon={action.icon}
                            loading={busy}
                            disabled={props.manager?.busyWalletId !== undefined && !busy}
                            onClick={() => props.manager?.confirm(action.kind, props.wallet.walletId, label)}>
                            {action.label}
                        </Button>
                    ))}
                </Space>
            )}
            {busy && props.manager?.stage && (
                <small className='worm-trading-connection__stage' role='status' aria-live='polite'>
                    {props.manager.stage}
                </small>
            )}
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
                    <dd>{optionalValue(position.liquidationPrice)}</dd>
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
        activity.reload();
    }, [activity, runtime]);
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
                        connection={activityByWallet.get(item.wallet.walletId)?.connection}
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
                            Liq {optionalValue(row.position.liquidationPrice)} · Realized {optionalValue(row.position.realizedPnL)}
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
                            <Button type='primary' onClick={refresh}>
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
                                    connection={activityByWallet.get(item.wallet.walletId)?.connection}
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
                        action={<Button onClick={activity.reload}>Try activity again</Button>}
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

    return (
        <AppPage
            title='Worm Trading'
            subtitle='Review confirmed wallet balances and official Worm position activity. Balances are not Worm collateral or available-to-order limits.'
            loading={loading}
            onRefresh={refresh}>
            {canManageConnections ? (
                <SensitiveWriteScope module={AccountDataModule.WormTrading}>
                    <ConnectionManagement onReload={reloadConnections}>{manager => renderContent(manager)}</ConnectionManagement>
                </SensitiveWriteScope>
            ) : (
                renderContent()
            )}
        </AppPage>
    );
};
