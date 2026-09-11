import {test, expect, type Browser, type BrowserContextOptions} from '@playwright/test';
import fs from 'node:fs';
const persist = async (info: any, name: string, value: any) => {
    const target = info.outputPath(name);
    fs.writeFileSync(target, value.body);
    await info.attach(name, {path: target, contentType: value.contentType});
};
const manifest = JSON.parse(fs.readFileSync(process.env.ATHENA_UI_E2E_MANIFEST!, 'utf8'));
const path = (suffix: string) => manifest.PathPrefix + suffix;
let network: Array<{method?: string; url: string; status?: number; error?: string}> = [];
let browserErrors: string[] = [];
test.beforeEach(() => {
    network = [];
    browserErrors = [];
});
test.afterEach(async ({}, info) => {
    await persist(info, 'browser-network.json', {body: JSON.stringify({network, browserErrors}, null, 2), contentType: 'application/json'});
    expect(browserErrors).toEqual([]);
});
async function realContext(browser: Browser, options: BrowserContextOptions) {
    const context = await browser.newContext(options);
    context.on('page', page => page.on('pageerror', error => browserErrors.push(error.message)));
    context.on('response', response => {
        const url = new URL(response.url());
        if (url.pathname.includes('/api/')) {
            network.push({method: response.request().method(), url: response.url(), status: response.status()});
            if (url.origin !== manifest.BaseURL || !url.pathname.startsWith(path('/api/v1/'))) browserErrors.push('API escaped explicit harness prefix: ' + response.url());
        }
    });
    context.on('requestfailed', request => network.push({method: request.method(), url: request.url(), error: request.failure()?.errorText}));
    return context;
}
test('three isolated real cookie sessions bootstrap member and administrator pages', async ({browser}, testInfo) => {
    for (const [role, state, suffix] of [
        ['member A', manifest.MemberAState, '/trader-sync'],
        ['member B', manifest.MemberBState, '/trader-sync'],
        ['admin', manifest.AdminState, '/admin/trader-sync/subscriptions']
    ]) {
        const context = await realContext(browser, {storageState: state});
        const page = await context.newPage();
        const errors: string[] = [];
        page.on('pageerror', error => errors.push(error.message));
        const response = page.waitForResponse(r => new URL(r.url()).pathname === path('/api/v1/app/bootstrap'));
        await page.goto(path(suffix));
        const bootstrap = await (await response).json();
        expect(bootstrap.session.status).toBe('APP_BOOTSTRAP_SESSION_STATUS_AUTHENTICATED');
        expect(bootstrap.session.user_info.administrator || false).toBe(role === 'admin');
        await expect(page.getByRole('heading', {name: 'Trader Sync', exact: true, level: 1})).toBeVisible();
        expect(errors).toEqual([]);
        await page.screenshot({path: testInfo.outputPath(role.replace(' ', '-') + '.png'), fullPage: true});
        await context.close();
    }
});
test('real add, independent same-wallet notes, source activity, lifecycle and owner boundary', async ({browser, request}, info) => {
    const a = await realContext(browser, {storageState: manifest.MemberAState});
    const b = await realContext(browser, {storageState: manifest.MemberBState});
    try {
        const wallet = '0x000000000000000000000000000000000000002a';
        const ids: string[] = [];
        for (const [context, note] of [
            [a, '😀'.repeat(20)],
            [b, 'member B private']
        ] as const) {
            const page = await context.newPage();
            await page.goto(path('/trader-sync/add'));
            await expect(page.getByRole('heading', {name: 'Add trader', exact: true})).toBeVisible();
            await page.getByLabel('Wallet address or Polymarket profile URL').fill(wallet);
            await page.getByRole('button', {name: 'Resolve trader', exact: true}).click();
            await page.getByLabel('Private note').fill(note);
            const created = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === path('/api/v1/trader-sync/subscriptions'));
            await page.getByRole('button', {name: 'Confirm subscription', exact: true}).click();
            const response = await created;
            expect(response.status()).toBe(200);
            const body = await response.json();
            ids.push(body.subscription.id);
            await expect(page.getByRole('heading', {name: 'Trader Sync', exact: true, level: 1})).toBeVisible();
        }
        expect(ids[0]).not.toBe(ids[1]);
        const page = a.pages()[0];
        await page.goto(path('/trader-sync/subscriptions/' + ids[0]));
        await expect(page.getByText('Monitoring', {exact: true}).first()).toBeVisible({timeout: 20000});
        const foreign = await a.request.get(path('/api/v1/trader-sync/subscriptions/' + ids[1]), {headers: {'X-Athena-Application-Realm': 'member'}});
        expect(foreign.status()).toBe(404);
        await page.goto(path('/trader-sync'));
        await expect(page.getByText('No activity yet.', {exact: false})).toBeVisible();
        const push = await request.post(manifest.ControlURL + '/push', {data: {Wallet: wallet, Count: 1}});
        expect(push.ok()).toBe(true);
        await expect(page.getByRole('button', {name: 'New activity available', exact: true})).toBeVisible({timeout: 25000});
        await expect(page.locator('article[data-activity-id]')).toHaveCount(0);
        await page.getByRole('button', {name: 'New activity available', exact: true}).click();
        await expect(page.locator('article[data-activity-id]')).toHaveCount(1);
        await page.screenshot({path: info.outputPath('live-activity.png'), fullPage: true});
        const sql = await (await request.get(manifest.ControlURL + '/sql')).json();
        expect(sql).toHaveLength(2);
        await persist(info, 'subscription-sql.json', {body: JSON.stringify(sql, null, 2), contentType: 'application/json'});
        await page.goto(path('/trader-sync/subscriptions/' + ids[0]));
        await page.getByRole('button', {name: 'Pause', exact: true}).click();
        await expect(page.getByText('Paused', {exact: true}).first()).toBeVisible();
        await page.getByRole('button', {name: 'Resume', exact: true}).click();
        await expect(page.getByText('Monitoring', {exact: true}).first()).toBeVisible({timeout: 20000});
    } finally {
        await a.close();
        await b.close();
    }
});
test('lost successful create response recovers original ID after server token expiry', async ({browser, request}, info) => {
    const context = await realContext(browser, {storageState: manifest.MemberAState});
    try {
        const page = await context.newPage();
        await page.goto(path('/trader-sync/add'));
        await page.getByLabel('Wallet address or Polymarket profile URL').fill('0x000000000000000000000000000000000000003b');
        await page.getByRole('button', {name: 'Resolve trader', exact: true}).click();
        await expect(page.getByRole('button', {name: 'Confirm subscription', exact: true})).toBeEnabled();
        await request.post(manifest.ControlURL + '/drop-create');
        const sent = page.waitForRequest(r => r.method() === 'POST' && new URL(r.url()).pathname === path('/api/v1/trader-sync/subscriptions'));
        await page.getByRole('button', {name: 'Confirm subscription', exact: true}).click();
        const original = (await sent).postDataJSON();
        await expect(page.getByRole('button', {name: /Recover subscription result$/})).toBeVisible();
        const before = await (await request.get(manifest.ControlURL + '/sql')).json();
        const row = before.find((x: any) => x.wallet === '0x000000000000000000000000000000000000003b');
        expect(row).toBeTruthy();
        expect((await request.post(manifest.ControlURL + '/expire-confirmations')).status()).toBe(204);
        const recovered = page.waitForResponse(r => r.request().method() === 'POST' && new URL(r.url()).pathname === path('/api/v1/trader-sync/subscriptions'));
        await page.getByRole('button', {name: /Recover subscription result$/}).click();
        const response = await recovered;
        expect(response.request().postDataJSON()).toEqual(original);
        expect((await response.json()).subscription.id).toBe(row.id);
        const after = await (await request.get(manifest.ControlURL + '/sql')).json();
        expect(after.length).toBe(before.length);
        await persist(info, 'recovered-persistent-id.json', {
            body: JSON.stringify({originalID: row.id, originalRequestId: original.requestId, count: after.length}),
            contentType: 'application/json'
        });
    } finally {
        await context.close();
    }
});
test('loopback Bot update connects real binding and preserves add draft round trip', async ({browser, request}, info) => {
    const context = await realContext(browser, {storageState: manifest.MemberBState});
    try {
        const page = await context.newPage();
        await page.goto(path('/trader-sync/add'));
        await page.getByLabel('Wallet address or Polymarket profile URL').fill('0x000000000000000000000000000000000000004c');
        const resolved = page.waitForResponse(r => new URL(r.url()).pathname === path('/api/v1/trader-sync/targets:resolve'));
        await page.getByRole('button', {name: 'Resolve trader', exact: true}).click();
        const target = (await (await resolved).json()).target;
        await page.getByLabel('Private note').fill('draft survives');
        await page.getByRole('button', {name: 'Open Notifications', exact: true}).click();
        await expect(page.getByRole('heading', {name: 'Notifications', exact: true, level: 1})).toBeVisible();
        const begin = page.waitForResponse(r => new URL(r.url()).pathname === path('/api/v1/notification-bindings/telegram/attempt') && r.request().method() === 'POST');
        await page.getByRole('button', {name: /Configure$/}).click();
        const body = await (await begin).json();
        const text = body.fallbackCommand || body.fallback_command;
        expect(text).toMatch(/^\/start /);
        expect((await request.post(manifest.ControlURL + '/telegram-update', {data: {Text: text, Chat: 1001}})).status()).toBe(204);
        await expect(page.getByText('Connected', {exact: true}).first()).toBeVisible({timeout: 15000});
        await page.screenshot({path: info.outputPath('real-binding.png'), fullPage: true});
        await page.getByRole('button', {name: 'Return to Trader Sync', exact: true}).click();
        await expect(page.getByLabel('Private note')).toHaveValue('draft survives');
        await page.getByRole('button', {name: 'Open Notifications', exact: true}).click();
        await page.clock.setFixedTime(new Date(Date.parse(target.expiresAt) + 1000));
        await page.getByRole('button', {name: 'Return to Trader Sync', exact: true}).click();
        await expect(page.getByText('Confirmation expired', {exact: true})).toBeVisible();
        await expect(page.getByRole('button', {name: 'Confirm subscription', exact: true})).toBeDisabled();
        await expect(page.getByLabel('Private note')).toHaveValue('draft survives');
        await persist(info, 'draft-expiry.json', {
            body: JSON.stringify({
                expiresAt: target.expiresAt,
                browserTime: await page.evaluate(() => new Date().toISOString()),
                method: 'Browser Date fixed after original server expiry while away; no elapsed-wall-time claim'
            }),
            contentType: 'application/json'
        });
    } finally {
        await context.close();
    }
});
test('real revocation clears private content and regrant requires manual resume', async ({browser, request}, info) => {
    const context = await realContext(browser, {storageState: manifest.MemberAState});
    try {
        const page = await context.newPage();
        await page.goto(path('/trader-sync'));
        await expect(page.locator('article[data-activity-id]')).toHaveCount(1);
        expect((await request.post(manifest.ControlURL + '/hold-read')).status()).toBe(204);
        await page.evaluate(() => window.dispatchEvent(new Event('focus')));
        await expect.poll(async () => (await (await request.get(manifest.ControlURL + '/hold-read')).json()).ready).toBe(true);
        expect((await request.post(manifest.ControlURL + '/grant', {data: {Owner: manifest.Owners[0], Enabled: false}})).ok()).toBe(true);
        await page.evaluate(() => window.dispatchEvent(new Event('focus')));
        await expect(page.getByText('Access pending', {exact: true}).first()).toBeVisible();
        await expect(page.locator('article[data-activity-id]')).toHaveCount(0);
        expect(await page.locator('body').innerText()).not.toContain('😀'.repeat(20));
        await request.post(manifest.ControlURL + '/release-read');
        await expect(page.locator('article[data-activity-id]')).toHaveCount(0);
        expect((await request.post(manifest.ControlURL + '/grant', {data: {Owner: manifest.Owners[0], Enabled: true}})).ok()).toBe(true);
        await page.getByRole('button', {name: 'Refresh permissions', exact: false}).click();
        await page.goto(path('/trader-sync/subscriptions'));
        await expect(page.getByText('Disabled by access change', {exact: true}).first()).toBeVisible();
        const sql = await (await request.get(manifest.ControlURL + '/sql')).json();
        expect(sql.filter((x: any) => x.accountId === manifest.Owners[0]).every((x: any) => x.desiredState === 'permission_disabled')).toBe(true);
        await persist(info, 'revoked-subscriptions.json', {body: JSON.stringify(sql), contentType: 'application/json'});
    } finally {
        await context.close();
    }
});
test('loopback Telegram produces failed, unknown, ordinary and summary results through real sources', async ({browser, request}, info) => {
    test.setTimeout(120000);
    const context = await realContext(browser, {storageState: manifest.MemberBState});
    try {
        const admin = await realContext(browser, {storageState: manifest.AdminState});
        try {
            await expect
                .poll(
                    async () => {
                        const response = await admin.request.get(path('/api/v1/admin/notification-runtime/status'), {headers: {'X-Athena-Application-Realm': 'admin'}});
                        expect(response.ok()).toBe(true);
                        return (await response.json()).status;
                    },
                    {timeout: 70000}
                )
                .toBe('running');
        } finally {
            await admin.close();
        }
        const page = await context.newPage();
        const headers = {'X-Athena-Application-Realm': 'member'};
        const wallet = '0x000000000000000000000000000000000000002a';
        const list = async () => {
            const response = await context.request.get(path('/api/v1/trader-sync/activities'), {headers});
            expect(response.ok()).toBe(true);
            return (await response.json()).activities || [];
        };
        await request.post(manifest.ControlURL + '/send-fault', {data: {Fault: 'failed'}});
        await request.post(manifest.ControlURL + '/push', {data: {Wallet: wallet, Count: 1}});
        await expect.poll(async () => (await list()).some((x: any) => x.delivery?.status === 'failed'), {timeout: 25000}).toBe(true);
        await request.post(manifest.ControlURL + '/send-fault', {data: {Fault: 'timeout'}});
        await request.post(manifest.ControlURL + '/push', {data: {Wallet: wallet, Count: 1}});
        await expect.poll(async () => (await list()).some((x: any) => x.delivery?.status === 'unknown'), {timeout: 25000}).toBe(true);
        await request.post(manifest.ControlURL + '/send-fault', {data: {Fault: ''}});
        await request.post(manifest.ControlURL + '/push', {data: {Wallet: wallet, Count: 12}});
        await expect.poll(async () => (await list()).some((x: any) => x.summaryProgress?.batchId), {timeout: 65000}).toBe(true);
        const activities = await list();
        const batch = activities.find((x: any) => x.summaryProgress?.batchId).summaryProgress.batchId;
        const ordinary = activities.find((x: any) => x.delivery?.status === 'sent');
        expect(ordinary).toBeTruthy();
        await page.goto(path('/trader-sync/summaries/' + batch));
        await expect(page.getByRole('heading', {name: 'Summary batch', exact: true, level: 1})).toBeVisible();
        await expect(page.locator('article[data-part-id]').first()).toBeVisible();
        await page.screenshot({path: info.outputPath('real-summary.png'), fullPage: true});
        await persist(info, 'real-notification-results.json', {body: JSON.stringify(activities, null, 2), contentType: 'application/json'});
        const foreign = await realContext(browser, {storageState: manifest.MemberAState});
        try {
            for (const suffix of ['/activities/' + activities[0].id, '/summaries/' + batch]) {
                const response = await foreign.request.get(path('/api/v1/trader-sync' + suffix), {headers});
                expect(response.status()).toBe(404);
            }
        } finally {
            await foreign.close();
        }
    } finally {
        await request.post(manifest.ControlURL + '/send-fault', {data: {Fault: ''}});
        await context.close();
    }
});
test('real note edit, keyboard cancellation and safe administrator summaries', async ({browser, request}, info) => {
    const context = await realContext(browser, {storageState: manifest.MemberBState});
    const admin = await realContext(browser, {storageState: manifest.AdminState});
    try {
        const rows = await (await request.get(manifest.ControlURL + '/sql')).json();
        const subscription = rows.find((x: any) => x.accountId === manifest.Owners[1]);
        const page = await context.newPage();
        await page.goto(path('/trader-sync/subscriptions/' + subscription.id));
        await page.getByLabel('Private wallet note').fill('updated B note');
        await page.getByRole('button', {name: 'Save note', exact: true}).click();
        await expect(page.getByLabel('Private wallet note')).toHaveValue('updated B note');
        const trigger = page.getByRole('button', {name: 'Cancel subscription', exact: true});
        await trigger.focus();
        await page.keyboard.press('Enter');
        const dialog = page.getByRole('dialog', {name: 'Cancel subscription?'});
        await expect(dialog).toBeVisible();
        await page.keyboard.press('Escape');
        await expect(trigger).toBeFocused();
        await trigger.click();
        await dialog.getByRole('button', {name: 'Confirm cancellation', exact: true}).click();
        await expect(page.getByText('Cancelled', {exact: true}).first()).toBeVisible();
        const cancelled = await (await request.get(manifest.ControlURL + '/sql')).json();
        expect(cancelled.find((x: any) => x.id === subscription.id).desiredState).toBe('cancelled');
        const ap = await admin.newPage();
        await ap.goto(path('/admin/trader-sync/subscriptions/' + subscription.id));
        await expect(ap.getByRole('heading', {name: 'Subscription summary', exact: true, level: 1})).toBeVisible();
        expect(await ap.locator('body').innerText()).not.toContain('updated B note');
        await expect(ap.getByRole('link', {name: /View activity/})).toHaveCount(0);
        const headers = {'X-Athena-Application-Realm': 'admin'};
        const denied = await admin.request.get(path('/api/v1/trader-sync/subscriptions/' + subscription.id), {headers});
        expect(denied.status()).toBe(403);
        const summary = await (await admin.request.get(path('/api/v1/admin/trader-sync/subscriptions/' + subscription.id), {headers})).json();
        expect(JSON.stringify(summary)).not.toContain('updated B note');
        const crosscheck = await (await request.get(manifest.ControlURL + '/admin-sql')).json();
        const checked = crosscheck.find((x: any) => x.subscriptionId === subscription.id);
        expect(summary.summary.activityCount).toBe(checked.activityCount);
        expect(summary.summary.associatedDeliveryCounts).toEqual(checked.deliveryCounts);
        const metrics = await (await admin.request.get(path('/api/v1/admin/trader-sync/status'), {headers})).json();
        await persist(info, 'safe-admin-summary.json', {body: JSON.stringify({summary, metrics, cancelled, crosscheck}, null, 2), contentType: 'application/json'});
        await ap.screenshot({path: info.outputPath('real-admin-summary.png'), fullPage: true});
    } finally {
        await context.close();
        await admin.close();
    }
});
test('anonymous deep link retains login return target', async ({page}) => {
    await page.goto(path('/trader-sync/activities/1'));
    await expect(page).toHaveURL(/\/login\?returnTo=/);
    const url = new URL(page.url());
    expect(url.searchParams.get('returnTo')).toBe('/trader-sync/activities/1');
});
test('real history snapshot shows new activity prompt without changing old page membership', async ({browser, request}, info) => {
    test.setTimeout(90000);
    const context = await realContext(browser, {storageState: manifest.MemberAState});
    const b = await realContext(browser, {storageState: manifest.MemberBState});
    try {
        const rows = await (await request.get(manifest.ControlURL + '/sql')).json();
        const sub = rows.find((x: any) => x.accountId === manifest.Owners[0] && x.wallet.endsWith('002a'));
        const page = await context.newPage();
        await page.goto(path('/trader-sync/subscriptions/' + sub.id));
        await page.getByRole('button', {name: 'Resume', exact: true}).click();
        await expect(page.getByText('Monitoring', {exact: true}).first()).toBeVisible({timeout: 20000});
        await page.goto(path('/trader-sync'));
        await expect(page.locator('article[data-activity-id]')).toHaveCount(1);
        expect((await request.post(manifest.ControlURL + '/push', {data: {Wallet: sub.wallet, Count: 50}})).ok()).toBe(true);
        await expect(page.getByRole('button', {name: 'New activity available', exact: true})).toBeVisible({timeout: 25000});
        await page.getByRole('button', {name: 'New activity available', exact: true}).click();
        await expect(page.locator('article[data-activity-id]')).toHaveCount(50);
        const first = await (await context.request.get(path('/api/v1/trader-sync/activities'), {headers: {'X-Athena-Application-Realm': 'member'}})).json();
        expect(first.page.nextCursor).toBeTruthy();
        const denied = await b.request.get(path('/api/v1/trader-sync/activities'), {
            headers: {'X-Athena-Application-Realm': 'member'},
            params: {'page.cursor': first.page.nextCursor}
        });
        const deniedBody = await denied.json();
        await persist(info, 'foreign-cursor.json', {body: JSON.stringify({status: denied.status(), body: deniedBody}, null, 2), contentType: 'application/json'});
        expect(denied.status()).toBe(400);
        expect(deniedBody.message).toBe('invalid cursor or cursor context');
        await page.getByRole('navigation', {name: 'Activity pages'}).getByRole('button', {name: 'Next', exact: true}).click();
        await expect(page.locator('article[data-activity-id]')).toHaveCount(1);
        const previous = await page.locator('article[data-activity-id]').getAttribute('data-activity-id');
        await request.post(manifest.ControlURL + '/push', {data: {Wallet: sub.wallet, Count: 1}});
        await expect(page.getByRole('button', {name: 'New activity available', exact: true})).toBeVisible({timeout: 25000});
        await expect(page.locator('article[data-activity-id]')).toHaveCount(1);
        await expect(page.locator('article[data-activity-id]')).toHaveAttribute('data-activity-id', previous!);
        await page.getByRole('button', {name: 'New activity available', exact: true}).click();
        await expect(page.locator('article[data-activity-id]')).toHaveCount(50);
        await persist(info, 'history-snapshot.json', {
            body: JSON.stringify({snapshot: first.page.snapshot, tailID: previous, stats: await (await request.get(manifest.ControlURL + '/snapshot')).json()}, null, 2),
            contentType: 'application/json'
        });
    } finally {
        await context.close();
        await b.close();
    }
});

