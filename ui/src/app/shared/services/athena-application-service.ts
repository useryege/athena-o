import requests from './requests';

export interface ProjectView {
    meta?: ProjectMeta;
    aveDetail?: AveDetail;
}

export interface ProjectListItem {
    contract?: string;
    name?: string;
    symbol?: string;
    creator?: string;
    txHash?: string;
    hasMintRisk?: boolean;
    isOpenSource?: boolean;
    isReportEvaluated?: boolean;
    isReportComplete?: boolean;
    isBlacklistedCreatorWallet?: boolean;
    isBlacklistedGenesisWallet?: boolean;
    isBlacklistedBytecode?: boolean;
    wethPairQuoteUsdtValue?: string;
    wethPairRemoveLiquidity?: boolean;
    usdtPairQuoteUsdtValue?: string;
    usdtPairRemoveLiquidity?: boolean;
    creatorAssetUsdtValue?: string;
    blockTime?: number;
    blockNumber?: number;
    txIndex?: number;
    aveLogo?: string;
    aveDetailAvailable?: boolean;
    aveIsHoneypot?: boolean;
    aveHasMintMethod?: boolean;
    aveIsMintable?: string;
    aveHolders?: number;
    aveMarketCap?: string;
}

export interface ProjectBaseView {
    blockTime?: number;
    blockNumber?: number;
    contract?: string;
    creator?: string;
    txHash?: string;
    txIndex?: number;
    createdAt?: string;
}

export interface ProjectReport {
    isReportEvaluated?: boolean;
    isReportComplete?: boolean;
    isBlacklistedCreatorWallet?: boolean;
    isBlacklistedGenesisWallet?: boolean;
    isBlacklistedBytecode?: boolean;
    hasMintRisk?: boolean;
    evaluatedAt?: string;
    updatedAt?: string;
}

export interface ProjectChainState {
    contract?: string;
    fetchedAt?: string;
    token?: TokenState;
    wethPair?: PairV2State;
    usdtPair?: PairV2State;
    assetState?: AssetState;
    genesisWalletAssetStates?: GenesisWalletAssetState[];
}

export interface ProjectMeta {
    blockTime?: number;
    blockNumber?: number;
    contract?: string;
    creator?: string;
    txHash?: string;
    txIndex?: number;
    creatorResult?: SimulateResult;
    genesisWallets?: GenesisWalletState[];
    creatorHistoricalProjects?: string[];
    genesisWalletsFetchedAt?: string;
    creatorHistoricalProjectsFetchedAt?: string;
    fetchAt?: string;
    token?: TokenState;
    wethPair?: PairV2State;
    usdtPair?: PairV2State;
    assetState?: AssetState;
    genesisWalletAssetStates?: GenesisWalletAssetState[];
}

export interface AveDetail {
    status?: number;
    msg?: string;
    dataType?: number;
    isAudited?: boolean;
    fetchedAt?: string;
    token?: AveTokenDetail;
    pairs?: AvePair[];
}

export interface ProjectAveState {
    contract?: string;
    detailAvailable?: boolean;
    detail?: AveDetail;
    status?: string;
    lastAttemptAt?: string;
    lastSuccessAt?: string;
    nextRunAt?: string;
    lastError?: string;
    stale?: boolean;
}

export interface AveTokenDetail {
    total?: string;
    launchPrice?: string;
    currentPriceEth?: string;
    currentPriceUsd?: string;
    priceChange1d?: string;
    priceChange24h?: string;
    priceChange1h?: string;
    lockAmount?: string;
    burnAmount?: string;
    otherAmount?: string;
    txAmount24h?: string;
    txVolumeU24h?: string;
    lockedPercent?: string;
    marketCap?: string;
    fdv?: string;
    tvl?: string;
    mainPairTvl?: string;
    tokenPriceChange5m?: string;
    tokenPriceChange1h?: string;
    tokenPriceChange4h?: string;
    tokenPriceChange24h?: string;
    tokenTxVolumeUsd5m?: string;
    tokenTxVolumeUsd1h?: string;
    tokenTxVolumeUsd4h?: string;
    tokenTxVolumeUsd24h?: string;
    tokenBuyVolumeU5m?: string;
    tokenSellVolumeU5m?: string;
    token?: string;
    chain?: string;
    decimal?: number;
    name?: string;
    symbol?: string;
    holders?: number;
    appendix?: string;
    riskLevel?: number;
    logoUrl?: string;
    riskInfo?: string;
    riskScore?: string;
    launchAt?: number;
    createdAt?: number;
    txCount24h?: number;
    lockPlatform?: string;
    isMintable?: string;
    updatedAt?: number;
    mainPair?: string;
    hasMintMethod?: boolean;
    isLpNotLocked?: boolean;
    hasNotRenounced?: boolean;
    hasNotAudited?: boolean;
    hasNotOpenSource?: boolean;
    isInBlacklist?: boolean;
    isHoneypot?: boolean;
    aveRiskLevel?: number;
}

export interface AvePair {
    reserve0?: string;
    reserve1?: string;
    token0PriceEth?: string;
    token0PriceUsd?: string;
    token1PriceEth?: string;
    token1PriceUsd?: string;
    priceChange?: string;
    priceChange24h?: string;
    priceChange1h?: string;
    volumeU?: string;
    lowU?: string;
    highU?: string;
    fee?: string;
    totalSupply?: string;
    txAmount?: string;
    pair?: string;
    chain?: string;
    amm?: string;
    token0Address?: string;
    token0Symbol?: string;
    token0Decimal?: number;
    token1Address?: string;
    token1Symbol?: string;
    token1Decimal?: number;
    targetToken?: string;
    priceChange1d?: string;
    createdAt?: number;
    txCount?: number;
    updatedAt?: number;
    marketCap?: string;
    fdv?: string;
    isFake?: boolean;
}

