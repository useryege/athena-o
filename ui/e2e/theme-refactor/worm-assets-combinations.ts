import {expect, test} from '@playwright/test';
import {assertThemeLayout, assertThemeLedger, openThemeCase} from './routes';

for (const id of ['worm-combinations', 'worm-assets-readonly']) {
    test(`theme:worm-combinations ${id} first failure is not successful empty`, async ({page}) => {
        const ledger = await openThemeCase(page, `${id}-failed`);
        await expect(page.getByText(id === 'worm-combinations' ? 'Request failed' : 'Wallet balances are unavailable', {exact: true})).toBeVisible();
        await expect(page.locator('.ant-empty:visible')).toHaveCount(0);
        await expect(page.locator('.resource-table-pagination:visible')).toHaveCount(0);
        assertThemeLedger(ledger);
    });
}

test('theme:worm-combinations mobile shows ordered choices and restores save after invalid selection removal', async ({page}) => {
    await page.setViewportSize({width: 390, height: 844});
    const ledger = await openThemeCase(page, 'worm-combinations-invalid');
    await expect(page.getByText('1 selected market is no longer available', {exact: true})).toBeVisible();
    await expect(page.getByRole('button', {name: 'Save changes', exact: true})).toBeDisabled();
    await page.getByRole('button', {name: 'Remove Will Ethereum exceed $5,000 in September?', exact: true}).click();
    await expect(page.getByRole('button', {name: 'Save changes', exact: true})).toBeEnabled();
    await expect(page.locator('.worm-combination-selection')).toHaveCount(1);
    assertThemeLedger(ledger);
});

for (const viewport of [
    {name: 'desktop', width: 1440, height: 900},
    {name: 'mobile', width: 390, height: 844}
]) {
    for (const id of ['worm-assets', 'worm-combinations', 'worm-combinations-edit', 'worm-combinations-new']) {
        test(`theme:worm-assets ${id} ${viewport.name} approved structure`, async ({page}, info) => {
            await page.setViewportSize(viewport);
            const ledger = await openThemeCase(page, id);
            if (id === 'worm-assets') {
                await expect(page.getByText('3vbqPKB1JYbeuh1Bwdcu1unwhWhmxZ5HYBFUJ7qFNVM1', {exact: true}).first()).toBeVisible();
                await expect(page.getByRole('tab', {name: 'Open positions · 2', exact: true})).toBeVisible();
                await expect(page.getByText('1.284500000', {exact: true})).toBeVisible();
                const access = page.locator('.worm-trading-balance-card__connection .worm-trading-connection').first();
                const state = await access.locator('.worm-trading-connection__state').boundingBox();
                const details = await access.locator('.worm-connection-details').boundingBox();
                expect(details!.y).toBeGreaterThanOrEqual(state!.y + state!.height);
                expect(Math.abs(details!.x - state!.x)).toBeLessThanOrEqual(1);
            } else if (id === 'worm-combinations-edit') {
                await expect(page.getByRole('textbox', {name: 'Combination name', exact: false})).toHaveValue('September market basket');
            } else if (id !== 'worm-combinations-new') {
                await expect(page.getByText('September market basket', {exact: true}).filter({visible: true}).first()).toBeVisible();
            }
            await page.evaluate(() => document.fonts.ready);
            if (id === 'worm-combinations-edit' || id === 'worm-combinations-new') {
                const save = page.locator('.worm-combination-summary__save');
                const size = await save.evaluate(el => ({
                    height: el.getBoundingClientRect().height,
                    width: el.getBoundingClientRect().width,
                    disabled: (el as HTMLButtonElement).disabled
                }));
                await info.attach('primary-action-size.json', {body: JSON.stringify(size), contentType: 'application/json'});
                expect(size.height, 'Save/Create must meet the strict 44px target, including disabled Create').toBeGreaterThanOrEqual(44);
                expect(size.width).toBeGreaterThanOrEqual(44);
            }
            await page.evaluate(() => window.scrollTo(0, 0));
            await assertThemeLayout(page);
            await page.screenshot({path: info.outputPath(`${id}-${viewport.name}.png`), fullPage: true, animations: 'disabled'});
            expect(ledger.requests.every(r => r.method === 'GET')).toBe(true);
            assertThemeLedger(ledger);
        });
    }
}

