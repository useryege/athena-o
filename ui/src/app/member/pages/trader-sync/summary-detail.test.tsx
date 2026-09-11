import renderer, {act} from 'react-test-renderer';
import {MemoryRouter, Route, Routes, createMemoryRouter, RouterProvider} from 'react-router-dom';
import {TraderSyncSummaryPage} from './summary-detail';
import {CursorNavigation} from './cursor-navigation';
import {clearTraderSyncState, readDetailSession} from './state';
import {ensureMemberBusinessServices, memberServices as services} from '../../services';
import {normalizeActivity, SummaryBatch, PartPage, ActivityPage} from '../../trader-sync-models';
import fixture from '../../testdata/trader-sync/05.json';
const activity = normalizeActivity(fixture.activity);
const batch: SummaryBatch = {
    id: '7',
    oldestAt: activity.recordedAt,
    settledFrom: activity.settledAt,
    settledTo: activity.settledAt,
    recordedFrom: activity.recordedAt,
    recordedTo: activity.recordedAt,
    activityCount: '1',
    targetCounts: [{wallet: activity.wallet, count: '1'}],
    partCounts: {total: '2', sent: '1', unknown: '1', pending: '0', sending: '0', failed: '0', cancelled: '0'},
    asOf: activity.recordedAt
};
const partPage = (id = '1', nextCursor: string | undefined = 'part-next'): PartPage => ({
    parts: [{id, index: Number(id), total: 2, delivery: {id, status: id === '1' ? 'sent' : 'unknown', reason: '', attemptCount: '1'}, associatedActivityCount: '1'}],
    page: {nextCursor},
    asOf: activity.recordedAt
});
const activityPage = (id = '12', refreshCursor = 'activity-refresh', nextCursor: string | undefined = 'activity-next'): ActivityPage => ({
    activities: [{...activity, id}],
    page: {snapshot: 'snapshot', refreshCursor, nextCursor, hasNewer: false}
});
let tree: renderer.ReactTestRenderer;
const text = () => JSON.stringify(tree.toJSON());
const nav = (label: string) => tree.root.findByProps({'aria-label': label});
const click = async (label: string, title: string) =>
    act(async () =>
        nav(label)
            .findAllByType('button')
            .find(n => n.props.children === title)!
            .props.onClick()
    );
const mount = async () =>
    act(async () => {
        tree = renderer.create(
            <MemoryRouter initialEntries={['/trader-sync/summaries/7']} future={{v7_startTransition: true, v7_relativeSplatPath: true}}>
                <Routes>
                    <Route path='/trader-sync/summaries/:batchId' element={<TraderSyncSummaryPage ownerId='A' />} />
                </Routes>
            </MemoryRouter>
        );
    });
