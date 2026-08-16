import requests from './requests';

export interface TokenContractCodeBlocklistEntry {
    codeHash?: string;
    note?: string;
    sourceChainID?: number;
    sourceContract?: string;
    createdAt?: string;
}

export interface TokenWalletBlocklistEntry {
    wallet?: string;
    note?: string;
    createdAt?: string;
}

export interface TokenChainCheckpoint {
    chainID?: number;
    chainName?: string;
    enabled?: boolean;
    cursorBlockNumber?: number;
    status?: string;
    createdAt?: string;
}

export interface TokenChain {
    chainID?: number;
    chainName?: string;
}

export interface TokenRuntimeConfiguration {
    chains: TokenChain[];
}

export interface TokenNodeStatus {
    chainID?: number;
    chainName?: string;
    endpoint?: string;
    available?: boolean;
    latencyMS?: number;
    reportedChainID?: number;
    latestBlockNumber?: number;
    referenceBlockNumber?: number;
    blockLag?: number;
    latestBlockTime?: string;
    syncing?: boolean;
    checkedAt?: string;
    error?: string;
}

export interface TokenContractCode {
    codeHash?: string;
    sourceCode?: string;
    sourceCodeFetchedAt?: string;
    createdAt?: string;
    deploymentCount?: number;
}

export interface TokenProject {
    projectID?: number;
    chainID?: number;
    name?: string;
    symbol?: string;
    contract?: string;
    txSender?: string;
    txHash?: string;
    txIndex?: number;
    deploymentNonce?: number;
    blockNumber?: number;
    blockTime?: number;
    codeHash?: string;
    createdAt?: string;
    decimals?: number;
    totalSupply?: string;
    wethPair?: string;
    usdtPair?: string;
}

export interface TokenProjectReportPairRisk {
    isCreated?: boolean;
    isRemoveLiquidity?: boolean;
    isMint?: boolean;
    quoteUsdtValueInt?: string;
    lastSwapAt?: string;
}

export interface TokenProjectReportRiskSummary {
    wethPair?: TokenProjectReportPairRisk;
    usdtPair?: TokenProjectReportPairRisk;
}

export interface TokenProjectReportEvaluation {
    status?: string;
    failedAttempts?: number;
    lastError?: string;
    updatedAt?: string;
    outcome?: string;
    evaluatedAt?: string;
}

export interface TokenProjectCurrentReport {
    revision?: number;
    completenessStatus?: string;
    builtAt?: string;
    riskSummary?: TokenProjectReportRiskSummary;
    evaluation?: TokenProjectReportEvaluation;
}

export interface TokenProjectListItem {
    projectID?: number;
    chainID?: number;
    name?: string;
    symbol?: string;
    blockTime?: number;
    createdAt?: string;
    researchStatus?: string;
    currentReport?: TokenProjectCurrentReport;
    logoURL?: string;
}

export interface TokenCollectionTask {
    taskID?: number;
    projectID?: number;
    dataType?: string;
    status?: string;
    attempts?: number;
    revision?: number;
    availableAt?: string;
    leaseExpiresAt?: string;
    lastError?: string;
    createdAt?: string;
    updatedAt?: string;
}

export interface TokenResearchState {
    projectID?: number;
    chainID?: number;
    contract?: string;
    status?: string;
    evidenceRevision?: number;
    currentReportRevision?: number;
    currentSelectionOutcome?: string;
    lastEvaluatedRevision?: number;
    lastEvaluatedAt?: string;
    attentionStartBlockNumber?: number;
    attentionStartBlockTime?: number;
    attentionExpiryBlockTime?: number;
    expiredBlockNumber?: number;
    expiredBlockTime?: number;
    createdAt?: string;
    updatedAt?: string;
}

export interface TokenReportRevision {
    reportRevisionID?: number;
    projectID?: number;
    chainID?: number;
    contract?: string;
    revision?: number;
    contentHash?: string;
    completenessStatus?: string;
    evidenceJSON?: string;
    reportJSON?: string;
    observedBlockNumber?: number;
    riskSummary?: TokenProjectReportRiskSummary;
    builtAt?: string;
    createdAt?: string;
}

export interface TokenSelection {
    selectionID?: number;
    projectID?: number;
    chainID?: number;
    contract?: string;
    outcome?: string;
    strategyKey?: string;
    strategyVersion?: string;
    reportRevision?: number;
    reasonCodes: string[];
    reasonDetail?: string;
    decidedAt?: string;
    createdAt?: string;
}

export interface TokenProjectObservation {
    observationID?: number;
    projectID?: number;
    dataType?: string;
    schemaVersion?: number;
    contentHash?: string;
    payloadJSON?: string;
    blockNumber?: number;
    observedAt?: string;
    lastCheckedAt?: string;
    createdAt?: string;
}

export interface TokenAveToken {
    address?: string;
    name?: string;
    symbol?: string;
    logoURL?: string;
    decimals?: number;
    totalSupply?: string;
    currentPriceUSD?: string;
    currentPriceETH?: string;
    marketCap?: string;
    fdv?: string;
    tvl?: string;
    mainPairTVL?: string;
    holders?: number;
    riskLevel?: number;
    riskScore?: string;
    riskInfo?: string;
    isMintableKnown?: boolean;
    isMintable?: boolean;
    hasMintMethod?: boolean;
    isLPNotLocked?: boolean;
    hasNotRenounced?: boolean;
    hasNotAudited?: boolean;
    hasNotOpenSource?: boolean;
    isInBlacklist?: boolean;
    isHoneypot?: boolean;
    launchAt?: string;
    updatedAt?: string;
}

export interface TokenAvePair {
    pair?: string;
    chainID?: number;
    amm?: string;
    token0Address?: string;
    token0Symbol?: string;
    token1Address?: string;
    token1Symbol?: string;
    reserve0?: string;
    reserve1?: string;
    volumeUSD?: string;
    marketCap?: string;
    fdv?: string;
    isFake?: boolean;
    createdAt?: string;
    updatedAt?: string;
}

export interface TokenAveObservation {
    chainID?: number;
    token?: TokenAveToken;
    pairs: TokenAvePair[];
    isAudited?: boolean;
}

export interface TokenChainPair {
    pairContract?: string;
    isCreated?: boolean;
    liquidity?: {
        totalSupply?: string;
        lockedLiquidity?: string;
        feeAddressHoldLiquidityBalance?: string;
        feeAddressHoldLiquidityRatio?: string;
    };
    baseBalance?: string;
    quoteBalance?: string;
    quoteUsdtValue?: string;
    quoteUsdtValueInt?: string;
    lastSwapAt?: string;
    isRemoveLiquidity?: boolean;
    isMint?: boolean;
}

export interface TokenChainStateObservation {
    tokenContract?: string;
    updatedAt?: string;
    token?: {
        isValidERC20?: boolean;
        name?: string;
        symbol?: string;
        decimals?: number;
        totalSupply?: string;
        wethPair?: string;
        usdtPair?: string;
    };
    isValidERC20?: boolean;
    wethPair?: TokenChainPair;
    usdtPair?: TokenChainPair;
}

export interface TokenWalletAssetState {
    chainID?: number;
    wallet?: string;
    wethBalance?: string;
    usdtBalance?: string;
    nativeBalance?: string;
    totalAssetUsdtValue?: string;
}

