import {Button, Collapse, Modal, Select, Space, Tag, Typography} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {AppPage, KeyValueGrid, MetricRow, ResponsiveResourceList, TruncatedText, useAsyncData} from '../components';
import {services} from '../../shared/services';
import {WormMarketDetail, WormMarketItem} from '../../shared/services/worm-service';
import {boolTag} from './shared';
import {WormMarketSummary, wormMarketLogo} from './worm-shared';

export const WormPage = () => {
    const [sortOption, setSortOption] = React.useState('trending');
    const [categorySlug, setCategorySlug] = React.useState('all');
    const [detailId, setDetailId] = React.useState('');
    const data = useAsyncData(() => services.worm.listMarkets({limit: 50, sortOption: sortOption as any, categorySlug: categorySlug as any}), [sortOption, categorySlug]);
    const detail = useAsyncData<WormMarketDetail>(() => (detailId ? services.worm.getMarket(detailId) : Promise.resolve(null as WormMarketDetail)) as any, [detailId]);
    const columns: ColumnsType<WormMarketItem> = [
        {
            title: 'Market',
            render: item => <WormMarketSummary item={item} onOpen={() => setDetailId(item.conditionId)} />
        },
        {title: 'Category', dataIndex: 'category'},
        {title: 'State', dataIndex: 'state'},
        {title: 'Last Price', dataIndex: 'lastTradePrice'},
        {title: 'Margin', render: item => boolTag(item.marginEnabled)}
    ];
    return (
        <AppPage
            title='Worm'
            subtitle='Polymarket-derived markets for Worm module'
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}
            filters={
                <Space wrap={true}>
                    <Select
                        value={sortOption}
                        style={{width: 160}}
                        onChange={setSortOption}
                        options={['new', 'trending', 'ending_soon', 'leverage'].map(value => ({value, label: value}))}
                    />
                    <Select
                        value={categorySlug}
                        style={{width: 150}}
                        onChange={setCategorySlug}
                        options={['all', 'politics', 'sports', 'crypto', 'tech', 'finance', 'wtf'].map(value => ({value, label: value}))}
                    />
                </Space>
            }>
            <ResponsiveResourceList
                rowKey='conditionId'
                items={data.data?.items || []}
                columns={columns}
                loading={data.loading}
                card={item => (
                    <div onClick={() => setDetailId(item.conditionId)}>
                        <WormMarketSummary item={item} />
                        <MetricRow
                            items={[
                                {label: 'Price', value: item.lastTradePrice},
                                {label: 'State', value: item.state},
                                {label: 'Created', value: item.created}
                            ]}
                        />
                    </div>
                )}
            />
            <Modal
                className='worm-detail-modal'
                open={!!detailId}
                title='Market Detail'
                onCancel={() => setDetailId('')}
                footer={<Button onClick={() => setDetailId('')}>Close</Button>}
                width={860}>
                <WormDetail detail={detail.data} />
            </Modal>
        </AppPage>
    );
};

const WormDetail = (props: {detail?: WormMarketDetail}) => {
    const detail = props.detail;
    if (!detail) {
        return null;
    }
    const image = wormMarketLogo(detail.market);
    const rulesText = formatJSONText(detail.rules);
    return (
        <Space className='worm-detail' orientation='vertical' style={{width: '100%'}}>
            <div className='worm-detail__header'>
                {image && <img src={image} alt='' />}
                <div className='worm-detail__heading'>
                    <Typography.Title level={4}>{detail.market.title || '-'}</Typography.Title>
                    <Space className='worm-detail__meta' wrap={true}>
                        {detail.market.eventTitle && <Tag>{detail.market.eventTitle}</Tag>}
                        {detail.market.category && <Tag>{detail.market.category}</Tag>}
                        {detail.market.state && <Tag>{detail.market.state}</Tag>}
                        {detail.market.marginEnabled && <Tag color='green'>Margin</Tag>}
                    </Space>
                </div>
            </div>
            <KeyValueGrid
                columns={2}
                items={[
                    {label: 'Condition', value: <TruncatedText value={detail.market.conditionId} copyable={true} />},
                    {label: 'Resolution', value: detail.resolutionDate},
                    {label: 'Maker Fee', value: detail.makerFee},
                    {label: 'Taker Fee', value: detail.takerFee},
                    {label: 'Stale', value: boolTag(detail.stale)}
                ]}
            />
            <Collapse
                items={[
                    {key: 'rules', label: 'Rules', children: <pre className='code-block'>{rulesText || 'No rules'}</pre>},
                    {key: 'config', label: 'Config', children: <pre className='code-block'>{JSON.stringify(detail.config || {}, null, 2)}</pre>}
                ]}
            />
        </Space>
    );
};

const formatJSONText = (value?: string) => {
    const text = String(value || '').trim();
    if (!text) {
        return '';
    }
    try {
        return JSON.stringify(JSON.parse(text), null, 2);
    } catch {
        return text;
    }
};
