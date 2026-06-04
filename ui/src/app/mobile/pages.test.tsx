import {Account, UserInfo} from '../shared/models';
import * as React from 'react';
import renderer from 'react-test-renderer';
import {WormMarketSummary, notificationTestTopics, visibleAccountsForUser} from './pages';
import {WormMarketItem} from '../shared/services/worm-service';

const accounts: Account[] = [
    {name: 'admin', enabled: true, capabilities: ['login'], tokens: []},
    {name: 'LINGJIE', enabled: true, capabilities: ['login'], tokens: []},
    {name: 'YUDIAN', enabled: true, capabilities: ['login'], tokens: []}
];

const user = (username: string): UserInfo => ({loggedIn: true, username, iss: 'athena', groups: []});

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
