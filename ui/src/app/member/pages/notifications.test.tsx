import renderer, {act} from 'react-test-renderer';
import {Button} from 'antd';
import {MemoryRouter, useLocation} from 'react-router-dom';
import {NotificationsPage} from './notifications';
import {AuthorizationCtx, Context} from '../../shared/context';
import {parseUserInfo} from '../../shared/models';
import {clearTraderSyncState, readAddDraft, saveAddDraft} from './trader-sync/state';
import {ensureMemberBusinessServices, memberServices as services} from '../services';
import {normalizeResolvedTarget} from '../trader-sync-models';
import fixture from '../testdata/trader-sync/resolve-saved-note-existing.json';

const ctx = {
    notifications: {success: jest.fn(), error: jest.fn(), info: jest.fn(), warning: jest.fn()},
    modal: {confirm: jest.fn(() => ({destroy: jest.fn()})), info: jest.fn(), error: jest.fn()},
    navigation: {goto: jest.fn(), replace: jest.fn()},
    baseHref: '/'
};
const Location = () => <output>{JSON.stringify(useLocation())}</output>;
let tree: renderer.ReactTestRenderer;
const mount = async (returnTo: string, ownerId = 'A') => {
    const user = parseUserInfo({accountId: ownerId, loggedIn: true});
    await act(async () => {
        tree = renderer.create(
            <AuthorizationCtx.Provider
                value={{user, isAdmin: false, access: () => 2, canRead: () => true, canWrite: () => true, revision: 1, lastCheckedAt: 0, refresh: jest.fn()}}
            >
                <Context.Provider value={ctx}>
                    <MemoryRouter future={{v7_startTransition: true, v7_relativeSplatPath: true}} initialEntries={[{pathname: '/notifications', state: {returnTo}}]}>
                        <NotificationsPage />
                        <Location />
                    </MemoryRouter>
                </Context.Provider>
            </AuthorizationCtx.Provider>
        );
    });
};
const returnButtons = () => tree.root.findAllByType(Button).filter(item => item.props.children === 'Return to Trader Sync');
beforeEach(() => {
    ensureMemberBusinessServices();
    jest.useFakeTimers();
    sessionStorage.clear();
    window.matchMedia = jest.fn().mockImplementation(query => ({
        matches: false,
        media: query,
        addListener: jest.fn(),
        removeListener: jest.fn(),
        addEventListener: jest.fn(),
        removeEventListener: jest.fn()
    }));
    jest.spyOn(services.memberNotifications, 'getTelegramSettings').mockResolvedValue({
        botAvailable: true,
        botUsername: 'bot',
        binding: {status: 'connected', boundAt: '2026-09-11T00:00:00Z', revision: 1, telegramUsername: 'tester', telegramDisplayName: 'Tester'}
    });
});
afterEach(() => {
    if (tree) act(() => tree.unmount());
    clearTraderSyncState();
    jest.restoreAllMocks();
    jest.useRealTimers();
});
test('exact allowpath and matching owner draft allow return without renewing token', async () => {
    const target = normalizeResolvedTarget(fixture.target);
    saveAddDraft({ownerId: 'A', input: target.wallet, note: 'draft', noteEdited: true, target, returnPath: '/trader-sync', scrollY: 100});
    await mount('/trader-sync/add');
    expect(returnButtons()).toHaveLength(1);
    await act(async () => {
        jest.advanceTimersByTime(301_000);
    });
    await act(async () => {
        returnButtons()[0].props.onClick();
    });
    expect(JSON.parse(tree.root.findByType('output').props.children).pathname).toBe('/trader-sync/add');
    expect(readAddDraft('A')?.target?.expiresAt).toBe(target.expiresAt);
});
test.each(['/trader-sync/add?token=secret', '//evil.test/trader-sync/add', '/trader-sync/add/'])('rejects non-exact returnTo %s', async path => {
    saveAddDraft({ownerId: 'A', input: 'wallet', note: '', noteEdited: false, returnPath: '/trader-sync', scrollY: 0});
    await mount(path);
    expect(returnButtons()).toHaveLength(0);
});
test('missing draft and different owner cannot return', async () => {
    await mount('/trader-sync/add');
    expect(returnButtons()).toHaveLength(0);
    act(() => tree.unmount());
    saveAddDraft({ownerId: 'A', input: 'wallet', note: '', noteEdited: false, returnPath: '/trader-sync', scrollY: 0});
    await mount('/trader-sync/add', 'B');
    expect(returnButtons()).toHaveLength(0);
});
test('disconnect keeps its single confirmation and explains queue permission boundary', async () => {
    await mount('/trader-sync/add');
    act(() =>
        tree.root
            .findAllByType(Button)
            .find(item => item.props.children === 'Disconnect')!
            .props.onClick()
    );
    expect(ctx.modal.confirm).toHaveBeenCalledTimes(1);
    expect((ctx.modal.confirm.mock.calls[0] as any)[0].content).toContain('already authorized');
    expect(JSON.stringify(tree.toJSON())).toContain('not backfilled');
});
test('retains the existing visible three-second single-flight polling and aborts on unmount', async () => {
    let finish!: (value: any) => void;
    const pending = Object.assign(
        new Promise<any>(resolve => {
            finish = resolve;
        }),
        {abort: jest.fn()}
    );
    jest.mocked(services.memberNotifications.getTelegramSettings).mockReturnValue(pending);
    Object.defineProperty(document, 'visibilityState', {configurable: true, value: 'visible'});
    await mount('/trader-sync/add');
    await act(async () => {
        jest.advanceTimersByTime(6000);
        window.dispatchEvent(new Event('focus'));
    });
    expect(services.memberNotifications.getTelegramSettings).toHaveBeenCalledTimes(1);
    await act(async () => {
        finish({botAvailable: true, botUsername: 'bot'});
    });
    Object.defineProperty(document, 'visibilityState', {configurable: true, value: 'hidden'});
    await act(async () => {
        jest.advanceTimersByTime(6000);
    });
    expect(services.memberNotifications.getTelegramSettings).toHaveBeenCalledTimes(1);
    Object.defineProperty(document, 'visibilityState', {configurable: true, value: 'visible'});
    const next = Object.assign(new Promise<any>(() => undefined), {abort: jest.fn()});
    jest.mocked(services.memberNotifications.getTelegramSettings).mockReturnValue(next);
    await act(async () => {
        document.dispatchEvent(new Event('visibilitychange'));
    });
    expect(services.memberNotifications.getTelegramSettings).toHaveBeenCalledTimes(2);
    act(() => tree.unmount());
    expect(next.abort).toHaveBeenCalled();
});
test('clearing module state removes a visible return entry synchronously', async () => {
    saveAddDraft({ownerId: 'A', input: 'wallet', note: '', noteEdited: false, returnPath: '/trader-sync', scrollY: 0});
    await mount('/trader-sync/add');
    expect(returnButtons()).toHaveLength(1);
    act(clearTraderSyncState);
    expect(returnButtons()).toHaveLength(0);
});

