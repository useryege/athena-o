import {AccountDataAccess, AccountDataModule, accountDataModules, parseAccountDataAccess, parseAccountDataModule} from './access-modules';

export interface VersionMessage {
    Version: string;
    BuildDate: string;
    GoVersion: string;
    Compiler: string;
    Platform: string;
    KustomizeVersion: string;
    HelmVersion: string;
    KubectlVersion: string;
    JsonnetVersion: string;
}

export interface Plugin {
    name: string;
}

export interface AuthSettings {
    url: string;
    statusBadgeEnabled: boolean;
    statusBadgeRootUrl: string;
    googleAnalytics: {
        trackingID: string;
        anonymizeUsers: boolean;
    };
    help: {
        chatUrl: string;
        chatText: string;
        binaryUrls: Record<string, string>;
    };
    userLoginsDisabled: boolean;
    kustomizeVersions: string[];
    uiCssURL: string;
    uiBannerContent: string;
    uiBannerURL: string;
    uiBannerPermanent: boolean;
    uiBannerPosition: string;
    execEnabled: boolean;
    appsInAnyNamespaceEnabled: boolean;
    hydratorEnabled: boolean;
    syncWithReplaceAllowed: boolean;
}

export {AccountDataAccess, AccountDataModule, parseAccountDataAccess, parseAccountDataModule} from './access-modules';

export interface UserInfo {
    loggedIn: boolean;
    username: string;
    iss: string;
    administrator: boolean;
    access: AccountAccess;
}

export enum AppBootstrapSessionStatus {
    Anonymous = 'APP_BOOTSTRAP_SESSION_STATUS_ANONYMOUS',
    Authenticated = 'APP_BOOTSTRAP_SESSION_STATUS_AUTHENTICATED',
    AccountMaintenance = 'APP_BOOTSTRAP_SESSION_STATUS_ACCOUNT_MAINTENANCE'
}

export const parseAppBootstrapSessionStatus = (value: unknown): AppBootstrapSessionStatus => {
    switch (value) {
        case 1:
        case AppBootstrapSessionStatus.Anonymous:
            return AppBootstrapSessionStatus.Anonymous;
        case 2:
        case AppBootstrapSessionStatus.Authenticated:
            return AppBootstrapSessionStatus.Authenticated;
        case 3:
        case AppBootstrapSessionStatus.AccountMaintenance:
            return AppBootstrapSessionStatus.AccountMaintenance;
        default:
            throw new Error(`Unsupported app bootstrap session status: ${String(value)}`);
    }
};

export interface AppBootstrapSession {
    status: AppBootstrapSessionStatus;
    userInfo?: UserInfo;
}

export interface AppBootstrap {
    settings: AuthSettings;
    session: AppBootstrapSession;
}

export const parseUserInfo = (value: any): UserInfo => ({
    loggedIn: Boolean(value?.loggedIn),
    username: value?.username || '',
    iss: value?.iss || '',
    administrator: Boolean(value?.administrator),
    access: parseAccountAccess(value?.access)
});

export interface Token {
    id: string;
    issuedAt: number;
    expiresAt: number;
}

export interface Account {
    name: string;
    administrator: boolean;
    access: AccountAccess;
    capabilities: string[];
    tokens: Token[];
}

export interface AccountAccess {
    loginEnabled: boolean;
    revision: number;
    moduleAccess: AccountModuleAccess[];
}

export interface AccountModuleAccess {
    module: AccountDataModule;
    dataAccess: AccountDataAccess;
}

export const parseAccountAccess = (value: any): AccountAccess => {
    const values = Array.isArray(value?.moduleAccess) ? value.moduleAccess : Array.isArray(value?.module_access) ? value.module_access : [];
    const parsed = new Map<AccountDataModule, AccountDataAccess>();
    values.forEach((item: any) => {
        const module = parseAccountDataModule(item?.module);
        if (module !== undefined) {
            parsed.set(module, parseAccountDataAccess(item?.dataAccess ?? item?.data_access));
        }
    });
    return {
        loginEnabled: Boolean(value?.loginEnabled ?? value?.login_enabled),
        revision: Number(value?.revision || 0),
        moduleAccess: accountDataModules.map(definition => ({
            module: definition.module,
            dataAccess: Math.min(parsed.get(definition.module) || AccountDataAccess.None, definition.maxAccess) as AccountDataAccess
        }))
    };
};
