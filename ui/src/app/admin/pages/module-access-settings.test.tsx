import requests from '../../shared/services/requests';
import * as React from 'react';
import renderer, {act} from 'react-test-renderer';
import {Button} from 'antd';
import {ModuleAccessSettings} from './module-access-settings';
import {moduleAccessService} from '../../shared/module-access-service';
import {AuthorizationCtx} from '../../shared/context';
import {parseUserInfo} from '../../shared/models';
import {beginAdminReadSession, endAdminReadSession} from '../read-scope';
const user = parseUserInfo({accountId: 'admin-a', iss: 'issuer', loggedIn: true, administrator: true});
const rows = ['trader_sync', 'solana', 'market_radar', 'managed_oo', 'profit_sharing', 'worm'].map(module_key => ({module_key, state: 2}));
let tree: renderer.ReactTestRenderer;
const deferred = () => {
    let resolve!: (v: any) => void;
    let reject!: (e: Error) => void;
    const promise = Object.assign(
        new Promise<any>((r, j) => {
            resolve = r;
            reject = j;
        }),
        {abort: jest.fn()}
    );
    return {promise, resolve, reject};
};
const mount = async (active = true) => {
    await act(async () => {
        tree = renderer.create(
            <AuthorizationCtx.Provider value={{user, isAdmin: true} as any}>
                <ModuleAccessSettings active={active} />
            </AuthorizationCtx.Provider>
        );
    });
};
beforeEach(() => {
    jest.useFakeTimers();
    beginAdminReadSession(user);
    jest.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible');
    jest.spyOn(moduleAccessService, 'settings').mockResolvedValue(rows as any);
    window.matchMedia = jest.fn().mockReturnValue({matches: false, addEventListener: jest.fn(), removeEventListener: jest.fn()});
});
afterEach(() => {
    act(() => tree?.unmount());
    endAdminReadSession();
    jest.restoreAllMocks();
    jest.useRealTimers();
});
test('inactive tab does not read; active tab renders six explicit actions and polls single flight every five seconds', async () => {
    await mount(false);
    expect(moduleAccessService.settings).not.toHaveBeenCalled();
    await act(async () =>
        tree.update(
            <AuthorizationCtx.Provider value={{user, isAdmin: true} as any}>
                <ModuleAccessSettings active />
            </AuthorizationCtx.Provider>
        )
    );
    expect(JSON.stringify(tree.toJSON())).toContain('Token');
    expect(
        new Set(
            tree.root
                .findAllByType(Button)
                .filter(b => b.props.children === 'Open access')
                .map(b => b.props['aria-label'])
        ).size
    ).toBe(6);
    const pending = deferred();
    jest.mocked(moduleAccessService.settings).mockReturnValue(pending.promise);
    await act(async () => jest.advanceTimersByTimeAsync(15000));
    expect(moduleAccessService.settings).toHaveBeenCalledTimes(2);
});
test('save discards an older list; an unknown write result rereads without replaying the write', async () => {
    await mount();
    const old = deferred();
    jest.mocked(moduleAccessService.settings).mockReturnValueOnce(old.promise);
    await act(async () => jest.advanceTimersByTimeAsync(5000));
    const save = deferred();
    const write = jest.spyOn(moduleAccessService, 'save').mockReturnValue(save.promise);
    const latest = deferred();
    jest.mocked(moduleAccessService.settings).mockReturnValue(latest.promise);
    await act(async () => {
        tree.root
            .findAllByType(Button)
            .find(b => b.props.children === 'Open access')!
            .props.onClick();
    });
    expect(write).toHaveBeenCalledWith('trader_sync', 1);
    expect(old.promise.abort).toHaveBeenCalledTimes(1);
    await act(async () => old.resolve(rows.map(row => ({...row, state: 1}))));
    expect(JSON.stringify(tree.toJSON())).not.toContain('Close access');
    await act(async () => save.reject(new Error('response lost')));
    expect(write).toHaveBeenCalledTimes(1);
    expect(
        tree.root
            .findAllByType(Button)
            .filter(b => b.props.children === 'Open access')
            .every(b => b.props.disabled)
    ).toBe(true);
    await act(async () => latest.resolve(rows.map(row => ({...row, state: row.module_key === 'trader_sync' ? 1 : 2}))));
    expect(JSON.stringify(tree.toJSON())).toContain('Close access');
    expect(write).toHaveBeenCalledTimes(1);
});

