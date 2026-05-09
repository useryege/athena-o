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
                <div className='projects-list__cell'>{renderValue(project.token?.decimals)}</div>
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
                <div className='projects-list__cell'>{renderValue(project.meta?.txIndex)}</div>
                <div className='projects-list__cell' title={project.meta?.txHash || ''}>
                    {renderShortValue(project.meta?.txHash)}
                </div>
            </div>
        </div>
    );
};
