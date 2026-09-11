// ReactTestRenderer cannot host DOM portals; preserve the real modal with an inline container.
jest.mock('antd', () => {
    const actual = jest.requireActual('antd');
    return {...actual, Modal: (props: object) => require('react').createElement(actual.Modal, {...props, getContainer: false})};
});
import renderer, {act} from 'react-test-renderer';
import {Button, Input, Modal} from 'antd';
import {MemoryRouter, Route, Routes, useLocation} from 'react-router-dom';
import {TraderSyncSubscriptionPage} from './subscription-detail';
import {clearTraderSyncState, readAddDraft, readSubscriptionEdit} from './state';
import {ensureMemberBusinessServices, memberServices as services} from '../../services';
import {normalizeSubscription, Subscription, HistoryEntry} from '../../trader-sync-models';
import fixture from '../../testdata/trader-sync/01.json';
const sub = (): Subscription => ({...normalizeSubscription(fixture.subscription), status: 'paused', note: 'server', revision: '9007199254740993', noteRevision: '17'});
let tree: renderer.ReactTestRenderer;
const button = (label: string) => tree.root.findAllByType(Button).find(item => item.props.children === label)!;
const content = () => JSON.stringify(tree.toJSON());
const note = () => tree.root.findAllByType(Input).find(item => item.props.id === 'subscription-note')!;
const Location = () => <output>{useLocation().pathname}</output>;
const mount = async () => {
    await act(async () => {
        tree = renderer.create(
            <MemoryRouter future={{v7_startTransition: true, v7_relativeSplatPath: true}} initialEntries={[`/trader-sync/subscriptions/${sub().id}`]}>
                <Routes>
                    <Route path='/trader-sync/subscriptions/:subscriptionId' element={<TraderSyncSubscriptionPage ownerId='A' />} />
                    <Route path='*' element={<Location />} />
                </Routes>
            </MemoryRouter>
        );
    });
};
beforeEach(() => {
    ensureMemberBusinessServices();
    window.matchMedia = jest.fn().mockImplementation(query => ({matches: false, media: query, addListener: jest.fn(), removeListener: jest.fn()}));
    jest.spyOn(window, 'scrollTo').mockImplementation(() => undefined);
    Object.defineProperty(globalThis.crypto, 'randomUUID', {configurable: true, value: jest.fn(() => 'request-one')});
    jest.spyOn(services.traderSync, 'getSubscription').mockResolvedValue(sub());
    jest.spyOn(services.traderSync, 'listSubscriptionHistory').mockResolvedValue({entries: [], page: {}, asOf: sub().updatedAt});
});
afterEach(() => {
    if (tree) act(() => tree.unmount());
    clearTraderSyncState();
    jest.restoreAllMocks();
    jest.useRealTimers();
});
test('resume takes server Preparing fact and exact revision; summary poll never reloads history', async () => {
    jest.useFakeTimers();
    const resume = jest.spyOn(services.traderSync, 'resumeSubscription').mockResolvedValue({...sub(), status: 'pending_baseline', revision: '9007199254740994'});
    await mount();
    expect(button('Save note')).toBeDefined();
    expect(button('Resume')).toBeDefined();
    await act(async () => button('Resume').props.onClick());
    expect(resume).toHaveBeenCalledWith(sub().id, {expectedRevision: sub().revision, requestId: 'request-one'});
    expect(content()).toContain('Preparing monitoring');
    expect(button('Resume')).toBeUndefined();
    await act(async () => jest.advanceTimersByTime(5000));
    expect(services.traderSync.listSubscriptionHistory).toHaveBeenCalledTimes(1);
});
test('cancel has exactly one complete-identity confirmation and cancelled target resubscribes through fresh Add draft', async () => {
    jest.mocked(services.traderSync.getSubscription).mockResolvedValue({
        ...sub(),
        targetDisplay: {...sub().targetDisplay, displayName: {value: 'Known trader', evidence: {...sub().targetDisplay.displayName.evidence, availability: 'available'}}}
    });
    const cancel = jest.spyOn(services.traderSync, 'cancelSubscription').mockResolvedValue({...sub(), status: 'cancelled'});
    await mount();
    expect(button('Save note')).toBeDefined();
    await act(async () => button('Cancel subscription').props.onClick());
    expect(cancel).not.toHaveBeenCalled();
    const modal = tree.root.findByType(Modal);
    expect(modal.props.open).toBe(true);
    expect(modal.findAllByType('div').some(item => item.children.includes('Known trader'))).toBe(true);
    expect(content()).toContain(sub().wallet);
    await act(async () => modal.props.onOk());
    expect(cancel).toHaveBeenCalledTimes(1);
    expect(button('Resume')).toBeUndefined();
    expect(content()).toContain('retained');
    await act(async () => button('Subscribe again').props.onClick());
    expect(readAddDraft('A')).toMatchObject({input: sub().wallet, noteEdited: false, scrollY: 0});
    expect(readAddDraft('A')?.target).toBeUndefined();
    expect(content()).toContain('/trader-sync/add');
});
test('unknown mutation locks note and retries exact original payload after unmount', async () => {
    const resume = jest.spyOn(services.traderSync, 'resumeSubscription').mockRejectedValue({timeout: true});
    await mount();
    expect(button('Save note')).toBeDefined();
    await act(async () => button('Resume').props.onClick());
    const original = resume.mock.calls[0][1];
    expect(button('Save note').props.disabled).toBe(true);
    act(() => tree.unmount());
    await mount();
    expect(button('Save note')).toBeDefined();
    await act(async () => button('Recover request result').props.onClick());
    expect(resume.mock.calls[1][1]).toBe(original);
});
test('Aborted fetches latest without replay, note conflict keeps local value and next explicit save uses note revision', async () => {
    const save = jest
        .spyOn(services.traderSync, 'updateTargetNote')
        .mockRejectedValueOnce({status: 409, response: {body: {code: 10, message: 'note revision conflict'}}})
        .mockResolvedValue({wallet: sub().wallet, note: '', revision: '19'});
    await mount();
    expect(button('Save note')).toBeDefined();
    await act(async () => note().props.onChange({target: {value: ''}}));
    jest.mocked(services.traderSync.getSubscription).mockResolvedValue({...sub(), note: 'latest server', noteRevision: '18'});
    await act(async () => button('Save note').props.onClick());
    expect(note().props.value).toBe('');
    expect(content()).toContain('latest server');
    expect(save).toHaveBeenCalledTimes(1);
    await act(async () => button('Save note').props.onClick());
    expect(save.mock.calls[1][1]).toMatchObject({note: '', expectedRevision: '18'});
});
test('history reaches item 51, preserves server ties and cached previous page across automatic recovery', async () => {
    jest.useFakeTimers();
    const entries: HistoryEntry[] = Array.from({length: 51}, (_, i) => ({
        id: `record-${String(51 - i).padStart(2, '0')}`,
        kind: 'interruption',
        sortAt: '2026-09-11T00:00:00Z',
        interruption: {reason: `reason-${i}`, uncertainty: 'actual_start_unknown', possibleMissing: true}
    }));
    const history = jest
        .mocked(services.traderSync.listSubscriptionHistory)
        .mockImplementation((_id, input) =>
            Promise.resolve({entries: input?.cursor ? entries.slice(50) : entries.slice(0, 50), page: input?.cursor ? {} : {nextCursor: 'opaque-next'}, asOf: sub().updatedAt})
        );
    await mount();
    expect(button('Save note')).toBeDefined();
    expect(content().indexOf('reason-0')).toBeLessThan(content().indexOf('reason-1'));
    expect(
        tree.root
            .findAll(item => item.type === 'li' && item.props['data-history-id'])
            .slice(0, 2)
            .map(item => item.props['data-history-id'])
    ).toEqual(['record-51', 'record-50']);
    await act(async () => button('More history').props.onClick());
    expect(content()).toContain('reason-50');
    await act(async () => button('Previous history').props.onClick());
    expect(content()).toContain('reason-0');
    jest.mocked(services.traderSync.getSubscription).mockResolvedValue({...sub(), status: 'healthy'});
    await act(async () => jest.advanceTimersByTime(5000));
    expect(button('Pause')).toBeDefined();
    expect(content()).toContain('reason-0');
    expect(history).toHaveBeenCalledTimes(2);
    expect(content()).toContain('Unknown');
    expect(content()).not.toContain('51 missed');
});

