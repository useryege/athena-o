import {expect, test, type Page} from '@playwright/test';
import {themeCases} from './cases';
import {assertThemeLayout, assertThemeLedger, installThemeCase} from './routes';
import type {ThemeCase, ThemeLedger} from './contracts';

export const executionScenario = (id: string) => structuredClone(themeCases.find(item => item.id === id)!) as ThemeCase;
const prefix = process.env.ATHENA_UI_E2E_PATH_PREFIX || '';
export const executionOpen = async (page: Page, data: ThemeCase) => {
    await page.clock.install({time: new Date('2026-09-14T08:00:30Z')});
    await page.clock.pauseAt(new Date('2026-09-14T08:00:30Z'));
    const ledger = await installThemeCase(page, data);
    await page.goto(`${prefix}${data.route}`);
    await expect(page.getByRole('heading', {name: data.heading, level: 1, exact: true})).toBeVisible();
    await page.evaluate(() => document.fonts.ready);
    return ledger;
};
export const executionLedger = (ledger: ThemeLedger, writes: Array<{path: string; body: unknown}> = [], repeatedReads: string[] = []) => {
    const recordedWrites = ledger.requests.filter(r => r.method !== 'GET');
    expect(recordedWrites).toHaveLength(writes.length);
    recordedWrites.forEach((request, i) => {
        expect(request).toEqual({method: 'POST', path: writes[i].path, realm: 'member', query: '', body: writes[i].body});
    });
    for (const request of ledger.requests.filter(r => r.method === 'GET')) {
        expect(request.realm).toBe('member');
        expect(request.body).toBeUndefined();
        expect(request.query).toBe(
            request.path.endsWith('/wallet-connections')
                ? '?page=1&pageSize=100'
                : request.path.endsWith('/steps')
                  ? '?page=1&pageSize=50'
                  : request.path.endsWith('/executions')
                    ? '?page=1&pageSize=20'
                    : request.path.endsWith('/combinations')
                      ? '?page=1&pageSize=1'
                      : ''
        );
    }
    expect(ledger.requests.filter(r => r.path === '/api/v1/app/bootstrap')).toHaveLength(1);
    // Clock is paused. Each source reads once unless explicitly refreshed or changed by a command.
    for (const path of new Set(ledger.requests.filter(r => r.method === 'GET').map(r => r.path)))
        expect(
            ledger.requests.filter(r => r.method === 'GET' && r.path === path),
            path
        ).toHaveLength(repeatedReads.includes(path) ? 2 : 1);
    assertThemeLedger(ledger);
};
const runReply = (data: ThemeCase) => data.replies.find(r => /\/executions\/[^/]+$/.test(r.path))!;
const planReply = (data: ThemeCase) => data.replies.find(r => /\/execution-plans\/[^/]+$/.test(r.path))!;
const stepReply = (data: ThemeCase) => data.replies.find(r => r.path.endsWith('/steps'))!;
const post = (data: ThemeCase, path: string, json: unknown, delayMs = 0) => data.replies.push({method: 'POST', path, realm: 'member', status: 200, json, delayMs});
const commandBody = (revision: number) => ({commandId: expect.stringMatching(/^[0-9a-f-]{36}$/), expectedRevision: revision});
const capture = async (page: Page, info: any, name: string) => {
    await page.evaluate(() => document.fonts.ready);
    await page.mouse.move(0, 0);
    await page.evaluate(() => window.scrollTo(0, 0));
    await assertThemeLayout(page);
    await page.screenshot({path: info.outputPath(`${name}.png`), fullPage: true, animations: 'disabled'});
};