import {themeCases} from './cases';
import {installThemeCase} from './routes';
import type {ThemeCase, ThemeLedger} from './contracts';
import type {Page} from '@playwright/test';
const scenario = (id: string) => structuredClone(themeCases.find(item => item.id === id)!) as ThemeCase;
const go = async (page: Page, data: ThemeCase) => {
    const ledger = await installThemeCase(page, data);
    await page.goto(`${process.env.ATHENA_UI_E2E_PATH_PREFIX || ''}${data.route}`);
    await expect(page.getByRole('heading', {level: 1, name: data.heading, exact: true})).toBeVisible();
    return ledger;
};
const checkReads = (ledger: ThemeLedger) => {
    for (const item of ledger.requests.filter(r => r.method === 'GET')) {
        expect(item.realm).toBe('member');
        expect(item.body).toBeUndefined();
        expect(item.query).toBe(
            item.path.endsWith('/wallet-connections')
                ? '?page=1&pageSize=100'
                : /\/(combinations|wallet-balances|wallet-activity|items)$/.test(item.path)
                  ? '?page=1&pageSize=20'
                  : ''
        );
    }
    expect(ledger.requests.filter(r => r.path === '/api/v1/app/bootstrap')).toHaveLength(1);
    assertThemeLedger(ledger);
};
const writeReplies = (data: ThemeCase, path: string, method: string, json: unknown, status = 200) => data.replies.push({path, method, realm: 'member', json, status});
const eventReply = (data: ThemeCase) => data.replies.find(r => r.path.includes('/events/'))!;

for (const id of ['worm-assets-readonly', 'worm-assets-zero', 'worm-assets-partial', 'worm-assets-disconnected', 'worm-assets-twenty']) {
    test(`theme:worm-assets ${id} independent source states and no accidental mutation`, async ({page}, info) => {
        await page.setViewportSize({width: 390, height: 844});
        const ledger = await openThemeCase(page, id);
        if (id === 'worm-assets-readonly') {
            await expect(page.getByRole('button', {name: /Manage selection|Cash out/i})).toHaveCount(0);
        } else if (id === 'worm-assets-zero') await expect(page.getByText('No wallets are selected for Worm Trading.', {exact: true})).toBeVisible();
        else if (id === 'worm-assets-partial') {
            await expect(page.getByText('Position streams are unavailable for 3 wallets.', {exact: true})).toBeVisible();
            await expect(page.getByText(/No open positions/)).toHaveCount(0);
            await expect(page.getByText('Unavailable', {exact: true}).first()).toBeVisible();
        } else if (id === 'worm-assets-disconnected') {
            await expect(page.getByText(/unknown connection outcome. Automatic retry is blocked/).first()).toBeVisible();
        } else {
            await page.getByRole('button', {name: /Manage selection/}).click();
            const dialog = page.getByRole('dialog', {name: /Manage Worm Trading wallets/});
            await expect(dialog.getByText('Maximum 20 wallets selected. Deselect one to choose another.', {exact: true})).toBeVisible();
            await expect(dialog.getByRole('checkbox').last()).toBeDisabled();
            await dialog.getByRole('checkbox').first().uncheck();
            await expect(dialog.getByRole('checkbox').last()).toBeEnabled();
            await dialog.getByRole('checkbox').last().check();
            await expect(dialog.getByRole('checkbox').first()).toBeDisabled();
            await page.keyboard.press('Escape');
            await expect(page.getByRole('button', {name: /Manage selection/})).toBeFocused();
        }
        await page.evaluate(() => window.scrollTo(0, 0));
        await assertThemeLayout(page);
        await page.screenshot({path: info.outputPath(`${id}.png`), fullPage: true, animations: 'disabled'});
        expect(ledger.requests.filter(r => r.method !== 'GET')).toHaveLength(0);
        checkReads(ledger);
    });
}

