import {AccountDataAccess, accountDataModules, allowedAccessLevels} from './access-modules';
test('Trader Sync can only grant no access or full access', () => {
    const definition = accountDataModules.find(item => item.id === 'trader_sync');
    expect(definition).toBeDefined();
    expect(allowedAccessLevels(definition!)).toEqual([AccountDataAccess.None, AccountDataAccess.ReadWrite]);
});
