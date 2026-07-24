import {Empty, Tooltip, Typography} from 'antd';
import * as React from 'react';
import {TokenProjectTrendSeries} from '../shared/services/token-service';

const WIDTH = 920;
const HEIGHT = 260;
const PADDING = {top: 18, right: 18, bottom: 30, left: 18};

const numericPoints = (series?: TokenProjectTrendSeries) =>
    (series?.points || []).map(point => ({...point, numericValue: Number(point.value)})).filter(point => point.observedAt && Number.isFinite(point.numericValue));

const formatMetric = (value: number, unit?: string) => {
    const formatted = new Intl.NumberFormat(undefined, {
        notation: Math.abs(value) >= 100000 ? 'compact' : 'standard',
        maximumFractionDigits: Math.abs(value) < 1 ? 8 : 2
    }).format(value);
    return unit?.toLowerCase() === 'usd' ? `$${formatted}` : unit === 'count' ? formatted : `${formatted}${unit ? ` ${unit}` : ''}`;
};

export const ProjectTrendChart = (props: {series?: TokenProjectTrendSeries}) => {
    const points = React.useMemo(() => numericPoints(props.series), [props.series]);
    const [hoverIndex, setHoverIndex] = React.useState<number>();
    const titleID = React.useId();

    if (points.length === 0) {
        return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='No trend observations in this range' />;
    }

    const values = points.map(point => point.numericValue);
    const minimum = Math.min(...values);
    const maximum = Math.max(...values);
    const spread = maximum - minimum || Math.abs(maximum) * 0.04 || 1;
    const chartWidth = WIDTH - PADDING.left - PADDING.right;
    const chartHeight = HEIGHT - PADDING.top - PADDING.bottom;
    const xAt = (index: number) => PADDING.left + (index / Math.max(points.length - 1, 1)) * chartWidth;
    const yAt = (value: number) => PADDING.top + ((maximum + spread * 0.08 - value) / (spread * 1.16)) * chartHeight;
    const coordinates = points.map((point, index) => `${xAt(index)},${yAt(point.numericValue)}`).join(' ');
    const selectedIndex = hoverIndex ?? points.length - 1;
    const selected = points[selectedIndex];
    const first = points[0];
    const last = points[points.length - 1];
    const change = first.numericValue === 0 ? undefined : ((last.numericValue - first.numericValue) / Math.abs(first.numericValue)) * 100;

    const onPointerMove = (event: React.PointerEvent<SVGSVGElement>) => {
        const bounds = event.currentTarget.getBoundingClientRect();
        const localX = ((event.clientX - bounds.left) / bounds.width) * WIDTH;
        const index = Math.round(((localX - PADDING.left) / chartWidth) * Math.max(points.length - 1, 1));
        setHoverIndex(Math.max(0, Math.min(points.length - 1, index)));
    };

    return (
        <div className='project-trend'>
            <div className='project-trend__summary'>
                <span>
                    <Typography.Text type='secondary'>Current</Typography.Text>
                    <strong>{formatMetric(last.numericValue, props.series?.unit)}</strong>
                </span>
                <span>
                    <Typography.Text type='secondary'>Minimum</Typography.Text>
                    <strong>{formatMetric(minimum, props.series?.unit)}</strong>
                </span>
                <span>
                    <Typography.Text type='secondary'>Maximum</Typography.Text>
                    <strong>{formatMetric(maximum, props.series?.unit)}</strong>
                </span>
                <span>
                    <Typography.Text type='secondary'>Change</Typography.Text>
                    <strong>{change === undefined ? 'Unknown' : `${change >= 0 ? '+' : ''}${change.toFixed(2)}%`}</strong>
                </span>
            </div>
            <div className='project-trend__canvas'>
                <svg viewBox={`0 0 ${WIDTH} ${HEIGHT}`} role='img' aria-labelledby={titleID} onPointerMove={onPointerMove} onPointerLeave={() => setHoverIndex(undefined)}>
                    <title id={titleID}>
                        {props.series?.label || 'Trend'}, {points.length} observations. Current {formatMetric(last.numericValue, props.series?.unit)}.
                    </title>
                    {[0.25, 0.5, 0.75].map(ratio => (
                        <line
                            key={ratio}
                            className='project-trend__grid'
                            x1={PADDING.left}
                            x2={WIDTH - PADDING.right}
                            y1={PADDING.top + chartHeight * ratio}
                            y2={PADDING.top + chartHeight * ratio}
                        />
                    ))}
                    <polyline className='project-trend__line' points={coordinates} />
                    <line className='project-trend__cursor' x1={xAt(selectedIndex)} x2={xAt(selectedIndex)} y1={PADDING.top} y2={HEIGHT - PADDING.bottom} />
                    <circle className='project-trend__point' cx={xAt(selectedIndex)} cy={yAt(selected.numericValue)} r={5} />
                    <text className='project-trend__axis-label' x={PADDING.left} y={HEIGHT - 8}>
                        {new Date(first.observedAt || '').toLocaleString()}
                    </text>
                    <text className='project-trend__axis-label' x={WIDTH - PADDING.right} y={HEIGHT - 8} textAnchor='end'>
                        {new Date(last.observedAt || '').toLocaleString()}
                    </text>
                </svg>
                <Tooltip
                    open={hoverIndex !== undefined}
                    title={
                        <span>
                            {formatMetric(selected.numericValue, props.series?.unit)}
                            <br />
                            {new Date(selected.observedAt || '').toLocaleString()}
                        </span>
                    }>
                    <span
                        className='project-trend__tooltip-anchor'
                        style={{left: `${(xAt(selectedIndex) / WIDTH) * 100}%`, top: `${(yAt(selected.numericValue) / HEIGHT) * 100}%`}}
                    />
                </Tooltip>
            </div>
        </div>
    );
};
