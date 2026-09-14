import {LoadingOutlined, ReloadOutlined} from '@ant-design/icons';
import {Button, Empty, Space, Tag, Typography} from 'antd';
import * as React from 'react';
import {AppPage, useAsyncData} from '../../components';
import {Context, useAuthorization} from '../../shared/context';
import {AccountDataModule} from '../../shared/access-modules';
import {formatBeijingUnixSeconds} from '../../shared/format';
import {memberServices as services} from '../services';
import {SportsHistorySyncStatus} from '../../shared/services/sports-history-service';
import {SportsHistoryEventCardItem, SportsPriceHistorySeriesItem} from '../../shared/services/sports-models';
import {SportsHistoryEventCard, moneylineMarketKeys, sportsLiveCardHistory} from './sports-market-card';

const sportsHistoryLeagues = ['ATP', 'WTA'];
const sportsHistorySyncPollingIdleMs = 10000;
const sportsHistorySyncPollingActiveMs = 2000;

const sportsHistorySyncTime = (value?: number) => formatBeijingUnixSeconds(value) || 'Not available';

const SportsHistorySyncStatusBar = (props: {status?: SportsHistorySyncStatus; refreshing: boolean}) => {
    const state = props.refreshing ? 'syncing' : props.status?.state || 'idle';
    const labels = {idle: 'Idle', syncing: 'Syncing', succeeded: 'Succeeded', failed: 'Failed'};
    const colors = {idle: 'default', syncing: 'processing', succeeded: 'success', failed: 'error'} as const;
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

export const SportsHistoryPage = () => {
    const authorization = useAuthorization();
    const canWrite = authorization.canWrite(AccountDataModule.SportsHistory);
    const canWriteRef = React.useRef(canWrite);
    canWriteRef.current = canWrite;
    const activeRequest = React.useRef(0);
    React.useEffect(
        () => () => {
            activeRequest.current += 1;
            canWriteRef.current = false;
        },
        []
    );
    const ctx = React.useContext(Context);
    const events = useAsyncData(() => services.sportsHistory.listEvents(200), []);
    const syncStatus = useAsyncData(() => services.sportsHistory.getSyncStatus(), []);
    const [refreshing, setRefreshing] = React.useState(false);
    const [scratchMode, setScratchMode] = React.useState(false);
    const [scratchResetVersion, setScratchResetVersion] = React.useState(0);
    const marketKeys = React.useMemo(() => moneylineMarketKeys(events.data?.items), [events.data?.items]);
    const marketKeySignature = React.useMemo(() => marketKeys.join('|'), [marketKeys]);
    const history = useAsyncData(() => {
        if (marketKeys.length === 0) {
            return Promise.resolve({items: []}) as Promise<{items: SportsPriceHistorySeriesItem[]}> & {abort?: () => void};
        }
        return services.sportsHistory.batchGetPriceHistories(marketKeys, 360);
    }, [marketKeySignature]);
    const historyByMarketKey = React.useMemo(() => {
        const out = new Map<string, SportsPriceHistorySeriesItem[]>();
        (history.data?.items || []).forEach(item => {
            const items = out.get(item.marketKey) || [];
            items.push(item);
            out.set(item.marketKey, items);
        });
        return out;
    }, [history.data?.items, history.error]);
    const items = events.data?.items || [];
    const byLeague = React.useMemo(() => {
        const groups = new Map<string, SportsHistoryEventCardItem[]>();
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
        if (!canWrite) {
            activeRequest.current += 1;
            setRefreshing(false);
        }
    }, [canWrite]);

    React.useEffect(() => {
        const interval = pollingActive ? sportsHistorySyncPollingActiveMs : sportsHistorySyncPollingIdleMs;
        const timer = window.setInterval(() => syncStatusReloadRef.current(), interval);
        return () => window.clearInterval(timer);
    }, [pollingActive]);

    const refresh = React.useCallback(async () => {
        if (!canWrite) {
            return;
        }
        const request = ++activeRequest.current;
        setRefreshing(true);
        syncStatus.reload();
        try {
            await services.sportsHistory.refresh();
            if (request !== activeRequest.current || !canWriteRef.current) return;
            ctx.notifications.success('Sports history refreshed');
        } catch (err: any) {
            if (request === activeRequest.current && canWriteRef.current) {
                ctx.notifications.error('Sports history refresh failed', err?.message || 'Could not refresh sports history data.');
            }
        } finally {
            if (request === activeRequest.current && canWriteRef.current) {
                setRefreshing(false);
                syncStatus.reload();
                reloadAll();
            }
        }
    }, [canWrite, ctx.notifications, reloadAll, syncStatus.reload]);
    const toggleScratchMode = React.useCallback((checked: boolean) => {
        setScratchMode(checked);
        if (checked) {
            setScratchResetVersion(version => version + 1);
        }
    }, []);
    const resetScratch = React.useCallback(() => setScratchResetVersion(version => version + 1), []);

    return (
        <div className='market-intelligence-page sports-page'>
            <AppPage
                title='Sports History'
                subtitle='Completed ATP and WTA events from the last 72 hours.'
                onRefresh={reloadAll}
                loading={events.loading}
                error={events.error || syncStatus.error || history.error}
                extra={
                    <Space className='sports-history-actions' wrap={true}>
                        <Button aria-pressed={scratchMode} onClick={() => toggleScratchMode(!scratchMode)}>
                            Scratch
                        </Button>
                        <Button icon={<ReloadOutlined />} disabled={!scratchMode} onClick={resetScratch}>
                            Reset
                        </Button>
                        {canWrite && (
                            <Button type='primary' icon={<ReloadOutlined />} loading={refreshing || serverSyncing} disabled={refreshing || serverSyncing} onClick={refresh}>
                                Refresh data
                            </Button>
                        )}
                    </Space>
                }>
                {syncStatus.data && <SportsHistorySyncStatusBar status={syncStatus.data} refreshing={refreshing} />}
                {events.data && (
                    <div className='radar-snapshot'>
                        <span className={events.data.stale || events.error ? 'radar-warmup' : 'radar-connected'}>
                            {events.data.stale || events.error ? 'Stale snapshot' : 'Snapshot available'}
                        </span>
                        <span>Fetched {formatBeijingUnixSeconds(events.data.fetchedAt) || 'Unavailable'} · UTC+8</span>
                    </div>
                )}
                {history.error && Boolean(history.data?.items.length) && <p className='radar-warmup'>Price history is stale. Refresh to retry.</p>}
                <div className='sports-live-sections sports-history-sections'>
                    {Boolean(events.data) && !events.error && !events.loading && items.length === 0 && <Empty description='No data' />}
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
                                        <SportsHistoryEventCard
                                            item={item}
                                            history={sportsLiveCardHistory(item, historyByMarketKey)}
                                            historyState={history.error ? 'failed' : history.loading ? 'loading' : undefined}
                                            scratchMode={scratchMode}
                                            scratchResetVersion={scratchResetVersion}
                                            key={`${item.eventKey}:${scratchMode ? scratchResetVersion : 'direct'}`}
                                        />
                                    ))}
                                </div>
                            </section>
                        );
                    })}
                </div>
            </AppPage>
        </div>
    );
};
