import {TextEncoder} from 'util';
import {ModuleAccessProvider, ModuleAccessBoundary} from '../../shared/module-access';
import {AccountIdentityProvider} from '../../shared/models';
import {moduleAccessService} from '../../shared/module-access-service';
import renderer, {act} from 'react-test-renderer';
import {StyleProvider} from '@ant-design/cssinjs';
import {Button} from 'antd';
import {MemoryRouter, Routes, Route} from 'react-router-dom';
import {Context, AuthorizationCtx} from '../../shared/context';
import {WormTradingExecutionDetailPage} from './worm-trading-executions';
import {WormTradingExecutionPreviewPage} from './worm-trading-execution-preview';
import {MemberApp} from '../app';
import {parseUserInfo, AppBootstrapSessionStatus} from '../../shared/models';
import {ensureMemberBusinessServices, memberServices as services} from '../services';
import cases from '../../../../e2e/theme-refactor/fixtures/worm-executions.json';

const fixture = cases.find(item => item.id === 'worm-execution-detail')!;
const replies = fixture.replies as any[];
const raw = replies[1].json;
const normalizedRun = {
    ...raw,
    combinationId: raw.combination.id,
    combinationName: raw.combination.name,
    combinationRevision: raw.combination.revision,
    blockCode: '',
    failureCode: '',
    pauseCode: ''
};
const bootstrap = replies[0].json;
let tree: renderer.ReactTestRenderer;
const identity = (iss: string) => parseUserInfo({...bootstrap.session.userInfo, iss});
beforeEach(() => {
    jest.spyOn(moduleAccessService, 'states').mockResolvedValue(
        ['trader_sync', 'solana', 'market_radar', 'managed_oo', 'profit_sharing', 'worm'].map(module_key => ({module_key, state: 1})) as any
    );
    Object.defineProperty(window.crypto, 'randomUUID', {configurable: true, value: () => '22000000-0000-4000-8000-000000009999'});
    ensureMemberBusinessServices();
    localStorage.clear();
    sessionStorage.clear();
    window.matchMedia = jest.fn().mockImplementation(query => ({
        matches: false,
        media: query,
        addListener: jest.fn(),
        removeListener: jest.fn(),
        addEventListener: jest.fn(),
        removeEventListener: jest.fn()
    }));
    jest.spyOn(services.authService, 'bootstrap').mockResolvedValue({
        settings: bootstrap.settings,
        session: {status: AppBootstrapSessionStatus.Authenticated, userInfo: identity('old-issuer')}
    });
    jest.spyOn(services.users, 'get').mockResolvedValue(identity('old-issuer'));
    jest.spyOn(services.wormTrading, 'getExecutionRun').mockResolvedValue(normalizedRun);
    jest.spyOn(services.wormTrading, 'listExecutionRunSteps').mockResolvedValue({
        ...replies[2].json,
        items: replies[2].json.items.map((step: any) => ({
            providerState: '',
            providerOrderState: '',
            completionSource: '',
            completionPositionPubkey: '',
            completionPositionRequestPubkey: '',
            completionPositionCreatedAt: 0,
            completedAt: 0,
            positionRequestId: '',
            ...step
        }))
    });
});
afterEach(() => {
    if (tree) act(() => tree.unmount());
    jest.restoreAllMocks();
});
const mount = async () => {
    window.history.replaceState(null, '', fixture.route);
    await act(async () => {
        tree = renderer.create(<MemberApp />);
    });
};
test('execution detail reloads its frozen data for a new issuer with the same account and revision', async () => {
    await mount();
    jest.mocked(services.wormTrading.getExecutionRun).mockResolvedValue({...normalizedRun, combinationName: 'New issuer frozen basket'});
    jest.mocked(services.users.get).mockResolvedValue(identity('new-issuer'));
    await act(async () => window.dispatchEvent(new Event('focus')));
    expect(JSON.stringify(tree.toJSON())).toContain('New issuer frozen basket');
    expect(JSON.stringify(tree.toJSON())).not.toContain('September market basket');
});
test('first failed execution history does not present successful empty records or pagination', async () => {
    jest.spyOn(services.wormTrading, 'listExecutionRuns').mockRejectedValue(new Error('Execution history unavailable'));
    window.history.replaceState(null, '', '/worm-trading/executions');
    await act(async () => {
        tree = renderer.create(<MemberApp />);
    });
    const output = JSON.stringify(tree.toJSON());
    expect(output).toContain('Execution history unavailable');
    expect(output).not.toContain('No executions on this page');
    expect(output).not.toContain('resource-table-pagination');
});

