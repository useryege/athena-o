import {Tooltip} from 'argo-ui';
import * as React from 'react';
import {Link} from 'react-router-dom';

import {ProjectListItem} from '../../../shared/services/athena-application-service';

const renderValue = (value: string | number | undefined) => (value === undefined || value === '' ? '-' : value);

const formatBlockTime = (blockTime: number | undefined): {date: string; time: string} | null => {
    if (!blockTime || !Number.isFinite(blockTime) || blockTime <= 0) {
        return null;
    }

    const date = new Date(blockTime * 1000);
    if (Number.isNaN(date.getTime())) {
        return null;
    }

    const formatter = new Intl.DateTimeFormat('zh-CN', {
        timeZone: 'Asia/Shanghai',
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false
    });
    const parts = formatter.formatToParts(date);
    const partValue = (type: Intl.DateTimeFormatPartTypes) => parts.find(part => part.type === type)?.value ?? '';
    const year = partValue('year');
    const month = partValue('month');
    const day = partValue('day');
    const hour = partValue('hour');
    const minute = partValue('minute');
    const second = partValue('second');

    if (!year || !month || !day || !hour || !minute || !second) {
        return null;
    }
    return {date: `${year}-${month}-${day}`, time: `${hour}:${minute}:${second}`};
};

const PolicyBadge = ({label, value, negativeWhenTrue = true}: {label: string; value?: boolean; negativeWhenTrue?: boolean}) => {
    const known = value !== undefined;
    const isNegative = known && (negativeWhenTrue ? !!value : !value);
    const modifier = known ? (isNegative ? 'negative' : 'positive') : 'neutral';
    return (
        <span className={`projects-list__badge projects-list__badge--${modifier}`} title={label}>
            {label}: {known ? (value ? 'Yes' : 'No') : '-'}
        </span>
    );
};

const CopyAddressButton = ({address, label}: {address?: string; label: string}) => {
    const [copied, setCopied] = React.useState(false);

    const copyAddress = React.useCallback(
        async (event: React.MouseEvent<HTMLButtonElement>) => {
            event.preventDefault();
            event.stopPropagation();
            if (!address) {
                return;
            }
            await navigator.clipboard.writeText(address);
            setCopied(true);
            window.setTimeout(() => setCopied(false), 3000);
        },
        [address]
    );

    if (!address) {
        return null;
    }

    return (
        <Tooltip content={copied ? 'Copied!' : `Copy ${label}`} hideOnClick={false}>
            <button type='button' className='projects-list__copy-button' aria-label={`Copy ${label}`} onClick={copyAddress}>
                <i className='fa fa-copy' />
            </button>
        </Tooltip>
    );
};

export const ProjectListRow = ({project, index, to}: {project: ProjectListItem; index: number; to?: string}) => {
    const blockTime = formatBlockTime(project.blockTime);

    const rowContent = (
        <div className='projects-list__row'>
            <div className='projects-list__cell projects-list__cell--rank'>#{index + 1}</div>
            <div className='projects-list__cell projects-list__cell--contract' title={project.contract || ''}>
                <span>{renderValue(project.contract)}</span>
                <CopyAddressButton address={project.contract} label='contract address' />
            </div>
            <div className='projects-list__cell projects-list__cell--contract' title={project.creator || ''}>
                <span>{renderValue(project.creator)}</span>
                <CopyAddressButton address={project.creator} label='creator address' />
            </div>
            <div className='projects-list__cell projects-list__cell--tx' title={project.txHash || ''}>
                {renderValue(project.txHash)}
            </div>
            <div className='projects-list__cell'>{renderValue(project.blockNumber)}</div>
            <div className='projects-list__cell'>{renderValue(project.txIndex)}</div>
            <div className='projects-list__cell projects-list__cell--policy'>
                <PolicyBadge label='Evaluated' value={project.isPolicyEvaluated} negativeWhenTrue={false} />
                <PolicyBadge label='Mint Risk' value={project.hasMintRisk} />
            </div>
            <div className='projects-list__cell projects-list__cell--policy'>
                <PolicyBadge label='Creator' value={project.isBlacklistedCreatorWallet} />
                <PolicyBadge label='Genesis' value={project.isBlacklistedGenesisWallet} />
                <PolicyBadge label='Bytecode' value={project.isBlacklistedBytecode} />
            </div>
            <div className='projects-list__cell projects-list__cell--block-time'>
                {blockTime ? (
                    <>
                        <span>{blockTime.date}</span>
                        <span>{blockTime.time}</span>
                    </>
                ) : (
                    '-'
                )}
            </div>
        </div>
    );

    if (to) {
        return (
            <Link className='argo-table-list__row projects-list__row-link' to={to}>
                {rowContent}
            </Link>
        );
    }

    return <div className='argo-table-list__row'>{rowContent}</div>;
};
