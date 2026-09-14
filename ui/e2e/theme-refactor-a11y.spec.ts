import {expect, test} from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import fs from 'node:fs';
import {themeCases} from './theme-refactor/cases';
import {assertThemeLayout, assertThemeLedger, openThemeCase} from './theme-refactor/routes';

const tags = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'];

for (const scenario of themeCases.filter(item => ['member-login', 'admin-login', 'member-register', 'admin-register', 'member-profile', 'admin-profile', 'member-access', 'admin-access', 'member-security', 'member-notifications', 'member-help', 'admin-help', 'member-bootstrap-error', 'member-pending', 'admin-forbidden'].includes(item.id))) {
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