const authorization = {
    user: {...identity('old-issuer'), identity: {...identity('old-issuer').identity, provider: 'ACCOUNT_IDENTITY_PROVIDER_DEVELOPMENT' as any}},
    revision: 1,
    canWrite: () => true,
    canRead: () => true,
    access: () => 2,
    isAdmin: false,
    lastCheckedAt: 0,
    refresh: async () => {}
};
let confirmation: any;
const notifications = {success: jest.fn(), error: jest.fn(), warning: jest.fn(), info: jest.fn()};
const directMount = async (openAuthorization = true, guarded = false) => {
    Object.values(notifications).forEach(fn => fn.mockClear());
    await act(async () => {
        tree = renderer.create(
            <StyleProvider mock='server'>
                <MemoryRouter future={{v7_startTransition: true, v7_relativeSplatPath: true}} initialEntries={[fixture.route]}>
                    <AuthorizationCtx.Provider value={authorization}>
                        <Context.Provider
                            value={{
                                notifications,
                                modal: {
                                    confirm: options => {
                                        confirmation = options;
                                        return {destroy: jest.fn()};
                                    },
                                    info: jest.fn(),
                                    error: jest.fn()
                                },
                                navigation: {goto: jest.fn(), replace: jest.fn()},
                                baseHref: ''
                            }}
                        >
                            <Routes>
                                <Route
                                    path='/worm-trading/executions/:id'
                                    element={
                                        guarded ? (
                                            <ModuleAccessProvider realm='member' identity='execution-test'>
                                                <ModuleAccessBoundary moduleKey='worm'>
                                                    <WormTradingExecutionDetailPage />
                                                </ModuleAccessBoundary>
                                            </ModuleAccessProvider>
                                        ) : (
                                            <WormTradingExecutionDetailPage />
                                        )
                                    }
                                />
                            </Routes>
                        </Context.Provider>
                    </AuthorizationCtx.Provider>
                </MemoryRouter>
            </StyleProvider>
        );
    });
    if (openAuthorization)
        act(() =>
            tree.root
                .findAllByType(Button)
                .find(button => button.props.children === 'Authorize')!
                .props.onClick()
        );
};
test('late execution authorization after leaving cannot announce success', async () => {
    let resolve!: (value: any) => void;
    jest.spyOn(services.wormTrading, 'authorizeDevelopmentExecutionRun').mockReturnValue(new Promise<any>(done => (resolve = done)));
    await directMount();
    let pending!: Promise<void>;
    act(() => {
        pending = confirmation.onOk();
    });
    act(() => tree.unmount());
    await act(async () => {
        resolve({...normalizedRun, state: 'AUTHORIZED', allowedActions: ['START', 'TERMINATE']});
        await pending;
    });
    expect(notifications.success).not.toHaveBeenCalled();
    expect(notifications.error).not.toHaveBeenCalled();
});
test('authorization admits only one in-flight command even before React renders pending state', async () => {
    let resolve!: (value: any) => void;
    const call = jest.spyOn(services.wormTrading, 'authorizeDevelopmentExecutionRun').mockReturnValue(new Promise<any>(done => (resolve = done)));
    await directMount();
    let first!: Promise<void>;
    let second!: Promise<void>;
    act(() => {
        first = confirmation.onOk();
        second = confirmation.onOk();
    });
    expect(call).toHaveBeenCalledTimes(1);
    await act(async () => {
        resolve({...normalizedRun, state: 'AUTHORIZED', allowedActions: ['START', 'TERMINATE']});
        await Promise.all([first, second]);
    });
});

