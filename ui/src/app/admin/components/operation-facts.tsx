import * as React from 'react';

/** Label/value facts for narrow operation panels; container layout keeps full diagnostic values readable. */
export const OperationFacts = (props: {items: Array<{label: React.ReactNode; value: React.ReactNode}>; columns?: number}) => (
    <dl className={`admin-operation-facts${props.columns === 1 ? ' admin-operation-facts--single' : ''}`}>
        {props.items.map(item => (
            <div key={String(item.label)}>
                <dt>{item.label}</dt>
                <dd>{item.value ?? '—'}</dd>
            </div>
        ))}
    </dl>
);
