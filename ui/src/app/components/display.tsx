import {Col, Descriptions, Flex, Input, Row, Tag, Typography} from 'antd';
import * as React from 'react';

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
    <Descriptions className='key-value-grid' bordered={true} size='small' column={props.columns || 3}>
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
            <Col key={item.label} span={4}>
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