export interface TokenSimulationResult {
    projectID?: number;
    wallet?: string;
    canMintFromDeadViaTransferFrom?: boolean;
    canMintFromZeroViaTransferFrom?: boolean;
    canMintFromWethPairViaTransferFrom?: boolean;
    canMintFromUsdtPairViaTransferFrom?: boolean;
    canMintViaTransferToWethPair?: boolean;
    canMintViaTransferToUsdtPair?: boolean;
}

export interface TokenProjectRelatedWallet {
    projectID?: number;
    wallet?: string;
    role?: string;
    createdAt?: string;
}

export interface TokenProjectInitialRecipient {
    recipientID?: number;
    projectID?: number;
    wallet?: string;
    ratioBPS?: number;
    rankIndex?: number;
    sourceTxHash?: string;
    sourceBlockNumber?: number;
    createdAt?: string;
}

export interface TokenCollectionSchedule {
    projectID?: number;
    dataType?: string;
    status?: string;
    retryIntervalSecs?: number;
    nextRunAt?: string;
    latestTaskRevision?: number;
    consecutiveFailures?: number;
    lastError?: string;
    lastCheckedAt?: string;
    createdAt?: string;
    updatedAt?: string;
}

export interface TokenProjectDetail {
    project?: TokenProject;
    researchState?: TokenResearchState;
    currentReport?: TokenReportRevision;
    currentReportEvaluation?: TokenProjectReportEvaluation;
    currentSelection?: TokenSelection;
    ave?: TokenAveObservation;
    chainState?: TokenChainStateObservation;
    walletAssets: TokenWalletAssetState[];
    simulations: TokenSimulationResult[];
    contractSource?: {codeHash?: string; sourceAvailable?: boolean};
    relatedWallets: TokenProjectRelatedWallet[];
    initialRecipients: TokenProjectInitialRecipient[];
    collectionSchedules: TokenCollectionSchedule[];
    walletTransactionCounts: Array<{wallet?: string; transactionCount?: number}>;
    transactionCount?: number;
    currentObservations: TokenProjectObservation[];
    generatedAt?: string;
}

export type TokenProjectSwapPairKind = 'weth' | 'usdt';

export type TokenProjectSwapDirection = 'buy' | 'sell' | 'complex';

export interface TokenProjectSwapAsset {
    address?: string;
    symbol?: string;
    decimals?: number;
    tokenIndex?: number;
}

export interface TokenProjectSwapBlock {
    sampleIndex?: number;
    blockNumber?: string;
    blockTime?: string;
    eventCount?: number;
    transactionCount?: number;
    transactionOriginCount?: number;
    buyEventCount?: number;
    sellEventCount?: number;
    complexEventCount?: number;
    previousBlockGap?: string;
    previousTimeGapSeconds?: string;
    baseAmountInRaw?: string;
    baseAmountOutRaw?: string;
    quoteAmountInRaw?: string;
    quoteAmountOutRaw?: string;
    buyQuoteAmountRaw?: string;
    sellQuoteAmountRaw?: string;
    openPrice?: string;
    highPrice?: string;
    lowPrice?: string;
    closePrice?: string;
    vwap?: string;
}

export interface TokenProjectSwapPairActivity {
    pairKind?: TokenProjectSwapPairKind;
    pairAddress?: string;
    status?: string;
    swapBlockCount?: number;
    targetSwapBlockCount?: number;
    startBlockNumber?: string;
    startBlockTime?: string;
    firstSwapBlockNumber?: string;
    firstSwapBlockTime?: string;
    lastSwapBlockNumber?: string;
    lastSwapBlockTime?: string;
    absoluteExpiryBlockTime?: string;
    nextExpiryBlockTime?: string;
    completedBlockNumber?: string;
    completedBlockTime?: string;
    expiredBlockNumber?: string;
    expiredBlockTime?: string;
    expiredReason?: string;
    baseAsset?: TokenProjectSwapAsset;
    quoteAsset?: TokenProjectSwapAsset;
    eventCount?: number;
    transactionCount?: number;
    transactionOriginCount?: number;
    buyEventCount?: number;
    sellEventCount?: number;
    complexEventCount?: number;
    baseAmountInRaw?: string;
    baseAmountOutRaw?: string;
    quoteAmountInRaw?: string;
    quoteAmountOutRaw?: string;
    buyQuoteAmountRaw?: string;
    sellQuoteAmountRaw?: string;
    blocks: TokenProjectSwapBlock[];
}

export interface TokenProjectSwapActivity {
    projectID?: number;
    chainID?: number;
    generatedAt?: string;
    pairs: TokenProjectSwapPairActivity[];
}

export interface TokenProjectSwapEvent {
    transactionHash?: string;
    transactionIndex?: string;
    logIndex?: string;
    txFrom?: string;
    sender?: string;
    toAddress?: string;
    amount0In?: string;
    amount1In?: string;
    amount0Out?: string;
    amount1Out?: string;
    baseAmountInRaw?: string;
    baseAmountOutRaw?: string;
    quoteAmountInRaw?: string;
    quoteAmountOutRaw?: string;
    direction?: TokenProjectSwapDirection;
    effectivePrice?: string;
}

export interface TokenProjectTrendPoint {
    observedAt?: string;
    value?: string;
}

export interface TokenProjectTrendSeries {
    key?: string;
    label?: string;
    unit?: string;
    dataType?: string;
    points: TokenProjectTrendPoint[];
}

export interface TokenProjectTrends {
    range?: string;
    observedFrom?: string;
    generatedAt?: string;
    series: TokenProjectTrendSeries[];
}

export interface TokenWalletNormalTransaction {
    wallet?: string;
    transactionHash?: string;
    blockNumber?: number;
    blockTimestamp?: string;
    transactionIndex?: number;
    nonce?: number;
    fromAddress?: string;
    toAddress?: string;
    value?: string;
    gas?: number;
    gasPrice?: string;
    gasUsed?: number;
    input?: string;
    methodID?: string;
    functionName?: string;
    receiptStatus?: string;
    isError?: boolean;
    collectedAt?: string;
}

export interface PagedResponse<T> {
    items: T[];
    total: number;
    page: number;
    pageSize: number;
}

const numberValue = (value: unknown): number | undefined => {
    if (typeof value === 'number') {
        return value;
    }
    if (typeof value === 'string' && value !== '') {
        const parsed = Number(value);
        return Number.isNaN(parsed) ? undefined : parsed;
    }
    return undefined;
};

const exactStringValue = (value: unknown): string | undefined => {
    if (value === undefined || value === null) {
        return undefined;
    }
    return String(value);
};

function normalizeContractCodeBlocklistEntry(item: any): TokenContractCodeBlocklistEntry {
    return {
        codeHash: item.codeHash ?? item.code_hash,
        note: item.note,
        sourceChainID: numberValue(item.sourceChainID ?? item.sourceChainId ?? item.source_chain_id),
        sourceContract: item.sourceContract ?? item.source_contract,
        createdAt: item.createdAt ?? item.created_at
    };
}

function normalizeWalletBlocklistEntry(item: any): TokenWalletBlocklistEntry {
    return {
        wallet: item.wallet,
        note: item.note,
        createdAt: item.createdAt ?? item.created_at
    };
}

