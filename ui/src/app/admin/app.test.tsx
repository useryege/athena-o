import {moduleAccessService} from '../shared/module-access-service';
import * as React from 'react';
import renderer, {act} from 'react-test-renderer';
import {Button, Input} from 'antd';
import {AuthorizationCtx, useAuthorization, type AuthorizationState} from '../shared/context';
import {parseUserInfo, AppBootstrapSessionStatus} from '../shared/models';
import {useVisibleQuery} from '../shared/use-visible-query';
import {endAdminReadSession, useAdminReadScope} from './read-scope';
import requests from '../shared/services/requests';
import {adminServices as services, ensureAdminBusinessServices} from './services';
import {AdminApp} from './app';

let mockRealProfile = false;
let mockAuth: AuthorizationState;
let mockLoad: () => Promise<string>;
const mockUser = () => parseUserInfo({accountId: 'admin-A', iss: 'issuer-one', loggedIn: true, administrator: true, username: 'admin'});
jest.mock('../session/bootstrap', () => ({
    SessionBootstrap: ({children}: any) => children({session: {status: AppBootstrapSessionStatus.Authenticated, userInfo: mockUser()}, settings: {}}, {hideSidebar: false})
}));
jest.mock('./routes', () => ({
    ...jest.requireActual('./routes'),
    AccountCenterPage: (props: any) => {
        mockAuth = useAuthorization();
        const data = useVisibleQuery(mockLoad, useAdminReadScope('shell-test'), 10000);
        if (mockRealProfile) {
            const Page = jest.requireActual('../shared/pages/account-center').AccountCenterPage;
            return <Page {...props} />;
        }
        return <span>{data.data}</span>;
    }
}));
let tree: renderer.ReactTestRenderer;
const text = () => JSON.stringify(tree.toJSON());
const mount = async () => {
    await act(async () => {
        tree = renderer.create(<AdminApp />);
    });
};
const late = () => {
    let resolve!: (v: string) => void;
    const promise = Object.assign(new Promise<string>(r => (resolve = r)), {abort: jest.fn()});
    return {promise, resolve};
};
beforeEach(() => {
    jest.spyOn(moduleAccessService, 'states').mockResolvedValue(
        ['trader_sync', 'solana', 'market_radar', 'managed_oo', 'profit_sharing', 'worm'].map(module_key => ({module_key, state: 1})) as any
    );
    mockRealProfile = false;
    window.matchMedia = jest.fn().mockImplementation(query => ({
        matches: false,
        media: query,
        addListener: jest.fn(),
        removeListener: jest.fn(),
        addEventListener: jest.fn(),
        removeEventListener: jest.fn()
    }));
    window.history.replaceState({}, '', '/account/profile');
    mockLoad = jest.fn().mockResolvedValue('cached administrator summary');
    jest.spyOn(services.users, 'get').mockResolvedValue(mockUser());
});
afterEach(() => {
    act(() => {
        tree?.unmount();
        endAdminReadSession();
        requests.endAuthorizationSession();
    });
    jest.restoreAllMocks();
});
test('real Shell clears cached data when refresh discovers logout before redirect can unmount', async () => {
    await mount();
    expect(text()).toContain('cached administrator summary');
    jest.mocked(services.users.get).mockResolvedValue(parseUserInfo({loggedIn: false}));
    await act(async () => mockAuth.refresh());
    expect(text()).not.toContain('cached administrator summary');
});
test('administrator identity control belongs to the top bar and the desktop navigation uses the approved width', async () => {
    await mount();
    const header = tree.root.findByProps({className: 'athena-shell__header'});
    const sider = tree.root.findByProps({className: 'athena-shell__sider'});
    expect(header.findAllByProps({'aria-label': 'Open account menu'})).toHaveLength(1);
    expect(sider.findAllByProps({'aria-label': 'Open account menu'})).toHaveLength(0);
    expect(sider.props.width).toBe(224);
});
test('removed administrator Appearance URL uses the existing not-found fallback', async () => {
    window.history.replaceState({}, '', '/account/appearance');
    await mount();
    expect(text()).toContain('Page not found');
});
test('real Shell swaps same-account issuer, fences old reads and reopens only the new scope', async () => {
    const pending = late();
    mockLoad = jest.fn().mockReturnValueOnce(pending.promise).mockResolvedValue('new issuer summary');
    await mount();
    jest.mocked(services.users.get).mockResolvedValue({...mockUser(), iss: 'issuer-two'});
    await act(async () => mockAuth.refresh());
    await act(async () => pending.resolve('old issuer late result'));
    expect(text()).toContain('new issuer summary');
    expect(text()).not.toContain('old issuer late result');
});
test('real Shell known 401 clears hook data even when navigation remains in the mounted test', async () => {
    await mount();
    expect(text()).toContain('cached administrator summary');
    await act(async () => {
        (requests.get('/test-auth-event') as any).emit('error', {status: 401});
    });
    expect(text()).not.toContain('cached administrator summary');
});
test('real Shell admin-required denial clears immediately while role refresh is still pending', async () => {
    await mount();
    const pending = late();
    jest.mocked(services.users.get).mockReturnValue(pending.promise as any);
    await act(async () => {
        (requests.get('/test-admin-event') as any).emit('error', {status: 403, response: {body: {reason: 'ACCOUNT_ADMIN_REQUIRED'}}});
    });
    expect(text()).not.toContain('cached administrator summary');
});
test('non-authorization errors preserve the active administrator summary', async () => {
    await mount();
    await act(async () => {
        (requests.get('/test-runtime-error') as any).emit('error', {status: 503});
    });
    expect(text()).toContain('cached administrator summary');
});
test('real Shell unmount invalidates any read scope left by a still-mounted observer', async () => {
    await mount();
    let observer: renderer.ReactTestRenderer;
    function Observer() {
        const data = useVisibleQuery(() => Promise.resolve('observer data'), useAdminReadScope('observer'), 10000);
        return <span>{data.data}</span>;
    }
    await act(async () => {
        observer = renderer.create(
            <AuthorizationCtx.Provider value={mockAuth}>
                <Observer />
            </AuthorizationCtx.Provider>
        );
    });
    expect(JSON.stringify(observer!.toJSON())).toContain('observer data');
    act(() => tree.unmount());
    expect(JSON.stringify(observer!.toJSON())).not.toContain('observer data');
    act(() => observer!.unmount());
});
test('a user refresh started before a known 401 cannot reauthorize when its old response arrives late', async () => {
    await mount();
    let resolve!: (value: any) => void;
    const promise = new Promise<any>(r => (resolve = r));
    jest.mocked(services.users.get).mockReturnValue(promise);
    let refresh: Promise<void>;
    act(() => {
        refresh = mockAuth.refresh();
    });
    await act(async () => {
        (requests.get('/late-refresh-401') as any).emit('error', {status: 401});
        resolve(mockUser());
        await refresh;
    });
    expect(text()).not.toContain('cached administrator summary');
});
test.each(['/trader-sync/subscriptions', '/trader-sync/subscriptions/sub-1'])('actual administrator lazy route %s renders its safe summary surface', async path => {
    window.history.replaceState({}, '', path);
    ensureAdminBusinessServices();
    jest.spyOn(services.adminTraderSync, 'listSubscriptionSummaries').mockResolvedValue({summaries: [], page: {}, asOf: '2026-09-11T01:00:00Z'});
    jest.spyOn(services.adminTraderSync, 'getSubscriptionSummary').mockResolvedValue({
        subscriptionId: 'sub-1',
        accountId: 'owner',
        username: 'safe-user',
        email: 'safe@example.test',
        wallet: 'safe-wallet',
        status: 'healthy',
        createdAt: '',
        updatedAt: '',
        observation: {state: 'healthy', reason: ''},
        associatedDeliveryCounts: {},
        asOf: ''
    });
    await mount();
    expect(text()).toContain(path.endsWith('sub-1') ? 'safe-wallet' : 'Current subscriptions');
    expect(document.title).toBe('Trader Sync · Athena Admin');
});

