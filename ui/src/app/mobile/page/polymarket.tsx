import {LoadingOutlined, ReloadOutlined} from '@ant-design/icons';
import type {ColumnsType} from 'antd/es/table';
import {Button, Empty, Space, Tag, Typography} from 'antd';
import * as React from 'react';
import {AppPage, CardTitle, MetricRow, ResponsiveResourceList, useAsyncData} from '../components';
import {Context} from '../../shared/context';
import {services} from '../../shared/services';
import {
    PolymarketHotMarketItem,
    PolymarketMoverMarketItem,
    PolymarketRealtimeMarketItem,
    PolymarketSportsHistoryEventCardItem,
    PolymarketSportsHistorySyncStatus,
    PolymarketSportsLiveEventCardItem,
    PolymarketSportsLivePriceHistorySeriesItem
} from '../../shared/services/polymarket-service';
import {
    FifwcSportsLiveEventCard,
    LegacySportsLiveEventCard,
    SportsHistoryEventCard,
    isFifwcSportsLiveEvent,
    moneylineMarketKeys,
    sportsLiveCardHistory,
    sportsLiveSectionKey
} from './polymarket-sports-live-card';
import {fmt, fmtNumber} from './shared';

const PolymarketListPage = <T extends PolymarketHotMarketItem | PolymarketRealtimeMarketItem | PolymarketMoverMarketItem>(props: {
    title: string;
    load: () => Promise<{items: T[]; fetchedAt?: number; stale?: boolean; [key: string]: any}> & {abort?: () => void};
}) => {
    const data = useAsyncData(props.load, []);
    const columns: ColumnsType<T> = [
        {title: 'Market', render: item => <CardTitle title={(item as any).question} subtitle={(item as any).eventSlug} image={(item as any).image} />},
        {title: 'Volume', render: item => fmtNumber((item as any).volume24hr || (item as any).volumeNum)},
        {title: 'Liquidity', render: item => fmtNumber((item as any).liquidityNum)},
        {title: 'Spread', render: item => fmt((item as any).spread)},
        {title: 'Updated', dataIndex: 'updatedAt'}
    ];
    return (
        <AppPage
            title={props.title}
            subtitle={`Fetched ${fmt(data.data?.fetchedAt)} ${data.data?.stale ? '(stale)' : ''}`}
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}>
            <ResponsiveResourceList
                rowKey={item => (item as any).conditionId}
                items={data.data?.items || []}
                columns={columns}
                loading={data.loading}
                card={item => (
                    <>
                        <CardTitle title={(item as any).question} subtitle={(item as any).eventSlug} image={(item as any).image} />
                        <MetricRow
                            items={[
                                {label: 'Volume', value: fmtNumber((item as any).volume24hr || (item as any).volumeNum)},
                                {label: 'Liquidity', value: fmtNumber((item as any).liquidityNum)},
                                {label: 'Tokens', value: (item as any).tokens?.length || 0}
                            ]}
                        />
                    </>
                )}
            />
        </AppPage>
    );
};

export const PolymarketHotPage = () => <PolymarketListPage title='Polymarket Hot Markets' load={() => services.polymarket.listHotMarkets(100)} />;
export const PolymarketRealtimePage = () => <PolymarketListPage title='Polymarket Realtime' load={() => services.polymarket.listRealtimeMarkets(100)} />;
export const PolymarketMoversPage = () => <PolymarketListPage title='Polymarket Movers' load={() => services.polymarket.listMovers(100)} />;

const sportsLiveRefreshIntervalMs = 3000;

type SportsLiveSection = {
    key: string;
    title: string;
    items: PolymarketSportsLiveEventCardItem[];
};

const sportsLiveSectionTitle = (key: string) => key.toUpperCase();

const sportsLiveSections = (items: PolymarketSportsLiveEventCardItem[] = []): SportsLiveSection[] => {
    const sections: SportsLiveSection[] = [];
    const sectionByKey = new Map<string, SportsLiveSection>();
    items.forEach(item => {
        const key = sportsLiveSectionKey(item);
        let section = sectionByKey.get(key);
        if (!section) {
            section = {key, title: sportsLiveSectionTitle(key), items: []};
            sectionByKey.set(key, section);
            sections.push(section);
        }
        section.items.push(item);
    });
    return sections;
};

