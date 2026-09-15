import {expect, test} from '@playwright/test';
import {assertThemeLayout, assertThemeLedger, openThemeCase} from './routes';

for (const id of ['hot', 'realtime', 'movers', 'live', 'history', 'corners', 'proposals', 'disputes']) {
    test(`theme:markets ${id} first failure excludes successful empty state`, async ({page}) => {
        const ledger = await openThemeCase(page, `markets-${id}-failed`);
        await expect(page.getByText('Request failed', {exact: true})).toBeVisible();
        await expect(page.locator('.ant-empty:visible')).toHaveCount(0);
        await expect(page.locator('.resource-table-pagination:visible')).toHaveCount(0);
        await expect(page.locator('.world-cup-corners-kpis:visible')).toHaveCount(0);
        assertThemeLedger(ledger);
    });
}

import {themeCases} from './cases';
import {installThemeCase} from './routes';
import type {ThemeCase, ThemeLedger} from './contracts';
const mainIds = ['hot', 'realtime', 'movers', 'live', 'history', 'corners', 'proposals', 'disputes'];
const scenario = (id: string) => structuredClone(themeCases.find(item => item.id === `markets-${id}`)!) as ThemeCase;
const visible = (page: import('@playwright/test').Page, text: string) => page.getByText(text, {exact: true}).filter({visible: true}).first();
const go = async (page: import('@playwright/test').Page, data: ThemeCase) => {
    const ledger = await installThemeCase(page, data);
    await page.goto(`${process.env.ATHENA_UI_E2E_PATH_PREFIX || ''}${data.route}`);
    await expect(page.getByRole('heading', {level: 1, name: data.heading, exact: true})).toBeVisible();
    return ledger;
};
const checkReads = (ledger: ThemeLedger) => {
    for (const item of ledger.requests) {
        expect(item.realm).toBe('member');
        if (item.path === '/api/v1/app/bootstrap') {
            expect(item.method).toBe('GET');
            expect(item.query).toBe('');
        } else if (item.path.includes('market-radar/') || item.path.endsWith('/events')) {
            expect(item.method).toBe('GET');
            expect(item.query).toBe(item.path.includes('market-radar/') ? '?limit=100' : '?limit=200');
        } else if (/managed-oo\/(proposals|disputes)$/.test(item.path)) {
            expect(item.method).toBe('GET');
            expect(item.query).toBe('?page=1&page_size=50');
        } else if (item.path.endsWith('price-history:batchGet')) {
            expect(item.method).toBe('POST');
            expect(item.query).toBe('');
            expect(item.body).toMatchObject({limit_per_token: 360});
            expect((item.body as {market_keys: string[]}).market_keys).toEqual(
                item.path.includes('sports-live') ? ['live-fifwc-eng', 'live-fifwc-fra', 'live-fifwc-draw', 'live-atp-ml'] : ['hist-atp-ml', 'hist-wta-ml']
            );
        } else {
            expect(item.method).toBe('GET');
            expect(item.query).toBe('');
        }
    }
    expect(ledger.requests.filter(item => item.path === '/api/v1/app/bootstrap')).toHaveLength(1);
    assertThemeLedger(ledger);
};
for (const viewport of [
    {name: 'desktop', width: 1440, height: 900},
    {name: 'mobile', width: 390, height: 844}
]) {
    for (const id of mainIds)
        test(`theme:markets ${id} ${viewport.name} approved structure`, async ({page}, info) => {
            await page.setViewportSize(viewport);
            const ledger = await openThemeCase(page, `markets-${id}`);
            if (['hot', 'realtime', 'movers'].includes(id)) {
                await expect(visible(page, 'Will Bitcoin reach $120,000 by September 30?')).toBeVisible();
                if (id === 'hot') {
                    await expect(visible(page, '2,481,920.52')).toBeVisible();
                    await expect(visible(page, '481,240.23')).toBeVisible();
                }
                if (id === 'realtime') {
                    await expect(visible(page, '+1.2 pp')).toBeVisible();
                    await expect(visible(page, 'Warmup')).toBeVisible();
                }
                if (id === 'movers') {
                    await expect(visible(page, 'Mover score')).toBeVisible();
                    await expect(visible(page, '3.78')).toBeVisible();
                }
            } else if (id === 'live' || id === 'history') {
                await expect(page.getByRole('img', {name: 'Moneyline price history'}).first()).toBeVisible();
                await expect(visible(page, id === 'live' ? '1–0' : '6–4, 7–6')).toBeVisible();
                await expect(visible(page, id === 'live' ? '0.615' : '1.000')).toBeVisible();
                await expect(page.getByRole('slider', {name: 'History time'}).first()).toBeVisible();
                await expect(page.getByRole('button', {name: id === 'history' ? 'Reload saved data' : 'Refresh data', exact: true})).toHaveCount(1); // read-only reload, no manual synchronization
            } else if (id === 'corners') {
                await expect(visible(page, '62.5%')).toBeVisible();
                await expect(visible(page, '7.75')).toBeVisible();
                await expect(visible(page, '8')).toBeVisible();
                await expect(page.getByText('64 matches')).toHaveCount(0);
            } else {
                await expect(visible(page, '1000000000000000000')).toBeVisible();
                await expect(visible(page, id === 'proposals' ? '0x00000000000000000000000000000000000003b3' : '0x000000000000000000000000000000000000041b')).toBeVisible();
                await expect(page.getByRole('button', {name: 'Parse block', exact: true})).toBeDisabled();
            }
            await page.evaluate(() => document.fonts.ready);
            await page.evaluate(() => window.scrollTo(0, 0));
            await assertThemeLayout(page);
            await page.screenshot({path: info.outputPath(`${id}-${viewport.name}.png`), fullPage: true, animations: 'disabled'});
            checkReads(ledger);
        });
}

