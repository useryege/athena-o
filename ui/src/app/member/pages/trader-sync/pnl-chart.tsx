import * as React from 'react';
import {Typography} from 'antd';
import type {FieldEvidence, PnLView} from '../../trader-sync-models';
import {formatBeijingDateTime} from '../../../shared/format';

export const evidenceReason = (evidence: FieldEvidence) => `Unavailable (${evidence.reasonCode || 'reason not provided'})`;
export const providerMoney = (value: string) => (value.startsWith('-') ? `-$${value.slice(1)}` : `$${value}`);
const pointTime = (value: string) => {
    const milliseconds = Number(value) * 1000;
    return Number.isFinite(milliseconds) && !Number.isNaN(new Date(milliseconds).getTime()) ? formatBeijingDateTime(new Date(milliseconds))! : `Time unavailable (${value})`;
};
/** Only bounded drawing coordinates become Number; all business values stay strings. */
const normalized = (values: string[]): number[] => {
    const decimals = Math.max(...values.map(value => (value.split('.')[1] || '').length));
    const integers = values.map(value => {
        const negative = value.startsWith('-');
        const [whole, fraction = ''] = (negative ? value.slice(1) : value).split('.');
        return BigInt(whole + fraction.padEnd(decimals, '0')) * (negative ? -1n : 1n);
    });
    const min = integers.reduce((a, b) => (a < b ? a : b));
    const max = integers.reduce((a, b) => (a > b ? a : b));
    return integers.map(value => (max === min ? 0.5 : Number(((value - min) * 1000000n) / (max - min)) / 1000000));
};

export const PnLChart = ({view}: {view: PnLView}) => {
    const id = React.useId();
    const points = view.curve.points;
    const available = view.curve.evidence.availability === 'available' && points.length > 0;
    const xs = available ? normalized(points.map(point => point.t)) : [];
    const ys = available ? normalized(points.map(point => point.p)) : [];
    const amount = view.amount.evidence.availability === 'available' && view.amount.value !== undefined ? providerMoney(view.amount.value) : evidenceReason(view.amount.evidence);
    return (
        <div className='trader-sync-pnl'>
            <Typography.Title level={3}>
                {view.period} P/L:{' '}
                <Typography.Text copyable={view.amount.evidence.availability === 'available' && view.amount.value !== undefined ? {text: view.amount.value} : false}>
                    {amount}
                </Typography.Text>
            </Typography.Title>
            <Typography.Paragraph type='secondary'>$ follows Polymarket’s display; currency code is not provided. Times shown in UTC+8.</Typography.Paragraph>
            {available ? (
                <>
                    <svg viewBox='0 0 640 240' role='img' aria-labelledby={`${id}-title ${id}-summary`}>
                        <title id={`${id}-title`}>{view.period} P/L curve</title>
                        <desc id={`${id}-summary`}>
                            {points.length} observations. First {points[0].p}; last {points[points.length - 1].p}. Exact values are available below.
                        </desc>
                        <path className='trader-sync-pnl__axis' d='M40 16 V204 H624' />
                        <text x='40' y='232'>
                            Time (UTC+8)
                        </text>
                        <text x='44' y='14'>
                            P/L ($)
                        </text>
                        <polyline
                            fill='none'
                            stroke='currentColor'
                            strokeWidth='2'
                            points={points.map((_point, index) => `${40 + xs[index] * 584},${196 - ys[index] * 172}`).join(' ')}
                        />
                        {points.map((point, index) => (
                            <circle key={`${point.t}-${index}`} cx={40 + xs[index] * 584} cy={196 - ys[index] * 172} r='3'>
                                <title>
                                    {pointTime(point.t)} UTC+8: {providerMoney(point.p)}
                                </title>
                            </circle>
                        ))}
                    </svg>
                    <Typography.Paragraph>
                        {pointTime(points[0].t)} – {pointTime(points[points.length - 1].t)} UTC+8. First: {points[0].p}; last: {points[points.length - 1].p}.
                    </Typography.Paragraph>
                    <details>
                        <summary>Exact curve values ({points.length})</summary>
                        <div className='trader-sync-pnl__values'>
                            <table>
                                <caption>Provider P/L ($)</caption>
                                <thead>
                                    <tr>
                                        <th>Time (UTC+8)</th>
                                        <th>P/L</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {points.map((point, index) => (
                                        <tr key={`${point.t}-${index}`}>
                                            <td>{pointTime(point.t)}</td>
                                            <td>
                                                <Typography.Text copyable={{text: point.p}}>{providerMoney(point.p)}</Typography.Text>
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
                    </details>
                </>
            ) : (
                <Typography.Paragraph role='status'>
                    {view.curve.evidence.availability === 'unavailable' ? evidenceReason(view.curve.evidence) : 'Curve unavailable (no observations)'}
                </Typography.Paragraph>
            )}
            <Typography.Paragraph type='secondary'>
                Interval: {view.interval || 'Unavailable'} · Fidelity: {view.fidelity || 'Unavailable'} · Queried:{' '}
                {formatBeijingDateTime(view.curve.evidence.queriedAt) || 'Unavailable'} UTC+8
            </Typography.Paragraph>
            <Typography.Paragraph type='secondary'>
                Provider reference time:{' '}
                {view.referenceTime.evidence.availability === 'available' && view.referenceTime.value
                    ? formatBeijingDateTime(view.referenceTime.value) + ' UTC+8'
                    : evidenceReason(view.referenceTime.evidence)}
                . Provider period timezone:{' '}
                {view.timezone.evidence.availability === 'available' && view.timezone.value ? view.timezone.value : evidenceReason(view.timezone.evidence)}.
            </Typography.Paragraph>
        </div>
    );
};
