import requests from './requests';

export interface ProjectView {
    meta?: ProjectMeta;
    chainState?: ProjectChainState;
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
    sourceCodeBlacklist?: SourceCodeBlacklistState;
    isArchived?: boolean;
    genesisWallets?: GenesisWalletState[];
    creatorOtherProjectContracts?: string[];
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
    items?: ProjectView[];
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

export interface SourceCodeBlacklistState {
    hasBlacklistFields?: boolean;
    blacklistFields?: string[];
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

export const PROJECT_SCOPE = {
    UNSPECIFIED: 0,
    ACTIVE: 1,
    ARCHIVED: 2,
    ALL: 3
} as const;

export type ProjectScope = (typeof PROJECT_SCOPE)[keyof typeof PROJECT_SCOPE];

export interface SourceCodeBlacklistField {
    id?: number;
    field?: string;
}

export interface ListSourceCodeBlacklistFieldsResponse {
    items?: SourceCodeBlacklistField[];
}

export interface AddSourceCodeBlacklistFieldResponse {
    item?: SourceCodeBlacklistField;
}

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
    public listProjects(
        scope: ProjectScope = PROJECT_SCOPE.ACTIVE,
        page = 1,
        pageSize = 20
    ): Promise<{items: ProjectView[]; total: number; page: number; pageSize: number}> & {abort?: () => void} {
        const req = requests.get('/projects').query({scope, page, pageSize});
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
        const req = requests.get('/project-options');
        const promise = req.then(res => {
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
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listSourceCodeBlacklistFields(): Promise<SourceCodeBlacklistField[]> & {abort?: () => void} {
        const req = requests.get('/source-code/blacklist-fields');
        const promise = req.then(res => (res.body as ListSourceCodeBlacklistFieldsResponse).items || []) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public addSourceCodeBlacklistField(field: string): Promise<SourceCodeBlacklistField> & {abort?: () => void} {
        const req = requests.post('/source-code/blacklist-fields').send({field});
        const promise = req.then(res => (res.body as AddSourceCodeBlacklistFieldResponse).item) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public deleteSourceCodeBlacklistField(field: string): Promise<void> & {abort?: () => void} {
        const req = requests.delete(`/source-code/blacklist-fields/${encodeURIComponent(field)}`);
        const promise = req.then(() => {}) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listBytecodeBlacklistContracts(): Promise<BytecodeBlacklistContract[]> & {abort?: () => void} {
        const req = requests.get('/bytecode/blacklist-contracts');
        const promise = req.then(res => {
            const body = (res.body as ListBytecodeBlacklistContractsResponse) || {};
            const items = body.items || [];
            return items.map(item => ({
                contract: item.contract,
                codeHash: item.codeHash ?? (item as any).code_hash,
                note: item.note,
                createdAt: item.createdAt ?? (item as any).created_at
            }));
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public addBytecodeBlacklistContract(contract: string, note: string): Promise<BytecodeBlacklistContract> & {abort?: () => void} {
        const req = requests.post('/bytecode/blacklist-contracts').send({contract, note});
        const promise = req.then(res => {
            const item = ((res.body as AddBytecodeBlacklistContractResponse) || {}).item || {};
            return {
                contract: item.contract,
                codeHash: item.codeHash ?? (item as any).code_hash,
                note: item.note,
                createdAt: item.createdAt ?? (item as any).created_at
            } as BytecodeBlacklistContract;
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public updateBytecodeBlacklistContractNote(contract: string, note: string): Promise<BytecodeBlacklistContract> & {abort?: () => void} {
        const req = requests.post(`/bytecode/blacklist-contracts/${encodeURIComponent(contract)}/note`).send({contract, note});
        const promise = req.then(res => {
            const item = ((res.body as UpdateBytecodeBlacklistContractNoteResponse) || {}).item || {};
            return {
                contract: item.contract,
                codeHash: item.codeHash ?? (item as any).code_hash,
                note: item.note,
                createdAt: item.createdAt ?? (item as any).created_at
            } as BytecodeBlacklistContract;
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public deleteBytecodeBlacklistContract(contract: string): Promise<void> & {abort?: () => void} {
        const req = requests.delete(`/bytecode/blacklist-contracts/${encodeURIComponent(contract)}`);
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

    public archiveProject(contract: string): Promise<void> & {abort?: () => void} {
        const req = requests.post(`/project/${encodeURIComponent(contract)}/archive`).send({contract});
        const promise = req.then(() => {}) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public unarchiveProject(contract: string): Promise<void> & {abort?: () => void} {
        const req = requests.post(`/project/${encodeURIComponent(contract)}/unarchive`).send({contract});
        const promise = req.then(() => {}) as any;
        promise.abort = () => req.abort();
        return promise;
    }
}
