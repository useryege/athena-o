import {test, expect} from '@playwright/test';
import fs from 'node:fs';
import {installTraderSyncRoutes, memberPath, assertFixture, scenarioActivity, subscriptionID, fixtureStates, routeLedgers} from './trader-sync-fixtures';
test.afterEach(async ({page}, info) => {
    fs.writeFileSync(info.outputPath('fixture-api-ledger.json'), JSON.stringify(routeLedgers.get(page), null, 2));
    await assertFixture(page);
});
for (const [scenario, label] of [
    ['in-app-only', 'Notification results'],
    ['ordinary-pending', 'Queued'],
    ['ordinary-sending', 'Sending'],
    ['ordinary-sent', 'Telegram accepted'],
    ['ordinary-failed', 'Delivery failed'],
    ['ordinary-unknown', 'Delivery unknown'],
    ['summary-waiting', 'Waiting for summary'],
    ['summary-cancelled', 'Summary cancelled before freeze']
]) {
    test('notification ' + scenario, async ({page}, info) => {
        await installTraderSyncRoutes(page, scenario);
        await page.goto(memberPath('/trader-sync/activities/1'));
        await expect(page.getByRole('heading', {name: 'Activity', exact: true, level: 1})).toBeVisible();
        await expect(page.getByText(label, {exact: scenario !== 'summary-cancelled'}).first()).toBeVisible();
        if (scenario === 'ordinary-sent') {
            await page.getByText('Delivery evidence', {exact: true}).click();
            await expect(page.getByText('Submit time not recorded').first()).toBeVisible();
            await expect(page.getByText('Submission-to-result duration cannot be determined.').first()).toBeVisible();
        }
        await page.screenshot({path: info.outputPath(scenario + '.png'), fullPage: true});
    });
}
test('desktop and mobile product documents remain within viewport', async ({page}, info) => {
    await installTraderSyncRoutes(page, 'precision-combo');
    for (const width of [1440, 1280, 900, 390])
        for (const theme of ['light', 'dark'] as const) {
            await page.emulateMedia({colorScheme: theme});
            await page.setViewportSize({width, height: 900});
            await page.goto(memberPath('/trader-sync/activities/1'));
            await expect(page.getByRole('heading', {name: 'Activity', exact: true, level: 1})).toBeVisible();
            await expect(page.locator('base')).toHaveAttribute('href', memberPath('/'));
            await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
            if (width === 1440) {
                const colors = await contrastRows(page);
                fs.writeFileSync(info.outputPath(`contrast-${theme}.json`), JSON.stringify(colors, null, 2));
                expect.soft(colors.filter(x => !x.exempt && !x.outOfScope && x.ratio < x.required)).toEqual([]);
            }
            expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
            await page.screenshot({path: info.outputPath(`${width}-${theme}.png`), fullPage: true});
        }
});
test('101 part mixed and independent all-sent batches keep whole counts', async ({page}, info) => {
    await installTraderSyncRoutes(page, 'summary-frozen-mixed');
    await page.goto(memberPath('/trader-sync/summaries/901'));
    await expect(page.getByRole('heading', {name: 'Summary batch', exact: true, level: 1})).toBeVisible();
    await expect(page.locator('article[data-part-id]')).toHaveCount(50);
    await expect(page.getByText('All parts accepted by Telegram')).toHaveCount(0);
    const nav = page.getByRole('navigation', {name: 'Summary part pages'});
    await nav.getByRole('button', {name: 'Next', exact: true}).click();
    await expect(page.getByRole('heading', {name: 'Part 51 of 101', exact: true})).toBeVisible();
    await expect(page.locator('article[data-part-id]')).toHaveCount(50);
    await nav.getByRole('button', {name: 'Next', exact: true}).click();
    await expect(page.getByRole('heading', {name: 'Part 101 of 101', exact: true})).toBeVisible();
    await expect(page.locator('article[data-part-id]')).toHaveCount(1);
    await page.screenshot({path: info.outputPath('mixed-tail.png'), fullPage: true});
});
test('independent all-sent batch states complete', async ({page}) => {
    await installTraderSyncRoutes(page, 'summary-frozen-all-sent');
    await page.goto(memberPath('/trader-sync/summaries/902'));
    await expect(page.getByText('All parts accepted by Telegram')).toBeVisible();
});
test('51 history records paginate using real product navigation', async ({page}) => {
    await installTraderSyncRoutes(page, 'monitoring');
    await page.goto(memberPath(`/trader-sync/subscriptions/${subscriptionID}`));
    await expect(page.getByRole('heading', {name: 'Subscription', exact: true, level: 1})).toBeVisible();
    const next = page.getByRole('button', {name: 'More history', exact: true});
    await expect(page.locator('[data-history-id]')).toHaveCount(50);
    await next.click();
    await expect(page.locator('[data-history-id]')).toHaveCount(1);
    await expect(next).toBeDisabled();
    await page.getByRole('button', {name: 'Previous history', exact: true}).click();
    await expect(next).toBeEnabled();
});
test('exact clipboard values, 101 Combo legs and selection survive polling', async ({page, context}, info) => {
    await context.grantPermissions(['clipboard-read', 'clipboard-write']);
    await installTraderSyncRoutes(page, 'precision-combo');
    await page.goto(memberPath('/trader-sync/activities/1'));
    await expect(page.getByRole('heading', {name: 'Activity', exact: true, level: 1})).toBeVisible();
    const fact = page.locator('.trader-sync-fact-values > div').filter({has: page.getByText('Trade value', {exact: true})});
    await fact.getByRole('button', {name: 'Copy', exact: true}).click();
    expect(await page.evaluate(() => navigator.clipboard.readText())).toBe('0.000001');
    await page.getByText('Original amounts and source evidence', {exact: true}).click();
    const position = page.locator('details dl > div').filter({has: page.locator('dt').filter({hasText: /^Position ID$/})});
    await position.getByRole('button', {name: 'Copy', exact: true}).click();
    expect(await page.evaluate(() => navigator.clipboard.readText())).toBe('90071992547409931234567890');
    await page.getByText('101 conditions', {exact: true}).click();
    await expect(page.locator('.trader-sync-combo ol > li')).toHaveCount(101);
    await expect(page.getByText('Not all conditions met', {exact: false})).toBeVisible();
    const first = page.locator('.trader-sync-combo ol > li').first();
    await first.scrollIntoViewIfNeeded();
    await first.evaluate(el => {
        const range = document.createRange();
        range.selectNodeContents(el);
        const selection = getSelection()!;
        selection.removeAllRanges();
        selection.addRange(range);
    });
    const selected = await page.evaluate(() => getSelection()!.toString());
    const before = await page.evaluate(() => scrollY);
    await page.waitForResponse(r => new URL(r.url()).pathname === memberPath('/api/v1/trader-sync/activities/1'));
    expect(await page.evaluate(() => getSelection()!.toString())).toBe(selected);
    expect(Math.abs((await page.evaluate(() => scrollY)) - before)).toBeLessThan(2);
    const link = first.getByRole('link', {name: 'View market'});
    await expect(link).toHaveAttribute('target', '_blank');
    await expect(link).toHaveAttribute('href', 'https://polymarket.com/event/synthetic-browser-condition');
    await expect(page).toHaveURL(new RegExp('/trader-sync/activities/1$'));
    await page.screenshot({path: info.outputPath('selected-combo.png')});
});
test('long first activity page restores actual scroll after one-row tail', async ({page}, info) => {
    await installTraderSyncRoutes(page, 'history-scroll');
    await page.goto(memberPath('/trader-sync'));
    await expect(page.locator('article[data-activity-id]')).toHaveCount(50);
    const next = page.getByRole('navigation', {name: 'Activity pages'}).getByRole('button', {name: 'Next', exact: true});
    await next.scrollIntoViewIfNeeded();
    const before = await page.evaluate(() => scrollY);
    expect(before).toBeGreaterThan(1000);
    await next.click();
    await expect(page.locator('article[data-activity-id]')).toHaveCount(1);
    await page.getByRole('navigation', {name: 'Activity pages'}).getByRole('button', {name: 'Previous', exact: true}).click();
    await expect(page.locator('article[data-activity-id]')).toHaveCount(50);
    await expect.poll(() => page.evaluate(() => scrollY)).toBeCloseTo(before, 0);
    await page.screenshot({path: info.outputPath('restored-scroll.png')});
});
test('summary activity return belongs to the current navigation', async ({page}) => {
    await installTraderSyncRoutes(page, 'summary-frozen-mixed');
    await page.goto(memberPath('/trader-sync/summaries/901'));
    await page.getByRole('link', {name: 'View activity', exact: true}).click();
    await page.getByRole('link', {name: 'Back to summary batch', exact: true}).click();
    await expect(page.getByRole('heading', {name: 'Summary batch', exact: true, level: 1})).toBeVisible();
    await page.getByRole('link', {name: 'Back to Trader Sync', exact: true}).click();
    await page.getByRole('link', {name: 'View activity', exact: true}).click();
    await expect(page.getByRole('link', {name: 'Back to summary batch', exact: true})).toHaveCount(0);
    await expect(page.getByRole('link', {name: 'Back to Trader Sync', exact: true})).toBeVisible();
});
test('administrator runtime gauges, epochs and synthetic windows preserve sources', async ({page}, info) => {
    await installTraderSyncRoutes(page, 'admin-runtime');
    await page.goto(memberPath('/admin/service-status'));
    await expect(page.getByRole('heading', {name: 'Service Status', exact: true, level: 1})).toBeVisible();
    await expect(page.locator('base')).toHaveAttribute('href', memberPath('/admin/'));
    await expect(page.locator('meta[name="athena-deployment-base-href"]')).toHaveAttribute('content', memberPath('/'));
    await page.getByRole('tab', {name: /^Notifications/}).click();
    await expect(page.getByText('55000 ms', {exact: true})).toBeVisible();
    await expect(page.getByText('5000 ms', {exact: true})).toBeVisible();
    await page.getByRole('tab', {name: /^Trader Sync/}).click();
    await expect(page.getByText('synthetic_window', {exact: true}).filter({visible: true})).toBeVisible();
    await expect(page.getByText('Current gauge', {exact: true}).first()).toBeVisible();
    await page.screenshot({path: info.outputPath('admin-runtime.png'), fullPage: true});
});
test('administrator summaries preserve large counts and exclude private controls on mobile', async ({page}, info) => {
    await installTraderSyncRoutes(page, 'admin-summary');
    for (const width of [1440, 1280, 900, 390])
        for (const theme of ['light', 'dark'] as const) {
            await page.emulateMedia({colorScheme: theme});
            await page.setViewportSize({width, height: 900});
            await page.goto(memberPath('/admin/trader-sync/subscriptions'));
            await expect(page.getByRole('heading', {name: 'Trader Sync', exact: true, level: 1})).toBeVisible();
            await expect(page.getByText('9007199254740993', {exact: true}).filter({visible: true})).toBeVisible();
            await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
            if (width === 1440) {
                const colors = await contrastRows(page);
                fs.writeFileSync(info.outputPath(`contrast-${theme}.json`), JSON.stringify(colors, null, 2));
                expect.soft(colors.filter(x => !x.exempt && !x.outOfScope && x.ratio < x.required)).toEqual([]);
            }
            expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
            await expect(page.getByRole('button', {name: /Pause|Resume|Cancel subscription|Resend/})).toHaveCount(0);
            await page.screenshot({path: info.outputPath(`admin-${width}-${theme}.png`), fullPage: true});
        }
});
test('new member text contrast meets AA using computed rendered colors', async ({page}, info) => {
    await installTraderSyncRoutes(page, 'precision-combo');
    await page.goto(memberPath('/trader-sync/activities/1'));
    await expect(page.getByRole('heading', {name: 'Activity', exact: true, level: 1})).toBeVisible();
    const rows = await contrastRows(page);
    fs.writeFileSync(info.outputPath('computed-text-contrast.json'), JSON.stringify(rows, null, 2));
    expect(rows.length).toBeGreaterThan(10);
    expect(rows.filter(x => !x.exempt && !x.outOfScope && x.ratio < x.required)).toEqual([]);
});
test('administrator 403 clears summaries and userinfo failure offers controlled retry', async ({page}, info) => {
    await installTraderSyncRoutes(page, 'admin-summary');
    await page.goto(memberPath('/admin/trader-sync/subscriptions'));
    await expect(page.getByText('9007199254740993', {exact: true}).filter({visible: true})).toBeVisible();
    const state = fixtureStates.get(page)!;
    state.denyAdmin = true;
    state.failUserInfo = true;
    await page.getByRole('button', {name: 'Refresh data', exact: true}).click();
    await expect(page.getByText('Could not verify administrator access', {exact: true})).toBeVisible();
    await expect(page.getByText('9007199254740993', {exact: true})).toHaveCount(0);
    await page.screenshot({path: info.outputPath('admin-recheck-error.png')});
    state.denyAdmin = false;
    state.failUserInfo = false;
    await page.getByRole('button', {name: /Retry access check$/}).click();
    await expect(page.getByText('9007199254740993', {exact: true}).filter({visible: true})).toBeVisible();
});
for (const [route, title, scenario] of [
    ['/trader-sync', 'Trader Sync', 'monitoring'],
    ['/trader-sync/add', 'Add trader', 'monitoring'],
    ['/trader-sync/subscriptions', 'Subscriptions', 'monitoring'],
    [`/trader-sync/subscriptions/${subscriptionID}`, 'Subscription', 'monitoring'],
    ['/trader-sync/summaries/901', 'Summary batch', 'summary-frozen-mixed']
]) {
    test('full page viewport and theme matrix ' + title, async ({page}, info) => {
        await installTraderSyncRoutes(page, scenario);
        const results = [];
        for (const width of [1440, 1280, 900, 390])
            for (const theme of ['light', 'dark'] as const) {
                await page.emulateMedia({colorScheme: theme});
                await page.setViewportSize({width, height: 900});
                await page.goto(memberPath(route));
                await expect(page.getByRole('heading', {name: title, exact: true, level: 1})).toBeVisible();
                await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
                if (width === 1440) {
                    const colors = await contrastRows(page);
                    fs.writeFileSync(info.outputPath(`contrast-${theme}.json`), JSON.stringify(colors, null, 2));
                    expect.soft(colors.filter(x => !x.exempt && !x.outOfScope && x.ratio < x.required)).toEqual([]);
                }
                expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
                results.push({
                    width,
                    theme,
                    ...(await page.evaluate(() => ({
                        colorScheme: getComputedStyle(document.documentElement).colorScheme,
                        bodyBackground: getComputedStyle(document.body).backgroundColor
                    })))
                });
                await page.screenshot({path: info.outputPath(`${width}-${theme}.png`), fullPage: width === 390});
            }
        fs.writeFileSync(info.outputPath('theme-results.json'), JSON.stringify(results, null, 2));
    });
}
test('keyboard modal focus returns to cancellation trigger', async ({page}, info) => {
    await installTraderSyncRoutes(page, 'monitoring');
    await page.goto(memberPath(`/trader-sync/subscriptions/${subscriptionID}`));
    const trigger = page.getByRole('button', {name: 'Cancel subscription', exact: true});
    await trigger.focus();
    await page.keyboard.press('Enter');
    const dialog = page.getByRole('dialog', {name: 'Cancel subscription?'});
    await expect(dialog).toBeVisible();
    await expect.poll(() => dialog.evaluate(el => el.contains(document.activeElement))).toBe(true);
    const focus = [];
    for (let i = 0; i < 5; i++) {
        await page.keyboard.press('Tab');
        await expect.poll(() => dialog.evaluate(el => el.contains(document.activeElement))).toBe(true);
        focus.push(await page.evaluate(() => document.activeElement?.outerHTML));
    }
    const buttons = dialog.getByRole('button');
    await buttons.first().focus();
    await page.keyboard.press('Shift+Tab');
    await expect(buttons.last()).toBeFocused();
    await page.keyboard.press('Tab');
    await expect(buttons.first()).toBeFocused();
    fs.writeFileSync(info.outputPath('keyboard-focus.json'), JSON.stringify(focus, null, 2));
    expect(await dialog.evaluate(el => el.contains(document.activeElement))).toBe(true);
    await page.keyboard.press('Escape');
    await expect(dialog).toHaveCount(0);
    await expect(trigger).toBeFocused();
    await page.screenshot({path: info.outputPath('returned-focus.png')});
});
test.describe('touch viewport', () => {
    test.use({hasTouch: true, viewport: {width: 390, height: 844}});
    test('touch opens all Combo conditions without leaving activity', async ({page}) => {
        await installTraderSyncRoutes(page, 'precision-combo');
        await page.goto(memberPath('/trader-sync/activities/1'));
        await page.getByText('101 conditions', {exact: true}).tap();
        await expect(page.locator('.trader-sync-combo ol > li')).toHaveCount(101);
        await page.getByText('Original amounts and source evidence', {exact: true}).tap();
        await expect(page.getByText('90071992547409931234567890', {exact: true})).toBeVisible();
        await expect(page).toHaveURL(/\/trader-sync\/activities\/1$/);
    });
});
test('huge monetary value copies exact decimal digits', async ({page, context}) => {
    await context.grantPermissions(['clipboard-read', 'clipboard-write']);
    await installTraderSyncRoutes(page, 'precision-huge');
    await page.goto(memberPath('/trader-sync/activities/1'));
    const fact = page.locator('.trader-sync-fact-values > div').filter({has: page.getByText('Trade value', {exact: true})});
    await fact.getByRole('button', {name: 'Copy', exact: true}).click();
    expect(await page.evaluate(() => navigator.clipboard.readText())).toBe('90071992547409931234.56789');
});
test('market external link opens separately without triggering activity navigation', async ({page, context}) => {
    await installTraderSyncRoutes(page, 'precision-combo');
    await context.route('https://polymarket.com/**', route => route.fulfill({contentType: 'text/html', body: 'Controlled external market landing'}));
    await page.goto(memberPath('/trader-sync'));
    await page.getByText('101 conditions', {exact: true}).click();
    const popup = page.waitForEvent('popup');
    await page.getByRole('link', {name: 'View market', exact: true}).first().click();
    const opened = await popup;
    await expect(opened).toHaveURL('https://polymarket.com/event/synthetic-browser-condition');
    await expect(page).toHaveURL(new RegExp(memberPath('/trader-sync') + '$'));
    await opened.close();
});

