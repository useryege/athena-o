import {moduleAccessService} from './shared/module-access-service';
import {
    captureTraderSyncScope,
    clearTraderSyncState,
    readAddDraft,
    saveAddDraft,
    saveNewSubscriptionFocus,
    readNewSubscriptionFocus,
    readActivitySession
} from './member/pages/trader-sync/state';
import renderer, {act} from 'react-test-renderer';
import {RouterProvider} from 'react-router-dom';
import {Button, Form, Input, Menu} from 'antd';
import {selfAccountServices} from './session/services';
import {MemberApp as App, loadAppBootstrapWithRetry} from './member/app';
import {parseUserInfo, AppBootstrap, AppBootstrapSessionStatus, AuthSettings} from './shared/models';
import {ensureMemberBusinessServices, memberServices as services} from './member/services';

const authSettings: AuthSettings = {
    url: '',
    statusBadgeEnabled: false,
    statusBadgeRootUrl: '',
    googleAnalytics: {
        trackingID: '',
        anonymizeUsers: false
    },
    help: {
        chatUrl: '',
        chatText: '',
        binaryUrls: {}
    },
    userLoginsDisabled: false,
    kustomizeVersions: [],
    uiCssURL: '',
    uiBannerContent: '',
    uiBannerURL: '',
    uiBannerPermanent: false,
    uiBannerPosition: '',
    execEnabled: false,
    appsInAnyNamespaceEnabled: false,
    hydratorEnabled: false,
    syncWithReplaceAllowed: false
};

const authenticatedBootstrap: AppBootstrap = {
    settings: authSettings,
    session: {
        status: AppBootstrapSessionStatus.Authenticated,
        userInfo: parseUserInfo({
            loggedIn: true,
            username: 'admin',
            iss: 'athena',
            administrator: true,
            accountId: 'admin-account'
        })
    }
};
const anonymousBootstrap: AppBootstrap = {settings: authSettings, session: {status: AppBootstrapSessionStatus.Anonymous}};

beforeAll(() => {
    if (typeof globalThis.MessageChannel === 'undefined') {
        class TestMessageChannel {
            public port1 = {onmessage: null as ((event: unknown) => void) | null};
            public port2 = {
                postMessage: () => {
                    window.setTimeout(() => this.port1.onmessage?.({}), 0);
                }
            };
        }
        (globalThis as any).MessageChannel = TestMessageChannel;
        (window as any).MessageChannel = TestMessageChannel;
    }
});

beforeEach(() => {
    jest.spyOn(moduleAccessService, 'states').mockResolvedValue(
        ['trader_sync', 'solana', 'market_radar', 'managed_oo', 'profit_sharing', 'worm'].map(module_key => ({module_key, state: 1})) as any
    );
    localStorage.clear();
    window.history.replaceState(null, '', '/');
    window.matchMedia =
        window.matchMedia ||
        jest.fn().mockImplementation(query => ({
            matches: false,
            media: query,
            onchange: null,
            addListener: jest.fn(),
            removeListener: jest.fn(),
            addEventListener: jest.fn(),
            removeEventListener: jest.fn(),
            dispatchEvent: jest.fn()
        }));
});

afterEach(() => {
    jest.restoreAllMocks();
});

const containsText = (node: renderer.ReactTestRendererJSON | (renderer.ReactTestRendererJSON | string)[] | string | null, text: string): boolean => {
    if (!node) {
        return false;
    }
    if (typeof node === 'string') {
        return node.includes(text);
    }
    if (Array.isArray(node)) {
        return node.some(child => containsText(child, text));
    }
    return containsText(node.children as renderer.ReactTestRendererJSON[] | string[] | null, text);
};

test('loadAppBootstrapWithRetry retries transient bootstrap failures', async () => {
    const load = jest.fn<Promise<AppBootstrap>, []>().mockRejectedValueOnce(new Error('backend is not ready')).mockResolvedValue(authenticatedBootstrap);
    const sleep = jest.fn<Promise<void>, [number]>(() => Promise.resolve());

    await expect(loadAppBootstrapWithRetry(load, [500], sleep)).resolves.toEqual(authenticatedBootstrap);

    expect(load).toHaveBeenCalledTimes(2);
    expect(sleep).toHaveBeenCalledWith(500);
});

