import {ReloadOutlined} from '@ant-design/icons';
import {Alert, Button, Skeleton, Space, Tooltip, Typography} from 'antd';
import * as React from 'react';

export const AppPage = (props: {
    title: string;
    subtitle?: React.ReactNode;
    extra?: React.ReactNode;
    filters?: React.ReactNode;
    children: React.ReactNode;
    loading?: boolean;
    error?: Error;
    stale?: boolean;
    onRefresh?: () => void;
    refreshLabel?: string;
}) => {
    return (
        <div className='app-page' aria-busy={props.loading || undefined}>
            <header className='app-page__header'>
                <div className='app-page__heading'>
                    <Typography.Title level={1}>{props.title}</Typography.Title>
                    {props.subtitle && <Typography.Text type='secondary'>{props.subtitle}</Typography.Text>}
                </div>
                <Space className='app-page__actions' wrap={true}>
                    {props.onRefresh && (
                        <Tooltip title={props.refreshLabel || 'Refresh data'}>
                            <Button aria-label={props.refreshLabel || 'Refresh data'} icon={<ReloadOutlined />} loading={props.loading} onClick={props.onRefresh}>
                                {props.refreshLabel || 'Refresh'}
                            </Button>
                        </Tooltip>
                    )}
                    {props.extra}
                </Space>
            </header>
            {props.filters && <div className='app-page__filters'>{props.filters}</div>}
            {props.error && (
                <Alert
                    className='app-page__alert'
                    type='error'
                    title='Request failed'
                    description={props.error.message}
                    showIcon={true}
                    action={props.onRefresh && <Button onClick={props.onRefresh}>Retry</Button>}
                />
            )}
            {props.stale && <Alert type='warning' title='Stale data — showing the last successful read' />}
            {props.loading && !props.children ? <Skeleton active={true} /> : props.children}
        </div>
    );
};

export const BrandMark = (props: {size?: 'small' | 'large'}) => (
    <svg className={`brand-mark brand-mark--${props.size || 'small'}`} viewBox='0 0 28 28' aria-hidden='true'>
        <path d='m4 23 10-19 10 19M8 17h12M11 23h6' />
    </svg>
);

export const Section = (props: {title: string; extra?: React.ReactNode; children: React.ReactNode}) => {
    const headingID = React.useId();
    return (
        <section className='section-panel' aria-labelledby={headingID}>
            <div className='section-panel__header'>
                <Typography.Title id={headingID} level={2}>
                    {props.title}
                </Typography.Title>
                {props.extra && <div className='section-panel__extra'>{props.extra}</div>}
            </div>
            <div className='section-panel__body'>{props.children}</div>
        </section>
    );
};