async function contrastRows(page: import('@playwright/test').Page, selector = 'main') {
    await page
        .locator('main')
        .getByRole('button', {name: 'More history', exact: true})
        .isVisible()
        .then(async visible => {
            if (visible) await expect(page.locator('[data-history-id]')).toHaveCount(50);
        });
    await page.evaluate(async () => {
        await Promise.all(
            document
                .getAnimations()
                .filter(a => a.effect?.getComputedTiming().iterations !== Infinity)
                .map(a => a.finished.catch(() => {}))
        );
    });
    return await page.locator(selector).evaluate(root => {
        const canvas = document.createElement('canvas');
        canvas.width = canvas.height = 1;
        const ctx = canvas.getContext('2d')!;
        const rgba = (s: string) => {
            ctx.clearRect(0, 0, 1, 1);
            ctx.fillStyle = s;
            ctx.fillRect(0, 0, 1, 1);
            return Array.from(ctx.getImageData(0, 0, 1, 1).data).map(x => x / 255);
        };
        const composite = (f: number[], b: number[]) => [f[0] * f[3] + b[0] * (1 - f[3]), f[1] * f[3] + b[1] * (1 - f[3]), f[2] * f[3] + b[2] * (1 - f[3]), 1];
        const lum = (c: number[]) =>
            c
                .slice(0, 3)
                .map(x => (x <= 0.04045 ? x / 12.92 : ((x + 0.055) / 1.055) ** 2.4))
                .reduce((a, x, i) => a + x * [0.2126, 0.7152, 0.0722][i], 0);
        const rows = [...root.querySelectorAll('*')]
            .filter(el => [...el.childNodes].some(n => n.nodeType === Node.TEXT_NODE && n.textContent!.trim()) && el.getBoundingClientRect().height > 0)
            .map(el => {
                const css = getComputedStyle(el);
                let chain: Element[] = [];
                for (let node: Element | null = el; node; node = node.parentElement) chain.unshift(node);
                let bg = [1, 1, 1, 1];
                for (const node of chain) bg = composite(rgba(getComputedStyle(node).backgroundColor), bg);
                const fg = composite(rgba(css.color), bg),
                    a = lum(fg),
                    b = lum(bg),
                    ratio = (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05),
                    large = parseFloat(css.fontSize) >= 24 || (parseFloat(css.fontSize) >= 18.66 && parseInt(css.fontWeight) >= 700);
                return {
                    text: el.textContent!.trim().slice(0, 90),
                    foreground: css.color,
                    background: bg,
                    ratio,
                    required: large ? 3 : 4.5,
                    exempt: el.closest('[disabled], [aria-disabled="true"]') ? 'inactive UI control' : null,
                    outOfScope:
                        location.pathname.endsWith('/service-status') &&
                        (el.closest('section')?.querySelector('h2')?.textContent === 'Services' ||
                            (el.classList.contains('ant-tag') && !el.classList.contains('trader-sync-runtime-status')))
                            ? 'Existing service health / notification flag outside new Trader Sync text scope'
                            : null,
                    element: el.outerHTML.slice(0, 250)
                };
            });
        for (const el of root.querySelectorAll('input[placeholder]')) {
            const css = getComputedStyle(el, '::placeholder');
            let bg = [1, 1, 1, 1];
            const chain: Element[] = [];
            for (let n: Element | null = el; n; n = n.parentElement) chain.unshift(n);
            for (const n of chain) bg = composite(rgba(getComputedStyle(n).backgroundColor), bg);
            const fg = composite(rgba(css.color), bg),
                a = lum(fg),
                b = lum(bg);
            rows.push({
                text: '::placeholder ' + el.getAttribute('placeholder'),
                foreground: css.color,
                background: bg,
                ratio: (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05),
                required: 4.5,
                exempt: el.matches(':disabled') ? 'inactive UI control' : (el as HTMLInputElement).value ? 'placeholder not displayed' : null,
                outOfScope: null,
                element: el.outerHTML
            });
        }
        return rows;
    });
}
test('resolved confirmation and cancellation modal color roles', async ({page}, info) => {
    await installTraderSyncRoutes(page, 'monitoring');
    for (const theme of ['light', 'dark'] as const) {
        await page.emulateMedia({colorScheme: theme});
        await page.goto(memberPath('/trader-sync/add'));
        await page.getByLabel('Wallet address or Polymarket profile URL').fill('0x000000000000000000000000000000000000002a');
        await page.getByRole('button', {name: 'Resolve trader', exact: true}).click();
        await expect(page.getByLabel('Private note')).toBeVisible();
        await page.getByLabel('Private note').fill('😀'.repeat(21));
        const rows = await contrastRows(page);
        fs.writeFileSync(info.outputPath(`confirmation-${theme}.json`), JSON.stringify(rows, null, 2));
        expect.soft(rows.filter(x => !x.exempt && !x.outOfScope && x.ratio < x.required)).toEqual([]);
        await page.goto(memberPath(`/trader-sync/subscriptions/${subscriptionID}`));
        await page.getByRole('button', {name: 'Cancel subscription', exact: true}).click();
        await expect(page.getByRole('dialog')).toBeVisible();
        const modal = await contrastRows(page, '[role="dialog"]');
        fs.writeFileSync(info.outputPath(`modal-${theme}.json`), JSON.stringify(modal, null, 2));
        expect.soft(modal.filter(x => !x.exempt && !x.outOfScope && x.ratio < x.required)).toEqual([]);
        await page.keyboard.press('Escape');
    }
});

