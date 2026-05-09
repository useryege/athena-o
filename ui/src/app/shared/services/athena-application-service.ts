import requests from './requests';

export interface ProjectView {
    meta?: ProjectMeta;
    token?: TokenState;
    wethV2Pool?: PairV2State;
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
    sourceCodeABI?: string;
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
}

export interface ListProjectsResponse {
    items?: ProjectView[];
}

export class AthenaApplicationService {
    public listProjects(): Promise<ProjectView[]> {
        return requests.get('/project/list').then(res => (res.body as ListProjectsResponse).items || []);
    }
}
