import * as React from 'react';

import {ProjectView} from '../../../shared/services/athena-application-service';

const renderValue = (value: string | number | undefined) => (value === undefined || value === '' ? '-' : value);

const isOpenSource = (project: ProjectView) => {
    const sourceCode = project.meta?.sourceCode || '';
    return sourceCode.trim().length > 0;
};

const formatBlockTime = (blockTime: number | undefined) => {
    if (!blockTime || !Number.isFinite(blockTime) || blockTime <= 0) {
        return '-';
    }

    const date = new Date(blockTime * 1000);
    if (Number.isNaN(date.getTime())) {
        return '-';
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
        return '-';
    }

    return `${year}-${month}-${day} ${hour}:${minute}:${second}`;
};

export const ProjectListRow = ({project, index, onClick}: {project: ProjectView; index: number; onClick?: () => void}) => {
    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (onClick && (e.key === 'Enter' || e.key === ' ')) {
            e.preventDefault();
            onClick();
        }
    };

    return (
        <div
            className='argo-table-list__row'
            onClick={onClick}
            onKeyDown={handleKeyDown}
            role={onClick ? 'button' : undefined}
            tabIndex={onClick ? 0 : undefined}
            style={{cursor: onClick ? 'pointer' : 'default'}}>
            <div className='projects-list__row'>
                <div className='projects-list__cell projects-list__cell--rank'>#{index + 1}</div>
                <div className='projects-list__cell' title={project.chainState?.token?.name || ''}>
                    {renderValue(project.chainState?.token?.name)}
                </div>
                <div className='projects-list__cell'>{renderValue(project.chainState?.token?.symbol)}</div>
                <div className='projects-list__cell'>
                    {project.meta?.sourceCodeBlacklist?.hasBlacklistFields !== undefined ? (
                        <span className={`project-details__badge project-details__badge--${project.meta.sourceCodeBlacklist.hasBlacklistFields ? 'negative' : 'positive'}`}>
                            {project.meta.sourceCodeBlacklist.hasBlacklistFields ? 'Yes' : 'No'}
                        </span>
                    ) : (
                        '-'
                    )}
                </div>
                <div className='projects-list__cell'>
                    {project.meta?.creatorResult
                        ? (() => {
                              const hasRisk =
                                  project.meta.creatorResult.canMintViaTransferToWethPair ||
                                  project.meta.creatorResult.canMintViaTransferToUsdtPair ||
                                  project.meta.creatorResult.canMintFromDeadViaTransferFrom ||
                                  project.meta.creatorResult.canMintFromZeroViaTransferFrom ||
                                  project.meta.creatorResult.canMintFromWethPairViaTransferFrom ||
                                  project.meta.creatorResult.canMintFromUsdtPairViaTransferFrom;
                              return <span className={`project-details__badge project-details__badge--${hasRisk ? 'negative' : 'positive'}`}>{hasRisk ? 'Yes' : 'No'}</span>;
                          })()
                        : '-'}
                </div>
                <div className='projects-list__cell'>
                    <span className={`project-details__badge project-details__badge--${isOpenSource(project) ? 'positive' : 'negative'}`}>
                        {isOpenSource(project) ? 'Yes' : 'No'}
                    </span>
                </div>
                <div className='projects-list__cell'>{formatBlockTime(project.meta?.blockTime)}</div>
            </div>
        </div>
    );
};