function normalizeCheckpoint(item: any): TokenChainCheckpoint {
    return {
        chainID: numberValue(item.chainID ?? item.chainId ?? item.chain_id),
        chainName: item.chainName ?? item.chain_name,
        enabled: item.enabled,
        cursorBlockNumber: numberValue(item.cursorBlockNumber ?? item.cursor_block_number),
        status: item.status,
        createdAt: item.createdAt ?? item.created_at
    };
}

function normalizeChainOption(item: any): TokenChain {
    return {
        chainID: numberValue(item.chainID ?? item.chainId ?? item.chain_id),
        chainName: item.chainName ?? item.chain_name
    };
}

function normalizeNodeStatus(item: any): TokenNodeStatus {
    return {
        chainID: numberValue(item.chainID ?? item.chainId ?? item.chain_id),
        chainName: item.chainName ?? item.chain_name,
        endpoint: item.endpoint,
        available: item.available,
        latencyMS: numberValue(item.latencyMS ?? item.latencyMs ?? item.latency_ms),
        reportedChainID: numberValue(item.reportedChainID ?? item.reportedChainId ?? item.reported_chain_id),
        latestBlockNumber: numberValue(item.latestBlockNumber ?? item.latest_block_number),
        referenceBlockNumber: numberValue(item.referenceBlockNumber ?? item.reference_block_number),
        blockLag: numberValue(item.blockLag ?? item.block_lag),
        latestBlockTime: item.latestBlockTime ?? item.latest_block_time,
        syncing: item.syncing,
        checkedAt: item.checkedAt ?? item.checked_at,
        error: item.error
    };
}

function normalizeContractCode(item: any): TokenContractCode {
    return {
        codeHash: item.codeHash ?? item.code_hash,
        sourceCode: item.sourceCode ?? item.source_code,
        sourceCodeFetchedAt: item.sourceCodeFetchedAt ?? item.source_code_fetched_at,
        createdAt: item.createdAt ?? item.created_at,
        deploymentCount: numberValue(item.deploymentCount ?? item.deployment_count)
    };
}

function normalizeProject(item: any): TokenProject {
    return {
        projectID: numberValue(item.projectID ?? item.projectId ?? item.project_id),
        chainID: numberValue(item.chainID ?? item.chainId ?? item.chain_id),
        name: item.name,
        symbol: item.symbol,
        contract: item.contract,
        txSender: item.txSender ?? item.tx_sender,
        txHash: item.txHash ?? item.tx_hash,
        txIndex: numberValue(item.txIndex ?? item.tx_index),
        deploymentNonce: numberValue(item.deploymentNonce ?? item.deployment_nonce),
        blockNumber: numberValue(item.blockNumber ?? item.block_number),
        blockTime: numberValue(item.blockTime ?? item.block_time),
        codeHash: item.codeHash ?? item.code_hash,
        createdAt: item.createdAt ?? item.created_at,
        decimals: numberValue(item.decimals),
        totalSupply: item.totalSupply ?? item.total_supply,
        wethPair: item.wethPair ?? item.weth_pair,
        usdtPair: item.usdtPair ?? item.usdt_pair
    };
}

function normalizeProjectReportPairRisk(item: any): TokenProjectReportPairRisk | undefined {
    if (!item) {
        return undefined;
    }
    return {
        isCreated: item.isCreated ?? item.is_created,
        isRemoveLiquidity: item.isRemoveLiquidity ?? item.is_remove_liquidity,
        isMint: item.isMint ?? item.is_mint,
        quoteUsdtValueInt: item.quoteUsdtValueInt ?? item.quote_usdt_value_int,
        lastSwapAt: item.lastSwapAt ?? item.last_swap_at
    };
}

function normalizeProjectReportRiskSummary(item: any): TokenProjectReportRiskSummary | undefined {
    if (!item) {
        return undefined;
    }
    return {
        wethPair: normalizeProjectReportPairRisk(item.wethPair ?? item.weth_pair),
        usdtPair: normalizeProjectReportPairRisk(item.usdtPair ?? item.usdt_pair)
    };
}

function normalizeProjectReportEvaluation(item: any): TokenProjectReportEvaluation | undefined {
    if (!item) {
        return undefined;
    }
    return {
        status: item.status,
        failedAttempts: numberValue(item.failedAttempts ?? item.failed_attempts),
        lastError: item.lastError ?? item.last_error,
        updatedAt: item.updatedAt ?? item.updated_at,
        outcome: item.outcome,
        evaluatedAt: item.evaluatedAt ?? item.evaluated_at
    };
}

function normalizeProjectCurrentReport(item: any): TokenProjectCurrentReport | undefined {
    if (!item) {
        return undefined;
    }
    return {
        revision: numberValue(item.revision),
        completenessStatus: item.completenessStatus ?? item.completeness_status,
        builtAt: item.builtAt ?? item.built_at,
        riskSummary: normalizeProjectReportRiskSummary(item.riskSummary ?? item.risk_summary),
        evaluation: normalizeProjectReportEvaluation(item.evaluation)
    };
}

function normalizeProjectListItem(item: any): TokenProjectListItem {
    return {
        projectID: numberValue(item.projectID ?? item.projectId ?? item.project_id),
        chainID: numberValue(item.chainID ?? item.chainId ?? item.chain_id),
        name: item.name,
        symbol: item.symbol,
        blockTime: numberValue(item.blockTime ?? item.block_time),
        createdAt: item.createdAt ?? item.created_at,
        researchStatus: item.researchStatus ?? item.research_status,
        currentReport: normalizeProjectCurrentReport(item.currentReport ?? item.current_report),
        logoURL: item.logoURL ?? item.logoUrl ?? item.logo_url
    };
}

function normalizeTask(item: any): TokenCollectionTask {
    return {
        taskID: numberValue(item.taskID ?? item.taskId ?? item.task_id),
        projectID: numberValue(item.projectID ?? item.projectId ?? item.project_id),
        dataType: item.dataType ?? item.data_type,
        status: item.status,
        revision: numberValue(item.revision),
        attempts: numberValue(item.attempts),
        availableAt: item.availableAt ?? item.available_at,
        leaseExpiresAt: item.leaseExpiresAt ?? item.lease_expires_at,
        lastError: item.lastError ?? item.last_error,
        createdAt: item.createdAt ?? item.created_at,
        updatedAt: item.updatedAt ?? item.updated_at
    };
}

function normalizeResearchState(item: any): TokenResearchState {
    return {
        projectID: numberValue(item.projectID ?? item.projectId ?? item.project_id),
        chainID: numberValue(item.chainID ?? item.chainId ?? item.chain_id),
        contract: item.contract,
        status: item.status,
        evidenceRevision: numberValue(item.evidenceRevision ?? item.evidence_revision),
        currentReportRevision: numberValue(item.currentReportRevision ?? item.current_report_revision),
        currentSelectionOutcome: item.currentSelectionOutcome ?? item.current_selection_outcome,
        lastEvaluatedRevision: numberValue(item.lastEvaluatedRevision ?? item.last_evaluated_revision),
        lastEvaluatedAt: item.lastEvaluatedAt ?? item.last_evaluated_at,
        attentionStartBlockNumber: numberValue(item.attentionStartBlockNumber ?? item.attention_start_block_number),
        attentionStartBlockTime: numberValue(item.attentionStartBlockTime ?? item.attention_start_block_time),
        attentionExpiryBlockTime: numberValue(item.attentionExpiryBlockTime ?? item.attention_expiry_block_time),
        expiredBlockNumber: numberValue(item.expiredBlockNumber ?? item.expired_block_number),
        expiredBlockTime: numberValue(item.expiredBlockTime ?? item.expired_block_time),
        createdAt: item.createdAt ?? item.created_at,
        updatedAt: item.updatedAt ?? item.updated_at
    };
}

