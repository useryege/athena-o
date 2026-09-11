import * as React from 'react';
import renderer, {act} from 'react-test-renderer';
import {useVisibleQuery, type AbortablePromise, type ReadScope, type VisibleQuery} from './use-visible-query';
import {captureTraderSyncScope, clearTraderSyncState} from '../member/pages/trader-sync/state';

const deferred = () => {
    let resolve!: (value: string) => void;
    let reject!: (error: Error) => void;
    const promise = Object.assign(
        new Promise<string>((yes, no) => {
            resolve = yes;
            reject = no;
        }),
        {abort: jest.fn()}
    );
    // Allow the unimplemented RED scaffold to leave a deferred unused; state assertions still prove delivery.
    void promise.catch(() => undefined);
    return {promise, resolve, reject};
};
let tree: renderer.ReactTestRenderer | undefined;
let state: VisibleQuery<string>;
let visible = true;
let childMounts = 0;
const renders: Array<string | undefined> = [];
function Child() {
    React.useEffect(() => {
        childMounts++;
    }, []);
    return <span>data</span>;
}
function Probe({load, scope}: {load: () => AbortablePromise<string>; scope: ReadScope}) {
    state = useVisibleQuery(load, scope, 5000);
    renders.push(state.data);
    return state.data && !state.loading ? <Child /> : null;
}
const mount = (load: () => AbortablePromise<string>, scope = captureTraderSyncScope('A')) => {
    act(() => {
        tree = renderer.create(<Probe load={load} scope={scope} />);
    });
    return scope;
};
const finish = async (job: ReturnType<typeof deferred>, value = 'data') => {
    await act(async () => job.resolve(value));
};
const fail = async (job: ReturnType<typeof deferred>, message = 'offline') => {
    await act(async () => job.reject(new Error(message)));
};
const tick = async (ms = 5000) => {
    await act(async () => {
        await jest.advanceTimersByTimeAsync(ms);
    });
};
const visibility = (value: boolean) => {
    act(() => {
        visible = value;
        document.dispatchEvent(new Event('visibilitychange'));
    });
};
beforeEach(() => {
    jest.useFakeTimers();
    visible = true;
    childMounts = 0;
    renders.length = 0;
    jest.spyOn(document, 'visibilityState', 'get').mockImplementation(() => (visible ? 'visible' : 'hidden'));
    jest.spyOn(document, 'hidden', 'get').mockImplementation(() => !visible);
});
afterEach(() => {
    act(() => {
        tree?.unmount();
    });
    tree = undefined;
    clearTraderSyncState();
    jest.restoreAllMocks();
    jest.useRealTimers();
});
test('first request remains single flight across multiple five-second periods', async () => {
    const first = deferred(),
        second = deferred();
    const load = jest.fn().mockReturnValueOnce(first.promise).mockReturnValue(second.promise);
    mount(load);
    expect(state.loading).toBe(true);
    await tick(15000);
    expect(load).toHaveBeenCalledTimes(1);
    await finish(first);
    expect(state.data).toBe('data');
    expect(state.loading).toBe(false);
    await tick();
    expect(load).toHaveBeenCalledTimes(2);
    expect(state.loading).toBe(false);
    expect(childMounts).toBe(1);
});
test('hidden pauses polling, visible immediately refreshes and focus deduplicates', async () => {
    const first = deferred(),
        next = deferred();
    const load = jest.fn().mockReturnValueOnce(first.promise).mockReturnValue(next.promise);
    mount(load);
    await finish(first);
    visibility(false);
    await tick(15000);
    expect(load).toHaveBeenCalledTimes(1);
    visibility(true);
    act(() => {
        window.dispatchEvent(new Event('focus'));
    });
    expect(load).toHaveBeenCalledTimes(2);
    await tick();
    expect(load).toHaveBeenCalledTimes(2);
});
test('initially hidden does not load until visible', () => {
    visible = false;
    const load = jest.fn().mockReturnValue(deferred().promise);
    mount(load);
    expect(load).not.toHaveBeenCalled();
    visibility(true);
    expect(load).toHaveBeenCalledTimes(1);
});
test('refresh failure preserves data and mounted subtree, marks stale, and recovers', async () => {
    const a = deferred(),
        b = deferred(),
        c = deferred();
    const load = jest.fn().mockReturnValueOnce(a.promise).mockReturnValueOnce(b.promise).mockReturnValue(c.promise);
    mount(load);
    await finish(a);
    act(() => state.reload());
    await fail(b);
    expect(state).toMatchObject({data: 'data', loading: false, stale: true});
    expect(state.error?.message).toBe('offline');
    expect(childMounts).toBe(1);
    act(() => state.reload());
    await finish(c, 'recovered');
    expect(state).toMatchObject({data: 'recovered', stale: false, loading: false});
    expect(state.error).toBeUndefined();
    expect(childMounts).toBe(1);
});
test.each(['resolve', 'reject'] as const)('clear while still mounted aborts, clears shown data/error, rejects late %s', async completion => {
    const a = deferred(),
        b = deferred(),
        c = deferred();
    const load = jest.fn().mockReturnValueOnce(a.promise).mockReturnValueOnce(b.promise).mockReturnValue(c.promise);
    mount(load);
    await finish(a);
    act(() => state.reload());
    await fail(b);
    act(() => state.reload());
    expect(state.data).toBe('data');
    act(clearTraderSyncState);
    expect(c.promise.abort).toHaveBeenCalledTimes(1);
    expect(state).toMatchObject({loading: false, stale: false});
    expect(state.data).toBeUndefined();
    expect(state.error).toBeUndefined();
    if (completion === 'resolve') await finish(c, 'leak');
    else await fail(c, 'late-error');
    expect(state.data).toBeUndefined();
    expect(state.error).toBeUndefined();
    await tick();
    expect(load).toHaveBeenCalledTimes(3);
});
test.each(['resolve', 'reject'] as const)('new resource key aborts old request; late %s/finally cannot publish or release new slot', async completion => {
    const a = deferred(),
        old = deferred(),
        next = deferred();
    const load = jest.fn().mockReturnValueOnce(a.promise).mockReturnValueOnce(old.promise).mockReturnValue(next.promise);
    const scope = mount(load);
    await finish(a, 'old resource');
    act(() => state.reload());
    const before = renders.length;
    act(() => tree!.update(<Probe load={load} scope={{...scope, key: scope.key + ':resource-2'}} />));
    expect(renders.slice(before)).not.toContain('old resource');
    expect(old.promise.abort).toHaveBeenCalledTimes(1);
    expect(state.data).toBeUndefined();
    expect(state.loading).toBe(true);
    if (completion === 'resolve') await finish(old, 'leak');
    else await fail(old, 'late-error');
    expect(state.error).toBeUndefined();
    await tick(10000);
    expect(load).toHaveBeenCalledTimes(3);
    await finish(next, 'new resource');
    expect(state.data).toBe('new resource');
});
test('same key inline callbacks do not loop and polling uses latest load', async () => {
    const first = deferred(),
        next = deferred();
    const a = jest.fn(() => first.promise),
        b = jest.fn(() => next.promise);
    const scope = mount(() => a());
    await finish(first);
    act(() => tree!.update(<Probe load={() => b()} scope={{...scope}} />));
    expect(b).not.toHaveBeenCalled();
    await tick();
    expect(a).toHaveBeenCalledTimes(1);
    expect(b).toHaveBeenCalledTimes(1);
});
test('already invalid scope is cleared during subscription installation without starting a request', () => {
    const scope = captureTraderSyncScope('A');
    clearTraderSyncState();
    const load = jest.fn(() => deferred().promise);
    mount(load, scope);
    expect(load).not.toHaveBeenCalled();
    expect(state.loading).toBe(false);
    expect(state.data).toBeUndefined();
});
test('invalidation inside load before request assignment still aborts the returned request', async () => {
    const pending = deferred();
    mount(() => {
        clearTraderSyncState();
        return pending.promise;
    });
    expect(pending.promise.abort).toHaveBeenCalledTimes(1);
    await finish(pending, 'leak');
    expect(state.data).toBeUndefined();
});
test('scope without subscriptions checks predicate on late error/success and cannot reload', async () => {
    let current = true;
    const pending = deferred();
    const load = jest.fn(() => pending.promise);
    mount(load, {key: 'admin/resource', isCurrent: () => current});
    current = false;
    await fail(pending);
    expect(state.error).toBeUndefined();
    act(() => state.reload());
    expect(load).toHaveBeenCalledTimes(1);
});
test('same-owner session and issuer/regrant boundaries never revive prior request', async () => {
    const a = deferred(),
        b = deferred();
    const load = jest.fn().mockReturnValueOnce(a.promise).mockReturnValue(b.promise);
    mount(load);
    act(clearTraderSyncState);
    act(() => tree!.update(<Probe load={load} scope={captureTraderSyncScope('A')} />));
    await finish(a, 'old-session');
    expect(state.data).toBeUndefined();
    await tick();
    expect(load).toHaveBeenCalledTimes(2);
    await finish(b, 'new-session');
    expect(state.data).toBe('new-session');
});
test('unmount aborts and removes the same listeners; later events/timers do not request', async () => {
    const addDocument = jest.spyOn(document, 'addEventListener'),
        removeDocument = jest.spyOn(document, 'removeEventListener');
    const addWindow = jest.spyOn(window, 'addEventListener'),
        removeWindow = jest.spyOn(window, 'removeEventListener');
    const pending = deferred();
    const load = jest.fn(() => pending.promise);
    mount(load);
    act(() => tree!.unmount());
    tree = undefined;
    expect(pending.promise.abort).toHaveBeenCalledTimes(1);
    expect(removeDocument).toHaveBeenCalledWith('visibilitychange', addDocument.mock.calls.find(call => call[0] === 'visibilitychange')![1]);
    expect(removeWindow).toHaveBeenCalledWith('focus', addWindow.mock.calls.find(call => call[0] === 'focus')![1]);
    visibility(true);
    act(() => {
        window.dispatchEvent(new Event('focus'));
    });
    await tick(10000);
    await finish(pending, 'late');
    expect(load).toHaveBeenCalledTimes(1);
});
