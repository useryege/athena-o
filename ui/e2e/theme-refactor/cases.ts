import marketCases from './fixtures/markets.json';
import sportsCases from './fixtures/sports.json';
import managedOOCases from './fixtures/managed-oo.json';
import walletCases from './fixtures/wallets.json';
import solanaCases from './fixtures/solana.json';
import governanceCases from './fixtures/governance.json';
import adminCases from './fixtures/admin-operations.json';
import identityCases from './fixtures/identity.json';
import type {ThemeCase, ThemeReply} from './contracts';

const settings = {
    url: 'http://127.0.0.1',
    statusBadgeEnabled: false,
    statusBadgeRootUrl: '',
    googleAnalytics: {trackingID: '', anonymizeUsers: false},
    help: {chatUrl: '', chatText: '', binaryUrls: {}},
    userLoginsDisabled: false,
    kustomizeVersions: [],
    uiCssURL: '',
    uiBannerContent: '',
    uiBannerURL: '',
    uiBannerPermanent: false,
    uiBannerPosition: '',
    execEnabled: false,
    appsInAnyNamespaceEnabled: false,
    hydratorEnabled: false,
    syncWithReplaceAllowed: false
};

const moduleAccess = (active: boolean) => [
    {
        module: 'ACCOUNT_DATA_MODULE_WALLET',
        dataAccess: active ? 'ACCOUNT_DATA_ACCESS_READ' : 'ACCOUNT_DATA_ACCESS_NONE'
    }
];

const user = (options: {administrator?: boolean; active?: boolean} = {}) => ({
    loggedIn: true,
    accountId: options.administrator ? '22222222-2222-4222-8222-222222222222' : '11111111-1111-4111-8111-111111111111',
    username: options.administrator ? 'fixture.admin' : 'fixture.member',
    iss: 'fixture',
    administrator: Boolean(options.administrator),
    profile: {displayName: options.administrator ? 'Fixture Admin' : 'Fixture Member', tier: 'ACCOUNT_TIER_STANDARD', avatarUrl: '', revision: 4},
    identity: {
        provider: 'ACCOUNT_IDENTITY_PROVIDER_GOOGLE',
        verifiedEmail: options.administrator ? 'admin@example.invalid' : 'member@example.invalid',
        solanaAddress: '',
        createdAt: 1783999440,
        lastLoginAt: 1789351200
    },
    access: {loginEnabled: true, apiKeyEnabled: false, profitSharingEnabled: false, revision: 7, moduleAccess: moduleAccess(Boolean(options.active))}
});

const bootstrap = (realm: 'member' | 'admin', session: unknown, status = 200): ThemeReply => ({
    method: 'GET',
    path: '/api/v1/app/bootstrap',
    realm,
    status,
    json: status === 200 ? {settings, session} : session
});

const authenticated = (options: {administrator?: boolean; active?: boolean} = {}) => ({
    status: 'APP_BOOTSTRAP_SESSION_STATUS_AUTHENTICATED',
    userInfo: user(options)
});

export const themeCases: ThemeCase[] = [
    ...(marketCases as ThemeCase[]),
    ...(sportsCases as ThemeCase[]),
    ...(managedOOCases as ThemeCase[]),
    ...(identityCases as ThemeCase[]),
    ...(walletCases as ThemeCase[]),
    ...(solanaCases as ThemeCase[]),
    ...(governanceCases as ThemeCase[]),
    ...(adminCases as ThemeCase[]),
    {
        id: 'member-bootstrap-error',
        route: '/login',
        realm: 'member',
        heading: 'API 服务暂不可用',
        replies: [bootstrap('member', {error: {code: 14, message: 'controlled bootstrap unavailable'}}, 503)]
    },
    {id: 'member-pending', route: '/account/access', realm: 'member', heading: 'Account Center', replies: [bootstrap('member', authenticated())]},
    {id: 'member-shell', route: '/help', realm: 'member', heading: 'Help', replies: [bootstrap('member', authenticated({active: true}))]},
    {id: 'admin-shell', route: '/admin/help', realm: 'admin', heading: 'Help', replies: [bootstrap('admin', authenticated({administrator: true, active: true}))]},
    {id: 'admin-forbidden', route: '/admin/accounts', realm: 'admin', heading: 'Administrator access required', replies: [bootstrap('admin', authenticated())]}
];