test('loadAppBootstrapWithRetry returns the final bootstrap error', async () => {
    const finalError = new Error('still unavailable');
    const load = jest.fn<Promise<AppBootstrap>, []>().mockRejectedValue(finalError);

    await expect(loadAppBootstrapWithRetry(load, [500, 1000], () => Promise.resolve())).rejects.toThrow('still unavailable');
    expect(load).toHaveBeenCalledTimes(3);
});

test('Bootstrap renders recoverable bootstrap failure and retries on demand', async () => {
    jest.useFakeTimers();
    const bootstrap = jest
        .spyOn(services.authService, 'bootstrap')
        .mockRejectedValueOnce(new Error('api offline'))
        .mockRejectedValueOnce(new Error('api offline'))
        .mockRejectedValueOnce(new Error('api offline'))
        .mockRejectedValueOnce(new Error('api offline'))
        .mockRejectedValueOnce(new Error('api offline'))
        .mockResolvedValue(authenticatedBootstrap);

    let tree: renderer.ReactTestRenderer;
    await act(async () => {
        tree = renderer.create(<App />);
    });
    await act(async () => {
        await jest.advanceTimersByTimeAsync(6500);
    });

    expect(containsText(tree.toJSON(), 'API 服务暂不可用')).toBe(true);

    await act(async () => {
        tree.root.findAllByType(Button)[0].props.onClick();
    });

    expect(bootstrap).toHaveBeenCalledTimes(6);
    bootstrap.mockRestore();
    jest.useRealTimers();
});

test('Bootstrap redirects logged-out protected routes to login', async () => {
    jest.spyOn(services.authService, 'bootstrap').mockResolvedValue(anonymousBootstrap);
    window.history.replaceState(null, '', '/projects');

    let tree: renderer.ReactTestRenderer;
    await act(async () => {
        tree = renderer.create(<App />);
    });
    await act(async () => {
        await Promise.resolve();
        await Promise.resolve();
    });

    expect(window.location.pathname).toBe('/login');
    const methods = tree.root.findByProps({'role': 'group', 'aria-label': 'Sign-in methods'});
    const buttons = methods.findAllByType(Button);
    expect(buttons).toHaveLength(2);
    expect(buttons.every(button => typeof button.props.onClick === 'function' && !button.props.disabled)).toBe(true);
});

