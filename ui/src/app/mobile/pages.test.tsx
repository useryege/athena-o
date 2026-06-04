import {Account, UserInfo} from '../shared/models';
import * as React from 'react';
import {MemoryRouter, Route, Routes, useLocation} from 'react-router-dom';
import renderer, {act} from 'react-test-renderer';
import {LoginPage, WormMarketSummary, notificationTestTopics, visibleAccountsForUser} from './pages';
import {WormMarketItem} from '../shared/services/worm-service';
import {services} from '../shared/services';

const accounts: Account[] = [
    {name: 'admin', enabled: true, capabilities: ['login'], tokens: []},
    {name: 'LINGJIE', enabled: true, capabilities: ['login'], tokens: []},
    {name: 'YUDIAN', enabled: true, capabilities: ['login'], tokens: []}
];

const user = (username: string): UserInfo => ({loggedIn: true, username, iss: 'athena', groups: []});
const loggedOutUser: UserInfo = {loggedIn: false, username: '', iss: '', groups: []};

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

const LocationProbe = () => {
    const location = useLocation();
    return <span data-testid='location'>{location.pathname}</span>;
};

const renderLoginRoute = async (initialEntry = '/login') => {
    let tree: renderer.ReactTestRenderer;
    await act(async () => {
        tree = renderer.create(
            <MemoryRouter initialEntries={[initialEntry]} future={{v7_startTransition: true, v7_relativeSplatPath: true}}>
                <LocationProbe />
                <Routes>
                    <Route path='/login' element={<LoginPage />} />
                    <Route path='/settings' element={<span>settings destination</span>} />
                    <Route path='/projects' element={<span>projects destination</span>} />
                </Routes>
            </MemoryRouter>
        );
    });
    return tree;
};

const currentLocation = (tree: renderer.ReactTestRenderer) => tree.root.findByProps({'data-testid': 'location'}).children.join('');

test('LoginPage redirects successful local login to settings and ignores return_url', async () => {
    jest.spyOn(services.users, 'get').mockResolvedValue(loggedOutUser);
    const login = jest.spyOn(services.users, 'login').mockResolvedValue({token: 'created'});

    const tree = await renderLoginRoute('/login?return_url=/projects');
    const form = tree.root.findByProps({layout: 'vertical'});

    await act(async () => {
        await form.props.onFinish({username: 'admin', password: 'password'});
    });

    expect(login).toHaveBeenCalledWith('admin', 'password');
    expect(currentLocation(tree)).toBe('/settings');
});

test('LoginPage redirects already logged-in users to settings', async () => {
    jest.spyOn(services.users, 'get').mockResolvedValue(user('admin'));

    const tree = await renderLoginRoute('/login');

    expect(currentLocation(tree)).toBe('/settings');
});

test('LoginPage keeps logged-out users on login form', async () => {
    jest.spyOn(services.users, 'get').mockResolvedValue(loggedOutUser);

    const tree = await renderLoginRoute('/login');

    expect(currentLocation(tree)).toBe('/login');
    expect(tree.root.findAllByProps({htmlType: 'submit'})).toHaveLength(1);
});

test('visibleAccountsForUser returns all accounts for admin', () => {
    expect(visibleAccountsForUser(accounts, user('admin')).map(account => account.name)).toEqual(['admin', 'LINGJIE', 'YUDIAN']);
});

test('visibleAccountsForUser returns self and admin for regular users', () => {
    expect(visibleAccountsForUser(accounts, user('LINGJIE')).map(account => account.name)).toEqual(['admin', 'LINGJIE']);
});

test('visibleAccountsForUser returns admin for users without a local account', () => {
    expect(visibleAccountsForUser(accounts, user('sso@example.com')).map(account => account.name)).toEqual(['admin']);
});

test('notificationTestTopics contains the three stable test topics', () => {
    expect(notificationTestTopics).toEqual([
        {topic: 'token', label: '[TOKEN] 代币通知'},
        {topic: 'poly-mover', label: '[POLY] 市场异动'},
        {topic: 'poly-kickoff', label: '[POLY] 开赛通知'}
    ]);
});

const wormMarket = (overrides: Partial<WormMarketItem> = {}): WormMarketItem => ({
    conditionId: 'condition-1',
    title: 'Market title',
    logo: '',
    lastTradePrice: '0.42',
    state: 'open',
    category: 'sports',
    eventTitle: 'Event title',
    eventLogo: '',
    marginEnabled: false,
    ...overrides
});

const imageSources = (tree: renderer.ReactTestRendererJSON | renderer.ReactTestRendererJSON[] | null): string[] => {
    if (!tree) {
        return [];
    }
    const nodes = Array.isArray(tree) ? tree : [tree];
    return nodes.flatMap(node => {
        const own = node.type === 'img' ? [String(node.props.src || '')] : [];
        const children = (node.children || []).filter((child): child is renderer.ReactTestRendererJSON => typeof child !== 'string');
        return [...own, ...imageSources(children)];
    });
};

test('WormMarketSummary prefers market logo', () => {
    const tree = renderer.create(<WormMarketSummary item={wormMarket({logo: 'https://cdn.example/market.png', eventLogo: 'https://cdn.example/event.png'})} />).toJSON();
    expect(imageSources(tree)).toEqual(['https://cdn.example/market.png']);
});

test('WormMarketSummary falls back to event logo', () => {
    const tree = renderer.create(<WormMarketSummary item={wormMarket({eventLogo: 'https://cdn.example/event.png'})} />).toJSON();
    expect(imageSources(tree)).toEqual(['https://cdn.example/event.png']);
});

test('WormMarketSummary omits image when no logo is available', () => {
    const tree = renderer.create(<WormMarketSummary item={wormMarket()} />).toJSON();
    expect(imageSources(tree)).toEqual([]);
});
