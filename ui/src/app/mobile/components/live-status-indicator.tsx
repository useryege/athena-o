import * as React from 'react';

export const LiveStatusIndicator = (props: {live: boolean; label?: React.ReactNode}) => {
    if (!props.live) {
        return null;
    }
    return (
        <span className='live-status-indicator'>
            <span className='live-status-indicator__dot' />
            <span>{props.label || 'Live'}</span>
        </span>
    );
};
