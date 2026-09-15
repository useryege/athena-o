import {expect, test, type Page} from '@playwright/test';
import {themeCases} from './cases';
import {assertThemeLayout, assertThemeLedger, installThemeCase, openThemeCase} from './routes';
import type {ThemeCase} from './contracts';

const scenario = (id: string) => structuredClone(themeCases.find(item => item.id === id)!) as ThemeCase;
const detailReply = (data: ThemeCase) => data.replies.find(item => item.path.includes('/profit-sharing/rounds/'))!;
const go = async (page: Page, data: ThemeCase) => {
    const ledger = await installThemeCase(page, data);
    await page.goto(`${process.env.ATHENA_UI_E2E_PATH_PREFIX || ''}${data.route}`);
    await expect(page.getByRole('heading', {level: 1, name: data.heading, exact: true})).toBeVisible();
    return ledger;
};

for (const viewport of [
    {name: 'desktop', width: 1440, height: 900},
    {name: 'mobile', width: 390, height: 844}
]) {
    for (const id of ['wallets', 'solana', 'member-rounds', 'member-collecting', 'admin-rounds', 'admin-collecting']) {
        test(`theme:foundations ${id} ${viewport.name} approved structure`, async ({page}, info) => {
            await page.setViewportSize(viewport);
            const ledger = await openThemeCase(page, `foundations-${id}`);
            if (id === 'wallets') {
                await expect(page.getByRole('button', {name: /Import Wallets$/})).toBeVisible();
                await expect(page.getByRole('button', {name: /Create Wallets$/})).toBeVisible();
                await expect(page.getByRole('button', {name: 'Open Research wallet wallet', exact: true})).toBeVisible();
                const address = page.locator('.wallet-record:visible .athena-identifier').first();
                await expect(address).toHaveText('0x14fBE31a86579D941fa03808FE9BB43234c7ed73');
                expect(await address.evaluate(el => el.scrollWidth <= el.clientWidth)).toBe(true);
                const searchInput = await page.getByRole('searchbox', {name: 'Remark or address'}).boundingBox();
                const searchButton = await page.getByRole('button', {name: 'Search', exact: true}).boundingBox();
                expect(searchInput!.y).toBeCloseTo(searchButton!.y, 1);
                expect(searchButton!.x).toBeGreaterThan(searchInput!.x);
            } else if (id === 'solana') {
                await expect(page.getByText('Lunar Current', {exact: true}).filter({visible: true})).toBeVisible();
                if (viewport.name === 'mobile') await expect(page.locator('.resource-table-compact')).toBeVisible();
                await expect(page.getByText('Metadata read failed', {exact: true}).filter({visible: true})).toBeVisible();
                await expect(page.locator('.solana-source--error:visible')).toHaveCSS('color', 'rgb(230, 191, 114)');
            } else if (id.endsWith('rounds')) {
                const link = page.getByRole('link', {name: 'Research Operations · Q3', exact: true});
                await expect(link).toBeVisible();
                await expect(link).toHaveAttribute('href', /profit-sharing\/research-operations-q3$/);
            } else {
                await expect(page.getByRole('region', {name: 'Round progress'})).toBeVisible();
                if (id.startsWith('admin')) await expect(page.getByText('Proposals remain sealed while collection is open.', {exact: true})).toBeVisible();
                else await expect(page.getByRole('textbox', {name: 'Responsibility for Alex Chen'})).toBeVisible();
            }
            await assertThemeLayout(page);
            await page.screenshot({path: info.outputPath(`${id}-${viewport.name}.png`), fullPage: true, animations: 'disabled'});
            expect(ledger.requests.filter(item => item.method !== 'GET')).toHaveLength(0);
            assertThemeLedger(ledger);
        });
    }
}