for (const [route, title, scenario] of [
    ['/admin/service-status', 'Service Status', 'admin-runtime'],
    [`/admin/trader-sync/subscriptions/${subscriptionID}`, 'Subscription summary', 'admin-summary']
]) {
    test('administrator full viewport and theme matrix ' + title, async ({page}, info) => {
        await installTraderSyncRoutes(page, scenario);
        for (const width of [1440, 1280, 900, 390])
            for (const theme of ['light', 'dark'] as const) {
                await page.emulateMedia({colorScheme: theme});
                await page.setViewportSize({width, height: 900});
                await page.goto(memberPath(route));
                await expect(page.getByRole('heading', {name: title, exact: true, level: 1})).toBeVisible();
                await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
                expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
                if (width === 1440) {
                    const rows = await contrastRows(page);
                    fs.writeFileSync(info.outputPath(`contrast-${theme}.json`), JSON.stringify(rows, null, 2));
                    expect.soft(rows.filter(x => !x.exempt && !x.outOfScope && x.ratio < x.required)).toEqual([]);
                }
                await page.screenshot({path: info.outputPath(`${width}-${theme}.png`), fullPage: width === 390});
            }
    });
}
test('summary retains both independent pages and actual scroll when returning from activity', async ({page}, info) => {
    await installTraderSyncRoutes(page, 'summary-frozen-history');
    await page.goto(memberPath('/trader-sync/summaries/901'));
    await expect(page.locator('article[data-activity-id]')).toHaveCount(50);
    const parts = page.getByRole('navigation', {name: 'Summary part pages'}),
        activities = page.getByRole('navigation', {name: 'Summary activity pages'});
    await parts.getByRole('button', {name: 'Next', exact: true}).click();
    await expect(page.getByRole('heading', {name: 'Part 51 of 101', exact: true})).toBeVisible();
    await activities.getByRole('button', {name: 'Next', exact: true}).click();
    await expect(page.locator('article[data-activity-id]')).toHaveCount(1);
    const link = page.getByRole('link', {name: 'View activity', exact: true});
    await link.scrollIntoViewIfNeeded();
    const before = await page.evaluate(() => scrollY);
    await link.click();
    await page.getByRole('link', {name: 'Back to summary batch', exact: true}).click();
    await expect(page.locator('article[data-activity-id]')).toHaveCount(1);
    await expect(page.getByRole('heading', {name: 'Part 51 of 101', exact: true})).toBeVisible();
    await expect.poll(() => page.evaluate(() => scrollY)).toBeCloseTo(before, 0);
    fs.writeFileSync(info.outputPath('summary-return-scroll.json'), JSON.stringify({before, after: await page.evaluate(() => scrollY)}));
    await activities.getByRole('button', {name: 'Previous', exact: true}).click();
    await expect(page.locator('article[data-activity-id]')).toHaveCount(50);
});