test('theme:worm-assets selected wallet with open positions cannot be removed', async ({page}, info) => {
    await page.setViewportSize({width: 390, height: 844});
    const ledger = await openThemeCase(page, 'worm-assets');
    await page.getByRole('button', {name: /Manage selection/}).click();
    const dialog = page.getByRole('dialog');
    await expect(dialog.getByRole('checkbox').first()).toBeDisabled();
    await expect(dialog.getByRole('checkbox').nth(1)).toBeDisabled();
    await expect(dialog.getByRole('checkbox').nth(2)).toBeEnabled();
    await page.screenshot({path: info.outputPath('selection-mobile.png'), animations: 'disabled'});
    await page.keyboard.press('Escape');
    checkReads(ledger);
});

test('theme:worm-assets exact single Cash Out authorizes directly into queue', async ({page}, info) => {
    const data = scenario('worm-assets-cashout');
    const ledger = await go(page, data);
    await page
        .getByRole('button', {name: /^Cash out YES position/})
        .first()
        .click();
    const dialog = page.getByRole('dialog');
    await expect(dialog.getByText('Full-position market exit', {exact: true})).toBeVisible();
    await page.screenshot({path: info.outputPath('cashout-confirm.png'), animations: 'disabled'});
    await dialog.getByRole('button', {name: 'Continue to authorization', exact: true}).click();
    await expect(page.getByRole('button', {name: /Closing/}).first()).toBeVisible();
    const writes = ledger.requests.filter(r => r.method !== 'GET');
    expect(writes).toHaveLength(2);
    expect(writes[0]).toMatchObject({
        method: 'POST',
        path: '/api/v1/worm-trading/position-cash-outs',
        realm: 'member',
        query: '',
        body: {walletId: 101, positionPubkey: '6jRQNiUuT8TCSCCCgqaimnXxzKRzxVShYKMunN4q1oG5'}
    });
    expect(writes[1]).toMatchObject({
        method: 'POST',
        path: '/auth/worm-trading/position-cash-outs/22000000-0000-4000-8000-000000000010/development',
        query: '',
        realm: 'member',
        body: {expectedRevision: 1}
    });
    expect((writes[0].body as any).commandId).toMatch(/^[0-9a-f-]{36}$/);
    await expect(page.getByRole('button', {name: /^Start$/})).toHaveCount(0);
    checkReads(ledger);
});

test('theme:worm-assets unknown single Close offers only read-only reconciliation without reissuing', async ({page}) => {
    const ledger = await openThemeCase(page, 'worm-assets-cashout-unknown');
    await page
        .getByRole('button', {name: /Check status/})
        .first()
        .click();
    await expect.poll(() => ledger.requests.filter(r => r.method === 'POST').length).toBe(1);
    expect(ledger.requests.filter(r => r.method === 'POST')[0].path).toBe('/api/v1/worm-trading/position-cash-outs/22000000-0000-4000-8000-000000000010:reconcile');
    await expect(page.getByRole('button', {name: /Check status/}).first()).toBeEnabled();
    checkReads(ledger);
});

