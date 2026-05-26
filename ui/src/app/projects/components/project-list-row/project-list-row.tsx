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

const projectInitial = (project: ProjectListItem) => (project.symbol || project.name || '?').trim().slice(0, 1).toUpperCase() || '?';

const trimCompactValue = (value: number) => value.toFixed(2).replace(/\.?0+$/, '');

const formatCompactNumber = (value: number) => {
    const absValue = Math.abs(value);
    if (absValue >= 1_000_000_000_000) {
        return `${trimCompactValue(value / 1_000_000_000_000)}T`;
    }
    if (absValue >= 1_000_000_000) {
        return `${trimCompactValue(value / 1_000_000_000)}B`;
    }
    if (absValue >= 1_000_000) {
        return `${trimCompactValue(value / 1_000_000)}M`;
    }
    if (absValue >= 1_000) {
        return `${trimCompactValue(value / 1_000)}K`;
    }
    return trimCompactValue(value);
};

const parseDecimalValue = (value: string | undefined) => {
    const normalized = (value || '').trim().replace(/,/g, '');
    if (!normalized) {
        return null;
    }
    const numericValue = Number(normalized);
    return Number.isFinite(numericValue) ? numericValue : null;
};

const formatAveMarketCap = (value: string | undefined) => {
    const numericValue = parseDecimalValue(value);
    if (numericValue !== null) {
        return `$${formatCompactNumber(numericValue)}`;
    }
    return renderValue(value);
};

const renderAveHolders = (project: ProjectListItem) => {
    if (!project.aveDetailAvailable) {
        return '-';
    }
    return typeof project.aveHolders === 'number' ? project.aveHolders.toLocaleString('en-US') : '-';
};

const renderAveRiskBadge = (label: string, isRisk: boolean) => (
    <span className={`projects-list__badge projects-list__badge--${isRisk ? 'negative' : 'positive'}`} title={label}>
        {label}: {isRisk ? 'Yes' : 'No'}
    </span>
);

const renderAveMintableBadge = (value: string | undefined) => {
    const normalized = (value || '').trim().toLowerCase();
    if (normalized === '1' || normalized === 'true' || normalized === 'yes') {
        return (
            <span className='projects-list__badge projects-list__badge--negative' title='Mintable'>
                Mintable: Yes
            </span>
        );
    }
    if (normalized === '0' || normalized === 'false' || normalized === 'no') {
        return (
            <span className='projects-list__badge projects-list__badge--positive' title='Mintable'>
                Mintable: No
            </span>
        );
    }
    return (
        <span className='projects-list__badge projects-list__badge--neutral' title='Mintable'>
            Mintable: -
        </span>
    );
};

const renderAveRisk = (project: ProjectListItem) => {
    if (!project.aveDetailAvailable) {
        return '-';
    }
    return (
        <div className='projects-list__ave-risk'>
            {renderAveRiskBadge('Honeypot', !!project.aveIsHoneypot)}
            {renderAveRiskBadge('Mint Method', !!project.aveHasMintMethod)}
            {renderAveMintableBadge(project.aveIsMintable)}
        </div>
    );
};

export const ProjectListRow = ({project, index, usdtDecimals, to}: {project: ProjectListItem; index: number; usdtDecimals?: number; to?: string}) => {
    const [copied, setCopied] = React.useState(false);
    const [logoFailed, setLogoFailed] = React.useState(false);
    const blockTime = formatBlockTime(project.blockTime);
    const showLogo = !!project.aveLogo && !logoFailed;

    React.useEffect(() => {
        setLogoFailed(false);
    }, [project.aveLogo]);

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
            <div className='projects-list__cell projects-list__cell--name' title={project.symbol ? `${project.name || '-'}(${project.symbol})` : project.name || ''}>
                <span className='projects-list__logo' aria-hidden='true'>
                    {showLogo ? <img src={project.aveLogo} alt='' onError={() => setLogoFailed(true)} /> : <span>{projectInitial(project)}</span>}
                </span>
                <span className='projects-list__name-text'>{renderProjectName(project)}</span>
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
                <span className={`projects-list__badge projects-list__badge--${project.hasMintRisk ? 'negative' : 'positive'}`}>{project.hasMintRisk ? 'Yes' : 'No'}</span>
            </div>
            <div className='projects-list__cell'>
                <span className={`projects-list__badge projects-list__badge--${project.isOpenSource ? 'positive' : 'negative'}`}>{project.isOpenSource ? 'Yes' : 'No'}</span>
            </div>
            <div className='projects-list__cell'>{renderAveRisk(project)}</div>
            <div className='projects-list__cell'>{renderAveHolders(project)}</div>
            <div className='projects-list__cell'>{project.aveDetailAvailable ? formatAveMarketCap(project.aveMarketCap) : '-'}</div>
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
