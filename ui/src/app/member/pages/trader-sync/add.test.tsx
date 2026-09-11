import renderer, {act} from 'react-test-renderer';
import {Button, Input} from 'antd';
import {MemoryRouter, useLocation} from 'react-router-dom';
import {TraderSyncAddPage} from './add';
import {clearTraderSyncState, readAddDraft, saveAddDraft, readNewSubscriptionFocus} from './state';
import {ensureMemberBusinessServices, memberServices as services} from '../../services';
import {normalizeResolvedTarget, normalizeSubscription} from '../../trader-sync-models';
import fixture from '../../testdata/trader-sync/resolve-saved-note-existing.json';
import subscriptionFixture from '../../testdata/trader-sync/01.json';
import {Context} from '../../../shared/context';

const pending = <T,>() => {
    let resolve!: (value: T) => void, reject!: (reason: unknown) => void;
    const promise = Object.assign(
        new Promise<T>((yes, no) => {
            resolve = yes;
            reject = no;
        }),
        {abort: jest.fn()}
    );
    return {promise, resolve, reject};
};
const target = () => {
    const value = normalizeResolvedTarget(fixture.target);
    delete value.existingSubscription;
    delete value.savedNote;
    value.quota = {used: 0, limit: 10};
    value.expiresAt = new Date(Date.now() + 300_000).toISOString();
    return value;
};
const ctx = {
    notifications: {success: jest.fn(), error: jest.fn(), info: jest.fn(), warning: jest.fn()},
    modal: {confirm: jest.fn(), info: jest.fn(), error: jest.fn()},
    navigation: {goto: jest.fn(), replace: jest.fn()},
    baseHref: '/'
};
const Location = () => <output>{JSON.stringify(useLocation())}</output>;
let tree: renderer.ReactTestRenderer;
const mount = async (owner = 'A') => {
    await act(async () => {
        tree = renderer.create(
            <Context.Provider value={ctx}>
                <MemoryRouter future={{v7_startTransition: true, v7_relativeSplatPath: true}} initialEntries={['/trader-sync/add']}>
                    <TraderSyncAddPage ownerId={owner} />
                    <Location />
                </MemoryRouter>
            </Context.Provider>
        );
    });
};
const button = (label: string) => tree.root.findAllByType(Button).find(item => item.props.children === label)!;
const input = () => {
    const fields = tree.root.findAllByType(Input).filter(item => item.props.id === 'trader-sync-input');
    expect(fields).toHaveLength(1);
    return fields[0];
};
const note = () => tree.root.findAllByType(Input).find(item => item.props.id === 'trader-sync-note')!;
const resolveTarget = async (value = target()) => {
    jest.spyOn(services.traderSync, 'resolveTarget').mockReturnValue(Object.assign(Promise.resolve(value), {abort: jest.fn()}));
    await act(async () => {
        input().props.onChange({target: {value: value.wallet}});
    });
    await act(async () => {
        tree.root.findByType('form').props.onSubmit({preventDefault: jest.fn()});
    });
    return value;
};
beforeEach(() => {
    ensureMemberBusinessServices();
    jest.spyOn(services.memberNotifications, 'getTelegramSettings').mockResolvedValue({botAvailable: true, botUsername: 'bot'});
    window.matchMedia = jest.fn().mockImplementation(query => ({
        matches: false,
        media: query,
        addListener: jest.fn(),
        removeListener: jest.fn(),
        addEventListener: jest.fn(),
        removeEventListener: jest.fn()
    }));
    Object.defineProperty(globalThis.crypto, 'randomUUID', {configurable: true, value: jest.fn(() => 'request-one')});
});
afterEach(() => {
    if (tree) act(() => tree.unmount());
    clearTraderSyncState();
    jest.restoreAllMocks();
    jest.useRealTimers();
});