for (const state of ['frozen', 'paused', 'unknown', 'building']) {
    test(`theme:worm-assets batch ${state} preserves frozen order and USDC gates`, async ({page}, info) => {
        await page.setViewportSize({width: 390, height: 844});
        const ledger = await openThemeCase(page, `worm-assets-batch-${state}`);
        const panel = page.locator('.worm-position-cash-out-batch-panel');
        await expect(panel).toBeVisible();
        if (state === 'building') {
            await expect(panel.getByText('Building authoritative position snapshot', {exact: true})).toBeVisible();
            await expect(panel.getByRole('button', {name: /authorize/})).toHaveCount(0);
        } else {
            const order = panel.locator('.worm-position-cash-out-batch-items li');
            await expect(order).toHaveCount(2);
            await expect(order.nth(0)).toContainText('Atlas');
            await expect(order.nth(1)).toContainText('Orion');
            if (state === 'frozen') {
                await panel.getByRole('button', {name: /Review & authorize/}).click();
                const dialog = page.getByRole('dialog');
                await expect(dialog.getByText('Strict confirmed USDC gate after every Close', {exact: true})).toBeVisible();
                await expect(dialog.getByText(/newer and strictly higher/)).toBeVisible();
                await page.screenshot({path: info.outputPath('batch-frozen-review-mobile.png'), animations: 'disabled'});
                // Do not cancel this durable batch merely to close the evidence dialog.
            } else {
                await expect(panel.getByText(state === 'paused' ? 'Later Cash Outs are paused' : 'Close outcome requires read-only reconciliation', {exact: true})).toBeVisible();
                await panel.getByRole('button', {name: /Check status/}).click();
                await expect.poll(() => ledger.requests.filter(r => r.method === 'POST').length).toBe(1);
                expect(ledger.requests.filter(r => r.method === 'POST')[0].path).toMatch(/:check-status$/);
                await expect(panel.getByText('0/2 positions completed', {exact: true})).toBeVisible();
                await expect(panel.getByRole('button', {name: /Continue/})).toHaveCount(state === 'paused' ? 1 : 0);
            }
        }
        await page.evaluate(() => window.scrollTo(0, 0));
        await page.screenshot({path: info.outputPath(`batch-${state}-mobile.png`), fullPage: true, animations: 'disabled'});
        checkReads(ledger);
    });
}

test('theme:worm-combinations side replacement, ordering and trimmed Unicode name are sent together', async ({page}) => {
    const data = scenario('worm-combinations-edit');
    const detail = data.replies.find(r => r.path.includes('/combinations/'))!;
    writeReplies(data, detail.path, 'PUT', {...(detail.json as any), revision: 8, name: '星际 🌌'});
    const ledger = await go(page, data);
    await page.getByRole('button', {name: 'Will Bitcoin reach $120,000 by September 30?, NO, 31.6¢ last trade', exact: true}).click();
    await expect(page.locator('.worm-combination-selection')).toHaveCount(2);
    await page.getByRole('button', {name: 'Move Will Ethereum exceed $5,000 in September? up', exact: true}).click();
    await page.getByRole('textbox', {name: 'Combination name', exact: false}).fill('  星际 🌌  ');
    await page.getByRole('button', {name: 'Save changes', exact: true}).click();
    await expect(page.getByRole('heading', {name: 'Worm Trading Combinations', exact: true})).toBeVisible();
    const write = ledger.requests.find(r => r.method === 'PUT')!;
    expect(write.body).toEqual({
        name: '星际 🌌',
        expectedRevision: 7,
        items: [
            {eventConditionId: '7YttLkHDoNj9wyDur5VvxRV6cJrPCGF6Da8VtmZGhAaB', marketConditionId: '8QttLkHDoNj9wyDur5VvxRV6cJrPCGF6Da8VtmZGhAaB', side: 'NO'},
            {eventConditionId: '7YttLkHDoNj9wyDur5VvxRV6cJrPCGF6Da8VtmZGhAaB', marketConditionId: '9QttLkHDoNj9wyDur5VvxRV6cJrPCGF6Da8VtmZGhAaB', side: 'NO'}
        ]
    });
    expect(write.query).toBe('');
    checkReads(ledger);
});