// Real MemberApp/Shell/router; transport is replaced, identity lifecycle is not.
describe('Trader Sync real Shell cleanup', () => {
    let tree: renderer.ReactTestRenderer;
    const member = (accountId = 'owner-A', iss = 'athena', level = 'read_write', revision = 1) =>
        parseUserInfo({loggedIn: true, accountId, username: accountId, iss, access: {loginEnabled: true, revision, moduleAccess: [{module: 'trader_sync', dataAccess: level}]}});
    const mountMember = async (level = 'read_write', path = '/trader-sync/add') => {
        ensureMemberBusinessServices();
        jest.spyOn(services.authService, 'bootstrap').mockResolvedValue({
            settings: authSettings,
            session: {status: AppBootstrapSessionStatus.Authenticated, userInfo: member('owner-A', 'athena', level)}
        });
        jest.spyOn(services.memberNotifications, 'getTelegramSettings').mockResolvedValue({botAvailable: true, botUsername: 'bot'});
        jest.spyOn(services.users, 'get').mockResolvedValue(member());
        window.history.replaceState(null, '', path);
        await act(async () => {
            tree = renderer.create(<App />);
        });
        await act(async () => {
            await Promise.resolve();
        });
    };
    test('module closure retains the authorized menu and URL but clears the form and persisted draft', async () => {
        jest.useFakeTimers();
        await mountMember();
        const input = tree.root.findAllByType(Input).find(item => item.props.id === 'trader-sync-input')!;
        await act(async () => input.props.onChange({target: {value: 'private-wallet'}}));
        expect(readAddDraft('owner-A')?.input).toBe('private-wallet');
        jest.mocked(moduleAccessService.states).mockResolvedValue(
            ['trader_sync', 'solana', 'market_radar', 'managed_oo', 'profit_sharing', 'worm'].map(module_key => ({module_key, state: 2})) as any
        );
        await act(async () => jest.advanceTimersByTimeAsync(2000));
        expect(JSON.stringify(tree.toJSON())).toContain('This module is not open yet');
        expect(tree.root.findAllByType(Input).some(item => item.props.id === 'trader-sync-input')).toBe(false);
        expect(readAddDraft('owner-A')).toBeUndefined();
        expect(window.location.pathname).toBe('/trader-sync/add');
        expect(tree.root.findAllByType(Menu).some(menu => JSON.stringify(menu.props.items).includes('"key":"/trader-sync"'))).toBe(true);
        jest.useRealTimers();
    });
    const seed = () => {
        const scope = captureTraderSyncScope('owner-A');
        saveAddDraft({ownerId: 'owner-A', input: 'private-wallet', note: 'private-note', noteEdited: true, returnPath: '/trader-sync', scrollY: 0});
        saveNewSubscriptionFocus('owner-A', 'new-subscription');
        return scope;
    };
    beforeEach(() => {
        Object.defineProperty(document, 'visibilityState', {configurable: true, value: 'visible'});
    });
    afterEach(() => {
        if (tree) act(() => tree.unmount());
        clearTraderSyncState();
        sessionStorage.clear();
    });
    test.each([
        ['owner-B', 'athena'],
        ['owner-A', 'new-issuer']
    ])('identity refresh %s/%s clears before publishing replacement', async (owner, issuer) => {
        await mountMember();
        const scope = seed();
        const seen: boolean[] = [];
        const unsubscribe = scope.subscribeInvalidation!(() => seen.push(readAddDraft('owner-A') === undefined));
        jest.mocked(services.users.get).mockResolvedValue(member(owner, issuer));
        await act(async () => {
            window.dispatchEvent(new Event('focus'));
        });
        expect(scope.isCurrent()).toBe(false);
        expect(seen).toEqual([true]);
        expect(readNewSubscriptionFocus('owner-A')).toBeUndefined();
        const resolveButton = tree.root.findAllByType(Button).find(item => item.props.children === 'Resolve trader');
        expect(resolveButton).toBeDefined();
        const addressInput = tree.root.findAllByType(Input).find(item => item.props.id === 'trader-sync-input')!;
        await act(async () => {
            addressInput.props.onChange({target: {value: 'new input'}});
        });
        expect(tree.root.findAllByType(Button).find(item => item.props.children === 'Resolve trader')!.props.disabled).toBe(false);
        unsubscribe();
    });
    test('RW loss clears scope and redirects protected Add route', async () => {
        await mountMember();
        const scope = seed();
        jest.mocked(services.users.get).mockResolvedValue(member('owner-A', 'athena', 'none', 2));
        await act(async () => {
            window.dispatchEvent(new Event('focus'));
        });
        expect(scope.isCurrent()).toBe(false);
        expect(readAddDraft('owner-A')).toBeUndefined();
        expect(window.location.pathname).toBe('/account/access');
    });
    test('endSession after logged-out refresh clears same-owner memory', async () => {
        await mountMember();
        const scope = seed();
        jest.mocked(services.users.get).mockResolvedValue({...member(), loggedIn: false});
        await act(async () => {
            window.dispatchEvent(new Event('focus'));
        });
        expect(scope.isCurrent()).toBe(false);
        expect(readAddDraft('owner-A')).toBeUndefined();
        expect(readNewSubscriptionFocus('owner-A')).toBeUndefined();
        expect(window.location.pathname).toBe('/login');
    });
    test.each(['none', 'read'])('initial %s grant cannot enter Add', async level => {
        await mountMember(level);
        expect(window.location.pathname).toBe('/account/access');
        expect(containsText(tree.toJSON(), 'Wallet address or Polymarket profile URL')).toBe(false);
    });
    test.each(['/trader-sync/subscriptions', '/trader-sync/subscriptions/sub-id'])('subscription deep link %s denies missing grant without a business request', async path => {
        ensureMemberBusinessServices();
        const list = jest.spyOn(services.traderSync, 'listSubscriptions');
        const detail = jest.spyOn(services.traderSync, 'getSubscription');
        const history = jest.spyOn(services.traderSync, 'listSubscriptionHistory');
        await mountMember('none', path);
        expect(window.location.pathname).toBe('/account/access');
        expect(list).not.toHaveBeenCalled();
        expect(detail).not.toHaveBeenCalled();
        expect(history).not.toHaveBeenCalled();
    });
    test('subscription list has a real Shell title and breadcrumb', async () => {
        const scroll = jest.spyOn(window, 'scrollTo').mockImplementation(() => undefined);
        ensureMemberBusinessServices();
        jest.spyOn(services.traderSync, 'listSubscriptions').mockResolvedValue({subscriptions: [], page: {}, quota: {used: 0, limit: 10}, asOf: '2026-09-11T00:00:00Z'});
        await mountMember('read_write', '/trader-sync/subscriptions');
        expect(document.title).toBe('Subscriptions · Athena');
        expect(scroll).toHaveBeenCalledWith({top: 0, behavior: 'instant'});
        expect(containsText(tree.toJSON(), 'Trader Sync')).toBe(true);
    });
    test('member identity control belongs to the top bar and the desktop navigation uses the approved width', async () => {
        ensureMemberBusinessServices();
        jest.spyOn(services.traderSync, 'listSubscriptions').mockResolvedValue({subscriptions: [], page: {}, quota: {used: 0, limit: 10}, asOf: '2026-09-11T00:00:00Z'});
        await mountMember('read_write', '/trader-sync/subscriptions');
        const header = tree.root.findByProps({className: 'athena-shell__header'});
        const sider = tree.root.findByProps({className: 'athena-shell__sider'});
        expect(header.findAllByProps({'aria-label': 'Open account menu'})).toHaveLength(1);
        expect(sider.findAllByProps({'aria-label': 'Open account menu'})).toHaveLength(0);
        expect(sider.props.width).toBe(224);
    });
    test('removed member Appearance URL uses the existing not-found fallback', async () => {
        await mountMember('read_write', '/account/appearance');
        expect(containsText(tree.toJSON(), 'Page not found')).toBe(true);
    });
    test.each(['none', 'read'])('home entry excludes %s and rejects the home deep link', async level => {
        await mountMember(level, '/trader-sync');
        expect(window.location.pathname).toBe('/account/access');
        expect(tree.root.findAllByType(Menu).some(menu => JSON.stringify(menu.props.items).includes('"key":"/trader-sync"'))).toBe(false);
    });
    test('Solana navigation hides and its deep link requires read access', async () => {
        ensureMemberBusinessServices();
        const list = jest.spyOn(services.solana, 'listProjects');
        const status = jest.spyOn(services.solana, 'getDiscoveryStatus');
        await mountMember('none', '/solana');
        expect(window.location.pathname).toBe('/account/access');
        expect(tree.root.findAllByType(Menu).some(menu => JSON.stringify(menu.props.items).includes('"key":"/solana"'))).toBe(false);
        expect(list).not.toHaveBeenCalled();
        expect(status).not.toHaveBeenCalled();
    });
    test('RW home has its actual page, navigation and title', async () => {
        ensureMemberBusinessServices();
        jest.spyOn(services.traderSync, 'listSubscriptions').mockResolvedValue({subscriptions: [], page: {}, quota: {used: 0, limit: 10}, asOf: '2026-09-11T00:00:00Z'});
        jest.spyOn(services.traderSync, 'listActivities').mockResolvedValue({activities: [], page: {snapshot: 'opaque', refreshCursor: 'signed', hasNewer: false}});
        await mountMember('read_write', '/trader-sync');
        expect(document.title).toBe('Trader Sync · Athena');
        expect(containsText(tree.toJSON(), 'Activity Alerts')).toBe(true);
        expect(tree.root.findAllByType(Menu).some(menu => JSON.stringify(menu.props.items).includes('"key":"/trader-sync"'))).toBe(true);
    });
    test('home loses RW, clears its saved page and ignores the pending activity response', async () => {
        ensureMemberBusinessServices();
        jest.spyOn(services.traderSync, 'listSubscriptions').mockResolvedValue({subscriptions: [], page: {}, quota: {used: 0, limit: 10}, asOf: '2026-09-11T00:00:00Z'});
        const {normalizeActivity} = await import('./member/trader-sync-models');
        const fixture = await import('./member/testdata/trader-sync/05.json');
        const page = {
            activities: [{...normalizeActivity(fixture.activity), noteSnapshot: 'private-home-note'}],
            page: {snapshot: 'opaque', refreshCursor: 'signed', hasNewer: false}
        };
        const list = jest.spyOn(services.traderSync, 'listActivities').mockResolvedValue(page);
        await mountMember('read_write', '/trader-sync');
        expect(containsText(tree.toJSON(), 'private-home-note')).toBe(true);
        let finish!: (value: typeof page) => void;
        list.mockImplementation(
            () =>
                new Promise(resolve => {
                    finish = resolve;
                })
        );
        jest.mocked(services.users.get).mockResolvedValue(member('owner-A', 'athena', 'none', 2));
        await act(async () => window.dispatchEvent(new Event('focus')));
        expect(window.location.pathname).toBe('/account/access');
        expect(readActivitySession('owner-A')).toBeUndefined();
        await act(async () => finish(page));
        expect(readActivitySession('owner-A')).toBeUndefined();
        expect(containsText(tree.toJSON(), 'private-home-note')).toBe(false);
    });
    test('home restores its page after a real subscription detail route round trip', async () => {
        ensureMemberBusinessServices();
        jest.spyOn(window, 'scrollTo').mockImplementation(() => undefined);
        const {normalizeSubscription} = await import('./member/trader-sync-models');
        const fixture = await import('./member/testdata/trader-sync/01.json');
        const subscription = normalizeSubscription(fixture.subscription);
        jest.spyOn(services.traderSync, 'listSubscriptions').mockResolvedValue({
            subscriptions: [subscription],
            page: {},
            quota: {used: 1, limit: 10},
            asOf: subscription.updatedAt
        });
        jest.spyOn(services.traderSync, 'getSubscription').mockResolvedValue(subscription);
        jest.spyOn(services.traderSync, 'listSubscriptionHistory').mockResolvedValue({entries: [], page: {}, asOf: subscription.updatedAt});
        const list = jest
            .spyOn(services.traderSync, 'listActivities')
            .mockResolvedValue({activities: [], page: {snapshot: 'opaque', refreshCursor: 'signed-return', hasNewer: false}});
        await mountMember('read_write', '/trader-sync');
        Object.defineProperty(window, 'scrollY', {configurable: true, value: 240});
        await act(async () => {
            await tree.root.findByType(RouterProvider).props.router.navigate(`/trader-sync/subscriptions/${subscription.id}`);
        });
        expect(document.title).toBe('Subscription · Athena');
        await act(async () => {
            await tree.root.findByType(RouterProvider).props.router.navigate('/trader-sync');
        });
        expect(list.mock.calls.at(-1)![0]?.refreshCursor).toBe('signed-return');
        expect(window.scrollTo).toHaveBeenLastCalledWith({top: 240, behavior: 'instant'});
        Object.defineProperty(window, 'scrollY', {configurable: true, value: 0});
    });
    test.each([
        ['/trader-sync/activities/12', 'read'],
        ['/trader-sync/activities/12', 'none'],
        ['/trader-sync/summaries/7', 'read'],
        ['/trader-sync/summaries/7', 'none']
    ])('real detail route %s rejects %s without private reads', async (path, level) => {
        ensureMemberBusinessServices();
        const activity = jest.spyOn(services.traderSync, 'getActivity');
        const batch = jest.spyOn(services.traderSync, 'getSummaryBatch');
        await mountMember(level, path);
        expect(window.location.pathname).toBe('/account/access');
        expect(activity).not.toHaveBeenCalled();
        expect(batch).not.toHaveBeenCalled();
    });
    test('ordinary Telegram and list activity deep links use the real page and ignore owner query overrides', async () => {
        ensureMemberBusinessServices();
        jest.spyOn(window, 'scrollTo').mockImplementation(() => undefined);
        const {normalizeActivity} = await import('./member/trader-sync-models');
        const fixture = await import('./member/testdata/trader-sync/05.json');
        const item = {...normalizeActivity(fixture.activity), noteSnapshot: 'original activity note'};
        const get = jest.spyOn(services.traderSync, 'getActivity').mockResolvedValue(item);
        await mountMember('read_write', '/trader-sync/activities/12?ownerId=owner-B');
        expect(document.title).toBe('Activity · Athena');
        expect(containsText(tree.toJSON(), 'original activity note')).toBe(true);
        expect(get).toHaveBeenCalledWith('12');
        const {readDetailSession} = await import('./member/pages/trader-sync/state');
        expect(readDetailSession('owner-A', 'activity/12')).toBeDefined();
        expect(readDetailSession('owner-B', 'activity/12')).toBeUndefined();
        jest.mocked(services.users.get).mockResolvedValue(member('owner-A', 'athena', 'none', 2));
        await act(async () => window.dispatchEvent(new Event('focus')));
        expect(window.location.pathname).toBe('/account/access');
        expect(containsText(tree.toJSON(), 'original activity note')).toBe(false);
        expect(readDetailSession('owner-A', 'activity/12')).toBeUndefined();
    });
    test('summary Telegram deep link loads the real batch and its independent members', async () => {
        ensureMemberBusinessServices();
        jest.spyOn(window, 'scrollTo').mockImplementation(() => undefined);
        const stamp = '2026-09-11T00:00:00Z';
        jest.spyOn(services.traderSync, 'getSummaryBatch').mockResolvedValue({
            id: '7',
            oldestAt: stamp,
            settledFrom: stamp,
            settledTo: stamp,
            recordedFrom: stamp,
            recordedTo: stamp,
            activityCount: '0',
            targetCounts: [],
            partCounts: {total: '0', pending: '0', sending: '0', sent: '0', failed: '0', unknown: '0', cancelled: '0'},
            asOf: stamp
        });
        jest.spyOn(services.traderSync, 'listSummaryParts').mockResolvedValue({parts: [], page: {}, asOf: stamp});
        jest.spyOn(services.traderSync, 'listActivities').mockResolvedValue({activities: [], page: {snapshot: 'batch-snapshot', refreshCursor: 'batch-refresh', hasNewer: false}});
        await mountMember('read_write', '/trader-sync/summaries/7');
        expect(document.title).toBe('Summary batch · Athena');
        expect(containsText(tree.toJSON(), 'Activities in this batch')).toBe(true);
        expect(services.traderSync.listActivities).toHaveBeenCalledWith(expect.objectContaining({summaryBatchId: '7'}));
        expect(tree.root.findByProps({'aria-label': 'Summary part pages'})).toBeDefined();
    });
    test.each(['/trader-sync/activities/12', '/trader-sync/summaries/7'])('logged-out deep link %s retains its login destination', async path => {
        jest.spyOn(services.authService, 'bootstrap').mockResolvedValue(anonymousBootstrap);
        window.history.replaceState(null, '', path);
        await act(async () => {
            tree = renderer.create(<App />);
        });
        expect(window.location.pathname).toBe('/login');
        const {readLoginReturnTo} = await import('./shared/login-navigation');
        expect(readLoginReturnTo(window.location.search)).toBe(path);
    });
    test('detail login return respects the existing deployment base helper', async () => {
        const {deploymentPath} = await import('./shared/runtime-base');
        const {loginPathFor, readLoginReturnTo} = await import('./shared/login-navigation');
        const meta = document.createElement('meta');
        meta.name = 'athena-deployment-base-href';
        meta.content = '/athena/';
        document.head.appendChild(meta);
        try {
            const path = '/trader-sync/summaries/7';
            expect(deploymentPath(loginPathFor(path))).toBe('/athena/login?returnTo=%2Ftrader-sync%2Fsummaries%2F7');
            expect(deploymentPath(readLoginReturnTo('?returnTo=%2Ftrader-sync%2Factivities%2F12'))).toBe('/athena/trader-sync/activities/12');
        } finally {
            meta.remove();
        }
    });
    test('RW Add route renders the actual independent page', async () => {
        await mountMember();
        expect(containsText(tree.toJSON(), 'Wallet address or Polymarket profile URL')).toBe(true);
    });
});

