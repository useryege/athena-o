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

export interface EtherscanGatewayProbeCounts {
    success: number;
    rateLimit: number;
    authentication: number;
    plan: number;
    invalidRequest: number;
    malformed: number;
    upstream: number;
    other: number;
}

export interface EtherscanGatewayProbeKeySummary {
    keyLabel: string;
    counts: EtherscanGatewayProbeCounts;
}

export interface EtherscanGatewayProbeGatewaySummary {
    gateway: string;
    counts: EtherscanGatewayProbeCounts;
}

export interface EtherscanGatewayProbeRun {
    runID: string;
    status: string;
    result: string;
    createdAt?: number;
    startedAt?: number;
    finishedAt?: number;
    intervalMS: number;
    requestsPerKey: number;
    keyCount: number;
    gatewayCount: number;
    total: number;
    requiredSuccess: number;
    counts: EtherscanGatewayProbeCounts;
    elapsedMS: number;
    startSpreadMS: number;
    keySummaries: EtherscanGatewayProbeKeySummary[];
    gatewaySummaries: EtherscanGatewayProbeGatewaySummary[];
    samples: string[];
    errorMessage?: string;
}

export interface RunEtherscanGatewayProbeInput {
    intervalMS: number;
    requestsPerKey: number;
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

function normalizeProbeCounts(item: any): EtherscanGatewayProbeCounts {
    item = item || {};
    return {
        success: Number(item.success || 0),
        rateLimit: Number(item.rateLimit ?? item.rate_limit ?? 0),
        authentication: Number(item.authentication || 0),
        plan: Number(item.plan || 0),
        invalidRequest: Number(item.invalidRequest ?? item.invalid_request ?? 0),
        malformed: Number(item.malformed || 0),
        upstream: Number(item.upstream || 0),
        other: Number(item.other || 0)
    };
}

function normalizeProbeRun(item: any): EtherscanGatewayProbeRun {
    item = item || {};
    return {
        runID: String(item.runId || item.run_id || ''),
        status: String(item.status || ''),
        result: String(item.result || ''),
        createdAt: Number(item.createdAt || item.created_at || 0) || undefined,
        startedAt: Number(item.startedAt || item.started_at || 0) || undefined,
        finishedAt: Number(item.finishedAt || item.finished_at || 0) || undefined,
        intervalMS: Number(item.intervalMs ?? item.interval_ms ?? 0),
        requestsPerKey: Number(item.requestsPerKey ?? item.requests_per_key ?? 0),
        keyCount: Number(item.keyCount ?? item.key_count ?? 0),
        gatewayCount: Number(item.gatewayCount ?? item.gateway_count ?? 0),
        total: Number(item.total || 0),
        requiredSuccess: Number(item.requiredSuccess ?? item.required_success ?? 0),
        counts: normalizeProbeCounts(item.counts),
        elapsedMS: Number(item.elapsedMs ?? item.elapsed_ms ?? 0),
        startSpreadMS: Number(item.startSpreadMs ?? item.start_spread_ms ?? 0),
        keySummaries: ((item.keySummaries || item.key_summaries || []) as any[]).map(summary => ({
            keyLabel: String(summary.keyLabel || summary.key_label || ''),
            counts: normalizeProbeCounts(summary.counts)
        })),
        gatewaySummaries: ((item.gatewaySummaries || item.gateway_summaries || []) as any[]).map(summary => ({
            gateway: String(summary.gateway || ''),
            counts: normalizeProbeCounts(summary.counts)
        })),
        samples: ((item.samples || []) as any[]).map(sample => String(sample)),
        errorMessage: String(item.errorMessage || item.error_message || '')
    };
}

export class ServiceStatusService {
    private readonly serviceStatusReadScope = {feature: 'admin-service-status' as const, mode: 'read' as const};
    private readonly etherscanReadScope = {feature: 'admin-etherscan' as const, mode: 'read' as const};
    private readonly etherscanWriteScope = {feature: 'admin-etherscan' as const, mode: 'write' as const};

    public list(): Promise<ListServiceStatusesResult> & {abort?: () => void} {
        const req = requests.get('/service-statuses', this.serviceStatusReadScope);
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
        const req = requests.get('/etherscan-gateway-statuses', this.etherscanReadScope);
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

    public runEtherscanGatewayProbe(input: RunEtherscanGatewayProbeInput): Promise<EtherscanGatewayProbeRun> & {abort?: () => void} {
        const req = requests.post('/etherscan-gateway-probe-runs', this.etherscanWriteScope).send({
            interval_ms: input.intervalMS,
            requests_per_key: input.requestsPerKey
        });
        const promise = req.then(res => normalizeProbeRun(res.body || {})) as Promise<EtherscanGatewayProbeRun> & {abort?: () => void};
        promise.abort = () => req.abort();
        return promise;
    }

    public getEtherscanGatewayProbeRun(runID: string): Promise<EtherscanGatewayProbeRun> & {abort?: () => void} {
        const req = requests.get(`/etherscan-gateway-probe-runs/${encodeURIComponent(runID)}`, this.etherscanReadScope);
        const promise = req.then(res => normalizeProbeRun(res.body || {})) as Promise<EtherscanGatewayProbeRun> & {abort?: () => void};
        promise.abort = () => req.abort();
        return promise;
    }

    public getLatestEtherscanGatewayProbeRun(): Promise<EtherscanGatewayProbeRun> & {abort?: () => void} {
        const req = requests.get('/etherscan-gateway-probe-runs/latest', this.etherscanReadScope);
        const promise = req.then(res => normalizeProbeRun(res.body || {})) as Promise<EtherscanGatewayProbeRun> & {abort?: () => void};
        promise.abort = () => req.abort();
        return promise;
    }
}
