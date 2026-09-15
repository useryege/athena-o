import {recordAxeReview} from './theme-refactor/axe-review';
import {expect, test} from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import fs from 'node:fs';
import {themeCases} from './theme-refactor/cases';
import {assertThemeLayout, assertThemeLedger, installThemeCase, openThemeCase} from './theme-refactor/routes';

const tags = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'];

for (const scenario of themeCases.filter(item =>
    [
        'worm-assets',
        'worm-assets-readonly',
        'worm-assets-partial',
        'worm-assets-batch-paused',
        'worm-assets-batch-unknown',
        'worm-combinations',
        'worm-combinations-edit',
        'worm-combinations-new',
        'worm-combinations-invalid',
        'markets-hot',
        'markets-realtime',
        'markets-movers',
        'markets-live',
        'markets-history',
        'markets-history-writable',
        'markets-corners',
        'markets-proposals',
        'markets-disputes',
        'foundations-wallets',
        'foundations-solana',
        'foundations-member-rounds',
        'foundations-member-collecting',
        'foundations-member-voting',
        'foundations-admin-rounds',
        'foundations-admin-collecting',
        'foundations-admin-draft',
        'admin-accounts',
        'admin-services',
        'admin-gateways',
        'admin-notifications',
        'admin-notification-detail',
        'member-login',
        'admin-login',
        'member-register',
        'admin-register',
        'member-profile',
        'admin-profile',
        'member-access',
        'admin-access',
        'member-security',
        'member-notifications',
        'member-help',
        'admin-help',
        'member-bootstrap-error',
        'member-pending',
        'admin-forbidden'
    ].includes(item.id)
)) {
    test(`theme:a11y ${scenario.id}`, async ({page}, info) => {
        await page.setViewportSize({width: 390, height: 844});
        const ledger = await openThemeCase(page, scenario.id);
        await assertThemeLayout(page);
        await expect(page.locator('.ant-form-show-help-item-appear-active, .ant-form-show-help-item-enter-active')).toHaveCount(0);
        const results = await new AxeBuilder({page}).withTags(tags).analyze();

        const rawResult = info.outputPath('axe-results.json');
        fs.writeFileSync(rawResult, JSON.stringify(results, null, 2));
        await recordAxeReview(page, results, info);
        await info.attach('axe-results.json', {path: rawResult, contentType: 'application/json'});
        expect(
            results.violations.map(violation => ({id: violation.id, impact: violation.impact, nodes: violation.nodes.map(node => node.target)})),
            '[ATHENA_A11Y_VIOLATION] WCAG 2.0/2.1 A and AA violations; see attached original axe results'
        ).toEqual([]);
        assertThemeLedger(ledger);
    });
}

for (const state of [
    {id: 'markets-proposals', tab: 'Evidence'},
    {id: 'admin-accounts', tab: 'Access'},
    {id: 'admin-accounts', tab: 'Profile'},
    {id: 'admin-services', tab: 'Notifications'},
    {id: 'admin-services', tab: 'Trader Sync'},
    {id: 'admin-gateways', tab: 'Live Probe'},
    {id: 'admin-notifications', tab: 'Test notification'}
]) {
    test(`theme:a11y ${state.id} ${state.tab}`, async ({page}, info) => {
        await page.setViewportSize({width: 390, height: 844});
        const ledger = await openThemeCase(page, state.id);
        if (state.id === 'admin-accounts') await page.locator('.admin-accounts-mobile-card').first().click();
        if (state.id === 'markets-proposals') await page.getByRole('button', {name: 'View evidence', exact: true}).filter({visible: true}).first().click();
        else if (state.id === 'admin-notifications') await page.getByRole('button', {name: 'Test Notification', exact: true}).click();
        else await page.getByRole('tab', {name: new RegExp(`^${state.tab}`)}).click();
        if (state.id === 'markets-proposals') await page.evaluate(() => window.scrollTo(0, 0));
        if (state.id === 'admin-notifications') await expect(page.getByRole('dialog')).not.toHaveClass(/(?:^|\s)ant-zoom-(?:appear|enter)(?:-active)?(?:\s|$)/);
        await assertThemeLayout(page);
        await expect(page.locator('.ant-form-show-help-item-appear-active, .ant-form-show-help-item-enter-active')).toHaveCount(0);
        const results = await new AxeBuilder({page}).withTags(tags).analyze();

        const rawResult = info.outputPath('axe-results.json');
        fs.writeFileSync(rawResult, JSON.stringify(results, null, 2));
        await recordAxeReview(page, results, info);
        await info.attach('axe-results.json', {path: rawResult, contentType: 'application/json'});
        expect(
            results.violations.map(v => ({id: v.id, nodes: v.nodes.map(n => n.target)})),
            '[ATHENA_A11Y_VIOLATION] See original axe results'
        ).toEqual([]);
        await page.mouse.move(0, 0);
        await page.screenshot({path: info.outputPath('admin-auxiliary.png'), fullPage: true, animations: 'disabled'});
        assertThemeLedger(ledger);
    });
}

