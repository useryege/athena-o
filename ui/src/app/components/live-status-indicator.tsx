import * as React from 'react';

export const LiveStatusIndicator = (props: {live: boolean; label?: React.ReactNode}) => {
    if (!props.live) {
        return null;
    }
    return (
        <span className='live-status-indicator' role='status'>
            <span className='live-status-indicator__dot' aria-hidden='true' />
            <span>{props.label || 'Live'}</span>
        </span>
    );
};
