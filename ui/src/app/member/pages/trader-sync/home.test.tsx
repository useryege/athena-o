import renderer, {act} from 'react-test-renderer';
import {MemoryRouter} from 'react-router-dom';
import {TraderSyncHomePage} from './home';
import {clearTraderSyncState} from './state';
import {ensureMemberBusinessServices, memberServices as services} from '../../services';
import {normalizeActivity, normalizeSubscription, ActivityPage} from '../../trader-sync-models';
import fixture from '../../testdata/trader-sync/01.json';
import activityFixture from '../../testdata/trader-sync/05.json';
let tree: renderer.ReactTestRenderer;
const activity = normalizeActivity(activityFixture.activity);
const sub = normalizeSubscription(fixture.subscription);
const page = (ids = ['12', '11'], refreshCursor = 'signed-current-page'): ActivityPage => ({
    activities: ids.map(id => ({...activity, id})),
    page: {snapshot: 'opaque-snapshot', refreshCursor, nextCursor: 'signed-next-page', hasNewer: false}
});
const button = (text: string) => tree.root.findAllByType('button').find(n => n.props.children === text);
const mount = async () =>
    act(async () => {
        tree = renderer.create(
            <MemoryRouter future={{v7_startTransition: true, v7_relativeSplatPath: true}}>
                <TraderSyncHomePage ownerId='A' />
            </MemoryRouter>
        );
    });
beforeEach(() => {
    ensureMemberBusinessServices();
    jest.useFakeTimers();
    window.matchMedia = jest.fn().mockImplementation(query => ({matches: false, media: query, addListener: jest.fn(), removeListener: jest.fn()}));
    jest.spyOn(window, 'scrollTo').mockImplementation(() => undefined);
    jest.spyOn(services.traderSync, 'listSubscriptions').mockResolvedValue({subscriptions: [sub], page: {}, quota: {used: 1, limit: 10}, asOf: sub.updatedAt});
    jest.spyOn(services.memberNotifications, 'getTelegramSettings').mockResolvedValue({} as any);
});
afterEach(() => {
    if (tree) act(() => tree.unmount());
    clearTraderSyncState();
    jest.restoreAllMocks();
    jest.useRealTimers();
});
test('refreshes current fixed page and loads latest only on explicit action', async () => {
    const list = jest.spyOn(services.traderSync, 'listActivities').mockResolvedValue(page());
    await mount();
    expect(list).toHaveBeenCalledWith(expect.objectContaining({pageSize: 50}));
    const incoming = page();
    incoming.page.hasNewer = true;
    list.mockResolvedValue(incoming);
    await act(async () => jest.advanceTimersByTime(5000));
    expect(list).toHaveBeenLastCalledWith(expect.objectContaining({refreshCursor: 'signed-current-page'}));
    expect(list.mock.calls.at(-1)![0]?.cursor).toBeUndefined();
    expect(button('New activity available')).toBeDefined();
    await act(async () => button('New activity available')!.props.onClick());
    expect(list.mock.calls.at(-1)![0]?.cursor).toBeUndefined();
    expect(list.mock.calls.at(-1)![0]?.refreshCursor).toBeUndefined();
});
test('protocol failure retains current rows and offers retry', async () => {
    const list = jest.spyOn(services.traderSync, 'listActivities').mockResolvedValue(page());
    await mount();
    list.mockResolvedValue(page(['13', '12']));
    await act(async () => jest.advanceTimersByTime(5000));
    expect(JSON.stringify(tree.toJSON())).toContain('protocol');
    expect(tree.root.findAll(n => n.props['data-activity-id']).map(n => n.props['data-activity-id'])).toEqual(['12', '11']);
});
test('next uses cursor and previous restores original refresh token', async () => {
    const list = jest.spyOn(services.traderSync, 'listActivities').mockResolvedValue(page());
    await mount();
    expect(button('Next')).toBeDefined();
    list.mockResolvedValue(page(['10'], 'historical-refresh'));
    await act(async () => button('Next')!.props.onClick());
    expect(list.mock.calls.at(-1)![0]?.cursor).toBe('signed-next-page');
    list.mockResolvedValue(page());
    await act(async () => button('Previous')!.props.onClick());
    await act(async () => jest.advanceTimersByTime(5000));
    expect(list.mock.calls.at(-1)![0]?.refreshCursor).toBe('signed-current-page');
});