for (const state of ['selection', 'cashout', 'leave']) {
    test(`theme:a11y worm-dialog ${state}`, async ({page}, info) => {
        await page.setViewportSize({width: 390, height: 844});
        const ledger = await openThemeCase(page, state === 'selection' ? 'worm-assets-twenty' : state === 'cashout' ? 'worm-assets-cashout' : 'worm-combinations-edit');
        if (state === 'selection') await page.getByRole('button', {name: /Manage selection/}).click();
        else if (state === 'cashout')
            await page
                .getByRole('button', {name: /^Cash out YES position/})
                .first()
                .click();
        else {
            await page.getByRole('textbox', {name: 'Combination name', exact: false}).fill('Unsaved draft');
            await page.getByRole('button', {name: /Saved combinations$/}).click();
        }
        await expect(page.getByRole('dialog')).toBeVisible();
        await expect(page.getByRole('dialog')).not.toHaveClass(/(?:^|\s)ant-zoom-(?:appear|enter)(?:-active)?(?:\s|$)/);
        if (state === 'cashout') {
            await page.getByRole('region', {name: 'Cash out position details'}).focus();
            await expect(page.getByRole('region', {name: 'Cash out position details'})).toBeFocused();
            await page.keyboard.press('End');
            await expect.poll(() => page.locator('.ant-modal-confirm-body').evaluate(element => element.scrollTop)).toBeGreaterThan(0);
        }
        await expect(page.locator('.ant-form-show-help-item-appear-active, .ant-form-show-help-item-enter-active')).toHaveCount(0);
        const results = await new AxeBuilder({page}).withTags(tags).analyze();

        const rawResult = info.outputPath('axe-results.json');
        fs.writeFileSync(rawResult, JSON.stringify(results, null, 2));
        await recordAxeReview(page, results, info);
        await info.attach('axe-results.json', {path: rawResult, contentType: 'application/json'});
        expect(
            results.violations.map(v => ({id: v.id, nodes: v.nodes.map(n => n.target)})),
            '[ATHENA_A11Y_VIOLATION] See original axe results'
        ).toEqual([]);
        assertThemeLedger(ledger);
    });
}

for (const id of ['worm-preview', 'worm-executions', 'worm-execution-detail', 'worm-execution-unknown', 'worm-execution-authorized']) {
    test(`theme:worm-executions a11y ${id}`, async ({page}, info) => {
        await page.setViewportSize({width: 390, height: 844});
        await page.clock.setFixedTime(new Date('2026-09-14T08:00:30Z'));
        const ledger = await openThemeCase(page, id);
        await page.evaluate(() => document.fonts.ready);
        await assertThemeLayout(page);
        await expect(page.locator('.ant-form-show-help-item-appear-active, .ant-form-show-help-item-enter-active')).toHaveCount(0);
        const results = await new AxeBuilder({page}).withTags(tags).analyze();

        fs.writeFileSync(info.outputPath('axe-results.json'), JSON.stringify(results, null, 2));

        await recordAxeReview(page, results, info);
        await info.attach('axe-results.json', {path: info.outputPath('axe-results.json'), contentType: 'application/json'});
        expect(
            results.violations.map(v => ({id: v.id, nodes: v.nodes.map(n => n.target)})),
            '[ATHENA_A11Y_VIOLATION] See original axe results'
        ).toEqual([]);
        assertThemeLedger(ledger);
    });
}
for (const action of ['Authorize', 'Terminate']) {
    test(`theme:worm-executions a11y ${action} dialog`, async ({page}, info) => {
        await page.setViewportSize({width: 390, height: 844});
        const ledger = await openThemeCase(page, 'worm-execution-detail');
        await page.getByRole('button', {name: action, exact: true}).click();
        await expect(page.getByRole('dialog')).toBeVisible();
        await page.waitForTimeout(500);
        await expect(page.locator('.ant-form-show-help-item-appear-active, .ant-form-show-help-item-enter-active')).toHaveCount(0);
        const results = await new AxeBuilder({page}).withTags(tags).analyze();

        fs.writeFileSync(info.outputPath('axe-results.json'), JSON.stringify(results, null, 2));

        await recordAxeReview(page, results, info);
        await info.attach('axe-results.json', {path: info.outputPath('axe-results.json'), contentType: 'application/json'});
        expect(
            results.violations.map(v => ({id: v.id, nodes: v.nodes.map(n => n.target)})),
            '[ATHENA_A11Y_VIOLATION] See original axe results'
        ).toEqual([]);
        assertThemeLedger(ledger);
    });
}

