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
}