test('theme:foundations admin can create an empty draft without any eligible accounts', async ({page}) => {
    const data = scenario('foundations-admin-rounds');
    data.replies.find(item => item.path === '/api/v1/account')!.json = {items: [], totalSize: 0};
    data.replies.push({method: 'POST', path: '/api/v1/profit-sharing/rounds', realm: 'admin', status: 503, json: {message: 'Draft save unavailable'}});
    const ledger = await go(page, data);
    await page.getByRole('button', {name: /New round$/}).click();
    await page.getByRole('textbox', {name: 'Round title'}).fill('Empty planning round');
    await page.getByRole('textbox', {name: 'Round URL slug'}).fill('empty-planning');
    while (await page.getByRole('button', {name: /^Remove participant/}).count())
        await page
            .getByRole('button', {name: /^Remove participant/})
            .last()
            .click();
    await page.getByRole('button', {name: /Create round$/}).click();
    await expect(page.getByText('Draft save unavailable', {exact: true})).toBeVisible();
    expect(ledger.requests.filter(item => item.method === 'POST')).toEqual([
        {method: 'POST', path: '/api/v1/profit-sharing/rounds', realm: 'admin', query: '', body: {slug: 'empty-planning', title: 'Empty planning round', participants: []}}
    ]);
    assertThemeLedger(ledger);
});

// Losing an eligible participant must close the Open gate after a fresh authoritative read.
test('theme:foundations revoked roster eligibility disables opening collection', async ({page}) => {
    const data = scenario('foundations-admin-draft');
    const round = detailReply(data).json as any;
    const roster = (detailReply(scenario('foundations-admin-collecting')).json as any).participants;
    round.participants = roster;
    round.participantCount = 5;
    const ledger = await go(page, data);
    await expect(page.getByRole('button', {name: /Open collection$/})).toBeEnabled();
    (data.replies.find(item => item.path === '/api/v1/account')!.json as any).items.pop();
    await page.getByRole('button', {name: 'Refresh data', exact: true}).click();
    await expect.poll(() => ledger.requests.filter(item => item.path === '/api/v1/account').length).toBe(2);
    await expect(page.getByRole('button', {name: /Open collection$/})).toBeDisabled();
    expect(ledger.requests.filter(item => item.method !== 'GET')).toHaveLength(0);
    assertThemeLedger(ledger);
});

// Reusing a mounted route after an issuer switch must not preserve the prior identity's local proposal.
test('theme:foundations issuer replacement clears the member proposal draft and confirmation', async ({page}) => {
    const data = scenario('foundations-member-collecting');
    const user = structuredClone((data.replies[0].json as any).session.userInfo);
    data.replies.push({method: 'GET', path: '/api/v1/session/userinfo', realm: 'member', status: 200, json: user});
    const ledger = await go(page, data);
    const responsibility = page.getByRole('textbox', {name: 'Responsibility for Alex Chen'});
    await responsibility.fill('Old issuer private draft');
    await page.getByRole('button', {name: /Submit proposal$/}).click();
    await expect(page.getByRole('dialog')).toBeVisible();
    user.iss = 'replacement-issuer';
    await page.evaluate(() => window.dispatchEvent(new Event('focus')));
    await expect.poll(() => ledger.requests.filter(item => item.path === '/api/v1/session/userinfo').length).toBe(1);
    await expect(page.getByRole('dialog')).toBeHidden();
    await expect(responsibility).not.toHaveValue('Old issuer private draft');
    expect(ledger.requests.filter(item => item.method !== 'GET')).toHaveLength(0);
    assertThemeLedger(ledger);
});

for (const realm of ['member', 'admin'] as const) {
    for (const phase of ['draft', 'voting', 'closed']) {
        test(`theme:foundations ${realm} ${phase} preserves stage visibility`, async ({page}, info) => {
            await page.setViewportSize({width: 390, height: 844});
            const ledger = await openThemeCase(page, `foundations-${realm}-${phase}`);
            if (phase === 'draft') {
                if (realm === 'member') await expect(page.getByText('This round is being prepared.', {exact: true})).toBeVisible();
                else {
                    await expect(page.getByRole('button', {name: /Open collection$/})).toBeDisabled();
                    await expect(page.getByRole('textbox', {name: 'Round URL slug'})).toHaveAttribute('readonly', '');
                }
            }
            if (phase === 'voting') {
                await expect(page.getByText(/Authors and vote totals remain hidden|Proposal authors and vote totals stay hidden/)).toBeVisible();
                await expect(page.getByText(/Proposed by /)).toHaveCount(0);
                await expect(page.locator('.profit-sharing-proposal-card').getByText(/\d+ votes/)).toHaveCount(0);
                if (realm === 'member') {
                    await expect(page.locator('.profit-sharing-proposal-card').filter({hasText: 'Your proposal'}).getByRole('radio')).toBeDisabled();
                    await expect(page.getByRole('button', {name: /Submit vote$/})).toBeDisabled();
                } else await expect(page.getByRole('button', {name: /Close ballot$/})).toBeDisabled();
            }
            if (phase === 'closed') {
                await expect(page.getByRole('heading', {name: 'Final result', exact: true})).toBeVisible();
                await expect(page.getByText('Winner', {exact: true})).toBeVisible();
                await expect(page.getByRole('button', {name: /Submit proposal|Submit vote|Update vote|Close ballot/})).toHaveCount(0);
            }
            await assertThemeLayout(page);
            await page.screenshot({path: info.outputPath(`${realm}-${phase}.png`), fullPage: true, animations: 'disabled'});
            expect(ledger.requests.filter(item => item.method !== 'GET')).toHaveLength(0);
            assertThemeLedger(ledger);
        });
    }
}

