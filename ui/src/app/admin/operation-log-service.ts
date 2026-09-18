import requests from '../shared/services/requests';

type AbortablePromise<T> = Promise<T> & {abort?: () => void};
const operationLogReadScope = {feature: 'admin-operation-logs' as const, mode: 'read' as const};

export interface OperationLogSummary {
    operationId: string;
    startedAt: string;
    actorAccountId: string;
    actorUsername: string;
    actorRole: string;
    realm: string;
    credentialKind: string;
    identityVerified: boolean;
    moduleCode: string;
    actionCode: string;
    primaryResourceType: string;
    primaryResourceId: string;
    outcome: string;
    observation: string;
    reasonCode: string;
    durationMs: string;
    businessState: string;
    responseWriteFailed: boolean;
    provider: string;
    targetAccountId: string;
}

export interface OperationLogDetail extends OperationLogSummary {
    finishedAt: string;
    requestId: string;
    parentOperationId: string;
    businessRequestId: string;
    effects: string[];
    resources: any[];
    changes: any[];
    counts: any;
    protocol: any;
    source: any;
}

export interface OperationLogPage {
    items: OperationLogSummary[];
    nextCursor: string;
    snapshotToken: string;
    snapshotAt: string;
    pageSize: number;
}

export interface OperationLogFilters {
    pageSize?: number;
    cursor?: string;
    outcome?: string;
    moduleCode?: string;
    actionCode?: string;
    actorQuery?: string;
}

const value = (item: any, ...keys: string[]) => {
    for (const key of keys) if (item?.[key] !== undefined && item?.[key] !== null) return item[key];
    return undefined;
};
const stringValue = (item: any, ...keys: string[]) => String(value(item, ...keys) || '');
const boolValue = (item: any, ...keys: string[]) => Boolean(value(item, ...keys));
const unwrap = (item: any) => (item && typeof item === 'object' && 'value' in item ? item.value : item);

const normalizeSummary = (item: any = {}): OperationLogSummary => ({
    operationId: stringValue(item, 'operationId', 'operation_id'),
    startedAt: String(unwrap(value(item, 'startedAt', 'started_at')) || ''),
    actorAccountId: String(unwrap(value(item, 'actorAccountId', 'actor_account_id')) || ''),
    actorUsername: String(unwrap(value(item, 'actorUsername', 'actor_username')) || ''),
    actorRole: stringValue(item, 'actorRole', 'actor_role'),
    realm: stringValue(item, 'realm'),
    credentialKind: stringValue(item, 'credentialKind', 'credential_kind'),
    identityVerified: boolValue(unwrap(value(item, 'identityVerified', 'identity_verified'))),
    moduleCode: stringValue(item, 'moduleCode', 'module_code'),
    actionCode: stringValue(item, 'actionCode', 'action_code'),
    primaryResourceType: String(unwrap(value(item, 'primaryResourceType', 'primary_resource_type')) || ''),
    primaryResourceId: String(unwrap(value(item, 'primaryResourceId', 'primary_resource_id')) || ''),
    outcome: stringValue(item, 'outcome'),
    observation: stringValue(item, 'observation'),
    reasonCode: String(unwrap(value(item, 'reasonCode', 'reason_code')) || ''),
    durationMs: String(unwrap(value(item, 'durationMs', 'duration_ms')) || ''),
    businessState: String(unwrap(value(item, 'businessState', 'business_state')) || ''),
    responseWriteFailed: boolValue(unwrap(value(item, 'responseWriteFailed', 'response_write_failed'))),
    provider: String(unwrap(value(item, 'provider')) || ''),
    targetAccountId: String(unwrap(value(item, 'targetAccountId', 'target_account_id')) || '')
});

const normalizeDetail = (item: any = {}): OperationLogDetail => {
    const summary = normalizeSummary(item.summary || item);
    return {
        ...summary,
        finishedAt: String(unwrap(value(item, 'finishedAt', 'finished_at')) || ''),
        requestId: String(unwrap(value(item, 'requestId', 'request_id')) || ''),
        parentOperationId: String(unwrap(value(item, 'parentOperationId', 'parent_operation_id')) || ''),
        businessRequestId: String(unwrap(value(item, 'businessRequestId', 'business_request_id')) || ''),
        effects: (value(item, 'effect', 'effects') || []).map(String),
        resources: value(item, 'resources', 'resourceFacts', 'resource_facts') || [],
        changes: value(item, 'changes', 'changeFacts', 'change_facts') || [],
        counts: value(item, 'countFacts', 'counts', 'count_facts') || {},
        protocol: value(item, 'protocolResult', 'protocol', 'protocol_result') || {},
        source: value(item, 'sourceFacts', 'source', 'source_facts') || {}
    };
};

export class OperationLogService {
    public list(filters: OperationLogFilters = {}): AbortablePromise<OperationLogPage> {
        const query: Record<string, string | number> = {page_size: filters.pageSize || 30};
        if (filters.cursor) query.cursor = filters.cursor;
        if (filters.outcome) query.outcome = filters.outcome;
        if (filters.moduleCode) query.module_code = filters.moduleCode;
        if (filters.actionCode) query.action_code = filters.actionCode;
        if (filters.actorQuery) query.actor_query = filters.actorQuery;
        const request = requests.get('/admin/operation-logs', operationLogReadScope).query(query);
        const promise = request.then((response: any) => {
            const body = response.body || {};
            const page = body.page || {};
            return {
                items: (body.items || []).map(normalizeSummary),
                nextCursor: stringValue(page, 'nextCursor', 'next_cursor'),
                snapshotToken: stringValue(page, 'snapshotToken', 'snapshot_token'),
                snapshotAt: stringValue(page, 'snapshotAt', 'snapshot_at'),
                pageSize: Number(value(page, 'pageSize', 'page_size') || query.page_size)
            };
        }) as AbortablePromise<OperationLogPage>;
        promise.abort = () => request.abort();
        return promise;
    }

    public get(operationId: string, snapshotToken = ''): AbortablePromise<OperationLogDetail> {
        const request = requests.get(`/admin/operation-logs/${encodeURIComponent(operationId)}`, operationLogReadScope);
        if (snapshotToken) request.query({snapshot_token: snapshotToken});
        const promise = request.then((response: any) => normalizeDetail((response.body || {}).item || response.body || {})) as AbortablePromise<OperationLogDetail>;
        promise.abort = () => request.abort();
        return promise;
    }

    public getRuntimeStatus(): AbortablePromise<any> {
        const request = requests.get('/admin/operation-log-runtime', operationLogReadScope);
        const promise = request.then((response: any) => (response.body || {}).status || response.body || {}) as AbortablePromise<any>;
        promise.abort = () => request.abort();
        return promise;
    }

    public getCaptureStatus(): AbortablePromise<any> {
        const request = requests.get('/admin/operation-log-capture-status', operationLogReadScope);
        const promise = request.then((response: any) => (response.body || {}).status || response.body || {}) as AbortablePromise<any>;
        promise.abort = () => request.abort();
        return promise;
    }

    public listActions(): AbortablePromise<any[]> {
        const request = requests.get('/admin/operation-log-actions', operationLogReadScope);
        const promise = request.then((response: any) => (response.body || {}).actions || []) as AbortablePromise<any[]>;
        promise.abort = () => request.abort();
        return promise;
    }
}