test('real Shell refresh discovering lost administrator role replaces summaries with Forbidden', async () => {
    await mount();
    expect(text()).toContain('cached administrator summary');
    jest.mocked(services.users.get).mockResolvedValue({...mockUser(), administrator: false});
    await act(async () => mockAuth.refresh());
    expect(text()).toContain('Administrator access required');
    expect(text()).not.toContain('cached administrator summary');
});

test('role recheck failure is visible and retry stays single flight before fresh authorized data returns', async () => {
    const oldRead = late();
    mockLoad = jest.fn().mockResolvedValueOnce('cached administrator summary').mockReturnValueOnce(oldRead.promise).mockResolvedValue('fresh authorized summary');
    await mount();
    await act(async () => window.dispatchEvent(new Event('focus')));
    jest.mocked(services.users.get).mockRejectedValue(new Error('role lookup offline'));
    await act(async () => {
        (requests.get('/recheck-failure') as any).emit('error', {status: 403, response: {body: {reason: 'ACCOUNT_ADMIN_REQUIRED'}}});
    });
    expect(text()).toContain('role lookup offline');
    expect(tree.root.findAllByProps({role: 'alert'}).length).toBeGreaterThan(0);
    expect(text()).not.toContain('cached administrator summary');
    await act(async () => oldRead.resolve('late invalidated summary'));
    expect(text()).not.toContain('late invalidated summary');
    let resolve!: (value: any) => void;
    jest.mocked(services.users.get).mockReturnValue(new Promise<any>(done => (resolve = done)));
    const retry = tree.root.findAllByType(Button).find(button => button.props.children === 'Retry access check')!;
    await act(async () => {
        retry.props.onClick();
        retry.props.onClick();
    });
    expect(services.users.get).toHaveBeenCalledTimes(2);
    expect(text()).toContain('Checking administrator access');
    expect(tree.root.findAllByType(Button).find(button => button.props.children === 'Retry access check')!.props.disabled).toBe(true);
    expect(text()).not.toContain('fresh authorized summary');
    await act(async () => resolve({...mockUser(), iss: 'verified-issuer'}));
    expect(text()).toContain('fresh authorized summary');
    expect(text()).not.toContain('role lookup offline');
    expect(mockAuth.user.iss).toBe('verified-issuer');
});

