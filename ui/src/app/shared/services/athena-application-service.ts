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
}

export interface ProjectChainState {
    token?: TokenState;
    wethPair?: PairV2State;
    usdtPair?: PairV2State;
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

export interface ListArchivedProjectsResponse {
    items?: ProjectView[];
    total?: number;
    page?: number;
    pageSize?: number;
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

    public getProject(contract: string, scope: ProjectScope = PROJECT_SCOPE.ACTIVE): Promise<ProjectView> & {abort?: () => void} {
        const req = requests.get(`/projects/${encodeURIComponent(contract)}`).query({scope});
        const promise = req.then(res => (res.body as GetProjectResponse).item) as any;
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

    public listArchivedProjects(page = 1, pageSize = 20): Promise<{items: ProjectView[]; total: number; page: number; pageSize: number}> & {abort?: () => void} {
        const req = this.listProjects(PROJECT_SCOPE.ARCHIVED, page, pageSize) as any;
        const promise = req.then((res: ListArchivedProjectsResponse) => {
            const body = (res || {}) as ListArchivedProjectsResponse;
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

    public getArchivedProject(contract: string): Promise<ProjectView> & {abort?: () => void} {
        const req = this.getProject(contract, PROJECT_SCOPE.ARCHIVED) as any;
        const promise = req.then((item: ProjectView) => item) as any;
        promise.abort = () => req.abort();
        return promise;
    }
}