const setActive = async (active: boolean) => {
    await act(async () =>
        tree.update(
            <AuthorizationCtx.Provider value={{user, isAdmin: true} as any}>
                <ModuleAccessSettings active={active} />
            </AuthorizationCtx.Provider>
        )
    );
};
const action = (label: string) => tree.root.findAllByType(Button).find(button => button.props['aria-label'] === label)!;

test.each([
    {outcome: 'success', completion: 'before reread'},
    {outcome: 'failure', completion: 'before reread'},
    {outcome: 'success', completion: 'after reread'},
    {outcome: 'failure', completion: 'after reread'}
])('pending save recovers after leaving the tab: old $outcome $completion never replays PUT', async ({outcome, completion}) => {
    await mount();
    const oldWrite = deferred();
    const transport = {then: oldWrite.promise.then.bind(oldWrite.promise), abort: oldWrite.promise.abort};
    const send = jest.fn().mockReturnValue(transport);
    const put = jest.spyOn(requests, 'put').mockReturnValue({send} as any);
    await act(async () => action('Open Trader Sync access').props.onClick());
    expect(action('Open Trader Sync access').props.loading).toBe(true);
    await setActive(false);
    expect(oldWrite.promise.abort).toHaveBeenCalledTimes(1);
    const latest = deferred();
    jest.mocked(moduleAccessService.settings).mockReturnValue(latest.promise);
    await setActive(true);
    expect(action('Open Trader Sync access').props.disabled).toBe(true);
    const completeOld = async () =>
        act(async () => {
            if (outcome === 'success') oldWrite.resolve({body: {setting: {module_key: 'trader_sync', state: 1}}});
            else oldWrite.reject(new Error('late response lost'));
        });
    if (completion === 'before reread') {
        await completeOld();
        expect(action('Open Trader Sync access').props.disabled).toBe(true);
    }
    await act(async () => latest.resolve(rows));
    if (completion === 'after reread') await completeOld();
    expect(action('Open Trader Sync access').props.loading).toBe(false);
    expect(action('Open Trader Sync access').props.disabled).toBe(false);
    expect(tree.root.findAllByType(Button).find(button => button.props.children === 'Refresh access settings')!.props.disabled).toBe(false);
    expect(JSON.stringify(tree.toJSON())).not.toContain('late response lost');
    expect(put).toHaveBeenCalledTimes(1);
    expect(put).toHaveBeenCalledWith('/admin/module-access-settings/trader_sync', {feature: 'admin-service-status', mode: 'write'});
    expect(send).toHaveBeenCalledWith({state: 1});
});

test('two pending rows keep independent saving state and reread only after both complete', async () => {
    await mount();
    const trader = deferred(),
        worm = deferred();
    const write = jest.spyOn(moduleAccessService, 'save').mockImplementation(key => (key === 'trader_sync' ? trader.promise : worm.promise));
    await act(async () => {
        action('Open Trader Sync access').props.onClick();
        action('Open Worm access').props.onClick();
    });
    expect(action('Open Trader Sync access').props.loading).toBe(true);
    expect(action('Open Worm access').props.loading).toBe(true);
    await act(async () => trader.resolve({module_key: 'trader_sync', state: 1}));
    expect(action('Close Trader Sync access').props.loading).toBe(false);
    expect(action('Open Worm access').props.loading).toBe(true);
    expect(moduleAccessService.settings).toHaveBeenCalledTimes(1);
    jest.mocked(moduleAccessService.settings).mockResolvedValue(rows.map(row => ({...row, state: row.module_key === 'trader_sync' || row.module_key === 'worm' ? 1 : 2})) as any);
    await act(async () => worm.resolve({module_key: 'worm', state: 1}));
    expect(action('Close Trader Sync access').props.disabled).toBe(false);
    expect(action('Close Worm access').props.disabled).toBe(false);
    expect(moduleAccessService.settings).toHaveBeenCalledTimes(2);
    expect(write).toHaveBeenCalledTimes(2);
});
