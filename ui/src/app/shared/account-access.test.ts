import {parseAccountAccess} from './models';
import {AccountDataAccess, AccountDataModule} from './access-modules';
import {cloneAccountAccess, accountAccessEqual, moduleAccessLevel, replaceModuleAccess} from './account-access';
import {AdminAccountsService} from '../admin/accounts-service';
import requests from './services/requests';

afterEach(() => jest.restoreAllMocks());
test('parse, edit, clone and submit retain seven retained grants including hidden Token', async () => {
    const original = parseAccountAccess({revision: 7, loginEnabled: true, moduleAccess: [
        {module: 'trader_sync', dataAccess: 'read_write'},
        {module: 'solana', dataAccess: 'read'},
        {module: 'token', dataAccess: 'read'}
    ]});
    const edited = replaceModuleAccess(original, AccountDataModule.TraderSync, AccountDataAccess.None);
    expect(accountAccessEqual(original, edited)).toBe(false);
    expect(accountAccessEqual(edited, cloneAccountAccess(edited))).toBe(true);
    expect(moduleAccessLevel(original, AccountDataModule.TraderSync)).toBe(2);
    let payload: any;
    jest.spyOn(requests, 'put').mockImplementation((() => ({send: (value: any) => {
        payload = value;
        return Promise.resolve({body: {id: 'owner', access: value}});
    }})) as any);
    const saved = await new AdminAccountsService().updateAccess('owner', edited);
    expect(payload.moduleAccess).toHaveLength(7);
    expect(payload.moduleAccess.map((item: {module: number}) => item.module).sort((a: number, b: number) => a - b)).toEqual([1, 4, 8, 9, 11, 12, 13]);
    expect(payload.moduleAccess.find((item: any) => item.module === 12)).toEqual({module: 12, dataAccess: 0});
    expect(payload.moduleAccess.find((item: any) => item.module === 13)).toEqual({module: 13, dataAccess: 1});
    expect(payload.moduleAccess.find((item: any) => item.module === 8)).toEqual({module: 8, dataAccess: 1});
    expect(accountAccessEqual(saved.access, edited)).toBe(true);
    expect(moduleAccessLevel(replaceModuleAccess(original, AccountDataModule.TraderSync, AccountDataAccess.Read), AccountDataModule.TraderSync)).toBe(0);
});
