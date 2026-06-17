import {LinkOutlined} from '@ant-design/icons';
import type {ColumnsType} from 'antd/es/table';
import {Button, Empty, Space, Tag, Typography} from 'antd';
import * as React from 'react';
import {AppPage, CardTitle, MetricRow, ResponsiveResourceList, SearchBar, StatusTag, useAsyncData} from '../components';
import {services} from '../../shared/services';
import {
    PolymarketDisputedMarketItem,
    PolymarketHotMarketItem,
    PolymarketMoverMarketItem,
    PolymarketRealtimeMarketItem,
    PolymarketSportsHistoryEventCardItem,
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
import {boolTag, fmt, fmtNumber, useKeywordParam} from './shared';

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

const disputedStatusTrail = (value?: string) => {
    const raw = String(value || '').trim();
    if (!raw) {
        return [];
    }
    try {
        const parsed = JSON.parse(raw);
        if (Array.isArray(parsed)) {
            return parsed.map(item => String(item || '').trim()).filter(Boolean);
        }
    } catch {
        // fall through to plain text fallback
    }
    return raw
        .replace(/^\[/, '')
        .replace(/\]$/, '')
        .split(',')
        .map(item => item.replace(/^"+|"+$/g, '').trim())
        .filter(Boolean);
};

const disputedMarketURL = (item: PolymarketDisputedMarketItem) => {
    const marketSlug = item.marketSlug.trim();
    const eventSlug = item.eventSlug.trim();
    if (eventSlug && marketSlug && eventSlug !== marketSlug) {
        return `https://polymarket.com/event/${encodeURIComponent(eventSlug)}/${encodeURIComponent(marketSlug)}`;
    }
    const slug = marketSlug || eventSlug;
    return slug ? `https://polymarket.com/event/${encodeURIComponent(slug)}` : '';
};

const DisputedStatus = (props: {item: PolymarketDisputedMarketItem}) => {
    const trail = disputedStatusTrail(props.item.umaResolutionStatuses);
    return (
        <Space orientation='vertical' size={2}>
            <Tag color='red'>{props.item.umaResolutionStatus || 'disputed'}</Tag>
            {trail.length > 0 && <Typography.Text type='secondary'>{trail.join(' > ')}</Typography.Text>}
        </Space>
    );
};

const DisputedOpenButton = (props: {item: PolymarketDisputedMarketItem}) => {
    const url = disputedMarketURL(props.item);
    return <Button aria-label='Open Polymarket' disabled={!url} href={url || undefined} icon={<LinkOutlined />} rel='noreferrer' size='small' target='_blank' type='text' />;
};

export const PolymarketDisputedPage = () => {
    const data = useAsyncData(() => services.polymarket.listDisputedMarkets(100), []);
    const [keyword, setKeyword] = useKeywordParam('q');
    const items = data.data?.items || [];
    const filteredItems = React.useMemo(() => {
        const query = keyword.trim().toLowerCase();
        if (!query) {
            return items;
        }
        return items.filter(item =>
            [item.question, item.marketSlug, item.eventSlug, item.conditionId, item.marketKey].map(value => String(value || '').toLowerCase()).some(value => value.includes(query))
        );
    }, [items, keyword]);
    const columns: ColumnsType<PolymarketDisputedMarketItem> = [
        {title: 'Market', render: item => <CardTitle title={item.question} subtitle={item.eventSlug || item.marketSlug} image={item.image} />},
        {title: 'UMA', render: item => <DisputedStatus item={item} />},
        {title: 'Volume 24h', render: item => fmtNumber(item.volume24hr || item.volumeNum)},
        {title: 'Liquidity', render: item => fmtNumber(item.liquidityNum)},
        {title: 'Spread', render: item => fmt(item.spread)},
        {
            title: 'State',
            render: item => (
                <Space size={4} wrap={true}>
                    {boolTag(item.active)}
                    <StatusTag value={item.closed ? 'Closed' : 'Open'} positive={!item.closed} negative={item.closed} />
                </Space>
            )
        },
        {title: 'Last Seen', dataIndex: 'lastSeenAt'},
        {title: 'Open', render: item => <DisputedOpenButton item={item} />}
    ];
    return (
        <AppPage
            title='Polymarket Disputed'
            subtitle={`Fetched ${fmt(data.data?.fetchedAt)} · ${items.length} markets ${data.data?.stale ? '(stale)' : ''}`}
            loading={data.loading}
            error={data.error}
            onRefresh={data.reload}
            filters={<SearchBar value={keyword} placeholder='Search disputed markets' onChange={setKeyword} />}>
            <ResponsiveResourceList
                rowKey='marketKey'
                items={filteredItems}
                columns={columns}
                loading={data.loading}
                card={item => (
                    <>
                        <CardTitle title={item.question} subtitle={item.eventSlug || item.marketSlug} image={item.image} tags={<DisputedOpenButton item={item} />} />
                        <Space size={6} wrap={true}>
                            <DisputedStatus item={item} />
                            {boolTag(item.active)}
                            <StatusTag value={item.closed ? 'Closed' : 'Open'} positive={!item.closed} negative={item.closed} />
                            <StatusTag value='Order Book' positive={item.enableOrderBook} negative={!item.enableOrderBook} />
                        </Space>
                        <MetricRow
                            items={[
                                {label: 'Volume 24h', value: fmtNumber(item.volume24hr || item.volumeNum)},
                                {label: 'Liquidity', value: fmtNumber(item.liquidityNum)},
                                {label: 'Spread', value: fmt(item.spread)},
                                {label: 'Last Trade', value: fmt(item.lastTradePrice)}
                            ]}
                        />
                    </>
                )}
            />
        </AppPage>
    );
};

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

export const PolymarketSportsHistoryPage = () => {
    const events = useAsyncData(() => services.polymarket.listSportsHistoryEvents(200), []);
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

    return (
        <AppPage
            title='Sports History'
            subtitle={`Last 72 hours · Fetched ${fmt(events.data?.fetchedAt)} ${events.data?.stale ? '(stale)' : ''}`}
            loading={events.loading}
            error={events.error}
            onRefresh={reloadAll}>
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