beforeEach(() => {
    ensureMemberBusinessServices();
    jest.useFakeTimers();
    window.matchMedia = jest.fn().mockImplementation(query => ({matches: false, media: query, addListener: jest.fn(), removeListener: jest.fn()}));
    jest.spyOn(window, 'scrollTo').mockImplementation(() => undefined);
    jest.spyOn(services.traderSync, 'getSummaryBatch').mockResolvedValue(batch);
    jest.spyOn(services.traderSync, 'listSummaryParts').mockResolvedValue(partPage());
    jest.spyOn(services.traderSync, 'listActivities').mockResolvedValue(activityPage());
});
afterEach(() => {
    if (tree) act(() => tree.unmount());
    clearTraderSyncState();
    jest.restoreAllMocks();
    jest.useRealTimers();
});
test('batch totals distinguish activities from parts and never call mixed results complete', async () => {
    await mount();
    expect(text()).toContain('Activities in this batch');
    expect(text()).toContain('Telegram accepted');
    expect(text()).not.toContain('All parts accepted by Telegram');
    jest.mocked(services.traderSync.getSummaryBatch).mockResolvedValue({...batch, partCounts: {...batch.partCounts, sent: '2', unknown: '0'}});
    await act(async () => jest.advanceTimersByTime(5000));
    expect(text()).toContain('All parts accepted by Telegram');
    jest.mocked(services.traderSync.getSummaryBatch).mockResolvedValue({...batch, partCounts: {...batch.partCounts, total: '0', sent: '0', unknown: '0'}});
    await act(async () => jest.advanceTimersByTime(5000));
    expect(text()).not.toContain('All parts accepted by Telegram');
});
test('parts and activities navigate independently and refresh with their own original tokens', async () => {
    await mount();
    jest.mocked(services.traderSync.listSummaryParts).mockResolvedValue(partPage('2', undefined));
    await click('Summary part pages', 'Next');
    expect(text()).toContain('Delivery unknown');
    expect(services.traderSync.listSummaryParts).toHaveBeenLastCalledWith('7', expect.objectContaining({cursor: 'part-next'}));
    expect(tree.root.findAll(n => n.props['data-activity-id']).map(n => n.props['data-activity-id'])).toEqual(['12']);
    jest.mocked(services.traderSync.listActivities).mockResolvedValue(activityPage('11', 'second-refresh', undefined));
    await click('Summary activity pages', 'Next');
    expect(services.traderSync.listActivities).toHaveBeenLastCalledWith(expect.objectContaining({summaryBatchId: '7', cursor: 'activity-next'}));
    await act(async () => jest.advanceTimersByTime(5000));
    expect(services.traderSync.listActivities).toHaveBeenLastCalledWith({summaryBatchId: '7', pageSize: 50, refreshCursor: 'second-refresh'});
    const query = jest.mocked(services.traderSync.listActivities).mock.calls.at(-1)![0]!;
    expect(query.from).toBeUndefined();
    expect(query.to).toBeUndefined();
    jest.mocked(services.traderSync.listSummaryParts).mockResolvedValue(partPage());
    await click('Summary part pages', 'Previous');
    expect(tree.root.findAll(n => n.props['data-activity-id']).map(n => n.props['data-activity-id'])).toEqual(['11']);
});
test('return restores both pages and committed scroll and cleanup fences the session', async () => {
    await mount();
    jest.mocked(services.traderSync.listSummaryParts).mockResolvedValue(partPage('2', undefined));
    await click('Summary part pages', 'Next');
    Object.defineProperty(window, 'scrollY', {configurable: true, value: 900});
    act(() => window.dispatchEvent(new Event('scroll')));
    act(() => tree.unmount());
    await mount();
    expect(tree.root.findAll(n => n.props['data-part-id']).map(n => n.props['data-part-id'])).toEqual(['2']);
    expect(window.scrollTo).toHaveBeenLastCalledWith({top: 900, behavior: 'instant'});
    act(() => clearTraderSyncState());
    expect(readDetailSession('A', 'summary/7')).toBeUndefined();
    expect(text()).not.toContain(activity.wallet);
    Object.defineProperty(window, 'scrollY', {configurable: true, value: 0});
});
test('existing cursor navigation keeps its default accessible label', () => {
    act(() => {
        tree = renderer.create(<CursorNavigation canPrevious={false} onPrevious={() => undefined} onNext={() => undefined} />);
    });
    expect(nav('Activity pages')).toBeDefined();
});

test('navigation waits for an in-flight background part read without losing the request', async () => {
    await mount();
    let finish!: (page: PartPage) => void;
    jest.mocked(services.traderSync.listSummaryParts).mockImplementationOnce(
        () =>
            new Promise(resolve => {
                finish = resolve;
            })
    );
    await act(async () => jest.advanceTimersByTime(5000));
    await click('Summary part pages', 'Next');
    expect(services.traderSync.listSummaryParts).toHaveBeenCalledTimes(2);
    jest.mocked(services.traderSync.listSummaryParts).mockResolvedValue(partPage('2', undefined));
    await act(async () => finish(partPage()));
    await act(async () => jest.advanceTimersByTime(0));
    expect(tree.root.findAll(n => n.props['data-part-id']).map(n => n.props['data-part-id'])).toEqual(['2']);
});
test('a stale page keeps its fixed members and discloses the last update time', async () => {
    await mount();
    jest.mocked(services.traderSync.listSummaryParts).mockRejectedValue(new Error('offline'));
    await act(async () => jest.advanceTimersByTime(5000));
    expect(text()).toContain('Data may be out of date');
    expect(text()).toContain('Last updated');
    expect(tree.root.findAll(n => n.props['data-part-id']).map(n => n.props['data-part-id'])).toEqual(['1']);
});

