import {Select, Space} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import {AppPage, MetricRow, ResponsiveResourceList, useAsyncData} from '../components';
import {services} from '../../shared/services';
import {DEFAULT_WORM_MARKET_CATEGORY, DEFAULT_WORM_MARKET_SORT, WormMarketItem} from '../../shared/services/worm-service';
import {boolTag} from './shared';
import {WormMarketSummary} from './worm-shared';

export const WormPage = () => {
    const data = useAsyncData(
        () =>
            services.worm.listMarkets({
                limit: 50,
                sortOption: DEFAULT_WORM_MARKET_SORT,
                categorySlug: DEFAULT_WORM_MARKET_CATEGORY
            }),
        []
    );
    const columns: ColumnsType<WormMarketItem> = [
        {
            title: 'Market',
            render: item => <WormMarketSummary item={item} />
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
                        disabled={true}
                        value={DEFAULT_WORM_MARKET_SORT}
                        style={{width: 160}}
                        options={['new', 'trending', 'ending_soon', 'leverage'].map(value => ({value, label: value}))}
                    />
                    <Select
                        disabled={true}
                        value={DEFAULT_WORM_MARKET_CATEGORY}
                        style={{width: 150}}
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
                    <div>
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
        </AppPage>
    );
};
