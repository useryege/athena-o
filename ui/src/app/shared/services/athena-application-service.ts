import requests from './requests';

export interface ProjectView {
    meta?: ProjectMeta;
    chainState?: ProjectChainState;
}

export interface ProjectListItem {
    contract?: string;
    name?: string;
    symbol?: string;
    hasMintRisk?: boolean;
    isOpenSource?: boolean;
    wethPairQuoteUsdtValue?: string;
    wethPairRemoveLiquidity?: boolean;
    usdtPairQuoteUsdtValue?: string;
    usdtPairRemoveLiquidity?: boolean;
    creatorAssetUsdtValue?: string;
    blockTime?: number;
    blockNumber?: number;
    txIndex?: number;
}

export interface ProjectMeta {
    blockTime?: number;
    blockNumber?: number;
    contract?: string;
    creator?: string;
    txHash?: string;
    txIndex?: number;
    sourceCode?: string;
    creatorResult?: SimulateResult;
    genesisWallets?: GenesisWalletState[];
    creatorHistoricalProjects?: string[];
    sourceQualityReport?: string;
    sourceQualityReportFetchedAt?: string;
    sourceCodeFetchedAt?: string;
    codeBinHashFetchedAt?: string;
    genesisWalletsFetchedAt?: string;
    creatorHistoricalProjectsFetchedAt?: string;
    isOpenSource?: boolean;
    sourceCodeHash?: string;
    codeBinHash?: string;
}

export interface GenesisWalletState {
    wallet?: string;
    netAmount?: string;
    ratioBps?: number;
    rank?: number;
}

export interface ProjectChainState {
    token?: TokenState;
    wethPair?: PairV2State;
    usdtPair?: PairV2State;
    assetState?: AssetState;
    genesisWalletAssetStates?: GenesisWalletAssetState[];
}

export interface TokenState {
    name?: string;
    symbol?: string;
    decimals?: number;
    totalSupply?: string;
    isValidERC20?: boolean;
}

export interface PairV2State {
    isCreated?: boolean;
    contract?: string;
    token0?: string;
    token1?: string;
    totalSupply?: string;
    reserve0?: string;
    reserve1?: string;
    blockTimestampLast?: number;
    baseBalance?: string;
    quoteBalance?: string;
    quoteUsdtValue?: string;
    lockedLiquidity?: string;
    feeAddressHoldLiquidityBalance?: string;
    isRemoveLiquidity?: boolean;
    feeAddressHoldLiquidityRatio?: string;
}

export interface AssetState {
    tokenBalance?: string;
    wethBalance?: string;
    usdtBalance?: string;
    nativeBalance?: string;
    usdtValue?: string;
}

export interface GenesisWalletAssetState {
    wallet?: string;
    assetState?: AssetState;
}

export interface ListProjectsResponse {
    items?: ProjectListItem[];
    total?: number;
    page?: number;
    pageSize?: number;
}

export interface SimulateResult {
    canMintFromDeadViaTransferFrom?: boolean;
    canMintFromZeroViaTransferFrom?: boolean;
    canMintFromWethPairViaTransferFrom?: boolean;
    canMintFromUsdtPairViaTransferFrom?: boolean;
    canMintViaTransferToWethPair?: boolean;
    canMintViaTransferToUsdtPair?: boolean;
}

export interface GetProjectResponse {
    item?: ProjectView;
}

export interface ProjectEventLog {
    id?: number;
    contract?: string;
    eventType?: number;
    occurredAt?: string;
    message?: string;
    payload?: string;
    createdAt?: string;
}

export interface ListProjectEventLogsResponse {
    items?: ProjectEventLog[];
}

export interface ProjectComment {
    id?: number;
    contract?: string;
    username?: string;
    content?: string;
    createdAt?: string;
}

export interface AddProjectCommentResponse {
    item?: ProjectComment;
}

export interface ListProjectCommentsResponse {
    items?: ProjectComment[];
    total?: number;
    page?: number;
    pageSize?: number;
}

export interface ProjectOptions {
    factoryContract?: string;
    wethContract?: string;
    usdtContract?: string;
    wethDecimals?: number;
    usdtDecimals?: number;
}

export interface GetProjectOptionsResponse {
    options?: ProjectOptions;
}

