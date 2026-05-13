import * as React from 'react';

import {ProjectView} from '../../../shared/services/athena-application-service';

const renderValue = (value: string | number | undefined) => (value === undefined || value === '' ? '-' : value);

const renderShortValue = (value: string | undefined, maxLength = 12) => {
    if (!value) {
        return '-';
    }
    return value.length > maxLength ? `${value.substring(0, maxLength)}...` : value;
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
                <div className='projects-list__cell' title={project.token?.name || ''}>
                    {renderValue(project.token?.name)}
                </div>
                <div className='projects-list__cell'>{renderValue(project.token?.symbol)}</div>
                <div className='projects-list__cell'>
                    {project.token?.isValidERC20 !== undefined ? (
                        <span className={`project-details__badge project-details__badge--${project.token.isValidERC20 ? 'positive' : 'negative'}`}>
                            {project.token.isValidERC20 ? 'Yes' : 'No'}
                        </span>
                    ) : (
                        '-'
                    )}
                </div>
                <div className='projects-list__cell'>
                    {project.analysis?.sourceCodeBlacklist?.hasBlacklistFields !== undefined ? (
                        <span className={`project-details__badge project-details__badge--${project.analysis.sourceCodeBlacklist.hasBlacklistFields ? 'negative' : 'positive'}`}>
                            {project.analysis.sourceCodeBlacklist.hasBlacklistFields ? 'Yes' : 'No'}
                        </span>
                    ) : (
                        '-'
                    )}
                </div>
                <div className='projects-list__cell'>
                    {project.simulate?.creatorResult
                        ? (() => {
                              const hasRisk =
                                  project.simulate.creatorResult.canMintViaTransfer ||
                                  project.simulate.creatorResult.canMintFromDeadViaTransferFrom ||
                                  project.simulate.creatorResult.canMintFromZeroViaTransferFrom ||
                                  project.simulate.creatorResult.canMintFromPairViaTransferFrom;
                              return <span className={`project-details__badge project-details__badge--${hasRisk ? 'negative' : 'positive'}`}>{hasRisk ? 'Yes' : 'No'}</span>;
                          })()
                        : '-'}
                </div>
                <div className='projects-list__cell' title={project.token?.totalSupply || ''}>
                    {renderShortValue(project.token?.totalSupply)}
                </div>
                <div className='projects-list__cell' title={project.meta?.contract || ''}>
                    {renderShortValue(project.meta?.contract)}
                </div>
                <div className='projects-list__cell' title={project.meta?.creator || ''}>
                    {renderShortValue(project.meta?.creator)}
                </div>
                <div className='projects-list__cell'>{renderValue(project.meta?.blockNumber)}</div>
                <div className='projects-list__cell' title={project.meta?.txHash || ''}>
                    {renderShortValue(project.meta?.txHash)}
                </div>
            </div>
        </div>
    );
};
