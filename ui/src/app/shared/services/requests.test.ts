import requests from './requests';
import {AccountDataModule} from '../models';

afterEach(() => {
    jest.restoreAllMocks();
});

test('onError emits new request errors without replaying old ones', () => {
    (requests.get('/before-subscribe') as any).emit('error', {status: 401});

    const observed: number[] = [];
    const subscription = requests.onError.subscribe(err => observed.push(err.status));
    expect(observed).toEqual([]);

    (requests.get('/after-subscribe') as any).emit('error', {status: 401});

    expect(observed).toEqual([401]);
    subscription.unsubscribe();
});

// Exercise real request registration and abort handlers without sending HTTP.
test('Trader Sync write revocation aborts only writes; full module revocation aborts reads', () => {
    requests.configureAuthorizationRealm('member');
    requests.beginAuthorizationSession('owner-A');
    const read = requests.get('/trader-sync/activities', {module: AccountDataModule.TraderSync, mode: 'read'});
    const write = requests.post('/trader-sync/subscriptions', {module: AccountDataModule.TraderSync, mode: 'write'});
    const other = requests.get('/wallets', {module: AccountDataModule.Wallet, mode: 'read'});
    const readAbort = jest.spyOn(read, 'abort'),
        writeAbort = jest.spyOn(write, 'abort'),
        otherAbort = jest.spyOn(other, 'abort');
    requests.abortAuthorizationRequests(AccountDataModule.TraderSync, 'write');
    expect(writeAbort).toHaveBeenCalledTimes(1);
    expect(readAbort).not.toHaveBeenCalled();
    expect(otherAbort).not.toHaveBeenCalled();
    requests.abortAuthorizationRequests(AccountDataModule.TraderSync);
    expect(readAbort).toHaveBeenCalledTimes(1);
    expect(writeAbort).toHaveBeenCalledTimes(1);
    expect(otherAbort).not.toHaveBeenCalled();
    requests.endAuthorizationSession();
    expect(otherAbort).toHaveBeenCalledTimes(1);
});
test('owner switch aborts registered Trader Sync requests; completed requests are removed', () => {
    requests.configureAuthorizationRealm('member');
    requests.beginAuthorizationSession('A');
    const pending = requests.get('/trader-sync/activities', {module: AccountDataModule.TraderSync, mode: 'read'});
    const completed = requests.patch('/trader-sync/targets/wallet/note', {module: AccountDataModule.TraderSync, mode: 'write'});
    const pendingAbort = jest.spyOn(pending, 'abort'),
        completedAbort = jest.spyOn(completed, 'abort');
    (completed as any).emit('end');
    requests.beginAuthorizationSession('B');
    expect(pendingAbort).toHaveBeenCalledTimes(1);
    expect(completedAbort).not.toHaveBeenCalled();
    requests.endAuthorizationSession();
});
test('admin Trader Sync feature abort is isolated from other admin reads', () => {
    jest.isolateModules(() => {
        const adminRequests = require('./requests').default;
        adminRequests.configureAuthorizationRealm('admin');
        adminRequests.beginAuthorizationSession('administrator');
        const pending = adminRequests.get('/admin/trader-sync/subscriptions', {feature: 'admin-trader-sync', mode: 'read'});
        const other = adminRequests.get('/admin/notification-runtime/status', {feature: 'admin-notifications', mode: 'read'});
        const abort = jest.spyOn(pending, 'abort'),
            otherAbort = jest.spyOn(other, 'abort');
        adminRequests.abortAuthorizationFeatureRequests('admin-trader-sync', 'read');
        expect(abort).toHaveBeenCalledTimes(1);
        expect(otherAbort).not.toHaveBeenCalled();
        adminRequests.endAuthorizationSession();
        expect(otherAbort).toHaveBeenCalledTimes(1);
    });
});

test('module access errors preserve the ErrorInfo module without treating it as account maintenance', () => {
    const {requestErrorDetails, isAccountMaintenanceError} = require('./requests');
    const error = {
        status: 503,
        response: {
            body: {
                code: 14,
                message: 'Module closed',
                details: [
                    {'@type': 'type.googleapis.com/google.rpc.ErrorInfo', 'domain': 'athena.module_access', 'reason': 'MODULE_ACCESS_CLOSED', 'metadata': {module_key: 'worm'}}
                ]
            }
        }
    };
    expect(requestErrorDetails(error)).toMatchObject({reason: 'MODULE_ACCESS_CLOSED', moduleKey: 'worm'});
    expect(isAccountMaintenanceError(error)).toBe(false);
});

test('module invalidation cancels feature consumers and suppresses their late authorization errors', () => {
    requests.configureAuthorizationRealm('member');
    requests.beginAuthorizationSession('feature-owner');
    const profit = requests.get('/profit-sharing', {feature: 'profit-sharing', mode: 'read'});
    const adminTrader = requests.get('/admin/trader-sync', {feature: 'admin-trader-sync', mode: 'read'});
    const core = requests.get('/account', {feature: 'self-account', mode: 'read'});
    const profitAbort = jest.spyOn(profit, 'abort'),
        adminAbort = jest.spyOn(adminTrader, 'abort'),
        coreAbort = jest.spyOn(core, 'abort');
    const observed = jest.fn();
    const subscription = requests.onError.subscribe(observed);
    requests.abortModuleAccessRequests('profit_sharing');
    requests.abortModuleAccessRequests('trader_sync');
    (profit as any).emit('error', {status: 401});
    (adminTrader as any).emit('error', {status: 403});
    expect(profitAbort).toHaveBeenCalledTimes(1);
    expect(adminAbort).toHaveBeenCalledTimes(1);
    expect(coreAbort).not.toHaveBeenCalled();
    expect(observed).not.toHaveBeenCalled();
    subscription.unsubscribe();
    requests.endAuthorizationSession();
});

test('current module guard stops another request before React unmounts the old business body', () => {
    requests.configureAuthorizationRealm('member');
    requests.beginAuthorizationSession('guard-owner');
    const release = requests.registerModuleAccessGuard('member', key => key !== 'worm' && key !== 'profit_sharing');
    expect(() => requests.post('/worm-trading/start', {module: AccountDataModule.WormTrading, mode: 'write'})).toThrow('Module access');
    expect(() => requests.get('/profit-sharing', {feature: 'profit-sharing', mode: 'read'})).toThrow('Module access');
    expect(() => requests.get('/wallets', {module: AccountDataModule.Wallet, mode: 'read'})).not.toThrow();
    release();
    requests.endAuthorizationSession();
});
