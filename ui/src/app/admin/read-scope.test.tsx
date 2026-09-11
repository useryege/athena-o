import * as React from 'react';
import renderer, {act} from 'react-test-renderer';
import {AuthorizationCtx, type AuthorizationState} from '../shared/context';
import {parseUserInfo} from '../shared/models';
import {useVisibleQuery, type VisibleQuery} from '../shared/use-visible-query';
import {beginAdminReadSession, endAdminReadSession, useAdminReadScope} from './read-scope';
const user = (accountId = 'A', iss = 'issuer-1') => parseUserInfo({accountId, iss, loggedIn: true, administrator: true});
let tree: renderer.ReactTestRenderer;
let state: VisibleQuery<string>;
const pending = () => {
    let resolve!: (v: string) => void;
    const promise = Object.assign(new Promise<string>(r => (resolve = r)), {abort: jest.fn()});
    return {promise, resolve};
};
function Probe({load, resource = 'list'}: {load: () => Promise<string>; resource?: string}) {
    state = useVisibleQuery(load, useAdminReadScope(resource), 10000);
    return <span>{state.data}</span>;
}
const view = (load: () => Promise<string>, account = user(), isAdmin = true, resource = 'list') => (
    <AuthorizationCtx.Provider value={{user: account, isAdmin} as AuthorizationState}>
        <Probe load={load} resource={resource} />
    </AuthorizationCtx.Provider>
);
beforeEach(() => {
    jest.useFakeTimers();
    beginAdminReadSession(user());
});
afterEach(() => {
    act(() => {
        tree?.unmount();
        endAdminReadSession();
    });
    jest.useRealTimers();
});
test('known session loss synchronously clears visible cached data and fences late success without navigation', async () => {
    const late = pending();
    const load = jest.fn().mockResolvedValueOnce('old administrator data').mockReturnValue(late.promise);
    await act(async () => {
        tree = renderer.create(view(load));
    });
    expect(state.data).toBe('old administrator data');
    act(() => state.reload());
    act(() => endAdminReadSession());
    expect(state.data).toBeUndefined();
    await act(async () => late.resolve('late private summary'));
    expect(state.data).toBeUndefined();
});
test('same account with a different issuer invalidates the old request and admits only new identity results', async () => {
    const old = pending();
    const load = jest.fn().mockReturnValueOnce(old.promise).mockResolvedValue('new issuer data');
    act(() => {
        tree = renderer.create(view(load));
    });
    act(() => {
        beginAdminReadSession(user('A', 'issuer-2'));
        tree.update(view(load, user('A', 'issuer-2')));
    });
    await act(async () => old.resolve('old issuer data'));
    expect(state.data).toBe('new issuer data');
    expect(old.promise.abort).toHaveBeenCalledTimes(1);
});
test('lost isAdmin clears cached values immediately and never reads using account filters as identity', async () => {
    const load = jest.fn().mockResolvedValue('data');
    await act(async () => {
        tree = renderer.create(view(load));
    });
    act(() => tree.update(view(load, user(), false)));
    expect(state.data).toBeUndefined();
    await act(async () => jest.advanceTimersByTimeAsync(10000));
    expect(load).toHaveBeenCalledTimes(1);
});
test('page unmount aborts its request but keeps the administrator session usable on the next page', async () => {
    const old = pending();
    act(() => {
        tree = renderer.create(view(() => old.promise));
    });
    act(() => tree.unmount());
    expect(old.promise.abort).toHaveBeenCalledTimes(1);
    await act(async () => {
        tree = renderer.create(view(() => Promise.resolve('next page'), user(), true, 'detail'));
        old.resolve('old page');
    });
    expect(state.data).toBe('next page');
});
