import requests from './requests';

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
    isOpenSource?: boolean;
    isBytecodeBlacklisted?: boolean;
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

let cachedBytecodeBlacklistEntries: BytecodeBlacklistEntry[] | undefined;
let bytecodeBlacklistEntriesRequest: (Promise<BytecodeBlacklistEntry[]> & {abort?: () => void}) | null = null;

function normalizeEntry(item: BytecodeBlacklistEntry | any): BytecodeBlacklistEntry {
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
        sourceQualityReportOrigin: item.sourceQualityReportOrigin ?? item.source_quality_report_origin
    };
}

function normalizeDeployment(item: BytecodeDeployment | any): BytecodeDeployment {
    return {
        chainID: item.chainID ?? item.chain_id,
        contract: item.contract,
        firstSeenAt: item.firstSeenAt ?? item.first_seen_at,
        updatedAt: item.updatedAt ?? item.updated_at
    };
}

export class AthenaSolidityService {
    public getContractSourceInfo(contract: string, chainID?: number): Promise<ContractSourceInfo> & {abort?: () => void} {
        const req = requests.get(`/solidity/contracts/${encodeURIComponent(contract)}/source`);
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
                isOpenSource: item.isOpenSource ?? item.is_open_source,
                isBytecodeBlacklisted: item.isBytecodeBlacklisted ?? item.is_bytecode_blacklisted
            } as ContractSourceInfo;
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listBytecodes(options: {page?: number; pageSize?: number; codeHash?: string} = {}): Promise<PagedResponse<BytecodeListItem>> & {abort?: () => void} {
        const req = requests.get('/solidity/bytecodes').query({
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
        const req = requests.get(`/solidity/bytecodes/${encodeURIComponent(codeHash)}`);
        const promise = req.then(res => normalizeBytecodeDetail(res.body || {})) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public listBytecodeDeployments(
        codeHash: string,
        options: {page?: number; pageSize?: number; chainID?: number; contract?: string} = {}
    ): Promise<PagedResponse<BytecodeDeployment>> & {abort?: () => void} {
        const req = requests.get(`/solidity/bytecodes/${encodeURIComponent(codeHash)}/deployments`).query({
            page: options.page,
            page_size: options.pageSize,
            chain_id: options.chainID,
            contract: options.contract || undefined
        });
        const promise = req.then(res => {
            const body = (res.body || {}) as any;
            return {
                items: (body.items || []).map(normalizeDeployment),
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
        const req = requests.get('/solidity/bytecode/blacklist');
        const promise = req
            .then(res => {
                const body = (res.body as ListBytecodeBlacklistEntriesResponse) || {};
                return (body.items || []).map(normalizeEntry);
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
        const req = requests.post('/solidity/bytecode/blacklist').send({
            sourceContract,
            source_contract: sourceContract,
            sourceChainID,
            source_chain_id: sourceChainID,
            note
        });
        const promise = req
            .then(res => normalizeEntry(((res.body as AddBytecodeBlacklistEntryResponse) || {}).item || {}))
            .then(item => {
                cachedBytecodeBlacklistEntries = undefined;
                return item;
            }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public updateBytecodeBlacklistNote(codeHash: string, note: string): Promise<BytecodeBlacklistEntry> & {abort?: () => void} {
        const req = requests.post(`/solidity/bytecode/blacklist/${encodeURIComponent(codeHash)}/note`).send({codeHash, code_hash: codeHash, note});
        const promise = req
            .then(res => normalizeEntry(((res.body as UpdateBytecodeBlacklistNoteResponse) || {}).item || {}))
            .then(item => {
                cachedBytecodeBlacklistEntries = undefined;
                return item;
            }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public deleteBytecodeBlacklist(codeHash: string): Promise<void> & {abort?: () => void} {
        const req = requests.delete(`/solidity/bytecode/blacklist/${encodeURIComponent(codeHash)}`);
        const promise = req.then(() => {
            cachedBytecodeBlacklistEntries = undefined;
        }) as any;
        promise.abort = () => req.abort();
        return promise;
    }
}