function normalizeReportRevision(item: any): TokenReportRevision {
    return {
        reportRevisionID: numberValue(item.reportRevisionID ?? item.reportRevisionId ?? item.report_revision_id),
        projectID: numberValue(item.projectID ?? item.projectId ?? item.project_id),
        chainID: numberValue(item.chainID ?? item.chainId ?? item.chain_id),
        contract: item.contract,
        revision: numberValue(item.revision),
        contentHash: item.contentHash ?? item.content_hash,
        completenessStatus: item.completenessStatus ?? item.completeness_status,
        evidenceJSON: item.evidenceJSON ?? item.evidenceJson ?? item.evidence_json,
        reportJSON: item.reportJSON ?? item.reportJson ?? item.report_json,
        observedBlockNumber: numberValue(item.observedBlockNumber ?? item.observed_block_number),
        riskSummary: normalizeProjectReportRiskSummary(item.riskSummary ?? item.risk_summary),
        builtAt: item.builtAt ?? item.built_at,
        createdAt: item.createdAt ?? item.created_at
    };
}

function normalizeSelection(item: any): TokenSelection {
    return {
        selectionID: numberValue(item.selectionID ?? item.selectionId ?? item.selection_id),
        projectID: numberValue(item.projectID ?? item.projectId ?? item.project_id),
        chainID: numberValue(item.chainID ?? item.chainId ?? item.chain_id),
        contract: item.contract,
        outcome: item.outcome,
        strategyKey: item.strategyKey ?? item.strategy_key,
        strategyVersion: item.strategyVersion ?? item.strategy_version,
        reportRevision: numberValue(item.reportRevision ?? item.report_revision),
        reasonCodes: item.reasonCodes ?? item.reason_codes ?? [],
        reasonDetail: item.reasonDetail ?? item.reason_detail,
        decidedAt: item.decidedAt ?? item.decided_at,
        createdAt: item.createdAt ?? item.created_at
    };
}

function normalizeObservation(item: any): TokenProjectObservation {
    return {
        observationID: numberValue(item.observationID ?? item.observationId ?? item.observation_id),
        projectID: numberValue(item.projectID ?? item.projectId ?? item.project_id),
        dataType: item.dataType ?? item.data_type,
        schemaVersion: numberValue(item.schemaVersion ?? item.schema_version),
        contentHash: item.contentHash ?? item.content_hash,
        payloadJSON: item.payloadJSON ?? item.payloadJson ?? item.payload_json,
        blockNumber: numberValue(item.blockNumber ?? item.block_number),
        observedAt: item.observedAt ?? item.observed_at,
        lastCheckedAt: item.lastCheckedAt ?? item.last_checked_at,
        createdAt: item.createdAt ?? item.created_at
    };
}

function normalizeAveToken(item: any): TokenAveToken {
    return {
        address: item.address,
        name: item.name,
        symbol: item.symbol,
        logoURL: item.logoURL ?? item.logoUrl ?? item.logo_url,
        decimals: numberValue(item.decimals),
        totalSupply: item.totalSupply ?? item.total_supply,
        currentPriceUSD: item.currentPriceUSD ?? item.currentPriceUsd ?? item.current_price_usd,
        currentPriceETH: item.currentPriceETH ?? item.currentPriceEth ?? item.current_price_eth,
        marketCap: item.marketCap ?? item.market_cap,
        fdv: item.fdv,
        tvl: item.tvl,
        mainPairTVL: item.mainPairTVL ?? item.mainPairTvl ?? item.main_pair_tvl,
        holders: numberValue(item.holders),
        riskLevel: numberValue(item.riskLevel ?? item.risk_level),
        riskScore: item.riskScore ?? item.risk_score,
        riskInfo: item.riskInfo ?? item.risk_info,
        isMintableKnown: item.isMintableKnown ?? item.is_mintable_known,
        isMintable: item.isMintable ?? item.is_mintable,
        hasMintMethod: item.hasMintMethod ?? item.has_mint_method,
        isLPNotLocked: item.isLPNotLocked ?? item.isLpNotLocked ?? item.is_lp_not_locked,
        hasNotRenounced: item.hasNotRenounced ?? item.has_not_renounced,
        hasNotAudited: item.hasNotAudited ?? item.has_not_audited,
        hasNotOpenSource: item.hasNotOpenSource ?? item.has_not_open_source,
        isInBlacklist: item.isInBlacklist ?? item.is_in_blacklist,
        isHoneypot: item.isHoneypot ?? item.is_honeypot,
        launchAt: item.launchAt ?? item.launch_at,
        updatedAt: item.updatedAt ?? item.updated_at
    };
}

function normalizeAvePair(item: any): TokenAvePair {
    return {
        pair: item.pair,
        chainID: numberValue(item.chainID ?? item.chainId ?? item.chain_id),
        amm: item.amm,
        token0Address: item.token0Address ?? item.token0_address,
        token0Symbol: item.token0Symbol ?? item.token0_symbol,
        token1Address: item.token1Address ?? item.token1_address,
        token1Symbol: item.token1Symbol ?? item.token1_symbol,
        reserve0: item.reserve0,
        reserve1: item.reserve1,
        volumeUSD: item.volumeUSD ?? item.volumeUsd ?? item.volume_usd,
        marketCap: item.marketCap ?? item.market_cap,
        fdv: item.fdv,
        isFake: item.isFake ?? item.is_fake,
        createdAt: item.createdAt ?? item.created_at,
        updatedAt: item.updatedAt ?? item.updated_at
    };
}

function normalizeChainPair(item: any): TokenChainPair {
    const liquidity = item?.liquidity;
    return {
        pairContract: item?.pairContract ?? item?.pair_contract,
        isCreated: item?.isCreated ?? item?.is_created,
        liquidity: liquidity
            ? {
                  totalSupply: liquidity.totalSupply ?? liquidity.total_supply,
                  lockedLiquidity: liquidity.lockedLiquidity ?? liquidity.locked_liquidity,
                  feeAddressHoldLiquidityBalance: liquidity.feeAddressHoldLiquidityBalance ?? liquidity.fee_address_hold_liquidity_balance,
                  feeAddressHoldLiquidityRatio: liquidity.feeAddressHoldLiquidityRatio ?? liquidity.fee_address_hold_liquidity_ratio
              }
            : undefined,
        baseBalance: item?.baseBalance ?? item?.base_balance,
        quoteBalance: item?.quoteBalance ?? item?.quote_balance,
        quoteUsdtValue: item?.quoteUsdtValue ?? item?.quote_usdt_value,
        quoteUsdtValueInt: item?.quoteUsdtValueInt ?? item?.quote_usdt_value_int,
        lastSwapAt: item?.lastSwapAt ?? item?.last_swap_at,
        isRemoveLiquidity: item?.isRemoveLiquidity ?? item?.is_remove_liquidity,
        isMint: item?.isMint ?? item?.is_mint
    };
}