for (const id of mainIds)
    test(`theme:markets ${id} successful empty is separate`, async ({page}) => {
        const ledger = await openThemeCase(page, `markets-${id}-empty`);
        if (id === 'corners') await expect(visible(page, '0')).toBeVisible();
        else await expect(page.locator('.ant-empty:visible').first()).toBeVisible();
        await expect(page.getByText('Request failed', {exact: true})).toHaveCount(0);
        checkReads(ledger);
    });
for (const id of ['hot', 'realtime', 'movers']) {
    for (const state of ['known-zero', 'missing', 'stale', 'window-warmup', 'record-long'])
        test(`theme:markets ${id} ${state} facts`, async ({page}, info) => {
            await page.setViewportSize({width: 390, height: 844});
            const ledger = await openThemeCase(page, `markets-${id}-${state}`);
            if (state === 'known-zero') {
                const row = page.locator('.radar-record:visible').first();
                await expect(row.getByText('0', {exact: true}).first()).toBeVisible();
                await row.locator('summary').click();
                await expect(row.getByText('123,456.789', {exact: true})).toBeVisible();
            } else if (state === 'missing') await expect(visible(page, 'Unavailable')).toBeVisible();
            else if (state === 'stale') await expect(visible(page, 'Stale snapshot')).toBeVisible();
            else if (state === 'window-warmup' && id === 'realtime') {
                await expect(visible(page, '0 pp')).toBeVisible();
                await expect(visible(page, 'Warmup')).toBeVisible();
                await expect(page.locator('.radar-warmup:visible').first()).toHaveCSS('color', 'rgb(230, 191, 114)');
            }
            await page.evaluate(() => window.scrollTo(0, 0));
            await assertThemeLayout(page);
            await page.screenshot({path: info.outputPath(`${id}-${state}.png`), fullPage: true});
            checkReads(ledger);
        });
}

