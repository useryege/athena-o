import {Button, Segmented, Select, Space} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {AppPage, MetricRow, ResponsiveResourceList, StatusTag, useAsyncData} from '../components';
import {services} from '../../shared/services';
import {
    DEFAULT_WORM_MARKET_CATEGORY,
    DEFAULT_WORM_MARKET_IGNORED_FILTER,
    DEFAULT_WORM_MARKET_SORT,
    WormMarketIgnoredFilter,
    WormMarketItem
} from '../../shared/services/worm-service';
import {WormMarketSummary} from './worm-shared';

const openWormMarket = (item: WormMarketItem) => {
    if (!item.conditionId) {
        return;
    }
    window.open(`https://www.worm.wtf/market/${encodeURIComponent(item.conditionId)}`, '_blank', 'noopener,noreferrer');
};

export const WormPage = () => {
    const [ignoredFilter, setIgnoredFilter] = React.useState<WormMarketIgnoredFilter>(DEFAULT_WORM_MARKET_IGNORED_FILTER);
    const [selectedRowKeys, setSelectedRowKeys] = React.useState<React.Key[]>([]);
    const data = useAsyncData(
        () =>
            services.worm.listMarkets({
                limit: 50,
                sortOption: DEFAULT_WORM_MARKET_SORT,
                categorySlug: DEFAULT_WORM_MARKET_CATEGORY,
                ignoredFilter
            }),
        [ignoredFilter]
    );
    React.useEffect(() => setSelectedRowKeys([]), [ignoredFilter]);
    const selectedConditionIds = selectedRowKeys.map(String).filter(Boolean);
    const updateIgnored = async (ignored: boolean) => {
        if (selectedConditionIds.length === 0) {
            return;
        }
        await services.worm.batchUpdateMarketsIgnored(selectedConditionIds, ignored);
        setSelectedRowKeys([]);
        data.reload();
    };
    const columns: ColumnsType<WormMarketItem> = [
        {
            title: 'Market',
            render: item => <WormMarketSummary item={item} />
        },
        {title: 'Ignored', render: item => <StatusTag value={item.ignored ? 'Ignored' : 'Active'} negative={item.ignored} positive={!item.ignored} />},
        {title: 'Last Price', dataIndex: 'lastTradePrice'}
    ];
    return (
        <AppPage
            title='Worm'
            subtitle='Polymarket-derived markets for Worm module. Hold X to select rows, ESC to clear selection.'
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}
            filters={
                <Space wrap={true}>
                    <Segmented
                        value={ignoredFilter}
                        onChange={value => setIgnoredFilter(value as WormMarketIgnoredFilter)}
                        options={[
                            {value: 'active', label: 'Active'},
                            {value: 'ignored', label: 'Ignored'},
                            {value: 'all', label: 'All'}
                        ]}
                    />
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
                    {selectedConditionIds.length > 0 && ignoredFilter !== 'ignored' && (
                        <Button danger={true} onClick={() => void updateIgnored(true)}>
                            Ignore ({selectedConditionIds.length})
                        </Button>
                    )}
                    {selectedConditionIds.length > 0 && ignoredFilter !== 'active' && (
                        <Button onClick={() => void updateIgnored(false)}>
                            Unignore ({selectedConditionIds.length})
                        </Button>
                    )}
                </Space>
            }>
            <ResponsiveResourceList
                rowKey='conditionId'
                items={data.data?.items || []}
                columns={columns}
                loading={data.loading}
                selectedRowKeys={selectedRowKeys}
                onSelectionChange={keys => setSelectedRowKeys(keys)}
                enableHoverKeyboardSelect={true}
                card={item => (
                    <div>
                        <WormMarketSummary item={item} />
                        <MetricRow
                            items={[
                                {label: 'Price', value: item.lastTradePrice},
                                {label: 'Ignored', value: item.ignored ? 'Ignored' : 'Active', tone: item.ignored ? 'bad' : 'good'},
                                {label: 'Move', value: item.livePriceChange},
                                {label: 'Created', value: item.created}
                            ]}
                        />
                    </div>
                )}
                onItemClick={openWormMarket}
            />
        </AppPage>
    );
};
