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
                            <Button aria-label={props.refreshLabel || 'Refresh data'} icon={<ReloadOutlined />} loading={props.loading} onClick={props.onRefresh} />
                        </Tooltip>
                    )}
                    {props.extra}
                </Space>
            </header>
            {props.filters && <div className='app-page__filters'>{props.filters}</div>}
            {props.error && <Alert className='app-page__alert' type='error' title='Request failed' description={props.error.message} showIcon={true} />}
            {props.loading && !props.children ? <Skeleton active={true} /> : props.children}
        </div>
    );
};

export const BrandMark = (props: {size?: 'small' | 'large'}) => (
    <span className={`brand-mark brand-mark--${props.size || 'small'}`} aria-hidden='true'>
        A
    </span>
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