const previewFixture = cases.find(item => item.id === 'worm-preview')!;
const previewMount = async (combinationRevision = 7) => {
    const reply = (suffix: string) => (previewFixture.replies as any[]).find(item => item.path.endsWith(suffix)).json;
    const rawPlan = reply('/22000000-0000-4000-8000-000000000002');
    const plan = {
        ...rawPlan,
        combinationId: rawPlan.combination.id,
        combinationName: rawPlan.combination.name,
        combinationRevision: rawPlan.combination.revision,
        usabilityCode: '',
        expiresAt: Math.floor(Date.now() / 1000) + 600
    };
    jest.spyOn(services.wormTrading, 'getMarketCombination').mockResolvedValue({...reply('/22000000-0000-4000-8000-000000000001'), revision: combinationRevision});
    jest.spyOn(services.wormTrading, 'getWalletSelection').mockResolvedValue(reply('/wallet-selection'));
    jest.spyOn(services.wormTrading, 'listWalletConnections').mockResolvedValue(reply('/wallet-connections'));
    jest.spyOn(services.wormTrading, 'getExecutionPlan').mockResolvedValue(plan);
    jest.spyOn(services.wormTrading, 'listExecutionPlanSteps').mockResolvedValue(reply('/steps'));
    await act(async () => {
        tree = renderer.create(
            <StyleProvider mock='server'>
                <MemoryRouter future={{v7_startTransition: true, v7_relativeSplatPath: true}} initialEntries={[previewFixture.route]}>
                    <AuthorizationCtx.Provider value={authorization}>
                        <Context.Provider
                            value={{notifications, modal: {confirm: jest.fn(), info: jest.fn(), error: jest.fn()}, navigation: {goto: jest.fn(), replace: jest.fn()}, baseHref: ''}}
                        >
                            <Routes>
                                <Route path='/worm-trading/combinations/:id/execute' element={<WormTradingExecutionPreviewPage />} />
                                <Route path='/worm-trading/executions/:id' element={<div>Prepared destination</div>} />
                            </Routes>
                        </Context.Provider>
                    </AuthorizationCtx.Provider>
                </MemoryRouter>
            </StyleProvider>
        );
    });
    return tree.root.findAllByType(Button).find(button => button.props.children === 'Prepare live execution')!;
};
test('Prepare admits one frozen Run request before the pending render', async () => {
    let resolve!: (value: any) => void;
    const call = jest.spyOn(services.wormTrading, 'createExecutionRun').mockReturnValue(new Promise<any>(done => (resolve = done)));
    const button = await previewMount();
    expect(button.props.disabled).toBe(false);
    act(() => {
        button.props.onClick();
        button.props.onClick();
    });
    expect(call).toHaveBeenCalledTimes(1);
    await act(async () => resolve(normalizedRun));
});
test('late Prepare rejection after leaving cannot announce an error', async () => {
    notifications.error.mockClear();
    let reject!: (error: Error) => void;
    jest.spyOn(services.wormTrading, 'createExecutionRun').mockReturnValue(new Promise<any>((_, done) => (reject = done)));
    const button = await previewMount();
    act(() => button.props.onClick());
    act(() => tree.unmount());
    await act(async () => reject(new Error('Late response')));
    expect(notifications.error).not.toHaveBeenCalled();
});

test('Prepare blocks a changed combination before a plan usability refresh', async () => {
    const button = await previewMount(8);
    expect(button.props.disabled).toBe(true);
});