test('resolves to six snapshot periods with 1Y default and counts note by Unicode code point', async () => {
    await mount();
    await resolveTarget();
    const periods = tree.root.findAllByType(Button).filter(item => item.props['aria-pressed'] !== undefined);
    expect(periods.map(item => item.props.children)).toEqual(['1D', '1W', '1M', '1Y', 'YTD', 'ALL']);
    expect(button('1Y').props['aria-pressed']).toBe(true);
    await act(async () => {
        button('YTD').props.onClick();
    });
    expect(button('YTD').props['aria-pressed']).toBe(true);
    expect(services.traderSync.resolveTarget).toHaveBeenCalledTimes(1);
    await act(async () => {
        note().props.onChange({target: {value: '😀'.repeat(20)}});
    });
    expect(button('Confirm subscription').props.disabled).toBe(false);
    await act(async () => {
        note().props.onChange({target: {value: '😀'.repeat(21)}});
    });
    expect(button('Confirm subscription').props.disabled).toBe(true);
    expect(note().props.maxLength).toBeUndefined();
});
test('input change cancels old resolution and discards late card', async () => {
    await mount();
    const request = pending<ReturnType<typeof target>>();
    jest.spyOn(services.traderSync, 'resolveTarget').mockReturnValue(request.promise);
    await act(async () => {
        input().props.onChange({target: {value: 'first'}});
    });
    await act(async () => {
        tree.root.findByType('form').props.onSubmit({preventDefault: jest.fn()});
    });
    await act(async () => {
        input().props.onChange({target: {value: 'second'}});
    });
    expect(request.promise.abort).toHaveBeenCalled();
    await act(async () => {
        request.resolve(target());
    });
    expect(button('Confirm subscription')).toBeUndefined();
    expect(readAddDraft('A')?.target).toBeUndefined();
});
test('same owner returns with draft; another resolved wallet loads its own saved note', async () => {
    await mount();
    const first = await resolveTarget();
    await act(async () => {
        note().props.onChange({target: {value: 'my draft'}});
    });
    act(() => tree.unmount());
    await mount();
    expect(note().props.value).toBe('my draft');
    const second = {...first, wallet: '0x' + 'd'.repeat(40), savedNote: {wallet: '0x' + 'd'.repeat(40), note: 'second saved', revision: '1'}};
    await resolveTarget(second);
    expect(note().props.value).toBe('second saved');
    expect(readAddDraft('A')?.noteEdited).toBe(false);
    act(() => tree.unmount());
    await mount('B');
    expect(input().props.value).toBe('');
    expect(button('Confirm subscription')).toBeUndefined();
});
test('unknown Create retries original payload after expiry and prevents duplicate clicks', async () => {
    jest.useFakeTimers();
    await mount();
    const resolved = await resolveTarget();
    const request = pending<ReturnType<typeof normalizeSubscription>>();
    const create = jest
        .spyOn(services.traderSync, 'createSubscription')
        .mockReturnValueOnce(request.promise)
        .mockResolvedValue(normalizeSubscription(subscriptionFixture.subscription));
    await act(async () => {
        button('Confirm subscription').props.onClick();
        button('Confirm subscription').props.onClick();
    });
    expect(create).toHaveBeenCalledTimes(1);
    const original = readAddDraft('A')!.createRequest;
    expect(original).toEqual({confirmationToken: resolved.confirmationToken, requestId: 'request-one'});
    await act(async () => {
        request.reject({timeout: true});
    });
    await act(async () => {
        jest.advanceTimersByTime(301_000);
    });
    expect(button('Recover subscription result').props.disabled).toBe(false);
    await act(async () => {
        button('Recover subscription result').props.onClick();
    });
    expect(create.mock.calls[1][0]).toBe(original);
    expect(readNewSubscriptionFocus('A')).toBe(subscriptionFixture.subscription.id);
});
test('explicit server rejection keeps draft but does not offer unknown-result recovery', async () => {
    await mount();
    await resolveTarget();
    jest.spyOn(services.traderSync, 'createSubscription').mockRejectedValue({status: 403, response: {body: {code: 7, message: 'Permission denied'}}});
    await act(async () => {
        note().props.onChange({target: {value: 'keep me'}});
    });
    await act(async () => {
        button('Confirm subscription').props.onClick();
    });
    expect(button('Recover subscription result')).toBeUndefined();
    expect(readAddDraft('A')?.note).toBe('keep me');
    expect(JSON.stringify(tree.toJSON())).toContain('Permission denied');
});
test('unsent expired token disables confirmation, Notifications receives only fixed return path', async () => {
    jest.useFakeTimers();
    await mount();
    const resolved = await resolveTarget();
    await act(async () => {
        jest.advanceTimersByTime(301_000);
    });
    expect(button('Confirm subscription').props.disabled).toBe(true);
    await act(async () => {
        button('Open Notifications').props.onClick();
    });
    const location = JSON.parse(tree.root.findByType('output').props.children);
    expect(location.pathname).toBe('/notifications');
    expect(location.state).toEqual({returnTo: '/trader-sync/add'});
    expect(readAddDraft('A')?.target?.expiresAt).toBe(resolved.expiresAt);
});
test('scope invalidation aborts pending Create and fences its late success', async () => {
    await mount();
    await resolveTarget();
    const request = pending<ReturnType<typeof normalizeSubscription>>();
    jest.spyOn(services.traderSync, 'createSubscription').mockReturnValue(request.promise);
    await act(async () => {
        button('Confirm subscription').props.onClick();
    });
    act(clearTraderSyncState);
    expect(request.promise.abort).toHaveBeenCalled();
    await act(async () => {
        request.resolve(normalizeSubscription(subscriptionFixture.subscription));
    });
    expect(readNewSubscriptionFocus('A')).toBeUndefined();
    expect(readAddDraft('A')).toBeUndefined();
    expect(button('Confirm subscription')).toBeUndefined();
});

