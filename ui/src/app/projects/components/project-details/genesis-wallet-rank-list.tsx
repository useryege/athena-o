import * as React from 'react';

import {AssetState, GenesisWalletAssetState, GenesisWalletState} from '../../../shared/services/athena-application-service';
import {formatUsdtValue} from '../pair-metrics-cell/pair-metrics-cell';

const renderValue = (value: string | number | undefined) => {
    if (value === undefined || value === '') {
        return '-';
    }
    return String(value);
};

const renderRatioFromBps = (ratioBps?: number) => {
    if (ratioBps === undefined || ratioBps === null || Number.isNaN(ratioBps)) {
        return '-';
    }
    return `${(ratioBps / 100).toFixed(2)}%`;
};

const buildAssetStateMap = (items?: GenesisWalletAssetState[]) => {
    const walletMap = new Map<string, AssetState | undefined>();
    (items || []).forEach(item => {
        const wallet = (item.wallet || '').toLowerCase();
        if (!wallet) {
            return;
        }
        walletMap.set(wallet, item.assetState);
    });
    return walletMap;
};

export const GenesisWalletRankList = ({
    genesisWallets,
    genesisWalletAssetStates,
    usdtDecimals
}: {
    genesisWallets?: GenesisWalletState[];
    genesisWalletAssetStates?: GenesisWalletAssetState[];
    usdtDecimals?: number | string;
}) => {
    if (!genesisWallets || genesisWallets.length === 0) {
        return <div className='project-details__field-value'>No genesis wallet metadata available</div>;
    }

    const assetStateByWallet = buildAssetStateMap(genesisWalletAssetStates);

    return (
        <div className='project-details__rank-list'>
            {genesisWallets.map((item, index) => {
                const assetState = assetStateByWallet.get((item.wallet || '').toLowerCase());
                const rank = item.rank ?? index;
                return (
                    <div key={`${item.wallet || ''}-${rank}`} className='project-details__rank-item'>
                        <div className='project-details__rank-header'>Rank {renderValue(rank)}</div>
                        <div className='project-details__rank-line'>Wallet: {renderValue(item.wallet)}</div>
                        <div className='project-details__rank-line'>
                            Net Amount: {renderValue(item.netAmount)} | Ratio: {renderRatioFromBps(item.ratioBps)}
                        </div>
                        <div className='project-details__rank-assets'>
                            <div className='project-details__rank-asset'>Asset Token: {renderValue(assetState?.tokenBalance)}</div>
                            <div className='project-details__rank-asset'>Asset WETH: {renderValue(assetState?.wethBalance)}</div>
                            <div className='project-details__rank-asset'>Asset USDT: {renderValue(assetState?.usdtBalance)}</div>
                            <div className='project-details__rank-asset'>Asset Native: {renderValue(assetState?.nativeBalance)}</div>
                            <div className='project-details__rank-asset project-details__rank-asset--wide'>
                                Total Asset (USDT): {formatUsdtValue(assetState?.usdtValue, usdtDecimals)}
                            </div>
                        </div>
                    </div>
                );
            })}
        </div>
    );
};
