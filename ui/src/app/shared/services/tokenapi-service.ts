import requests from './requests';

export interface TokenAPIBytecodeBlacklist {
    codeHash?: string;
    note?: string;
    sourceChainID?: number;
    sourceContract?: string;
    createdAt?: string;
}

export interface TokenAPIWalletBlacklist {
    wallet?: string;
    note?: string;
    createdAt?: string;
}

export interface TokenAPIChainIngestCheckpoint {
    chainID?: number;
    chainName?: string;
    enabled?: boolean;
    cursorBlockNumber?: number;
    status?: string;
    createdAt?: string;
}

export interface TokenAPIChainOption {
    chainID?: number;
    chainName?: string;
}

export interface TokenAPIOptions {
    chains: TokenAPIChainOption[];
}

export interface TokenAPINodeStatus {
    chainID?: number;
    chainName?: string;
    endpoint?: string;
    available?: boolean;
    latencyMS?: number;
    reportedChainID?: number;
    latestBlockNumber?: number;
    checkedAt?: string;
    error?: string;
}

export interface TokenAPIContractCode {
    codeHash?: string;
    sourceCode?: string;
    sourceCodeFetchedAt?: string;
    createdAt?: string;
    deploymentCount?: number;
}

export interface TokenAPIProject {
    projectID?: number;
    chainID?: number;
    name?: string;
    symbol?: string;
    contract?: string;
    creator?: string;
    txHash?: string;
    txIndex?: number;
    blockNumber?: number;
    blockTime?: number;
    codeHash?: string;
    createdAt?: string;
}

