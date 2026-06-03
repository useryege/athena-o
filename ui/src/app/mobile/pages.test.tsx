import {Account, UserInfo} from '../shared/models';
import {visibleAccountsForUser} from './pages';

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
