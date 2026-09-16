import {expect, test, type Page} from '@playwright/test';
import fs from 'node:fs';

const prefix = process.env.ATHENA_UI_E2E_PATH_PREFIX || '';
const mode = process.env.ATHENA_WORM_RETIREMENT_SCENARIO || 'healthy';
if (!['healthy', 'provider-down', 'account-down', 'restored'].includes(mode)) throw new Error(`Unknown retirement scenario: ${mode}`);
const eventID = process.env.ATHENA_WORM_RETIREMENT_EVENT_ID;
if (!eventID) throw new Error('ATHENA_WORM_RETIREMENT_EVENT_ID is required for the real catalog GET');
const baselinePath = process.env.ATHENA_WORM_RETIREMENT_BASELINE;
const target = new URL(process.env.ATHENA_UI_E2E_BASE_URL!);

test.beforeEach(async ({context}) => {
    await context.route('**/*', async route => {
        const request = route.request();
        const url = new URL(request.url());
        if (!['GET', 'HEAD', 'OPTIONS'].includes(request.method()) || url.origin !== target.origin) {
            await route.abort('blockedbyclient');
            throw new Error(`Read-only acceptance blocked ${request.method()} ${url.origin}${url.pathname}`);
        }
        await route.continue();
    });
});

async function read(page: Page, path: string, realm = 'member') {
    return page.evaluate(
        async ({url, realm}) => {
            const response = await fetch(url, {headers: {'X-Athena-Application-Realm': realm}, credentials: 'same-origin'});
            return {status: response.status, body: await response.json()};
        },
        {url: `${prefix}/api/v1${path}`, realm}
    );
}

for (const width of [1440, 390]) {
    test(`@member worm-retirement ${mode} read-only Trading routes ${width}px`, async ({page}, info) => {
        await page.setViewportSize({width, height: 900});
        await page.goto(`${prefix}/account/profile`);
        await expect(page.getByRole('heading', {level: 1})).toBeVisible();
        await expect(page.getByText('Worm Markets', {exact: true})).toHaveCount(0);
        const results: Record<string, any> = {};
        for (const path of [
            '/worm-trading/combinations?page=1&pageSize=20',
            '/worm-trading/executions?page=1&pageSize=20',
            '/worm-trading/wallet-connections?page=1&pageSize=100'
        ]) {
            results[path] = await read(page, path);
            expect(results[path].status, path).toBe(200);
        }
        const catalog = await read(page, `/worm-trading/events/${encodeURIComponent(eventID!)}`);
        if (mode === 'provider-down' || mode === 'account-down') expect(catalog.status).toBe(503);
        else expect(catalog.status).toBe(200);
        {
            const combination = results['/worm-trading/combinations?page=1&pageSize=20'].body.items[0];
            const execution = results['/worm-trading/executions?page=1&pageSize=20'].body.items[0];
            const routes = [
                ['/worm-trading', 'Worm Trading Assets'],
                ['/worm-trading/combinations', 'Worm Trading Combinations'],
                ['/worm-trading/combinations/new', 'New combination'],
                ['/worm-trading/executions', 'Worm Trading Executions']
            ];
            if (combination)
                routes.push(
                    [`/worm-trading/combinations/${combination.id}/edit`, 'Edit combination'],
                    [`/worm-trading/combinations/${combination.id}/execute`, 'Execution preview']
                );
            if (execution) routes.push([`/worm-trading/executions/${execution.id}`, 'Worm Trading Execution']);
            for (const [route] of routes) {
                await page.goto(`${prefix}${route}`);
                await expect(page.getByRole('heading', {level: 1})).toBeVisible();
                await expect(page.getByText('Page not found', {exact: true})).toHaveCount(0);
                await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
            }
            await page.goto(`${prefix}/worm-trading/combinations/new`);
            const eventInput = page.getByLabel('Worm market URL or Event Condition ID', {exact: true});
            await eventInput.fill(eventID!);
            await page.getByRole('button', {name: /Add event$/}).click();
            if (mode === 'provider-down' || mode === 'account-down') {
                await expect(eventInput).toHaveAttribute('aria-invalid', 'true');
                await expect(page.locator('.worm-combination-event-input').getByRole('alert')).toBeVisible();
            } else {
                await expect(page.locator('.worm-combination-event')).toHaveCount(1);
            }
            await page.screenshot({path: info.outputPath(`catalog-${mode}-${width}.png`), fullPage: true});
            await page.goto(`${prefix}/worm-trading/combinations`);
            if (width < 768) await page.getByRole('button', {name: 'Open navigation', exact: true}).click();
            for (const name of ['wallet Assets', 'file-text Combinations', 'history Executions'])
                await expect(page.getByRole('navigation', {name: 'Primary navigation', exact: true}).getByRole('menuitem', {name, exact: true})).toBeVisible();
            await expect(page.getByText('Worm Markets', {exact: true})).toHaveCount(0);
            await info.attach('actual-route-coverage.json', {
                body: JSON.stringify({routes: routes.map(([route]) => route), unavailableHistory: {combination: !combination, execution: !execution}}),
                contentType: 'application/json'
            });
            if (baselinePath) {
                const stable = Object.fromEntries(Object.entries(results).map(([path, result]) => [path, result.body.items]));
                if (mode === 'healthy' && width === 1440) fs.writeFileSync(baselinePath, JSON.stringify(stable));
                else expect(stable).toEqual(JSON.parse(fs.readFileSync(baselinePath, 'utf8')));
            }
        }
        await info.attach('read-results.json', {body: JSON.stringify({results, catalog}), contentType: 'application/json'});
        await page.screenshot({path: info.outputPath(`member-${mode}-${width}.png`), fullPage: true});
        // Profile belongs to the server account path, separate from Trading's owned account-reader proxy.
        await page.goto(`${prefix}/account/profile`);
        await expect(page.getByRole('heading', {level: 1, name: 'Account Center', exact: true})).toBeVisible();
        await expect(page.getByRole('region', {name: 'Profile', exact: true})).toBeVisible();
        expect((await read(page, '/wallets')).status).toBe(200);
    });

    test(`@admin worm-retirement ${mode} read-only access directory ${width}px`, async ({page}, info) => {
        await page.setViewportSize({width, height: 900});
        await page.goto(`${prefix}/admin/accounts`);
        await expect(page.getByRole('heading', {level: 1})).toBeVisible();
        await expect(page.getByText('Worm Markets', {exact: true})).toHaveCount(0);
        const bootstrap = await read(page, '/app/bootstrap', 'admin');
        expect(bootstrap.status).toBe(200);
        expect((bootstrap.body.session.userInfo || bootstrap.body.session.user_info).administrator).toBe(true);
        await info.attach('admin-bootstrap.json', {body: JSON.stringify(bootstrap), contentType: 'application/json'});
        await page.screenshot({path: info.outputPath(`admin-${mode}-${width}.png`), fullPage: true});
    });
}