test('theme:worm-combinations new selection validates 80 Unicode codepoints and control characters before creation', async ({page}) => {
    const data = scenario('worm-combinations-new');
    const ledger = await go(page, data);
    await page.getByRole('textbox', {name: 'Worm market URL or Event Condition ID', exact: true}).fill('https://evil.invalid/market/x');
    await page.getByRole('button', {name: /Add event$/}).click();
    await expect(page.getByText('Use an HTTPS market URL from worm.wtf.', {exact: true})).toBeVisible();
    await page.getByRole('textbox', {name: 'Worm market URL or Event Condition ID', exact: true}).fill((eventReply(data).json as any).eventConditionId);
    await page.getByRole('button', {name: /Add event$/}).click();
    await page.getByRole('button', {name: 'Will Bitcoin reach $120,000 by September 30?, YES, 68.4¢ last trade', exact: true}).click();
    await page.getByRole('button', {name: 'Create combination', exact: true}).click();
    const dialog = page.getByRole('dialog');
    const name = dialog.getByRole('textbox', {name: 'Combination name', exact: false});
    await name.fill('🌌'.repeat(80));
    await expect(dialog.getByRole('button', {name: 'Create combination', exact: true})).toBeEnabled();
    await name.fill('🌌'.repeat(81));
    await expect(dialog.getByRole('button', {name: 'Create combination', exact: true})).toBeDisabled();
    await name.fill('name\u007f');
    await expect(dialog.getByRole('button', {name: 'Create combination', exact: true})).toBeDisabled();
    expect(ledger.requests.every(r => r.method === 'GET')).toBe(true);
    const saved = scenario('worm-combinations-edit').replies.find(r => r.path.includes('/combinations/'))!.json as any;
    writeReplies(data, '/api/v1/worm-trading/combinations', 'POST', {...saved, name: 'New basket', revision: 1, items: saved.items.slice(0, 1)});
    await name.fill('  New basket  ');
    await dialog.getByRole('button', {name: 'Create combination', exact: true}).click();
    await expect(page.getByRole('heading', {name: 'Worm Trading Combinations', exact: true})).toBeVisible();
    const writes = ledger.requests.filter(r => r.method === 'POST');
    expect(writes).toHaveLength(1);
    expect(writes[0]).toMatchObject({
        realm: 'member',
        path: '/api/v1/worm-trading/combinations',
        query: '',
        body: {
            name: 'New basket',
            items: [{eventConditionId: '7YttLkHDoNj9wyDur5VvxRV6cJrPCGF6Da8VtmZGhAaB', marketConditionId: '9QttLkHDoNj9wyDur5VvxRV6cJrPCGF6Da8VtmZGhAaB', side: 'YES'}]
        }
    });
    checkReads(ledger);
});

test('theme:worm-combinations leaving a changed draft requires explicit discard', async ({page}, info) => {
    await page.setViewportSize({width: 390, height: 844});
    const ledger = await openThemeCase(page, 'worm-combinations-edit');
    await page.getByRole('textbox', {name: 'Combination name', exact: false}).fill('Unsaved private draft');
    await page.getByRole('button', {name: /Saved combinations$/}).click();
    const dialog = page.getByRole('dialog');
    await expect(dialog.getByText('Discard unsaved combination changes?', {exact: true}).filter({visible: true})).toBeVisible();
    await page.screenshot({path: info.outputPath('combination-draft-leave-mobile.png'), animations: 'disabled'});
    await dialog.getByRole('button', {name: 'Cancel', exact: true}).click();
    await expect(page.getByRole('textbox', {name: 'Combination name', exact: false})).toHaveValue('Unsaved private draft');
    await page.getByRole('button', {name: /Saved combinations$/}).click();
    await page.getByRole('button', {name: 'Discard and leave', exact: true}).click();
    await expect(page.getByRole('heading', {name: 'Worm Trading Combinations', exact: true})).toBeVisible();
    checkReads(ledger);
});