for (const viewport of [
    {name: 'desktop', width: 1440, height: 900},
    {name: 'mobile', width: 390, height: 844}
]) {
    for (const id of ['worm-preview', 'worm-executions', 'worm-execution-detail'])
        test(`theme:worm-executions ${id} ${viewport.name} approved page`, async ({page}, info) => {
            await page.setViewportSize(viewport);
            const data = executionScenario(id);
            const ledger = await executionOpen(page, data);
            await expect(page.getByText('September market basket', {exact: true}).filter({visible: true}).first()).toBeVisible();
            if (id === 'worm-preview') {
                await expect(page.getByRole('button', {name: 'Prepare live execution', exact: true})).toBeEnabled();
                await expect(page.getByText('20.040000 USDC', {exact: true})).toBeVisible();
                await page.locator('.worm-preview-wallet-section').first().locator('summary').click();
                await expect(page.locator('.worm-preview-wallet-section').first()).toContainText('Estimate fully filled: No');
                await expect(page.locator('.worm-preview-wallet-section').first()).toContainText('125.000000 → 119.990000');
                await page.locator('.worm-preview-wallet-section').first().locator('summary').click();
                await expect(page.locator('.worm-preview-read-only')).toHaveCSS('border-top-color', 'rgb(37, 42, 48)');
            } else if (id === 'worm-execution-detail') await expect(page.getByRole('button', {name: 'Authorize', exact: true})).toBeVisible();
            if (id === 'worm-execution-detail' && viewport.name === 'mobile') {
                const last = page.locator('.resource-table-compact .worm-step-evidence').last();
                await last.scrollIntoViewIfNeeded();
                const evidence = await last.boundingBox();
                const bar = await page.locator('.worm-execution-sticky-actions').boundingBox();
                expect(evidence!.y + evidence!.height).toBeLessThanOrEqual(bar!.y);
            }
            await capture(page, info, `${id}-${viewport.name}`);
            executionLedger(ledger);
        });
}
for (const id of ['worm-preview-building', 'worm-preview-failed', 'worm-preview-expired', 'worm-preview-changed'])
    test(`theme:worm-executions ${id} never prepares`, async ({page}, info) => {
        const ledger = await executionOpen(page, executionScenario(id));
        await expect(page.getByRole('button', {name: 'Prepare live execution', exact: true})).toBeDisabled();
        if (id.endsWith('building') || id.endsWith('failed')) {
            await expect(page.getByText('20.040000 USDC', {exact: true})).toHaveCount(0);
            await expect(page.getByText('No classified steps are available.', {exact: true})).toHaveCount(0);
        }
        await capture(page, info, id);
        executionLedger(ledger);
    });
for (const state of ['failed', 'pending', 'empty'])
    test(`theme:worm-executions history ${state} is exclusive`, async ({page}) => {
        const ledger = await executionOpen(page, executionScenario('worm-executions-' + state));
        if (state === 'failed') await expect(page.getByText('Execution history is unavailable', {exact: true})).toBeVisible();
        if (state === 'empty') await expect(page.getByText('No execution runs yet', {exact: true})).toBeVisible();
        else {
            await expect(page.locator('.ant-empty:visible')).toHaveCount(0);
            await expect(page.locator('.resource-table-pagination:visible')).toHaveCount(0);
        }
        executionLedger(ledger);
    });
