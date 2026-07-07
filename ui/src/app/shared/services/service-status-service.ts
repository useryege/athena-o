import requests from './requests';

export type ServiceHealthStatus = 'SERVING' | 'NOT_SERVING' | 'UNKNOWN' | 'SERVICE_UNKNOWN' | 'UNREACHABLE';

export interface ServiceStatus {
    name: string;
    status: ServiceHealthStatus;
    errorMessage?: string;
}

export interface ListServiceStatusesResult {
    items: ServiceStatus[];
    checkedAt?: number;
}

export interface EtherscanGatewayStatus {
    address: string;
    reachable: boolean;
    started: boolean;
    status: string;
    etherscanBaseURL: string;
    latencyMS?: number;
    checkedAt?: number;
    errorMessage?: string;
}

export interface ListEtherscanGatewayStatusesResult {
    items: EtherscanGatewayStatus[];
    checkedAt?: number;
}

function normalizeEtherscanGatewayStatus(item: any): EtherscanGatewayStatus {
    return {
        address: String(item.address || ''),
        reachable: Boolean(item.reachable),
        started: Boolean(item.started),
        status: String(item.status || ''),
        etherscanBaseURL: String(item.etherscanBaseUrl || item.etherscan_base_url || ''),
        latencyMS: Number(item.latencyMs ?? item.latency_ms ?? 0),
        checkedAt: Number(item.checkedAt || item.checked_at || 0) || undefined,
        errorMessage: String(item.errorMessage || item.error_message || '')
    };
}

export class ServiceStatusService {
    public list(): Promise<ListServiceStatusesResult> & {abort?: () => void} {
        const req = requests.get('/service-statuses');
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: (body.items || []).map((item: any) => ({
                    name: String(item.name || ''),
                    status: String(item.status || 'UNKNOWN') as ServiceHealthStatus,
                    errorMessage: String(item.errorMessage || item.error_message || '')
                })),
                checkedAt: Number(body.checkedAt || body.checked_at || 0) || undefined
            };
        }) as Promise<ListServiceStatusesResult> & {abort?: () => void};
        promise.abort = () => req.abort();
        return promise;
    }

    public listEtherscanGatewayStatuses(): Promise<ListEtherscanGatewayStatusesResult> & {abort?: () => void} {
        const req = requests.get('/etherscan-gateway-statuses');
        const promise = req.then(res => {
            const body = res.body || {};
            return {
                items: ((body.items || []) as any[]).map(normalizeEtherscanGatewayStatus),
                checkedAt: Number(body.checkedAt || body.checked_at || 0) || undefined
            };
        }) as Promise<ListEtherscanGatewayStatusesResult> & {abort?: () => void};
        promise.abort = () => req.abort();
        return promise;
    }
}
