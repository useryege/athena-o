import {applyActivityRefresh, activityDateRange} from './activity-session';
import {normalizeActivity, ActivityPage} from '../../trader-sync-models';
import fixture from '../../testdata/trader-sync/01.json';
import activityFixture from '../../testdata/trader-sync/05.json';
const activity = normalizeActivity(activityFixture.activity);
const page = (ids: string[]): ActivityPage => ({
    activities: ids.map(id => ({...activity, id})),
    page: {snapshot: 'opaque-snapshot', refreshCursor: 'signed-current-page', hasNewer: false}
});
test('fixed members keep their order while notification results and hasNewer refresh', () => {
    const current = page(['12', '11']);
    const incoming = page(['12', '11']);
    incoming.page.hasNewer = true;
    incoming.activities[0] = {...incoming.activities[0], notificationReason: 'updated'};
    expect(applyActivityRefresh(current, incoming)).toEqual(incoming);
    expect(current.activities[0].notificationReason).not.toBe('updated');
});
test.each([['13', '12'], ['11', '12'], ['12']])('rejects changed ordered membership %s and preserves original', (...ids) => {
    const current = page(['12', '11']);
    expect(() => applyActivityRefresh(current, page(ids))).toThrow('protocol');
    expect(current.activities.map(a => a.id)).toEqual(['12', '11']);
});
test.each([{snapshot: 'other'}, {snapshot: undefined}, {refreshCursor: undefined}])('rejects invalid refresh tokens %j', tokens => {
    const current = page(['12']);
    const incoming = page(['12']);
    Object.assign(incoming.page, tokens);
    expect(() => applyActivityRefresh(current, incoming)).toThrow('protocol');
});
test('empty page stays empty when newer records exist', () => {
    const current = page([]);
    const incoming = page([]);
    incoming.page.hasNewer = true;
    expect(applyActivityRefresh(current, incoming).activities).toEqual([]);
    expect(() => applyActivityRefresh(current, page(['13']))).toThrow('protocol');
});
test('date filters use UTC+8 inclusive start and exclusive next day', () =>
    expect(activityDateRange('2026-09-10', '2026-09-11')).toEqual({from: '2026-09-09T16:00:00.000Z', to: '2026-09-11T16:00:00.000Z'}));

test('owner activity sessions clear with scope and reject late saves', async () => {
    const {captureTraderSyncScope, clearTraderSyncState, saveActivitySession, readActivitySession} = await import('./state');
    const scope = captureTraderSyncScope('A');
    const session = {query: {pageSize: 50}, current: page(['12']), previous: [], scrollY: 120};
    saveActivitySession('A', session, scope);
    expect(readActivitySession('A')).toBe(session);
    expect(readActivitySession('B')).toBeUndefined();
    clearTraderSyncState();
    saveActivitySession('A', session, scope);
    expect(readActivitySession('A')).toBeUndefined();
});
