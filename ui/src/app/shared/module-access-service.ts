import requests from './services/requests';
import type {AbortablePromise} from './use-visible-query';

export const moduleAccessDefinitions = [
    {key: 'trader_sync', label: 'Trader Sync'},
    {key: 'solana', label: 'Solana'},
    {key: 'market_radar', label: 'Market Radar'},
    {key: 'managed_oo', label: 'Managed OO'},
    {key: 'profit_sharing', label: 'Profit Sharing'},
    {key: 'worm', label: 'Worm'}
] as const;
export type ModuleKey = (typeof moduleAccessDefinitions)[number]['key'];
export interface ModuleAccessState {
    module_key: ModuleKey;
    state: 1 | 2;
}
export interface ModuleAccessSetting extends ModuleAccessState {
    updated_by_account_id?: string;
    updated_by_username?: string;
    updated_at?: string;
}
export const isModuleKey = (value: unknown): value is ModuleKey => moduleAccessDefinitions.some(item => item.key === value);
export const isModuleAccessReason = (value: unknown) => value === 'MODULE_ACCESS_CLOSED' || value === 'MODULE_ACCESS_UNAVAILABLE';

const parseRows = <T extends ModuleAccessState>(value: unknown): T[] => {
    if (
        !Array.isArray(value) ||
        value.length !== moduleAccessDefinitions.length ||
        value.some(row => !row || !isModuleKey(row.module_key) || (row.state !== 1 && row.state !== 2)) ||
        new Set(value.map(row => row.module_key)).size !== moduleAccessDefinitions.length
    ) {
        throw new Error('Module access could not be confirmed: invalid response');
    }
    return value;
};
const read = <T>(request: ReturnType<typeof requests.get>, parse: (body: any) => T): AbortablePromise<T> => {
    const result = request.then(response => parse(response.body)) as AbortablePromise<T>;
    result.abort = () => request.abort();
    return result;
};
export const moduleAccessService = {
    states: () => read(requests.get('/module-access-states', {session: true}), body => parseRows<ModuleAccessState>(body.states)),
    settings: () => read(requests.get('/admin/module-access-settings', {feature: 'admin-service-status', mode: 'read'}), body => parseRows<ModuleAccessSetting>(body.settings)),
    save: (key: ModuleKey, state: 1 | 2) =>
        read(requests.put(`/admin/module-access-settings/${key}`, {feature: 'admin-service-status', mode: 'write'}).send({state}), body => {
            const row = body.setting;
            if (!row || row.module_key !== key || row.state !== state) throw new Error('The saved module access setting could not be confirmed');
            return row as ModuleAccessSetting;
        })
};
