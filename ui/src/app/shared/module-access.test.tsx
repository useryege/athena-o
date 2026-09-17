import {Context} from './context';
import * as React from 'react';
import {act, render, screen} from '@testing-library/react';
import {ModuleAccessProvider, ModuleAccessBoundary, useModuleAccessLease} from './module-access';
import {moduleAccessService} from './module-access-service';
import requests from './services/requests';
import {AccountDataModule} from './access-modules';

const states = (state = 1) => ['trader_sync', 'solana', 'market_radar', 'managed_oo', 'profit_sharing', 'worm'].map(module_key => ({module_key, state}));
const deferred = () => {
    let resolve!: (value: any) => void;
    const promise = Object.assign(
        new Promise<any>(r => {
            resolve = r;
        }),
        {abort: jest.fn()}
    );
    return {resolve, promise};
};
let visible = true;
let online = true;
const mount = () =>
    render(
        <ModuleAccessProvider realm='member' identity='a'>
            <span>core</span>
            <ModuleAccessBoundary moduleKey='worm'>
                <input aria-label='business draft' defaultValue='business body' />
            </ModuleAccessBoundary>
        </ModuleAccessProvider>
    );
beforeEach(() => {
    jest.useFakeTimers();
    visible = true;
    online = true;
    jest.spyOn(document, 'visibilityState', 'get').mockImplementation(() => (visible ? 'visible' : 'hidden'));
    jest.spyOn(navigator, 'onLine', 'get').mockImplementation(() => online);
    jest.spyOn(moduleAccessService, 'states').mockResolvedValue(states() as any);
    requests.configureAuthorizationRealm('member');
    requests.beginAuthorizationSession('a');
});
afterEach(() => {
    jest.restoreAllMocks();
    jest.useRealTimers();
    requests.endAuthorizationSession();
});
const tick = async (ms: number) => {
    await act(async () => {
        await jest.advanceTimersByTimeAsync(ms);
    });
};
test('core renders immediately; business waits for confirmation and closes while polling continues', async () => {
    const read = deferred();
    jest.mocked(moduleAccessService.states).mockReturnValueOnce(read.promise);
    mount();
    expect(screen.getByText('core')).toBeTruthy();
    expect(screen.queryByRole('textbox')).toBeNull();
    await act(async () => read.resolve(states()));
    expect(screen.getByRole('textbox')).toBeTruthy();
    jest.mocked(moduleAccessService.states).mockResolvedValue(states(2) as any);
    await tick(2000);
    expect(screen.queryByRole('textbox')).toBeNull();
    expect(screen.getByText('This module is not open yet')).toBeTruthy();
    jest.mocked(moduleAccessService.states).mockResolvedValue(states() as any);
    await tick(2000);
    expect(screen.getByRole('textbox')).toBeTruthy();
});
test('freshness starts at request launch; a slow response and overlapping timers cannot extend it', async () => {
    const read = deferred();
    jest.mocked(moduleAccessService.states).mockReturnValue(read.promise);
    mount();
    await tick(4000);
    expect(moduleAccessService.states).toHaveBeenCalledTimes(1);
    await act(async () => read.resolve(states()));
    expect(screen.getByRole('textbox')).toBeTruthy();
    const next = deferred();
    jest.mocked(moduleAccessService.states).mockReturnValue(next.promise);
    await tick(1001);
    expect(screen.queryByRole('textbox')).toBeNull();
});
test('an expired response cannot open business', async () => {
    const read = deferred();
    jest.mocked(moduleAccessService.states).mockReturnValue(read.promise);
    mount();
    await tick(5001);
    await act(async () => read.resolve(states()));
    expect(screen.queryByRole('textbox')).toBeNull();
});
test.each(['visibilitychange', 'offline', 'focus', 'pageshow', 'online'])('%s invalidates before refresh and rejects an older OPEN response', async event => {
    mount();
    await tick(0);
    expect(screen.getByRole('textbox')).toBeTruthy();
    const old = deferred();
    jest.mocked(moduleAccessService.states).mockReturnValueOnce(old.promise);
    await tick(2000);
    const fresh = deferred();
    jest.mocked(moduleAccessService.states).mockReturnValue(fresh.promise);
    if (event === 'visibilitychange') visible = false;
    if (event === 'offline') online = false;
    act(() => (event === 'visibilitychange' ? document : window).dispatchEvent(new Event(event)));
    expect(screen.queryByRole('textbox')).toBeNull();
    await act(async () => old.resolve(states()));
    expect(screen.queryByRole('textbox')).toBeNull();
    if (!visible || !online) {
        visible = true;
        online = true;
        act(() => window.dispatchEvent(new Event('focus')));
    }
    await act(async () => fresh.resolve(states()));
    expect(screen.getByRole('textbox')).toBeTruthy();
});
test('module closure cancels its requests and invalidates a captured lease across reopen', async () => {
    let current!: () => boolean;
    const Business = () => {
        const capture = useModuleAccessLease('worm');
        return (
            <button
                onClick={() => {
                    current = capture();
                }}
            >
                capture
            </button>
        );
    };
    render(
        <ModuleAccessProvider realm='member' identity='a'>
            <ModuleAccessBoundary moduleKey='worm'>
                <Business />
            </ModuleAccessBoundary>
        </ModuleAccessProvider>
    );
    await tick(0);
    act(() => screen.getByText('capture').click());
    expect(current()).toBe(true);
    const worm = requests.get('/worm-trading/balances', {module: AccountDataModule.WormTrading, mode: 'read'});
    const wallet = requests.get('/wallets', {module: AccountDataModule.Wallet, mode: 'read'});
    const abortWorm = jest.spyOn(worm, 'abort'),
        abortWallet = jest.spyOn(wallet, 'abort');
    jest.mocked(moduleAccessService.states).mockResolvedValue(states(2) as any);
    await tick(2000);
    expect(abortWorm).toHaveBeenCalledTimes(1);
    expect(abortWallet).not.toHaveBeenCalled();
    expect(current()).toBe(false);
    jest.mocked(moduleAccessService.states).mockResolvedValue(states() as any);
    await tick(2000);
    expect(current()).toBe(false);
});
test('separate provider instances do not share OPEN decisions', async () => {
    const second = deferred();
    jest.mocked(moduleAccessService.states)
        .mockResolvedValueOnce(states() as any)
        .mockReturnValue(second.promise);
    render(
        <>
            <ModuleAccessProvider realm='member' identity='a'>
                <ModuleAccessBoundary moduleKey='worm'>member business</ModuleAccessBoundary>
            </ModuleAccessProvider>
            <ModuleAccessProvider realm='admin' identity='b'>
                <ModuleAccessBoundary moduleKey='worm'>admin business</ModuleAccessBoundary>
            </ModuleAccessProvider>
        </>
    );
    await tick(0);
    expect(screen.getByText('member business')).toBeTruthy();
    expect(screen.queryByText('admin business')).toBeNull();
});