let cachedProjectOptions: ProjectOptions | undefined;
let projectOptionsRequest: (Promise<ProjectOptions | undefined> & {abort?: () => void}) | null = null;
let cachedBytecodeBlacklistContracts: BytecodeBlacklistContract[] | undefined;
let bytecodeBlacklistContractsRequest: (Promise<BytecodeBlacklistContract[]> & {abort?: () => void}) | null = null;

export interface BytecodeBlacklistContract {
    contract?: string;
    codeHash?: string;
    note?: string;
    createdAt?: string;
}

export interface ListBytecodeBlacklistContractsResponse {
    items?: BytecodeBlacklistContract[];
}

export interface AddBytecodeBlacklistContractResponse {
    item?: BytecodeBlacklistContract;
}

export interface UpdateBytecodeBlacklistContractNoteResponse {
    item?: BytecodeBlacklistContract;
}

export interface SourcecodeBlacklistContract {
    contract?: string;
    sourceHash?: string;
    note?: string;
    createdAt?: string;
}

export interface ListSourcecodeBlacklistContractsResponse {
    items?: SourcecodeBlacklistContract[];
}

export interface AddSourcecodeBlacklistContractResponse {
    item?: SourcecodeBlacklistContract;
}

export interface UpdateSourcecodeBlacklistContractNoteResponse {
    item?: SourcecodeBlacklistContract;
}

export interface WalletBlacklistEntry {
    wallet?: string;
    note?: string;
    createdAt?: string;
}

export interface ListWalletBlacklistEntriesResponse {
    items?: WalletBlacklistEntry[];
}

export interface AddWalletBlacklistEntryResponse {
    item?: WalletBlacklistEntry;
}

export interface UpdateWalletBlacklistEntryNoteResponse {
    item?: WalletBlacklistEntry;
}

