import {AccountAccess} from './models';
import {AccountDataAccess, AccountDataModule, accountAccessDisplayModules, accountDataModuleDefinition, accountDataModules, normalizeModuleGrant} from './access-modules';

export type ModuleAccessLevels = Record<AccountDataModule, AccountDataAccess>;

export const moduleAccessLevels = (access?: AccountAccess): ModuleAccessLevels => {
    const levels = {} as ModuleAccessLevels;
    const source = new Map((access?.moduleAccess || []).map(item => [item.module, item.dataAccess]));
    accountDataModules.forEach(definition => {
        const effective = source.get(definition.module) || AccountDataAccess.None;
        levels[definition.module] = normalizeModuleGrant(definition, effective);
    });
    return levels;
};

export const moduleAccessLevel = (access: AccountAccess | undefined, module: AccountDataModule): AccountDataAccess => {
    const definition = accountDataModuleDefinition(module);
    const effective = access?.moduleAccess.find(item => item.module === module)?.dataAccess || AccountDataAccess.None;
    return definition ? normalizeModuleGrant(definition, effective) : AccountDataAccess.None;
};

export const replaceModuleAccess = (access: AccountAccess, module: AccountDataModule, dataAccess: AccountDataAccess): AccountAccess => ({
    ...access,
    moduleAccess: accountDataModules.map(definition => ({
        module: definition.module,
        dataAccess: definition.module === module ? normalizeModuleGrant(definition, dataAccess) : moduleAccessLevel(access, definition.module)
    }))
});

export const cloneAccountAccess = (access: AccountAccess): AccountAccess => ({
    loginEnabled: access.loginEnabled,
    apiKeyEnabled: access.apiKeyEnabled,
    profitSharingEnabled: access.profitSharingEnabled,
    revision: access.revision,
    moduleAccess: accountDataModules.map(definition => ({
        module: definition.module,
        dataAccess: moduleAccessLevel(access, definition.module)
    }))
});

export const accountAccessEqual = (left?: AccountAccess, right?: AccountAccess): boolean =>
    Boolean(left && right) &&
    left?.loginEnabled === right?.loginEnabled &&
    left?.apiKeyEnabled === right?.apiKeyEnabled &&
    left?.profitSharingEnabled === right?.profitSharingEnabled &&
    accountDataModules.every(definition => moduleAccessLevel(left, definition.module) === moduleAccessLevel(right, definition.module));

export const moduleAccessSummary = (access: AccountAccess | undefined): string => {
    let readWrite = 0;
    let read = 0;
    let none = 0;
    accountAccessDisplayModules.forEach(definition => {
        switch (moduleAccessLevel(access, definition.module)) {
            case AccountDataAccess.ReadWrite:
                readWrite++;
                break;
            case AccountDataAccess.Read:
                read++;
                break;
            default:
                none++;
        }
    });
    return `${readWrite} full · ${read} read · ${none} none`;
};

export const moduleAccessLevelsEqual = (left: ModuleAccessLevels, right: ModuleAccessLevels): boolean =>
    accountDataModules.every(definition => left[definition.module] === right[definition.module]);
