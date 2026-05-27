import requests from './requests';

export type WalletChain = 'ETH' | 'BSC' | 'BASE' | 'SOLANA';

export interface WalletItem {
    id: number;
    chain: WalletChain | string;
    address: string;
    alias?: string;
    source?: string;
    derivationPath?: string;
    createdAt?: string;
    updatedAt?: string;
}

export interface WalletDetail extends WalletItem {
    privateKey?: string;
    mnemonic?: string;
}

export interface ListWalletsOptions {
    chain?: WalletChain | string;
    query?: string;
    page?: number;
    pageSize?: number;
}

export interface ListWalletsResult {
    items: WalletItem[];
    total: number;
    page: number;
    pageSize: number;
}

const readValue = (item: any, ...names: string[]) => {
    for (const name of names) {
        if (item?.[name] !== undefined && item?.[name] !== null) {
            return item[name];
        }
    }
    return undefined;
};

const readString = (item: any, ...names: string[]) => String(readValue(item, ...names) || '');
const readNumber = (item: any, ...names: string[]) => Number(readValue(item, ...names) || 0) || 0;

const normalizeWallet = (item: any = {}): WalletDetail => ({
    id: readNumber(item, 'id'),
    chain: readString(item, 'chain'),
    address: readString(item, 'address'),
    alias: readString(item, 'alias'),
    source: readString(item, 'source'),
    derivationPath: readString(item, 'derivationPath', 'derivation_path'),
    createdAt: readString(item, 'createdAt', 'created_at'),
    updatedAt: readString(item, 'updatedAt', 'updated_at'),
    privateKey: readString(item, 'privateKey', 'private_key'),
    mnemonic: readString(item, 'mnemonic')
});

export class WalletService {
    public listWallets(options: ListWalletsOptions = {}): Promise<ListWalletsResult> & {abort?: () => void} {
        const query: any = {
            page: options.page || 1,
            page_size: options.pageSize || 20
        };
        if (options.chain) {
            query.chain = options.chain;
        }
        if (options.query) {
            query.query = options.query;
        }
        const req = requests.get('/wallets').query(query);
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: (body.items || []).map(normalizeWallet),
                total: readNumber(body, 'total'),
                page: readNumber(body, 'page') || query.page,
                pageSize: readNumber(body, 'pageSize', 'page_size') || query.page_size
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public getWallet(id: number, revealSecrets = false): Promise<WalletDetail> & {abort?: () => void} {
        const req = requests.get(`/wallets/${encodeURIComponent(String(id))}`).query({reveal_secrets: revealSecrets});
        const promise = req.then(res => normalizeWallet((res.body || {}).item || {})) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public createWallet(chain: WalletChain | string, alias: string): Promise<WalletDetail> & {abort?: () => void} {
        const req = requests.post('/wallets/create').send({chain, alias});
        const promise = req.then(res => normalizeWallet((res.body || {}).item || {})) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public importPrivateKey(chain: WalletChain | string, privateKey: string, alias: string): Promise<WalletDetail> & {abort?: () => void} {
        const req = requests.post('/wallets/import-private-key').send({chain, private_key: privateKey, alias});
        const promise = req.then(res => normalizeWallet((res.body || {}).item || {})) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public importMnemonic(chain: WalletChain | string, mnemonic: string, alias: string): Promise<WalletDetail> & {abort?: () => void} {
        const req = requests.post('/wallets/import-mnemonic').send({chain, mnemonic, alias});
        const promise = req.then(res => normalizeWallet((res.body || {}).item || {})) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public updateAlias(id: number, alias: string): Promise<WalletItem> & {abort?: () => void} {
        const req = requests.post(`/wallets/${encodeURIComponent(String(id))}/alias`).send({id, alias});
        const promise = req.then(res => normalizeWallet((res.body || {}).item || {})) as any;
        promise.abort = () => req.abort();
        return promise;
    }
}
