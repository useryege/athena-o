import {expect, test, type Page} from '@playwright/test';
import type {ThemeCase} from './theme-refactor/contracts';
import {themeCases} from './theme-refactor/cases';
import {assertThemeLedger, installThemeCase} from './theme-refactor/routes';
import {assertFixture, installTraderSyncRoutes, memberPath} from './trader-sync-fixtures';

const modules = [
    {key: 'trader_sync', label: 'Trader Sync', route: '/trader-sync/activities/1'},
    {key: 'solana', label: 'Solana', scenario: 'foundations-solana'},
    {key: 'market_radar', label: 'Market Radar', scenario: 'markets-hot'},
    {key: 'managed_oo', label: 'Managed OO', scenario: 'markets-proposals'},
    {key: 'profit_sharing', label: 'Profit Sharing', scenario: 'foundations-member-collecting'},
    {key: 'worm', label: 'Worm Trading', scenario: 'worm-combinations-new'}
];
const viewports = [
    {name: 'desktop', width: 1440, height: 900},
    {name: 'mobile', width: 390, height: 844}
];

function withSession(scenario: ThemeCase): ThemeCase {
    const copy = structuredClone(scenario);
    if (!copy.replies.some(reply => reply.path === '/api/v1/session/userinfo')) {
        const bootstrap = copy.replies.find(reply => reply.path === '/api/v1/app/bootstrap')!.json as any;
        copy.replies.push({method: 'GET', path: '/api/v1/session/userinfo', realm: copy.realm, status: 200, json: bootstrap.session.userInfo || bootstrap.session.user_info});
    }
    return copy;
}

async function accessRoutes(page: Page, realm: 'member' | 'admin') {
    const state = {
        rows: modules.map(item => ({module_key: item.key, state: 1})),
        reads: 0,
        writes: [] as {key: string; state: number}[],
        dropSave: false,
        holdRead: false,
        holdSave: false,
        releaseSave: undefined as (() => void) | undefined
    };
    page.once('close', () => state.releaseSave?.());
    await page.route('**/api/v1/**', async route => {
        if (!new URL(route.request().url()).pathname.includes('/module-access')) {
            await route.fallback();
            return;
        }
        const request = route.request();
        expect(request.headers()['x-athena-application-realm']).toBe(realm);
        const path = new URL(request.url()).pathname;
        expect(path.startsWith(memberPath('/api/v1/'))).toBe(true);
        if (request.method() === 'PUT') {
            const key = path.split('/').at(-1)!;
            const {state: value} = request.postDataJSON();
            state.writes.push({key, state: value});
            const row = state.rows.find(item => item.module_key === key)!;
            row.state = value;
            if (state.dropSave) {
                state.dropSave = false;
                await route.abort('failed');
                return;
            }
            const result = structuredClone(row);
            if (state.holdSave) {
                state.holdSave = false;
                await new Promise<void>(resolve => {
                    state.releaseSave = resolve;
                });
            }
            await route.fulfill({json: {setting: result}});
            return;
        }
        state.reads++;
        if (state.holdRead) {
            await route.abort('failed');
            return;
        }
        await route.fulfill({json: path.endsWith('/module-access-states') ? {states: state.rows} : {settings: state.rows}});
    });
    return state;
}

