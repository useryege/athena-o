import {expect, test} from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import fs from 'node:fs';
import {themeCases} from './theme-refactor/cases';
import {assertThemeLayout, assertThemeLedger, openThemeCase} from './theme-refactor/routes';

const tags = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'];

for (const scenario of themeCases.filter(item =>
    [
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
        const results = await new AxeBuilder({page}).withTags(tags).analyze();
        const rawResult = info.outputPath('axe-results.json');
        fs.writeFileSync(rawResult, JSON.stringify(results, null, 2));
        await info.attach('axe-results.json', {path: rawResult, contentType: 'application/json'});
        expect(
            results.violations.map(violation => ({id: violation.id, impact: violation.impact, nodes: violation.nodes.map(node => node.target)})),
            '[ATHENA_A11Y_VIOLATION] WCAG 2.0/2.1 A and AA violations; see attached original axe results'
        ).toEqual([]);
        assertThemeLedger(ledger);
    });
}

for (const state of [
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
        if (state.id === 'admin-notifications') await page.getByRole('button', {name: 'Test Notification', exact: true}).click();
        else await page.getByRole('tab', {name: new RegExp(`^${state.tab}`)}).click();
        await assertThemeLayout(page);
        const results = await new AxeBuilder({page}).withTags(tags).analyze();
        const rawResult = info.outputPath('axe-results.json');
        fs.writeFileSync(rawResult, JSON.stringify(results, null, 2));
        await info.attach('axe-results.json', {path: rawResult, contentType: 'application/json'});
        expect(results.violations.map(v => ({id: v.id, nodes: v.nodes.map(n => n.target)}))).toEqual([]);
        await page.mouse.move(0, 0);
        await page.screenshot({path: info.outputPath('admin-auxiliary.png'), fullPage: true, animations: 'disabled'});
        assertThemeLedger(ledger);
    });
}
