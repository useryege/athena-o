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

export interface UserInfo {
    loggedIn: boolean;
    username: string;
    iss: string;
    administrator: boolean;
    dataAccess: AccountDataAccess;
    authorizationRevision: number;
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

export enum AccountDataAccess {
    None = 0,
    Read = 1,
    ReadWrite = 2
}

export const parseAccountDataAccess = (value: unknown): AccountDataAccess => {
    if (value === AccountDataAccess.Read || value === 'ACCOUNT_DATA_ACCESS_READ' || value === 'read') {
        return AccountDataAccess.Read;
    }
    if (value === AccountDataAccess.ReadWrite || value === 'ACCOUNT_DATA_ACCESS_READ_WRITE' || value === 'read_write') {
        return AccountDataAccess.ReadWrite;
    }
    return AccountDataAccess.None;
};

export const parseUserInfo = (value: any): UserInfo => ({
    loggedIn: Boolean(value?.loggedIn),
    username: value?.username || '',
    iss: value?.iss || '',
    administrator: Boolean(value?.administrator),
    dataAccess: parseAccountDataAccess(value?.dataAccess),
    authorizationRevision: Number(value?.authorizationRevision || 0)
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
    dataAccess: AccountDataAccess;
    revision: number;
}
