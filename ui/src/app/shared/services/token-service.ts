import requests from './requests';
import {AccountDataModule} from '../access-modules';

const readScope = {module: AccountDataModule.Token, mode: 'read' as const};
const writeScope = {module: AccountDataModule.Token, mode: 'write' as const};

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
    updatedAt?: string;
}

export interface TokenChainProcessingAttempt {
    attemptID?: number;
    chainID?: number;
    blockNumber?: number;
    attemptNumber?: number;
    blockTime?: number;
    status?: string;
    terminalStage?: string;
    errorMessage?: string;
    checkpointReadDurationUS?: number;
    discoveryDurationUS?: number;
    validationDurationUS?: number;
    persistenceDurationUS?: number;
    totalDurationUS?: number;
    candidateCount?: number;
    validatedCount?: number;
    rejectedCount?: number;
    timingComplete?: boolean;
    startedAt?: string;
    completedAt?: string;
    createdAt?: string;
    updatedAt?: string;
}

export interface TokenChainProcessingSummary {
    chainID?: number;
    rangeStartBlockTime?: number;
    rangeEndBlockTime?: number;
    attemptCount?: number;
    runningCount?: number;
    succeededCount?: number;
    failedCount?: number;
    cancelledCount?: number;
    interruptedCount?: number;
    incompleteSucceededCount?: number;
    measuredSucceededCount?: number;
    failureRateBPS?: number;
    averageDurationUS?: number;
    averageCheckpointReadDurationUS?: number;
    averageDiscoveryDurationUS?: number;
    averageValidationDurationUS?: number;
    averagePersistenceDurationUS?: number;
    fastestBlockNumber?: number;
    fastestDurationUS?: number;
    slowestBlockNumber?: number;
    slowestDurationUS?: number;
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

export interface TokenProjectMarketSummary {
    logoURL?: string;
    currentPriceUSD?: string;
    marketCapUSD?: string;
    fdvUSD?: string;
    tvlUSD?: string;
    holders?: number;
}

export interface TokenProjectPairProfileSummary {
    kind?: string;
    address?: string;
    isCreated?: boolean;
    quoteUsdtValueInt?: string;
    reserveUpdatedAt?: number;
    pairTokenBalanceExceedsTotalSupply?: boolean;
    lpMinimumSupplyOnly?: boolean;
    fixedFeeAddressLpShareGte90Percent?: boolean;
}

export interface TokenProjectListItem {
    projectID?: number;
    chainID?: number;
    name?: string;
    symbol?: string;
    contract?: string;
    codeHash?: string;
    blockNumber?: number;
    blockTime?: number;
    txHash?: string;
    createdAt?: string;
    collectionStatus?: string;
    collectionSucceededCount?: number;
    collectionTerminalCount?: number;
    collectionTotalCount?: number;
    profileState?: string;
    completenessStatus?: string;
    profileBuiltAt?: string;
    market?: TokenProjectMarketSummary;
    wrappedNativePair?: TokenProjectPairProfileSummary;
    usdtPair?: TokenProjectPairProfileSummary;
}

export interface TokenCollectionResult {
    taskID?: number;
    projectID?: number;
    dataType?: string;
    schemaVersion?: number;
    payloadJSON?: string;
    contentHash?: string;
    blockNumber?: number;
    collectedAt?: string;
}

export interface TokenCollectionTask {
    taskID?: number;
    projectID?: number;
    dataType?: string;
    status?: string;
    failureCount?: number;
    availableAt?: string;
    claimGeneration?: number;
    lockedAt?: string;
    leaseExpiresAt?: string;
    lastError?: string;
    finishedAt?: string;
    createdAt?: string;
    updatedAt?: string;
    result?: TokenCollectionResult;
}

export interface TokenProjectProfileMarket extends TokenProjectMarketSummary {
    currentPriceETH?: string;
    mainPairTVLUSD?: string;
    launchAt?: string;
    providerUpdatedAt?: string;
    aveRisk?: TokenProjectProfileAveRisk;
}

export interface TokenProjectProfileAveRisk {
    riskLevel?: number;
    riskScore?: string;
    riskInfo?: string;
    audited?: boolean;
    mintable?: boolean;
    hasMintMethod?: boolean;
    liquidityPoolUnlocked?: boolean;
    ownershipNotRenounced?: boolean;
    notAudited?: boolean;
    notOpenSource?: boolean;
    inBlacklist?: boolean;
    honeypot?: boolean;
}

export interface TokenProjectProfileSource {
    codeHash?: string;
    verificationStatus?: string;
    artifactReference?: string;
}

export interface TokenProjectProfilePair {
    kind?: string;
    address?: string;
    chainState?: TokenProjectProfilePairChainState;
    market?: TokenProjectProfilePairMarket;
}

export interface TokenProjectProfilePairChainState {
    isCreated?: boolean;
    baseBalance?: string;
    quoteBalance?: string;
    quoteUsdtValue?: string;
    quoteUsdtValueInt?: string;
    reserveUpdatedAt?: number;
    liquidity?: TokenProjectProfilePairLiquidity;
    signals?: TokenProjectProfilePairSignals;
}

export interface TokenProjectProfilePairLiquidity {
    totalSupply?: string;
    lockedLiquidity?: string;
    fixedFeeAddressBalance?: string;
    fixedFeeAddressShare?: string;
}

export interface TokenProjectProfilePairSignals {
    pairTokenBalanceExceedsTotalSupply?: boolean;
    lpMinimumSupplyOnly?: boolean;
    fixedFeeAddressLpShareGte90Percent?: boolean;
}

export interface TokenProjectProfilePairMarket {
    amm?: string;
    token0Address?: string;
    token0Symbol?: string;
    token1Address?: string;
    token1Symbol?: string;
    reserve0?: string;
    reserve1?: string;
    volumeUSD?: string;
    marketCapUSD?: string;
    fdvUSD?: string;
    isFake?: boolean;
    createdAt?: string;
    updatedAt?: string;
}

export interface TokenProjectProfileWalletSummary {
    walletCount?: number;
    nativeBalanceTotal?: string;
    wrappedNativeBalanceTotal?: string;
    usdtBalanceTotal?: string;
    trackedAssetUsdtValueTotal?: string;
    initialRecipientCount?: number;
    initialRecipientAllocationBPS?: number;
    walletsWithSimulationSignals?: number;
    roleCounts: TokenProjectProfileWalletRoleCount[];
}

export interface TokenProjectProfileWalletRoleCount {
    role?: string;
    count?: number;
}

export interface TokenProjectProfileWallet {
    address?: string;
    roles: string[];
    initialRecipient?: TokenProjectProfileInitialRecipient;
    assets?: TokenProjectProfileWalletAssets;
    simulation?: TokenProjectProfileWalletSimulation;
    transactionSampleCapped?: boolean;
}

export interface TokenProjectProfileInitialRecipient {
    rank?: number;
    ratioBPS?: number;
}

export interface TokenProjectProfileWalletAssets {
    nativeBalance?: string;
    wrappedNativeBalance?: string;
    usdtBalance?: string;
    trackedAssetUsdtValue?: string;
}

export interface TokenProjectProfileWalletSimulation {
    transferFromDeadToWalletCallSucceeded?: boolean;
    transferFromZeroToWalletCallSucceeded?: boolean;
    transferFromWethPairToWalletCallSucceeded?: boolean;
    transferFromUsdtPairToWalletCallSucceeded?: boolean;
    transferFromWalletToWethPairCallSucceeded?: boolean;
    transferFromWalletToUsdtPairCallSucceeded?: boolean;
}

export interface TokenProjectProfileTransactions {
    walletCount?: number;
    transactionAssociationCount?: number;
    uniqueTransactionCount?: number;
    succeededTransactionCount?: number;
    failedTransactionCount?: number;
    totalInflowNativeValue?: string;
    totalOutflowNativeValue?: string;
    cappedWallets: string[];
    topMethods: TokenProjectProfileTransactionMethod[];
    topCounterparties: TokenProjectProfileTransactionCounterparty[];
}

export interface TokenProjectProfileTransactionMethod {
    methodID?: string;
    functionName?: string;
    count?: number;
}

export interface TokenProjectProfileTransactionCounterparty {
    address?: string;
    count?: number;
}

export interface TokenProjectProfileEvidence {
    taskID?: number;
    dataType?: string;
    status?: string;
    failureCount?: number;
    lastError?: string;
    resultSchemaVersion?: number;
    resultContentHash?: string;
    blockNumber?: number;
    collectedAt?: string;
}

export interface TokenProjectProfile {
    projectID?: number;
    schemaVersion?: number;
    completenessStatus?: string;
    failedDataTypes: string[];
    market?: TokenProjectProfileMarket;
    contractSource?: TokenProjectProfileSource;
    wrappedNativePair?: TokenProjectProfilePair;
    usdtPair?: TokenProjectProfilePair;
    walletSummary?: TokenProjectProfileWalletSummary;
    wallets: TokenProjectProfileWallet[];
    transactions?: TokenProjectProfileTransactions;
    evidence: TokenProjectProfileEvidence[];
    profileJSON?: string;
    contentHash?: string;
    builtAt?: string;
    createdAt?: string;
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

export interface TokenProjectDetail {
    project?: TokenProject;
    profile?: TokenProjectProfile;
    collectionStatus?: string;
    profileState?: string;
    collectionTasks: TokenCollectionTask[];
    relatedWallets: TokenProjectRelatedWallet[];
    initialRecipients: TokenProjectInitialRecipient[];
    walletTransactionCounts: Array<{wallet?: string; transactionCount?: number}>;
    transactionCount?: number;
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
        createdAt: item.createdAt ?? item.created_at,
        updatedAt: item.updatedAt ?? item.updated_at
    };
}

function normalizeChainProcessingAttempt(item: any): TokenChainProcessingAttempt {
    return {
        attemptID: numberValue(item.attemptID ?? item.attemptId ?? item.attempt_id),
        chainID: numberValue(item.chainID ?? item.chainId ?? item.chain_id),
        blockNumber: numberValue(item.blockNumber ?? item.block_number),
        attemptNumber: numberValue(item.attemptNumber ?? item.attempt_number),
        blockTime: numberValue(item.blockTime ?? item.block_time),
        status: item.status,
        terminalStage: item.terminalStage ?? item.terminal_stage,
        errorMessage: item.errorMessage ?? item.error_message,
        checkpointReadDurationUS: numberValue(item.checkpointReadDurationUS ?? item.checkpointReadDurationUs ?? item.checkpoint_read_duration_us),
        discoveryDurationUS: numberValue(item.discoveryDurationUS ?? item.discoveryDurationUs ?? item.discovery_duration_us),
        validationDurationUS: numberValue(item.validationDurationUS ?? item.validationDurationUs ?? item.validation_duration_us),
        persistenceDurationUS: numberValue(item.persistenceDurationUS ?? item.persistenceDurationUs ?? item.persistence_duration_us),
        totalDurationUS: numberValue(item.totalDurationUS ?? item.totalDurationUs ?? item.total_duration_us),
        candidateCount: numberValue(item.candidateCount ?? item.candidate_count),
        validatedCount: numberValue(item.validatedCount ?? item.validated_count),
        rejectedCount: numberValue(item.rejectedCount ?? item.rejected_count),
        timingComplete: item.timingComplete ?? item.timing_complete,
        startedAt: item.startedAt ?? item.started_at,
        completedAt: item.completedAt ?? item.completed_at,
        createdAt: item.createdAt ?? item.created_at,
        updatedAt: item.updatedAt ?? item.updated_at
    };
}

function normalizeChainProcessingSummary(item: any): TokenChainProcessingSummary {
    return {
        chainID: numberValue(item.chainID ?? item.chainId ?? item.chain_id),
        rangeStartBlockTime: numberValue(item.rangeStartBlockTime ?? item.range_start_block_time),
        rangeEndBlockTime: numberValue(item.rangeEndBlockTime ?? item.range_end_block_time),
        attemptCount: numberValue(item.attemptCount ?? item.attempt_count),
        runningCount: numberValue(item.runningCount ?? item.running_count),
        succeededCount: numberValue(item.succeededCount ?? item.succeeded_count),
        failedCount: numberValue(item.failedCount ?? item.failed_count),
        cancelledCount: numberValue(item.cancelledCount ?? item.cancelled_count),
        interruptedCount: numberValue(item.interruptedCount ?? item.interrupted_count),
        incompleteSucceededCount: numberValue(item.incompleteSucceededCount ?? item.incomplete_succeeded_count),
        measuredSucceededCount: numberValue(item.measuredSucceededCount ?? item.measured_succeeded_count),
        failureRateBPS: numberValue(item.failureRateBPS ?? item.failureRateBps ?? item.failure_rate_bps),
        averageDurationUS: numberValue(item.averageDurationUS ?? item.averageDurationUs ?? item.average_duration_us),
        averageCheckpointReadDurationUS: numberValue(item.averageCheckpointReadDurationUS ?? item.averageCheckpointReadDurationUs ?? item.average_checkpoint_read_duration_us),
        averageDiscoveryDurationUS: numberValue(item.averageDiscoveryDurationUS ?? item.averageDiscoveryDurationUs ?? item.average_discovery_duration_us),
        averageValidationDurationUS: numberValue(item.averageValidationDurationUS ?? item.averageValidationDurationUs ?? item.average_validation_duration_us),
        averagePersistenceDurationUS: numberValue(item.averagePersistenceDurationUS ?? item.averagePersistenceDurationUs ?? item.average_persistence_duration_us),
        fastestBlockNumber: numberValue(item.fastestBlockNumber ?? item.fastest_block_number),
        fastestDurationUS: numberValue(item.fastestDurationUS ?? item.fastestDurationUs ?? item.fastest_duration_us),
        slowestBlockNumber: numberValue(item.slowestBlockNumber ?? item.slowest_block_number),
        slowestDurationUS: numberValue(item.slowestDurationUS ?? item.slowestDurationUs ?? item.slowest_duration_us)
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

function normalizeProjectMarketSummary(item: any): TokenProjectMarketSummary | undefined {
    if (!item) {
        return undefined;
    }
    return {
        logoURL: item.logoURL ?? item.logoUrl ?? item.logo_url,
        currentPriceUSD: item.currentPriceUSD ?? item.currentPriceUsd ?? item.current_price_usd,
        marketCapUSD: item.marketCapUSD ?? item.marketCapUsd ?? item.market_cap_usd,
        fdvUSD: item.fdvUSD ?? item.fdvUsd ?? item.fdv_usd,
        tvlUSD: item.tvlUSD ?? item.tvlUsd ?? item.tvl_usd,
        holders: numberValue(item.holders)
    };
}

function normalizeProjectPairProfileSummary(item: any): TokenProjectPairProfileSummary | undefined {
    if (!item) {
        return undefined;
    }
    return {
        kind: item.kind,
        address: item.address,
        isCreated: item.isCreated ?? item.is_created,
        quoteUsdtValueInt: item.quoteUsdtValueInt ?? item.quote_usdt_value_int,
        reserveUpdatedAt: numberValue(item.reserveUpdatedAt ?? item.reserve_updated_at),
        pairTokenBalanceExceedsTotalSupply: item.pairTokenBalanceExceedsTotalSupply ?? item.pair_token_balance_exceeds_total_supply,
        lpMinimumSupplyOnly: item.lpMinimumSupplyOnly ?? item.lp_minimum_supply_only,
        fixedFeeAddressLpShareGte90Percent:
            item.fixedFeeAddressLpShareGte90Percent ?? item.fixedFeeAddressLPShareGte90Percent ?? item.fixed_fee_address_lp_share_gte_90_percent
    };
}

function normalizeProjectListItem(item: any): TokenProjectListItem {
    return {
        projectID: numberValue(item.projectID ?? item.projectId ?? item.project_id),
        chainID: numberValue(item.chainID ?? item.chainId ?? item.chain_id),
        name: item.name,
        symbol: item.symbol,
        contract: item.contract,
        codeHash: item.codeHash ?? item.code_hash,
        blockNumber: numberValue(item.blockNumber ?? item.block_number),
        blockTime: numberValue(item.blockTime ?? item.block_time),
        txHash: item.txHash ?? item.tx_hash,
        createdAt: item.createdAt ?? item.created_at,
        collectionStatus: item.collectionStatus ?? item.collection_status,
        collectionSucceededCount: numberValue(item.collectionSucceededCount ?? item.collection_succeeded_count),
        collectionTerminalCount: numberValue(item.collectionTerminalCount ?? item.collection_terminal_count),
        collectionTotalCount: numberValue(item.collectionTotalCount ?? item.collection_total_count),
        profileState: item.profileState ?? item.profile_state,
        completenessStatus: item.completenessStatus ?? item.completeness_status,
        profileBuiltAt: item.profileBuiltAt ?? item.profile_built_at,
        market: normalizeProjectMarketSummary(item.market),
        wrappedNativePair: normalizeProjectPairProfileSummary(item.wrappedNativePair ?? item.wrapped_native_pair),
        usdtPair: normalizeProjectPairProfileSummary(item.usdtPair ?? item.usdt_pair)
    };
}

function normalizeCollectionResult(item: any): TokenCollectionResult | undefined {
    if (!item) {
        return undefined;
    }
    return {
        taskID: numberValue(item.taskID ?? item.taskId ?? item.task_id),
        projectID: numberValue(item.projectID ?? item.projectId ?? item.project_id),
        dataType: item.dataType ?? item.data_type,
        schemaVersion: numberValue(item.schemaVersion ?? item.schema_version),
        payloadJSON: item.payloadJSON ?? item.payloadJson ?? item.payload_json,
        contentHash: item.contentHash ?? item.content_hash,
        blockNumber: numberValue(item.blockNumber ?? item.block_number),
        collectedAt: item.collectedAt ?? item.collected_at
    };
}

function normalizeTask(item: any): TokenCollectionTask {
    return {
        taskID: numberValue(item.taskID ?? item.taskId ?? item.task_id),
        projectID: numberValue(item.projectID ?? item.projectId ?? item.project_id),
        dataType: item.dataType ?? item.data_type,
        status: item.status,
        failureCount: numberValue(item.failureCount ?? item.failure_count),
        availableAt: item.availableAt ?? item.available_at,
        claimGeneration: numberValue(item.claimGeneration ?? item.claim_generation),
        lockedAt: item.lockedAt ?? item.locked_at,
        leaseExpiresAt: item.leaseExpiresAt ?? item.lease_expires_at,
        lastError: item.lastError ?? item.last_error,
        finishedAt: item.finishedAt ?? item.finished_at,
        createdAt: item.createdAt ?? item.created_at,
        updatedAt: item.updatedAt ?? item.updated_at,
        result: normalizeCollectionResult(item.result)
    };
}

function normalizeProjectProfileMarket(item: any): TokenProjectProfileMarket | undefined {
    const summary = normalizeProjectMarketSummary(item);
    if (!summary) {
        return undefined;
    }
    return {
        ...summary,
        currentPriceETH: item.currentPriceETH ?? item.currentPriceEth ?? item.current_price_eth,
        mainPairTVLUSD: item.mainPairTVLUSD ?? item.mainPairTvlUsd ?? item.main_pair_tvl_usd,
        launchAt: item.launchAt ?? item.launch_at,
        providerUpdatedAt: item.providerUpdatedAt ?? item.provider_updated_at,
        aveRisk: normalizeProjectProfileAveRisk(item.aveRisk ?? item.ave_risk)
    };
}

function normalizeProjectProfileAveRisk(item: any): TokenProjectProfileAveRisk | undefined {
    if (!item) {
        return undefined;
    }
    return {
        riskLevel: numberValue(item.riskLevel ?? item.risk_level),
        riskScore: item.riskScore ?? item.risk_score,
        riskInfo: item.riskInfo ?? item.risk_info,
        audited: item.audited,
        mintable: item.mintable,
        hasMintMethod: item.hasMintMethod ?? item.has_mint_method,
        liquidityPoolUnlocked: item.liquidityPoolUnlocked ?? item.liquidity_pool_unlocked,
        ownershipNotRenounced: item.ownershipNotRenounced ?? item.ownership_not_renounced,
        notAudited: item.notAudited ?? item.not_audited,
        notOpenSource: item.notOpenSource ?? item.not_open_source,
        inBlacklist: item.inBlacklist ?? item.in_blacklist,
        honeypot: item.honeypot
    };
}

function normalizeProjectProfilePairLiquidity(item: any): TokenProjectProfilePairLiquidity | undefined {
    if (!item) {
        return undefined;
    }
    return {
        totalSupply: item.totalSupply ?? item.total_supply,
        lockedLiquidity: item.lockedLiquidity ?? item.locked_liquidity,
        fixedFeeAddressBalance: item.fixedFeeAddressBalance ?? item.fixed_fee_address_balance,
        fixedFeeAddressShare: item.fixedFeeAddressShare ?? item.fixed_fee_address_share
    };
}

function normalizeProjectProfilePairSignals(item: any): TokenProjectProfilePairSignals | undefined {
    if (!item) {
        return undefined;
    }
    return {
        pairTokenBalanceExceedsTotalSupply: item.pairTokenBalanceExceedsTotalSupply ?? item.pair_token_balance_exceeds_total_supply,
        lpMinimumSupplyOnly: item.lpMinimumSupplyOnly ?? item.lp_minimum_supply_only,
        fixedFeeAddressLpShareGte90Percent:
            item.fixedFeeAddressLpShareGte90Percent ?? item.fixedFeeAddressLPShareGte90Percent ?? item.fixed_fee_address_lp_share_gte_90_percent
    };
}

function normalizeProjectProfilePairChainState(item: any): TokenProjectProfilePairChainState | undefined {
    if (!item) {
        return undefined;
    }
    return {
        isCreated: item.isCreated ?? item.is_created,
        baseBalance: item.baseBalance ?? item.base_balance,
        quoteBalance: item.quoteBalance ?? item.quote_balance,
        quoteUsdtValue: item.quoteUsdtValue ?? item.quote_usdt_value,
        quoteUsdtValueInt: item.quoteUsdtValueInt ?? item.quote_usdt_value_int,
        reserveUpdatedAt: numberValue(item.reserveUpdatedAt ?? item.reserve_updated_at),
        liquidity: normalizeProjectProfilePairLiquidity(item.liquidity),
        signals: normalizeProjectProfilePairSignals(item.signals)
    };
}

function normalizeProjectProfilePairMarket(item: any): TokenProjectProfilePairMarket | undefined {
    if (!item) {
        return undefined;
    }
    return {
        amm: item.amm ?? item.AMM,
        token0Address: item.token0Address ?? item.token0_address,
        token0Symbol: item.token0Symbol ?? item.token0_symbol,
        token1Address: item.token1Address ?? item.token1_address,
        token1Symbol: item.token1Symbol ?? item.token1_symbol,
        reserve0: item.reserve0,
        reserve1: item.reserve1,
        volumeUSD: item.volumeUSD ?? item.volumeUsd ?? item.volume_usd,
        marketCapUSD: item.marketCapUSD ?? item.marketCapUsd ?? item.market_cap_usd,
        fdvUSD: item.fdvUSD ?? item.fdvUsd ?? item.fdv_usd,
        isFake: item.isFake ?? item.is_fake,
        createdAt: item.createdAt ?? item.created_at,
        updatedAt: item.updatedAt ?? item.updated_at
    };
}

function normalizeProjectProfilePair(item: any): TokenProjectProfilePair | undefined {
    if (!item) {
        return undefined;
    }
    return {
        kind: item.kind,
        address: item.address,
        chainState: normalizeProjectProfilePairChainState(item.chainState ?? item.chain_state),
        market: normalizeProjectProfilePairMarket(item.market)
    };
}

function normalizeProjectProfileWalletSummary(item: any): TokenProjectProfileWalletSummary | undefined {
    if (!item) {
        return undefined;
    }
    return {
        walletCount: numberValue(item.walletCount ?? item.wallet_count),
        nativeBalanceTotal: item.nativeBalanceTotal ?? item.native_balance_total,
        wrappedNativeBalanceTotal: item.wrappedNativeBalanceTotal ?? item.wrapped_native_balance_total,
        usdtBalanceTotal: item.usdtBalanceTotal ?? item.usdt_balance_total,
        trackedAssetUsdtValueTotal: item.trackedAssetUsdtValueTotal ?? item.tracked_asset_usdt_value_total,
        initialRecipientCount: numberValue(item.initialRecipientCount ?? item.initial_recipient_count),
        initialRecipientAllocationBPS: numberValue(item.initialRecipientAllocationBPS ?? item.initialRecipientAllocationBps ?? item.initial_recipient_allocation_bps),
        walletsWithSimulationSignals: numberValue(item.walletsWithSimulationSignals ?? item.wallets_with_simulation_signals),
        roleCounts: ((item.roleCounts ?? item.role_counts ?? []) as any[]).map(roleCount => ({
            role: roleCount.role,
            count: numberValue(roleCount.count)
        }))
    };
}

function normalizeProjectProfileWallet(item: any): TokenProjectProfileWallet {
    const initialRecipient = item.initialRecipient ?? item.initial_recipient;
    const assets = item.assets;
    const simulation = item.simulation;
    return {
        address: item.address,
        roles: item.roles ?? [],
        initialRecipient: initialRecipient
            ? {
                  rank: numberValue(initialRecipient.rank),
                  ratioBPS: numberValue(initialRecipient.ratioBPS ?? initialRecipient.ratioBps ?? initialRecipient.ratio_bps)
              }
            : undefined,
        assets: assets
            ? {
                  nativeBalance: assets.nativeBalance ?? assets.native_balance,
                  wrappedNativeBalance: assets.wrappedNativeBalance ?? assets.wrapped_native_balance,
                  usdtBalance: assets.usdtBalance ?? assets.usdt_balance,
                  trackedAssetUsdtValue: assets.trackedAssetUsdtValue ?? assets.tracked_asset_usdt_value
              }
            : undefined,
        simulation: simulation
            ? {
                  transferFromDeadToWalletCallSucceeded:
                      simulation.transferFromDeadToWalletCallSucceeded ?? simulation.transfer_from_dead_to_wallet_call_succeeded,
                  transferFromZeroToWalletCallSucceeded:
                      simulation.transferFromZeroToWalletCallSucceeded ?? simulation.transfer_from_zero_to_wallet_call_succeeded,
                  transferFromWethPairToWalletCallSucceeded:
                      simulation.transferFromWethPairToWalletCallSucceeded ?? simulation.transfer_from_weth_pair_to_wallet_call_succeeded,
                  transferFromUsdtPairToWalletCallSucceeded:
                      simulation.transferFromUsdtPairToWalletCallSucceeded ?? simulation.transfer_from_usdt_pair_to_wallet_call_succeeded,
                  transferFromWalletToWethPairCallSucceeded:
                      simulation.transferFromWalletToWethPairCallSucceeded ?? simulation.transfer_from_wallet_to_weth_pair_call_succeeded,
                  transferFromWalletToUsdtPairCallSucceeded:
                      simulation.transferFromWalletToUsdtPairCallSucceeded ?? simulation.transfer_from_wallet_to_usdt_pair_call_succeeded
              }
            : undefined,
        transactionSampleCapped: item.transactionSampleCapped ?? item.transaction_sample_capped
    };
}

function normalizeProjectProfile(item: any): TokenProjectProfile | undefined {
    if (!item) {
        return undefined;
    }
    const source = item.contractSource ?? item.contract_source;
    const transactions = item.transactions;
    return {
        projectID: numberValue(item.projectID ?? item.projectId ?? item.project_id),
        schemaVersion: numberValue(item.schemaVersion ?? item.schema_version),
        completenessStatus: item.completenessStatus ?? item.completeness_status,
        failedDataTypes: item.failedDataTypes ?? item.failed_data_types ?? [],
        market: normalizeProjectProfileMarket(item.market),
        contractSource: source
            ? {
                  codeHash: source.codeHash ?? source.code_hash,
                  verificationStatus: source.verificationStatus ?? source.verification_status,
                  artifactReference: source.artifactReference ?? source.artifact_reference
              }
            : undefined,
        wrappedNativePair: normalizeProjectProfilePair(item.wrappedNativePair ?? item.wrapped_native_pair),
        usdtPair: normalizeProjectProfilePair(item.usdtPair ?? item.usdt_pair),
        walletSummary: normalizeProjectProfileWalletSummary(item.walletSummary ?? item.wallet_summary),
        wallets: ((item.wallets || []) as any[]).map(normalizeProjectProfileWallet),
        transactions: transactions
            ? {
                  walletCount: numberValue(transactions.walletCount ?? transactions.wallet_count),
                  transactionAssociationCount: numberValue(transactions.transactionAssociationCount ?? transactions.transaction_association_count),
                  uniqueTransactionCount: numberValue(transactions.uniqueTransactionCount ?? transactions.unique_transaction_count),
                  succeededTransactionCount: numberValue(transactions.succeededTransactionCount ?? transactions.succeeded_transaction_count),
                  failedTransactionCount: numberValue(transactions.failedTransactionCount ?? transactions.failed_transaction_count),
                  totalInflowNativeValue: transactions.totalInflowNativeValue ?? transactions.total_inflow_native_value,
                  totalOutflowNativeValue: transactions.totalOutflowNativeValue ?? transactions.total_outflow_native_value,
                  cappedWallets: transactions.cappedWallets ?? transactions.capped_wallets ?? [],
                  topMethods: ((transactions.topMethods ?? transactions.top_methods ?? []) as any[]).map(method => ({
                      methodID: method.methodID ?? method.methodId ?? method.method_id,
                      functionName: method.functionName ?? method.function_name,
                      count: numberValue(method.count)
                  })),
                  topCounterparties: ((transactions.topCounterparties ?? transactions.top_counterparties ?? []) as any[]).map(counterparty => ({
                      address: counterparty.address,
                      count: numberValue(counterparty.count)
                  }))
              }
            : undefined,
        evidence: ((item.evidence || []) as any[]).map(evidence => ({
            taskID: numberValue(evidence.taskID ?? evidence.taskId ?? evidence.task_id),
            dataType: evidence.dataType ?? evidence.data_type,
            status: evidence.status,
            failureCount: numberValue(evidence.failureCount ?? evidence.failure_count),
            lastError: evidence.lastError ?? evidence.last_error,
            resultSchemaVersion: numberValue(evidence.resultSchemaVersion ?? evidence.result_schema_version),
            resultContentHash: evidence.resultContentHash ?? evidence.result_content_hash,
            blockNumber: numberValue(evidence.blockNumber ?? evidence.block_number),
            collectedAt: evidence.collectedAt ?? evidence.collected_at
        })),
        profileJSON: item.profileJSON ?? item.profileJson ?? item.profile_json,
        contentHash: item.contentHash ?? item.content_hash,
        builtAt: item.builtAt ?? item.built_at,
        createdAt: item.createdAt ?? item.created_at
    };
}

function normalizeProjectDetail(item: any): TokenProjectDetail {
    return {
        project: item.project ? normalizeProject(item.project) : undefined,
        profile: normalizeProjectProfile(item.profile),
        collectionStatus: item.collectionStatus ?? item.collection_status,
        profileState: item.profileState ?? item.profile_state,
        collectionTasks: ((item.collectionTasks ?? item.collection_tasks ?? []) as any[]).map(normalizeTask),
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
        walletTransactionCounts: ((item.walletTransactionCounts ?? item.wallet_transaction_counts ?? []) as any[]).map(count => ({
            wallet: count.wallet,
            transactionCount: numberValue(count.transactionCount ?? count.transaction_count)
        })),
        transactionCount: numberValue(item.transactionCount ?? item.transaction_count),
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
        const req = requests.get('/tokens/runtime-configuration', readScope);
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
        const req = requests.get('/tokens/node-statuses', readScope);
        const promise = req.then(res => ((res.body?.nodeStatuses || res.body?.node_statuses || []) as any[]).map(normalizeNodeStatus)) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listContractCodeBlocklistEntries(): Promise<TokenContractCodeBlocklistEntry[]> & {abort?: () => void} {
        const req = requests.get('/tokens/policies/contract-code-blocklist-entries', readScope);
        const promise = req.then(res => ((res.body?.entries || []) as any[]).map(normalizeContractCodeBlocklistEntry)) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public createContractCodeBlocklistEntry(values: {note?: string; sourceChainID?: number; sourceContract?: string}): Promise<void> & {abort?: () => void} {
        const req = requests.post('/tokens/policies/contract-code-blocklist-entries', writeScope).send({
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
        const req = requests.patch(`/tokens/policies/contract-code-blocklist-entries/${encodeURIComponent(codeHash)}`, writeScope).send({codeHash, code_hash: codeHash, note});
        const promise = req.then(res => numberValue(res.body?.updatedCount ?? res.body?.updated_count) || 0) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public deleteContractCodeBlocklistEntry(codeHash: string): Promise<number> & {abort?: () => void} {
        const req = requests.delete(`/tokens/policies/contract-code-blocklist-entries/${encodeURIComponent(codeHash)}`, writeScope);
        const promise = req.then(res => numberValue(res.body?.deletedCount ?? res.body?.deleted_count) || 0) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listWalletBlocklistEntries(): Promise<TokenWalletBlocklistEntry[]> & {abort?: () => void} {
        const req = requests.get('/tokens/policies/wallet-blocklist-entries', readScope);
        const promise = req.then(res => ((res.body?.entries || []) as any[]).map(normalizeWalletBlocklistEntry)) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public createWalletBlocklistEntry(wallet: string, note: string): Promise<void> & {abort?: () => void} {
        const req = requests.post('/tokens/policies/wallet-blocklist-entries', writeScope).send({wallet, note});
        const promise = req.then(() => undefined) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public updateWalletBlocklistEntry(wallet: string, note: string): Promise<number> & {abort?: () => void} {
        const req = requests.patch(`/tokens/policies/wallet-blocklist-entries/${encodeURIComponent(wallet)}`, writeScope).send({wallet, note});
        const promise = req.then(res => numberValue(res.body?.updatedCount ?? res.body?.updated_count) || 0) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public deleteWalletBlocklistEntry(wallet: string): Promise<number> & {abort?: () => void} {
        const req = requests.delete(`/tokens/policies/wallet-blocklist-entries/${encodeURIComponent(wallet)}`, writeScope);
        const promise = req.then(res => numberValue(res.body?.deletedCount ?? res.body?.deleted_count) || 0) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listChainCheckpoints(): Promise<TokenChainCheckpoint[]> & {abort?: () => void} {
        const req = requests.get('/tokens/chain-checkpoints', readScope);
        const promise = req.then(res => ((res.body?.checkpoints || []) as any[]).map(normalizeCheckpoint)) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public updateChainCheckpoint(chainID: number, status: string): Promise<TokenChainCheckpoint | undefined> & {abort?: () => void} {
        const req = requests.patch(`/tokens/chain-checkpoints/${encodeURIComponent(String(chainID))}`, writeScope).send({status});
        const promise = req.then(res => {
            const item = res.body?.checkpoint;
            return item ? normalizeCheckpoint(item) : undefined;
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public getChainProcessingSummary(options: {chainID: number; windowSeconds: number; blockNumber?: number}): Promise<TokenChainProcessingSummary> & {abort?: () => void} {
        const req = requests.get('/tokens/chain-processing/summary', readScope).query({
            chain_id: options.chainID,
            window_seconds: options.windowSeconds,
            block_number: options.blockNumber || undefined
        });
        const promise = req.then(res => normalizeChainProcessingSummary(res.body?.summary || {})) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listChainProcessingAttempts(options: {
        chainID: number;
        windowSeconds: number;
        blockNumber?: number;
        status?: string;
        page?: number;
        pageSize?: number;
    }): Promise<PagedResponse<TokenChainProcessingAttempt>> & {abort?: () => void} {
        const req = requests.get('/tokens/chain-processing/attempts', readScope).query({
            chain_id: options.chainID,
            window_seconds: options.windowSeconds,
            block_number: options.blockNumber || undefined,
            status: options.status || undefined,
            page: options.page,
            page_size: options.pageSize
        });
        const promise = req.then(res => ({
            items: ((res.body?.attempts || []) as any[]).map(normalizeChainProcessingAttempt),
            total: numberValue(res.body?.total) || 0,
            page: numberValue(res.body?.page) || 1,
            pageSize: numberValue(res.body?.pageSize ?? res.body?.page_size) || 20
        })) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listContractCodes(
        options: {page?: number; pageSize?: number; codeHash?: string; orderBy?: string} = {}
    ): Promise<PagedResponse<TokenContractCode>> & {abort?: () => void} {
        const req = requests.get('/tokens/contract-codes', readScope).query({
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
        const req = requests.get(`/tokens/contract-codes/${encodeURIComponent(codeHash)}`, readScope);
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
            codeHash?: string;
            contract?: string;
            collectionStatus?: string;
            profileState?: string;
            pairBalanceSupplyStates?: string[];
            pairMinimumLPStates?: string[];
            pairFeeLPShareStates?: string[];
            pairQuoteUSDTMin?: string;
            pairQuoteUSDTMax?: string;
        } = {}
    ): Promise<PagedResponse<TokenProjectListItem>> & {abort?: () => void} {
        const req = requests.get('/tokens/projects', readScope).query({
            code_hash: options.codeHash || undefined,
            contract: options.contract || undefined,
            collection_status: options.collectionStatus || undefined,
            profile_state: options.profileState || undefined,
            pair_balance_supply_states: options.pairBalanceSupplyStates?.length ? options.pairBalanceSupplyStates : undefined,
            pair_minimum_lp_states: options.pairMinimumLPStates?.length ? options.pairMinimumLPStates : undefined,
            pair_fee_lp_share_states: options.pairFeeLPShareStates?.length ? options.pairFeeLPShareStates : undefined,
            pair_quote_usdt_min: options.pairQuoteUSDTMin || undefined,
            pair_quote_usdt_max: options.pairQuoteUSDTMax || undefined,
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
        const req = requests.get(`/tokens/projects/${encodeURIComponent(String(projectID))}`, readScope);
        const promise = req.then(res => {
            const body = res.body || {};
            return body.found === false || !body.detail ? undefined : normalizeProjectDetail(body.detail);
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public getProjectProfile(projectID: number): Promise<TokenProjectProfile | undefined> & {abort?: () => void} {
        const req = requests.get(`/tokens/projects/${encodeURIComponent(String(projectID))}/profile`, readScope);
        const promise = req.then(res => {
            const body = res.body || {};
            return body.found === false || !body.profile ? undefined : normalizeProjectProfile(body.profile);
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public getProjectSwapActivity(projectID: number): Promise<TokenProjectSwapActivity | undefined> & {abort?: () => void} {
        const req = requests.get(`/tokens/projects/${encodeURIComponent(String(projectID))}/swap-activity`, readScope);
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
            .get(
                `/tokens/projects/${encodeURIComponent(String(projectID))}/swap-pairs/${encodeURIComponent(pairKind)}/blocks/${encodeURIComponent(String(blockNumber))}/events`,
                readScope
            )
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

    public listProjectWalletNormalTransactions(
        projectID: number,
        options: {wallet?: string; receiptStatus?: string; methodID?: string; page?: number; pageSize?: number} = {}
    ): Promise<PagedResponse<TokenWalletNormalTransaction>> & {abort?: () => void} {
        const req = requests.get(`/tokens/projects/${encodeURIComponent(String(projectID))}/wallet-normal-transactions`, readScope).query({
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
        const req = requests.get('/tokens/collection-tasks', readScope).query({
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

    public getCollectionTask(taskID: number): Promise<TokenCollectionTask | undefined> & {abort?: () => void} {
        const req = requests.get(`/tokens/collection-tasks/${encodeURIComponent(String(taskID))}`, readScope);
        const promise = req.then(res => {
            const body = res.body || {};
            return body.found === false || !body.task ? undefined : normalizeTask(body.task);
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }
}
