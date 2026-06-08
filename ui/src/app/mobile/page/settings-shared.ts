import {Account, UserInfo} from '../../shared/models';

export const visibleAccountsForUser = (accounts: Account[], user?: UserInfo): Account[] => {
    if (user?.username === 'admin') {
        return accounts;
    }
    return accounts.filter(account => account.name === 'admin' || account.name === user?.username);
};