test.each([
    {action: 'pause', state: 'COMPLETED', allowedActions: [], sends: false},
    {action: 'terminate', state: 'COMPLETED', allowedActions: [], sends: false},
    {action: 'pause', state: 'RECONCILIATION_REQUIRED', allowedActions: ['TERMINATE', 'RECONCILE'], sends: false},
    {action: 'terminate', state: 'RECONCILIATION_REQUIRED', allowedActions: ['TERMINATE', 'RECONCILE'], sends: true}
])('$action rechecks latest allowedActions after convergence to $state', async ({action, state, allowedActions, sends}) => {
    const initial = {...normalizedRun, state: 'RUNNING', allowedActions: ['PAUSE', 'TERMINATE']};
    const latest = {...normalizedRun, state, allowedActions, revision: 17};
    jest.mocked(services.wormTrading.getExecutionRun).mockResolvedValue(initial);
    const pause = jest.spyOn(services.wormTrading, 'pauseExecutionRun').mockResolvedValue({run: latest} as any);
    const terminate = jest.spyOn(services.wormTrading, 'terminateExecutionRun').mockResolvedValue({run: latest} as any);
    await directMount(false);
    jest.mocked(services.wormTrading.getExecutionRun).mockResolvedValue(latest);
    await act(async () => {
        tree.root
            .findAllByType(Button)
            .find(button => typeof button.props.children === 'string' && (action === 'pause' ? button.props.children.startsWith('Pause') : button.props.children === 'Terminate'))!
            .props.onClick();
        if (action === 'terminate') await confirmation.onOk();
    });
    expect(services.wormTrading.getExecutionRun).toHaveBeenCalledTimes(2);
    expect(pause).not.toHaveBeenCalled();
    if (sends) {
        expect(terminate).toHaveBeenCalledTimes(1);
        expect(terminate).toHaveBeenCalledWith(latest.id, {commandId: expect.any(String), expectedRevision: 17});
    } else expect(terminate).not.toHaveBeenCalled();
    expect(JSON.stringify(tree.toJSON())).toContain(state === 'COMPLETED' ? 'Completed' : 'Reconciliation Required');
    expect(notifications.error).not.toHaveBeenCalled();
    if (allowedActions.includes('TERMINATE')) {
        const remaining = tree.root.findAllByType(Button).find(button => button.props.children === 'Terminate')!;
        expect(remaining.props.disabled).toBe(false);
        expect(remaining.props.loading).toBe(false);
    }
});

test('retirement preserves frozen execution and step history without Markets navigation', async () => {
    await mount();
    expect(services.wormTrading.getExecutionRun).toHaveBeenCalled();
    expect(services.wormTrading.listExecutionRunSteps).toHaveBeenCalled();
    expect(JSON.stringify(tree.toJSON())).toContain('September market basket');
    expect(JSON.stringify(tree.toJSON())).not.toContain('Worm Markets');
});

test('closing and reopening while Phantom is pending cannot verify or start the old authorization', async () => {
    jest.useFakeTimers();
    const previousEncoder = globalThis.TextEncoder;
    globalThis.TextEncoder = TextEncoder;
    const previousIdentity = authorization.user.identity;
    authorization.user.identity = {...previousIdentity, provider: AccountIdentityProvider.SolanaWallet, solanaAddress: 'login-address'};
    let resolveSignature!: (value: {signature: Uint8Array}) => void;
    const signMessage = jest.fn(
        () =>
            new Promise<{signature: Uint8Array}>(resolve => {
                resolveSignature = resolve;
            })
    );
    (window as any).phantom = {solana: {isPhantom: true, publicKey: {toString: () => 'login-address'}, signMessage}};
    jest.spyOn(services.wormTrading, 'createSolanaExecutionAuthorizationChallenge').mockResolvedValue({message: 'approve'} as any);
    const verify = jest.spyOn(services.wormTrading, 'verifySolanaExecutionAuthorization').mockResolvedValue(normalizedRun);
    const start = jest.spyOn(services.wormTrading, 'startExecutionRun');
    const resume = jest.spyOn(services.wormTrading, 'continueExecutionRun');
    await directMount(true, true);
    let pending!: Promise<void>;
    await act(async () => {
        pending = confirmation.onOk();
    });
    expect(signMessage).toHaveBeenCalledTimes(1);
    jest.mocked(moduleAccessService.states).mockResolvedValue(
        ['trader_sync', 'solana', 'market_radar', 'managed_oo', 'profit_sharing', 'worm'].map(module_key => ({module_key, state: 2})) as any
    );
    await act(async () => jest.advanceTimersByTimeAsync(2000));
    expect(JSON.stringify(tree.toJSON())).toContain('This module is not open yet');
    jest.mocked(moduleAccessService.states).mockResolvedValue(
        ['trader_sync', 'solana', 'market_radar', 'managed_oo', 'profit_sharing', 'worm'].map(module_key => ({module_key, state: 1})) as any
    );
    await act(async () => jest.advanceTimersByTimeAsync(2000));
    await act(async () => {
        resolveSignature({signature: new Uint8Array([1])});
        await pending;
    });
    expect(verify).not.toHaveBeenCalled();
    expect(start).not.toHaveBeenCalled();
    expect(resume).not.toHaveBeenCalled();
    expect(notifications.success).not.toHaveBeenCalled();
    authorization.user.identity = previousIdentity;
    globalThis.TextEncoder = previousEncoder;
    delete (window as any).phantom;
    jest.useRealTimers();
});

