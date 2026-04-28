import requests from './requests';

interface NodeGrpcURLPayload {
    node_grpc_url: string;
}

export class BlocksnifferService {
    public getNodeGrpcURL(): Promise<string> {
        return requests.get('/blocksniffer/node-grpc-url').then(res => (res.body as NodeGrpcURLPayload).node_grpc_url || '');
    }

    public setNodeGrpcURL(nodeGrpcURL: string): Promise<string> {
        return requests
            .put('/blocksniffer/node-grpc-url')
            .send({node_grpc_url: nodeGrpcURL})
            .then(res => (res.body as NodeGrpcURLPayload).node_grpc_url || '');
    }
}
