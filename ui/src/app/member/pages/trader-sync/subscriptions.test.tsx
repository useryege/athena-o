import renderer, {act} from 'react-test-renderer';
import {Button} from 'antd';
import {MemoryRouter} from 'react-router-dom';
import {TraderSyncSubscriptionsPage} from './subscriptions';
import {allowedSubscriptionActions} from './subscription-state';
import {clearTraderSyncState} from './state';
import {ensureMemberBusinessServices, memberServices as services} from '../../services';
import {normalizeSubscription, Subscription} from '../../trader-sync-models';
import fixture from '../../testdata/trader-sync/01.json';
let tree: renderer.ReactTestRenderer;
const sub = normalizeSubscription(fixture.subscription);
const button = (label: string) => tree.root.findAllByType(Button).find(item => item.props.children === label)!;
const deferredPage = () => {
    let resolve!: (value: Awaited<ReturnType<typeof services.traderSync.listSubscriptions>>) => void;
    const request = Object.assign(new Promise<Awaited<ReturnType<typeof services.traderSync.listSubscriptions>>>(yes => (resolve = yes)), {abort: jest.fn()});
    return {request, resolve};
};
beforeEach(() => {
    ensureMemberBusinessServices();
    window.matchMedia = jest.fn().mockImplementation(query => ({matches: false, media: query, addListener: jest.fn(), removeListener: jest.fn()}));
    jest.spyOn(window, 'scrollTo').mockImplementation(() => undefined);
});
afterEach(() => {
    if (tree) act(() => tree.unmount());
    clearTraderSyncState();
    jest.restoreAllMocks();
});
test.each([
    ['pending_baseline', ['cancel']],
    ['healthy', ['pause', 'cancel']],
    ['interrupted', ['pause', 'cancel']],
    ['paused', ['resume', 'cancel']],
    ['permission_disabled', ['resume', 'cancel']],
    ['cancelled', []]
])('actions for %s', (status, expected) => expect(allowedSubscriptionActions(status as Subscription['status'])).toEqual(expected));
test('current and cancelled have independent page cursors and show complete wallet and retained public name', async () => {
    const list = jest.spyOn(services.traderSync, 'listSubscriptions').mockImplementation(input =>
        Promise.resolve({
            subscriptions: [{...sub, note: 'desk', status: input?.view === 'cancelled' ? 'cancelled' : 'healthy'}],
            quota: {used: 1, limit: 10},
            asOf: sub.updatedAt,
            page: {nextCursor: input?.cursor ? undefined : `${input?.view}-next`}
        })
    );
    await act(async () => {
        tree = renderer.create(
            <MemoryRouter future={{v7_startTransition: true, v7_relativeSplatPath: true}}>
                <TraderSyncSubscriptionsPage ownerId='A' />
            </MemoryRouter>
        );
    });
    expect(list).toHaveBeenCalled();
    expect(list.mock.calls[0][0]).toMatchObject({view: 'current', pageSize: 50});
    expect(JSON.stringify(tree.toJSON())).toContain(sub.wallet);
    expect(JSON.stringify(tree.toJSON())).toContain('desk');
    await act(async () => button('Next').props.onClick());
    expect(list.mock.calls.at(-1)![0]?.cursor).toBe('current-next');
    await act(async () => button('Cancelled').props.onClick());
    expect(list.mock.calls.at(-1)![0]).toMatchObject({view: 'cancelled', cursor: undefined});
    expect(button('Resume')).toBeUndefined();
    await act(async () => button('Current').props.onClick());
    expect(list.mock.calls.at(-1)![0]?.cursor).toBe('current-next');
});

test('Latest subscriptions immediately reloads page one and repeated clicks remain single flight', async () => {
    const next = deferredPage();
    const page = {subscriptions: [sub], page: {}, quota: {used: 1, limit: 10}, asOf: sub.updatedAt};
    const list = jest.spyOn(services.traderSync, 'listSubscriptions').mockResolvedValueOnce(page).mockReturnValue(next.request);
    await act(async () => {
        tree = renderer.create(
            <MemoryRouter future={{v7_startTransition: true, v7_relativeSplatPath: true}}>
                <TraderSyncSubscriptionsPage ownerId='A' />
            </MemoryRouter>
        );
    });
    expect(list).toHaveBeenCalledTimes(1);
    act(() => button('Latest subscriptions').props.onClick());
    expect(list).toHaveBeenCalledTimes(2);
    expect(list.mock.calls[1][0]).toMatchObject({view: 'current', cursor: undefined});
    act(() => button('Latest subscriptions').props.onClick());
    expect(list).toHaveBeenCalledTimes(2);
    await act(async () => next.resolve(page));
});