test('Previous commits a long page before restoring scroll and later refresh does not repeat it', async () => {
    const longPage = {...partPage(), parts: Array.from({length: 50}, (_, i) => ({...partPage().parts[0], id: String(i + 1), index: i + 1, total: 51}))};
    jest.mocked(services.traderSync.listSummaryParts).mockResolvedValue(longPage);
    await mount();
    Object.defineProperty(window, 'scrollY', {configurable: true, value: 2400});
    const seen: Array<{top: number; count: number}> = [];
    jest.mocked(window.scrollTo).mockImplementation((options: number | ScrollToOptions) => {
        const top = typeof options === 'number' ? options : options.top || 0;
        const count = tree.root.findAll(n => n.props['data-part-id']).length;
        seen.push({top, count});
        Object.defineProperty(window, 'scrollY', {configurable: true, value: Math.min(top, count === 1 ? 100 : 3000)});
        window.dispatchEvent(new Event('scroll'));
    });
    try {
        jest.mocked(services.traderSync.listSummaryParts).mockResolvedValue({...partPage('51', undefined), parts: [{...partPage('51').parts[0], total: 51}]});
        await click('Summary part pages', 'Next');
        seen.length = 0;
        jest.mocked(services.traderSync.listSummaryParts).mockResolvedValue(longPage);
        await act(async () => {
            nav('Summary part pages').findAllByType('button')[0].props.onClick();
            Object.defineProperty(window, 'scrollY', {configurable: true, value: 100});
            window.dispatchEvent(new Event('scroll'));
        });
        expect(seen).toEqual([{top: 2400, count: 50}]);
        expect(readDetailSession('A', 'summary/7')?.scrollY).toBe(2400);
        await act(async () => jest.advanceTimersByTime(5000));
        expect(seen).toHaveLength(1);
    } finally {
        Object.defineProperty(window, 'scrollY', {configurable: true, value: 0});
    }
});
test('invalidation before a Previous commit discards restoration and all retained private pages', async () => {
    await mount();
    jest.mocked(services.traderSync.listSummaryParts).mockResolvedValue(partPage('2', undefined));
    await click('Summary part pages', 'Next');
    jest.mocked(window.scrollTo).mockClear();
    await act(async () => {
        nav('Summary part pages').findAllByType('button')[0].props.onClick();
        clearTraderSyncState();
    });
    expect(window.scrollTo).not.toHaveBeenCalled();
    expect(readDetailSession('A', 'summary/7')).toBeUndefined();
    expect(text()).not.toContain(activity.wallet);
});
test('snapshot violations retain the fixed activity page while the other lane keeps updating', async () => {
    await mount();
    jest.mocked(services.traderSync.listActivities).mockResolvedValue(activityPage('99'));
    jest.mocked(services.traderSync.listSummaryParts).mockResolvedValue({
        ...partPage(),
        parts: [{...partPage().parts[0], delivery: {...partPage().parts[0].delivery, status: 'failed', reason: 'rejected'}}]
    });
    await act(async () => jest.advanceTimersByTime(5000));
    expect(tree.root.findAll(n => n.props['data-activity-id']).map(n => n.props['data-activity-id'])).toEqual(['12']);
    expect(text()).toContain('snapshot');
    expect(text()).toContain('Delivery failed');
});
test('hidden pages stop every batch lane and resume once when visible', async () => {
    await mount();
    Object.defineProperty(document, 'visibilityState', {configurable: true, value: 'hidden'});
    const reads = [services.traderSync.getSummaryBatch, services.traderSync.listActivities, services.traderSync.listSummaryParts];
    const counts = reads.map(read => jest.mocked(read).mock.calls.length);
    await act(async () => jest.advanceTimersByTime(15000));
    expect(reads.map(read => jest.mocked(read).mock.calls.length)).toEqual(counts);
    Object.defineProperty(document, 'visibilityState', {configurable: true, value: 'visible'});
    await act(async () => document.dispatchEvent(new Event('visibilitychange')));
    expect(reads.map(read => jest.mocked(read).mock.calls.length)).toEqual(counts.map(count => count + 1));
});

test('failed first member reads show an error without pretending to load forever or an empty result', async () => {
    jest.mocked(services.traderSync.listSummaryParts).mockRejectedValue(new Error('parts unavailable'));
    jest.mocked(services.traderSync.listActivities).mockRejectedValue(new Error('activities unavailable'));
    await mount();
    expect(text()).toContain('parts unavailable');
    expect(text()).toContain('activities unavailable');
    expect(text()).not.toContain('Loading parts');
    expect(text()).not.toContain('Loading activities');
    expect(text()).not.toContain('No activities.');
});