test('theme:worm-executions stale history retains confirmed records and marks stale', async ({page}) => {
    const data = executionScenario('worm-executions');
    const ledger = await executionOpen(page, data);
    await expect(page.getByText('September market basket', {exact: true}).filter({visible: true}).first()).toBeVisible();
    const reply = data.replies[1];
    reply.status = 503;
    reply.json = {error: {message: 'History refresh failed'}};
    await page.getByRole('button', {name: 'Refresh data', exact: true}).click();
    await expect(page.getByText('Execution history is stale', {exact: true})).toBeVisible();
    await expect(page.getByText('September market basket', {exact: true}).filter({visible: true}).first()).toBeVisible();
    await expect(page.locator('.ant-empty:visible')).toHaveCount(0);
    executionLedger(ledger, [], ['/api/v1/worm-trading/executions']);
});
test('theme:worm-executions changed current combination blocks Prepare even before plan usability refresh', async ({page}) => {
    const data = executionScenario('worm-preview');
    (data.replies.find(r => r.path.includes('/combinations/'))!.json as any).revision = 8;
    const ledger = await executionOpen(page, data);
    await expect(page.getByRole('button', {name: 'Prepare live execution', exact: true})).toBeDisabled();
    executionLedger(ledger);
});
test('theme:worm-executions empty and disconnected subset never implicitly selects or executes', async ({page}) => {
    const data = executionScenario('worm-preview-builder');
    const con = data.replies.find(r => r.path.endsWith('/wallet-connections'))!.json as any;
    con.items[1].connection.state = 'NOT_CONNECTED';
    con.items[1].pendingState = 'CONNECT_PENDING';
    const ledger = await executionOpen(page, data);
    await page.getByRole('button', {name: 'Choose Wallets', exact: true}).click();
    await expect(page.getByRole('button', {name: 'Continue to checks', exact: true})).toBeDisabled();
    await expect(page.getByRole('checkbox', {name: /Orion/})).toHaveCount(0);
    await expect(page.getByText('1 selected Wallet is unavailable', {exact: true})).toBeVisible();
    await page.getByRole('checkbox', {name: /Atlas/}).check();
    await expect(page.getByRole('button', {name: 'Continue to checks', exact: true})).toBeEnabled();
    await page.getByRole('button', {name: 'Clear', exact: true}).click();
    await expect(page.getByRole('button', {name: 'Continue to checks', exact: true})).toBeDisabled();
    executionLedger(ledger);
});
test('theme:worm-executions Prepare freezes only and authorization remains separate', async ({page}, info) => {
    const data = executionScenario('worm-preview');
    const detail = executionScenario('worm-execution-detail');
    data.replies.push(...detail.replies.slice(1));
    const run = runReply(detail).json as any;
    post(data, '/api/v1/worm-trading/executions', run, 300);
    const ledger = await executionOpen(page, data);
    const button = page.getByRole('button', {name: 'Prepare live execution', exact: true});
    await button.dblclick();
    await expect(page.getByRole('button', {name: 'Authorize', exact: true})).toBeVisible();
    await expect(page.getByRole('button', {name: 'Start', exact: true})).toHaveCount(0);
    executionLedger(ledger, [{path: '/api/v1/worm-trading/executions', body: {...commandBody(7), planId: run.planId}}]);
    await capture(page, info, 'prepared-only');
});
for (const state of ['running', 'pausing', 'terminating', 'unknown', 'completed'])
    test(`theme:worm-executions ${state} authoritative state`, async ({page}, info) => {
        await page.setViewportSize({width: 390, height: 844});
        const data = executionScenario('worm-execution-' + state);
        const ledger = await executionOpen(page, data);
        if (state === 'unknown') {
            await expect(page.getByRole('button', {name: 'Check authoritative status', exact: true})).toBeVisible();
            expect(await page.locator('.worm-execution-unknown-alert .ant-alert-section').evaluate(el => el.getBoundingClientRect().width)).toBeGreaterThan(200);
            await expect(page.getByRole('button', {name: /^(Start|Continue|Authorize)$/})).toHaveCount(0);
        }
        if (state === 'pausing' || state === 'terminating') {
            await expect(page.getByText('Awaiting Completion', {exact: true}).filter({visible: true}).first()).toBeVisible();
            await expect(page.getByRole('button', {name: /^(Start|Continue)$/})).toHaveCount(0);
        }
        if (state === 'completed') {
            await page.locator('.resource-table-compact .worm-step-evidence').first().locator('summary').click();
            await expect(page.getByText('PositionEvidenceFullIdentifier11111111111111111111', {exact: true}).filter({visible: true}).first()).toBeVisible();
            const step = (stepReply(data).json as any).items[0];
            expect(step.completionPositionCreatedAt).toBeLessThanOrEqual(step.completedAt);
            expect(step.completedAt).toBeLessThanOrEqual(step.updatedAt);
            expect(step.updatedAt).toBeLessThanOrEqual((runReply(data).json as any).updatedAt);
            await expect(page.locator('.resource-table-compact .worm-step-evidence').first()).toContainText('2026/9/14 16:00:15');
            await expect(page.locator('.resource-table-compact .worm-step-evidence').first()).toContainText('2026/9/14 16:00:20');
            await expect(page.getByRole('link', {name: /preview/i})).toHaveCount(0);
        }
        await capture(page, info, state);
        executionLedger(ledger);
    });