describe('T3 real member Shell identity boundaries', () => {
    let tree: renderer.ReactTestRenderer;
    const identity = (accountId = 'profile-A', iss = 'issuer-one', displayName = 'Original profile') =>
        parseUserInfo({
            loggedIn: true,
            accountId,
            iss,
            username: accountId,
            profile: {displayName, revision: 7},
            access: {loginEnabled: true, apiKeyEnabled: true, revision: 1}
        });
    const mount = async (path: string) => {
        ensureMemberBusinessServices();
        jest.spyOn(services.authService, 'bootstrap').mockResolvedValue({settings: authSettings, session: {status: AppBootstrapSessionStatus.Authenticated, userInfo: identity()}});
        jest.spyOn(services.users, 'get').mockResolvedValue(identity());
        jest.spyOn(services.memberNotifications, 'getTelegramSettings').mockResolvedValue({
            botAvailable: true,
            botUsername: 'bot',
            binding: {status: 'connected', boundAt: '', revision: 1, telegramDisplayName: 'Old Telegram', telegramUsername: 'old_user'}
        });
        jest.spyOn(services.memberSecurity, 'listTokens').mockResolvedValue([{id: 'old-identity-key', issuedAt: 1, expiresAt: 0}] as any);
        window.history.replaceState(null, '', path);
        await act(async () => {
            tree = renderer.create(<App />);
        });
    };
    afterEach(() => {
        if (tree) act(() => tree.unmount());
    });
    test.each([
        ['profile-B', 'issuer-one'],
        ['profile-A', 'issuer-two']
    ])('same-revision Profile switches to %s/%s without carrying the old draft', async (accountId, iss) => {
        await mount('/account/profile');
        const input = () => tree.root.findAllByType(Input).find(item => item.props.id === 'profile-display-name')!;
        await act(async () => input().props.onChange({target: {value: 'Private old draft'}}));
        jest.mocked(services.users.get).mockResolvedValue(identity(accountId, iss, 'Replacement profile'));
        await act(async () => window.dispatchEvent(new Event('focus')));
        expect(input().props.value).toBe('Replacement profile');
        expect(JSON.stringify(tree.toJSON())).not.toContain('Private old draft');
        expect(tree.root.findAllByType(Button).find(item => item.props.children === 'Save profile')!.props.disabled).toBe(true);
    });
    test.each([
        ['profile-B', 'issuer-one'],
        ['profile-A', 'issuer-two']
    ])('late Profile save cannot refresh replacement %s/%s', async (accountId, iss) => {
        await mount('/account/profile');
        let finish!: (value: any) => void;
        const pending = new Promise<any>(resolve => (finish = resolve));
        const save = jest.spyOn(selfAccountServices.accounts, 'updateProfile').mockReturnValue(pending);
        const input = () => tree.root.findAllByType(Input).find(item => item.props.id === 'profile-display-name')!;
        await act(async () => input().props.onChange({target: {value: 'Pending old draft'}}));
        await act(async () => tree.root.findByType(Form).props.onFinish());
        expect(save).toHaveBeenCalledWith('profile-A', 'Pending old draft', 7);
        jest.mocked(services.users.get).mockResolvedValue(identity(accountId, iss, 'Replacement profile'));
        await act(async () => window.dispatchEvent(new Event('focus')));
        jest.mocked(services.users.get).mockClear();
        await act(async () => finish({...identity().profile, displayName: 'Late old saved profile', revision: 8}));
        expect(services.users.get).not.toHaveBeenCalled();
        expect(input().props.value).toBe('Replacement profile');
    });
    test.each([
        ['profile-B', 'issuer-one'],
        ['profile-A', 'issuer-two']
    ])('Notifications switches to %s/%s and clears the old binding', async (accountId, iss) => {
        await mount('/notifications');
        expect(JSON.stringify(tree.toJSON())).toContain('Old Telegram');
        let finish!: (value: any) => void;
        const pending = Object.assign(new Promise<any>(resolve => (finish = resolve)), {abort: jest.fn()});
        jest.mocked(services.memberNotifications.getTelegramSettings).mockReturnValueOnce(pending).mockResolvedValue({botAvailable: true, botUsername: 'bot'});
        jest.mocked(services.users.get).mockResolvedValue(identity(accountId, iss));
        await act(async () => window.dispatchEvent(new Event('focus')));
        expect(JSON.stringify(tree.toJSON())).not.toContain('Old Telegram');
        await act(async () =>
            finish({
                botAvailable: true,
                botUsername: 'bot',
                binding: {status: 'connected', boundAt: '', revision: 1, telegramDisplayName: 'Late old Telegram', telegramUsername: 'old'}
            })
        );
        expect(JSON.stringify(tree.toJSON())).not.toContain('Late old Telegram');
    });
});