test('summary-to-activity round trip restores both independent cursor pages', async () => {
    const {TraderSyncActivityPage} = await import('./activity-detail');
    jest.mocked(services.traderSync.listActivities).mockResolvedValue({
        ...activityPage(),
        activities: [
            {...activity, id: '12'},
            {...activity, id: '11'}
        ]
    });
    jest.spyOn(services.traderSync, 'getActivity').mockResolvedValue({...activity, id: '11'});
    await act(async () => {
        tree = renderer.create(
            <MemoryRouter initialEntries={['/trader-sync/summaries/7']} future={{v7_startTransition: true, v7_relativeSplatPath: true}}>
                <Routes>
                    <Route path='/trader-sync/summaries/:batchId' element={<TraderSyncSummaryPage ownerId='A' />} />
                    <Route path='/trader-sync/activities/:activityId' element={<TraderSyncActivityPage ownerId='A' />} />
                </Routes>
            </MemoryRouter>
        );
    });
    jest.mocked(services.traderSync.listSummaryParts).mockResolvedValue(partPage('2', undefined));
    await click('Summary part pages', 'Next');
    jest.mocked(services.traderSync.listActivities).mockResolvedValue(activityPage('11', 'second-refresh', undefined));
    await click('Summary activity pages', 'Next');
    const event = () => ({button: 0, defaultPrevented: false, preventDefault: jest.fn(), metaKey: false, altKey: false, ctrlKey: false, shiftKey: false});
    await act(async () =>
        tree.root
            .findAllByType('a')
            .find(a => a.props.href === '/trader-sync/activities/11')!
            .props.onClick(event())
    );
    expect(text()).toContain('Trade facts');
    expect(tree.root.findAllByType('a').find(link => link.props.children === 'Back to summary batch')?.props.href).toBe('/trader-sync/summaries/7');
    expect(readDetailSession('A', 'activity/12')).toBeUndefined();
    await act(async () =>
        tree.root
            .findAllByType('a')
            .find(a => a.props.children === 'Back to summary batch')!
            .props.onClick(event())
    );
    expect(tree.root.findAll(n => n.props['data-part-id']).map(n => n.props['data-part-id'])).toEqual(['2']);
    expect(readDetailSession('A', 'summary/7')?.parts.index).toBe(1);
    expect(readDetailSession('A', 'summary/7')?.activities.index).toBe(1);
    expect(services.traderSync.listActivities).toHaveBeenLastCalledWith({summaryBatchId: '7', pageSize: 50, refreshCursor: 'second-refresh'});
});

test('an invalid part cursor can restart its lane without discarding the activity snapshot', async () => {
    await mount();
    jest.mocked(services.traderSync.listSummaryParts).mockRejectedValue({response: {body: {code: 3, message: 'invalid cursor'}}});
    await click('Summary part pages', 'Next');
    const recover = tree.root.findAllByType('button').find(button => button.props.children === 'Return to first part page');
    expect(recover).toBeDefined();
    jest.mocked(services.traderSync.listSummaryParts).mockResolvedValue(partPage());
    await act(async () => recover!.props.onClick());
    expect(services.traderSync.listSummaryParts).toHaveBeenLastCalledWith('7', {pageSize: 50, cursor: undefined, activityId: undefined});
    expect(readDetailSession('A', 'summary/7')?.activities.pages[0].page.refreshCursor).toBe('activity-refresh');
    expect(tree.root.findAll(n => n.props['data-part-id']).map(n => n.props['data-part-id'])).toEqual(['1']);
});

