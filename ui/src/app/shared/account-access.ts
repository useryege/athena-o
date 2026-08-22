import {AccountAccess} from './models';
import {AccountDataAccess, AccountDataModule, accountDataModuleDefinition, accountDataModules} from './access-modules';

export type ModuleAccessLevels = Record<AccountDataModule, AccountDataAccess>;

export const moduleAccessLevels = (access?: AccountAccess, administrator = false): ModuleAccessLevels => {
    const levels = {} as ModuleAccessLevels;
    const source = new Map((access?.moduleAccess || []).map(item => [item.module, item.dataAccess]));
    accountDataModules.forEach(definition => {
        const effective = administrator ? definition.maxAccess : source.get(definition.module) || AccountDataAccess.None;
        levels[definition.module] = Math.min(effective, definition.maxAccess) as AccountDataAccess;
    });
    return levels;
};

export const moduleAccessLevel = (access: AccountAccess | undefined, module: AccountDataModule, administrator = false): AccountDataAccess => {
    const maximum = accountDataModuleDefinition(module)?.maxAccess || AccountDataAccess.None;
    const effective = administrator ? maximum : access?.moduleAccess.find(item => item.module === module)?.dataAccess || AccountDataAccess.None;
    return Math.min(effective, maximum) as AccountDataAccess;
};

export const replaceModuleAccess = (access: AccountAccess, module: AccountDataModule, dataAccess: AccountDataAccess): AccountAccess => ({
    ...access,
    moduleAccess: accountDataModules.map(definition => ({
        module: definition.module,
        dataAccess: definition.module === module ? dataAccess : moduleAccessLevel(access, definition.module)
    }))
});

export const cloneAccountAccess = (access: AccountAccess): AccountAccess => ({
    loginEnabled: access.loginEnabled,
    revision: access.revision,
    moduleAccess: accountDataModules.map(definition => ({
        module: definition.module,
        dataAccess: moduleAccessLevel(access, definition.module)
    }))
});

export const accountAccessEqual = (left?: AccountAccess, right?: AccountAccess): boolean =>
    Boolean(left && right) &&
    left?.loginEnabled === right?.loginEnabled &&
    accountDataModules.every(definition => moduleAccessLevel(left, definition.module) === moduleAccessLevel(right, definition.module));

export const moduleAccessSummary = (access: AccountAccess | undefined, administrator = false): string => {
    if (administrator) {
        return 'Full access';
    }
    let readWrite = 0;
    let read = 0;
    let none = 0;
    accountDataModules.forEach(definition => {
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
