import {expect, test} from '@playwright/test';
import {themeCases} from './cases';
import {assertThemeLedger, installThemeCase} from './routes';

const reads = [
    ['member-security', '/api/v1/account/security/tokens', 'No API keys', 'Create API key'],
    ['admin-accounts', '/api/v1/account', 'No accounts match these filters', ''],
    ['admin-gateways', '/api/v1/etherscan-gateway-statuses', 'No Etherscan gateway IPs configured', ''],
    ['admin-services', '/api/v1/service-statuses', 'No service health records', ''],
    ['admin-notifications', '/api/v1/admin/system-notification-deliveries', 'No system notification deliveries', ''],
    ['foundations-wallets', '/api/v1/wallets', 'No wallets match these filters.', ''],
    ['foundations-member-rounds', '/api/v1/profit-sharing/rounds', 'No profit-sharing rounds are available', ''],
    ['foundations-admin-rounds', '/api/v1/profit-sharing/rounds', 'No profit-sharing rounds have been created', 'New round']
];
for (const [id, path, emptyText, independentAction] of reads) {
    test(`theme:read-states ${id} separates pending, first error, successful empty and stale data`, async ({page}, info) => {
        await page.setViewportSize({width: 390, height: 844});
        const scenario = structuredClone(themeCases.find(item => item.id === id)!);
        const reply = scenario.replies.find(item => item.method === 'GET' && item.path === path)!;
        const populated = structuredClone(reply.json);
        reply.status = 503;
        reply.json = {message: 'Controlled read unavailable'};
        reply.delayMs = 1200;
        const ledger = await installThemeCase(page, scenario);
        await page.goto(`${process.env.ATHENA_UI_E2E_PATH_PREFIX || ''}${scenario.route}`);
        await expect(page.getByRole('heading', {name: scenario.heading, exact: true, level: 1})).toBeVisible();
        await expect(page.getByText(emptyText, {exact: true})).toBeHidden();
        await expect(page.getByText('Controlled read unavailable', {exact: true})).toBeVisible();
        await expect(page.getByText(emptyText, {exact: true})).toBeHidden();
        if (id === 'admin-services') await expect(page.getByText('0 services', {exact: true})).toBeHidden();
        if (id === 'admin-accounts') await expect(page.getByText('0 registered identities', {exact: true})).toBeHidden();
        if (independentAction) await expect(page.getByRole('button', {name: independentAction, exact: true})).toBeEnabled();
        await info.attach(`${id}-first-error`, {body: await page.screenshot({fullPage: true, animations: 'disabled'}), contentType: 'image/png'});
        reply.delayMs = 0;
        reply.status = 200;
        reply.json = path.endsWith('/rounds') ? {rounds: []} : {items: [], totalSize: 0, total: 0};
        await page.getByRole('button', {name: 'Retry', exact: true}).first().click();
        await expect(page.getByText('Controlled read unavailable', {exact: true})).toBeHidden();
        await expect(page.getByText(emptyText, {exact: true})).toBeVisible();
        reply.json = populated;
        await page.reload();
        await expect(page.getByRole('heading', {name: scenario.heading, exact: true, level: 1})).toBeVisible();
        await expect(page.getByText(emptyText, {exact: true})).toBeHidden();
        // A settled successful read must survive a later failure, with an explicit stale indication.
        await expect.poll(() => ledger.requests.filter(item => item.path === path).length).toBeGreaterThanOrEqual(3);
        await page.waitForTimeout(200);
        reply.status = 503;
        reply.json = {message: 'Controlled read unavailable'};
        if (id === 'member-security') await page.getByRole('button', {name: 'Refresh API keys', exact: true}).click();
        else await page.getByRole('button', {name: 'Refresh data', exact: true}).click();
        await expect(page.getByText('Controlled read unavailable', {exact: true})).toBeVisible();
        await expect(page.getByText(/Stale .*last successful read/).filter({visible: true})).toBeVisible();
        await expect(page.getByText(emptyText, {exact: true})).toBeHidden();
        assertThemeLedger(ledger);
    });
}
