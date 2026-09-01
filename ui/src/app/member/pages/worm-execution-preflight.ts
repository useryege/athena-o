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