test('failure before a Create request is dispatched is not an unknown outcome', async () => {
    await mount();
    await resolveTarget();
    jest.spyOn(services.traderSync, 'createSubscription').mockImplementation(() => {
        throw new Error('Request could not be started');
    });
    await act(async () => {
        button('Confirm subscription').props.onClick();
    });
    expect(button('Recover subscription result')).toBeUndefined();
    expect(JSON.stringify(tree.toJSON())).toContain('Request could not be started');
});
test('returns to the saved Add scroll position without resolving again', async () => {
    const resolved = target();
    saveAddDraft({
        ownerId: 'A',
        input: resolved.wallet,
        target: resolved,
        wallet: resolved.wallet,
        note: 'desk',
        noteEdited: true,
        returnPath: '/trader-sync?subscriptionId=older',
        scrollY: 120
    });
    const scroll = jest.spyOn(window, 'scrollTo').mockImplementation(() => undefined);
    const service = jest.spyOn(services.traderSync, 'resolveTarget');
    await mount();
    expect(scroll).toHaveBeenCalledWith({top: 120, behavior: 'instant'});
    expect(service).not.toHaveBeenCalled();
});
test('new Resolve after unknown outcome obtains a new intent while retaining same-wallet note', async () => {
    await mount();
    await resolveTarget();
    await act(async () => {
        note().props.onChange({target: {value: 'edited'}});
    });
    const create = jest.spyOn(services.traderSync, 'createSubscription').mockRejectedValue({status: 503});
    await act(async () => {
        button('Confirm subscription').props.onClick();
    });
    const original = create.mock.calls[0][0];
    jest.mocked(services.traderSync.resolveTarget).mockResolvedValue({...target(), confirmationToken: 'fresh-token'});
    jest.mocked(crypto.randomUUID).mockReturnValue('request-two' as `${string}-${string}-${string}-${string}-${string}`);
    await act(async () => {
        tree.root.findByType('form').props.onSubmit({preventDefault: jest.fn()});
    });
    expect(note().props.value).toBe('edited');
    await act(async () => {
        button('Confirm subscription').props.onClick();
    });
    expect(create.mock.calls[1][0]).toEqual({confirmationToken: 'fresh-token', requestId: 'request-two', note: {value: 'edited'}});
    expect(create.mock.calls[1][0]).not.toEqual(original);
});
test('definite Resolve failure preserves input and never exposes a confirmation', async () => {
    await mount();
    jest.spyOn(services.traderSync, 'resolveTarget').mockRejectedValue({status: 400, response: {body: {code: 3, message: 'Unsupported profile URL'}}});
    await act(async () => {
        input().props.onChange({target: {value: 'bad URL'}});
    });
    await act(async () => {
        tree.root.findByType('form').props.onSubmit({preventDefault: jest.fn()});
    });
    expect(input().props.value).toBe('bad URL');
    expect(button('Confirm subscription')).toBeUndefined();
    expect(JSON.stringify(tree.toJSON())).toContain('Unsupported profile URL');
});
test('existing subscription and quota full each expose their concrete destination', async () => {
    await mount();
    const existing = target();
    existing.existingSubscription = {id: 'existing-id', status: 'healthy', revision: '1'};
    await resolveTarget(existing);
    expect(button('Confirm subscription')).toBeUndefined();
    expect(button('View subscription')).toBeDefined();
    await act(async () => {
        button('View subscription').props.onClick();
    });
    expect(JSON.parse(tree.root.findByType('output').props.children).pathname).toBe('/trader-sync/subscriptions/existing-id');
    await resolveTarget({...target(), quota: {used: 10, limit: 10}});
    expect(button('Confirm subscription')).toBeUndefined();
    expect(button('Manage subscriptions')).toBeDefined();
});
test('edited empty note is sent explicitly and successful return preserves original filter', async () => {
    const resolved = target();
    resolved.savedNote = {wallet: resolved.wallet, note: 'server-note', revision: '1'};
    saveAddDraft({
        ownerId: 'A',
        input: resolved.wallet,
        target: resolved,
        wallet: resolved.wallet,
        note: 'server-note',
        noteEdited: false,
        returnPath: '/trader-sync?subscriptionId=older',
        scrollY: 0
    });
    await mount();
    const create = jest.spyOn(services.traderSync, 'createSubscription').mockResolvedValue(normalizeSubscription(subscriptionFixture.subscription));
    await act(async () => {
        note().props.onChange({target: {value: ''}});
    });
    await act(async () => {
        button('Confirm subscription').props.onClick();
    });
    expect(create.mock.calls[0][0].note).toEqual({value: ''});
    expect(JSON.parse(tree.root.findByType('output').props.children).search).toBe('?subscriptionId=older');
});
test('unknown Create survives unmount and recovers original object after expired return', async () => {
    await mount();
    const resolved = await resolveTarget();
    const original = {confirmationToken: resolved.confirmationToken, requestId: 'original-id', note: {value: 'first payload'}};
    const saved = readAddDraft('A')!;
    act(() => tree.unmount());
    saveAddDraft({...saved, target: {...resolved, expiresAt: '2000-01-01T00:00:00Z'}, note: 'first payload', createRequest: original});
    const create = jest.spyOn(services.traderSync, 'createSubscription').mockRejectedValue({timeout: true});
    await mount();
    await act(async () => {
        button('Recover subscription result').props.onClick();
    });
    expect(create).toHaveBeenCalledWith(original);
    expect(create.mock.calls[0][0]).toBe(original);
});
test('old failure and finally cannot clear the newer resolve or its busy slot', async () => {
    await mount();
    const old = pending<ReturnType<typeof target>>(),
        fresh = pending<ReturnType<typeof target>>();
    const service = jest.spyOn(services.traderSync, 'resolveTarget').mockReturnValueOnce(old.promise).mockReturnValueOnce(fresh.promise);
    await act(async () => {
        input().props.onChange({target: {value: 'old'}});
    });
    await act(async () => {
        tree.root.findByType('form').props.onSubmit({preventDefault: jest.fn()});
    });
    await act(async () => {
        input().props.onChange({target: {value: 'fresh'}});
    });
    await act(async () => {
        tree.root.findByType('form').props.onSubmit({preventDefault: jest.fn()});
    });
    await act(async () => {
        old.reject(new Error('old failure'));
    });
    await act(async () => {
        tree.root.findByType('form').props.onSubmit({preventDefault: jest.fn()});
    });
    expect(service).toHaveBeenCalledTimes(2);
    await act(async () => {
        fresh.resolve(target());
    });
    expect(button('Confirm subscription')).toBeDefined();
    expect(JSON.stringify(tree.toJSON())).not.toContain('old failure');
});
test('unreadable expiry never permits a new confirmation', async () => {
    await mount();
    await resolveTarget({...target(), expiresAt: 'not-a-time'});
    expect(button('Confirm subscription').props.disabled).toBe(true);
});
test('scope invalidation aborts the auxiliary Telegram status read too', async () => {
    const read = pending<{botAvailable: boolean; botUsername: string}>();
    jest.mocked(services.memberNotifications.getTelegramSettings).mockReturnValue(read.promise);
    await mount();
    act(clearTraderSyncState);
    expect(read.promise.abort).toHaveBeenCalled();
    await act(async () => {
        read.resolve({botAvailable: true, botUsername: 'late'});
    });
    expect(readAddDraft('A')).toBeUndefined();
});
test('confirmation keeps complete wallet and exact raw values, and never invents missing verification', async () => {
    await mount();
    const resolved = target();
    resolved.verified = {evidence: resolved.verified.evidence};
    resolved.positionValue = {evidence: {...resolved.positionValue.evidence, availability: 'unavailable', reasonCode: 'missing_value'}, value: '123'};
    await resolveTarget(resolved);
    const wallet = tree.root.findAllByProps({className: 'trader-sync-confirmation__wallet'}).find(item => item.props.copyable);
    expect(wallet!.props.children).toBe(resolved.wallet);
    expect(wallet!.props.copyable).toEqual({text: resolved.wallet});
    const content = JSON.stringify(tree.toJSON());
    expect(content).toContain('9007199254740993');
    expect(content).toContain('$0');
    expect(content).toContain('missing_value');
    expect(content).not.toContain('Not verified');
    expect(tree.root.findAll(item => item.props.copyable?.text === '123')).toHaveLength(0);
});