for (const id of ['worm-assets', 'worm-combinations', 'worm-combinations-edit', 'worm-combinations-new'])
    for (const width of [320, 720]) {
        test(`theme:worm-assets ${id} ${width}px root 200% preserves text and local geometry`, async ({page}, info) => {
            await page.setViewportSize({width, height: 900});
            const ledger = await openThemeCase(page, id);
            const heading = page.getByRole('heading', {level: 1});
            const initial = await heading.evaluate(el => parseFloat(getComputedStyle(el).fontSize));
            const hasSave = id === 'worm-combinations-edit' || id === 'worm-combinations-new';
            const save = page.locator('.worm-combination-summary__save');
            await page.evaluate(() => document.fonts.ready);
            const initialSave = hasSave ? await save.evaluate(el => ({height: el.getBoundingClientRect().height, font: parseFloat(getComputedStyle(el).fontSize)})) : undefined;
            await page.evaluate(() => (document.documentElement.style.fontSize = '32px'));
            if (initialSave) {
                const enlarged = await save.evaluate(el => ({height: el.getBoundingClientRect().height, font: parseFloat(getComputedStyle(el).fontSize)}));
                await info.attach('primary-action-root32.json', {body: JSON.stringify({initial: initialSave, enlarged}), contentType: 'application/json'});
                expect(enlarged.font).toBeCloseTo(initialSave.font * 2, 1);
                expect(enlarged.height, 'The 2.75rem primary target must grow to at least 88px at root 200%').toBeGreaterThanOrEqual(88);
                expect(enlarged.height).toBeGreaterThanOrEqual(initialSave.height * 2);
            }
            await expect.poll(() => heading.evaluate(el => parseFloat(getComputedStyle(el).fontSize))).toBeCloseTo(initial * 2, 1);
            if (id === 'worm-assets') {
                await expect(page.locator('.worm-trading-wallet__address code').first()).toHaveCSS('font-size', '26px');
                await expect(page.locator('.worm-trading-wallet__address code').first()).toHaveText('3vbqPKB1JYbeuh1Bwdcu1unwhWhmxZ5HYBFUJ7qFNVM1');
            }
            const overflow = await page
                .locator('.worm-combination-outcome-button:visible,.worm-combination-summary:visible,.worm-trading-wallet:visible,.worm-trading-asset:visible')
                .evaluateAll(elements => elements.map(el => ({name: el.className, width: el.clientWidth, scroll: el.scrollWidth})).filter(el => el.scroll > el.width + 1));
            expect(overflow).toEqual([]);
            await page.evaluate(() => window.scrollTo(0, 0));
            await assertThemeLayout(page);
            await page.screenshot({path: info.outputPath(`${id}-${width}-root32.png`), fullPage: true, animations: 'disabled'});
            checkReads(ledger);
        });
    }

test('theme:worm-assets 390px selection keeps actions fixed while its body scrolls', async ({page}, info) => {
    await page.setViewportSize({width: 390, height: 844});
    const ledger = await openThemeCase(page, 'worm-assets-twenty');
    await page.getByRole('button', {name: /Manage selection/}).click();
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    await page.waitForTimeout(500);
    const body = dialog.locator('.worm-wallet-selection-modal__body');
    const footer = dialog.locator('.worm-wallet-selection-modal__footer');
    const before = await footer.boundingBox();
    expect(before!.y + before!.height).toBeLessThanOrEqual(844);
    expect(await body.evaluate(el => el.scrollHeight > el.clientHeight)).toBe(true);
    await body.evaluate(el => (el.scrollTop = el.scrollHeight));
    expect(Math.abs((await footer.boundingBox())!.y - before!.y)).toBeLessThanOrEqual(1);
    const buttons = footer.getByRole('button');
    const a = await buttons.nth(0).boundingBox(),
        b = await buttons.nth(1).boundingBox();
    expect(a!.x).toBeCloseTo(b!.x, 1);
    expect(a!.width).toBeCloseTo(b!.width, 1);
    await page.screenshot({path: info.outputPath('selection-scroll-mobile.png'), animations: 'disabled'});
    await page.keyboard.press('Escape');
    await expect(page.getByRole('button', {name: /Manage selection/})).toBeFocused();
    checkReads(ledger);
});