for (const id of ['hot', 'live', 'history', 'corners', 'proposals'])
    test(`theme:markets ${id} refresh failure retains stale facts`, async ({page}) => {
        const data = scenario(id);
        const ledger = await go(page, data);
        const firstData = data.replies[1];
        await expect.poll(() => ledger.requests.filter(r => r.path === firstData.path).length).toBeGreaterThan(0);
        await expect(page.locator('.radar-record:visible,.sports-live-card:visible,.world-cup-corners-kpis:visible,.managed-oo-record:visible').first()).toBeVisible();
        firstData.status = 503;
        firstData.json = {message: 'Refresh source unavailable'};
        await page.getByRole('button', {name: id === 'history' ? 'Reload saved data' : 'Refresh data', exact: true}).click();
        await expect(visible(page, 'Request failed')).toBeVisible();
        await expect(
            page
                .getByText(/Stale (snapshot|saved events|dataset)/)
                .filter({visible: true})
                .first()
        ).toBeVisible();
        await expect(page.locator('.radar-record:visible,.sports-live-card:visible,.world-cup-corners-kpis:visible,.managed-oo-record:visible').first()).toBeVisible();
        assertThemeLedger(ledger);
    });

test('theme:markets Sources and evidence preserve complete original fields', async ({page}, info) => {
    await page.setViewportSize({width: 390, height: 844});
    let ledger = await openThemeCase(page, 'markets-live');
    await page.locator('.sports-event-sources summary').first().click();
    await expect(visible(page, 'live-fifwc')).toBeVisible();
    await expect(visible(page, '0x1111111111111111111111111111111111111111111111111111111111111111')).toBeVisible();
    checkReads(ledger);
    ledger = await openThemeCase(page, 'markets-proposals');
    await page.getByRole('button', {name: 'View evidence', exact: true}).filter({visible: true}).first().click();
    const drawer = page.getByRole('dialog');
    await expect(drawer.getByText('1000000000000000000', {exact: true})).toBeVisible();
    await expect(drawer.getByText('1789372800', {exact: true})).toBeVisible();
    await expect.poll(async () => Math.round((await drawer.boundingBox())?.x ?? -1)).toBe(0);
    await drawer.locator('.ant-drawer-body').evaluate(el => (el.scrollTop = 0));
    await page.screenshot({path: info.outputPath('evidence-summary.png')});
    await drawer.locator('summary').click();
    await expect(drawer.getByText('Raw topics', {exact: true})).toBeVisible();
    await drawer.getByRole('button', {name: 'Close evidence', exact: true}).focus();
    await page.screenshot({path: info.outputPath('evidence.png')});
    await page.keyboard.press('Escape');
    await expect(drawer).toBeHidden();
    await expect(page.getByRole('button', {name: 'View evidence', exact: true}).filter({visible: true}).first()).toBeFocused();
    checkReads(ledger);
});

test('theme:markets parse pending and failure is a declared fixture write only', async ({page}, info) => {
    const data = scenario('proposals');
    data.replies.push({method: 'POST', path: '/api/v1/managed-oo/blocks/70841621:scan', realm: 'member', status: 503, json: {message: 'Controlled parse failure'}, delayMs: 1200});
    const ledger = await go(page, data);
    await page.getByRole('spinbutton', {name: 'Polygon block number'}).fill('70841621');
    await page.getByRole('button', {name: 'Parse block', exact: true}).click();
    await expect(page.getByRole('spinbutton', {name: 'Polygon block number'})).toBeDisabled();
    await page.screenshot({path: info.outputPath('parse-pending.png'), fullPage: true});
    await expect(visible(page, 'Block parse failed')).toBeVisible();
    await expect(page.getByRole('spinbutton', {name: 'Polygon block number'})).toHaveValue('70841621');
    expect(ledger.requests.filter(item => item.method === 'POST')).toEqual([
        {method: 'POST', path: '/api/v1/managed-oo/blocks/70841621:scan', realm: 'member', query: '', body: {}}
    ]);
    await page.screenshot({path: info.outputPath('parse-failed.png'), fullPage: true});
    assertThemeLedger(ledger);
});

