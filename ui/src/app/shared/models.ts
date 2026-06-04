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
    groups: string[];
}

export interface Token {
    id: string;
    issuedAt: number;
    expiresAt: number;
}

export interface Account {
    name: string;
    enabled: boolean;
    capabilities: string[];
    tokens: Token[];
}