test('failed conflict refresh blocks another write until the latest revision is explicitly fetched', async () => {
    const save = jest.spyOn(services.traderSync, 'updateTargetNote').mockRejectedValue({status: 409, response: {body: {code: 10, message: 'note revision conflict'}}});
    await mount();
    await act(async () => note().props.onChange({target: {value: 'draft'}}));
    jest.mocked(services.traderSync.getSubscription).mockRejectedValueOnce({status: 503});
    await act(async () => button('Save note').props.onClick());
    expect(button('Save note').props.disabled).toBe(true);
    expect(button('Refresh latest state')).toBeDefined();
    jest.mocked(services.traderSync.getSubscription).mockResolvedValue({...sub(), note: 'latest', noteRevision: '20'});
    await act(async () => button('Refresh latest state').props.onClick());
    expect(note().props.value).toBe('draft');
    expect(button('Save note').props.disabled).toBe(false);
    expect(save).toHaveBeenCalledTimes(1);
});
test('a saved note does not pin its old revision after later server changes', async () => {
    jest.useFakeTimers();
    const save = jest.spyOn(services.traderSync, 'updateTargetNote').mockResolvedValue({wallet: sub().wallet, note: 'first', revision: '18'});
    await mount();
    await act(async () => note().props.onChange({target: {value: 'first'}}));
    await act(async () => button('Save note').props.onClick());
    jest.mocked(services.traderSync.getSubscription).mockResolvedValue({...sub(), note: 'another editor', noteRevision: '20'});
    await act(async () => jest.advanceTimersByTime(5000));
    expect(note().props.value).toBe('another editor');
    await act(async () => note().props.onChange({target: {value: 'second'}}));
    await act(async () => button('Save note').props.onClick());
    expect(save.mock.calls[1][1].expectedRevision).toBe('20');
});
test('note response preserves lifecycle changes learned while the request was in flight', async () => {
    jest.useFakeTimers();
    let resolve!: (value: {wallet: string; note: string; revision: string}) => void;
    jest.spyOn(services.traderSync, 'updateTargetNote').mockReturnValue(
        new Promise(done => {
            resolve = done;
        })
    );
    await mount();
    await act(async () => note().props.onChange({target: {value: 'draft'}}));
    await act(async () => button('Save note').props.onClick());
    jest.mocked(services.traderSync.getSubscription).mockResolvedValue({...sub(), status: 'cancelled', revision: '9007199254740994'});
    await act(async () => jest.advanceTimersByTime(5000));
    expect(button('Subscribe again')).toBeDefined();
    await act(async () => resolve({wallet: sub().wallet, note: 'draft', revision: '18'}));
    expect(button('Subscribe again')).toBeDefined();
    expect(button('Resume')).toBeUndefined();
});
test('scope invalidation aborts pending writes and late success cannot repopulate private state', async () => {
    let resolve!: (value: Subscription) => void;
    const pending = Object.assign(
        new Promise<Subscription>(done => {
            resolve = done;
        }),
        {abort: jest.fn()}
    );
    jest.spyOn(services.traderSync, 'resumeSubscription').mockReturnValue(pending);
    await mount();
    await act(async () => button('Resume').props.onClick());
    act(clearTraderSyncState);
    expect(pending.abort).toHaveBeenCalledTimes(1);
    await act(async () => resolve({...sub(), status: 'healthy'}));
    expect(content()).not.toContain(sub().wallet);
    expect(readSubscriptionEdit('A', sub().id).intent).toBeUndefined();
});
test('code 10 discards lifecycle intent, while code 6 HTTP409 is a definite failure and never auto-replays', async () => {
    const resume = jest
        .spyOn(services.traderSync, 'resumeSubscription')
        .mockRejectedValueOnce({status: 409, response: {body: {code: 10, message: 'subscription revision conflict'}}})
        .mockRejectedValue({status: 409, response: {body: {code: 6, message: 'request ID already used with another payload'}}});
    await mount();
    jest.mocked(services.traderSync.getSubscription).mockResolvedValue({...sub(), revision: '9007199254740994'});
    await act(async () => button('Resume').props.onClick());
    expect(resume).toHaveBeenCalledTimes(1);
    expect(button('Recover request result')).toBeUndefined();
    await act(async () => button('Resume').props.onClick());
    expect(resume.mock.calls[1][1].expectedRevision).toBe('9007199254740994');
    expect(button('Recover request result')).toBeUndefined();
    expect(content()).toContain('request ID already used with another payload');
});

test('failed history next page keeps a route back to cached history and latest records', async () => {
    jest.mocked(services.traderSync.listSubscriptionHistory)
        .mockResolvedValueOnce({entries: [], page: {nextCursor: 'expired-cursor'}, asOf: sub().updatedAt})
        .mockRejectedValue({status: 400, response: {body: {code: 3, message: 'Invalid cursor'}}});
    await mount();
    await act(async () => button('More history').props.onClick());
    expect(button('Previous history')).toBeDefined();
    expect(button('Latest history')).toBeDefined();
    expect(button('Previous history').props.disabled).toBe(false);
    await act(async () => button('Previous history').props.onClick());
    expect(content()).toContain('No observation history recorded yet.');
});
test('missing subscription renders an explicit NotFound fact with no history request', async () => {
    jest.mocked(services.traderSync.getSubscription).mockRejectedValue({status: 404, response: {body: {code: 5, message: 'subscription not found'}}});
    await mount();
    expect(content()).toContain('Subscription not found');
    expect(content()).not.toContain('[object Object]');
    expect(services.traderSync.listSubscriptionHistory).not.toHaveBeenCalled();
});
