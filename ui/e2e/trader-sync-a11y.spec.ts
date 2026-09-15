import {recordAxeReview} from './theme-refactor/axe-review';
import {test, expect, type Page} from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import fs from 'node:fs';
import {assertFixture, installTraderSyncRoutes, memberPath, subscriptionID} from './trader-sync-fixtures';

const tags = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'];
const viewports = [
    {name: 'desktop', width: 1440, height: 900},
    {name: 'mobile', width: 390, height: 844}
] as const;
const themes = ['light', 'dark'] as const;

const states: Array<{name: string; fixture: string; route: string; ready: (page: Page) => Promise<void>}> = [
    {
        name: 'member subscriptions list',
        fixture: 'monitoring',
        route: '/trader-sync/subscriptions',
        ready: async page => {
            await expect(page.getByRole('heading', {name: 'Subscriptions', exact: true, level: 1})).toBeVisible();
            await expect(page.getByText('1 / 10 current subscriptions.').first()).toBeVisible();
        }
    },
    {
        name: 'member add form',
        fixture: 'monitoring',
        route: '/trader-sync/add',
        ready: async page => {
            await expect(page.getByRole('heading', {name: 'Add trader', exact: true, level: 1})).toBeVisible();
            await expect(page.getByLabel('Wallet address or Polymarket profile URL')).toBeVisible();
        }
    },
    {
        name: 'member cancel confirmation',
        fixture: 'monitoring',
        route: `/trader-sync/subscriptions/${subscriptionID}`,
        ready: async page => {
            await expect(page.getByRole('heading', {name: 'Subscription', exact: true, level: 1})).toBeVisible();
            await page.getByRole('button', {name: 'Cancel subscription', exact: true}).click();
            const dialog = page.getByRole('dialog', {name: 'Cancel subscription?'});
            await expect(dialog).toBeVisible();
            await expect(dialog).not.toHaveClass(/(?:^|\s)ant-zoom-(?:appear|enter)(?:-active)?(?:\s|$)/);
        }
    },
    {
        name: 'administrator sync status',
        fixture: 'admin-runtime',
        route: '/admin/service-status',
        ready: async page => {
            await expect(page.getByRole('heading', {name: 'Service Status', exact: true, level: 1})).toBeVisible();
            await page.getByRole('tab', {name: /^Trader Sync/}).click();
            await expect(page.getByText('synthetic_window', {exact: true}).filter({visible: true})).toBeVisible();
        }
    }
];

test.afterEach(async ({page}) => {
    await assertFixture(page);
});

for (const state of states)
    for (const viewport of viewports)
        for (const theme of themes)
            test(`${state.name} · ${viewport.name} · ${theme}`, async ({page}, info) => {
                await page.setViewportSize({width: viewport.width, height: viewport.height});
                await page.emulateMedia({colorScheme: theme});
                await installTraderSyncRoutes(page, state.fixture);
                await page.goto(memberPath(state.route));
                await state.ready(page);
                await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');

                const results = await new AxeBuilder({page}).withTags(tags).analyze();

                const rawResult = info.outputPath('axe-results.json');
                fs.writeFileSync(rawResult, JSON.stringify(results, null, 2));
                await recordAxeReview(page, results, info);
                await info.attach('axe-results.json', {path: rawResult, contentType: 'application/json'});
                expect(
                    results.violations.map(violation => ({id: violation.id, impact: violation.impact, nodes: violation.nodes.map(node => node.target)})),
                    '[ATHENA_A11Y_VIOLATION] WCAG 2.0/2.1 A and AA violations; see attached original axe results'
                ).toEqual([]);
            });