export const PolymarketSportsLivePage = () => {
    const events = useAsyncData(() => services.polymarket.listSportsLiveEvents(200), []);
    const marketKeys = React.useMemo(() => moneylineMarketKeys(events.data?.items), [events.data?.items]);
    const marketKeySignature = React.useMemo(() => marketKeys.join('|'), [marketKeys]);
    const history = useAsyncData(() => {
        if (marketKeys.length === 0) {
            return Promise.resolve({items: []}) as Promise<{items: PolymarketSportsLivePriceHistorySeriesItem[]}> & {abort?: () => void};
        }
        return services.polymarket.batchGetSportsLivePriceHistory(marketKeys, 360);
    }, [marketKeySignature]);
    const eventsReloadRef = React.useRef(events.reload);
    const historyReloadRef = React.useRef(history.reload);
    const marketKeyCountRef = React.useRef(marketKeys.length);
    eventsReloadRef.current = events.reload;
    historyReloadRef.current = history.reload;
    marketKeyCountRef.current = marketKeys.length;
    const reloadAll = React.useCallback(() => {
        eventsReloadRef.current();
        if (marketKeyCountRef.current > 0) {
            historyReloadRef.current();
        }
    }, []);

    React.useEffect(() => {
        const timer = window.setInterval(reloadAll, sportsLiveRefreshIntervalMs);
        return () => window.clearInterval(timer);
    }, [reloadAll]);

    const historyByMarketKey = React.useMemo(() => {
        const out = new Map<string, PolymarketSportsLivePriceHistorySeriesItem[]>();
        if (history.error) {
            return out;
        }
        (history.data?.items || []).forEach(item => {
            const items = out.get(item.marketKey) || [];
            items.push(item);
            out.set(item.marketKey, items);
        });
        return out;
    }, [history.data?.items, history.error]);
    const items = events.data?.items || [];
    const sections = React.useMemo(() => sportsLiveSections(items), [items]);
    return (
        <AppPage
            title='Sports Live'
            subtitle={`Fetched ${fmt(events.data?.fetchedAt)} ${events.data?.stale ? '(stale)' : ''}`}
            loading={events.loading}
            error={events.error}
            onRefresh={reloadAll}>
            <div className='sports-live-sections'>
                {!events.loading && items.length === 0 && <Empty description='No data' />}
                {sections.map(section => (
                    <section className='sports-live-section' key={section.key}>
                        <div className='sports-live-section__header'>
                            <Typography.Title level={5}>{section.title}</Typography.Title>
                            <Typography.Text className='sports-live-section__count' type='secondary'>
                                {section.items.length} events
                            </Typography.Text>
                        </div>
                        <div className='sports-live-section__body'>
                            {section.items.map(item => {
                                const Card = isFifwcSportsLiveEvent(item) ? FifwcSportsLiveEventCard : LegacySportsLiveEventCard;
                                return <Card item={item} history={sportsLiveCardHistory(item, historyByMarketKey)} key={item.eventKey} />;
                            })}
                        </div>
                    </section>
                ))}
            </div>
        </AppPage>
    );
};

const sportsHistoryLeagues = ['ATP', 'WTA'];
const sportsHistorySyncPollingIdleMs = 10000;
const sportsHistorySyncPollingActiveMs = 2000;

const sportsHistorySyncTime = (value?: number) => (value ? new Date(value * 1000).toLocaleString() : 'Not available');

const SportsHistorySyncStatusBar = (props: {status?: PolymarketSportsHistorySyncStatus; refreshing: boolean}) => {
    const state = props.refreshing ? 'syncing' : props.status?.state || 'idle';
    const labels = {idle: 'Idle', syncing: 'Syncing', succeeded: 'Succeeded', failed: 'Failed'};
    const colors = {idle: 'default', syncing: 'blue', succeeded: 'green', failed: 'red'} as const;
    let detail = props.refreshing ? 'Starting synchronization' : `Last successful ${sportsHistorySyncTime(props.status?.lastSuccessAt)}`;
    if (!props.refreshing && state === 'syncing') {
        detail = `Started ${sportsHistorySyncTime(props.status?.startedAt)}`;
    } else if (state === 'succeeded') {
        detail = `Completed ${sportsHistorySyncTime(props.status?.completedAt)}`;
    } else if (state === 'failed') {
        detail = `Failed ${sportsHistorySyncTime(props.status?.completedAt)}`;
    }
    return (
        <div className={`sports-history-sync sports-history-sync--${state}`}>
            <Space size={8} wrap={true}>
                <Typography.Text strong={true}>Sync</Typography.Text>
                <Tag color={colors[state]} icon={state === 'syncing' ? <LoadingOutlined spin={true} /> : undefined}>
                    {labels[state]}
                </Tag>
                <Typography.Text type='secondary'>{detail}</Typography.Text>
            </Space>
            {state === 'failed' && props.status?.errorMessage && <Typography.Text type='danger'>{props.status.errorMessage}</Typography.Text>}
        </div>
    );
};

