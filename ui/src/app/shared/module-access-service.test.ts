import {moduleAccessService} from './module-access-service';
import requests from './services/requests';
const rows = ['trader_sync', 'solana', 'market_radar', 'managed_oo', 'profit_sharing', 'worm'].map(module_key => ({module_key, state: 1}));
afterEach(() => jest.restoreAllMocks());
test.each(
    [[], rows.slice(1), rows.map(row => ({...row, state: 'MODULE_ACCESS_STATE_OPEN'})), [...rows.slice(1), rows[1]], rows.map(row => ({moduleKey: row.module_key, state: 1}))].map(
        states => [states]
    )
)('malformed or ambiguous wire states fail closed', async states => {
    jest.spyOn(requests, 'get').mockReturnValue({then: (fn: any) => Promise.resolve({body: {states}}).then(fn), abort: jest.fn()} as any);
    await expect(moduleAccessService.states()).rejects.toThrow('could not be confirmed');
});
test('state transport preserves the six snake_case numeric settings and cancellation', async () => {
    const abort = jest.fn();
    const get = jest.spyOn(requests, 'get').mockReturnValue({then: (fn: any) => Promise.resolve({body: {states: rows}}).then(fn), abort} as any);
    const response = moduleAccessService.states();
    await expect(response).resolves.toEqual(rows);
    response.abort?.();
    expect(get).toHaveBeenCalledWith('/module-access-states', {session: true});
    expect(abort).toHaveBeenCalledTimes(1);
});
test('saving uses one explicit numeric target and returns server attribution', async () => {
    const setting = {module_key: 'worm', state: 2, updated_by_username: 'operator', updated_at: '2026-09-17T07:08:09.123456789Z'};
    const send = jest.fn().mockReturnValue({then: (fn: any) => Promise.resolve({body: {setting}}).then(fn), abort: jest.fn()});
    const put = jest.spyOn(requests, 'put').mockReturnValue({send} as any);
    await expect(moduleAccessService.save('worm', 2)).resolves.toEqual(setting);
    expect(put).toHaveBeenCalledWith('/admin/module-access-settings/worm', {feature: 'admin-service-status', mode: 'write'});
    expect(send).toHaveBeenCalledWith({state: 2});
});
