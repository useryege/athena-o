import * as React from 'react';
import renderer, {act} from 'react-test-renderer';
import {Button} from 'antd';
import {App, loadAuthSettingsWithRetry} from './app';
import {AuthSettings} from './shared/models';
import {services} from './shared/services';

const authSettings: AuthSettings = {
    url: '',
    statusBadgeEnabled: false,
    statusBadgeRootUrl: '',
    googleAnalytics: {
        trackingID: '',
        anonymizeUsers: false
    },
    dexConfig: {
        connectors: []
    },
    oidcConfig: null,
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

const loggedOutUser = {loggedIn: false, username: '', iss: '', groups: []};
const loggedInUser = {loggedIn: true, username: 'admin', iss: 'athena', groups: []};

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

test('loadAuthSettingsWithRetry retries transient settings failures', async () => {
    const load = jest
        .fn<Promise<AuthSettings>, []>()
        .mockRejectedValueOnce(new Error('backend is not ready'))
        .mockResolvedValue(authSettings);
    const sleep = jest.fn<Promise<void>, [number]>(() => Promise.resolve());

    await expect(loadAuthSettingsWithRetry(load, [500], sleep)).resolves.toEqual(authSettings);

    expect(load).toHaveBeenCalledTimes(2);
    expect(sleep).toHaveBeenCalledWith(500);
});

test('loadAuthSettingsWithRetry returns the final settings error', async () => {
    const finalError = new Error('still unavailable');
    const load = jest.fn<Promise<AuthSettings>, []>().mockRejectedValue(finalError);

    await expect(loadAuthSettingsWithRetry(load, [500, 1000], () => Promise.resolve())).rejects.toThrow('still unavailable');
    expect(load).toHaveBeenCalledTimes(3);
});

test('Bootstrap renders recoverable settings failure and retries on demand', async () => {
    jest.useFakeTimers();
    const settings = jest
        .spyOn(services.authService, 'settings')
        .mockRejectedValueOnce(new Error('api offline'))
        .mockRejectedValueOnce(new Error('api offline'))
        .mockRejectedValueOnce(new Error('api offline'))
        .mockRejectedValueOnce(new Error('api offline'))
        .mockRejectedValueOnce(new Error('api offline'))
        .mockResolvedValue(authSettings);
    jest.spyOn(services.users, 'get').mockResolvedValue(loggedInUser);

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

    expect(settings).toHaveBeenCalledTimes(6);
    settings.mockRestore();
    jest.useRealTimers();
});

test('Bootstrap redirects logged-out protected routes to login', async () => {
    jest.spyOn(services.authService, 'settings').mockResolvedValue(authSettings);
    jest.spyOn(services.users, 'get').mockResolvedValue(loggedOutUser);
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