function normalizeProjectDetail(item: any): TokenProjectDetail {
    const ave = item.ave;
    const chainState = item.chainState ?? item.chain_state;
    const chainToken = chainState?.token;
    return {
        project: item.project ? normalizeProject(item.project) : undefined,
        researchState: item.researchState || item.research_state ? normalizeResearchState(item.researchState ?? item.research_state) : undefined,
        currentReport: item.currentReport || item.current_report ? normalizeReportRevision(item.currentReport ?? item.current_report) : undefined,
        currentReportEvaluation: normalizeProjectReportEvaluation(item.currentReportEvaluation ?? item.current_report_evaluation),
        currentSelection: item.currentSelection || item.current_selection ? normalizeSelection(item.currentSelection ?? item.current_selection) : undefined,
        ave: ave
            ? {
                  chainID: numberValue(ave.chainID ?? ave.chainId ?? ave.chain_id),
                  token: ave.token ? normalizeAveToken(ave.token) : undefined,
                  pairs: ((ave.pairs || []) as any[]).map(normalizeAvePair),
                  isAudited: ave.isAudited ?? ave.is_audited
              }
            : undefined,
        chainState: chainState
            ? {
                  tokenContract: chainState.tokenContract ?? chainState.token_contract,
                  updatedAt: chainState.updatedAt ?? chainState.updated_at,
                  token: chainToken
                      ? {
                            isValidERC20: chainToken.isValidERC20 ?? chainToken.isValidErc20 ?? chainToken.is_valid_erc20,
                            name: chainToken.name,
                            symbol: chainToken.symbol,
                            decimals: numberValue(chainToken.decimals),
                            totalSupply: chainToken.totalSupply ?? chainToken.total_supply,
                            wethPair: chainToken.wethPair ?? chainToken.weth_pair,
                            usdtPair: chainToken.usdtPair ?? chainToken.usdt_pair
                        }
                      : undefined,
                  isValidERC20: chainState.isValidERC20 ?? chainState.isValidErc20 ?? chainState.is_valid_erc20,
                  wethPair: chainState.wethPair || chainState.weth_pair ? normalizeChainPair(chainState.wethPair ?? chainState.weth_pair) : undefined,
                  usdtPair: chainState.usdtPair || chainState.usdt_pair ? normalizeChainPair(chainState.usdtPair ?? chainState.usdt_pair) : undefined
              }
            : undefined,
        walletAssets: ((item.walletAssets ?? item.wallet_assets ?? []) as any[]).map(asset => ({
            chainID: numberValue(asset.chainID ?? asset.chainId ?? asset.chain_id),
            wallet: asset.wallet,
            wethBalance: asset.wethBalance ?? asset.weth_balance,
            usdtBalance: asset.usdtBalance ?? asset.usdt_balance,
            nativeBalance: asset.nativeBalance ?? asset.native_balance,
            totalAssetUsdtValue: asset.totalAssetUsdtValue ?? asset.total_asset_usdt_value
        })),
        simulations: ((item.simulations || []) as any[]).map(simulation => ({
            projectID: numberValue(simulation.projectID ?? simulation.projectId ?? simulation.project_id),
            wallet: simulation.wallet,
            canMintFromDeadViaTransferFrom: simulation.canMintFromDeadViaTransferFrom ?? simulation.can_mint_from_dead_via_transfer_from,
            canMintFromZeroViaTransferFrom: simulation.canMintFromZeroViaTransferFrom ?? simulation.can_mint_from_zero_via_transfer_from,
            canMintFromWethPairViaTransferFrom: simulation.canMintFromWethPairViaTransferFrom ?? simulation.can_mint_from_weth_pair_via_transfer_from,
            canMintFromUsdtPairViaTransferFrom: simulation.canMintFromUsdtPairViaTransferFrom ?? simulation.can_mint_from_usdt_pair_via_transfer_from,
            canMintViaTransferToWethPair: simulation.canMintViaTransferToWethPair ?? simulation.can_mint_via_transfer_to_weth_pair,
            canMintViaTransferToUsdtPair: simulation.canMintViaTransferToUsdtPair ?? simulation.can_mint_via_transfer_to_usdt_pair
        })),
        contractSource:
            item.contractSource || item.contract_source
                ? {
                      codeHash: (item.contractSource ?? item.contract_source).codeHash ?? (item.contractSource ?? item.contract_source).code_hash,
                      sourceAvailable: (item.contractSource ?? item.contract_source).sourceAvailable ?? (item.contractSource ?? item.contract_source).source_available
                  }
                : undefined,
        relatedWallets: ((item.relatedWallets ?? item.related_wallets ?? []) as any[]).map(wallet => ({
            projectID: numberValue(wallet.projectID ?? wallet.projectId ?? wallet.project_id),
            wallet: wallet.wallet,
            role: wallet.role,
            createdAt: wallet.createdAt ?? wallet.created_at
        })),
        initialRecipients: ((item.initialRecipients ?? item.initial_recipients ?? []) as any[]).map(recipient => ({
            recipientID: numberValue(recipient.recipientID ?? recipient.recipientId ?? recipient.recipient_id),
            projectID: numberValue(recipient.projectID ?? recipient.projectId ?? recipient.project_id),
            wallet: recipient.wallet,
            ratioBPS: numberValue(recipient.ratioBPS ?? recipient.ratioBps ?? recipient.ratio_bps),
            rankIndex: numberValue(recipient.rankIndex ?? recipient.rank_index),
            sourceTxHash: recipient.sourceTxHash ?? recipient.source_tx_hash,
            sourceBlockNumber: numberValue(recipient.sourceBlockNumber ?? recipient.source_block_number),
            createdAt: recipient.createdAt ?? recipient.created_at
        })),
        collectionSchedules: ((item.collectionSchedules ?? item.collection_schedules ?? []) as any[]).map(schedule => ({
            projectID: numberValue(schedule.projectID ?? schedule.projectId ?? schedule.project_id),
            dataType: schedule.dataType ?? schedule.data_type,
            status: schedule.status,
            retryIntervalSecs: numberValue(schedule.retryIntervalSecs ?? schedule.retry_interval_secs),
            nextRunAt: schedule.nextRunAt ?? schedule.next_run_at,
            latestTaskRevision: numberValue(schedule.latestTaskRevision ?? schedule.latest_task_revision),
            consecutiveFailures: numberValue(schedule.consecutiveFailures ?? schedule.consecutive_failures),
            lastError: schedule.lastError ?? schedule.last_error,
            lastCheckedAt: schedule.lastCheckedAt ?? schedule.last_checked_at,
            createdAt: schedule.createdAt ?? schedule.created_at,
            updatedAt: schedule.updatedAt ?? schedule.updated_at
        })),
        walletTransactionCounts: ((item.walletTransactionCounts ?? item.wallet_transaction_counts ?? []) as any[]).map(count => ({
            wallet: count.wallet,
            transactionCount: numberValue(count.transactionCount ?? count.transaction_count)
        })),
        transactionCount: numberValue(item.transactionCount ?? item.transaction_count),
        currentObservations: ((item.currentObservations ?? item.current_observations ?? []) as any[]).map(normalizeObservation),
        generatedAt: item.generatedAt ?? item.generated_at
    };
}

