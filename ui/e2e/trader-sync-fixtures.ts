import {expect, Page} from '@playwright/test';
import fs from 'node:fs';
import path from 'node:path';
const read = (name: string) => JSON.parse(fs.readFileSync(path.join(__dirname, '../src/app/member/testdata/trader-sync', name + '.json'), 'utf8'));
export const memberPath = (value: string) => (process.env.ATHENA_UI_E2E_PATH_PREFIX || '').replace(/\/$/, '') + value;
export const adminPath = (value: string) => memberPath('/admin' + value);
export const subscriptionID = read('01').subscription.id;
export const baseActivity = read('05').activity;
const clone = <T>(v: T): T => structuredClone(v);
const counts = (total: string, sent = '0') => ({total, pending: String(Number(total) - Number(sent)), sending: '0', sent, failed: '0', unknown: '0', cancelled: '0'});
export function scenarioActivity(scenario: string) {
    const a = clone(scenario.startsWith('ordinary') ? read('06').activity : scenario.startsWith('summary') ? read('07').activity : baseActivity);
    if (scenario === 'summary-cancelled') return clone(read('15').activities[0]);
    if (scenario.startsWith('ordinary-') && scenario !== 'ordinary-sent') {
        const status = scenario.slice('ordinary-'.length);
        a.delivery = {
            id: '800',
            status,
            reason: status === 'failed' ? 'telegram_rejected' : status === 'unknown' ? 'response_lost' : '',
            attemptCount: status === 'pending' ? '0' : '1'
        };
        if (status !== 'pending') {
            a.delivery.authorizedAt = a.recordedAt;
            if (status !== 'sending') a.delivery.resultAt = a.recordedAt;
        }
    }
    if (scenario === 'precision-huge') {
        a.collateralRaw = '90071992547409931234567890';
    }
    if (scenario === 'precision-combo') {
        a.collateralRaw = '1';
        a.priceNumerator = '1';
        a.priceDenominator = '10000000';
        a.metadata.relationship = 'NOT(AND(legs))';
        a.metadata.legsEvidence = {availability: 'available', reasonCode: '', source: 'synthetic_browser_fixture', queriedAt: a.recordedAt};
        a.metadata.market = {
            ...a.metadata.market,
            evidence: {availability: 'available', reasonCode: '', source: 'synthetic_browser_fixture', queriedAt: a.recordedAt},
            id: 'synthetic-combo',
            title: 'Synthetic Combo market',
            url: 'https://polymarket.com/event/synthetic-browser-condition'
        };
        a.metadata.legs = Array.from({length: 101}, (_, i) => ({
            positionId: String(9000 + i),
            market: {
                ...clone(a.metadata.market),
                evidence: {availability: 'available', reasonCode: '', source: 'synthetic_browser_fixture', queriedAt: a.recordedAt},
                id: String(i + 1),
                title: 'Synthetic condition ' + (i + 1),
                url: 'https://polymarket.com/event/synthetic-browser-condition',
                positionId: String(9000 + i),
                outcome: 'Yes'
            }
        }));
    }
    if (scenario.includes('frozen')) {
        a.summaryProgress.phase = 'frozen';
        a.summaryProgress.batchId = scenario.includes('all-sent') ? '902' : '901';
        a.summaryProgress.relatedPartCounts = counts('3', scenario.includes('all-sent') ? '3' : '0');
        a.summaryProgress.batchPartCounts = counts('101', scenario.includes('all-sent') ? '101' : '0');
        if (!scenario.includes('all-sent')) Object.assign(a.summaryProgress.batchPartCounts, {pending: '96', sending: '1', sent: '1', failed: '1', unknown: '1', cancelled: '1'});
    }
    return a;
}
export const fixtureStates = new WeakMap<
    Page,
    {denyAdmin: boolean; failUserInfo: boolean; holdUserInfo: boolean; userInfoReady: boolean; releaseUserInfo?: () => void; adminUnauthorized: boolean}
