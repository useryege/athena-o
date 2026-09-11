import type {SuperAgentRequest} from 'superagent';
import {AccountDataModule} from '../shared/models';
import requests from '../shared/services/requests';
import type {AbortablePromise} from '../shared/use-visible-query';
import * as models from './trader-sync-models';

export interface PageInput {
    pageSize?: number;
    cursor?: string;
}
export interface SubscriptionQuery extends PageInput {
    view?: 'current' | 'cancelled';
    state?: string;
}
export interface ActivityQuery extends PageInput {
    subscriptionId?: string;
    from?: string;
    to?: string;
    summaryBatchId?: string;
    refreshCursor?: string;
}
export interface PartQuery extends PageInput {
    activityId?: string;
}
export interface CreateRequest {
    confirmationToken: string;
    requestId: string;
    note?: {value: string};
}
export interface ChangeRequest {
    expectedRevision: string;
    requestId: string;
}
export interface NoteRequest extends ChangeRequest {
    note: string;
}

const readScope = {module: AccountDataModule.TraderSync, mode: 'read'} as const;
const writeScope = {module: AccountDataModule.TraderSync, mode: 'write'} as const;
const base = '/trader-sync';
const compact = (values: Record<string, string | number | undefined>) => Object.fromEntries(Object.entries(values).filter(([, value]) => value !== undefined));
const pageQuery = (input: PageInput) => ({'page.page_size': input.pageSize, 'page.cursor': input.cursor});
const abortable = <T>(request: SuperAgentRequest, normalize: (body: any) => T): AbortablePromise<T> => {
    const result = request.then(response => normalize(response.body)) as AbortablePromise<T>;
    result.abort = () => request.abort();
    return result;
};

export class MemberTraderSyncService {
    public resolveTarget(input: string): AbortablePromise<models.ResolvedTarget> {
        return abortable(requests.post(`${base}/targets:resolve`, readScope).send({input}), body => models.normalizeResolvedTarget(body?.target));
    }
    public createSubscription(input: CreateRequest): AbortablePromise<models.Subscription> {
        return abortable(requests.post(`${base}/subscriptions`, writeScope).send(input), body => models.normalizeSubscription(body?.subscription));
    }
    public listSubscriptions(input: SubscriptionQuery = {}): AbortablePromise<models.SubscriptionPage> {
        return abortable(
            requests.get(`${base}/subscriptions`, readScope).query(compact({...pageQuery(input), view: input.view, state: input.state})),
            models.normalizeSubscriptionPage
        );
    }
    public getSubscription(id: string): AbortablePromise<models.Subscription> {
        return abortable(requests.get(`${base}/subscriptions/${encodeURIComponent(id)}`, readScope), body => models.normalizeSubscription(body?.subscription));
    }
    public listSubscriptionHistory(id: string, input: PageInput = {}): AbortablePromise<models.HistoryPage> {
        return abortable(requests.get(`${base}/subscriptions/${encodeURIComponent(id)}/history`, readScope).query(compact(pageQuery(input))), models.normalizeHistoryPage);
    }
    public pauseSubscription(id: string, input: ChangeRequest): AbortablePromise<models.Subscription> {
        return this.changeSubscription(id, 'pause', input);
    }
    public resumeSubscription(id: string, input: ChangeRequest): AbortablePromise<models.Subscription> {
        return this.changeSubscription(id, 'resume', input);
    }
    public cancelSubscription(id: string, input: ChangeRequest): AbortablePromise<models.Subscription> {
        return this.changeSubscription(id, 'cancel', input);
    }
    private changeSubscription(id: string, action: 'pause' | 'resume' | 'cancel', input: ChangeRequest): AbortablePromise<models.Subscription> {
        return abortable(requests.post(`${base}/subscriptions/${encodeURIComponent(id)}:${action}`, writeScope).send(input), body =>
            models.normalizeSubscription(body?.subscription)
        );
    }
    public updateTargetNote(wallet: string, input: NoteRequest): AbortablePromise<models.TargetNote> {
        return abortable(requests.patch(`${base}/targets/${encodeURIComponent(wallet)}/note`, writeScope).send(input), body => models.normalizeTargetNote(body?.note));
    }
    public listActivities(input: ActivityQuery = {}): AbortablePromise<models.ActivityPage> {
        if (input.cursor !== undefined && input.refreshCursor !== undefined) throw new Error('Page cursor and refresh cursor are mutually exclusive');
        const query = compact({
            ...pageQuery(input),
            subscription_id: input.subscriptionId,
            from: input.from,
            to: input.to,
            summary_batch_id: input.summaryBatchId,
            refresh_cursor: input.refreshCursor
        });
        return abortable(requests.get(`${base}/activities`, readScope).query(query), models.normalizeActivityPage);
    }
    public getActivity(id: string): AbortablePromise<models.Activity> {
        return abortable(requests.get(`${base}/activities/${encodeURIComponent(id)}`, readScope), body => models.normalizeActivity(body?.activity));
    }
    public getSummaryBatch(id: string): AbortablePromise<models.SummaryBatch> {
        return abortable(requests.get(`${base}/summaries/${encodeURIComponent(id)}`, readScope), body => models.normalizeSummaryBatch(body?.batch));
    }
    public listSummaryParts(id: string, input: PartQuery = {}): AbortablePromise<models.PartPage> {
        return abortable(
            requests.get(`${base}/summaries/${encodeURIComponent(id)}/parts`, readScope).query(compact({...pageQuery(input), activity_id: input.activityId})),
            models.normalizePartPage
        );
    }
}
