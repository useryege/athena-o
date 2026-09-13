import {AccountDataAccess, AccountDataModule, accountAccessDisplayModules, accountDataModules, allowedAccessLevels, parseAccountDataModule} from './access-modules';
test('Trader Sync can only grant no access or full access', () => {
    const definition = accountDataModules.find(item => item.id === 'trader_sync');
    expect(definition).toBeDefined();
    expect(allowedAccessLevels(definition!)).toEqual([AccountDataAccess.None, AccountDataAccess.ReadWrite]);
});

test('Solana is a selectable read-only module in the complete access matrix', () => {
    const definition = accountDataModules.find(item => item.id === 'solana');
    expect(definition).toMatchObject({module: AccountDataModule.Solana, group: 'token-risk', maxAccess: AccountDataAccess.Read});
    expect(accountAccessDisplayModules).toContain(definition);
    expect(allowedAccessLevels(definition!)).toEqual([AccountDataAccess.None, AccountDataAccess.Read]);
    expect(parseAccountDataModule('ACCOUNT_DATA_MODULE_SOLANA')).toBe(AccountDataModule.Solana);
});
