import {CopyOutlined, LinkOutlined, WalletOutlined} from '@ant-design/icons';
import {Alert, Avatar, Button, Card, Empty, Pagination, Result, Skeleton, Tag, Tooltip, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {useNavigate} from 'react-router-dom';
import {AppPage, ResourceTable, useCachedAsyncData} from '../components';
import {AccountDataModule} from '../shared/access-modules';
import {Context, useAuthorization} from '../shared/context';
import {formatBeijingUnixSeconds} from '../shared/format';
import {services, WormTradingAssetBalance, WormTradingStatus, WormTradingTokenAssetBalance, WormTradingWalletBalanceItem} from '../shared/services';
import {requestErrorMessage} from '../shared/services/requests';
import {usePagedParams} from './shared';

const wormTradingPageSize = 20;
const wormTradingPageSizes = [wormTradingPageSize];

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

const assetAvailable = (asset?: WormTradingAssetBalance) => asset?.availability === 'AVAILABLE' || asset?.availability === 'BALANCE_AVAILABILITY_AVAILABLE';

const WormTradingWalletAvatar = (props: {item: WormTradingWalletBalanceItem; size?: number}) => {
    const wallet = props.item.wallet;
    const presetGlyph = walletPresetGlyphs[wallet.avatarPresetId];
    const palette = defaultAvatarPalettes[hashWalletAddress(wallet.address) % defaultAvatarPalettes.length];
    const uploaded = wallet.avatarKind.toLowerCase() === 'upload' && wallet.avatarUrl;
    const className = ['wallet-avatar', presetGlyph ? `wallet-avatar--${wallet.avatarPresetId}` : 'wallet-avatar--generated'].join(' ');
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

const AssetValue = (props: {asset: WormTradingAssetBalance; symbol: 'SOL' | 'USDC'; token?: WormTradingTokenAssetBalance}) => {
    const available = assetAvailable(props.asset);
    const amount = props.asset.amount || '0';
    const detail = available
        ? [
              props.token ? `${props.token.tokenAccountCount} token ${props.token.tokenAccountCount === 1 ? 'account' : 'accounts'}` : undefined,
              `slot ${formatIntegerString(props.asset.observedSlot)}`
          ]
              .filter(Boolean)
              .join(' · ')
        : titleCase(props.asset.errorCode || props.asset.availability) || 'Balance unavailable';
    const exact = available ? `${props.asset.atomicAmount || '0'} atomic units · ${props.asset.decimals} decimals` : detail;
    return (
        <div className={available ? 'worm-trading-asset' : 'worm-trading-asset worm-trading-asset--unavailable'}>
            <Tooltip title={exact}>
                <strong>{available ? `${amount} ${props.symbol}` : 'Unavailable'}</strong>
            </Tooltip>
            <small>{detail}</small>
        </div>
    );
};

const WalletIdentity = (props: {item: WormTradingWalletBalanceItem; onCopy: () => void; compact?: boolean}) => (
    <div className={props.compact ? 'worm-trading-wallet worm-trading-wallet--compact' : 'worm-trading-wallet'}>
        <WormTradingWalletAvatar item={props.item} size={props.compact ? 42 : 46} />
        <div className='worm-trading-wallet__main'>
            <Typography.Text strong={true} ellipsis={{tooltip: props.item.wallet.remark || 'Solana wallet'}}>
                {props.item.wallet.remark || 'Solana wallet'}
            </Typography.Text>
            <span className='worm-trading-wallet__address'>
                <Tooltip title={props.item.wallet.address}>
                    <code>{shortAddress(props.item.wallet.address)}</code>
                </Tooltip>
                <Tooltip title='Copy address'>
                    <Button type='text' size='small' aria-label={`Copy ${props.item.wallet.remark || 'Solana wallet'} address`} icon={<CopyOutlined />} onClick={props.onCopy} />
                </Tooltip>
            </span>
        </div>
    </div>
);

const RuntimeSummary = (props: {status?: WormTradingStatus; loading: boolean; fallbackNetwork?: string; fallbackCommitment?: string}) => {
    if (props.loading && !props.status) {
        return (
            <section className='worm-trading-runtime' aria-label='Loading Worm Trading runtime status'>
                <Skeleton active={true} paragraph={{rows: 2}} />
            </section>
        );
    }

    const status = props.status;
    const runtimeState = status?.status.toLowerCase() || '';
    const ready = Boolean(status?.started && runtimeState === 'running' && status.rpcReachable && status.batchSupported && status.genesisVerified && status.usdcVerified);
    const state = !status ? 'Status unavailable' : !status.started ? 'Stopped' : runtimeState === 'configuration_error' ? 'Configuration error' : ready ? 'Ready' : 'Degraded';
    const stateColor = ready ? 'green' : !status?.started || runtimeState === 'configuration_error' ? 'red' : 'gold';
    return (
        <section className='worm-trading-runtime' aria-labelledby='worm-trading-runtime-heading'>
            <div className='worm-trading-runtime__heading'>
                <span>
                    <LinkOutlined aria-hidden='true' />
                    <Typography.Text id='worm-trading-runtime-heading' strong={true}>
                        Solana connection
                    </Typography.Text>
                </span>
                <Tag color={stateColor}>{state}</Tag>
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
                    <dt>USDC mint</dt>
                    <dd title={status?.usdcMint}>{shortAddress(status?.usdcMint, 7, 7)}</dd>
                </div>
            </dl>
            {status?.lastErrorCategory && !ready && <Typography.Text className='worm-trading-runtime__error'>Latest issue: {titleCase(status.lastErrorCategory)}</Typography.Text>}
        </section>
    );
};

const WalletBalanceCard = (props: {item: WormTradingWalletBalanceItem; onCopy: () => void}) => (
    <Card className='worm-trading-balance-card' size='small'>
        <div className='worm-trading-balance-card__header'>
            <WalletIdentity item={props.item} compact={true} onCopy={props.onCopy} />
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
        {props.item.usdc.mint && (
            <div className='worm-trading-balance-card__mint'>
                <span>Mint</span>
                <code title={props.item.usdc.mint}>{shortAddress(props.item.usdc.mint, 7, 7)}</code>
            </div>
        )}
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

export const WormTradingPage = () => {
    const ctx = React.useContext(Context);
    const authorization = useAuthorization();
    const {page, pageSize, setPage} = usePagedParams(wormTradingPageSize, wormTradingPageSizes);
    const currentPage = Number.isSafeInteger(page) && page > 0 ? page : 1;
    const accountID = authorization.user.accountId;
    const runtime = useCachedAsyncData(`worm-trading:status:${accountID}`, () => services.wormTrading.getStatus(), {
        staleTimeMs: 0,
        module: AccountDataModule.WormTrading
    });
    const balances = useCachedAsyncData(
        `worm-trading:wallet-balances:${accountID}:${currentPage}:${pageSize}`,
        () => services.wormTrading.listWalletBalances(currentPage, pageSize),
        {staleTimeMs: 0, module: AccountDataModule.WormTrading}
    );

    React.useEffect(() => {
        if (!balances.data) {
            return;
        }
        const lastPage = Math.max(1, Math.ceil(balances.data.total / pageSize));
        if (currentPage > lastPage) {
            setPage(lastPage, pageSize);
        }
    }, [balances.data, currentPage, pageSize, setPage]);

    const copyAddress = React.useCallback(
        async (item: WormTradingWalletBalanceItem) => {
            try {
                await navigator.clipboard.writeText(item.wallet.address);
                ctx.notifications.success('Address copied');
            } catch {
                ctx.notifications.error('Could not copy address', 'Select the address and copy it manually.');
            }
        },
        [ctx.notifications]
    );

    const columns: ColumnsType<WormTradingWalletBalanceItem> = [
        {
            title: 'Wallet',
            key: 'wallet',
            width: 390,
            render: (_, item) => <WalletIdentity item={item} onCopy={() => void copyAddress(item)} />
        },
        {
            title: 'SOL',
            key: 'sol',
            width: 190,
            render: (_, item) => <AssetValue asset={item.sol} symbol='SOL' />
        },
        {
            title: 'USDC',
            key: 'usdc',
            width: 220,
            render: (_, item) => <AssetValue asset={item.usdc} symbol='USDC' token={item.usdc} />
        },
        {
            title: 'Balance status',
            key: 'status',
            width: 140,
            render: (_, item) => <BalanceStatusTag status={item.status} />
        }
    ];

    const refresh = () => {
        runtime.reload();
        balances.reload();
    };
    const loading = runtime.loading || runtime.refreshing || balances.loading || balances.refreshing;
    const error = balances.error || runtime.error;
    const fetchedAt = formatBeijingUnixSeconds(balances.data?.fetchedAt);
    const items = balances.data?.items || [];
    const initialBalanceError = balances.error && !balances.data;

    return (
        <AppPage
            title='Worm Trading'
            subtitle='Review confirmed on-chain SOL and Circle native USDC balances. These values are not Worm collateral or available-to-order limits.'
            loading={loading}
            error={initialBalanceError ? undefined : error}
            onRefresh={refresh}>
            <RuntimeSummary status={runtime.data} loading={runtime.loading} fallbackNetwork={balances.data?.network} fallbackCommitment={balances.data?.commitment} />

            {initialBalanceError ? (
                <Result
                    status='error'
                    title='Wallet balances are unavailable'
                    subTitle={requestErrorMessage(initialBalanceError, 'The balance request failed. No wallet was treated as empty or zero.')}
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
                                {fetchedAt ? ` · Fetched ${fetchedAt}` : ''}
                            </Typography.Text>
                        </div>
                        {balances.refreshing && (
                            <Typography.Text className='worm-trading-balances__refreshing' role='status' aria-live='polite'>
                                Refreshing balances…
                            </Typography.Text>
                        )}
                    </div>
                    <ResourceTable<WormTradingWalletBalanceItem>
                        rowKey={item => item.wallet.walletId || item.wallet.address}
                        label='Worm Trading wallet balances'
                        items={items}
                        loading={balances.loading && !balances.data}
                        columns={columns}
                        scrollX={940}
                        compactRender={item => <WalletBalanceCard item={item} onCopy={() => void copyAddress(item)} />}
                        compactEmptyDescription='No Solana wallets are available.'
                    />
                    {(balances.data?.total || 0) > pageSize && (
                        <Pagination
                            className='worm-trading-pagination'
                            responsive={true}
                            current={currentPage}
                            pageSize={pageSize}
                            total={balances.data?.total || 0}
                            showSizeChanger={false}
                            showTotal={total => `${total} wallets`}
                            onChange={nextPage => setPage(nextPage, pageSize)}
                        />
                    )}
                </section>
            )}

            {balances.data && balances.data.total > 0 && items.length === 0 && !balances.loading && (
                <Alert
                    type='warning'
                    showIcon={true}
                    title='This page contains no wallet rows.'
                    description='Move to an earlier page or refresh after your wallet inventory changes.'
                />
            )}
        </AppPage>
    );
};