test('a later home visit to the same activity returns home while history keeps its original summary source', async () => {
    const {TraderSyncActivityPage} = await import('./activity-detail');
    const {TraderSyncHomePage} = await import('./home');
    const {readActivitySession} = await import('./state');
    jest.spyOn(services.traderSync, 'getActivity').mockResolvedValue({...activity, id: '11'});
    jest.spyOn(services.traderSync, 'listSubscriptions').mockResolvedValue({subscriptions: [], page: {}, quota: {used: 0, limit: 10}, asOf: activity.recordedAt});
    jest.spyOn(services.memberNotifications, 'getTelegramSettings').mockResolvedValue({} as any);
    jest.mocked(services.traderSync.listActivities).mockImplementation(input => Promise.resolve(input?.summaryBatchId ? activityPage('11') : activityPage('11', 'home-refresh')));
    const router = createMemoryRouter(
        [
            {path: '/trader-sync/summaries/:batchId', element: <TraderSyncSummaryPage ownerId='A' />},
            {path: '/trader-sync/activities/:activityId', element: <TraderSyncActivityPage ownerId='A' />},
            {path: '/trader-sync', element: <TraderSyncHomePage ownerId='A' />}
        ],
        {initialEntries: ['/trader-sync/summaries/7'], future: {v7_relativeSplatPath: true}}
    );
    await act(async () => {
        tree = renderer.create(<RouterProvider router={router} future={{v7_startTransition: true}} />);
    });
    const follow = async (label: string) =>
        act(async () =>
            tree.root
                .findAllByType('a')
                .find(link => link.props.children === label)!
                .props.onClick({button: 0, defaultPrevented: false, preventDefault: jest.fn()})
        );
    await follow('View activity');
    const originalSummaryEntry = router.state.location.key;
    expect(tree.root.findAllByType('a').find(link => link.props.children === 'Back to summary batch')?.props.href).toBe('/trader-sync/summaries/7');
    await follow('Back to summary batch');
    await follow('Back to Trader Sync');
    // Set the filter through the real Home controls; no manual cache clearing.
    const dates = tree.root.findAllByType('input').filter(input => input.props.type === 'date');
    act(() => dates[0].props.onChange({target: {value: '2026-09-10'}}));
    await act(async () => tree.root.findByType('form').props.onSubmit({preventDefault: jest.fn()}));
    Object.defineProperty(window, 'scrollY', {configurable: true, value: 640});
    act(() => window.dispatchEvent(new Event('scroll')));
    await follow('View activity');
    expect(tree.root.findAllByType('a').find(link => link.props.children === 'Back to Trader Sync')?.props.href).toBe('/trader-sync');
    expect(tree.root.findAllByType('a').some(link => link.props.children === 'Back to summary batch')).toBe(false);
    await follow('Back to Trader Sync');
    expect(readActivitySession('A')?.query.from).toBe('2026-09-09T16:00:00.000Z');
    expect(readActivitySession('A')?.scrollY).toBe(640);
    expect(jest.mocked(services.traderSync.listActivities).mock.calls.at(-1)![0]?.refreshCursor).toBe('home-refresh');
    // Back to the Home-origin activity retains Home, then to the prior Summary-origin entry retains Summary.
    await act(async () => router.navigate(-1));
    expect(tree.root.findAllByType('a').some(link => link.props.children === 'Back to Trader Sync')).toBe(true);
    await act(async () => router.navigate(-3));
    expect(router.state.location.key).toBe(originalSummaryEntry);
    expect(tree.root.findAllByType('a').find(link => link.props.children === 'Back to summary batch')?.props.href).toBe('/trader-sync/summaries/7');
    Object.defineProperty(window, 'scrollY', {configurable: true, value: 0});
});

test('direct entry has no stale source and a different summary supplies only its current history entry', async () => {
    const {TraderSyncActivityPage} = await import('./activity-detail');
    jest.spyOn(services.traderSync, 'getActivity').mockResolvedValue({...activity, id: '11'});
    jest.mocked(services.traderSync.listActivities).mockResolvedValue(activityPage('11'));
    const router = createMemoryRouter(
        [
            {path: '/trader-sync/summaries/:batchId', element: <TraderSyncSummaryPage ownerId='A' />},
            {path: '/trader-sync/activities/:activityId', element: <TraderSyncActivityPage ownerId='A' />}
        ],
        {initialEntries: ['/trader-sync/summaries/7'], future: {v7_relativeSplatPath: true}}
    );
    await act(async () => {
        tree = renderer.create(<RouterProvider router={router} future={{v7_startTransition: true}} />);
    });
    const open = async () =>
        act(async () =>
            tree.root
                .findAllByType('a')
                .find(link => link.props.children === 'View activity')!
                .props.onClick({button: 0, defaultPrevented: false, preventDefault: jest.fn()})
        );
    await open();
    await act(async () => router.navigate('/trader-sync/activities/11'));
    expect(tree.root.findAllByType('a').find(link => link.props.children === 'Back to Trader Sync')?.props.href).toBe('/trader-sync');
    await act(async () => router.navigate('/trader-sync/summaries/8'));
    await open();
    expect(tree.root.findAllByType('a').find(link => link.props.children === 'Back to summary batch')?.props.href).toBe('/trader-sync/summaries/8');
    const summary8Entry = router.state.location.key;
    await act(async () => router.navigate(-3));
    expect(tree.root.findAllByType('a').find(link => link.props.children === 'Back to summary batch')?.props.href).toBe('/trader-sync/summaries/7');
    await act(async () => router.navigate(3));
    expect(router.state.location.key).toBe(summary8Entry);
    expect(tree.root.findAllByType('a').find(link => link.props.children === 'Back to summary batch')?.props.href).toBe('/trader-sync/summaries/8');
});