test('retry after a role recheck failure cannot reopen an account that is no longer an administrator', async () => {
    await mount();
    jest.mocked(services.users.get).mockRejectedValue(new Error('role lookup offline'));
    await act(async () => {
        (requests.get('/recheck-denied') as any).emit('error', {status: 403, response: {body: {reason: 'ACCOUNT_ADMIN_REQUIRED'}}});
    });
    expect(text()).toContain('role lookup offline');
    jest.mocked(services.users.get).mockResolvedValue({...mockUser(), administrator: false});
    await act(async () =>
        tree.root
            .findAllByType(Button)
            .find(button => button.props.children === 'Retry access check')!
            .props.onClick()
    );
    expect(text()).toContain('Administrator access required');
    expect(text()).not.toContain('cached administrator summary');
});

test('a pending role recheck retry cannot reopen data after a newer known 401', async () => {
    await mount();
    jest.mocked(services.users.get).mockRejectedValue(new Error('role lookup offline'));
    await act(async () => {
        (requests.get('/recheck-late') as any).emit('error', {status: 403, response: {body: {reason: 'ACCOUNT_ADMIN_REQUIRED'}}});
    });
    expect(text()).toContain('role lookup offline');
    let resolve!: (value: any) => void;
    jest.mocked(services.users.get).mockReturnValue(new Promise<any>(done => (resolve = done)));
    await act(async () =>
        tree.root
            .findAllByType(Button)
            .find(button => button.props.children === 'Retry access check')!
            .props.onClick()
    );
    await act(async () => {
        (requests.get('/recheck-newer-401') as any).emit('error', {status: 401});
        resolve(mockUser());
    });
    expect(text()).not.toContain('cached administrator summary');
});

test.each([
    ['admin-B', 'issuer-one'],
    ['admin-A', 'issuer-two']
])('real administrator Profile drops same-revision draft on %s/%s', async (accountId, iss) => {
    mockRealProfile = true;
    await mount();
    const input = () => tree.root.findAllByType(Input).find(item => item.props.id === 'profile-display-name')!;
    await act(async () => input().props.onChange({target: {value: 'Private admin draft'}}));
    jest.mocked(services.users.get).mockResolvedValue(
        parseUserInfo({loggedIn: true, administrator: true, accountId, iss, username: 'replacement-admin', profile: {displayName: 'Replacement admin', revision: 0}})
    );
    await act(async () => mockAuth.refresh());
    expect(input().props.value).toBe('Replacement admin');
    expect(text()).not.toContain('Private admin draft');
    expect(tree.root.findAllByType(Button).find(item => item.props.children === 'Save profile')!.props.disabled).toBe(true);
});