test('administrator newer 401 fences an older successful permission response', async ({page}, info) => {
    await installTraderSyncRoutes(page, 'admin-summary');
    await page.goto(memberPath('/admin/account/profile'));
    const state = fixtureStates.get(page)!;
    state.holdUserInfo = true;
    try {
        await page.locator('input[autocomplete="name"]').fill('Updated administrator');
        await page.getByRole('button', {name: 'Save profile', exact: true}).click();
        await expect.poll(() => state.userInfoReady).toBe(true);
        state.adminUnauthorized = true;
        await page.getByRole('menuitem', {name: /Trader Sync$/}).click();
        await expect(page).toHaveURL(/\/admin\/login\?returnTo=/);
        state.releaseUserInfo!();
        await expect(page.getByText('9007199254740993', {exact: true})).toHaveCount(0);
        expect(new URL(page.url()).pathname).toBe(memberPath('/admin/login'));
        await page.screenshot({path: info.outputPath('admin-401-late-response.png')});
    } finally {
        state.releaseUserInfo?.();
    }
});

test('single dark Trader Sync button roles', async ({page}, info) => {
    await installTraderSyncRoutes(page, 'monitoring');
    for (const width of [1440, 390]) {
        await page.setViewportSize({width, height: 900});
        await page.goto(memberPath('/trader-sync/add'));
        await page.getByLabel('Wallet address or Polymarket profile URL').fill('0x000000000000000000000000000000000000002a');
        // Styling is fixed dark and does not need a conditional root marker.
        await page.locator('html').evaluate(element => element.removeAttribute('data-theme'));
        const add = page.getByRole('button', {name: 'Resolve trader', exact: true});
        await expect(add).toBeEnabled();
        await expect(add).toHaveCSS('color', 'rgb(6, 8, 11)');
        await expect(add).toHaveCSS('background-color', 'rgb(0, 255, 167)');
        await page.screenshot({path: info.outputPath(`trader-add-${width}.png`), fullPage: true});

        await page.goto(memberPath('/trader-sync/subscriptions'));
        const view = page.locator('.trader-sync-view').first();
        await expect(view).toBeVisible();
        await page.locator('html').evaluate(element => element.removeAttribute('data-theme'));
        await expect(view).toHaveCSS('color', 'rgb(6, 8, 11)');
        await expect(view).toHaveCSS('background-color', 'rgb(0, 255, 167)');
        await page.screenshot({path: info.outputPath(`trader-view-${width}.png`), fullPage: true});

        await page.goto(memberPath(`/trader-sync/subscriptions/${subscriptionID}`));
        await page.getByRole('button', {name: 'Cancel subscription', exact: true}).click();
        await page.locator('html').evaluate(element => element.removeAttribute('data-theme'));
        const cancel = page.locator('.trader-sync-cancel-modal .ant-btn-primary.ant-btn-dangerous');
        await expect(cancel).toBeVisible();
        await expect(cancel).toHaveCSS('color', 'rgb(6, 8, 11)');
        await expect(cancel).toHaveCSS('background-color', 'rgb(245, 140, 155)');
        await page.getByRole('dialog').screenshot({path: info.outputPath(`trader-cancel-${width}.png`)});
        await page.keyboard.press('Escape');
    }
});
