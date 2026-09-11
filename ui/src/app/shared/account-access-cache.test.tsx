import * as React from 'react';
import renderer, {act} from 'react-test-renderer';
import {useCachedAsyncData, setAsyncDataCacheSession, clearAsyncDataCache} from '../components/data';
import {revokeLostModuleAccess} from '../member/app';
import {moduleAccessLevels} from './account-access';
import {parseAccountAccess} from './models';
import {AccountDataModule} from './access-modules';

test('RW to NONE clears saved data, aborts pending requests and ignores a late response', async () => {
    setAsyncDataCacheSession('member', 'owner', 1);
    let resolveLate: (value: string) => void;
    let aborted = false;
    const late = new Promise<string>(resolve => {resolveLate = resolve;}) as Promise<string> & {abort?: () => void};
    late.abort = () => {aborted = true;};
    let state: any;
    const Probe = () => {
        state = useCachedAsyncData('trader-note', () => late, {module: AccountDataModule.TraderSync, staleTimeMs: 60000});
        return <span>{state.data || 'empty'}</span>;
    };
    let tree: renderer.ReactTestRenderer;
    await act(async () => {tree = renderer.create(<Probe />);});
    await act(async () => {resolveLate!('private saved note');});
    expect(JSON.stringify(tree!.toJSON())).toContain('private saved note');
    const pending = new Promise<string>(resolve => {resolveLate = resolve;}) as typeof late;
    pending.abort = () => {aborted = true;};
    const PendingProbe = () => {
        state = useCachedAsyncData('trader-note', () => pending, {module: AccountDataModule.TraderSync, staleTimeMs: 60000});
        return <span>{state.data || 'empty'}</span>;
    };
    await act(async () => {tree!.update(<PendingProbe />);});
    await act(async () => {state.reload();});
    const before = moduleAccessLevels(parseAccountAccess({moduleAccess: [{module: 'trader_sync', dataAccess: 'read_write'}]}));
    const after = moduleAccessLevels(parseAccountAccess({}));
    await act(async () => {revokeLostModuleAccess(before, after);tree!.unmount();});
    expect(aborted).toBe(true);
    await act(async () => {resolveLate!('late private note');});
    let fresh: renderer.ReactTestRenderer;
    const freshLoad = () => new Promise<string>(() => {});
    const FreshProbe = () => {const value = useCachedAsyncData('trader-note', freshLoad, {module: AccountDataModule.TraderSync, staleTimeMs: 60000});return <span>{value.data || 'empty'}</span>;};
    await act(async () => {fresh = renderer.create(<FreshProbe />);});
    expect(JSON.stringify(fresh!.toJSON())).toContain('empty');
    await act(async () => {fresh!.unmount();clearAsyncDataCache();});
});