function normalizeProjectSwapAsset(item: any): TokenProjectSwapAsset | undefined {
    if (!item) {
        return undefined;
    }
    return {
        address: item.address,
        symbol: item.symbol,
        decimals: numberValue(item.decimals),
        tokenIndex: numberValue(item.tokenIndex ?? item.token_index)
    };
}

function normalizeProjectSwapBlock(item: any): TokenProjectSwapBlock {
    return {
        sampleIndex: numberValue(item.sampleIndex ?? item.sample_index),
        blockNumber: exactStringValue(item.blockNumber ?? item.block_number),
        blockTime: item.blockTime ?? item.block_time,
        eventCount: numberValue(item.eventCount ?? item.event_count),
        transactionCount: numberValue(item.transactionCount ?? item.transaction_count),
        transactionOriginCount: numberValue(item.transactionOriginCount ?? item.transaction_origin_count),
        buyEventCount: numberValue(item.buyEventCount ?? item.buy_event_count),
        sellEventCount: numberValue(item.sellEventCount ?? item.sell_event_count),
        complexEventCount: numberValue(item.complexEventCount ?? item.complex_event_count),
        previousBlockGap: exactStringValue(item.previousBlockGap ?? item.previous_block_gap),
        previousTimeGapSeconds: exactStringValue(item.previousTimeGapSeconds ?? item.previous_time_gap_seconds),
        baseAmountInRaw: item.baseAmountInRaw ?? item.base_amount_in_raw,
        baseAmountOutRaw: item.baseAmountOutRaw ?? item.base_amount_out_raw,
        quoteAmountInRaw: item.quoteAmountInRaw ?? item.quote_amount_in_raw,
        quoteAmountOutRaw: item.quoteAmountOutRaw ?? item.quote_amount_out_raw,
        buyQuoteAmountRaw: item.buyQuoteAmountRaw ?? item.buy_quote_amount_raw,
        sellQuoteAmountRaw: item.sellQuoteAmountRaw ?? item.sell_quote_amount_raw,
        openPrice: item.openPrice ?? item.open_price,
        highPrice: item.highPrice ?? item.high_price,
        lowPrice: item.lowPrice ?? item.low_price,
        closePrice: item.closePrice ?? item.close_price,
        vwap: item.vwap
    };
}

function normalizeProjectSwapPairActivity(item: any): TokenProjectSwapPairActivity {
    const pairKind = item.pairKind ?? item.pair_kind;
    return {
        pairKind: pairKind === 'weth' || pairKind === 'usdt' ? pairKind : undefined,
        pairAddress: item.pairAddress ?? item.pair_address,
        status: item.status,
        swapBlockCount: numberValue(item.swapBlockCount ?? item.swap_block_count),
        targetSwapBlockCount: numberValue(item.targetSwapBlockCount ?? item.target_swap_block_count),
        startBlockNumber: exactStringValue(item.startBlockNumber ?? item.start_block_number),
        startBlockTime: item.startBlockTime ?? item.start_block_time,
        firstSwapBlockNumber: exactStringValue(item.firstSwapBlockNumber ?? item.first_swap_block_number),
        firstSwapBlockTime: item.firstSwapBlockTime ?? item.first_swap_block_time,
        lastSwapBlockNumber: exactStringValue(item.lastSwapBlockNumber ?? item.last_swap_block_number),
        lastSwapBlockTime: item.lastSwapBlockTime ?? item.last_swap_block_time,
        absoluteExpiryBlockTime: item.absoluteExpiryBlockTime ?? item.absolute_expiry_block_time,
        nextExpiryBlockTime: item.nextExpiryBlockTime ?? item.next_expiry_block_time,
        completedBlockNumber: exactStringValue(item.completedBlockNumber ?? item.completed_block_number),
        completedBlockTime: item.completedBlockTime ?? item.completed_block_time,
        expiredBlockNumber: exactStringValue(item.expiredBlockNumber ?? item.expired_block_number),
        expiredBlockTime: item.expiredBlockTime ?? item.expired_block_time,
        expiredReason: item.expiredReason ?? item.expired_reason,
        baseAsset: normalizeProjectSwapAsset(item.baseAsset ?? item.base_asset),
        quoteAsset: normalizeProjectSwapAsset(item.quoteAsset ?? item.quote_asset),
        eventCount: numberValue(item.eventCount ?? item.event_count),
        transactionCount: numberValue(item.transactionCount ?? item.transaction_count),
        transactionOriginCount: numberValue(item.transactionOriginCount ?? item.transaction_origin_count),
        buyEventCount: numberValue(item.buyEventCount ?? item.buy_event_count),
        sellEventCount: numberValue(item.sellEventCount ?? item.sell_event_count),
        complexEventCount: numberValue(item.complexEventCount ?? item.complex_event_count),
        baseAmountInRaw: item.baseAmountInRaw ?? item.base_amount_in_raw,
        baseAmountOutRaw: item.baseAmountOutRaw ?? item.base_amount_out_raw,
        quoteAmountInRaw: item.quoteAmountInRaw ?? item.quote_amount_in_raw,
        quoteAmountOutRaw: item.quoteAmountOutRaw ?? item.quote_amount_out_raw,
        buyQuoteAmountRaw: item.buyQuoteAmountRaw ?? item.buy_quote_amount_raw,
        sellQuoteAmountRaw: item.sellQuoteAmountRaw ?? item.sell_quote_amount_raw,
        blocks: ((item.blocks || []) as any[]).map(normalizeProjectSwapBlock)
    };
}

function normalizeProjectSwapActivity(item: any): TokenProjectSwapActivity {
    return {
        projectID: numberValue(item.projectID ?? item.projectId ?? item.project_id),
        chainID: numberValue(item.chainID ?? item.chainId ?? item.chain_id),
        generatedAt: item.generatedAt ?? item.generated_at,
        pairs: ((item.pairs || []) as any[]).map(normalizeProjectSwapPairActivity)
    };
}

function normalizeProjectSwapEvent(item: any): TokenProjectSwapEvent {
    const direction = item.direction;
    return {
        transactionHash: item.transactionHash ?? item.transaction_hash,
        transactionIndex: exactStringValue(item.transactionIndex ?? item.transaction_index),
        logIndex: exactStringValue(item.logIndex ?? item.log_index),
        txFrom: item.txFrom ?? item.tx_from,
        sender: item.sender,
        toAddress: item.toAddress ?? item.to_address,
        amount0In: item.amount0In ?? item.amount0_in,
        amount1In: item.amount1In ?? item.amount1_in,
        amount0Out: item.amount0Out ?? item.amount0_out,
        amount1Out: item.amount1Out ?? item.amount1_out,
        baseAmountInRaw: item.baseAmountInRaw ?? item.base_amount_in_raw,
        baseAmountOutRaw: item.baseAmountOutRaw ?? item.base_amount_out_raw,
        quoteAmountInRaw: item.quoteAmountInRaw ?? item.quote_amount_in_raw,
        quoteAmountOutRaw: item.quoteAmountOutRaw ?? item.quote_amount_out_raw,
        direction: direction === 'buy' || direction === 'sell' || direction === 'complex' ? direction : undefined,
        effectivePrice: item.effectivePrice ?? item.effective_price
    };
}

