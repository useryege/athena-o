import type {ColumnsType} from 'antd/es/table';
import {AppPage, CardTitle, ResourceTable, useAsyncData} from '../components';
import {formatBeijingDateTime, formatBeijingUnixSeconds} from '../shared/format';
import {services} from '../shared/services';
import {MarketRadarHotMarketItem, MarketRadarMoverMarketItem, MarketRadarRealtimeMarketItem} from '../shared/services/market-radar-service';
import {fmt, fmtNumber} from './shared';

const MarketRadarListPage = <T extends MarketRadarHotMarketItem | MarketRadarRealtimeMarketItem | MarketRadarMoverMarketItem>(props: {
    title: string;
    load: () => Promise<{items: T[]; fetchedAt?: number; stale?: boolean; [key: string]: any}> & {abort?: () => void};
}) => {
    const data = useAsyncData(props.load, []);
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
            <ResourceTable rowKey={item => (item as any).conditionId} items={data.data?.items || []} columns={columns} loading={data.loading} />
        </AppPage>
    );
};

export const MarketRadarHotPage = () => <MarketRadarListPage title='Hot Markets' load={() => services.marketRadar.listHotMarkets(100)} />;
export const MarketRadarRealtimePage = () => <MarketRadarListPage title='Realtime Markets' load={() => services.marketRadar.listRealtimeMarkets(100)} />;
export const MarketRadarMoversPage = () => <MarketRadarListPage title='Market Movers' load={() => services.marketRadar.listMovers(100)} />;
