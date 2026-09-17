import * as agent from 'superagent';
import requests from '../shared/services/requests';
import {AdminTraderSyncService} from './trader-sync-service';

// Keep the actual service, scope registration, module guard and abort handlers.
// Only the HTTP send is held so admission and cancellation are observable without a server.
let sent: Array<{request: agent.Request; complete: (error: unknown, response?: unknown) => void}>;
let release: (() => void) | undefined;
const runtime = {collectorConnected: false, collectorEpoch: 'epoch-a', filterRevision: '2', metrics: [], asOf: '2026-09-17T09:00:00Z'};
beforeEach(() => {
    sent = [];
    requests.configureAuthorizationRealm('admin');
    requests.beginAuthorizationSession('administrator');
    jest.spyOn(agent.Request.prototype, 'end').mockImplementation(function (this: agent.Request, callback: any) {
        sent.push({request: this, complete: callback});
        return this;
    });
});
afterEach(() => {
    release?.();
    release = undefined;
    requests.endAuthorizationSession();
    jest.restoreAllMocks();
});
const completeRuntime = () => {
    const transport = sent.find(item => item.request.url.endsWith('/admin/trader-sync/status'))!;
    transport.request.emit('end');
    transport.complete(null, {body: {status: runtime}});
};
test.each(['unknown', 'closed'])('core runtime reads remain available while Trader Sync access is %s', async state => {
    release = requests.registerModuleAccessGuard('admin', key => key !== 'trader_sync' || state === 'open');
    const service = new AdminTraderSyncService();
    const pending = service.getRuntimeStatus();
    expect(() => service.listSubscriptionSummaries()).toThrow('Module access');
    expect(() => service.getSubscriptionSummary('subscription')).toThrow('Module access');
    completeRuntime();
    await expect(pending).resolves.toEqual(runtime);
    expect(sent).toHaveLength(1);
});
test('closing aborts the pending subscription while the pending core runtime still completes', async () => {
    let open = true;
    release = requests.registerModuleAccessGuard('admin', key => key !== 'trader_sync' || open);
    const service = new AdminTraderSyncService();
    const health = service.getRuntimeStatus();
    const subscription = service.listSubscriptionSummaries();
    void health.catch(() => undefined);
    void subscription.catch(() => undefined);
    const runtimeRequest = sent.find(item => item.request.url.endsWith('/status'))!.request;
    const subscriptionRequest = sent.find(item => item.request.url.endsWith('/subscriptions'))!.request;
    const runtimeAbort = jest.spyOn(runtimeRequest, 'abort');
    const subscriptionAbort = jest.spyOn(subscriptionRequest, 'abort');
    open = false;
    requests.abortModuleAccessRequests('trader_sync');
    expect(runtimeAbort).not.toHaveBeenCalled();
    expect(subscriptionAbort).toHaveBeenCalledTimes(1);
    expect(() => service.getSubscriptionSummary('subscription')).toThrow('Module access');
    completeRuntime();
    await expect(health).resolves.toEqual(runtime);
});
