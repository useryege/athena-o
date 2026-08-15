import {Empty, Typography} from 'antd';
import * as React from 'react';
import {AppPage, useAsyncData} from '../components';
import {formatBeijingUnixSeconds} from '../shared/format';
import {services} from '../shared/services';
import {SportsLiveEventCardItem, SportsPriceHistorySeriesItem} from '../shared/services/sports-models';
import {FifwcSportsLiveEventCard, LegacySportsLiveEventCard, isFifwcSportsLiveEvent, moneylineMarketKeys, sportsLiveCardHistory, sportsLiveSectionKey} from './sports-market-card';

const sportsLiveRefreshIntervalMs = 3000;

type SportsLiveSection = {
    key: string;
    title: string;
    items: SportsLiveEventCardItem[];
};

const sportsLiveSectionTitle = (key: string) => key.toUpperCase();

const sportsLiveSections = (items: SportsLiveEventCardItem[] = []): SportsLiveSection[] => {
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

export const SportsLivePage = () => {
    const events = useAsyncData(() => services.sportsLive.listEvents(200), []);
    const marketKeys = React.useMemo(() => moneylineMarketKeys(events.data?.items), [events.data?.items]);
    const marketKeySignature = React.useMemo(() => marketKeys.join('|'), [marketKeys]);
    const history = useAsyncData(() => {
        if (marketKeys.length === 0) {
            return Promise.resolve({items: []}) as Promise<{items: SportsPriceHistorySeriesItem[]}> & {abort?: () => void};
        }
        return services.sportsLive.batchGetPriceHistories(marketKeys, 360);
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
        const out = new Map<string, SportsPriceHistorySeriesItem[]>();
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
            subtitle={`Fetched ${formatBeijingUnixSeconds(events.data?.fetchedAt) || '-'} ${events.data?.stale ? '(stale)' : ''}`}
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
