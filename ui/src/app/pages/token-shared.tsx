import {Tag} from 'antd';
import bscIcon from '../../assets/images/bsc.png';
import ethIcon from '../../assets/images/eth.png';
import solanaIcon from '../../assets/images/solana.png';

const chainIconAssets = {
    eth: ethIcon,
    bsc: bscIcon,
    solana: solanaIcon
};

const tokenChainDisplayByID: Record<number, {label: string; icon?: string}> = {
    1: {label: 'ETH', icon: chainIconAssets.eth},
    56: {label: 'BSC', icon: chainIconAssets.bsc}
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
