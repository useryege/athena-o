import {ReloadOutlined} from '@ant-design/icons';
import {Alert, Button, Skeleton, Space, Typography} from 'antd';
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
}) => {
    return (
        <div className='app-page'>
            <div className='app-page__header'>
                <div className='app-page__heading'>
                    <Typography.Title level={2}>{props.title}</Typography.Title>
                    {props.subtitle && <Typography.Text type='secondary'>{props.subtitle}</Typography.Text>}
                </div>
                <Space className='app-page__actions' wrap={true}>
                    {props.onRefresh && <Button icon={<ReloadOutlined />} onClick={props.onRefresh} />}
                    {props.extra}
                </Space>
            </div>
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

export const Section = (props: {title: string; extra?: React.ReactNode; children: React.ReactNode}) => (
    <section className='section-panel'>
        <div className='section-panel__header'>
            <Typography.Title level={5}>{props.title}</Typography.Title>
            {props.extra && <div className='section-panel__extra'>{props.extra}</div>}
        </div>
        <div className='section-panel__body'>{props.children}</div>
    </section>
);
