import requests from './requests';

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

export interface WalletBlacklistEntry {
    wallet?: string;
    note?: string;
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

interface ListWalletBlacklistEntriesResponse {
    items?: WalletBlacklistEntry[];
}

interface AddBytecodeBlacklistEntryResponse {
    item?: BytecodeBlacklistEntry;
}

interface AddWalletBlacklistEntryResponse {
    item?: WalletBlacklistEntry;
}

interface UpdateBytecodeBlacklistNoteResponse {
    item?: BytecodeBlacklistEntry;
}

interface UpdateWalletBlacklistEntryNoteResponse {
    item?: WalletBlacklistEntry;
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
let cachedWalletBlacklistEntries: WalletBlacklistEntry[] | undefined;
let walletBlacklistEntriesRequest: (Promise<WalletBlacklistEntry[]> & {abort?: () => void}) | null = null;

function normalizeBytecodeBlacklistEntry(item: BytecodeBlacklistEntry | any): BytecodeBlacklistEntry {
    return {
        codeHash: item.codeHash ?? item.code_hash,
        note: item.note,
        sourceChainID: item.sourceChainID ?? item.source_chain_id,
        sourceContract: item.sourceContract ?? item.source_contract,
        createdAt: item.createdAt ?? item.created_at
    };
}

function normalizeWalletBlacklistEntry(item: WalletBlacklistEntry | any): WalletBlacklistEntry {
    return {
        wallet: item.wallet,
        note: item.note,
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

    public listWalletBlacklistEntries(): Promise<WalletBlacklistEntry[]> & {abort?: () => void} {
        if (cachedWalletBlacklistEntries) {
            const promise = Promise.resolve(cachedWalletBlacklistEntries) as Promise<WalletBlacklistEntry[]> & {abort?: () => void};
            promise.abort = () => {};
            return promise;
        }
        if (walletBlacklistEntriesRequest) {
            const promise = walletBlacklistEntriesRequest as Promise<WalletBlacklistEntry[]> & {abort?: () => void};
            promise.abort = () => {};
            return promise;
        }
        const req = requests.get('/application/wallet/blacklist');
        const promise = req
            .then(res => {
                const body = (res.body as ListWalletBlacklistEntriesResponse) || {};
                return (body.items || []).map(normalizeWalletBlacklistEntry);
            })
            .then(
                items => {
                    cachedWalletBlacklistEntries = items;
                    walletBlacklistEntriesRequest = null;
                    return items;
                },
                err => {
                    walletBlacklistEntriesRequest = null;
                    throw err;
                }
            ) as any;
        promise.abort = () => {};
        walletBlacklistEntriesRequest = promise;
        return promise;
    }

    public addWalletBlacklistEntry(wallet: string, note: string): Promise<WalletBlacklistEntry> & {abort?: () => void} {
        const req = requests.post('/application/wallet/blacklist').send({wallet, note});
        const promise = req
            .then(res => normalizeWalletBlacklistEntry(((res.body as AddWalletBlacklistEntryResponse) || {}).item || {}))
            .then(item => {
                cachedWalletBlacklistEntries = undefined;
                return item;
            }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public updateWalletBlacklistEntryNote(wallet: string, note: string): Promise<WalletBlacklistEntry> & {abort?: () => void} {
        const req = requests.post(`/application/wallet/blacklist/${encodeURIComponent(wallet)}/note`).send({wallet, note});
        const promise = req
            .then(res => normalizeWalletBlacklistEntry(((res.body as UpdateWalletBlacklistEntryNoteResponse) || {}).item || {}))
            .then(item => {
                cachedWalletBlacklistEntries = undefined;
                return item;
            }) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public deleteWalletBlacklistEntry(wallet: string): Promise<void> & {abort?: () => void} {
        const req = requests.delete(`/application/wallet/blacklist/${encodeURIComponent(wallet)}`);
        const promise = req.then(() => {
            cachedWalletBlacklistEntries = undefined;
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
