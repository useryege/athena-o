import type {WormExecutionPreflightChecks} from '../../shared/services/worm-trading-service';

export type WormExecutionPreflightCheckKey = keyof WormExecutionPreflightChecks;

export interface WormExecutionPreflightCheckDefinition {
    key: WormExecutionPreflightCheckKey;
    title: string;
    description: string;
    ignoredDescription: string;
}

export interface WormExecutionMandatoryGuardDefinition {
    key: string;
    title: string;
    description: string;
}

export const wormExecutionMandatoryGuardDefinitions: WormExecutionMandatoryGuardDefinition[] = [
    {
        key: 'target-market-open-position',
        title: 'Target-market open position',
        description: 'Any YES or NO open position in the target market skips this order and every remaining order for the Wallet.'
    },
    {
        key: 'wallet-wide-in-flight-request',
        title: 'Wallet-wide in-flight request',
        description:
            'Any uncovered in-flight market or limit request, across every market and direction, skips this order and every remaining order for the Wallet. Requests linked from an observed open position are already covered.'
    }
];

export const wormExecutionPreflightCheckDefinitions: WormExecutionPreflightCheckDefinition[] = [
    {
        key: 'requireFullLiquidity',
        title: 'Full liquidity',
        description: 'Skip the order when Worm estimates that its fixed funds cannot be fully filled at 1×.',
        ignoredDescription: 'Allow the 1× order attempt when Worm estimates only a partial fill.'
    }
];

export const createDefaultWormExecutionPreflightChecks = (): WormExecutionPreflightChecks => ({
    requireFullLiquidity: true
});

export const disabledWormExecutionPreflightChecks = (checks: WormExecutionPreflightChecks) => wormExecutionPreflightCheckDefinitions.filter(definition => !checks[definition.key]);