for (const id of ['worm-combinations', 'worm-assets-readonly'])
    test(`theme:worm-combinations ${id} successful empty and stale refresh remain distinct`, async ({page}) => {
        const data = scenario(`${id}-empty`);
        const ledger = await go(page, data);
        await expect(page.getByText(id === 'worm-combinations' ? 'No saved combinations' : 'No wallets are selected for Worm Trading.', {exact: true})).toBeVisible();
        await expect(page.getByText('Request failed', {exact: true})).toHaveCount(0);
        checkReads(ledger);
    });

test('theme:worm-combinations refresh failure retains the saved list with a stale notice', async ({page}) => {
    const data = scenario('worm-combinations');
    const ledger = await go(page, data);
    await expect(page.getByText('September market basket', {exact: true}).filter({visible: true})).toBeVisible();
    const list = data.replies.find(r => r.path.endsWith('/combinations'))!;
    list.status = 503;
    list.json = {message: 'Controlled refresh unavailable'};
    await page.getByRole('button', {name: /Refresh$/}).click();
    await expect(page.getByText('Saved combinations may be stale', {exact: true})).toBeVisible();
    await expect(page.getByText('September market basket', {exact: true}).filter({visible: true})).toBeVisible();
    await expect(page.locator('.ant-empty:visible')).toHaveCount(0);
    checkReads(ledger);
});

test('theme:worm-assets five-minute credential lease requires explicit authorization and unknown outcome is never retried', async ({page}) => {
    const data = scenario('worm-assets');
    const bootstrap = data.replies[0].json as any;
    bootstrap.session.userInfo.identity.provider = 'ACCOUNT_IDENTITY_PROVIDER_DEVELOPMENT';
    const inv = data.replies.find(r => r.path.endsWith('/wallet-connections'))!.json as any;
    const atlas = inv.items[0];
    atlas.connection = {state: 'NOT_CONNECTED', warningCode: '', connectedAt: 0};
    atlas.pendingState = 'CONNECT_PENDING';
    atlas.needsAttention = true;
    writeReplies(
        data,
        '/api/v1/worm-trading/wallet-connections/101',
        'POST',
        {reason: 'WORM_TRADING_REAUTH_REQUIRED', message: 'Fresh five-minute credential authorization required'},
        403
    );
    writeReplies(data, '/auth/worm-trading/development', 'POST', {expiresAt: Math.floor(Date.now() / 1000) + 601});
    writeReplies(data, '/api/v1/session/userinfo', 'GET', bootstrap.session.userInfo);
    await page.clock.install();
    const ledger = await go(page, data);
    await expect(page.getByText('Authorize Worm wallet connections', {exact: true})).toBeVisible();
    await expect(page.getByText(/permits Worm credential management for five minutes/)).toBeVisible();
    await page.clock.fastForward(301000);
    expect(ledger.requests.filter(r => r.method === 'POST')).toHaveLength(1);
    const connection = data.replies.find(r => r.path.endsWith('/wallet-connections/101'))!;
    connection.status = 200;
    connection.json = {walletId: 101, address: atlas.wallet.address, state: 'RECONNECT_REQUIRED', warningCode: 'CONNECT_OUTCOME_UNKNOWN', connectedAt: 0};
    atlas.connection = {state: 'RECONNECT_REQUIRED', warningCode: 'CONNECT_OUTCOME_UNKNOWN', connectedAt: 0};
    await page.getByRole('button', {name: /Authorize and apply/}).click();
    await expect(page.getByText(/unknown connection outcome|did not confirm whether it created the credential/).first()).toBeVisible();
    await page.clock.fastForward(301000);
    expect(ledger.requests.filter(r => r.method === 'POST' && r.path.endsWith('/wallet-connections/101'))).toHaveLength(1);
    expect(ledger.requests.filter(r => r.path === '/auth/worm-trading/development')).toHaveLength(1);
    checkReads(ledger);
});

