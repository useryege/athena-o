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
