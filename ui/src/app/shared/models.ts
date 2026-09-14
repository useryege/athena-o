import {AccountDataAccess, AccountDataModule, accountDataModules, normalizeModuleGrant, parseAccountDataAccess, parseAccountDataModule} from './access-modules';

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
    accountId: string;
    username: string;
    iss: string;
    administrator: boolean;
    access: AccountAccess;
    identity: AccountIdentity;
    profile: AccountProfile;
}

export enum AccountTier {
    Standard = 'ACCOUNT_TIER_STANDARD',
    Pro = 'ACCOUNT_TIER_PRO'
}

export interface AccountProfile {
    displayName: string;
    tier: AccountTier;
    avatarUrl: string;
    revision: number;
}

export const parseAccountTier = (value: unknown): AccountTier => {
    switch (value) {
        case 2:
        case AccountTier.Pro:
        case 'pro':
            return AccountTier.Pro;
        default:
            return AccountTier.Standard;
    }
};

export const parseAccountProfile = (value: any, fallbackName = ''): AccountProfile => ({
    displayName: String(value?.displayName ?? value?.display_name ?? fallbackName),
    tier: parseAccountTier(value?.tier),
    avatarUrl: String(value?.avatarUrl ?? value?.avatar_url ?? ''),
    revision: Number(value?.revision || 0)
});

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
    accountId: String(value?.accountId ?? value?.account_id ?? ''),
    username: value?.username || '',
    iss: value?.iss || '',
    administrator: Boolean(value?.administrator),
    access: parseAccountAccess(value?.access),
    identity: parseAccountIdentity(value?.identity),
    profile: parseAccountProfile(value?.profile, value?.username || '')
});

export interface Token {
    id: string;
    issuedAt: number;
    expiresAt: number;
}

export interface Account {
    id: string;
    username: string;
    administrator: boolean;
    access: AccountAccess;
    profile: AccountProfile;
    identity: AccountIdentity;
    status: AccountStatus;
}

export interface AccountAccess {
    loginEnabled: boolean;
    apiKeyEnabled: boolean;
    profitSharingEnabled: boolean;
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
        apiKeyEnabled: Boolean(value?.apiKeyEnabled ?? value?.api_key_enabled),
        profitSharingEnabled: Boolean(value?.profitSharingEnabled ?? value?.profit_sharing_enabled),
        revision: Number(value?.revision || 0),
        moduleAccess: accountDataModules.map(definition => ({
            module: definition.module,
            dataAccess: normalizeModuleGrant(definition, parsed.get(definition.module) || AccountDataAccess.None)
        }))
    };
};

export enum AccountIdentityProvider {
    Unspecified = 'ACCOUNT_IDENTITY_PROVIDER_UNSPECIFIED',
    Google = 'ACCOUNT_IDENTITY_PROVIDER_GOOGLE',
    Development = 'ACCOUNT_IDENTITY_PROVIDER_DEVELOPMENT',
    SolanaWallet = 'ACCOUNT_IDENTITY_PROVIDER_SOLANA_WALLET'
}

export interface AccountIdentity {
    provider: AccountIdentityProvider;
    verifiedEmail: string;
    solanaAddress: string;
    createdAt: number;
    lastLoginAt: number;
}

export const parseAccountIdentityProvider = (value: unknown): AccountIdentityProvider => {
    switch (value) {
        case 1:
        case AccountIdentityProvider.Google:
        case 'google':
            return AccountIdentityProvider.Google;
        case 2:
        case AccountIdentityProvider.Development:
        case 'development':
            return AccountIdentityProvider.Development;
        case 3:
        case AccountIdentityProvider.SolanaWallet:
        case 'solana_wallet':
            return AccountIdentityProvider.SolanaWallet;
        default:
            return AccountIdentityProvider.Unspecified;
    }
};

export const parseAccountIdentity = (value: any): AccountIdentity => ({
    provider: parseAccountIdentityProvider(value?.provider),
    verifiedEmail: String(value?.verifiedEmail ?? value?.verified_email ?? ''),
    solanaAddress: String(value?.solanaAddress ?? value?.solana_address ?? ''),
    createdAt: Number(value?.createdAt ?? value?.created_at ?? 0),
    lastLoginAt: Number(value?.lastLoginAt ?? value?.last_login_at ?? 0)
});

export enum AccountStatus {
    Unspecified = 'ACCOUNT_STATUS_UNSPECIFIED',
    Pending = 'ACCOUNT_STATUS_PENDING',
    Active = 'ACCOUNT_STATUS_ACTIVE',
    Blocked = 'ACCOUNT_STATUS_BLOCKED'
}

export const parseAccountStatus = (value: unknown): AccountStatus => {
    switch (value) {
        case 1:
        case AccountStatus.Pending:
        case 'pending':
            return AccountStatus.Pending;
        case 2:
        case AccountStatus.Active:
        case 'active':
            return AccountStatus.Active;
        case 3:
        case AccountStatus.Blocked:
        case 'blocked':
            return AccountStatus.Blocked;
        default:
            return AccountStatus.Unspecified;
    }
};

export const accountStatusForAccess = (access: AccountAccess, administrator = false): AccountStatus => {
    if (!access.loginEnabled) {
        return AccountStatus.Blocked;
    }
    if (administrator || access.profitSharingEnabled || access.moduleAccess.some(item => item.dataAccess > AccountDataAccess.None)) {
        return AccountStatus.Active;
    }
    return AccountStatus.Pending;
};
