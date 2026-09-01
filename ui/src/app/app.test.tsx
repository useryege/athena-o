import * as React from 'react';
import renderer, {act} from 'react-test-renderer';
import {Button} from 'antd';
import {MemberApp as App, loadAppBootstrapWithRetry} from './member/app';
import {AccountDataAccess, AppBootstrap, AppBootstrapSessionStatus, AuthSettings} from './shared/models';
import {memberServices as services} from './member/services';

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
        userInfo: {
            loggedIn: true,
            username: 'admin',
            iss: 'athena',
            administrator: true,
            dataAccess: AccountDataAccess.ReadWrite,
            authorizationRevision: 0
        }
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

const containsText = (node: renderer.ReactTestRendererJSON | renderer.ReactTestRendererJSON[] | string | null, text: string): boolean => {
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
    const load = jest
        .fn<Promise<AppBootstrap>, []>()
        .mockRejectedValueOnce(new Error('backend is not ready'))
        .mockResolvedValue(authenticatedBootstrap);
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
    expect(containsText(tree.toJSON(), 'Log in')).toBe(true);
});
