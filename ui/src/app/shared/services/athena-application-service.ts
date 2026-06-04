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

export interface ProjectDiscoveryStatus {
    started: boolean;
    status: string;
}

export interface ContractSourceInfo {
    contract?: string;
    chainID?: number;
    codeBinHash?: string;
    sourceCode?: string;
    sourceCodeHash?: string;
    sourceCodeFetchedAt?: string;
    sourceCodeOrigin?: string;
    sourceQualityReport?: string;
    sourceQualityReportFetchedAt?: string;
    sourceQualityReportOrigin?: string;
    sourceQualityPromptVersion?: number;
    isOpenSource?: boolean;
    isBytecodeBlacklisted?: boolean;
}

export interface SourceQualityPrompt {
    id?: number;
    version?: number;
    name?: string;
    systemPrompt?: string;
    isActive?: boolean;
    createdAt?: string;
    updatedAt?: string;
}

export interface BytecodeBlacklistEntry {
    codeHash?: string;
    note?: string;
    sourceChainID?: number;
    sourceContract?: string;
    createdAt?: string;
}

export interface BytecodeListItem {
    codeHash?: string;
    runtimeBytecodeSize?: number;
    deploymentCount?: number;
    isOpenSource?: boolean;
    isBytecodeBlacklisted?: boolean;
    createdAt?: string;
    updatedAt?: string;
}

export interface BytecodeDetail extends BytecodeListItem {
    runtimeBytecode?: string;
    sourceCode?: string;
    sourceCodeHash?: string;
    sourceCodeFetchedAt?: string;
    sourceCodeOrigin?: string;
    sourceQualityReport?: string;
    sourceQualityReportFetchedAt?: string;
    sourceQualityReportOrigin?: string;
    sourceQualityPromptVersion?: number;
}

export interface BytecodeDeployment {
    chainID?: number;
    contract?: string;
    firstSeenAt?: string;
    updatedAt?: string;
}

export interface PagedResponse<T> {
    items: T[];
    total: number;
    page: number;
    pageSize: number;
}

interface ListBytecodeBlacklistEntriesResponse {
    items?: BytecodeBlacklistEntry[];
}

interface AddBytecodeBlacklistEntryResponse {
    item?: BytecodeBlacklistEntry;
}

interface UpdateBytecodeBlacklistNoteResponse {
    item?: BytecodeBlacklistEntry;
}

interface ListSourceQualityPromptsResponse {
    items?: SourceQualityPrompt[];
}

interface SourceQualityPromptResponse {
    item?: SourceQualityPrompt;
}

let cachedProjectOptions: ProjectOptions | undefined;
let projectOptionsRequest: (Promise<ProjectOptions | undefined> & {abort?: () => void}) | null = null;
let cachedBytecodeBlacklistEntries: BytecodeBlacklistEntry[] | undefined;
let bytecodeBlacklistEntriesRequest: (Promise<BytecodeBlacklistEntry[]> & {abort?: () => void}) | null = null;

function normalizeBytecodeBlacklistEntry(item: BytecodeBlacklistEntry | any): BytecodeBlacklistEntry {
    return {
        codeHash: item.codeHash ?? item.code_hash,
        note: item.note,
        sourceChainID: item.sourceChainID ?? item.source_chain_id,
        sourceContract: item.sourceContract ?? item.source_contract,
        createdAt: item.createdAt ?? item.created_at
    };
}

function normalizeBytecode(item: BytecodeListItem | any): BytecodeListItem {
    return {
        codeHash: item.codeHash ?? item.code_hash,
        runtimeBytecodeSize: item.runtimeBytecodeSize ?? item.runtime_bytecode_size,
        deploymentCount: item.deploymentCount ?? item.deployment_count,
        isOpenSource: item.isOpenSource ?? item.is_open_source,
        isBytecodeBlacklisted: item.isBytecodeBlacklisted ?? item.is_bytecode_blacklisted,
        createdAt: item.createdAt ?? item.created_at,
        updatedAt: item.updatedAt ?? item.updated_at
    };
}

function normalizeBytecodeDetail(item: BytecodeDetail | any): BytecodeDetail {
    return {
        ...normalizeBytecode(item),
        runtimeBytecode: item.runtimeBytecode ?? item.runtime_bytecode,
        sourceCode: item.sourceCode ?? item.source_code,
        sourceCodeHash: item.sourceCodeHash ?? item.source_code_hash,
        sourceCodeFetchedAt: item.sourceCodeFetchedAt ?? item.source_code_fetched_at,
        sourceCodeOrigin: item.sourceCodeOrigin ?? item.source_code_origin,
        sourceQualityReport: item.sourceQualityReport ?? item.source_quality_report,
        sourceQualityReportFetchedAt: item.sourceQualityReportFetchedAt ?? item.source_quality_report_fetched_at,
        sourceQualityReportOrigin: item.sourceQualityReportOrigin ?? item.source_quality_report_origin,
        sourceQualityPromptVersion: item.sourceQualityPromptVersion ?? item.source_quality_prompt_version
    };
}