export const PolymarketSportsHistoryPage = (props: {canRefresh: boolean}) => {
    const ctx = React.useContext(Context);
    const events = useAsyncData(() => services.polymarket.listSportsHistoryEvents(200), []);
    const syncStatus = useAsyncData(() => services.polymarket.getSportsHistorySyncStatus(), []);
    const [refreshing, setRefreshing] = React.useState(false);
    const marketKeys = React.useMemo(() => moneylineMarketKeys(events.data?.items), [events.data?.items]);
    const marketKeySignature = React.useMemo(() => marketKeys.join('|'), [marketKeys]);
    const history = useAsyncData(() => {
        if (marketKeys.length === 0) {
            return Promise.resolve({items: []}) as Promise<{items: PolymarketSportsLivePriceHistorySeriesItem[]}> & {abort?: () => void};
        }
        return services.polymarket.batchGetSportsHistoryPriceHistory(marketKeys, 360);
    }, [marketKeySignature]);
    const historyByMarketKey = React.useMemo(() => {
        const out = new Map<string, PolymarketSportsLivePriceHistorySeriesItem[]>();
        if (history.error) {
            return out;
        }
        (history.data?.items || []).forEach(item => {
            const items = out.get(item.marketKey) || [];
            items.push(item);
            out.set(item.marketKey, items);
        });
        return out;
    }, [history.data?.items, history.error]);
    const items = events.data?.items || [];
    const byLeague = React.useMemo(() => {
        const groups = new Map<string, PolymarketSportsHistoryEventCardItem[]>();
        sportsHistoryLeagues.forEach(league => groups.set(league, []));
        items.forEach(item => {
            const league = item.league.toUpperCase();
            const group = groups.get(league);
            if (group) {
                group.push(item);
            }
        });
        groups.forEach(group => group.sort((left, right) => (right.startTime || '').localeCompare(left.startTime || '')));
        return groups;
    }, [items]);
    const reloadAll = React.useCallback(() => {
        events.reload();
        if (marketKeys.length > 0) {
            history.reload();
        }
    }, [events.reload, history.reload, marketKeys.length]);
    const syncStatusReloadRef = React.useRef(syncStatus.reload);
    syncStatusReloadRef.current = syncStatus.reload;
    const serverSyncing = syncStatus.data?.state === 'syncing';
    const pollingActive = refreshing || serverSyncing;

    React.useEffect(() => {
        const interval = pollingActive ? sportsHistorySyncPollingActiveMs : sportsHistorySyncPollingIdleMs;
        const timer = window.setInterval(() => syncStatusReloadRef.current(), interval);
        return () => window.clearInterval(timer);
    }, [pollingActive]);

    const refresh = React.useCallback(async () => {
        setRefreshing(true);
        syncStatus.reload();
        try {
            await services.polymarket.refreshSportsHistory();
            ctx.notifications.success('Sports history refreshed');
        } catch (err: any) {
            ctx.notifications.error('Sports history refresh failed', err?.message || 'Could not refresh sports history data.');
        } finally {
            setRefreshing(false);
            syncStatus.reload();
            reloadAll();
        }
    }, [ctx.notifications, reloadAll, syncStatus.reload]);

    return (
        <AppPage
            title='Sports History'
            subtitle={`Last 72 hours · Fetched ${fmt(events.data?.fetchedAt)} ${events.data?.stale ? '(stale)' : ''}`}
            loading={events.loading}
            error={events.error || syncStatus.error}
            extra={
                props.canRefresh ? (
                    <Button type='primary' icon={<ReloadOutlined />} loading={refreshing || serverSyncing} disabled={refreshing || serverSyncing} onClick={refresh}>
                        Refresh data
                    </Button>
                ) : undefined
            }>
            <SportsHistorySyncStatusBar status={syncStatus.data} refreshing={refreshing} />
            <div className='sports-live-sections sports-history-sections'>
                {!events.loading && items.length === 0 && <Empty description='No data' />}
                {sportsHistoryLeagues.map(league => {
                    const leagueItems = byLeague.get(league) || [];
                    if (leagueItems.length === 0) {
                        return null;
                    }
                    return (
                        <section className='sports-live-section' key={league}>
                            <div className='sports-live-section__header'>
                                <Typography.Title level={5}>{league}</Typography.Title>
                                <Typography.Text className='sports-live-section__count' type='secondary'>
                                    {leagueItems.length} events
                                </Typography.Text>
                            </div>
                            <div className='sports-live-section__body'>
                                {leagueItems.map(item => (
                                    <SportsHistoryEventCard item={item} history={sportsLiveCardHistory(item, historyByMarketKey)} key={item.eventKey} />
                                ))}
                            </div>
                        </section>
                    );
                })}
            </div>
        </AppPage>
    );
};
