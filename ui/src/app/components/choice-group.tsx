import {Radio} from 'antd';
import * as React from 'react';

export type ChoiceValue = string | number;

export interface ChoiceGroupOption<T extends ChoiceValue> {
    value: T;
    label: React.ReactNode;
    disabled?: boolean;
}

export const ChoiceGroup = <T extends ChoiceValue>(props: {
    ariaLabel: string;
    value?: T;
    options: ChoiceGroupOption<T>[];
    onChange?: (value: T) => void;
    size?: 'small' | 'middle';
    className?: string;
    disabled?: boolean;
}) => {
    const className = ['choice-group', props.size === 'small' ? 'choice-group--small' : undefined, props.className].filter(Boolean).join(' ');
    return (
        <Radio.Group aria-label={props.ariaLabel} className={className} value={props.value} disabled={props.disabled} onChange={event => props.onChange?.(event.target.value as T)}>
            {props.options.map(option => (
                <Radio.Button key={String(option.value)} value={option.value} disabled={option.disabled}>
                    {option.label}
                </Radio.Button>
            ))}
        </Radio.Group>
    );
};