test('Latest subscriptions returns from history to the first page without reloading the old cursor', async () => {
    const first = {subscriptions: [sub], page: {nextCursor: 'current-next'}, quota: {used: 1, limit: 10}, asOf: sub.updatedAt};
    const list = jest.spyOn(services.traderSync, 'listSubscriptions').mockResolvedValue({...first, page: {}}).mockResolvedValueOnce(first);
    await act(async () => {
        tree = renderer.create(
            <MemoryRouter future={{v7_startTransition: true, v7_relativeSplatPath: true}}>
                <TraderSyncSubscriptionsPage ownerId='A' />
            </MemoryRouter>
        );
    });
    await act(async () => button('Next').props.onClick());
    expect(list.mock.calls.at(-1)![0]?.cursor).toBe('current-next');
    await act(async () => button('Latest subscriptions').props.onClick());
    expect(list.mock.calls.at(-1)![0]?.cursor).toBeUndefined();
    expect(list.mock.calls.filter(call => call[0]?.cursor === 'current-next')).toHaveLength(1);
});

test('switching views restores their separate scroll positions', async () => {
    jest.spyOn(services.traderSync, 'listSubscriptions').mockResolvedValue({subscriptions: [sub], page: {}, quota: {used: 1, limit: 10}, asOf: sub.updatedAt});
    await act(async () => {
        tree = renderer.create(
            <MemoryRouter future={{v7_startTransition: true, v7_relativeSplatPath: true}}>
                <TraderSyncSubscriptionsPage ownerId='A' />
            </MemoryRouter>
        );
    });
    Object.defineProperty(window, 'scrollY', {configurable: true, value: 120});
    await act(async () => button('Cancelled').props.onClick());
    Object.defineProperty(window, 'scrollY', {configurable: true, value: 240});
    await act(async () => button('Current').props.onClick());
    expect(window.scrollTo).toHaveBeenLastCalledWith({top: 120, behavior: 'instant'});
    Object.defineProperty(window, 'scrollY', {configurable: true, value: 0});
});

test('failed background list refresh retains exact page facts with gateway reason and no loading replacement', async () => {
    jest.useFakeTimers();
    const list = jest.spyOn(services.traderSync, 'listSubscriptions').mockResolvedValue({subscriptions: [sub], page: {}, quota: {used: 1, limit: 10}, asOf: sub.updatedAt});
    await act(async () => {
        tree = renderer.create(
            <MemoryRouter future={{v7_startTransition: true, v7_relativeSplatPath: true}}>
                <TraderSyncSubscriptionsPage ownerId='A' />
            </MemoryRouter>
        );
    });
    list.mockRejectedValue({status: 503, response: {body: {code: 14, message: 'Collector database unavailable'}}});
    await act(async () => jest.advanceTimersByTime(5000));
    const text = JSON.stringify(tree.toJSON());
    expect(text).toContain(sub.wallet);
    expect(text).toContain('Collector database unavailable');
    expect(text).toContain('out of date');
    expect(text).not.toContain('resource-table-compact__loading');
    jest.useRealTimers();
});

test.each(['paused', 'cancelled'] as const)('list translates the fixed %s queue notice without stripping user notes', async status => {
    // Exact system payload from internal/tradersync/store/reads.go; remaining fields use the recorded gateway fixture.
    const queueNotice = '已排队通知仍会继续发送，可能稍后收到';
    jest.spyOn(services.traderSync, 'listSubscriptions').mockResolvedValue({
        subscriptions: [{...sub, status, queueNotice, note: '我的备注'}],
        page: {},
        quota: {used: 1, limit: 10},
        asOf: sub.updatedAt
    });
    await act(async () => {
        tree = renderer.create(
            <MemoryRouter future={{v7_startTransition: true, v7_relativeSplatPath: true}}>
                <TraderSyncSubscriptionsPage ownerId='A' />
            </MemoryRouter>
        );
    });
    const text = JSON.stringify(tree.toJSON());
    expect(text).not.toContain(queueNotice);
    expect(text).toContain('Already queued notifications continue and may arrive later.');
    expect(text).toContain('我的备注');
});
