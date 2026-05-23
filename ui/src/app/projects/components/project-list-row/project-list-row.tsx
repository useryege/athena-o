import {Tooltip} from 'argo-ui';
import * as React from 'react';
import {Link} from 'react-router-dom';

import {ProjectListItem} from '../../../shared/services/athena-application-service';
import {formatUsdtValue, PairMetricsCell} from '../pair-metrics-cell/pair-metrics-cell';

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

const renderProjectName = (project: ProjectListItem) => {
    const name = renderValue(project.name);
    if (!project.symbol) {
        return name;
    }
    return `${name}(${project.symbol})`;
};

export const ProjectListRow = ({
    project,
    index,
    usdtDecimals,
    defaultIsArchived,
    to
}: {
    project: ProjectListItem;
    index: number;
    usdtDecimals?: number;
    defaultIsArchived?: boolean;
    to?: string;
}) => {
    const [copied, setCopied] = React.useState(false);
    const blockTime = formatBlockTime(project.blockTime);
    const isArchived = project.isArchived ?? defaultIsArchived ?? false;

    const copyContract = React.useCallback(
        async (event: React.MouseEvent<HTMLButtonElement>) => {
            event.preventDefault();
            event.stopPropagation();
            if (!project.contract) {
                return;
            }
            await navigator.clipboard.writeText(project.contract);
            setCopied(true);
            window.setTimeout(() => setCopied(false), 3000);
        },
        [project.contract]
    );

    const rowContent = (
        <div className='projects-list__row'>
            <div className='projects-list__cell projects-list__cell--rank'>#{index + 1}</div>
            <div className='projects-list__cell' title={project.symbol ? `${project.name || '-'}(${project.symbol})` : project.name || ''}>
                {renderProjectName(project)}
            </div>
            <div className='projects-list__cell projects-list__cell--contract' title={project.contract || ''}>
                <span>{renderValue(project.contract)}</span>
                {project.contract && (
                    <Tooltip content={copied ? 'Copied!' : 'Copy contract address'} hideOnClick={false}>
                        <button type='button' className='projects-list__copy-button' aria-label='Copy contract address' onClick={copyContract}>
                            <i className='fa fa-copy' />
                        </button>
                    </Tooltip>
                )}
            </div>
            <div className='projects-list__cell'>
                <span className={`project-details__badge project-details__badge--${isArchived ? 'negative' : 'positive'}`}>{isArchived ? 'Archived' : 'Active'}</span>
            </div>
            <div className='projects-list__cell'>
                <span className={`project-details__badge project-details__badge--${project.hasMintRisk ? 'negative' : 'positive'}`}>{project.hasMintRisk ? 'Yes' : 'No'}</span>
            </div>
            <div className='projects-list__cell'>
                <span className={`project-details__badge project-details__badge--${project.isOpenSource ? 'positive' : 'negative'}`}>{project.isOpenSource ? 'Yes' : 'No'}</span>
            </div>
            <div className='projects-list__cell'>
                <PairMetricsCell quoteValue={project.wethPairQuoteUsdtValue} removeLiquidity={project.wethPairRemoveLiquidity} usdtDecimals={usdtDecimals} />
            </div>
            <div className='projects-list__cell'>
                <PairMetricsCell quoteValue={project.usdtPairQuoteUsdtValue} removeLiquidity={project.usdtPairRemoveLiquidity} usdtDecimals={usdtDecimals} />
            </div>
            <div className='projects-list__cell'>{formatUsdtValue(project.creatorAssetUsdtValue, usdtDecimals)}</div>
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
