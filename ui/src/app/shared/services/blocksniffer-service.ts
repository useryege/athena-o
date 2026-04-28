import requests from './requests';

interface EvmNodeWsURLPayload {
    evmNodeWsURL: string;
}

export class BlocksnifferService {
    public getEvmNodeWsURL(): Promise<string> {
        return requests.get('/blocksniffer/evm-node-ws-url').then(res => (res.body as EvmNodeWsURLPayload).evmNodeWsURL || '');
    }

    public setEvmNodeWsURL(evmNodeWsURL: string): Promise<string> {
        return requests
            .put('/blocksniffer/evm-node-ws-url')
            .send({evmNodeWsURL})
            .then(res => (res.body as EvmNodeWsURLPayload).evmNodeWsURL || '');
    }
}