test('closing cancels the execution driver timer; reopening only reads authoritative state', async () => {
    jest.useFakeTimers();
    const authorized = {...normalizedRun, state: 'AUTHORIZED', allowedActions: ['START', 'TERMINATE']};
    const running = {...normalizedRun, state: 'RUNNING', allowedActions: ['EXECUTE_NEXT', 'PAUSE', 'TERMINATE'], nextStepOrdinal: 1};
    jest.mocked(services.wormTrading.getExecutionRun).mockResolvedValue(authorized as any);
    const start = jest.spyOn(services.wormTrading, 'startExecutionRun').mockResolvedValue({run: running, coordinatorToken: 'coordinator'} as any);
    const execute = jest
        .spyOn(services.wormTrading, 'executeNextExecutionStep')
        .mockResolvedValue({run: {...running, allowedActions: ['PAUSE', 'TERMINATE']}, coordinatorToken: 'coordinator'} as any);
    const heartbeat = jest.spyOn(services.wormTrading, 'heartbeatExecutionRun');
    const pause = jest.spyOn(services.wormTrading, 'pauseExecutionRun');
    const terminate = jest.spyOn(services.wormTrading, 'terminateExecutionRun');
    await directMount(false, true);
    await act(async () =>
        tree.root
            .findAllByType(Button)
            .find(button => button.props.children === 'Start')!
            .props.onClick()
    );
    expect(start).toHaveBeenCalledTimes(1);
    expect(execute).toHaveBeenCalledTimes(1);
    jest.mocked(moduleAccessService.states).mockResolvedValue(
        ['trader_sync', 'solana', 'market_radar', 'managed_oo', 'profit_sharing', 'worm'].map(module_key => ({module_key, state: 2})) as any
    );
    await act(async () => jest.advanceTimersByTimeAsync(2000));
    expect(JSON.stringify(tree.toJSON())).toContain('This module is not open yet');
    jest.mocked(services.wormTrading.getExecutionRun).mockResolvedValue(running as any);
    jest.mocked(moduleAccessService.states).mockResolvedValue(
        ['trader_sync', 'solana', 'market_radar', 'managed_oo', 'profit_sharing', 'worm'].map(module_key => ({module_key, state: 1})) as any
    );
    await act(async () => jest.advanceTimersByTimeAsync(20000));
    expect(JSON.stringify(tree.toJSON())).toContain('Running');
    expect(execute).toHaveBeenCalledTimes(1);
    expect(heartbeat).not.toHaveBeenCalled();
    expect(pause).not.toHaveBeenCalled();
    expect(terminate).not.toHaveBeenCalled();
    jest.useRealTimers();
});