test('theme:worm-assets authorizing a frozen batch queues it without a separate Start', async ({page}) => {
    const data = scenario('worm-assets-batch-frozen');
    (data.replies[0].json as any).session.userInfo.identity.provider = 'ACCOUNT_IDENTITY_PROVIDER_DEVELOPMENT';
    const path = '/auth/worm-trading/position-cash-out-batches/22000000-0000-4000-8000-000000000030/development';
    const batch = data.replies.find(r => r.path.endsWith('/active'))!.json as any;
    const queued = {...batch, state: 'QUEUED', revision: 2, allowedActions: ['PAUSE', 'TERMINATE']};
    writeReplies(data, path, 'POST', queued);
    const ledger = await go(page, data);
    await page
        .locator('.worm-position-cash-out-batch-panel')
        .getByRole('button', {name: /Review & authorize/})
        .click();
    const dialog = page.getByRole('dialog');
    await dialog.getByRole('button', {name: 'Continue to one-time authorization', exact: true}).click();
    await expect(page.locator('.worm-position-cash-out-batch-panel').getByText('Queued', {exact: true})).toBeVisible();
    expect(ledger.requests.filter(r => r.method === 'POST')).toHaveLength(1);
    expect(ledger.requests.find(r => r.method === 'POST')).toMatchObject({path, query: '', realm: 'member', body: {expectedRevision: 1}});
    await expect(page.getByRole('button', {name: 'Start', exact: true})).toHaveCount(0);
    checkReads(ledger);
});

test('theme:worm-assets stale balance and activity failures preserve facts with separate warnings', async ({page}) => {
    const data = scenario('worm-assets-readonly');
    const ledger = await go(page, data);
    await expect(page.getByText('248.320000', {exact: true})).toBeVisible();
    for (const reply of data.replies.filter(r => /wallet-balances|wallet-activity/.test(r.path))) {
        reply.status = 503;
        reply.json = {message: 'Controlled refresh unavailable'};
    }
    await page.getByRole('button', {name: 'Refresh data', exact: true}).click();
    await expect(page.getByText('Could not refresh balances', {exact: true})).toBeVisible();
    await expect(page.getByText('Could not refresh Worm activity', {exact: true})).toBeVisible();
    await expect(page.getByText('248.320000', {exact: true})).toBeVisible();
    await expect(page.getByText('No open positions for the connected wallets.', {exact: true})).toHaveCount(0);
    checkReads(ledger);
});

test('theme:worm-combinations first pending response does not imply zero combinations', async ({page}) => {
    const data = scenario('worm-combinations');
    data.replies.find(r => r.path.endsWith('/combinations'))!.delayMs = 1500;
    const ledger = await go(page, data);
    await expect(page.getByText('Loading saved combinations…', {exact: true})).toBeVisible();
    await expect(page.locator('.ant-empty:visible')).toHaveCount(0);
    await expect(page.getByText('September market basket', {exact: true}).filter({visible: true})).toBeVisible();
    checkReads(ledger);
});

test('theme:worm-combinations normal NO selection uses the same selection semantics as YES', async ({page}) => {
    const ledger = await openThemeCase(page, 'worm-combinations-edit');
    const yes = page.getByRole('button', {name: 'Will Bitcoin reach $120,000 by September 30?, YES, 68.4¢ last trade', exact: true});
    const no = page.getByRole('button', {name: 'Will Ethereum exceed $5,000 in September?, NO, 58.2¢ last trade', exact: true});
    await page.mouse.move(0, 0);
    for (const item of [yes, no]) {
        await expect(item).toHaveAttribute('aria-pressed', 'true');
        await expect(item).toHaveCSS('border-top-color', 'rgb(0, 255, 167)');
        await expect(item).toHaveCSS('color', 'rgb(0, 255, 167)');
    }
    const unselected = page.getByRole('button', {name: 'Will Bitcoin reach $120,000 by September 30?, NO, 31.6¢ last trade', exact: true});
    await expect(unselected).toHaveCSS('border-top-color', 'rgb(97, 113, 123)');
    checkReads(ledger);
});