for (const [id, failedPath] of [
    ['member-security', '/api/v1/account/security/tokens'],
    ['admin-accounts', '/api/v1/account'],
    ['admin-services', '/api/v1/service-statuses'],
    ['admin-gateways', '/api/v1/etherscan-gateway-statuses'],
    ['admin-notifications', '/api/v1/admin/system-notification-deliveries'],
    ['foundations-wallets', '/api/v1/wallets'],
    ['foundations-member-rounds', '/api/v1/profit-sharing/rounds'],
    ['foundations-admin-rounds', '/api/v1/profit-sharing/rounds']
]) {
    test(`theme:a11y first error ${id}`, async ({page}, info) => {
        const scenario = structuredClone(themeCases.find(item => item.id === id)!);
        Object.assign(scenario.replies.find(item => item.path === failedPath && item.method === 'GET')!, {status: 503, json: {message: 'Controlled read unavailable'}});
        await page.setViewportSize({width: 390, height: 844});
        const ledger = await installThemeCase(page, scenario);
        await page.goto(`${process.env.ATHENA_UI_E2E_PATH_PREFIX || ''}${scenario.route}`);
        await expect(page.getByText('Controlled read unavailable', {exact: true})).toBeVisible();
        await page.evaluate(() => document.fonts.ready);
        await expect(page.locator('.ant-form-show-help-item-appear-active, .ant-form-show-help-item-enter-active')).toHaveCount(0);
        const results = await new AxeBuilder({page}).withTags(tags).analyze();

        fs.writeFileSync(info.outputPath('axe-results.json'), JSON.stringify(results, null, 2));

        await recordAxeReview(page, results, info);
        await info.attach('axe-results.json', {path: info.outputPath('axe-results.json'), contentType: 'application/json'});
        expect(results.violations.map(v => ({id: v.id, nodes: v.nodes.map(n => n.target)}))).toEqual([]);
        assertThemeLedger(ledger);
    });
}

test('theme:a11y profile unsaved confirmation', async ({page}, info) => {
    await page.setViewportSize({width: 390, height: 844});
    const ledger = await openThemeCase(page, 'member-profile');
    await page.getByLabel('Display name', {exact: true}).fill('Unsaved research name');
    await page.getByLabel('Account section').click();
    await page.getByText('Access & session', {exact: true}).filter({visible: true}).click();
    const modal = page.getByRole('dialog');
    await expect(modal).toBeVisible();
    await expect(modal).not.toHaveClass(/ant-zoom-(?:appear|enter)/);
    await expect(page.locator('.ant-form-show-help-item-appear-active, .ant-form-show-help-item-enter-active')).toHaveCount(0);
    const results = await new AxeBuilder({page}).withTags(tags).analyze();

    fs.writeFileSync(info.outputPath('axe-results.json'), JSON.stringify(results, null, 2));

    await recordAxeReview(page, results, info);
    await info.attach('axe-results.json', {path: info.outputPath('axe-results.json'), contentType: 'application/json'});
    expect(results.violations.map(v => ({id: v.id, nodes: v.nodes.map(n => n.target)}))).toEqual([]);
    assertThemeLedger(ledger);
});