for (const id of ['proposals', 'disputes'])
    test(`theme:markets ${id} read-only hides parse`, async ({page}) => {
        const ledger = await openThemeCase(page, `markets-${id}-readonly`);
        await expect(page.getByRole('button', {name: 'Parse block', exact: true})).toHaveCount(0);
        await expect(visible(page, '1000000000000000000')).toBeVisible();
        checkReads(ledger);
    });

for (const id of mainIds)
    test(`theme:markets ${id} narrow and root32`, async ({page}, info) => {
        await page.setViewportSize({width: 320, height: 844});
        const ledger = await openThemeCase(page, `markets-${id}`);
        await expect(page.locator('.radar-record:visible,.sports-live-card:visible,.world-cup-corners-kpis:visible,.managed-oo-record:visible').first()).toBeVisible();
        await assertThemeLayout(page);
        await page.screenshot({path: info.outputPath(`${id}-320.png`), fullPage: true});
        await page.setViewportSize({width: 720, height: 900});
        const title = page.getByRole('heading', {level: 1});
        const subtitle = page.locator('.app-page__heading > .ant-typography-secondary');
        const numberInput = page.getByRole('spinbutton', {name: 'Polygon block number'});
        const beforeInput = (await numberInput.count()) ? await numberInput.evaluate(el => parseFloat(getComputedStyle(el).fontSize)) : undefined;
        const beforeBody = await subtitle.evaluate(el => parseFloat(getComputedStyle(el).fontSize));
        const before = await title.evaluate(el => parseFloat(getComputedStyle(el).fontSize));
        await page.evaluate(() => (document.documentElement.style.fontSize = '32px'));
        expect(await title.evaluate(el => parseFloat(getComputedStyle(el).fontSize))).toBeCloseTo(before * 2, 1);
        expect(await subtitle.evaluate(el => parseFloat(getComputedStyle(el).fontSize))).toBeCloseTo(beforeBody * 2, 1);
        if (beforeInput !== undefined) await expect.poll(() => numberInput.evaluate(el => parseFloat(getComputedStyle(el).fontSize))).toBeCloseTo(beforeInput * 2, 1);
        await assertThemeLayout(page);
        const bad = await page
            .locator('.radar-record:visible,.sports-live-card:visible,.managed-oo-record:visible,.world-cup-corners-stage-row:visible')
            .evaluateAll(els => els.filter(el => el.scrollWidth > el.clientWidth + 1).map(el => el.className));
        expect(bad).toEqual([]);
        await page.screenshot({path: info.outputPath(`${id}-root32.png`), fullPage: true});
        assertThemeLedger(ledger);
    });

test('theme:markets issuer replacement drops parse draft and ignores late scan completion', async ({page}) => {
    const data = scenario('proposals');
    const session = data.replies[0].json as {session: {userInfo: {iss: string}}};
    const user = structuredClone(session.session.userInfo);
    data.replies.push({method: 'GET', path: '/api/v1/session/userinfo', realm: 'member', status: 200, json: user});
    data.replies.push({
        method: 'POST',
        path: '/api/v1/managed-oo/blocks/70841621:scan',
        realm: 'member',
        status: 200,
        json: {blockNumber: 70841621, proposalCount: 3, disputeCount: 0},
        delayMs: 1800
    });
    const ledger = await go(page, data);
    await page.getByRole('spinbutton', {name: 'Polygon block number'}).fill('70841621');
    await page.getByRole('button', {name: /Parse block$/}).click();
    user.iss = 'new-issuer';
    await page.evaluate(() => window.dispatchEvent(new Event('focus')));
    await expect.poll(() => ledger.requests.filter(item => item.path === '/api/v1/session/userinfo').length).toBe(1);
    await expect(page.getByRole('spinbutton', {name: 'Polygon block number'})).toHaveValue('');
    await page.waitForTimeout(2000);
    expect(new URL(page.url()).search).toBe('');
    await expect(page.getByText('Block 70841621 parsed', {exact: true})).toHaveCount(0);
    assertThemeLedger(ledger);
});

