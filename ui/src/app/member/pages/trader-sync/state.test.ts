import {captureTraderSyncScope, clearTraderSyncState, readAddDraft, saveAddDraft, saveNewSubscriptionFocus, readNewSubscriptionFocus} from './state';
afterEach(clearTraderSyncState);
test('captures owner and generation; clearing invalidates same-owner relogin and different owners permanently', () => {
    const a = captureTraderSyncScope('owner-A'),
        b = captureTraderSyncScope('owner-B');
    expect(a.key).not.toBe(b.key);
    expect(a.key).toBe(captureTraderSyncScope('owner-A').key);
    clearTraderSyncState();
    expect(a.isCurrent()).toBe(false);
    expect(b.isCurrent()).toBe(false);
    const again = captureTraderSyncScope('owner-A');
    expect(again.key).not.toBe(a.key);
    expect(again.isCurrent()).toBe(true);
    clearTraderSyncState();
    expect(again.isCurrent()).toBe(false);
});
test('clear notifies mounted subscribers synchronously once and unsubscribed listeners never fire', () => {
    const scope = captureTraderSyncScope('A');
    const notified = jest.fn(() => expect(scope.isCurrent()).toBe(false));
    const removed = jest.fn();
    scope.subscribeInvalidation!(notified);
    const unsubscribe = scope.subscribeInvalidation!(removed);
    unsubscribe();
    clearTraderSyncState();
    expect(notified).toHaveBeenCalledTimes(1);
    expect(removed).not.toHaveBeenCalled();
    clearTraderSyncState();
    expect(notified).toHaveBeenCalledTimes(1);
});
test('subscription after invalidation is notified immediately and cannot become current again', () => {
    const old = captureTraderSyncScope('A');
    clearTraderSyncState();
    const notified = jest.fn();
    const unsubscribe = old.subscribeInvalidation!(notified);
    expect(notified).toHaveBeenCalledTimes(1);
    unsubscribe();
    expect(old.isCurrent()).toBe(false);
    expect(captureTraderSyncScope('A').isCurrent()).toBe(true);
});

test('draft clearing fences a pending read and clears new subscription focus', () => {
    const scope = captureTraderSyncScope('owner-a');
    saveAddDraft({ownerId: 'owner-a', input: '0xabc', note: 'desk', noteEdited: true, returnPath: '/trader-sync', scrollY: 120});
    saveNewSubscriptionFocus('owner-a', 'subscription-id');
    expect(readAddDraft('owner-a')?.note).toBe('desk');
    expect(readNewSubscriptionFocus('owner-a')).toBe('subscription-id');
    expect(readAddDraft('owner-b')).toBeUndefined();
    expect(readNewSubscriptionFocus('owner-b')).toBeUndefined();
    clearTraderSyncState();
    expect(readAddDraft('owner-a')).toBeUndefined();
    expect(readNewSubscriptionFocus('owner-a')).toBeUndefined();
    expect(scope.isCurrent()).toBe(false);
});
test('only one owner draft is retained', () => {
    saveAddDraft({ownerId: 'A', input: 'one', note: '', noteEdited: false, returnPath: '/trader-sync', scrollY: 0});
    saveAddDraft({ownerId: 'B', input: 'two', note: '', noteEdited: false, returnPath: '/trader-sync', scrollY: 0});
    expect(readAddDraft('A')).toBeUndefined();
    expect(readAddDraft('B')?.input).toBe('two');
});