for (const count of [0, 1, 2, 3, 4, 5]) {
    test(`theme:foundations draft roster ${count} retains exact-five opening gate`, async ({page}) => {
        const data = scenario('foundations-admin-draft');
        const round = detailReply(data).json as any;
        round.participants = (detailReply(scenario('foundations-admin-collecting')).json as any).participants.slice(0, count);
        round.participantCount = count;
        const ledger = await go(page, data);
        const open = page.getByRole('button', {name: /Open collection$/});
        if (count === 5) await expect(open).toBeEnabled();
        else await expect(open).toBeDisabled();
        await expect(page.getByRole('button', {name: /^Remove participant/})).toHaveCount(count);
        const add = page.getByRole('button', {name: /Add participant$/});
        if (count === 5) await expect(add).toBeDisabled();
        else await expect(add).toBeEnabled();
        expect(ledger.requests.filter(item => item.path === '/api/v1/account').map(item => item.query)).toEqual(['?page=1&pageSize=100&profitSharingEligibleOnly=true']);
        assertThemeLedger(ledger);
    });
}

test('theme:foundations member keeps incomplete precise draft after failure and discards on conflict', async ({page}, info) => {
    const data = scenario('foundations-member-collecting');
    const round = detailReply(data).json as any;
    const save = {
        method: 'PUT',
        path: `/api/v1/profit-sharing/rounds/${round.slug}/proposal`,
        realm: 'member' as const,
        status: 503,
        json: {message: 'Proposal store unavailable'}
    };
    data.replies.push(save);
    const ledger = await go(page, data);
    const responsibility = page.getByRole('textbox', {name: 'Responsibility for Alex Chen'});
    const share = page.getByRole('spinbutton', {name: 'Share for Alex Chen'});
    await responsibility.fill('');
    await share.fill('24.25');
    await page.getByRole('button', {name: /Save draft$/}).click();
    await expect(page.getByText('Proposal store unavailable', {exact: true})).toBeVisible();
    await expect(responsibility).toHaveValue('');
    await expect(share).toHaveValue('24.25');
    await expect(page.getByRole('button', {name: /Submit proposal$/})).toBeDisabled();
    const writes = ledger.requests.filter(item => item.method === 'PUT');
    expect(writes).toHaveLength(1);
    expect(writes[0].body).toEqual({
        expected_revision: round.myProposal.revision,
        items: round.myProposal.items.map((item: any, i: number) => ({
            participant_account_id: item.participantAccountId,
            responsibility: i === 0 ? '' : item.responsibility,
            share_basis_points: i === 0 ? 2425 : item.shareBasisPoints,
            share_basis_points_set: true
        }))
    });
    save.status = 409;
    save.json = {message: 'Revision changed'};
    round.myProposal.revision += 1;
    round.myProposal.items[0].responsibility = 'Authoritative responsibility';
    await page.getByRole('button', {name: /Save draft$/}).click();
    await expect(responsibility).toHaveValue('Authoritative responsibility');
    await expect(page.getByText('Proposal changed elsewhere', {exact: true})).toBeVisible();
    await page.screenshot({path: info.outputPath('member-conflict.png'), fullPage: true, animations: 'disabled'});
    assertThemeLedger(ledger);
});