for (const state of ['authorize', 'terminate'])
    test(`theme:worm-executions ${state} confirmation focus scroll and explicit control`, async ({page}, info) => {
        await page.setViewportSize({width: 390, height: 844});
        const data = executionScenario('worm-execution-detail');
        (data.replies[0].json as any).session.userInfo.identity.provider = 'ACCOUNT_IDENTITY_PROVIDER_DEVELOPMENT';
        const current = runReply(data).json as any;
        const next = structuredClone(current);
        next.revision = 2;
        if (state === 'authorize') {
            next.state = 'AUTHORIZED';
            next.allowedActions = ['START', 'TERMINATE'];
            next.authorization = {state: 'AUTHORIZED', proofKind: 'DEVELOPMENT', requiresReauthorization: false};
        } else {
            next.state = 'TERMINATE_REQUESTED';
            next.allowedActions = [];
        }
        const path = state === 'authorize' ? `/auth/worm-trading/executions/${current.id}/development` : `/api/v1/worm-trading/executions/${current.id}:terminate`;
        post(data, path, state === 'authorize' ? next : {run: next}, 300);
        const ledger = await executionOpen(page, data);
        const trigger = page.getByRole('button', {name: state === 'authorize' ? 'Authorize' : 'Terminate', exact: true});
        await trigger.click();
        const dialog = page.getByRole('dialog');
        await expect(dialog).toBeVisible();
        await page.waitForTimeout(250);
        await page.keyboard.press('Escape');
        await page.clock.runFor(400);
        await expect(dialog).toBeHidden();
        await expect(trigger).toBeFocused();
        await trigger.click();
        await page.clock.runFor(400);
        await expect(dialog).toHaveCSS('opacity', '1');
        const buttons = dialog.locator('.ant-modal-confirm-btns .ant-btn');
        const boxes = await buttons.evaluateAll(nodes =>
            nodes.map(n => ({x: n.getBoundingClientRect().x, right: n.getBoundingClientRect().right, width: n.getBoundingClientRect().width}))
        );
        expect(Math.abs(boxes[0].x - boxes[1].x)).toBeLessThanOrEqual(1);
        expect(Math.abs(boxes[0].width - boxes[1].width)).toBeLessThanOrEqual(1);
        expect(boxes.every(b => b.x >= 0 && b.right <= 390)).toBe(true);
        const body = dialog.locator('.ant-modal-confirm-content');
        await body.evaluate(el => (el.scrollTop = el.scrollHeight));
        await expect(buttons.last()).toBeInViewport();
        await page.screenshot({path: info.outputPath(`${state}-dialog-mobile.png`), animations: 'disabled'});
        await dialog.getByRole('button', {name: state === 'authorize' ? 'Authorize frozen run' : 'Terminate execution', exact: true}).click();
        if (state === 'authorize') await expect(page.getByRole('button', {name: 'Start', exact: true})).toBeVisible();
        else await expect(page.getByText('Terminate Requested', {exact: true}).filter({visible: true}).first()).toBeVisible();
        executionLedger(ledger, [{path, body: commandBody(1)}], state === 'terminate' ? [`/api/v1/worm-trading/executions/${current.id}`] : []);
    });
for (const id of ['worm-preview', 'worm-executions', 'worm-execution-detail'])
    for (const size of [
        {width: 320, root: 16},
        {width: 720, root: 32}
    ])
        test(`theme:worm-executions ${id} width${size.width} root${size.root} text then boundaries`, async ({page}, info) => {
            await page.setViewportSize({width: size.width, height: 900});
            const ledger = await executionOpen(page, executionScenario(id));
            const probe = page.locator('.app-page__heading > .ant-typography').last();
            const before = parseFloat(await probe.evaluate(el => getComputedStyle(el).fontSize));
            await page.evaluate(root => (document.documentElement.style.fontSize = `${root}px`), size.root);
            expect(parseFloat(await probe.evaluate(el => getComputedStyle(el).fontSize))).toBeCloseTo((before * size.root) / 16, 1);
            await capture(page, info, `${id}-${size.width}-root${size.root}`);
            const overflow = await page.locator('.worm-execution-theme').evaluate(el =>
                [...el.querySelectorAll<HTMLElement>('button,summary,.worm-execution-sticky-actions,.worm-preview-metrics dd')]
                    .filter(n => n.getClientRects().length && getComputedStyle(n).visibility !== 'hidden')
                    .filter(n => n.getBoundingClientRect().right > innerWidth + 1 || n.getBoundingClientRect().left < -1)
                    .map(n => n.textContent)
            );
            expect(overflow).toEqual([]);
            executionLedger(ledger);
        });