test('empty refresh stays empty, latest failure keeps hint and retry remains current-page refresh', async () => {
    const list = jest.spyOn(services.traderSync, 'listActivities').mockResolvedValue(page([]));
    await mount();
    list.mockResolvedValue({...page([]), page: {...page([]).page, hasNewer: true}});
    await act(async () => jest.advanceTimersByTime(5000));
    expect(tree.root.findAll(n => n.props['data-activity-id'])).toHaveLength(0);
    list.mockRejectedValue(new Error('latest offline'));
    await act(async () => button('New activity available')!.props.onClick());
    expect(button('New activity available')).toBeDefined();
    list.mockResolvedValue(page([]));
    await act(async () => jest.advanceTimersByTime(5000));
    expect(list.mock.calls.at(-1)![0]?.refreshCursor).toBe('signed-current-page');
});
test('date and target filters persist when explicitly loading new activity', async () => {
    const list = jest.spyOn(services.traderSync, 'listActivities').mockResolvedValue(page());
    await mount();
    await act(async () => button(sub.note || sub.targetDisplay.displayName.value || sub.wallet)!.props.onClick());
    const inputs = tree.root.findAllByType('input').filter(n => n.props.type === 'date');
    act(() => {
        inputs[0].props.onChange({target: {value: '2026-09-10'}});
        inputs[1].props.onChange({target: {value: '2026-09-11'}});
    });
    await act(async () => tree.root.findByType('form').props.onSubmit({preventDefault: () => undefined}));
    expect(list.mock.calls.at(-1)![0]).toMatchObject({
        subscriptionId: sub.id,
        from: '2026-09-09T16:00:00.000Z',
        to: '2026-09-11T16:00:00.000Z',
        cursor: undefined,
        refreshCursor: undefined
    });
});
test('cancelled sidebar target keeps its historical filter', async () => {
    const list = jest.spyOn(services.traderSync, 'listActivities').mockResolvedValue(page());
    await mount();
    await act(async () => button(sub.note || sub.targetDisplay.displayName.value || sub.wallet)!.props.onClick());
    jest.mocked(services.traderSync.listSubscriptions).mockResolvedValue({subscriptions: [], page: {}, quota: {used: 0, limit: 10}, asOf: sub.updatedAt});
    await act(async () => jest.advanceTimersByTime(5000));
    expect(JSON.stringify(tree.toJSON())).toContain('Its activity history remains selected');
    expect(list.mock.calls.at(-1)![0]?.subscriptionId).toBe(sub.id);
});
test('unmount return restores current page and scroll; a refresh during scrolling does not lose location', async () => {
    const list = jest.spyOn(services.traderSync, 'listActivities').mockResolvedValue(page());
    await mount();
    Object.defineProperty(window, 'scrollY', {configurable: true, value: 340});
    act(() => window.dispatchEvent(new Event('scroll')));
    await act(async () => jest.advanceTimersByTime(5000));
    act(() => tree.unmount());
    await mount();
    expect(list.mock.calls.at(-1)![0]?.refreshCursor).toBe('signed-current-page');
    expect(window.scrollTo).toHaveBeenLastCalledWith({top: 340, behavior: 'instant'});
    Object.defineProperty(window, 'scrollY', {configurable: true, value: 0});
});
test('scope invalidation empties rows synchronously and fences delayed activity responses', async () => {
    let resolve!: (value: ActivityPage) => void;
    const list = jest.spyOn(services.traderSync, 'listActivities').mockResolvedValue(page());
    await mount();
    list.mockImplementation(
        () =>
            new Promise(done => {
                resolve = done;
            })
    );
    await act(async () => jest.advanceTimersByTime(5000));
    act(() => clearTraderSyncState());
    expect(tree.root.findAll(n => n.props['data-activity-id'])).toHaveLength(0);
    await act(async () => resolve(page(['13'])));
    expect(tree.root.findAll(n => n.props['data-activity-id'])).toHaveLength(0);
});
test('three page reads remain single flight and stop while hidden', async () => {
    let resolve!: (value: ActivityPage) => void;
    const list = jest.spyOn(services.traderSync, 'listActivities').mockImplementation(
        () =>
            new Promise(done => {
                resolve = done;
            })
    );
    await mount();
    await act(async () => jest.advanceTimersByTime(15000));
    expect(list).toHaveBeenCalledTimes(1);
    await act(async () => resolve(page()));
    Object.defineProperty(document, 'visibilityState', {configurable: true, value: 'hidden'});
    const before = list.mock.calls.length;
    await act(async () => jest.advanceTimersByTime(10000));
    expect(list).toHaveBeenCalledTimes(before);
    Object.defineProperty(document, 'visibilityState', {configurable: true, value: 'visible'});
    await act(async () => document.dispatchEvent(new Event('visibilitychange')));
    expect(list).toHaveBeenCalledTimes(before + 1);
});

