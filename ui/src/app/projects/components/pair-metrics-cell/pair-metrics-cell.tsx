import * as React from 'react';

export interface PairMetricsCellProps {
    quoteValue?: string;
    removeLiquidity?: boolean;
    usdtDecimals?: number | string;
}

const normalizeDecimals = (decimals?: number | string) => {
    if (typeof decimals === 'number' && Number.isInteger(decimals) && decimals >= 0) {
        return decimals;
    }
    if (typeof decimals === 'string' && /^\d+$/.test(decimals)) {
        return Number(decimals);
    }
    return undefined;
};

const formatQuoteUsdt = (rawValue?: string, decimals?: number | string) => {
    if (rawValue === undefined || rawValue === null || rawValue === '') {
        return '-';
    }
    const normalizedDecimals = normalizeDecimals(decimals);
    if (normalizedDecimals === undefined) {
        return '-';
    }
    const normalizedRawValue = rawValue.trim();
    if (!/^\d+$/.test(normalizedRawValue)) {
        return '-';
    }

    try {
        const oneHundred = BigInt(100);
        const value = BigInt(normalizedRawValue);
        let factor = BigInt(1);
        for (let i = 0; i < normalizedDecimals; i++) {
            factor *= BigInt(10);
        }
        const scaled = value * oneHundred;
        const rounded = (scaled + factor / BigInt(2)) / factor;
        const integerPart = rounded / oneHundred;
        const fractionalPart = rounded % oneHundred;
        return `${integerPart.toString()}.${fractionalPart.toString().padStart(2, '0')}`;
    } catch {
        return '-';
    }
};

export const PairMetricsCell = ({quoteValue, removeLiquidity, usdtDecimals}: PairMetricsCellProps) => (
    <div className='pair-metrics-cell'>
        <div className='pair-metrics-cell__item'>
            <span className='pair-metrics-cell__label'>Quote</span>
            <span className='pair-metrics-cell__value'>{formatQuoteUsdt(quoteValue, usdtDecimals)}</span>
        </div>
        <div className='pair-metrics-cell__item'>
            <span className='pair-metrics-cell__label'>RmLiq</span>
            {removeLiquidity !== undefined ? (
                <span className={`project-details__badge project-details__badge--${removeLiquidity ? 'negative' : 'positive'}`}>
                    {removeLiquidity ? 'Yes' : 'No'}
                </span>
            ) : (
                <span className='pair-metrics-cell__value'>-</span>
            )}
        </div>
    </div>
);