export class AthenaApplicationService {
    public listProjects(page = 1, pageSize = 20): Promise<{items: ProjectListItem[]; total: number; page: number; pageSize: number}> & {abort?: () => void} {
        const req = requests.get('/projects').query({page, pageSize});
        const promise = req.then(res => {
            const body = (res.body || {}) as ListProjectsResponse;
            return {
                items: body.items || [],
                total: body.total || 0,
                page: body.page || page,
                pageSize: body.pageSize || pageSize
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public getProject(contract: string): Promise<ProjectView> & {abort?: () => void} {
        const req = requests.get(`/projects/${encodeURIComponent(contract)}`);
        const promise = req.then(res => (res.body as GetProjectResponse).item) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listProjectEventLogs(contract: string): Promise<ProjectEventLog[]> & {abort?: () => void} {
        const req = requests.get(`/projects/${encodeURIComponent(contract)}/events`);
        const promise = req.then(res => {
            const body = (res.body || {}) as ListProjectEventLogsResponse;
            const items = body.items || [];
            return items.map(item => ({
                id: item.id,
                contract: item.contract,
                eventType: item.eventType ?? (item as any).event_type,
                occurredAt: item.occurredAt ?? (item as any).occurred_at,
                message: item.message,
                payload: item.payload,
                createdAt: item.createdAt ?? (item as any).created_at
            }));
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public addProjectComment(contract: string, content: string): Promise<ProjectComment | undefined> & {abort?: () => void} {
        const req = requests.post(`/projects/${encodeURIComponent(contract)}/comments`).send({contract, content});
        const promise = req.then(res => {
            const item = ((res.body || {}) as AddProjectCommentResponse).item;
            if (!item) {
                return undefined;
            }
            return {
                id: item.id,
                contract: item.contract,
                username: item.username,
                content: item.content,
                createdAt: item.createdAt ?? (item as any).created_at
            } as ProjectComment;
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listProjectComments(contract: string, page = 1, pageSize = 5): Promise<{items: ProjectComment[]; total: number; page: number; pageSize: number}> & {abort?: () => void} {
        const req = requests.get(`/projects/${encodeURIComponent(contract)}/comments`).query({page, pageSize});
        const promise = req.then(res => {
            const body = (res.body || {}) as ListProjectCommentsResponse;
            const items = (body.items || []).map(item => ({
                id: item.id,
                contract: item.contract,
                username: item.username,
                content: item.content,
                createdAt: item.createdAt ?? (item as any).created_at
            }));
            return {
                items,
                total: body.total || 0,
                page: body.page || page,
                pageSize: body.pageSize || pageSize
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public getProjectOptions(): Promise<ProjectOptions | undefined> & {abort?: () => void} {
        if (cachedProjectOptions) {
            const promise = Promise.resolve(cachedProjectOptions) as Promise<ProjectOptions | undefined> & {abort?: () => void};
            promise.abort = () => {};
            return promise;
        }
        if (projectOptionsRequest) {
            const promise = projectOptionsRequest as Promise<ProjectOptions | undefined> & {abort?: () => void};
            promise.abort = () => {};
            return promise;
        }
        const req = requests.get('/project-options');
        const promise = req
            .then(res => {
                const body = (res.body || {}) as any;
                const options = (body.options || body) as any;
                if (!options || typeof options !== 'object') {
                    return undefined;
                }
                return {
                    factoryContract: options.factoryContract ?? options.factory_contract,
                    wethContract: options.wethContract ?? options.weth_contract,
                    usdtContract: options.usdtContract ?? options.usdt_contract,
                    wethDecimals: typeof options.wethDecimals === 'number' ? options.wethDecimals : typeof options.weth_decimals === 'number' ? options.weth_decimals : undefined,
                    usdtDecimals: typeof options.usdtDecimals === 'number' ? options.usdtDecimals : typeof options.usdt_decimals === 'number' ? options.usdt_decimals : undefined
                } as ProjectOptions;
            })
            .then(
                options => {
                    cachedProjectOptions = options;
                    projectOptionsRequest = null;
                    return options;
                },
                err => {
                    projectOptionsRequest = null;
                    throw err;
                }
            ) as any;
        promise.abort = () => {};
        projectOptionsRequest = promise;
        return promise;
    }

    public listBytecodeBlacklistContracts(): Promise<BytecodeBlacklistContract[]> & {abort?: () => void} {
        if (cachedBytecodeBlacklistContracts) {
            const promise = Promise.resolve(cachedBytecodeBlacklistContracts) as Promise<BytecodeBlacklistContract[]> & {abort?: () => void};
            promise.abort = () => {};
            return promise;
        }
        if (bytecodeBlacklistContractsRequest) {
            const promise = bytecodeBlacklistContractsRequest as Promise<BytecodeBlacklistContract[]> & {abort?: () => void};
            promise.abort = () => {};
            return promise;
        }
        const req = requests.get('/bytecode/blacklist-contracts');
        const promise = req
            .then(res => {
                const body = (res.body as ListBytecodeBlacklistContractsResponse) || {};
                const items = body.items || [];
                return items.map(item => ({
                    contract: item.contract,
                    codeHash: item.codeHash ?? (item as any).code_hash,
                    note: item.note,
                    createdAt: item.createdAt ?? (item as any).created_at
                }));
            })
            .then(
                items => {
                    cachedBytecodeBlacklistContracts = items;
                    bytecodeBlacklistContractsRequest = null;
                    return items;
                },
                err => {
                    bytecodeBlacklistContractsRequest = null;
                    throw err;
                }
            ) as any;
        promise.abort = () => {};
        bytecodeBlacklistContractsRequest = promise;
        return promise;
    }

    public addBytecodeBlacklistContract(contract: string, note: string): Promise<BytecodeBlacklistContract> & {abort?: () => void} {
        const req = requests.post('/bytecode/blacklist-contracts').send({contract, note});
        const promise = req
            .then(res => {
                const item = ((res.body as AddBytecodeBlacklistContractResponse) || {}).item || {};
                return {
                    contract: item.contract,
                    codeHash: item.codeHash ?? (item as any).code_hash,
                    note: item.note,
                    createdAt: item.createdAt ?? (item as any).created_at
                } as BytecodeBlacklistContract;
            })
            .then(item => {
                cachedBytecodeBlacklistContracts = undefined;
                return item;
            }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public updateBytecodeBlacklistContractNote(contract: string, note: string): Promise<BytecodeBlacklistContract> & {abort?: () => void} {
        const req = requests.post(`/bytecode/blacklist-contracts/${encodeURIComponent(contract)}/note`).send({contract, note});
        const promise = req
            .then(res => {
                const item = ((res.body as UpdateBytecodeBlacklistContractNoteResponse) || {}).item || {};
                return {
                    contract: item.contract,
                    codeHash: item.codeHash ?? (item as any).code_hash,
                    note: item.note,
                    createdAt: item.createdAt ?? (item as any).created_at
                } as BytecodeBlacklistContract;
            })
            .then(item => {
                cachedBytecodeBlacklistContracts = undefined;
                return item;
            }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public deleteBytecodeBlacklistContract(contract: string): Promise<void> & {abort?: () => void} {
        const req = requests.delete(`/bytecode/blacklist-contracts/${encodeURIComponent(contract)}`);
        const promise = req.then(() => {
            cachedBytecodeBlacklistContracts = undefined;
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listSourcecodeBlacklistContracts(): Promise<SourcecodeBlacklistContract[]> & {abort?: () => void} {
        const req = requests.get('/source-code/blacklist-contracts');
        const promise = req.then(res => {
            const body = (res.body as ListSourcecodeBlacklistContractsResponse) || {};
            const items = body.items || [];
            return items.map(item => ({
                contract: item.contract,
                sourceHash: item.sourceHash ?? (item as any).source_hash,
                note: item.note,
                createdAt: item.createdAt ?? (item as any).created_at
            }));
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public addSourcecodeBlacklistContract(contract: string, note: string): Promise<SourcecodeBlacklistContract> & {abort?: () => void} {
        const req = requests.post('/source-code/blacklist-contracts').send({contract, note});
        const promise = req.then(res => {
            const item = ((res.body as AddSourcecodeBlacklistContractResponse) || {}).item || {};
            return {
                contract: item.contract,
                sourceHash: item.sourceHash ?? (item as any).source_hash,
                note: item.note,
                createdAt: item.createdAt ?? (item as any).created_at
            } as SourcecodeBlacklistContract;
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public updateSourcecodeBlacklistContractNote(contract: string, note: string): Promise<SourcecodeBlacklistContract> & {abort?: () => void} {
        const req = requests.post(`/source-code/blacklist-contracts/${encodeURIComponent(contract)}/note`).send({contract, note});
        const promise = req.then(res => {
            const item = ((res.body as UpdateSourcecodeBlacklistContractNoteResponse) || {}).item || {};
            return {
                contract: item.contract,
                sourceHash: item.sourceHash ?? (item as any).source_hash,
                note: item.note,
                createdAt: item.createdAt ?? (item as any).created_at
            } as SourcecodeBlacklistContract;
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public deleteSourcecodeBlacklistContract(contract: string): Promise<void> & {abort?: () => void} {
        const req = requests.delete(`/source-code/blacklist-contracts/${encodeURIComponent(contract)}`);
        const promise = req.then(() => {}) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listWalletBlacklistEntries(): Promise<WalletBlacklistEntry[]> & {abort?: () => void} {
        const req = requests.get('/wallet-blacklist');
        const promise = req.then(res => {
            const body = (res.body as ListWalletBlacklistEntriesResponse) || {};
            const items = body.items || [];
            return items.map(item => ({
                wallet: item.wallet,
                note: item.note,
                createdAt: item.createdAt
            }));
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public addWalletBlacklistEntry(wallet: string, note: string): Promise<WalletBlacklistEntry> & {abort?: () => void} {
        const req = requests.post('/wallet-blacklist').send({wallet, note});
        const promise = req.then(res => {
            const item = ((res.body as AddWalletBlacklistEntryResponse) || {}).item || {};
            return {
                wallet: item.wallet,
                note: item.note,
                createdAt: item.createdAt
            } as WalletBlacklistEntry;
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public updateWalletBlacklistEntryNote(wallet: string, note: string): Promise<WalletBlacklistEntry> & {abort?: () => void} {
        const req = requests.post(`/wallet-blacklist/${encodeURIComponent(wallet)}/note`).send({wallet, note});
        const promise = req.then(res => {
            const item = ((res.body as UpdateWalletBlacklistEntryNoteResponse) || {}).item || {};
            return {
                wallet: item.wallet,
                note: item.note,
                createdAt: item.createdAt
            } as WalletBlacklistEntry;
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public deleteWalletBlacklistEntry(wallet: string): Promise<void> & {abort?: () => void} {
        const req = requests.delete(`/wallet-blacklist/${encodeURIComponent(wallet)}`);
        const promise = req.then(() => {}) as any;
        promise.abort = () => req.abort();
        return promise;
    }
}
