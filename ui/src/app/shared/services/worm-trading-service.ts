import {AccountDataModule} from '../access-modules';
import requests from './requests';

const readScope = {module: AccountDataModule.WormTrading, mode: 'read' as const};

export type AbortableWormTradingPromise<T> = Promise<T> & {abort?: () => void};
export type WormTradingWalletBalanceStatus = 'COMPLETE' | 'PARTIAL' | 'UNAVAILABLE';

export interface WormTradingStatus {
    started: boolean;
    status: string;
    network: string;
    commitment: string;
    rpcReachable: boolean;
    batchSupported: boolean;
    genesisVerified: boolean;
    genesisHash: string;
    usdcMint: string;
    usdcVerified: boolean;
    latestConfirmedSlot: string;
    lastProbeAt: number;
    lastSuccessAt: number;
    latencyMS: number;
    consecutiveFailures: number;
    lastErrorCategory: string;
}

export interface WormTradingWalletSummary {
    walletId: number;
    address: string;
    remark: string;
    avatarKind: string;
    avatarPresetId: string;
    avatarUrl: string;
}

export interface WormTradingAssetBalance {
    atomicAmount: string;
    amount: string;
    decimals: number;
    observedSlot: string;
    availability: string;
    errorCode: string;
}

export interface WormTradingTokenAssetBalance extends WormTradingAssetBalance {
    mint: string;
    tokenAccountCount: number;
}

export interface WormTradingWalletBalanceItem {
    wallet: WormTradingWalletSummary;
    sol: WormTradingAssetBalance;
    usdc: WormTradingTokenAssetBalance;
    status: WormTradingWalletBalanceStatus;
}

export interface ListWormTradingWalletBalancesResult {
    items: WormTradingWalletBalanceItem[];
    total: number;
    page: number;
    pageSize: number;
    network: string;
    commitment: string;
    fetchedAt: number;
}

const readValue = (item: any, ...names: string[]) => {
    for (const name of names) {
        if (item?.[name] !== undefined && item?.[name] !== null) {
            return item[name];
        }
    }
    return undefined;
};

const readString = (item: any, ...names: string[]) => String(readValue(item, ...names) ?? '');
const readNumber = (item: any, ...names: string[]) => Number(readValue(item, ...names) ?? 0) || 0;
const readBoolean = (item: any, ...names: string[]) => {
    const value = readValue(item, ...names);
    return value === true || value === 1 || String(value).toLowerCase() === 'true';
};

const normalizeBalanceStatus = (value: unknown): WormTradingWalletBalanceStatus => {
    switch (String(value || '').toUpperCase()) {
        case 'COMPLETE':
            return 'COMPLETE';
        case 'PARTIAL':
            return 'PARTIAL';
        default:
            return 'UNAVAILABLE';
    }
};

const normalizeAssetBalance = (item: any = {}): WormTradingAssetBalance => ({
    atomicAmount: readString(item, 'atomicAmount', 'atomic_amount'),
    amount: readString(item, 'amount'),
    decimals: readNumber(item, 'decimals'),
    observedSlot: readString(item, 'observedSlot', 'observed_slot'),
    availability: readString(item, 'availability').toUpperCase() || 'UNAVAILABLE',
    errorCode: readString(item, 'errorCode', 'error_code')
});

const normalizeWalletBalance = (item: any = {}): WormTradingWalletBalanceItem => {
    const wallet = item.wallet || {};
    const usdc = item.usdc || {};
    return {
        wallet: {
            walletId: readNumber(wallet, 'walletId', 'wallet_id'),
            address: readString(wallet, 'address'),
            remark: readString(wallet, 'remark'),
            avatarKind: readString(wallet, 'avatarKind', 'avatar_kind'),
            avatarPresetId: readString(wallet, 'avatarPresetId', 'avatar_preset_id'),
            avatarUrl: readString(wallet, 'avatarUrl', 'avatar_url')
        },
        sol: normalizeAssetBalance(item.sol),
        usdc: {
            ...normalizeAssetBalance(usdc),
            mint: readString(usdc, 'mint'),
            tokenAccountCount: readNumber(usdc, 'tokenAccountCount', 'token_account_count')
        },
        status: normalizeBalanceStatus(readValue(item, 'status'))
    };
};

const abortableRequest = <T>(request: any, map: (body: any) => T): AbortableWormTradingPromise<T> => {
    const promise = request.then((response: any) => map(response.body || {})) as AbortableWormTradingPromise<T>;
    promise.abort = () => request.abort();
    return promise;
};

export class WormTradingService {
    public getStatus(): AbortableWormTradingPromise<WormTradingStatus> {
        const request = requests.get('/worm-trading/status', readScope);
        return abortableRequest(request, body => ({
            started: readBoolean(body, 'started'),
            status: readString(body, 'status'),
            network: readString(body, 'network'),
            commitment: readString(body, 'commitment'),
            rpcReachable: readBoolean(body, 'rpcReachable', 'rpc_reachable'),
            batchSupported: readBoolean(body, 'batchSupported', 'batch_supported'),
            genesisVerified: readBoolean(body, 'genesisVerified', 'genesis_verified'),
            genesisHash: readString(body, 'genesisHash', 'genesis_hash'),
            usdcMint: readString(body, 'usdcMint', 'usdc_mint'),
            usdcVerified: readBoolean(body, 'usdcVerified', 'usdc_verified'),
            latestConfirmedSlot: readString(body, 'latestConfirmedSlot', 'latest_confirmed_slot'),
            lastProbeAt: readNumber(body, 'lastProbeAt', 'last_probe_at'),
            lastSuccessAt: readNumber(body, 'lastSuccessAt', 'last_success_at'),
            latencyMS: readNumber(body, 'latencyMs', 'latency_ms'),
            consecutiveFailures: readNumber(body, 'consecutiveFailures', 'consecutive_failures'),
            lastErrorCategory: readString(body, 'lastErrorCategory', 'last_error_category')
        }));
    }

    public listWalletBalances(page = 1, pageSize = 20): AbortableWormTradingPromise<ListWormTradingWalletBalancesResult> {
        const request = requests.get('/worm-trading/wallet-balances', readScope).query({page, pageSize});
        return abortableRequest(request, body => ({
            items: (Array.isArray(body.items) ? body.items : []).map(normalizeWalletBalance),
            total: readNumber(body, 'total'),
            page: readNumber(body, 'page') || page,
            pageSize: readNumber(body, 'pageSize', 'page_size') || pageSize,
            network: readString(body, 'network'),
            commitment: readString(body, 'commitment'),
            fetchedAt: readNumber(body, 'fetchedAt', 'fetched_at')
        }));
    }
}
