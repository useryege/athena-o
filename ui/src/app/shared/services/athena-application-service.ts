import requests from './requests';

export interface ProjectView {
    meta?: ProjectMeta;
    token?: TokenState;
    wethV2Pool?: PairV2State;
    sourceCode?: ProjectSourceCodeState;
    simulate?: ProjectSimulateState;
    analysis?: ProjectAnalysisState;
}

export interface ProjectMeta {
    projectID?: string;
    blockTime?: number;
    blockNumber?: number;
    contract?: string;
    creator?: string;
    txHash?: string;
    txIndex?: number;
}

export interface TokenState {
    name?: string;
    symbol?: string;
    decimals?: number;
    totalSupply?: string;
    sourceCode?: string;
    isValidERC20?: boolean;
}

export interface PairV2State {
    isContractCreated?: boolean;
    contract?: string;
    token0?: string;
    token1?: string;
    totalSupply?: string;
    reserve0?: string;
    reserve1?: string;
    blockTimestampLast?: number;
    tokenReserveBalance?: string;
    wethReserveBalance?: string;
    lockedLiquidity?: string;
}

export interface ListProjectsResponse {
    items?: ProjectView[];
}

export interface ProjectSourceCodeState {
    sourceCode?: string;
}

export interface ProjectSimulateState {
    creatorResult?: SimulateResult;
}

export interface SimulateResult {
    canMintFromDeadViaTransferFrom?: boolean;
    canMintFromZeroViaTransferFrom?: boolean;
    canMintFromPairViaTransferFrom?: boolean;
    canMintViaTransfer?: boolean;
}

export interface ProjectAnalysisState {
    sourceCodeBlacklist?: SourceCodeBlacklistState;
}

export interface SourceCodeBlacklistState {
    hasBlacklistFields?: boolean;
    blacklistFields?: string[];
}

export interface GetProjectResponse {
    item?: ProjectView;
}

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
    public listProjects(): Promise<ProjectView[]> & {abort?: () => void} {
        const req = requests.get('/project/list');
        const promise = req.then(res => (res.body as ListProjectsResponse).items || []) as any;
        promise.abort = () => req.abort();
        return promise;
    }

    public getProject(projectID: string): Promise<ProjectView> & {abort?: () => void} {
        const req = requests.get(`/project/${encodeURIComponent(projectID)}`);
        const promise = req.then(res => (res.body as GetProjectResponse).item) as any;
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
}