for (const action of ['pause', 'terminate', 'reconcile'] as const)
    test(`theme:worm-executions ${action} is single flight and never repeats a provider command`, async ({page}) => {
        const data = executionScenario(action === 'reconcile' ? 'worm-execution-unknown' : 'worm-execution-running');
        const run = runReply(data).json as any;
        const next = structuredClone(run);
        next.revision++;
        if (action !== 'reconcile') {
            next.state = action === 'pause' ? 'PAUSE_REQUESTED' : 'TERMINATE_REQUESTED';
            next.allowedActions = [];
        }
        const path =
            action === 'reconcile' ? `/api/v1/worm-trading/executions/${run.id}/steps/${run.currentStep.id}:reconcile` : `/api/v1/worm-trading/executions/${run.id}:${action}`;
        post(data, path, {run: next}, 300);
        const ledger = await executionOpen(page, data);
        if (action === 'terminate') {
            await page.getByRole('button', {name: 'Terminate', exact: true}).click();
            await page.getByRole('dialog').getByRole('button', {name: 'Terminate execution', exact: true}).click();
        } else await page.getByRole('button', {name: action === 'pause' ? 'Pause' : 'Check authoritative status', exact: true}).dblclick();
        await expect.poll(() => ledger.requests.filter(r => r.method === 'POST').length).toBe(1);
        await page.waitForTimeout(450);
        await expect(
            page
                .getByText(action === 'reconcile' ? 'Outcome Unknown' : 'Awaiting Completion', {exact: true})
                .filter({visible: true})
                .first()
        ).toBeVisible();
        executionLedger(ledger, [{path, body: commandBody(2)}], action === 'reconcile' ? [] : [`/api/v1/worm-trading/executions/${run.id}`]);
    });

test('theme:worm-executions explicit Start begins one driver and an unknown result stops dispatch', async ({page}) => {
    const data = executionScenario('worm-execution-authorized');
    const ready = runReply(data).json as any;
    const unknown = runReply(executionScenario('worm-execution-unknown')).json as any;
    const path = `/api/v1/worm-trading/executions/${ready.id}:start`;
    post(data, path, {run: {...unknown, revision: ready.revision + 1}, coordinatorToken: 'fixture-coordinator'}, 300);
    const ledger = await executionOpen(page, data);
    await expect(page.getByRole('button', {name: 'Start', exact: true})).toBeEnabled();
    executionLedger(ledger);
    await page.getByRole('button', {name: 'Start', exact: true}).dblclick();
    await expect(page.getByRole('button', {name: 'Check authoritative status', exact: true})).toBeVisible();
    executionLedger(ledger, [{path, body: commandBody(ready.revision)}], [`/api/v1/worm-trading/executions/${ready.id}/steps`]);
});

for (const count of [1, 20])
    test(`theme:worm-executions explicit ${count} wallet subset posts frozen revisions and order once`, async ({page}) => {
        const data = executionScenario('worm-preview-builder');
        const selection = data.replies.find(r => r.path.endsWith('/wallet-selection'))!.json as any;
        const inventory = data.replies.find(r => r.path.endsWith('/wallet-connections'))!.json as any;
        const template = inventory.items[0];
        inventory.items = Array.from({length: 20}, (_, i) => ({
            ...structuredClone(template),
            selectionOrdinal: i + 1,
            wallet: {...template.wallet, walletId: 101 + i, address: `WalletFullAddress${String(i + 1).padStart(30, '0')}`, remark: `Wallet ${i + 1}`}
        }));
        inventory.total = 20;
        selection.selectedItems = inventory.items.map((item: any, i: number) => ({ordinal: i + 1, walletId: item.wallet.walletId, address: item.wallet.address}));
        const path = '/api/v1/worm-trading/execution-plans';
        data.replies.push({method: 'POST', path, realm: 'member', status: 409, json: {error: {message: 'WALLET_SELECTION_CHANGED'}}, delayMs: 300});
        const ledger = await executionOpen(page, data);
        await page.getByRole('button', {name: 'Choose Wallets', exact: true}).click();
        await expect(page.getByRole('button', {name: 'Continue to checks', exact: true})).toBeDisabled();
        if (count === 20) await page.getByRole('button', {name: 'Select all eligible', exact: true}).click();
        else await page.getByRole('checkbox', {name: /^Wallet 1 /}).check();
        await expect(page.getByRole('checkbox', {checked: true})).toHaveCount(count);
        await page.getByRole('button', {name: 'Continue to checks', exact: true}).click();
        await page.getByRole('button', {name: 'Build read-only preview', exact: true}).dblclick();
        await expect(page.getByText('Worm wallet selection changed', {exact: true})).toBeVisible();
        executionLedger(
            ledger,
            [
                {
                    path,
                    body: {
                        combinationId: '22000000-0000-4000-8000-000000000001',
                        expectedCombinationRevision: 7,
                        expectedWalletSelectionRevision: 4,
                        walletIds: Array.from({length: count}, (_, i) => 101 + i)
                    }
                }
            ],
            ['/api/v1/worm-trading/wallet-selection', '/api/v1/worm-trading/wallet-connections']
        );
    });

