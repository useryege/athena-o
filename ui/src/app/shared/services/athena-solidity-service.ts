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