test('theme:foundations sealed submission can reopen using the current proposal revision', async ({page}) => {
    const data = scenario('foundations-member-collecting');
    const round = detailReply(data).json as any;
    round.myProposal.status = 'SUBMITTED';
    data.replies.push({
        method: 'POST',
        path: `/api/v1/profit-sharing/rounds/${round.slug}/proposal:reopen`,
        realm: 'member',
        status: 503,
        json: {message: 'Reopen temporarily unavailable'}
    });
    const ledger = await go(page, data);
    await expect(page.getByText('Your proposal is sealed.', {exact: true})).toBeVisible();
    await expect(page.getByRole('textbox', {name: /^Responsibility for/})).toHaveCount(0);
    await page.getByRole('button', {name: /Reopen proposal$/}).click();
    await page.getByRole('dialog').getByRole('button', {name: 'Reopen proposal', exact: true}).click();
    await expect(page.getByText('Reopen temporarily unavailable', {exact: true})).toBeVisible();
    expect(ledger.requests.filter(item => item.method === 'POST').map(item => item.body)).toEqual([{expected_revision: round.myProposal.revision}]);
    assertThemeLedger(ledger);
});

for (const phase of ['collecting', 'voting']) {
    test(`theme:foundations nonparticipant ${phase} stays read only`, async ({page}) => {
        const data = scenario(`foundations-member-${phase}`);
        (data.replies[0].json as any).session.userInfo.accountId = '77777777-7777-4777-8777-777777777777';
        const ledger = await go(page, data);
        await expect(
            page.getByText(phase === 'collecting' ? 'You are not a participant in this round.' : 'You are not eligible to vote in this round.', {exact: true})
        ).toBeVisible();
        await expect(page.getByRole('textbox', {name: /^Responsibility/})).toHaveCount(0);
        await expect(page.getByRole('radio')).toHaveCount(0);
        expect(ledger.requests.filter(item => item.method !== 'GET')).toHaveLength(0);
        assertThemeLedger(ledger);
    });
}

test('theme:foundations wallet copy is complete and read-only detail has no private actions', async ({page, context}, info) => {
    await page.setViewportSize({width: 390, height: 844});
    await context.grantPermissions(['clipboard-read', 'clipboard-write']);
    const ledger = await openThemeCase(page, 'foundations-wallets-readonly');
    await page.getByRole('button', {name: 'Copy Research wallet address', exact: true}).click();
    expect(await page.evaluate(() => navigator.clipboard.readText())).toBe('0x14fBE31a86579D941fa03808FE9BB43234c7ed73');
    await expect(page.getByRole('button', {name: /Import Wallets$|Create Wallets$/})).toHaveCount(0);
    await page.getByRole('button', {name: 'Open Research wallet wallet', exact: true}).click();
    const dialog = page.getByRole('dialog', {name: 'Wallet Details'});
    await expect(dialog).toBeVisible();
    await expect(dialog.getByRole('button', {name: /Edit$|View private key|Reveal/})).toHaveCount(0);
    await page.screenshot({path: info.outputPath('wallet-readonly-detail.png'), animations: 'disabled'});
    expect(ledger.requests.filter(item => item.method !== 'GET')).toHaveLength(0);
    assertThemeLedger(ledger);
});

test('theme:foundations failed wallet import retains the precise lines and can retry once', async ({page}, info) => {
    const data = scenario('foundations-wallets');
    const response = {method: 'POST', path: '/api/v1/wallets:batchImport', realm: 'member' as const, status: 400, json: {message: 'privateKeys[1] invalid test key'}};
    data.replies.push(response);
    const ledger = await go(page, data);
    await page.getByRole('button', {name: /Import Wallets$/}).click();
    const dialog = page.getByRole('dialog', {name: 'Import Wallets'});
    await dialog.getByRole('textbox', {name: 'Private keys', exact: true}).fill('synthetic-key-one\n\nsynthetic-key-two');
    await dialog.getByRole('button', {name: /Import 2/}).click();
    await expect(page.getByText(/Line 3.*invalid test key/)).toBeVisible();
    await expect(dialog.getByRole('textbox', {name: 'Private keys', exact: true})).toHaveValue('synthetic-key-one\n\nsynthetic-key-two');
    expect(ledger.requests.filter(item => item.method === 'POST').map(item => item.body)).toEqual([
        {walletType: 'EVM', privateKeys: ['synthetic-key-one', 'synthetic-key-two'], remark: '', avatarPresetId: ''}
    ]);
    await dialog.getByRole('button', {name: /Import 2/}).click();
    await expect.poll(() => ledger.requests.filter(item => item.method === 'POST').length).toBe(2);
    await page.screenshot({path: info.outputPath('wallet-import-failure.png'), animations: 'disabled'});
    assertThemeLedger(ledger);
});