test('new subscription focus consumes once while keeping a different activity filter', async () => {
    const {saveActivitySession, saveNewSubscriptionFocus, readNewSubscriptionFocus} = await import('./state');
    saveActivitySession('A', {query: {subscriptionId: 'older', pageSize: 50}, current: page(), previous: [], scrollY: 0});
    saveNewSubscriptionFocus('A', sub.id);
    const list = jest.spyOn(services.traderSync, 'listActivities').mockResolvedValue(page());
    await mount();
    expect(readNewSubscriptionFocus('A')).toBeUndefined();
    expect(list.mock.calls[0][0]?.subscriptionId).toBe('older');
    expect(tree.root.findByProps({'aria-controls': 'trader-sync-target-list'}).props['aria-expanded']).toBe(true);
    act(() => tree.root.findByProps({'aria-controls': 'trader-sync-target-list'}).props.onClick());
    await act(async () => jest.advanceTimersByTime(5000));
    expect(tree.root.findByProps({'aria-controls': 'trader-sync-target-list'}).props['aria-expanded']).toBe(false);
});
test('metadata selection buffers only the selected row while notification and other row update', async () => {
    const {ActivityFeed} = await import('./activity-feed');
    const selected = document.createElement('article');
    const other = document.createElement('article');
    const selection = {isCollapsed: false, rangeCount: 1, getRangeAt: () => ({intersectsNode: (node: Node) => node === selected})} as unknown as Selection;
    jest.spyOn(window, 'getSelection').mockImplementation(() => selection);
    const onOpen = () => undefined;
    const initial = page().activities;
    const render = (activities: typeof initial) => (
        <MemoryRouter future={{v7_startTransition: true, v7_relativeSplatPath: true}}>
            <ActivityFeed activities={activities} onOpen={onOpen} />
        </MemoryRouter>
    );
    await act(async () => {
        tree = renderer.create(render(initial), {createNodeMock: element => (element.type === 'article' ? (element.props['data-activity-id'] === '12' ? selected : other) : null)});
    });
    const next = initial.map((item, index) => ({
        ...item,
        notificationMode: 'ordinary' as const,
        delivery: {id: 'd', status: 'sent' as const, reason: '', attemptCount: '1'},
        metadata: {
            ...item.metadata,
            market: {
                ...item.metadata.market,
                evidence: {...item.metadata.market.evidence, availability: 'available' as const},
                title: index === 0 ? 'Selected new title' : 'Other new title'
            }
        },
        finalityAnomaly: {reason: 'conflict', detectedAt: '2026-09-11T00:00:00Z', publishedBlockHash: 'hash'}
    }));
    await act(async () => tree.update(render(next)));
    const text = JSON.stringify(tree.toJSON());
    expect(text).not.toContain('Selected new title');
    expect(text).toContain('Other new title');
    expect(text).toContain('Telegram confirmed receipt');
    expect(text).toContain('Finality anomaly');
    (selection as any).isCollapsed = true;
    act(() => document.dispatchEvent(new Event('selectionchange')));
    expect(JSON.stringify(tree.toJSON())).toContain('Selected new title');
});