export interface TokenAPIProjectDataCollectionTask {
    projectID?: number;
    dataType?: string;
    status?: string;
    attempts?: number;
    nextAttemptAt?: string;
    lastError?: string;
    createdAt?: string;
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

function normalizeBytecodeBlacklist(item: any): TokenAPIBytecodeBlacklist {
    return {
        codeHash: item.codeHash ?? item.code_hash,
        note: item.note,
        sourceChainID: numberValue(item.sourceChainID ?? item.sourceChainId ?? item.source_chain_id),
        sourceContract: item.sourceContract ?? item.source_contract,
        createdAt: item.createdAt ?? item.created_at
    };
}

function normalizeWalletBlacklist(item: any): TokenAPIWalletBlacklist {
    return {
        wallet: item.wallet,
        note: item.note,
        createdAt: item.createdAt ?? item.created_at
    };
}

function normalizeCheckpoint(item: any): TokenAPIChainIngestCheckpoint {
    return {
        chainID: numberValue(item.chainID ?? item.chainId ?? item.chain_id),
        chainName: item.chainName ?? item.chain_name,
        enabled: item.enabled,
        cursorBlockNumber: numberValue(item.cursorBlockNumber ?? item.cursor_block_number),
        status: item.status,
        createdAt: item.createdAt ?? item.created_at
    };
}

function normalizeChainOption(item: any): TokenAPIChainOption {
    return {
        chainID: numberValue(item.chainID ?? item.chainId ?? item.chain_id),
        chainName: item.chainName ?? item.chain_name
    };
}

function normalizeNodeStatus(item: any): TokenAPINodeStatus {
    return {
        chainID: numberValue(item.chainID ?? item.chainId ?? item.chain_id),
        chainName: item.chainName ?? item.chain_name,
        endpoint: item.endpoint,
        available: item.available,
        latencyMS: numberValue(item.latencyMS ?? item.latencyMs ?? item.latency_ms),
        reportedChainID: numberValue(item.reportedChainID ?? item.reportedChainId ?? item.reported_chain_id),
        latestBlockNumber: numberValue(item.latestBlockNumber ?? item.latest_block_number),
        checkedAt: item.checkedAt ?? item.checked_at,
        error: item.error
    };
}

function normalizeContractCode(item: any): TokenAPIContractCode {
    return {
        codeHash: item.codeHash ?? item.code_hash,
        sourceCode: item.sourceCode ?? item.source_code,
        sourceCodeFetchedAt: item.sourceCodeFetchedAt ?? item.source_code_fetched_at,
        createdAt: item.createdAt ?? item.created_at,
        deploymentCount: numberValue(item.deploymentCount ?? item.deployment_count)
    };
}

function normalizeProject(item: any): TokenAPIProject {
    return {
        projectID: numberValue(item.projectID ?? item.projectId ?? item.project_id),
        chainID: numberValue(item.chainID ?? item.chainId ?? item.chain_id),
        name: item.name,
        symbol: item.symbol,
        contract: item.contract,
        creator: item.creator,
        txHash: item.txHash ?? item.tx_hash,
        txIndex: numberValue(item.txIndex ?? item.tx_index),
        blockNumber: numberValue(item.blockNumber ?? item.block_number),
        blockTime: numberValue(item.blockTime ?? item.block_time),
        codeHash: item.codeHash ?? item.code_hash,
        createdAt: item.createdAt ?? item.created_at
    };
}

function normalizeTask(item: any): TokenAPIProjectDataCollectionTask {
    return {
        projectID: numberValue(item.projectID ?? item.projectId ?? item.project_id),
        dataType: item.dataType ?? item.data_type,
        status: item.status,
        attempts: numberValue(item.attempts),
        nextAttemptAt: item.nextAttemptAt ?? item.next_attempt_at,
        lastError: item.lastError ?? item.last_error,
        createdAt: item.createdAt ?? item.created_at
    };
}

export class TokenAPIService {
    public getOptions(): Promise<TokenAPIOptions> & {abort?: () => void} {
        const req = requests.get('/tokenapi/options');
        const promise = req.then(res => {
            const options = res.body?.options || {};
            return {
                chains: ((options.chains || []) as any[]).map(normalizeChainOption)
            };
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listNodeStatuses(): Promise<TokenAPINodeStatus[]> & {abort?: () => void} {
        const req = requests.get('/tokenapi/node-statuses');
        const promise = req.then(res => ((res.body?.nodeStatuses || res.body?.node_statuses || []) as any[]).map(normalizeNodeStatus)) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listBytecodeBlacklists(): Promise<TokenAPIBytecodeBlacklist[]> & {abort?: () => void} {
        const req = requests.get('/tokenapi/bytecode-blacklists');
        const promise = req.then(res => ((res.body?.bytecodeBlacklists || res.body?.bytecode_blacklists || []) as any[]).map(normalizeBytecodeBlacklist)) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public createBytecodeBlacklist(values: {note?: string; sourceChainID?: number; sourceContract?: string}): Promise<void> & {abort?: () => void} {
        const req = requests.post('/tokenapi/bytecode-blacklists').send({
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

    public updateBytecodeBlacklist(codeHash: string, note: string): Promise<number> & {abort?: () => void} {
        const req = requests.post(`/tokenapi/bytecode-blacklists/${encodeURIComponent(codeHash)}`).send({codeHash, code_hash: codeHash, note});
        const promise = req.then(res => numberValue(res.body?.updatedCount ?? res.body?.updated_count) || 0) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public deleteBytecodeBlacklist(codeHash: string): Promise<number> & {abort?: () => void} {
        const req = requests.delete(`/tokenapi/bytecode-blacklists/${encodeURIComponent(codeHash)}`);
        const promise = req.then(res => numberValue(res.body?.deletedCount ?? res.body?.deleted_count) || 0) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listWalletBlacklists(): Promise<TokenAPIWalletBlacklist[]> & {abort?: () => void} {
        const req = requests.get('/tokenapi/wallet-blacklists');
        const promise = req.then(res => ((res.body?.walletBlacklists || res.body?.wallet_blacklists || []) as any[]).map(normalizeWalletBlacklist)) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public createWalletBlacklist(wallet: string, note: string): Promise<void> & {abort?: () => void} {
        const req = requests.post('/tokenapi/wallet-blacklists').send({wallet, note});
        const promise = req.then(() => undefined) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public updateWalletBlacklist(wallet: string, note: string): Promise<number> & {abort?: () => void} {
        const req = requests.post(`/tokenapi/wallet-blacklists/${encodeURIComponent(wallet)}`).send({wallet, note});
        const promise = req.then(res => numberValue(res.body?.updatedCount ?? res.body?.updated_count) || 0) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public deleteWalletBlacklist(wallet: string): Promise<number> & {abort?: () => void} {
        const req = requests.delete(`/tokenapi/wallet-blacklists/${encodeURIComponent(wallet)}`);
        const promise = req.then(res => numberValue(res.body?.deletedCount ?? res.body?.deleted_count) || 0) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listChainIngestCheckpoints(): Promise<TokenAPIChainIngestCheckpoint[]> & {abort?: () => void} {
        const req = requests.get('/tokenapi/chain-ingest-checkpoints');
        const promise = req.then(res => ((res.body?.checkpoints || []) as any[]).map(normalizeCheckpoint)) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public updateChainIngestCheckpoint(chainID: number, status: string): Promise<TokenAPIChainIngestCheckpoint | undefined> & {abort?: () => void} {
        const req = requests.post(`/tokenapi/chain-ingest-checkpoints/${encodeURIComponent(String(chainID))}/status`).send({status});
        const promise = req.then(res => {
            const item = res.body?.checkpoint;
            return item ? normalizeCheckpoint(item) : undefined;
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listContractCodes(
        options: {page?: number; pageSize?: number; codeHash?: string; orderBy?: string} = {}
    ): Promise<PagedResponse<TokenAPIContractCode>> & {abort?: () => void} {
        const req = requests.get('/tokenapi/contract-codes').query({
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

    public getContractCode(codeHash: string): Promise<TokenAPIContractCode | undefined> & {abort?: () => void} {
        const req = requests.get(`/tokenapi/contract-codes/${encodeURIComponent(codeHash)}`);
        const promise = req.then(res => {
            const item = res.body?.contractCode || res.body?.contract_code;
            return item ? normalizeContractCode(item) : undefined;
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listProjects(
        options: {page?: number; pageSize?: number; chainID?: number; codeHash?: string; contract?: string} = {}
    ): Promise<PagedResponse<TokenAPIProject>> & {abort?: () => void} {
        const req = requests.get('/tokenapi/projects').query({
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

    public listProjectDataCollectionTasks(
        options: {page?: number; pageSize?: number; projectID?: number; dataType?: string; status?: string} = {}
    ): Promise<PagedResponse<TokenAPIProjectDataCollectionTask>> & {abort?: () => void} {
        const req = requests.get('/tokenapi/project-data-collection-tasks').query({
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
