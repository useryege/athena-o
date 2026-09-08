export enum AccountDataAccess {
    None = 0,
    Read = 1,
    ReadWrite = 2
}

export enum AccountDataModule {
    MarketRadar = 1,
    SportsLive = 2,
    SportsHistory = 3,
    ManagedOO = 4,
    WormMarkets = 5,
    WorldCupCorners = 7,
    Token = 8,
    Wallet = 9,
    WormTrading = 11
}

export type AccountDataModuleGroup = 'markets' | 'token-risk';

export interface AccountDataModuleDefinition {
    module: AccountDataModule;
    id: string;
    label: string;
    description: string;
    group: AccountDataModuleGroup;
    maxAccess: AccountDataAccess.Read | AccountDataAccess.ReadWrite;
    apiOnly?: boolean;
}

export const accountDataModuleGroups: Array<{key: AccountDataModuleGroup; label: string}> = [
    {key: 'markets', label: 'Markets'},
    {key: 'token-risk', label: 'Token & Risk'}
];

export const accountDataModules: AccountDataModuleDefinition[] = [
    {
        module: AccountDataModule.MarketRadar,
        id: 'market_radar',
        label: 'Market Radar',
        description: 'Hot, realtime, and moving Polymarket discovery views.',
        group: 'markets',
        maxAccess: AccountDataAccess.Read
    },
    {
        module: AccountDataModule.SportsLive,
        id: 'sports_live',
        label: 'Sports Live',
        description: 'Current sports events and live price histories.',
        group: 'markets',
        maxAccess: AccountDataAccess.Read
    },
    {
        module: AccountDataModule.SportsHistory,
        id: 'sports_history',
        label: 'Sports History',
        description: 'Completed sports history, sync state, and manual refresh.',
        group: 'markets',
        maxAccess: AccountDataAccess.ReadWrite
    },
    {
        module: AccountDataModule.ManagedOO,
        id: 'managed_oo',
        label: 'Managed OO',
        description: 'Managed Optimistic Oracle proposals, disputes, and block parsing.',
        group: 'markets',
        maxAccess: AccountDataAccess.ReadWrite
    },
    {
        module: AccountDataModule.WormMarkets,
        id: 'worm_markets',
        label: 'Worm Markets',
        description: 'Standalone Worm market data APIs.',
        group: 'markets',
        maxAccess: AccountDataAccess.Read,
        apiOnly: true
    },
    {
        module: AccountDataModule.WormTrading,
        id: 'worm_trading',
        label: 'Worm Trading',
        description: 'Owner-scoped Solana balances and Worm trading operations.',
        group: 'markets',
        maxAccess: AccountDataAccess.ReadWrite
    },
    {
        module: AccountDataModule.WorldCupCorners,
        id: 'world_cup_corners',
        label: 'World Cup Corners',
        description: 'Protected World Cup corner and result dataset.',
        group: 'markets',
        maxAccess: AccountDataAccess.Read
    },
    {
        module: AccountDataModule.Token,
        id: 'token',
        label: 'Token',
        description: 'Token discovery, one-time collection, project profiles, and chain operations.',
        group: 'token-risk',
        maxAccess: AccountDataAccess.ReadWrite
    },
    {
        module: AccountDataModule.Wallet,
        id: 'wallet',
        label: 'Wallet',
        description: 'Wallet inventory, creation, import, aliases, and sensitive secret access.',
        group: 'token-risk',
        maxAccess: AccountDataAccess.ReadWrite
    }
];

// Presentation only: authorization parsing and updates retain the complete module matrix.
export const accountAccessDisplayModules = accountDataModules.filter(definition => definition.module !== AccountDataModule.Token);

const moduleByID = new Map(accountDataModules.map(definition => [definition.id, definition.module]));
const moduleByEnumName = new Map(accountDataModules.map(definition => [`ACCOUNT_DATA_MODULE_${definition.id.toUpperCase()}`, definition.module]));

export const parseAccountDataAccess = (value: unknown): AccountDataAccess => {
    if (value === AccountDataAccess.Read || value === 'ACCOUNT_DATA_ACCESS_READ' || value === 'read') {
        return AccountDataAccess.Read;
    }
    if (value === AccountDataAccess.ReadWrite || value === 'ACCOUNT_DATA_ACCESS_READ_WRITE' || value === 'read_write') {
        return AccountDataAccess.ReadWrite;
    }
    return AccountDataAccess.None;
};

export const parseAccountDataModule = (value: unknown): AccountDataModule | undefined => {
    const numeric = Number(value);
    if (Number.isInteger(numeric) && accountDataModules.some(definition => definition.module === numeric)) {
        return numeric as AccountDataModule;
    }
    if (typeof value !== 'string') {
        return undefined;
    }
    return moduleByID.get(value.toLowerCase()) || moduleByEnumName.get(value.toUpperCase());
};

export const accountDataAccessLabel = (value: AccountDataAccess): string => {
    switch (value) {
        case AccountDataAccess.ReadWrite:
            return 'Read & write';
        case AccountDataAccess.Read:
            return 'Read only';
        default:
            return 'No access';
    }
};

export const accountDataModuleDefinition = (module: AccountDataModule): AccountDataModuleDefinition | undefined =>
    accountDataModules.find(definition => definition.module === module);
