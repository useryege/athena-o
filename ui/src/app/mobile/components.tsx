import {FilterOutlined, ReloadOutlined} from '@ant-design/icons';
import {Alert, Button, Card, Col, Descriptions, Drawer, Empty, Flex, Grid, Input, Pagination, Row, Skeleton, Space, Table, Tag, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';

export interface AsyncState<T> {
    data?: T;
    loading: boolean;
    error?: Error;
    reload: () => void;
}

export const useAsyncData = <T,>(load: () => Promise<T> & {abort?: () => void}, deps: React.DependencyList): AsyncState<T> => {
    const [data, setData] = React.useState<T>();
    const [loading, setLoading] = React.useState(true);
    const [error, setError] = React.useState<Error>();
    const [version, setVersion] = React.useState(0);

    React.useEffect(() => {
        let active = true;
        const req = load();
        setLoading(true);
        setError(undefined);
        req.then(
            next => {
                if (active) {
                    setData(next);
                    setLoading(false);
                }
            },
            err => {
                if (active) {
                    setError(err instanceof Error ? err : new Error(String(err?.message || err)));
                    setLoading(false);
                }
            }
        );
        return () => {
            active = false;
            req.abort?.();
        };
    }, [...deps, version]);

    return {data, loading, error, reload: () => setVersion(current => current + 1)};
};

export const useBreakpoint = () => {
    const screens = Grid.useBreakpoint();
    return {
        isMobile: !screens.md,
        isTablet: screens.md && !screens.lg
    };
};

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
    const {isMobile} = useBreakpoint();
    const [filtersOpen, setFiltersOpen] = React.useState(false);
    return (
        <div className='app-page'>
            <div className='app-page__header'>
                <div className='app-page__heading'>
                    <Typography.Title level={2}>{props.title}</Typography.Title>
                    {props.subtitle && <Typography.Text type='secondary'>{props.subtitle}</Typography.Text>}
                </div>
                <Space className='app-page__actions' wrap={true}>
                    {props.filters && isMobile && <Button icon={<FilterOutlined />} onClick={() => setFiltersOpen(true)} />}
                    {props.onRefresh && <Button icon={<ReloadOutlined />} onClick={props.onRefresh} />}
                    {props.extra}
                </Space>
            </div>
            {props.filters && !isMobile && <div className='app-page__filters'>{props.filters}</div>}
            {props.error && <Alert className='app-page__alert' type='error' title='Request failed' description={props.error.message} showIcon={true} />}
            {props.loading && !props.children ? <Skeleton active={true} /> : props.children}
            <Drawer title='Filters' open={filtersOpen} onClose={() => setFiltersOpen(false)} placement='right' size='min(92vw, 360px)'>
                {props.filters}
            </Drawer>
        </div>
    );
};

export const ResponsiveResourceList = <T,>(props: {
    rowKey: keyof T | ((record: T) => React.Key);
    items: T[];
    columns: ColumnsType<T>;
    loading?: boolean;
    emptyText?: string;
    card: (record: T) => React.ReactNode;
    page?: number;
    pageSize?: number;
    total?: number;
    onPageChange?: (page: number, pageSize: number) => void;
}) => {
    const {isMobile} = useBreakpoint();
    if (isMobile) {
        return (
            <div className='resource-list resource-list--mobile'>
                {props.loading && <Skeleton active={true} />}
                {!props.loading && props.items.length === 0 && <Empty description={props.emptyText || 'No data'} />}
                {props.items.map(item => {
                    const key = typeof props.rowKey === 'function' ? props.rowKey(item) : (item[props.rowKey] as React.Key);
                    return (
                        <Card key={key} className='resource-card' size='small'>
                            {props.card(item)}
                        </Card>
                    );
                })}
                {props.total !== undefined && props.onPageChange && (
                    <Pagination className='resource-list__pager' current={props.page} pageSize={props.pageSize} total={props.total} simple={true} onChange={props.onPageChange} />
                )}
            </div>
        );
    }
    return (
        <Table<T>
            className='resource-table'
            rowKey={props.rowKey as any}
            columns={props.columns as any}
            dataSource={props.items}
            loading={props.loading}
            pagination={
                props.total !== undefined && props.onPageChange
                    ? {current: props.page, pageSize: props.pageSize, total: props.total, showSizeChanger: true, onChange: props.onPageChange}
                    : false
            }
            scroll={{x: 'max-content'}}
        />
    );
};

export const CardTitle = (props: {title: React.ReactNode; subtitle?: React.ReactNode; image?: string; tags?: React.ReactNode}) => (
    <div className='card-title'>
        {props.image && <img src={props.image} alt='' />}
        <div className='card-title__main'>
            <Typography.Text strong={true}>{props.title || '-'}</Typography.Text>
            {props.subtitle && <Typography.Text type='secondary'>{props.subtitle}</Typography.Text>}
        </div>
        {props.tags}
    </div>
);

export const KeyValueGrid = (props: {items: Array<{label: React.ReactNode; value: React.ReactNode}>; columns?: number}) => (
    <Descriptions className='key-value-grid' bordered={true} size='small' column={{xs: 1, sm: 1, md: props.columns || 2, lg: props.columns || 3}}>
        {props.items.map(item => (
            <Descriptions.Item key={String(item.label)} label={item.label}>
                <span className='break-value'>{item.value ?? '-'}</span>
            </Descriptions.Item>
        ))}
    </Descriptions>
);

export const MetricRow = (props: {items: Array<{label: string; value: React.ReactNode; tone?: 'good' | 'bad' | 'warn'}>}) => (
    <Row gutter={[8, 8]} className='metric-row'>
        {props.items.map(item => (
            <Col key={item.label} xs={12} sm={8} md={6} lg={4}>
                <div className={`metric metric--${item.tone || 'neutral'}`}>
                    <span>{item.label}</span>
                    <strong>{item.value ?? '-'}</strong>
                </div>
            </Col>
        ))}
    </Row>
);

export const StatusTag = (props: {value?: React.ReactNode; positive?: boolean; negative?: boolean}) => (
    <Tag color={props.negative ? 'red' : props.positive ? 'green' : 'default'}>{props.value ?? '-'}</Tag>
);

export const SearchBar = (props: {value?: string; placeholder?: string; onChange: (value: string) => void; onSearch?: () => void}) => (
    <Input.Search
        className='search-bar'
        allowClear={true}
        value={props.value}
        placeholder={props.placeholder || 'Search'}
        onChange={event => props.onChange(event.target.value)}
        onSearch={props.onSearch}
    />
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

export const InlineActions = (props: {children: React.ReactNode}) => (
    <Flex className='inline-actions' gap={8} wrap='wrap'>
        {props.children}
    </Flex>
);

export const TruncatedText = (props: {value?: React.ReactNode; copyable?: boolean}) => (
    <Typography.Text className='truncate-text' copyable={props.copyable}>
        {props.value || '-'}
    </Typography.Text>
);