test('background reads keep navigation focusable and queue an explicit next action until the request finishes', async () => {
    const list = jest.spyOn(services.traderSync, 'listActivities').mockResolvedValue(page());
    await mount();
    let finish!: (value: ActivityPage) => void;
    list.mockImplementationOnce(
        () =>
            new Promise(resolve => {
                finish = resolve;
            })
    );
    await act(async () => jest.advanceTimersByTime(5000));
    expect(button('Next')!.props.disabled).toBe(false);
    await act(async () => button('Next')!.props.onClick());
    expect(list).toHaveBeenCalledTimes(2);
    list.mockResolvedValue(page(['10'], 'history-refresh'));
    await act(async () => finish(page()));
    await act(async () => jest.advanceTimersByTime(0));
    expect(list.mock.calls.at(-1)![0]?.cursor).toBe('signed-next-page');
});

test('new target focus waits until controls are enabled before consuming the intent', async () => {
    const {saveNewSubscriptionFocus, readNewSubscriptionFocus} = await import('./state');
    saveNewSubscriptionFocus('A', sub.id);
    let finish!: (value: ActivityPage) => void;
    jest.spyOn(services.traderSync, 'listActivities').mockImplementation(
        () =>
            new Promise(resolve => {
                finish = resolve;
            })
    );
    await mount();
    expect(readNewSubscriptionFocus('A')).toBe(sub.id);
    await act(async () => finish(page()));
    expect(readNewSubscriptionFocus('A')).toBeUndefined();
});

test('latest success clears the visited stack and focuses the heading; failure preserves both', async () => {
    const {readActivitySession} = await import('./state');
    const list = jest.spyOn(services.traderSync, 'listActivities').mockResolvedValue(page());
    const focus = jest.fn();
    await act(async () => {
        tree = renderer.create(
            <MemoryRouter future={{v7_startTransition: true, v7_relativeSplatPath: true}}>
                <TraderSyncHomePage ownerId='A' />
            </MemoryRouter>,
            {createNodeMock: element => (element.type === 'h2' && element.props.id === 'trader-sync-activity-title' ? {focus} : null)}
        );
    });
    const historical = page(['10'], 'historical-refresh');
    historical.page.hasNewer = true;
    list.mockResolvedValue(historical);
    await act(async () => button('Next')!.props.onClick());
    expect(readActivitySession('A')?.previous).toHaveLength(1);
    list.mockRejectedValue(new Error('temporarily offline'));
    await act(async () => button('New activity available')!.props.onClick());
    expect(readActivitySession('A')?.previous).toHaveLength(1);
    expect(focus).not.toHaveBeenCalled();
    list.mockResolvedValue({...page(['13']), page: {...page().page, snapshot: 'new-opaque-snapshot'}});
    await act(async () => button('New activity available')!.props.onClick());
    expect(readActivitySession('A')?.previous).toHaveLength(0);
    expect(focus).toHaveBeenCalledTimes(1);
    expect(list.mock.calls.at(-1)![0]).toMatchObject({cursor: undefined, refreshCursor: undefined});
});

test('changing page size requests a new snapshot without either cursor and clears page history', async () => {
    const {readActivitySession} = await import('./state');
    const list = jest.spyOn(services.traderSync, 'listActivities').mockResolvedValue(page());
    await mount();
    list.mockResolvedValue(page(['10'], 'historical-refresh'));
    await act(async () => button('Next')!.props.onClick());
    list.mockResolvedValue({...page(), page: {...page().page, snapshot: 'new-page-size-snapshot'}});
    await act(async () => tree.root.findByType('select').props.onChange({target: {value: '100'}}));
    expect(list.mock.calls.at(-1)![0]).toMatchObject({pageSize: 100, cursor: undefined, refreshCursor: undefined});
    expect(readActivitySession('A')?.previous).toHaveLength(0);
});