function normalizeSourceQualityPrompt(item: SourceQualityPrompt | any): SourceQualityPrompt {
    return {
        id: item.id,
        version: item.version,
        name: item.name,
        systemPrompt: item.systemPrompt ?? item.system_prompt,
        isActive: item.isActive ?? item.is_active,
        createdAt: item.createdAt ?? item.created_at,
        updatedAt: item.updatedAt ?? item.updated_at
    };
}

function normalizeBytecodeDeployment(item: BytecodeDeployment | any): BytecodeDeployment {
    return {
        chainID: item.chainID ?? item.chain_id,
        contract: item.contract,
        firstSeenAt: item.firstSeenAt ?? item.first_seen_at,
        updatedAt: item.updatedAt ?? item.updated_at
    };
}

const normalizeProjectDiscoveryStatus = (body: any): ProjectDiscoveryStatus => {
    const started = !!(body && (body.started ?? body.Started));
    const status = (body && (body.status || body.Status)) || (started ? 'running' : 'stopped');
    return {started, status};
};

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

    public getProjectDiscoveryStatus(): Promise<ProjectDiscoveryStatus> & {abort?: () => void} {
        const req = requests.get('/application/discovery/status');
        const promise = req.then(res => normalizeProjectDiscoveryStatus(res.body || {})) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public startProjectDiscovery(): Promise<ProjectDiscoveryStatus> & {abort?: () => void} {
        const req = requests.post('/application/discovery/start').send({});
        const promise = req.then(res => normalizeProjectDiscoveryStatus(res.body || {})) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public stopProjectDiscovery(): Promise<ProjectDiscoveryStatus> & {abort?: () => void} {
        const req = requests.post('/application/discovery/stop').send({});
        const promise = req.then(res => normalizeProjectDiscoveryStatus(res.body || {})) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public getProject(contract: string): Promise<ProjectView> & {abort?: () => void} {
        const req = requests.get(`/projects/${encodeURIComponent(contract)}`);
        const promise = req.then(res => ((res.body || {}) as GetProjectResponse).item || ({} as ProjectView)) as any;
        promise.abort = () => req.abort();
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

    public getContractSourceInfo(contract: string, chainID?: number): Promise<ContractSourceInfo> & {abort?: () => void} {
        const req = requests.get(`/application/contracts/${encodeURIComponent(contract)}/source`);
        if (chainID) {
            req.query({chain_id: chainID});
        }
        const promise = req.then(res => {
            const item = (res.body || {}) as any;
            return {
                contract: item.contract,
                chainID: item.chainID ?? item.chain_id,
                codeBinHash: item.codeBinHash ?? item.code_bin_hash,
                sourceCode: item.sourceCode ?? item.source_code,
                sourceCodeHash: item.sourceCodeHash ?? item.source_code_hash,
                sourceCodeFetchedAt: item.sourceCodeFetchedAt ?? item.source_code_fetched_at,
                sourceCodeOrigin: item.sourceCodeOrigin ?? item.source_code_origin,
                sourceQualityReport: item.sourceQualityReport ?? item.source_quality_report,
                sourceQualityReportFetchedAt: item.sourceQualityReportFetchedAt ?? item.source_quality_report_fetched_at,
                sourceQualityReportOrigin: item.sourceQualityReportOrigin ?? item.source_quality_report_origin,
                sourceQualityPromptVersion: item.sourceQualityPromptVersion ?? item.source_quality_prompt_version,
                isOpenSource: item.isOpenSource ?? item.is_open_source,
                isBytecodeBlacklisted: item.isBytecodeBlacklisted ?? item.is_bytecode_blacklisted
            } as ContractSourceInfo;
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listBytecodes(options: {page?: number; pageSize?: number; codeHash?: string} = {}): Promise<PagedResponse<BytecodeListItem>> & {abort?: () => void} {
        const req = requests.get('/application/bytecodes').query({
            page: options.page,
            page_size: options.pageSize,
            code_hash: options.codeHash || undefined
        });
        const promise = req.then(res => {
            const body = (res.body || {}) as any;
            return {
                items: (body.items || []).map(normalizeBytecode),
                total: body.total || 0,
                page: body.page || options.page || 1,
                pageSize: body.pageSize ?? body.page_size ?? options.pageSize ?? 20
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public getBytecode(codeHash: string): Promise<BytecodeDetail> & {abort?: () => void} {
        const req = requests.get(`/application/bytecodes/${encodeURIComponent(codeHash)}`);
        const promise = req.then(res => normalizeBytecodeDetail(res.body || {})) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listBytecodeDeployments(
        codeHash: string,
        options: {page?: number; pageSize?: number; chainID?: number; contract?: string} = {}
    ): Promise<PagedResponse<BytecodeDeployment>> & {abort?: () => void} {
        const req = requests.get(`/application/bytecodes/${encodeURIComponent(codeHash)}/deployments`).query({
            page: options.page,
            page_size: options.pageSize,
            chain_id: options.chainID,
            contract: options.contract || undefined
        });
        const promise = req.then(res => {
            const body = (res.body || {}) as any;
            return {
                items: (body.items || []).map(normalizeBytecodeDeployment),
                total: body.total || 0,
                page: body.page || options.page || 1,
                pageSize: body.pageSize ?? body.page_size ?? options.pageSize ?? 20
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listBytecodeBlacklistEntries(): Promise<BytecodeBlacklistEntry[]> & {abort?: () => void} {
        if (cachedBytecodeBlacklistEntries) {
            const promise = Promise.resolve(cachedBytecodeBlacklistEntries) as Promise<BytecodeBlacklistEntry[]> & {abort?: () => void};
            promise.abort = () => {};
            return promise;
        }
        if (bytecodeBlacklistEntriesRequest) {
            const promise = bytecodeBlacklistEntriesRequest as Promise<BytecodeBlacklistEntry[]> & {abort?: () => void};
            promise.abort = () => {};
            return promise;
        }
        const req = requests.get('/application/bytecode/blacklist');
        const promise = req
            .then(res => {
                const body = (res.body as ListBytecodeBlacklistEntriesResponse) || {};
                return (body.items || []).map(normalizeBytecodeBlacklistEntry);
            })
            .then(
                items => {
                    cachedBytecodeBlacklistEntries = items;
                    bytecodeBlacklistEntriesRequest = null;
                    return items;
                },
                err => {
                    bytecodeBlacklistEntriesRequest = null;
                    throw err;
                }
            ) as any;
        promise.abort = () => {};
        bytecodeBlacklistEntriesRequest = promise;
        return promise;
    }

    public addBytecodeBlacklistEntry(sourceContract: string, note: string, sourceChainID?: number): Promise<BytecodeBlacklistEntry> & {abort?: () => void} {
        const req = requests.post('/application/bytecode/blacklist').send({
            sourceContract,
            source_contract: sourceContract,
            sourceChainID,
            source_chain_id: sourceChainID,
            note
        });
        const promise = req
            .then(res => normalizeBytecodeBlacklistEntry(((res.body as AddBytecodeBlacklistEntryResponse) || {}).item || {}))
            .then(item => {
                cachedBytecodeBlacklistEntries = undefined;
                return item;
            }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public updateBytecodeBlacklistNote(codeHash: string, note: string): Promise<BytecodeBlacklistEntry> & {abort?: () => void} {
        const req = requests.post(`/application/bytecode/blacklist/${encodeURIComponent(codeHash)}/note`).send({codeHash, code_hash: codeHash, note});
        const promise = req
            .then(res => normalizeBytecodeBlacklistEntry(((res.body as UpdateBytecodeBlacklistNoteResponse) || {}).item || {}))
            .then(item => {
                cachedBytecodeBlacklistEntries = undefined;
                return item;
            }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public deleteBytecodeBlacklist(codeHash: string): Promise<void> & {abort?: () => void} {
        const req = requests.delete(`/application/bytecode/blacklist/${encodeURIComponent(codeHash)}`);
        const promise = req.then(() => {
            cachedBytecodeBlacklistEntries = undefined;
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listSourceQualityPrompts(): Promise<SourceQualityPrompt[]> & {abort?: () => void} {
        const req = requests.get('/application/source-quality/prompts');
        const promise = req.then(res => {
            const body = (res.body as ListSourceQualityPromptsResponse) || {};
            return (body.items || []).map(normalizeSourceQualityPrompt);
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public createSourceQualityPrompt(name: string, systemPrompt: string): Promise<SourceQualityPrompt> & {abort?: () => void} {
        const req = requests.post('/application/source-quality/prompts').send({
            name,
            systemPrompt,
            system_prompt: systemPrompt
        });
        const promise = req.then(res => normalizeSourceQualityPrompt(((res.body as SourceQualityPromptResponse) || {}).item || {})) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public updateSourceQualityPrompt(id: number, name: string, systemPrompt: string): Promise<SourceQualityPrompt> & {abort?: () => void} {
        const req = requests.post(`/application/source-quality/prompts/${encodeURIComponent(String(id))}`).send({
            id,
            name,
            systemPrompt,
            system_prompt: systemPrompt
        });
        const promise = req.then(res => normalizeSourceQualityPrompt(((res.body as SourceQualityPromptResponse) || {}).item || {})) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public activateSourceQualityPrompt(id: number): Promise<SourceQualityPrompt> & {abort?: () => void} {
        const req = requests.post(`/application/source-quality/prompts/${encodeURIComponent(String(id))}/activate`).send({id});
        const promise = req.then(res => normalizeSourceQualityPrompt(((res.body as SourceQualityPromptResponse) || {}).item || {})) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public deleteSourceQualityPrompt(id: number): Promise<void> & {abort?: () => void} {
        const req = requests.delete(`/application/source-quality/prompts/${encodeURIComponent(String(id))}`);
        const promise = req.then(() => undefined) as any;
        promise.abort = () => req.abort();
        return promise;
    }
}