for (const size of [
    {width: 320, scale: 1},
    {width: 720, scale: 2}
]) {
    for (const id of ['wallets', 'solana', 'member-collecting', 'admin-draft']) {
        test(`theme:foundations ${id} ${size.width}px root${size.scale * 100} enlarges text before boundary checks`, async ({page}, info) => {
            await page.setViewportSize({width: size.width, height: 900});
            const ledger = await openThemeCase(page, `foundations-${id}`);
            const text = page
                .locator(
                    id === 'wallets'
                        ? '.wallet-record:visible .foundation-record-link'
                        : id === 'solana'
                          ? '.solana-candidate .solana-token-identity > span'
                          : id === 'admin-draft'
                            ? '.profit-sharing-definition__participants-heading h3'
                            : '.profit-sharing-editor__intro h2'
                )
                .first();
            const original = await text.evaluate(element => parseFloat(getComputedStyle(element).fontSize));
            await page.evaluate(scale => (document.documentElement.style.fontSize = `${scale * 100}%`), size.scale);
            await expect.poll(() => text.evaluate(element => parseFloat(getComputedStyle(element).fontSize))).toBeCloseTo(original * size.scale, 1);
            await assertThemeLayout(page);
            const overflow = await page
                .locator('.foundation-page')
                .evaluate(root =>
                    [
                        ...root.querySelectorAll<HTMLElement>(
                            '.foundation-address, .foundation-facts dd, .profit-sharing-editor-row, .profit-sharing-definition-participant, .wallet-search'
                        )
                    ]
                        .filter(el => el.offsetWidth > 0 && el.scrollWidth > el.clientWidth + 1)
                        .map(el => `${el.className}:${el.scrollWidth}/${el.clientWidth}`)
                );
            expect(overflow).toEqual([]);
            if (id === 'wallets') {
                const search = page.locator('.wallet-search .ant-input-search-btn');
                const geometry = await search.evaluate(el => {
                    const range = document.createRange();
                    range.selectNodeContents(el.querySelector('span')!);
                    const rect = range.getBoundingClientRect();
                    const box = el.getBoundingClientRect();
                    return {text: rect.height, height: box.height, left: rect.left - box.left, right: box.right - rect.right};
                });
                expect(geometry.text).toBeLessThan(geometry.height);
                expect(geometry.left).toBeGreaterThanOrEqual(0);
                expect(geometry.right).toBeGreaterThanOrEqual(0);
            }
            await page.screenshot({path: info.outputPath(`${id}-${size.width}-root${size.scale * 100}.png`), fullPage: true, animations: 'disabled'});
            assertThemeLedger(ledger);
        });
    }
}

test('theme:foundations independent Solana failures retain saved facts and exact query semantics', async ({page}) => {
    const data = scenario('foundations-solana');
    const status = data.replies.find(item => item.path === '/api/v1/solana/status')!;
    const projects = data.replies.find(item => item.path === '/api/v1/solana/projects')!;
    (projects.json as any).totalSize = 26;
    const ledger = await go(page, data);
    await page.getByRole('listitem', {name: 'Next Page'}).getByRole('button').click();
    await expect.poll(() => ledger.requests.filter(item => item.path === projects.path).length).toBe(2);
    await page.getByRole('textbox', {name: 'Name, symbol or Mint'}).fill('  MiNt+Case  ');
    await page.getByRole('button', {name: 'Search candidates'}).click();
    await expect.poll(() => ledger.requests.filter(item => item.path === projects.path).length).toBe(3);
    expect(ledger.requests.filter(item => item.path === projects.path).map(item => item.query)).toEqual([
        '?page=1&pageSize=25&query=',
        '?page=2&pageSize=25&query=',
        '?page=1&pageSize=25&query=MiNt%2BCase'
    ]);
    status.status = 503;
    status.json = {message: 'Scanner request unavailable'};
    projects.status = 503;
    projects.json = {message: 'Candidate request unavailable'};
    await page.getByRole('button', {name: 'Refresh data'}).click();
    await expect(page.getByText('Scanner request unavailable', {exact: true})).toBeVisible();
    await expect(page.getByText('Candidate request unavailable', {exact: true})).toBeVisible();
    await expect(page.getByText('Lunar Current', {exact: true}).filter({visible: true})).toBeVisible();
    await expect(page.getByText('Last known candidates. This list may be stale.', {exact: true})).toBeVisible();
    status.status = 200;
    status.json = scenario('foundations-solana').replies.find(item => item.path === status.path)!.json;
    await page.getByRole('button', {name: 'Retry discovery status'}).click();
    await expect(page.getByText('Scanner request unavailable', {exact: true})).toBeHidden();
    await expect(page.getByText('Candidate request unavailable', {exact: true})).toBeVisible();
    expect(ledger.requests.filter(item => item.path === projects.path)).toHaveLength(4);
    expect(ledger.requests.filter(item => item.method !== 'GET')).toHaveLength(0);
    assertThemeLedger(ledger);
});

