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
    blockNumber?: number;
    blockTime?: number;
    codeHash?: string;
    createdAt?: string;
}

export interface TokenProjectReport {
    projectID?: number;
    chainID?: number;
    name?: string;
    symbol?: string;
    contract?: string;
    evaluationStatus?: string;
    evaluationAttempts?: number;
    evaluationLastError?: string;
    evaluationUpdatedAt?: string;
    reportDataAvailable?: boolean;
    wethPairIsCreated?: boolean;
    wethPairIsRemoveLiquidity?: boolean;
    wethPairIsMint?: boolean;
    wethPairQuoteUsdtValueInt?: string;
    wethPairLastSwapAt?: string;
    usdtPairIsCreated?: boolean;
    usdtPairIsRemoveLiquidity?: boolean;
    usdtPairIsMint?: boolean;
    usdtPairQuoteUsdtValueInt?: string;
    usdtPairLastSwapAt?: string;
    sourceUpdatedAt?: string;
    evaluatedAt?: string;
    createdAt?: string;
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
        blockNumber: numberValue(item.blockNumber ?? item.block_number),
        blockTime: numberValue(item.blockTime ?? item.block_time),
        codeHash: item.codeHash ?? item.code_hash,
        createdAt: item.createdAt ?? item.created_at
    };
}

function normalizeProjectReport(item: any): TokenProjectReport {
    const reportDataAvailable = item.reportDataAvailable ?? item.report_data_available ?? false;
    const reportBoolean = (camel: string, snake: string) => {
        if (!reportDataAvailable) {
            return undefined;
        }
        return item[camel] ?? item[snake] ?? false;
    };
    return {
        projectID: numberValue(item.projectID ?? item.projectId ?? item.project_id),
        chainID: numberValue(item.chainID ?? item.chainId ?? item.chain_id),
        name: item.name,
        symbol: item.symbol,
        contract: item.contract,
        evaluationStatus: item.evaluationStatus ?? item.evaluation_status,
        evaluationAttempts: numberValue(item.evaluationAttempts ?? item.evaluation_attempts),
        evaluationLastError: item.evaluationLastError ?? item.evaluation_last_error,
        evaluationUpdatedAt: item.evaluationUpdatedAt ?? item.evaluation_updated_at,
        reportDataAvailable,
        wethPairIsCreated: reportBoolean('wethPairIsCreated', 'weth_pair_is_created'),
        wethPairIsRemoveLiquidity: reportBoolean('wethPairIsRemoveLiquidity', 'weth_pair_is_remove_liquidity'),
        wethPairIsMint: reportBoolean('wethPairIsMint', 'weth_pair_is_mint'),
        wethPairQuoteUsdtValueInt: item.wethPairQuoteUsdtValueInt ?? item.weth_pair_quote_usdt_value_int,
        wethPairLastSwapAt: item.wethPairLastSwapAt ?? item.weth_pair_last_swap_at,
        usdtPairIsCreated: reportBoolean('usdtPairIsCreated', 'usdt_pair_is_created'),
        usdtPairIsRemoveLiquidity: reportBoolean('usdtPairIsRemoveLiquidity', 'usdt_pair_is_remove_liquidity'),
        usdtPairIsMint: reportBoolean('usdtPairIsMint', 'usdt_pair_is_mint'),
        usdtPairQuoteUsdtValueInt: item.usdtPairQuoteUsdtValueInt ?? item.usdt_pair_quote_usdt_value_int,
        usdtPairLastSwapAt: item.usdtPairLastSwapAt ?? item.usdt_pair_last_swap_at,
        sourceUpdatedAt: item.sourceUpdatedAt ?? item.source_updated_at,
        evaluatedAt: item.evaluatedAt ?? item.evaluated_at,
        createdAt: item.createdAt ?? item.created_at
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
        options: {page?: number; pageSize?: number; chainID?: number; codeHash?: string; contract?: string} = {}
    ): Promise<PagedResponse<TokenProject>> & {abort?: () => void} {
        const req = requests.get('/tokens/projects').query({
            chain_id: options.chainID,
            code_hash: options.codeHash || undefined,
            contract: options.contract || undefined,
            page: options.page,
            page_size: options.pageSize
        });
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: ((body.projects || []) as any[]).map(normalizeProject),
                total: numberValue(body.total) || 0,
                page: numberValue(body.page) || options.page || 1,
                pageSize: numberValue(body.pageSize ?? body.page_size) || options.pageSize || 20
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listProjectReports(
        options: {page?: number; pageSize?: number; chainID?: number; projectID?: number; contract?: string; evaluationStatus?: string} = {}
    ): Promise<PagedResponse<TokenProjectReport>> & {abort?: () => void} {
        const req = requests.get('/tokens/project-reports').query({
            chain_id: options.chainID,
            project_id: options.projectID,
            contract: options.contract || undefined,
            evaluation_status: options.evaluationStatus || undefined,
            page: options.page,
            page_size: options.pageSize
        });
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: ((body.projectReports || body.project_reports || []) as any[]).map(normalizeProjectReport),
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
}