export interface GenesisWalletState {
    wallet?: string;
    netAmount?: string;
    ratioBps?: number;
    rank?: number;
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

export interface GetProjectBaseResponse {
    item?: ProjectBaseView;
}

export interface GetProjectReportResponse {
    item?: ProjectReport;
}

export interface GetProjectChainStateResponse {
    item?: ProjectChainState;
}

export interface GetProjectSimulationResponse {
    item?: SimulateResult;
}

export interface GetProjectAveStateResponse {
    item?: ProjectAveState;
}

export interface RefreshProjectAveDetailResponse {
    item?: ProjectAveState;
}

export interface ListProjectGenesisWalletsResponse {
    items?: GenesisWalletState[];
}

export interface ListProjectCreatorHistoricalProjectsResponse {
    items?: string[];
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
        const baseReq = this.getProjectBase(contract);
        const chainReq = this.getProjectChainState(contract);
        const simulationReq = this.getProjectSimulation(contract);
        const aveReq = this.getProjectAveState(contract);
        const genesisReq = this.listProjectGenesisWallets(contract);
        const historyReq = this.listProjectCreatorHistoricalProjects(contract);
        const promise = Promise.all([baseReq, chainReq, simulationReq, aveReq, genesisReq, historyReq]).then(
            ([base, chainState, simulation, aveState, genesisWallets, creatorHistoricalProjects]) =>
                this.buildProjectView(base, chainState, simulation, aveState, genesisWallets, creatorHistoricalProjects)
        ) as any;
        promise.abort = () => {
            baseReq.abort?.();
            chainReq.abort?.();
            simulationReq.abort?.();
            aveReq.abort?.();
            genesisReq.abort?.();
            historyReq.abort?.();
        };
        return promise;
    }

    public getProjectBase(contract: string): Promise<ProjectBaseView | undefined> & {abort?: () => void} {
        const req = requests.get(`/projects/${encodeURIComponent(contract)}/base`);
        const promise = req.then(res => ((res.body || {}) as GetProjectBaseResponse).item) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public getProjectReport(contract: string): Promise<ProjectReport | undefined> & {abort?: () => void} {
        const req = requests.get(`/projects/${encodeURIComponent(contract)}/report`);
        const promise = req.then(res => ((res.body || {}) as GetProjectReportResponse).item) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public getProjectChainState(contract: string): Promise<ProjectChainState | undefined> & {abort?: () => void} {
        const req = requests.get(`/projects/${encodeURIComponent(contract)}/chain-state`);
        const promise = req.then(res => ((res.body || {}) as GetProjectChainStateResponse).item) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public getProjectSimulation(contract: string): Promise<SimulateResult | undefined> & {abort?: () => void} {
        const req = requests.get(`/projects/${encodeURIComponent(contract)}/simulation`);
        const promise = req.then(res => ((res.body || {}) as GetProjectSimulationResponse).item) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public getProjectAveState(contract: string): Promise<ProjectAveState | undefined> & {abort?: () => void} {
        const req = requests.get(`/projects/${encodeURIComponent(contract)}/ave`);
        const promise = req.then(res => ((res.body || {}) as GetProjectAveStateResponse).item) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public refreshProjectAveDetail(contract: string): Promise<ProjectAveState | undefined> & {abort?: () => void} {
        const req = requests.post(`/projects/${encodeURIComponent(contract)}/ave/refresh`).send({contract});
        const promise = req.then(res => ((res.body || {}) as RefreshProjectAveDetailResponse).item) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listProjectGenesisWallets(contract: string): Promise<GenesisWalletState[]> & {abort?: () => void} {
        const req = requests.get(`/projects/${encodeURIComponent(contract)}/genesis-wallets`);
        const promise = req.then(res => ((res.body || {}) as ListProjectGenesisWalletsResponse).items || []) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listProjectCreatorHistoricalProjects(contract: string): Promise<string[]> & {abort?: () => void} {
        const req = requests.get(`/projects/${encodeURIComponent(contract)}/creator-history`);
        const promise = req.then(res => ((res.body || {}) as ListProjectCreatorHistoricalProjectsResponse).items || []) as any;
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

    private buildProjectView(
        base?: ProjectBaseView,
        chainState?: ProjectChainState,
        simulation?: SimulateResult,
        aveState?: ProjectAveState,
        genesisWallets?: GenesisWalletState[],
        creatorHistoricalProjects?: string[]
    ): ProjectView {
        const meta: ProjectMeta = {
            blockTime: base?.blockTime,
            blockNumber: base?.blockNumber,
            contract: base?.contract,
            creator: base?.creator,
            txHash: base?.txHash,
            txIndex: base?.txIndex,
            fetchAt: chainState?.fetchedAt || base?.createdAt,
            creatorResult: simulation,
            genesisWallets,
            creatorHistoricalProjects,
            token: chainState?.token,
            wethPair: chainState?.wethPair,
            usdtPair: chainState?.usdtPair,
            assetState: chainState?.assetState,
            genesisWalletAssetStates: chainState?.genesisWalletAssetStates
        };
        return {meta, aveDetail: aveState?.detail};
    }
}
