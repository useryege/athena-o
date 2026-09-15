import type {ColumnsType} from 'antd/es/table';
import * as React from 'react';
import {AppPage, ResourceTable, Section, useAsyncData} from '../../components';
import {formatBeijingUnixSeconds} from '../../shared/format';
import {DEFAULT_PAGE_SIZE, PAGE_SIZE_OPTIONS} from '../../shared/pagination';
import {memberServices as services} from '../services';
import {MarketRadarHotMarketItem, MarketRadarMoverMarketItem, MarketRadarRealtimeMarketItem} from '../../shared/services/market-radar-service';
import {HotMarketRecord, RealtimeMarketRecord, MoverMarketRecord} from './market-radar-presentation';

type Snapshot<T> = {
    items: T[];
    fetchedAt?: number;
    stale?: boolean;
    connected?: boolean;
    candidateCount?: number;
    monitoredMarkets?: number;
    subscribedMarkets?: number;
    monitoredTokens?: number;
    subscribedTokens?: number;
    lastEventAt?: number;
};
const MarketRadarListPage = <T extends MarketRadarHotMarketItem | MarketRadarRealtimeMarketItem | MarketRadarMoverMarketItem>(props: {
    title: string;
    subtitle: string;
    section: string;
    description: string;
    record: (item: T) => React.ReactNode;
    load: () => Promise<Snapshot<T>> & {abort?: () => void};
}) => {
    const data = useAsyncData(props.load, []);
    const [page, setPage] = React.useState(1);
    const [pageSize, setPageSize] = React.useState(DEFAULT_PAGE_SIZE);
    const items = data.data?.items || [];
    const lastPage = Math.max(1, Math.ceil(items.length / pageSize));
    React.useEffect(() => {
        if (page > lastPage) setPage(lastPage);
    }, [lastPage, page]);
    const columns: ColumnsType<T> = [{title: props.section, render: (_value: unknown, item: T) => props.record(item)}];
    const snapshot = data.data;
    return (
        <div className='market-intelligence-page radar-page'>
            <AppPage title={props.title} subtitle={props.subtitle} loading={data.loading} error={data.error} onRefresh={data.reload}>
                {snapshot && (
                    <div className='radar-snapshot'>
                        <span>Fetched {formatBeijingUnixSeconds(snapshot.fetchedAt) || 'Unavailable'} · UTC+8</span>
                        <span className={snapshot.stale || data.error ? 'radar-warmup' : 'radar-connected'}>
                            {snapshot.stale || data.error ? 'Stale snapshot' : 'Current snapshot'}
                        </span>
                        {snapshot.connected !== undefined && (
                            <span className={snapshot.connected ? 'radar-connected' : 'radar-warmup'}>Sampling {snapshot.connected ? 'connected' : 'disconnected'}</span>
                        )}
                        <span>
                            {snapshot.candidateCount ?? 'Unknown'} candidates · {snapshot.monitoredMarkets ?? snapshot.subscribedMarkets ?? 'Unknown'} monitored markets
                        </span>
                    </div>
                )}
                {snapshot && (
                    <Section title={props.section} extra={`${items.length} markets`}>
                        <p className='radar-description'>{props.description}</p>
                        <div className='radar-list'>
                            <ResourceTable
                                rowKey='conditionId'
                                label={props.section}
                                items={items.slice((page - 1) * pageSize, page * pageSize)}
                                columns={columns}
                                compactRender={props.record}
                                loading={data.loading}
                                page={page}
                                pageSize={pageSize}
                                pageSizeOptions={PAGE_SIZE_OPTIONS}
                                total={items.length}
                                onPageChange={(next, size) => {
                                    setPage(next);
                                    setPageSize(size);
                                }}
                            />
                        </div>
                    </Section>
                )}
                {snapshot && snapshot.lastEventAt !== undefined && (
                    <p className='radar-description'>
                        Last sampled {formatBeijingUnixSeconds(snapshot.lastEventAt)} · UTC+8 · {snapshot.monitoredTokens ?? snapshot.subscribedTokens ?? 'Unknown'} monitored
                        tokens. Sampling uses periodic snapshots.
                    </p>
                )}
            </AppPage>
        </div>
    );
};
export const MarketRadarHotPage = () => (
    <MarketRadarListPage
        title='Hot Markets'
        subtitle='Active markets ranked by 24-hour trading activity.'
        section='Market activity'
        description='Volume and liquidity in USD. Prices show implied probability.'
        record={item => <HotMarketRecord item={item} />}
        load={() => services.marketRadar.listHotMarkets(100)}
    />
);
export const MarketRadarRealtimePage = () => (
    <MarketRadarListPage
        title='Realtime Markets'
        subtitle='Outcome prices and short-window changes from sampled market snapshots.'
        section='Price windows'
        description='Changes are percentage points (pp). Warmup needs more samples.'
        record={item => <RealtimeMarketRecord item={item} />}
        load={() => services.marketRadar.listRealtimeMarkets(100)}
    />
);
export const MarketRadarMoversPage = () => (
    <MarketRadarListPage
        title='Market Movers'
        subtitle='Markets ranked by the strongest weighted outcome-price movement.'
        section='Ranked movements'
        description='Score = |1m change| + 0.6 × |5m change| + 0.3 × |15m change|. Only fresh, non-warmup movements qualify. Direction follows the largest weighted contribution.'
        record={item => <MoverMarketRecord item={item} />}
        load={() => services.marketRadar.listMovers(100)}
    />
);