for (const id of ['live', 'history'])
    test(`theme:markets ${id} history read failure is not successful no-history`, async ({page}) => {
        const data = scenario(id);
        const history = data.replies.find(reply => reply.path.endsWith('price-history:batchGet'))!;
        history.status = 503;
        history.json = {message: 'Price history unavailable'};
        const ledger = await go(page, data);
        await expect(visible(page, 'Request failed')).toBeVisible();
        await expect(visible(page, id === 'live' ? '1–0' : '6–4, 7–6')).toBeVisible();
        await expect(page.getByText('No history', {exact: true})).toHaveCount(0);
        await expect(visible(page, 'History unavailable')).toBeVisible();
        await expect(page.getByText('Price history is stale. Refresh to retry.', {exact: true})).toHaveCount(0);
        assertThemeLedger(ledger);
    });

test('theme:markets radar pagination refresh preserves page and clamps after shrink', async ({page}) => {
    const data = scenario('hot');
    const reply = data.replies[1];
    const body = reply.json as {items: Array<{conditionId: string; question: string}>};
    const first = structuredClone(body.items[0]);
    body.items = Array.from({length: 22}, (_, i) => ({...first, conditionId: `page-condition-${i}`, question: `Page market ${i + 1}`}));
    const ledger = await go(page, data);
    await page.locator('.resource-table-pagination .ant-radio-button-wrapper').filter({hasText: /^10$/}).click();
    await expect(page.locator('.radar-record:visible')).toHaveCount(10);
    await page.locator('.ant-pagination-item-2').click();
    await expect(visible(page, 'Page market 11')).toBeVisible();
    await page.getByRole('button', {name: 'Refresh data', exact: true}).click();
    await expect.poll(() => ledger.requests.filter(item => item.path === reply.path).length).toBe(2);
    await expect(visible(page, 'Page market 11')).toBeVisible();
    body.items = body.items.slice(0, 5);
    await page.getByRole('button', {name: 'Refresh data', exact: true}).click();
    await expect(page.locator('.radar-record:visible')).toHaveCount(5);
    await expect(visible(page, 'Page market 1')).toBeVisible();
    expect(ledger.requests.filter(item => item.path === reply.path).map(item => item.query)).toEqual(['?limit=100', '?limit=100', '?limit=100']);
    assertThemeLedger(ledger);
});

test('theme:markets sports scratch preserves scores and snapshots while keyboard reveals timed history', async ({page}, info) => {
    await page.setViewportSize({width: 390, height: 844});
    const ledger = await openThemeCase(page, 'markets-history');
    await page.getByRole('button', {name: 'Scratch', exact: true}).click();
    await expect(visible(page, '6–4, 7–6')).toBeVisible();
    await expect(visible(page, '1.000')).toBeVisible();
    const slider = page.getByRole('slider', {name: 'History time'}).first();
    await slider.focus();
    await slider.press('Home');
    await slider.press('ArrowRight');
    await expect(page.locator('.sports-live-chart--scratch clipPath rect').first()).not.toHaveAttribute('width', '0');
    await page.getByRole('button', {name: /Reset$/}).click();
    await expect(page.locator('.sports-live-chart--scratch clipPath rect').first()).toHaveAttribute('width', '0');
    await page.screenshot({path: info.outputPath('history-scratch.png'), fullPage: true});
    assertThemeLedger(ledger);
});

for (const id of ['live', 'history'])
    test(`theme:markets ${id} successful no-history preserves event facts`, async ({page}, info) => {
        const data = scenario(id);
        data.replies.find(reply => reply.path.endsWith('price-history:batchGet'))!.json = {items: []};
        const ledger = await go(page, data);
        await expect(visible(page, 'No history')).toBeVisible();
        await expect(visible(page, id === 'live' ? '1–0' : '6–4, 7–6')).toBeVisible();
        await expect(page.getByRole('img', {name: 'Moneyline price history'})).toHaveCount(0);
        await page.screenshot({path: info.outputPath(`${id}-no-history.png`), fullPage: true});
        assertThemeLedger(ledger);
    });