>();
export const routeLedgers = new WeakMap<Page, {unexpected: string[]; requests: string[]; bootstrap: number}>();
export async function installTraderSyncRoutes(page: Page, scenario: string): Promise<void> {
    const state: {denyAdmin: boolean; failUserInfo: boolean; holdUserInfo: boolean; userInfoReady: boolean; releaseUserInfo?: () => void; adminUnauthorized: boolean} = {
        denyAdmin: false,
        failUserInfo: false,
        holdUserInfo: false,
        userInfoReady: false,
        adminUnauthorized: false
    };
    fixtureStates.set(page, state);
    const ledger = {unexpected: [] as string[], requests: [] as string[], bootstrap: 0};
    routeLedgers.set(page, ledger);
    const activity = scenarioActivity(scenario);
    const subscription = clone(read('01').subscription);
    subscription.status = 'healthy';
    subscription.observation.state = 'healthy';
    const isAdmin = scenario.startsWith('admin');
    const user = {
        loggedIn: true,
        accountId: '11111111-1111-4111-8111-111111111111',
        username: 'browserfixture',
        iss: 'fixture',
        administrator: isAdmin,
        access: {
            loginEnabled: true,
            apiKeyEnabled: false,
            profitSharingEnabled: false,
            revision: '1',
            moduleAccess: ['MARKET_RADAR', 'MANAGED_OO', 'WORM_TRADING', 'TOKEN', 'SOLANA', 'WALLET', 'TRADER_SYNC'].map(x => ({
                module: 'ACCOUNT_DATA_MODULE_' + x,
                dataAccess: x === 'TRADER_SYNC' ? 'ACCOUNT_DATA_ACCESS_READ_WRITE' : 'ACCOUNT_DATA_ACCESS_NONE'
            }))
        }
    };
    const settings = {url: 'http://127.0.0.1', help: {binaryUrls: {}}, googleAnalytics: {}, additionalUrls: [], userLoginsDisabled: false};
    await page.route('**/*', async route => {
        const req = route.request(),
            url = new URL(req.url());
        if (!url.pathname.includes('/api/')) return route.continue();
        const expected = memberPath('/api/v1');
        ledger.requests.push(req.method() + ' ' + url.pathname + url.search);
        if (!url.pathname.startsWith(expected + '/')) {
            ledger.unexpected.push('prefix ' + url.pathname);
            return route.fulfill({status: 500, json: {error: 'wrong prefix'}});
        }
        const api = url.pathname.slice(expected.length);
        const realm = req.headers()['x-athena-application-realm'];
        if (realm !== (isAdmin ? 'admin' : 'member')) {
            ledger.unexpected.push('realm ' + realm);
            return route.fulfill({status: 500, json: {error: 'wrong realm'}});
        }
        if (api === '/module-access-states')
            return route.fulfill({json: {states: ['trader_sync', 'solana', 'market_radar', 'managed_oo', 'profit_sharing', 'worm'].map(module_key => ({module_key, state: 1}))}});
        if (isAdmin && state.adminUnauthorized && api.startsWith('/admin/')) return route.fulfill({status: 401, json: {code: 16, message: 'Unauthenticated'}});
        if (api === '/session/userinfo' && state.holdUserInfo) {
            state.userInfoReady = true;
            await new Promise<void>(resolve => {
                state.releaseUserInfo = resolve;
            });
        }
        if (isAdmin && state.denyAdmin && api.startsWith('/admin/'))
            return route.fulfill({status: 403, json: {code: 7, message: 'administrator access changed', reason: 'ACCOUNT_ADMIN_REQUIRED'}});
        if (api === '/session/userinfo' && state.failUserInfo) return route.fulfill({status: 503, json: {code: 14, message: 'controlled userinfo unavailable'}});
        let body: any;
        if (api === '/app/bootstrap') {
            ledger.bootstrap++;
            body = {
                settings,
                session: state.adminUnauthorized ? {status: 'APP_BOOTSTRAP_SESSION_STATUS_ANONYMOUS'} : {status: 'APP_BOOTSTRAP_SESSION_STATUS_AUTHENTICATED', userInfo: user}
            };
        } else if (api.match(/^\/account\/[^/]+\/profile$/) && req.method() === 'PUT') body = {displayName: req.postDataJSON().displayName, revision: 1};
        else if (api === '/session/userinfo') body = user;
        else if (api === '/admin/trader-sync/status') {
            body = JSON.parse(fs.readFileSync(path.join(__dirname, '../src/app/admin/__fixtures__/trader-sync/tradersync-fix1-ordinary.json'), 'utf8'));
            body.status.metrics.push({
                name: 'synthetic_window',
                value: '3',
                unit: 'activities',
                kind: 'window',
                windowStart: '2026-09-11T00:59:00Z',
                windowEnd: '2026-09-11T01:00:00Z'
            });
        } else if (api === '/admin/notification-runtime/status')
            body = JSON.parse(fs.readFileSync(path.join(__dirname, '../src/app/admin/__fixtures__/trader-sync/notification-04-waiting.json'), 'utf8'));
        else if (api === '/service-statuses') body = {items: [{name: 'synthetic service', status: 'SERVING'}], checkedAt: 1789088400};
        else if (api.startsWith('/admin/trader-sync/subscriptions')) {
            const summary = {
                subscriptionId: subscriptionID,
                accountId: '22222222-2222-4222-8222-222222222222',
                username: 'alice',
                email: 'alice@example.test',
                wallet: subscription.wallet,
                status: 'healthy',
                createdAt: subscription.createdAt,
                updatedAt: subscription.updatedAt,
                observation: subscription.observation,
                activityCount: '9007199254740993',
                associatedDeliveryCounts: counts('5', '2'),
                asOf: subscription.updatedAt
            };
            body = api.endsWith('/subscriptions') ? {summaries: [summary], page: {}, asOf: summary.asOf} : {summary};
        } else if (api === '/notification-bindings/telegram') body = {botAvailable: true, botUsername: 'acceptance_bot'};
        else if (api === '/trader-sync/targets:resolve') {
            body = clone(read('resolve-saved-note-existing'));
            body.target.expiresAt = new Date(Date.now() + 300000).toISOString();
            delete body.target.existingSubscription;
        } else if (api === '/trader-sync/subscriptions') body = {subscriptions: [subscription], page: {}, quota: {used: 1, limit: 10}, asOf: subscription.updatedAt};
        else if (api === `/trader-sync/subscriptions/${subscriptionID}`) body = {subscription};
        else if (api === `/trader-sync/subscriptions/${subscriptionID}/history`) {
            const offset = url.searchParams.get('page.cursor') === 'history-50' ? 50 : 0;
            const entries = Array.from({length: 51}, (_, i) => {
                const e = clone(read('12').entries[0]);
                e.id = 'interval/' + `00000000-0000-4000-8000-${String(i + 1).padStart(12, '0')}`;
                if (i === 0) {
                    e.kind = 'interruption';
                    e.id = 'interruption/1';
                    delete e.interval;
                    e.interruption = {reason: 'disconnect', uncertainty: 'unknown_start', possibleMissing: true};
                }
                return e;
            });
            body = {entries: entries.slice(offset, offset + 50), page: offset ? {} : {nextCursor: 'history-50'}, asOf: subscription.updatedAt};
        } else if (api === '/trader-sync/activities') {
            const token = url.searchParams.get('refresh_cursor') || url.searchParams.get('page.cursor') || '';
            const tail = token.includes('50');
            const total = scenario === 'history-scroll' || scenario === 'summary-frozen-history' ? 51 : 1;
            const list = Array.from({length: total}, (_, i) => ({...clone(activity), id: String(total + 100 - i)}));
            body = {
                activities: tail ? list.slice(50) : list.slice(0, 50),
                page: {
                    snapshot: 'fixture-snapshot',
                    refreshCursor: tail ? 'fixture-refresh-50' : 'fixture-refresh-0',
                    ...(total > 50 && !tail ? {nextCursor: 'fixture-next-50'} : {}),
                    asOf: activity.recordedAt,
                    hasNewer: false
                }
            };
        } else if (/^\/trader-sync\/activities\/\d+$/.test(api)) body = {activity};
        else if (/^\/trader-sync\/summaries\/\d+$/.test(api)) {
            const batch = clone(read('09').batch);
            batch.id = activity.summaryProgress.batchId;
            batch.partCounts = activity.summaryProgress.batchPartCounts;
            body = {batch};
        } else if (/^\/trader-sync\/summaries\/\d+\/parts$/.test(api)) {
            const all = Array.from({length: 101}, (_, i) => {
                const part = clone(read('10').parts[0]);
                part.id = String(1000 + i);
                part.index = i + 1;
                part.total = 101;
                part.delivery.id = String(2000 + i);
                part.delivery.status = scenario.includes('all-sent') ? 'sent' : ['sending', 'sent', 'failed', 'unknown', 'cancelled'][i] || 'pending';
                part.delivery.reason = ['failed', 'unknown', 'cancelled'].includes(part.delivery.status) ? 'synthetic_browser_reason' : '';
                return part;
            });
            const related = url.searchParams.has('activity_id') ? all.filter((_, i) => [0, 50, 100].includes(i)) : all;
            const size = Number(url.searchParams.get('page.page_size') || 50),
                offset = Number((url.searchParams.get('page.cursor') || 'parts:0').split(':')[1]);
            body = {parts: related.slice(offset, offset + size), page: offset + size < related.length ? {nextCursor: 'parts:' + (offset + size)} : {}, asOf: activity.recordedAt};
        } else {
            ledger.unexpected.push(req.method() + ' ' + api);
            return route.fulfill({status: 500, json: {error: 'unexpected fixture API'}});
        }
        await route.fulfill({json: body});
    });
}
export async function assertFixture(page: Page) {
    const ledger = routeLedgers.get(page)!;
    expect(ledger.bootstrap).toBeGreaterThan(0);
    expect(ledger.unexpected).toEqual([]);
}
