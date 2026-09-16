import {expect, test} from '@playwright/test';
import {openThemeCase, assertThemeLedger} from './theme-refactor/routes';

for (const oldPath of ['/sports-live', '/sports-history', '/world-cup-corners']) {
    test(`removed module: ${oldPath}`, async ({page}) => {
        const oldRequests: string[] = [];
        page.on('request', request => {
            if (/\/api\/[^?]*(sports-live|sports-history|world-cup-corners)/.test(request.url())) oldRequests.push(request.url());
        });
        const ledger = await openThemeCase(page, 'member-shell');
        const prefix = (process.env.ATHENA_UI_E2E_PATH_PREFIX || '').replace(/\/$/, '');
        await page.goto(`${prefix}${oldPath}`);
        await expect(page.getByText('Page not found', {exact: true})).toBeVisible();
        for (const label of ['Sports', 'Sports Live', 'Sports History', 'World Cup Corners']) {
            await expect(page.getByRole('menuitem', {name: label, exact: true})).toHaveCount(0);
        }
        expect(oldRequests).toEqual([]);
        assertThemeLedger(ledger);
    });
}

test('removed module: administrator grants preserve Worm and Wallet', async ({page}) => {
    const ledger = await openThemeCase(page, 'admin-accounts');
    for (const label of ['Sports Live', 'Sports History', 'World Cup Corners']) {
        await expect(page.getByText(label, {exact: true})).toHaveCount(0);
    }
    for (const label of ['Worm Markets', 'Worm Trading', 'Wallet']) {
        await expect(page.getByText(label, {exact: true}).first()).toBeVisible();
    }
    assertThemeLedger(ledger);
});

for (const id of ['worm-assets', 'worm-combinations', 'worm-combinations-new', 'worm-combinations-edit', 'worm-preview', 'worm-executions', 'worm-execution-detail']) {
    test(`retained Worm route: ${id}`, async ({page}) => {
        const ledger = await openThemeCase(page, id);
        await expect(page.getByText('Page not found', {exact: true})).toHaveCount(0);
        if (id === 'worm-combinations-new') {
            await expect(page.getByRole('heading', {level: 1, name: 'New combination', exact: true})).toBeVisible();
        } else {
            await expect.poll(() => ledger.requests.some(request => request.path.startsWith('/api/v1/worm-trading/'))).toBe(true);
        }
        assertThemeLedger(ledger);
    });
}