for (const state of ['syncing', 'failed'])
    test(`theme:markets history sync ${state} stays independent of event data`, async ({page}, info) => {
        const data = scenario('history');
        data.replies.find(reply => reply.path.endsWith('sync-status'))!.json = {
            status: {state, startedAt: 1789372680, completedAt: 1789372740, errorMessage: state === 'failed' ? 'History sync source failed' : ''}
        };
        const ledger = await go(page, data);
        await expect(visible(page, state === 'failed' ? 'History sync source failed' : 'Syncing')).toBeVisible();
        await expect(visible(page, '6–4, 7–6')).toBeVisible();
        await page.screenshot({path: info.outputPath(`history-${state}.png`), fullPage: true});
        assertThemeLedger(ledger);
    });

test('theme:markets corners filter keeps 90-minute and full-match meanings separate', async ({page}, info) => {
    await page.setViewportSize({width: 390, height: 844});
    const ledger = await openThemeCase(page, 'markets-corners');
    await page.getByRole('textbox', {name: 'Search by team'}).fill('Netherlands');
    await expect(page.locator('.resource-table-compact__item:visible')).toHaveCount(1);
    const row = page.locator('.resource-table-compact__item:visible');
    await expect(row.getByText('6 · MISS', {exact: true})).toBeVisible();
    await expect(row.getByText('2–8', {exact: true})).toBeVisible();
    await expect(row.getByText('PEN 3–4', {exact: true})).toBeVisible();
    await page.getByRole('button', {name: 'View methodology & sources'}).click();
    await expect(page.getByText(/displayed sample and hit rates/)).toBeVisible();
    await page.screenshot({path: info.outputPath('corners-filter-method.png'), fullPage: true});
    expect(ledger.requests.filter(item => item.path.endsWith('/dataset'))).toHaveLength(1);
    assertThemeLedger(ledger);
});

test('theme:markets corners ascending order remains consistent after desktop to mobile resize', async ({page}) => {
    await page.setViewportSize({width: 1440, height: 900});
    const ledger = await openThemeCase(page, 'markets-corners');
    await page.getByRole('combobox', {name: 'Sort matches', exact: true}).click();
    await page.getByText('90-min total, lowest first', {exact: true}).click();
    await expect(page.locator('.ant-table-tbody .ant-table-row').first()).toContainText('Netherlands');
    await page.setViewportSize({width: 390, height: 844});
    await expect(page.locator('.resource-table-compact__item:visible').first()).toContainText('Netherlands');
    await expect(page.locator('.world-cup-corners-filters .ant-select-content').filter({hasText: '90-min total, lowest first'})).toBeVisible();
    checkReads(ledger);
});

