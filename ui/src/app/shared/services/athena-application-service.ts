import requests from './requests';

export interface ProjectView {
    meta?: ProjectMeta;
    initState?: ProjectInitState;
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

export interface ProjectInitState {
    name?: string;
    symbol?: string;
    decimals?: number;
    totalSupply?: string;
}

export interface ListProjectsResponse {
    items?: ProjectView[];
}

export class AthenaApplicationService {
    public listProjects(): Promise<ProjectView[]> {
        return requests.get('/project/list').then(res => (res.body as ListProjectsResponse).items || []);
    }
}