test('theme:foundations wallet permission rejection closes import and drops private draft', async ({page}) => {
    const data = scenario('foundations-wallets');
    const user = structuredClone((data.replies[0].json as any).session.userInfo);
    user.access.revision += 1;
    user.access.moduleAccess[0].dataAccess = 'ACCOUNT_DATA_ACCESS_READ';
    data.replies.push({method: 'GET', path: '/api/v1/session/userinfo', realm: 'member', status: 200, json: user});
    data.replies.push({
        method: 'POST',
        path: '/api/v1/wallets:batchImport',
        realm: 'member',
        status: 403,
        json: {code: 7, message: 'Wallet write access revoked', reason: 'ACCOUNT_DATA_ACCESS_DENIED'}
    });
    const ledger = await go(page, data);
    await page.getByRole('button', {name: /Import Wallets$/}).click();
    await page.getByRole('textbox', {name: 'Private keys', exact: true}).fill('synthetic-key-one');
    await page.getByRole('button', {name: 'Import 1 wallet', exact: true}).click();
    await expect(page.getByRole('dialog')).toBeHidden();
    await expect(page.getByRole('button', {name: /Import Wallets$/})).toHaveCount(0);
    expect(ledger.requests.filter(item => item.method === 'POST').map(item => item.body)).toEqual([
        {walletType: 'EVM', privateKeys: ['synthetic-key-one'], remark: '', avatarPresetId: ''}
    ]);
    assertThemeLedger(ledger);
});

test('theme:foundations governance permission rejection clears draft and removes editing', async ({page}) => {
    const data = scenario('foundations-member-collecting');
    const user = structuredClone((data.replies[0].json as any).session.userInfo);
    user.access.profitSharingEnabled = false;
    user.access.revision += 1;
    data.replies.push({method: 'GET', path: '/api/v1/session/userinfo', realm: 'member', status: 200, json: user});
    data.replies.push({
        method: 'PUT',
        path: '/api/v1/profit-sharing/rounds/research-operations-q3/proposal',
        realm: 'member',
        status: 403,
        json: {code: 7, message: 'Profit Sharing permission revoked', reason: 'ACCOUNT_PROFIT_SHARING_ACCESS_DENIED'}
    });
    const ledger = await go(page, data);
    await page.getByRole('textbox', {name: 'Responsibility for Alex Chen'}).fill('Revoked private draft');
    await page.getByRole('button', {name: /Save draft$/}).click();
    await expect(page.getByRole('textbox', {name: 'Responsibility for Alex Chen'})).toHaveCount(0);
    await expect(page.getByRole('button', {name: /Save draft$/})).toHaveCount(0);
    expect(ledger.requests.filter(item => item.method === 'PUT')).toHaveLength(1);
    assertThemeLedger(ledger);
});