test('theme:markets history writable separates saved reads from single-flight sync', async ({page}, info) => {
    await page.setViewportSize({width: 390, height: 844});
    const ledger = await openThemeCase(page, 'markets-history-writable');
    const read = page.getByRole('button', {name: 'Reload saved data', exact: true});
    const sync = page.getByRole('button', {name: 'Sync history', exact: true});
    await expect(read).toHaveCount(1);
    await expect(sync).toHaveCount(1);
    await expect(page.getByRole('button', {name: 'Refresh data', exact: true})).toHaveCount(0);
    const calls = (path: string) => ledger.requests.filter(item => item.path === path);
    const eventsPath = '/api/v1/sports-history/events';
    const historyPath = '/api/v1/sports-history/price-history:batchGet';
    const syncPath = '/api/v1/sports-history:refresh';
    await expect(page.getByRole('slider', {name: 'History time'}).first()).toBeVisible();
    await read.click();
    await expect.poll(() => calls(historyPath).length).toBe(2);
    expect(calls(syncPath)).toEqual([]);
    expect(calls(eventsPath).map(item => [item.method, item.query, item.realm])).toEqual([
        ['GET', '?limit=200', 'member'],
        ['GET', '?limit=200', 'member']
    ]);
    await sync.click();
    await expect(sync).toBeDisabled();
    await page.keyboard.press('Enter');
    await expect.poll(() => calls(syncPath).length).toBe(1);
    await page.screenshot({path: info.outputPath('history-sync-pending.png'), fullPage: true});
    await expect(page.getByText('Sports history refreshed', {exact: true})).toBeVisible();
    await expect(sync).toBeEnabled();
    await expect.poll(() => calls(historyPath).length).toBe(3);
    expect(calls(syncPath)).toEqual([{method: 'POST', path: syncPath, realm: 'member', query: '', body: {}}]);
    expect(calls(eventsPath).map(item => item.query)).toEqual(['?limit=200', '?limit=200', '?limit=200']);
    expect(calls(historyPath).map(item => ({method: item.method, realm: item.realm, query: item.query, body: item.body}))).toEqual(
        Array.from({length: 3}, () => ({
            method: 'POST',
            realm: 'member',
            query: '',
            body: {market_keys: ['hist-atp-ml', 'hist-wta-ml'], limit_per_token: 360}
        }))
    );
    expect(calls('/api/v1/sports-history/sync-status')).toHaveLength(3);
    assertThemeLedger(ledger);
});

test('theme:markets history read-only permits saved reload and hides write synchronization', async ({page}) => {
    const ledger = await openThemeCase(page, 'markets-history');
    await expect(page.getByRole('slider', {name: 'History time'}).first()).toBeVisible();
    await expect(page.getByRole('button', {name: 'Sync history', exact: true})).toHaveCount(0);
    await page.getByRole('button', {name: 'Reload saved data', exact: true}).click();
    await expect.poll(() => ledger.requests.filter(item => item.path.endsWith('price-history:batchGet')).length).toBe(2);
    expect(ledger.requests.filter(item => item.path === '/api/v1/sports-history:refresh')).toEqual([]);
    checkReads(ledger);
});

for (const id of ['hot', 'realtime', 'movers']) {
    test(`theme:markets wire ${id} omitted proto defaults retain false and known-zero counts`, async ({page}) => {
        const data = scenario(id);
        const body = data.replies[1].json as Record<string, unknown>;
        for (const field of [
            'stale',
            'connected',
            'candidateCount',
            'candidate_count',
            'monitoredMarkets',
            'monitored_markets',
            'monitoredTokens',
            'monitored_tokens',
            'subscribedMarkets',
            'subscribed_markets',
            'subscribedTokens',
            'subscribed_tokens'
        ])
            delete body[field];
        const ledger = await go(page, data);
        await expect(visible(page, 'Current snapshot')).toBeVisible();
        await expect(visible(page, '0 candidates · 0 monitored markets')).toBeVisible();
        if (id !== 'hot') {
            await expect(visible(page, 'Sampling disconnected')).toBeVisible();
            await expect(page.getByText(/0 monitored tokens/)).toBeVisible();
        }
        checkReads(ledger);
    });
    for (const invalid of [null, 'invalid'])
        test(`theme:markets wire ${id} explicit counter ${String(invalid)} is unknown`, async ({page}) => {
            const data = scenario(id);
            const body = data.replies[1].json as Record<string, unknown>;
            for (const field of ['candidateCount', 'monitoredMarkets', 'monitoredTokens', 'subscribedMarkets', 'subscribedTokens']) body[field] = invalid;
            const ledger = await go(page, data);
            await expect(visible(page, 'Unknown candidates · Unknown monitored markets')).toBeVisible();
            if (id !== 'hot') await expect(page.getByText(/Unknown monitored tokens/)).toBeVisible();
            checkReads(ledger);
        });
}
