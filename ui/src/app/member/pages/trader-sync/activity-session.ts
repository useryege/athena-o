import type {ActivityPage} from '../../trader-sync-models';
import type {ActivityQuery} from '../../trader-sync-service';
export {readActivitySession, saveActivitySession} from './state';
export interface ActivitySession {
    query: ActivityQuery;
    current?: ActivityPage;
    previous: Array<{query: ActivityQuery; page: ActivityPage; scrollY: number}>;
    scrollY: number;
}
export const validateActivityPage = (value: ActivityPage): ActivityPage => {
    if (!value.page.snapshot || !value.page.refreshCursor) throw new Error('Trader Sync protocol error: missing activity page tokens. Load latest activities to recover.');
    return value;
};
export const applyActivityRefresh = (current: ActivityPage, incoming: ActivityPage): ActivityPage => {
    validateActivityPage(current);
    validateActivityPage(incoming);
    if (
        current.page.snapshot !== incoming.page.snapshot ||
        current.activities.length !== incoming.activities.length ||
        current.activities.some((item, index) => item.id !== incoming.activities[index].id)
    ) {
        throw new Error('Trader Sync protocol error: activity snapshot or ordered membership changed. The previous page is retained.');
    }
    return {
        ...incoming,
        activities: incoming.activities.map((item, index) => (JSON.stringify(item) === JSON.stringify(current.activities[index]) ? current.activities[index] : item))
    };
};
export const activityDateRange = (from: string, to: string): Pick<ActivityQuery, 'from' | 'to'> => {
    const start = from ? new Date(`${from}T00:00:00+08:00`) : undefined;
    const end = to ? new Date(new Date(`${to}T00:00:00+08:00`).getTime() + 86400000) : undefined;
    if ((start && !Number.isFinite(start.getTime())) || (end && !Number.isFinite(end.getTime())) || (start && end && start >= end))
        throw new Error('Choose a valid settlement date range.');
    return {from: start?.toISOString(), to: end?.toISOString()};
};
