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