test('theme:foundations publish confirmation keeps revision and mobile keyboard boundaries on failure', async ({page}, info) => {
    await page.setViewportSize({width: 390, height: 844});
    const data = scenario('foundations-admin-collecting');
    data.replies.push({
        method: 'POST',
        path: '/api/v1/profit-sharing/rounds/research-operations-q3:publish',
        realm: 'admin',
        status: 503,
        json: {message: 'Publish temporarily unavailable'}
    });
    const ledger = await go(page, data);
    await page.getByRole('button', {name: /Publish proposals$/}).click();
    const dialog = page.getByRole('dialog');
    const cancel = dialog.getByRole('button', {name: 'Cancel', exact: true});
    const publish = dialog.getByRole('button', {name: 'Publish and open voting', exact: true});
    const a = (await cancel.boundingBox())!;
    const b = (await publish.boundingBox())!;
    expect(a.x).toBeCloseTo(b.x, 1);
    expect(a.width).toBeCloseTo(b.width, 1);
    expect(b.y).toBeGreaterThanOrEqual(a.y + a.height);
    await cancel.focus();
    for (let index = 0; index < 5; index++) {
        await page.keyboard.press('Tab');
        expect(await dialog.evaluate(el => el.contains(document.activeElement))).toBe(true);
    }
    await publish.click();
    await expect(page.getByText('Publish temporarily unavailable', {exact: true})).toBeVisible();
    await expect(dialog).toBeVisible();
    expect(ledger.requests.filter(item => item.method === 'POST')).toEqual([
        {method: 'POST', path: '/api/v1/profit-sharing/rounds/research-operations-q3:publish', realm: 'admin', query: '', body: {expected_revision: 8}}
    ]);
    await page.screenshot({path: info.outputPath('admin-publish-failure-mobile.png'), animations: 'disabled'});
    assertThemeLedger(ledger);
});

test('theme:foundations missing Solana discovery facts remain unknown', async ({page}) => {
    const data = scenario('foundations-solana');
    const status = data.replies.find(item => item.path === '/api/v1/solana/status')!;
    status.json = {};
    const ledger = await go(page, data);
    const panel = page.getByRole('region', {name: 'Discovery status'});
    await expect(panel.getByText('Unknown', {exact: true})).toHaveCount(6);
    expect(ledger.requests.filter(item => item.method !== 'GET')).toHaveLength(0);
    assertThemeLedger(ledger);
});

for (const width of [1440, 390]) {
    test(`theme:foundations Solana initial failure differs from successful empty response ${width}px`, async ({page}, info) => {
        await page.setViewportSize({width, height: 900});
        const data = scenario('foundations-solana');
        const projects = data.replies.find(item => item.path === '/api/v1/solana/projects')!;
        const status = data.replies.find(item => item.path === '/api/v1/solana/status')!;
        projects.status = 503;
        projects.json = {message: 'Candidate request unavailable'};
        status.status = 503;
        status.json = {message: 'Scanner request unavailable'};
        const ledger = await go(page, data);
        const candidates = page.getByRole('region', {name: 'Issuance candidates'});
        await expect(candidates.getByText('Candidate request unavailable', {exact: true})).toBeVisible();
        await expect(candidates.locator('.ant-empty')).toHaveCount(0);
        await expect(candidates.getByText('0 items', {exact: true})).toHaveCount(0);
        await expect(candidates.getByText('No discovered candidates yet', {exact: true})).toHaveCount(0);
        await expect(candidates.getByText('Last known candidates. This list may be stale.', {exact: true})).toHaveCount(0);
        await page.screenshot({path: info.outputPath(`solana-initial-failure-${width}.png`), fullPage: true});
        projects.status = 200;
        projects.json = {items: [], totalSize: 0, page: 1, pageSize: 25};
        await page.getByRole('button', {name: 'Retry candidates', exact: true}).click();
        await expect(candidates.getByText('Candidate request unavailable', {exact: true})).toHaveCount(0);
        await expect(candidates.locator('.ant-empty-description:visible')).toBeVisible();
        await expect(candidates.getByText('0 items', {exact: true})).toBeVisible();
        await expect(page.getByText('Scanner request unavailable', {exact: true})).toBeVisible();
        expect(ledger.requests.filter(item => item.path === status.path)).toHaveLength(1);
        expect(ledger.requests.filter(item => item.path === projects.path).map(item => item.query)).toEqual(['?page=1&pageSize=25&query=', '?page=1&pageSize=25&query=']);
        expect(ledger.requests.filter(item => item.method !== 'GET')).toHaveLength(0);
        await page.screenshot({path: info.outputPath(`solana-success-empty-${width}.png`), fullPage: true});
        assertThemeLedger(ledger);
    });
}