function normalizeWalletNormalTransaction(item: any): TokenWalletNormalTransaction {
    return {
        wallet: item.wallet,
        transactionHash: item.transactionHash ?? item.transaction_hash,
        blockNumber: numberValue(item.blockNumber ?? item.block_number),
        blockTimestamp: item.blockTimestamp ?? item.block_timestamp,
        transactionIndex: numberValue(item.transactionIndex ?? item.transaction_index),
        nonce: numberValue(item.nonce),
        fromAddress: item.fromAddress ?? item.from_address,
        toAddress: item.toAddress ?? item.to_address,
        value: item.value,
        gas: numberValue(item.gas),
        gasPrice: item.gasPrice ?? item.gas_price,
        gasUsed: numberValue(item.gasUsed ?? item.gas_used),
        input: item.input,
        methodID: item.methodID ?? item.methodId ?? item.method_id,
        functionName: item.functionName ?? item.function_name,
        receiptStatus: item.receiptStatus ?? item.receipt_status,
        isError: item.isError ?? item.is_error,
        collectedAt: item.collectedAt ?? item.collected_at
    };
}

export class TokenService {
    public getRuntimeConfiguration(): Promise<TokenRuntimeConfiguration> & {abort?: () => void} {
        const req = requests.get('/tokens/runtime-configuration');
        const promise = req.then(res => {
            const configuration = res.body?.configuration || {};
            return {
                chains: ((configuration.chains || []) as any[]).map(normalizeChainOption)
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listNodeStatuses(): Promise<TokenNodeStatus[]> & {abort?: () => void} {
        const req = requests.get('/tokens/node-statuses');
        const promise = req.then(res => ((res.body?.nodeStatuses || res.body?.node_statuses || []) as any[]).map(normalizeNodeStatus)) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listContractCodeBlocklistEntries(): Promise<TokenContractCodeBlocklistEntry[]> & {abort?: () => void} {
        const req = requests.get('/tokens/policies/contract-code-blocklist-entries');
        const promise = req.then(res => ((res.body?.entries || []) as any[]).map(normalizeContractCodeBlocklistEntry)) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public createContractCodeBlocklistEntry(values: {note?: string; sourceChainID?: number; sourceContract?: string}): Promise<void> & {abort?: () => void} {
        const req = requests.post('/tokens/policies/contract-code-blocklist-entries').send({
            note: values.note || '',
            sourceChainId: values.sourceChainID,
            source_chain_id: values.sourceChainID,
            sourceContract: values.sourceContract || '',
            source_contract: values.sourceContract || ''
        });
        const promise = req.then(() => undefined) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public updateContractCodeBlocklistEntry(codeHash: string, note: string): Promise<number> & {abort?: () => void} {
        const req = requests.patch(`/tokens/policies/contract-code-blocklist-entries/${encodeURIComponent(codeHash)}`).send({codeHash, code_hash: codeHash, note});
        const promise = req.then(res => numberValue(res.body?.updatedCount ?? res.body?.updated_count) || 0) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public deleteContractCodeBlocklistEntry(codeHash: string): Promise<number> & {abort?: () => void} {
        const req = requests.delete(`/tokens/policies/contract-code-blocklist-entries/${encodeURIComponent(codeHash)}`);
        const promise = req.then(res => numberValue(res.body?.deletedCount ?? res.body?.deleted_count) || 0) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listWalletBlocklistEntries(): Promise<TokenWalletBlocklistEntry[]> & {abort?: () => void} {
        const req = requests.get('/tokens/policies/wallet-blocklist-entries');
        const promise = req.then(res => ((res.body?.entries || []) as any[]).map(normalizeWalletBlocklistEntry)) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public createWalletBlocklistEntry(wallet: string, note: string): Promise<void> & {abort?: () => void} {
        const req = requests.post('/tokens/policies/wallet-blocklist-entries').send({wallet, note});
        const promise = req.then(() => undefined) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public updateWalletBlocklistEntry(wallet: string, note: string): Promise<number> & {abort?: () => void} {
        const req = requests.patch(`/tokens/policies/wallet-blocklist-entries/${encodeURIComponent(wallet)}`).send({wallet, note});
        const promise = req.then(res => numberValue(res.body?.updatedCount ?? res.body?.updated_count) || 0) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public deleteWalletBlocklistEntry(wallet: string): Promise<number> & {abort?: () => void} {
        const req = requests.delete(`/tokens/policies/wallet-blocklist-entries/${encodeURIComponent(wallet)}`);
        const promise = req.then(res => numberValue(res.body?.deletedCount ?? res.body?.deleted_count) || 0) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listChainCheckpoints(): Promise<TokenChainCheckpoint[]> & {abort?: () => void} {
        const req = requests.get('/tokens/chain-checkpoints');
        const promise = req.then(res => ((res.body?.checkpoints || []) as any[]).map(normalizeCheckpoint)) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public updateChainCheckpoint(chainID: number, status: string): Promise<TokenChainCheckpoint | undefined> & {abort?: () => void} {
        const req = requests.patch(`/tokens/chain-checkpoints/${encodeURIComponent(String(chainID))}`).send({status});
        const promise = req.then(res => {
            const item = res.body?.checkpoint;
            return item ? normalizeCheckpoint(item) : undefined;
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listContractCodes(
        options: {page?: number; pageSize?: number; codeHash?: string; orderBy?: string} = {}
    ): Promise<PagedResponse<TokenContractCode>> & {abort?: () => void} {
        const req = requests.get('/tokens/contract-codes').query({
            code_hash: options.codeHash || undefined,
            page: options.page,
            page_size: options.pageSize,
            order_by: options.orderBy || undefined
        });
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: ((body.contractCodes || body.contract_codes || []) as any[]).map(normalizeContractCode),
                total: numberValue(body.total) || 0,
                page: numberValue(body.page) || options.page || 1,
                pageSize: numberValue(body.pageSize ?? body.page_size) || options.pageSize || 20
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public getContractCode(codeHash: string): Promise<TokenContractCode | undefined> & {abort?: () => void} {
        const req = requests.get(`/tokens/contract-codes/${encodeURIComponent(codeHash)}`);
        const promise = req.then(res => {
            const item = res.body?.contractCode || res.body?.contract_code;
            return item ? normalizeContractCode(item) : undefined;
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listProjects(
        options: {
            page?: number;
            pageSize?: number;
            chainID?: number;
            projectID?: number;
            codeHash?: string;
            contract?: string;
            researchStatus?: string;
            reportState?: string;
            evaluationStatus?: string;
            selectionOutcome?: string;
            reportPairKind?: string;
            reportPairRemoveLiquidityStates?: string[];
            reportPairMintStates?: string[];
            reportPairQuoteUSDTMin?: string;
            reportPairQuoteUSDTMax?: string;
            reportPairQuoteMissingStates?: string[];
        } = {}
    ): Promise<PagedResponse<TokenProjectListItem>> & {abort?: () => void} {
        const req = requests.get('/tokens/projects').query({
            chain_id: options.chainID,
            project_id: options.projectID,
            code_hash: options.codeHash || undefined,
            contract: options.contract || undefined,
            research_status: options.researchStatus || undefined,
            report_state: options.reportState || undefined,
            evaluation_status: options.evaluationStatus || undefined,
            selection_outcome: options.selectionOutcome || undefined,
            report_pair_kind: options.reportPairKind || undefined,
            report_pair_remove_liquidity_states: options.reportPairRemoveLiquidityStates?.length ? options.reportPairRemoveLiquidityStates : undefined,
            report_pair_mint_states: options.reportPairMintStates?.length ? options.reportPairMintStates : undefined,
            report_pair_quote_usdt_min: options.reportPairQuoteUSDTMin || undefined,
            report_pair_quote_usdt_max: options.reportPairQuoteUSDTMax || undefined,
            report_pair_quote_missing_states: options.reportPairQuoteMissingStates?.length ? options.reportPairQuoteMissingStates : undefined,
            page: options.page,
            page_size: options.pageSize
        });
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: ((body.projects || []) as any[]).map(normalizeProjectListItem),
                total: numberValue(body.total) || 0,
                page: numberValue(body.page) || options.page || 1,
                pageSize: numberValue(body.pageSize ?? body.page_size) || options.pageSize || 20
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public getProjectDetail(projectID: number): Promise<TokenProjectDetail | undefined> & {abort?: () => void} {
        const req = requests.get(`/tokens/projects/${encodeURIComponent(String(projectID))}`);
        const promise = req.then(res => {
            const body = res.body || {};
            return body.found === false || !body.detail ? undefined : normalizeProjectDetail(body.detail);
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public getProjectSwapActivity(projectID: number): Promise<TokenProjectSwapActivity | undefined> & {abort?: () => void} {
        const req = requests.get(`/tokens/projects/${encodeURIComponent(String(projectID))}/swap-activity`);
        const promise = req.then(res => {
            const body = res.body || {};
            return body.found === false || !body.activity ? undefined : normalizeProjectSwapActivity(body.activity);
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listProjectSwapEvents(
        projectID: number,
        pairKind: TokenProjectSwapPairKind,
        blockNumber: string,
        options: {page?: number; pageSize?: number} = {}
    ): Promise<PagedResponse<TokenProjectSwapEvent>> & {abort?: () => void} {
        const req = requests
            .get(`/tokens/projects/${encodeURIComponent(String(projectID))}/swap-pairs/${encodeURIComponent(pairKind)}/blocks/${encodeURIComponent(String(blockNumber))}/events`)
            .query({page: options.page, page_size: options.pageSize});
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: ((body.events || []) as any[]).map(normalizeProjectSwapEvent),
                total: numberValue(body.total) || 0,
                page: numberValue(body.page) || options.page || 1,
                pageSize: numberValue(body.pageSize ?? body.page_size) || options.pageSize || 50
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listProjectTrends(projectID: number, range = '24h'): Promise<TokenProjectTrends> & {abort?: () => void} {
        const req = requests.get(`/tokens/projects/${encodeURIComponent(String(projectID))}/trends`).query({range});
        const promise = req.then(res => {
            const item = res.body?.trends || {};
            return {
                range: item.range,
                observedFrom: item.observedFrom ?? item.observed_from,
                generatedAt: item.generatedAt ?? item.generated_at,
                series: ((item.series || []) as any[]).map(series => ({
                    key: series.key,
                    label: series.label,
                    unit: series.unit,
                    dataType: series.dataType ?? series.data_type,
                    points: ((series.points || []) as any[]).map(point => ({
                        observedAt: point.observedAt ?? point.observed_at,
                        value: point.value
                    }))
                }))
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listProjectObservations(
        projectID: number,
        options: {dataType?: string; page?: number; pageSize?: number} = {}
    ): Promise<PagedResponse<TokenProjectObservation>> & {abort?: () => void} {
        const req = requests.get(`/tokens/projects/${encodeURIComponent(String(projectID))}/observations`).query({
            data_type: options.dataType || undefined,
            page: options.page,
            page_size: options.pageSize
        });
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: ((body.observations || []) as any[]).map(normalizeObservation),
                total: numberValue(body.total) || 0,
                page: numberValue(body.page) || options.page || 1,
                pageSize: numberValue(body.pageSize ?? body.page_size) || options.pageSize || 20
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listProjectWalletNormalTransactions(
        projectID: number,
        options: {wallet?: string; receiptStatus?: string; methodID?: string; page?: number; pageSize?: number} = {}
    ): Promise<PagedResponse<TokenWalletNormalTransaction>> & {abort?: () => void} {
        const req = requests.get(`/tokens/projects/${encodeURIComponent(String(projectID))}/wallet-normal-transactions`).query({
            wallet: options.wallet || undefined,
            receipt_status: options.receiptStatus || undefined,
            method_id: options.methodID || undefined,
            page: options.page,
            page_size: options.pageSize
        });
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: ((body.transactions || []) as any[]).map(normalizeWalletNormalTransaction),
                total: numberValue(body.total) || 0,
                page: numberValue(body.page) || options.page || 1,
                pageSize: numberValue(body.pageSize ?? body.page_size) || options.pageSize || 20
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listCollectionTasks(
        options: {page?: number; pageSize?: number; projectID?: number; dataType?: string; status?: string} = {}
    ): Promise<PagedResponse<TokenCollectionTask>> & {abort?: () => void} {
        const req = requests.get('/tokens/collection-tasks').query({
            project_id: options.projectID,
            data_type: options.dataType || undefined,
            status: options.status || undefined,
            page: options.page,
            page_size: options.pageSize
        });
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: ((body.tasks || []) as any[]).map(normalizeTask),
                total: numberValue(body.total) || 0,
                page: numberValue(body.page) || options.page || 1,
                pageSize: numberValue(body.pageSize ?? body.page_size) || options.pageSize || 20
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listReportRevisions(
        options: {projectID?: number; chainID?: number; page?: number; pageSize?: number} = {}
    ): Promise<PagedResponse<TokenReportRevision>> & {abort?: () => void} {
        const req = requests.get('/tokens/report-revisions').query({
            project_id: options.projectID,
            chain_id: options.chainID,
            page: options.page,
            page_size: options.pageSize
        });
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: ((body.reportRevisions ?? body.report_revisions ?? []) as any[]).map(normalizeReportRevision),
                total: numberValue(body.total) || 0,
                page: numberValue(body.page) || options.page || 1,
                pageSize: numberValue(body.pageSize ?? body.page_size) || options.pageSize || 20
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listSelections(
        options: {projectID?: number; chainID?: number; outcome?: string; page?: number; pageSize?: number} = {}
    ): Promise<PagedResponse<TokenSelection>> & {abort?: () => void} {
        const req = requests.get('/tokens/selections').query({
            project_id: options.projectID,
            chain_id: options.chainID,
            outcome: options.outcome || undefined,
            page: options.page,
            page_size: options.pageSize
        });
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: ((body.selections || []) as any[]).map(normalizeSelection),
                total: numberValue(body.total) || 0,
                page: numberValue(body.page) || options.page || 1,
                pageSize: numberValue(body.pageSize ?? body.page_size) || options.pageSize || 20
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }
}
