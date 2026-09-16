import {expect, test} from '@playwright/test';
import {assertThemeLayout, assertThemeLedger, openThemeCase} from './routes';

for (const id of ['hot', 'realtime', 'movers', 'proposals', 'disputes']) {
    test(`theme:markets ${id} first failure excludes successful empty state`, async ({page}) => {
        const ledger = await openThemeCase(page, `markets-${id}-failed`);
        await expect(page.getByText('Request failed', {exact: true})).toBeVisible();
        await expect(page.locator('.ant-empty:visible')).toHaveCount(0);
        await expect(page.locator('.resource-table-pagination:visible')).toHaveCount(0);
        assertThemeLedger(ledger);
    });
}

import {themeCases} from './cases';
import {installThemeCase} from './routes';
import type {ThemeCase, ThemeLedger} from './contracts';
const mainIds = ['hot', 'realtime', 'movers', 'proposals', 'disputes'];
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
        await expect(page.locator('.ant-empty:visible').first()).toBeVisible();
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

for (const id of ['hot', 'proposals'])
    test(`theme:markets ${id} refresh failure retains stale facts`, async ({page}) => {
        const data = scenario(id);
        const ledger = await go(page, data);
        const firstData = data.replies[1];
        await expect.poll(() => ledger.requests.filter(r => r.path === firstData.path).length).toBeGreaterThan(0);
        await expect(page.locator('.radar-record:visible,.managed-oo-record:visible').first()).toBeVisible();
        firstData.status = 503;
        firstData.json = {message: 'Refresh source unavailable'};
        await page.getByRole('button', {name: 'Refresh data', exact: true}).click();
        await expect(visible(page, 'Request failed')).toBeVisible();
        await expect(
            page
                .getByText(/Stale (snapshot|saved events|dataset)/)
                .filter({visible: true})
                .first()
        ).toBeVisible();
        await expect(page.locator('.radar-record:visible,.managed-oo-record:visible').first()).toBeVisible();
        assertThemeLedger(ledger);
    });

test('theme:markets Sources and evidence preserve complete original fields', async ({page}, info) => {
    await page.setViewportSize({width: 390, height: 844});
    const ledger = await openThemeCase(page, 'markets-proposals');
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
        await expect(page.locator('.radar-record:visible,.managed-oo-record:visible').first()).toBeVisible();
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
            .locator('.radar-record:visible,.managed-oo-record:visible')
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
