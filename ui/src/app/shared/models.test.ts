import {parseAccountAccess} from './models';
test('invalid Trader Sync read grant fails closed while a full grant survives parsing', () => {
    for (const [input, expected] of [['read', 0], ['read_write', 2]]) {
        const access = parseAccountAccess({moduleAccess: [{module: 'trader_sync', dataAccess: input}]});
        expect(access.moduleAccess).toHaveLength(10);
        expect(access.moduleAccess.find(item => item.module === 12)?.dataAccess).toBe(expected);
    }
});
