import {ExportOutlined} from '@ant-design/icons';
import {Tooltip, Typography} from 'antd';
import {TruncatedText} from '../components';
import {tokenExplorerURL} from './token-shared';

const hasText = (value?: string) => value !== undefined && value !== '';

export const formatProjectTimeText = (value?: string) => {
    if (!hasText(value)) {
        return undefined;
    }
    const parsed = new Date(value as string);
    return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleString();
};

export const ProjectTimeValue = (props: {value?: string; unixSeconds?: number}) => {
    const raw = props.value || (props.unixSeconds ? new Date(props.unixSeconds * 1000).toISOString() : undefined);
    const display = formatProjectTimeText(raw);
    if (!display) {
        return <Typography.Text type='secondary'>-</Typography.Text>;
    }
    const parsed = new Date(raw as string);
    const dateTime = Number.isNaN(parsed.getTime()) ? raw : parsed.toISOString();
    return (
        <Tooltip title={raw}>
            <time dateTime={dateTime}>{display}</time>
        </Tooltip>
    );
};

export const ProjectExplorerValue = (props: {chainID?: number; kind: 'address' | 'tx'; value?: string}) => {
    if (!props.value) {
        return <Typography.Text type='secondary'>-</Typography.Text>;
    }
    const url = tokenExplorerURL(props.chainID, props.kind, props.value);
    return (
        <span className='project-detail-identifier'>
            <TruncatedText value={props.value} copyable={true} singleLine={true} />
            {url && (
                <Tooltip title='Open in block explorer'>
                    <Typography.Link className='project-detail-identifier__link' href={url} target='_blank' rel='noreferrer' aria-label={`Open ${props.value} in block explorer`}>
                        <ExportOutlined />
                    </Typography.Link>
                </Tooltip>
            )}
        </span>
    );
};

export const exactDecimalFromRaw = (raw?: string, decimals = 0) => {
    if (!hasText(raw)) {
        return undefined;
    }
    const match = String(raw).match(/^(-?)(\d+)$/);
    if (!match || decimals < 0) {
        return String(raw);
    }
    const digits = match[2].padStart(decimals + 1, '0');
    const whole = decimals === 0 ? digits : digits.slice(0, -decimals);
    const fraction = decimals === 0 ? '' : digits.slice(-decimals).replace(/0+$/, '');
    return `${match[1]}${whole}${fraction ? `.${fraction}` : ''}`;
};

export const ProjectExactValue = (props: {value?: string; suffix?: string; tooltip?: string}) => {
    if (!hasText(props.value)) {
        return <Typography.Text type='secondary'>-</Typography.Text>;
    }
    const exact = `${props.value}${props.suffix ? ` ${props.suffix}` : ''}`;
    return (
        <Tooltip title={props.tooltip || exact}>
            <span className='project-detail-value__exact'>{exact}</span>
        </Tooltip>
    );
};

export const ProjectRawTokenAmount = (props: {raw?: string; decimals?: number; symbol?: string}) => {
    const display = exactDecimalFromRaw(props.raw, props.decimals);
    const suffix = props.symbol || '';
    return <ProjectExactValue value={display} suffix={suffix} tooltip={hasText(props.raw) ? `${display}${suffix ? ` ${suffix}` : ''} · ${props.raw} base units` : undefined} />;
};
