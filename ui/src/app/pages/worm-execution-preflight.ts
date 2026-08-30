import type {WormExecutionPreflightChecks} from '../shared/services/worm-trading-service';

export type WormExecutionPreflightCheckKey = keyof WormExecutionPreflightChecks;

export interface WormExecutionPreflightCheckDefinition {
    key: WormExecutionPreflightCheckKey;
    title: string;
    description: string;
    ignoredDescription: string;
}

export const wormExecutionPreflightCheckDefinitions: WormExecutionPreflightCheckDefinition[] = [
    {
        key: 'skipAlreadyHeld',
        title: 'Existing same-side position',
        description: 'Skip a market when the Wallet already holds the selected side.',
        ignoredDescription: 'Another purchase may be attempted even when the selected side is already held.'
    },
    {
        key: 'skipInFlightRequest',
        title: 'Same-side request in flight',
        description: 'Skip a market when a request for the selected side is still in flight.',
        ignoredDescription: 'Another request may be opened while a same-side request is still in flight.'
    },
    {
        key: 'skipOppositeSideExposure',
        title: 'Opposite-side exposure',
        description: 'Skip a market when the Wallet has an opposite-side position or request.',
        ignoredDescription: 'The selected side may be purchased despite an opposite-side position or request.'
    },
    {
        key: 'requireFullLiquidity',
        title: 'Full estimated liquidity',
        description: 'Skip a market when Worm estimates that the requested funds cannot be fully filled.',
        ignoredDescription: 'The order may be attempted even when Worm reports that it cannot be fully filled.'
    }
];

export const createDefaultWormExecutionPreflightChecks = (): WormExecutionPreflightChecks => ({
    skipAlreadyHeld: true,
    skipInFlightRequest: true,
    skipOppositeSideExposure: true,
    requireFullLiquidity: true
});

export const disabledWormExecutionPreflightChecks = (checks: WormExecutionPreflightChecks) => wormExecutionPreflightCheckDefinitions.filter(definition => !checks[definition.key]);
