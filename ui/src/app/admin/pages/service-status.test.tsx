import * as React from 'react';
import renderer, {act} from 'react-test-renderer';
import {MemoryRouter} from 'react-router-dom';
import {AuthorizationCtx, type AuthorizationState} from '../../shared/context';
import {parseUserInfo} from '../../shared/models';
import {clearAsyncDataCache, AppPage} from '../../components';
import {beginAdminReadSession, endAdminReadSession} from '../read-scope';
import {ServiceStatusPage} from './service-status';
import {adminServices as services, configureAdminSessionServices, ensureAdminBusinessServices} from '../services';
import {normalizeRuntimeStatus} from '../trader-sync-models';
// Renderer has no real layout nodes; tab visibility/keyboard behavior is covered by theme:admin Playwright.
jest.mock('antd', () => ({
    ...jest.requireActual('antd'),
    Tabs: (props: any) => (
        <div>
            {props.items.map((item: any) => (
                <div key={item.key}>{item.children}</div>
            ))}
        </div>
    )
}));
configureAdminSessionServices();
ensureAdminBusinessServices();
const user = parseUserInfo({accountId: 'admin-A', iss: 'issuer', loggedIn: true, administrator: true});
const deferred = <T,>() => {
    let resolve!: (v: T) => void;
    let reject!: (e: Error) => void;
    const promise = Object.assign(
        new Promise<T>((r, j) => {
            resolve = r;
            reject = j;
        }),
        {abort: jest.fn()}
    );
    return {promise, resolve, reject};
};
let tree: renderer.ReactTestRenderer;
let visible = true;
const text = () => JSON.stringify(tree.toJSON());
const runtime = {collectorConnected: true, collectorEpoch: '2', filterRevision: '3', metrics: [], asOf: '2026-09-11T01:00:00Z'};
const notification = {
    status: 'recovering',
    started: true,
    botAvailable: true,
    pollerActive: true,
    systemPendingCount: 7,
    recovery: {state: 'waiting', reason: 'non_first_start_barrier', remainingMillis: '9007199254740993', elapsedMillis: '5000', clockSource: 'sender_monotonic'}
};
const mount = async () => {
    await act(async () => {
        tree = renderer.create(
            <MemoryRouter future={{v7_startTransition: true, v7_relativeSplatPath: true}}>
                <AuthorizationCtx.Provider value={{user, isAdmin: true} as AuthorizationState}>
                    <ServiceStatusPage />
                </AuthorizationCtx.Provider>
            </MemoryRouter>
        );
    });
};
beforeEach(() => {
    window.matchMedia = jest.fn().mockImplementation(query => ({
        matches: false,
        media: query,
        addListener: jest.fn(),
        removeListener: jest.fn(),
        addEventListener: jest.fn(),
        removeEventListener: jest.fn()
    }));
    jest.spyOn(services.adminTraderSync, 'getRuntimeStatus').mockResolvedValue(runtime);
    jest.useFakeTimers();
    visible = true;
    beginAdminReadSession(user);
    jest.spyOn(document, 'visibilityState', 'get').mockImplementation(() => (visible ? 'visible' : 'hidden'));
    jest.spyOn(services.serviceStatus, 'list').mockResolvedValue({items: [{name: 'wallet', status: 'SERVING'}], checkedAt: 1789088400} as any);
    jest.spyOn(services.adminNotifications, 'getRuntimeStatus').mockResolvedValue(notification as any);
});
afterEach(() => {
    act(() => {
        tree?.unmount();
        endAdminReadSession();
    });
    clearAsyncDataCache();
    jest.restoreAllMocks();
    jest.useRealTimers();
});
test('Service Status does not overlap a pending health read across timers or manual refresh', async () => {
    const first = deferred<any>();
    jest.mocked(services.serviceStatus.list).mockReturnValue(first.promise);
    await mount();
    await act(async () => jest.advanceTimersByTimeAsync(20000));
    act(() => tree.root.findByType(AppPage).props.onRefresh());
    expect(services.serviceStatus.list).toHaveBeenCalledTimes(1);
    await act(async () => first.resolve({items: [], checkedAt: 1789088400}));
});
test('hidden page stops new reads and visible resumes all three immediately', async () => {
    const trader = jest.spyOn(services.adminTraderSync, 'getRuntimeStatus').mockResolvedValue(runtime);
    await mount();
    act(() => {
        visible = false;
        document.dispatchEvent(new Event('visibilitychange'));
    });
    await act(async () => jest.advanceTimersByTimeAsync(20000));
    expect(services.serviceStatus.list).toHaveBeenCalledTimes(1);
    expect(services.adminNotifications.getRuntimeStatus).toHaveBeenCalledTimes(1);
    expect(trader).toHaveBeenCalledTimes(1);
    await act(async () => {
        visible = true;
        document.dispatchEvent(new Event('visibilitychange'));
    });
    expect(trader).toHaveBeenCalledTimes(2);
    expect(services.serviceStatus.list).toHaveBeenCalledTimes(2);
});
test('one failed source keeps its prior counters and time, while the other sources publish independently', async () => {
    jest.spyOn(services.adminTraderSync, 'getRuntimeStatus').mockResolvedValue(runtime);
    await mount();
    const before = text();
    expect(before).toContain('9007199254740993 ms');
    jest.mocked(services.adminNotifications.getRuntimeStatus).mockRejectedValue(new Error('notification offline'));
    jest.mocked(services.adminTraderSync.getRuntimeStatus).mockResolvedValue({...runtime, filterRevision: 'new-filter', asOf: '2026-09-11T01:00:10Z'});
    await act(async () => jest.advanceTimersByTimeAsync(10000));
    expect(text()).toContain('notification offline');
    expect(text()).toContain('Stale');
    expect(text()).toContain('9007199254740993 ms');
    expect(text()).toContain('new-filter');
    expect(text()).toContain('09:00:10');
    act(() => endAdminReadSession());
    expect(text()).not.toContain('new-filter');
    expect(text()).not.toContain('9007199254740993');
});
test('runtime displays genuine metric units and per-kind provenance; unavailable raw data is not zero', async () => {
    // Synthetic window: captured Task13 gateway fixtures contain gauge/epoch only.
    const value = normalizeRuntimeStatus({
        ...runtime,
        metrics: [
            {name: 'raw_observation_available', value: '0', unit: 'boolean', kind: 'gauge'},
            {name: 'raw_queue_depth', unit: 'raw_logs', kind: 'gauge'},
            {name: 'synthetic_window', value: '3', unit: 'activities', kind: 'window', windowStart: '2026-09-11T00:59:00Z', windowEnd: '2026-09-11T01:00:00Z'},
            {name: 'epoch_count', value: '9007199254740999', unit: 'rounds', kind: 'epoch', serviceEpoch: 'opaque-producer'}
        ]
    });
    jest.spyOn(services.adminTraderSync, 'getRuntimeStatus').mockResolvedValue(value);
    await mount();
    expect(text()).toContain('Not observable');
    expect(text()).toContain('raw_logs');
    expect(text()).toContain('08:59:00');
    expect(text()).toContain('opaque-producer');
    expect(text()).toContain('9007199254740999');
});

