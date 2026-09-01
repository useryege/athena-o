import {Tag} from 'antd';
import * as React from 'react';
import bscIcon from '../../../assets/images/bsc.png';
import ethIcon from '../../../assets/images/eth.png';
import solanaIcon from '../../../assets/images/solana.png';

const chainIconAssets = {
    eth: ethIcon,
    bsc: bscIcon,
    solana: solanaIcon
};

interface TokenChainDisplay {
    label: string;
    icon?: string;
    explorer: string;
    native: string;
    wrapped: string;
    stable: string;
    stableDecimals: number;
}

const tokenChainDisplayByID: Record<number, TokenChainDisplay> = {
    1: {label: 'ETH', icon: chainIconAssets.eth, explorer: 'https://etherscan.io', native: 'ETH', wrapped: 'WETH', stable: 'USDT', stableDecimals: 6},
    56: {label: 'BSC', icon: chainIconAssets.bsc, explorer: 'https://bscscan.com', native: 'BNB', wrapped: 'WBNB', stable: 'USDT', stableDecimals: 18}
};

export const chainLabel = (chainID?: number) => {
    if (chainID === undefined) {
        return '-';
    }
    return tokenChainDisplayByID[chainID]?.label || String(chainID);
};

export const ChainBadge = (props: {chainID?: number}) => {
    const display = props.chainID === undefined ? undefined : tokenChainDisplayByID[props.chainID];
    return (
        <Tag className='chain-badge'>
            {display?.icon && <img src={display.icon} alt='' />}
            <span>{chainLabel(props.chainID)}</span>
        </Tag>
    );
};

export const TokenLogo = (props: {logoURL?: string; symbol?: string; size?: 'list' | 'detail'}) => {
    const logoURL = props.logoURL?.trim();
    const [failedURL, setFailedURL] = React.useState<string>();
    const size = props.size || 'list';
    const showImage = Boolean(logoURL && failedURL !== logoURL);
    return (
        <span className={`token-logo token-logo--${size}`} aria-hidden='true'>
            {showImage ? (
                <img src={logoURL} alt='' loading={size === 'list' ? 'lazy' : 'eager'} referrerPolicy='no-referrer' draggable={false} onError={() => setFailedURL(logoURL)} />
            ) : (
                <span>{props.symbol?.trim() || '?'}</span>
            )}
        </span>
    );
};

export const chainAssetLabels = (chainID?: number) => {
    const display = chainID === undefined ? undefined : tokenChainDisplayByID[chainID];
    return {
        native: display?.native || 'Native',
        wrapped: display?.wrapped || 'Wrapped native',
        stable: display?.stable || 'USDT',
        stableDecimals: display?.stableDecimals ?? 18
    };
};

export const tokenExplorerURL = (chainID: number | undefined, kind: 'address' | 'tx', value?: string) => {
    if (!value || chainID === undefined) {
        return undefined;
    }
    const explorer = tokenChainDisplayByID[chainID]?.explorer;
    return explorer ? `${explorer}/${kind}/${encodeURIComponent(value)}` : undefined;
};