test.each(['wormCredentialReason', 'wormTradingReason', 'wormExecutionReason', 'wormPositionCashOutReason', 'wormPositionCashOutBatchReason'])(
    '%s forces confirmation before business loader and discards only the interrupted authorization intent',
    async field => {
        const first = deferred();
        jest.mocked(moduleAccessService.states).mockReturnValue(first.promise);
        window.history.replaceState(null, '', `/worm-trading?${field}=MODULE_ACCESS_CLOSED`);
        window.sessionStorage.setItem('athena.member.worm-trading.pending-connection-action', 'old action');
        window.sessionStorage.setItem('athena.member.worm-trading.pending-position-cash-out', '["accepted-operation"]');
        const load = jest.fn();
        const Business = () => {
            React.useEffect(load, []);
            return <span>callback business</span>;
        };
        render(
            <ModuleAccessProvider realm='member' identity='callback' returnSearch={window.location.search}>
                <ModuleAccessBoundary moduleKey='worm'>
                    <Business />
                </ModuleAccessBoundary>
                <span>core callback</span>
            </ModuleAccessProvider>
        );
        expect(load).not.toHaveBeenCalled();
        expect(screen.getByText('core callback')).toBeTruthy();
        expect(window.sessionStorage.getItem('athena.member.worm-trading.pending-connection-action')).toBeNull();
        expect(window.sessionStorage.getItem('athena.member.worm-trading.pending-position-cash-out')).toBe('["accepted-operation"]');
        expect(window.location.search).toBe('');
        await act(async () => first.resolve(states()));
        expect(load).toHaveBeenCalledTimes(1);
    }
);

test('module closure destroys its portaled confirmation and invalidates saved callbacks after reopen', async () => {
    const destroy = jest.fn();
    let options: any;
    const mutation = jest.fn();
    const context = {
        modal: {
            confirm: (value: any) => {
                options = value;
                return {destroy};
            }
        },
        notifications: {},
        navigation: {},
        baseHref: ''
    } as any;
    const Business = () => {
        const ctx = React.useContext(Context);
        return <button onClick={() => ctx.modal.confirm({title: 'Cash Out', onOk: mutation})}>open confirmation</button>;
    };
    render(
        <Context.Provider value={context}>
            <ModuleAccessProvider realm='member' identity='modal'>
                <ModuleAccessBoundary moduleKey='worm'>
                    <Business />
                </ModuleAccessBoundary>
            </ModuleAccessProvider>
        </Context.Provider>
    );
    await tick(0);
    act(() => screen.getByText('open confirmation').click());
    jest.mocked(moduleAccessService.states).mockResolvedValue(states(2) as any);
    await tick(2000);
    expect(destroy).toHaveBeenCalledTimes(1);
    jest.mocked(moduleAccessService.states).mockResolvedValue(states() as any);
    await tick(2000);
    await act(async () => options.onOk());
    expect(mutation).not.toHaveBeenCalled();
});
