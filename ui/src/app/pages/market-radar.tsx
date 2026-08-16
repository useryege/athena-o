import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {AppPage, CardTitle, ResourceTable, useAsyncData} from '../components';
import {formatBeijingDateTime, formatBeijingUnixSeconds} from '../shared/format';
import {DEFAULT_PAGE_SIZE, PAGE_SIZE_OPTIONS} from '../shared/pagination';
import {services} from '../shared/services';
import {MarketRadarHotMarketItem, MarketRadarMoverMarketItem, MarketRadarRealtimeMarketItem} from '../shared/services/market-radar-service';
import {fmt, fmtNumber} from './shared';

const MarketRadarListPage = <T extends MarketRadarHotMarketItem | MarketRadarRealtimeMarketItem | MarketRadarMoverMarketItem>(props: {
    title: string;
    load: () => Promise<{items: T[]; fetchedAt?: number; stale?: boolean; [key: string]: any}> & {abort?: () => void};
}) => {
    const data = useAsyncData(props.load, []);
    const [page, setPage] = React.useState(1);
    const [pageSize, setPageSize] = React.useState(DEFAULT_PAGE_SIZE);
    const items = data.data?.items || [];
    const lastPage = Math.max(1, Math.ceil(items.length / pageSize));
    React.useEffect(() => {
        if (page > lastPage) {
            setPage(lastPage);
        }
    }, [lastPage, page]);
    const pagedItems = React.useMemo(() => items.slice((page - 1) * pageSize, page * pageSize), [items, page, pageSize]);
    const columns: ColumnsType<T> = [
        {title: 'Market', render: item => <CardTitle title={(item as any).question} subtitle={(item as any).eventSlug} image={(item as any).image} />},
        {title: 'Volume', render: item => fmtNumber((item as any).volume24hr || (item as any).volumeNum)},
        {title: 'Liquidity', render: item => fmtNumber((item as any).liquidityNum)},
        {title: 'Spread', render: item => fmt((item as any).spread)},
        {title: 'Updated', render: item => formatBeijingDateTime((item as any).updatedAt) || '-'}
    ];
    return (
        <AppPage
            title={props.title}
            subtitle={`Fetched ${formatBeijingUnixSeconds(data.data?.fetchedAt) || '-'} ${data.data?.stale ? '(stale)' : ''}`}
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}>
            <ResourceTable
                rowKey={item => (item as any).conditionId}
                items={pagedItems}
                columns={columns}
                loading={data.loading}
                page={page}
                pageSize={pageSize}
                pageSizeOptions={PAGE_SIZE_OPTIONS}
                total={items.length}
                onPageChange={(nextPage, nextPageSize) => {
                    setPage(nextPage);
                    setPageSize(nextPageSize);
                }}
            />
        </AppPage>
    );
};

export const MarketRadarHotPage = () => <MarketRadarListPage title='Hot Markets' load={() => services.marketRadar.listHotMarkets(100)} />;
export const MarketRadarRealtimePage = () => <MarketRadarListPage title='Realtime Markets' load={() => services.marketRadar.listRealtimeMarkets(100)} />;
export const MarketRadarMoversPage = () => <MarketRadarListPage title='Market Movers' load={() => services.marketRadar.listMovers(100)} />;