for (const size of [
    {width: 390, root: 16},
    {width: 320, root: 16},
    {width: 720, root: 32}
])
    test(`theme:worm-executions preview control words width${size.width} root${size.root}`, async ({page}, info) => {
        await page.setViewportSize({width: size.width, height: 900});
        const ledger = await executionOpen(page, executionScenario('worm-preview'));
        const button = page.getByRole('button', {name: 'Re-run preview', exact: true});
        const before = await button.evaluate(el => parseFloat(getComputedStyle(el).fontSize));
        await page.evaluate(root => (document.documentElement.style.fontSize = `${root}px`), size.root);
        expect(await button.evaluate(el => parseFloat(getComputedStyle(el).fontSize))).toBeCloseTo((before * size.root) / 16, 1);
        const words = await page
            .locator('.worm-preview-workflow .ant-steps-item-title, .worm-preview-plan-header .ant-btn, .worm-preview-prepare-panel .ant-btn')
            .evaluateAll(nodes =>
                nodes.flatMap(el => {
                    const walker = document.createTreeWalker(el, NodeFilter.SHOW_TEXT);
                    const broken: string[] = [];
                    let n: Node | null;
                    while ((n = walker.nextNode()))
                        for (const match of n.textContent!.matchAll(/\S+/g)) {
                            const range = document.createRange();
                            range.setStart(n, match.index!);
                            range.setEnd(n, match.index! + match[0].length);
                            const lines = new Set([...range.getClientRects()].map(rect => Math.round(rect.y)));
                            if (lines.size > 1) broken.push(match[0]);
                        }
                    return broken;
                })
            );
        await info.attach('control-word-layout.json', {
            body: JSON.stringify({size, words, steps: await page.locator('.worm-preview-workflow').evaluate(el => el.outerHTML)}),
            contentType: 'application/json'
        });
        expect(words).toEqual([]);
        await capture(page, info, 'preview-control-words');
        executionLedger(ledger);
    });

for (const state of ['completed', 'unknown'] as const)
    for (const action of ['pause', 'terminate'] as const)
        test(`theme:worm-executions refreshed permission ${action} after ${state}`, async ({page}) => {
            const data = executionScenario('worm-execution-running');
            const latestData = executionScenario('worm-execution-' + state);
            const latest = structuredClone(runReply(latestData).json) as any;
            latest.revision = 17;
            latest.updatedAt = Math.max(latest.updatedAt, (runReply(data).json as any).updatedAt + 1);
            const sends = state === 'unknown' && action === 'terminate';
            const path = `/api/v1/worm-trading/executions/${latest.id}:${action}`;
            const result = {...latest, revision: 18, state: 'TERMINATE_REQUESTED', allowedActions: []};
            if (sends) post(data, path, {run: result});
            const ledger = await executionOpen(page, data);
            await expect(page.getByRole('button', {name: 'Pause', exact: true})).toBeVisible();
            runReply(data).json = latest;
            stepReply(data).json = stepReply(latestData).json;
            await page.getByRole('button', {name: action === 'pause' ? 'Pause' : 'Terminate', exact: true}).click();
            if (action === 'terminate') await page.getByRole('dialog').getByRole('button', {name: 'Terminate execution', exact: true}).click();
            await expect(
                page
                    .getByText(sends ? 'Terminate Requested' : state === 'completed' ? 'Completed' : 'Reconciliation Required', {exact: true})
                    .filter({visible: true})
                    .first()
            ).toBeVisible();
            await expect(page.getByText(`Could not ${action} execution`, {exact: true})).toHaveCount(0);
            if (state === 'unknown' && action === 'pause') {
                await expect(page.getByRole('button', {name: 'Terminate', exact: true})).toBeEnabled();
                await expect(page.getByRole('button', {name: 'Check authoritative status', exact: true})).toBeEnabled();
            }
            executionLedger(ledger, sends ? [{path, body: commandBody(17)}] : [], [
                `/api/v1/worm-trading/executions/${latest.id}`,
                `/api/v1/worm-trading/executions/${latest.id}/steps`
            ]);
        });
