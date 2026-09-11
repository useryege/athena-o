import requests from '../shared/services/requests';
import type {AbortablePromise} from '../shared/use-visible-query';
import {normalizeRuntimeStatus, normalizeSubscriptionSummary, normalizeSummaryPage, type RuntimeStatus, type SubscriptionSummary, type SummaryPage} from './trader-sync-models';
const scope = {feature: 'admin-trader-sync', mode: 'read'} as const;
export interface ListSubscriptionSummariesInput {
    pageSize?: number;
    cursor?: string;
    accountId?: string;
    state?: string;
    wallet?: string;
    includeCancelled?: boolean;
}
export class AdminTraderSyncService {
    listSubscriptionSummaries(input: ListSubscriptionSummariesInput = {}): AbortablePromise<SummaryPage> {
        const request = requests.get('/admin/trader-sync/subscriptions', scope).query({
            'wallet': input.wallet,
            'include_cancelled': input.includeCancelled,
            'page.page_size': input.pageSize,
            'page.cursor': input.cursor,
            'account_id': input.accountId,
            'state': input.state
        });
        return Object.assign(
            request.then(response => normalizeSummaryPage(response.body)),
            {abort: () => request.abort()}
        );
    }
    getSubscriptionSummary(id: string): AbortablePromise<SubscriptionSummary> {
        const request = requests.get(`/admin/trader-sync/subscriptions/${encodeURIComponent(id)}`, scope);
        return Object.assign(
            request.then(response => normalizeSubscriptionSummary(response.body?.summary)),
            {abort: () => request.abort()}
        );
    }
    getRuntimeStatus(): AbortablePromise<RuntimeStatus> {
        const request = requests.get('/admin/trader-sync/status', scope);
        return Object.assign(
            request.then(response => normalizeRuntimeStatus(response.body?.status)),
            {abort: () => request.abort()}
        );
    }
}