for (const viewport of viewports) {
    for (const item of modules) {
        test(`module-access ${viewport.name} ${item.key} direct route closes, preserves navigation, and reopens`, async ({page}, info) => {
            await page.setViewportSize(viewport);
            const scenario = themeCases.find(entry => entry.id === item.scenario);
            const ledger = scenario ? await installThemeCase(page, withSession(scenario)) : undefined;
            if (!scenario) await installTraderSyncRoutes(page, 'ordinary-sent');
            const control = await accessRoutes(page, 'member');
            const row = control.rows.find(entry => entry.module_key === item.key)!;
            row.state = 2;
            const route = scenario?.route || item.route!;
            await page.goto(memberPath(route));
            await expect(page.getByText('This module is not open yet', {exact: true})).toBeVisible();
            expect(new URL(page.url()).pathname).toBe(memberPath(route.split('?')[0]));
            await expect(page.getByRole('button', {name: 'Open account menu'})).toBeVisible();
            if (viewport.name === 'mobile') await page.getByRole('button', {name: 'Open navigation'}).click();
            await expect(page.getByRole('menuitem', {name: new RegExp(item.label + '$')}).first()).toBeVisible();
            if (viewport.name === 'mobile') await page.keyboard.press('Escape');
            await page.screenshot({path: info.outputPath(`${item.key}-closed-${viewport.name}.png`), fullPage: true});
            row.state = 1;
            await page.getByRole('button', {name: 'Check access'}).click();
            await expect(page.getByRole('heading', {name: scenario?.heading || 'Activity', exact: true, level: 1})).toBeVisible();
            row.state = 2;
            await page.evaluate(() => window.dispatchEvent(new Event('focus')));
            await expect(page.getByText('This module is not open yet', {exact: true})).toBeVisible();
            await expect(page.getByRole('heading', {name: scenario?.heading || 'Activity', exact: true, level: 1})).toHaveCount(0);
            expect(control.writes).toEqual([]);
            if (ledger) assertThemeLedger(ledger);
            else await assertFixture(page);
        });
    }
    test(`module-access ${viewport.name} administrator six settings, unknown save reread and tab return`, async ({page}, info) => {
        await page.setViewportSize(viewport);
        const ledger = await installThemeCase(page, withSession(themeCases.find(item => item.id === 'admin-services')!));
        const control = await accessRoutes(page, 'admin');
        await page.goto(memberPath('/admin/service-status'));
        await page.getByRole('tab', {name: 'Module Access', exact: true}).click();
        // Each label must fit its hit target; visible neighboring labels cannot overlap.
        const tabs = page.locator('.admin-source-tabs > .ant-tabs-nav .ant-tabs-tab');
        const boxes = await tabs.evaluateAll(elements =>
            elements.map(element => {
                const tab = element.getBoundingClientRect();
                const label = element.querySelector('.ant-tabs-tab-btn')!.getBoundingClientRect();
                return {left: tab.left, right: tab.right, labelLeft: label.left, labelRight: label.right};
            })
        );
        expect(boxes).toHaveLength(4);
        for (const box of boxes) {
            expect(box.labelLeft).toBeGreaterThanOrEqual(box.left - 1);
            expect(box.labelRight).toBeLessThanOrEqual(box.right + 1);
        }
        for (let i = 1; i < boxes.length; i++) expect(boxes[i].labelLeft).toBeGreaterThanOrEqual(boxes[i - 1].labelRight + 4);
        await expect(page.getByText('Token access controls are deferred.')).toBeVisible();
        await expect(page.getByText('Closing access blocks new user requests. Background tasks and notifications continue running.')).toBeVisible();
        for (const item of modules) await expect(page.getByRole('button', {name: `Close ${item.key === 'worm' ? 'Worm' : item.label} access`, exact: true})).toBeVisible();
        control.dropSave = true;
        await page.getByRole('button', {name: 'Close Solana access', exact: true}).click();
        await expect(page.getByRole('button', {name: 'Open Solana access', exact: true})).toBeEnabled();
        await expect(page.getByText(/the change will not be sent again/).filter({visible: true})).toBeVisible();
        expect(control.writes).toEqual([{key: 'solana', state: 2}]);
        await page.evaluate(() => window.scrollTo(0, 0));
        await expect
            .poll(async () => {
                const active = await page.locator('.admin-source-tabs .ant-tabs-tab-active').boundingBox();
                const indicator = await page.locator('.admin-source-tabs .ant-tabs-ink-bar').boundingBox();
                return Boolean(active && indicator && indicator.x >= active.x - 1 && indicator.x + indicator.width <= active.x + active.width + 1);
            })
            .toBe(true);
        await page.screenshot({path: info.outputPath(`settings-unknown-${viewport.name}.png`), fullPage: true, animations: 'disabled'});
        await page.getByRole('tab', {name: /^Services/}).click();
        await expect(page.getByRole('heading', {name: 'Service Status', exact: true})).toBeVisible();
        await page.getByRole('tab', {name: 'Module Access', exact: true}).click();
        await expect(page.getByRole('button', {name: 'Open Solana access', exact: true})).toBeEnabled();
        await page.getByRole('button', {name: 'Open Solana access', exact: true}).click();
        await expect(page.getByRole('button', {name: 'Close Solana access', exact: true})).toBeEnabled();
        expect(control.writes).toEqual([
            {key: 'solana', state: 2},
            {key: 'solana', state: 1}
        ]);
        // A save started in the previous tab lifecycle cannot overwrite the
        // authoritative setting read after returning, or leave the row disabled.
        control.holdSave = true;
        await page.getByRole('button', {name: 'Close Solana access', exact: true}).click();
        await expect.poll(() => control.releaseSave !== undefined).toBe(true);
        await page.getByRole('tab', {name: /^Services/}).click();
        control.rows.find(row => row.module_key === 'solana')!.state = 1;
        await page.getByRole('tab', {name: 'Module Access', exact: true}).click();
        await expect(page.getByRole('button', {name: 'Close Solana access', exact: true})).toBeEnabled();
        control.releaseSave!();
        await page.getByRole('button', {name: 'Refresh access settings', exact: true}).click();
        await expect(page.getByRole('button', {name: 'Close Solana access', exact: true})).toBeEnabled();
        expect(control.writes).toHaveLength(3);
        assertThemeLedger(ledger);
    });
}

test('module-access offline and hidden recovery require a fresh authoritative read', async ({page}) => {
    const scenario = withSession(themeCases.find(item => item.id === 'foundations-solana')!);
    const ledger = await installThemeCase(page, scenario);
    const control = await accessRoutes(page, 'member');
    await page.goto(memberPath(scenario.route));
    await expect(page.getByRole('heading', {name: 'Solana', exact: true})).toBeVisible();
    control.holdRead = true;
    await page.evaluate(() => {
        Object.defineProperty(navigator, 'onLine', {configurable: true, value: false});
        window.dispatchEvent(new Event('offline'));
    });
    await expect(page.getByText('Module access could not be confirmed', {exact: true})).toBeVisible();
    await page.evaluate(() => {
        Object.defineProperty(navigator, 'onLine', {configurable: true, value: true});
        window.dispatchEvent(new Event('online'));
    });
    await expect(page.getByRole('heading', {name: 'Solana', exact: true})).toHaveCount(0);
    control.holdRead = false;
    control.rows.find(row => row.module_key === 'solana')!.state = 2;
    await page.evaluate(() => window.dispatchEvent(new Event('pageshow')));
    await expect(page.getByText('This module is not open yet', {exact: true})).toBeVisible();
    await page.evaluate(() => {
        Object.defineProperty(document, 'visibilityState', {configurable: true, value: 'hidden'});
        document.dispatchEvent(new Event('visibilitychange'));
    });
    await expect(page.getByText('Module access could not be confirmed', {exact: true})).toBeVisible();
    control.rows.find(row => row.module_key === 'solana')!.state = 1;
    await page.evaluate(() => {
        Object.defineProperty(document, 'visibilityState', {configurable: true, value: 'visible'});
        document.dispatchEvent(new Event('visibilitychange'));
    });
    await expect(page.getByRole('heading', {name: 'Solana', exact: true})).toBeVisible();
    assertThemeLedger(ledger);
});