test('failed setup has one recovery action while the existing Connected binding remains visible', async () => {
    jest.mocked(services.memberNotifications.getTelegramSettings).mockResolvedValue({
        botAvailable: true,
        botUsername: 'bot',
        binding: {status: 'connected', boundAt: '2026-09-11T00:00:00Z', revision: 1, telegramUsername: 'tester', telegramDisplayName: 'Tester'},
        attempt: {id: 'failed-attempt', status: 'failed', expiresAt: '2026-09-11T00:05:00Z', failureReason: 'binding_failed'}
    });
    await mount('');
    const recovery = tree.root.findAllByType(Button).filter(item => item.props.children === 'Create new link' || item.props.children === 'Reconnect');
    expect(recovery).toHaveLength(1);
    expect(recovery[0].props.children).toBe('Create new link');
    expect(JSON.stringify(tree.toJSON())).toContain('Connected');
    expect(JSON.stringify(tree.toJSON())).toContain('Tester');
});

test('disconnect confirmation names the current identity and focuses Keep connected', async () => {
    await mount('');
    act(() => tree.root.findAllByType(Button).find(item => item.props.children === 'Disconnect')!.props.onClick());
    const options = (ctx.modal.confirm.mock.calls.at(-1) as any)[0];
    expect(options.content).toContain('Tester (@tester)');
    expect(options.cancelText).toBe('Keep connected');
    expect(options.autoFocusButton).toBe('cancel');
});