test('real permission loss removes an open cancellation modal and its private note', async ({browser, request}, info) => {
    const context = await realContext(browser, {storageState: manifest.MemberAState});
    try {
        const rows = await (await request.get(manifest.ControlURL + '/sql')).json();
        const sub = rows.find((x: any) => x.accountId === manifest.Owners[0] && x.wallet.endsWith('002a'));
        const page = await context.newPage();
        await page.goto(path('/trader-sync/subscriptions/' + sub.id));
        await page.getByRole('button', {name: 'Cancel subscription', exact: true}).click();
        await expect(page.getByRole('dialog', {name: 'Cancel subscription?'})).toBeVisible();
        await request.post(manifest.ControlURL + '/grant', {data: {Owner: manifest.Owners[0], Enabled: false}});
        await page.evaluate(() => window.dispatchEvent(new Event('focus')));
        await expect(page.getByText('Access pending', {exact: true}).first()).toBeVisible();
        await expect(page.getByRole('dialog')).toHaveCount(0);
        expect(await page.locator('body').innerText()).not.toContain('😀'.repeat(20));
        await page.screenshot({path: info.outputPath('modal-revocation.png')});
    } finally {
        await request.post(manifest.ControlURL + '/grant', {data: {Owner: manifest.Owners[0], Enabled: true}});
        await context.close();
    }
});