test.each([
    ['02-initializing', 'recovering', 'Unavailable'],
    ['03-first-completed', 'running', '0 ms'],
    ['04-waiting', 'recovering', '55000 ms'],
    ['06-fatal', 'failed', '55000 ms'],
    ['07-stopped', 'stopped', '55000 ms'],
    ['08-recovery-failed', 'failed', 'Unavailable']
])('page reflects frozen %s recovery independently of top-level health', async (name, status, remaining) => {
    const body = require(`../__fixtures__/trader-sync/notification-${name}.json`);
    jest.mocked(services.adminNotifications.getRuntimeStatus).mockResolvedValue({...notification, status: body.status, recovery: body.recovery} as any);
    await mount();
    const section = tree.root.findAllByType(require('../../components').Section).find(item => item.props.title === 'Notification Runtime')!;
    expect(section.props.extra.props.children).toBe(status);
    const facts = section.findAllByType(require('../components/operation-facts').OperationFacts).flatMap(grid => grid.props.items);
    expect(facts.find(item => item.label === 'Recovery remaining')?.value).toBe(remaining);
});

test('source tabs declare independent source panels without additional reads', async () => {
    await mount();
    const tabs = tree.root.findByType(require('antd').Tabs);
    expect(tabs.props.defaultActiveKey).toBe('services');
    expect(tabs.props.items.map((item: any) => item.key)).toEqual(['services', 'notifications', 'trader']);
    const before = jest.mocked(services.adminNotifications.getRuntimeStatus).mock.calls.length;
    expect(tabs.props.items.find((item: any) => item.key === 'notifications').children).toBeDefined();
    expect(services.adminNotifications.getRuntimeStatus).toHaveBeenCalledTimes(before);
    expect(text()).not.toContain('Worm Markets');
});
