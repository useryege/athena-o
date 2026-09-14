import {expect, test, type Locator, type Page} from '@playwright/test';
import {assertThemeLayout, assertThemeLedger, openThemeCase} from './theme-refactor/routes';

const assertFullHeightMobileDrawer = async (page: Page, drawer: Locator) => {
    const viewport = page.viewportSize();
    if (!viewport) throw new Error('A fixed viewport is required for drawer assertions');
    const expectedWidth = Math.min(280, viewport.width - 48);
    await expect.poll(async () => (await drawer.boundingBox())?.x).toBeCloseTo(0, 0);
    await expect.poll(async () => (await drawer.boundingBox())?.y).toBeCloseTo(0, 0);
    await expect.poll(async () => (await drawer.boundingBox())?.width).toBeCloseTo(expectedWidth, 0);
    await expect.poll(async () => (await drawer.boundingBox())?.height).toBeCloseTo(viewport.height, 0);

    const backdrop = page.locator('.athena-shell__backdrop');
    const [backdropBox, brandBox, closeBox, firstGroupBox] = await Promise.all([
        backdrop.boundingBox(),
        drawer.locator('.athena-brand').boundingBox(),
        drawer.getByRole('button', {name: 'Close navigation'}).boundingBox(),
        drawer.locator('.ant-menu-item-group').first().boundingBox()
    ]);
    expect(backdropBox?.y).toBeCloseTo(0, 0);
    expect(backdropBox?.height).toBeCloseTo(viewport.height, 0);
    expect(brandBox).not.toBeNull();
    expect(closeBox).not.toBeNull();
    expect(firstGroupBox).not.toBeNull();
    expect(closeBox!.y).toBeGreaterThanOrEqual(brandBox!.y - 1);
    expect(closeBox!.y + closeBox!.height).toBeLessThanOrEqual(brandBox!.y + brandBox!.height + 1);
    expect(firstGroupBox!.y).toBeGreaterThanOrEqual(brandBox!.y + brandBox!.height - 1);
};

test('theme:core 系统浅色和旧偏好不能改变深色登录', async ({page}, info) => {
    await page.emulateMedia({colorScheme: 'light'});
    await page.addInitScript(() => {
        localStorage.setItem('athena.member.preferences', JSON.stringify({version: 6, theme: 'light'}));
    });
    const ledger = await openThemeCase(page, 'member-login');
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
    await expect(page.locator('body')).toHaveCSS('background-color', 'rgb(6, 8, 11)');
    await page.emulateMedia({colorScheme: 'dark'});
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
    await assertThemeLayout(page);
    const screenshot = info.outputPath('member-login-dark.png');
    await page.screenshot({path: screenshot, fullPage: true});
    await info.attach('member-login-dark', {path: screenshot, contentType: 'image/png'});
    assertThemeLedger(ledger);
});

for (const viewport of [
    {name: 'desktop', width: 1440, height: 900},
    {name: 'mobile', width: 390, height: 844}
]) {
    test(`theme:core member shell uses the approved ${viewport.name} application frame`, async ({page}, info) => {
        await page.setViewportSize(viewport);
        const ledger = await openThemeCase(page, 'member-shell');
        await expect(page.locator('.athena-shell__header')).toHaveCSS('min-height', '64px');
        const account = page.getByRole('button', {name: 'Open account menu'});
        await expect(account).toBeVisible();
        await expect(account.locator('.athena-account-avatar')).toHaveCSS('background-color', 'rgb(24, 29, 34)');
        await expect(account.locator('.athena-account-avatar')).toHaveCSS('border-top-color', 'rgb(97, 113, 123)');
        await expect(page.locator('.athena-shell__header').getByRole('button', {name: 'Open account menu'})).toHaveCount(1);
        await expect(page.locator('.athena-shell__sider').getByRole('button', {name: 'Open account menu'})).toHaveCount(0);
        if (viewport.name === 'desktop') {
            await expect(page.locator('.athena-shell__sider')).toHaveCSS('width', '224px');
            await expect(page.locator('.athena-shell__sider').getByRole('button', {name: 'Help'})).toHaveAttribute('aria-current', 'page');
            await expect(page.locator('.athena-shell__sider').getByText('Member workspace', {exact: true})).toBeVisible();
        } else {
            await expect(page.locator('.athena-shell__sider')).toHaveAttribute('aria-hidden', 'true');
            await expect(page.locator('.athena-shell__sider')).toHaveCSS('width', '0px');
            await expect(page.locator('.athena-shell__sider')).toHaveCSS('visibility', 'hidden');
            expect((await page.locator('.athena-shell > .ant-layout').boundingBox())?.x).toBe(0);
            await expect(page.locator('.athena-shell__sider').getByRole('menuitem')).toHaveCount(0);
            const toggle = page.getByRole('button', {name: 'Open navigation'});
            await toggle.click();
            const drawer = page.getByRole('dialog', {name: 'Primary navigation'});
            await expect(drawer).toBeVisible();
            await expect(drawer.getByRole('button', {name: 'Close navigation'})).toBeFocused();
            await assertFullHeightMobileDrawer(page, drawer);
            await page.keyboard.press('Escape');
            await expect(toggle).toBeFocused();
            await expect(page.locator('.athena-shell__sider')).toHaveCSS('visibility', 'hidden');
        }
        await assertThemeLayout(page);
        const screenshot = info.outputPath(`member-shell-${viewport.name}.png`);
        await page.screenshot({path: screenshot, fullPage: true});
        await info.attach(`member-shell-${viewport.name}`, {path: screenshot, contentType: 'image/png'});
        assertThemeLedger(ledger);
    });
}

test('theme:core admin mobile navigation traps focus and restores the toggle', async ({page}) => {
    await page.setViewportSize({width: 390, height: 844});
    const ledger = await openThemeCase(page, 'admin-shell');
    const hiddenSider = page.locator('.athena-shell__sider');
    await expect(hiddenSider).toHaveAttribute('aria-hidden', 'true');
    await expect(hiddenSider).toHaveCSS('width', '0px');
    await expect(hiddenSider).toHaveCSS('visibility', 'hidden');
    expect((await page.locator('.athena-shell > .ant-layout').boundingBox())?.x).toBe(0);
    await expect(hiddenSider.getByRole('menuitem')).toHaveCount(0);
    const toggle = page.getByRole('button', {name: 'Open navigation'});
    await toggle.click();
    const drawer = page.getByRole('dialog', {name: 'Administration navigation'});
    await expect(drawer).toBeVisible();
    await expect(drawer.getByRole('button', {name: 'Close navigation'})).toBeFocused();
    await assertFullHeightMobileDrawer(page, drawer);
    await expect(drawer.getByRole('button', {name: 'Help'})).toHaveAttribute('aria-current', 'page');
    await expect(drawer.getByText('Administration console', {exact: true})).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(drawer).toBeHidden();
    await expect(toggle).toBeFocused();
    assertThemeLedger(ledger);
});

for (const scenario of [
    {id: 'member-shell', drawerName: 'Primary navigation'},
    {id: 'admin-shell', drawerName: 'Administration navigation'}
]) {
    test(`theme:drawer ${scenario.id} covers the 320px viewport without crossing its brand row`, async ({page}) => {
        await page.setViewportSize({width: 320, height: 720});
        const ledger = await openThemeCase(page, scenario.id);
        const toggle = page.getByRole('button', {name: 'Open navigation'});
        await toggle.click();
        const drawer = page.getByRole('dialog', {name: scenario.drawerName});
        await expect(drawer.getByRole('button', {name: 'Close navigation'})).toBeFocused();
        await assertFullHeightMobileDrawer(page, drawer);
        await page.keyboard.press('Escape');
        await expect(drawer).toBeHidden();
        await expect(toggle).toBeFocused();
        assertThemeLedger(ledger);
    });
}

test('theme:harness records writes and rejects undeclared methods instead of fabricating success', async ({page}) => {
    const ledger = await openThemeCase(page, 'member-login');
    const response = await page.evaluate(async () => {
        const target = `${location.pathname.replace(/\/login$/, '')}/api/v1/app/bootstrap?source=contract`;
        const result = await fetch(target, {
            method: 'POST',
            headers: {'Content-Type': 'application/json', 'X-Athena-Application-Realm': 'member'},
            body: JSON.stringify({write: true})
        });
        return {status: result.status, body: await result.json()};
    });
    expect(response.status).toBe(500);
    expect(response.body.error.code).toBe('UNEXPECTED_THEME_REQUEST');
    expect(ledger.requests.at(-1)).toEqual({method: 'POST', path: '/api/v1/app/bootstrap', query: '?source=contract', realm: 'member', body: {write: true}});
    await page.evaluate(() => fetch('https://fixture-vendor.invalid/markets').catch(() => undefined));
    expect(ledger.unexpected).toContain('cross-origin request GET https://fixture-vendor.invalid/markets');
    if (new URL(page.url()).pathname.startsWith('/athena/')) {
        const unexpectedBeforeStaticAsset = ledger.unexpected.length;
        await page.evaluate(
            () =>
                new Promise<void>(resolve => {
                    const image = new Image();
                    image.onload = () => resolve();
                    image.onerror = () => resolve();
                    image.src = `${location.origin}/favicon.ico?theme-static=1`;
                })
        );
        expect(ledger.unexpected).toHaveLength(unexpectedBeforeStaticAsset);
        await page.evaluate(() => fetch(`${location.origin}/api/v1/app/bootstrap`).catch(() => undefined));
        expect(ledger.unexpected).toContain('request escaped deployment prefix GET /api/v1/app/bootstrap');
    }
    expect(() => assertThemeLedger(ledger)).toThrow(/undeclared|wrong-method/i);
});

test('theme:core shell typography and account portal follow 200% root text sizing', async ({page}, info) => {
    await page.setViewportSize({width: 720, height: 1000});
    const ledger = await openThemeCase(page, 'member-shell');
    await page.evaluate(() => {
        document.documentElement.style.fontSize = '32px';
    });
    await expect(page.locator('html')).toHaveCSS('font-size', '32px');
    const header = page.locator('.athena-shell__header');
    const headerBox = await header.boundingBox();
    expect(headerBox?.height).toBeLessThanOrEqual(96);
    await expect(page.locator('.athena-shell__breadcrumb .ant-breadcrumb-link').last()).toHaveCSS('font-size', '26px');

    const account = page.getByRole('button', {name: 'Open account menu'});
    await account.click();
    await expect(page.locator('.athena-account-menu .ant-dropdown-menu-item').filter({hasText: 'Profile'})).toHaveCSS('font-size', '28px');
    const identityClipping = await page
        .locator('.athena-account-menu__summary strong, .athena-account-menu__summary small')
        .evaluateAll(nodes => nodes.map(node => ({clientWidth: node.clientWidth, scrollWidth: node.scrollWidth})));
    expect(identityClipping.every(item => item.scrollWidth <= item.clientWidth + 1)).toBe(true);
    await page.keyboard.press('Escape');
    await expect(page.locator('.athena-account-menu')).toBeHidden();

    const toggle = page.getByRole('button', {name: 'Open navigation'});
    await toggle.click();
    const drawer = page.getByRole('dialog', {name: 'Primary navigation'});
    await assertFullHeightMobileDrawer(page, drawer);
    await expect(page.locator('.ant-tooltip')).toHaveCount(0);
    await assertThemeLayout(page);
    const screenshot = info.outputPath('member-shell-root-200.png');
    await page.screenshot({path: screenshot, fullPage: true});
    await info.attach('member-shell-root-200', {path: screenshot, contentType: 'image/png'});
    assertThemeLedger(ledger);
});

for (const viewport of [
    {name: 'desktop', width: 1440, height: 900},
    {name: 'mobile', width: 390, height: 844}
]) {
    for (const id of [
        'member-login',
        'admin-login',
        'member-register',
        'admin-register',
        'member-profile',
        'admin-profile',
        'member-security',
        'member-access',
        'admin-access',
        'member-notifications',
        'member-help',
        'admin-help'
    ]) {
        test(`theme:identity ${id} ${viewport.name}`, async ({page}, info) => {
            await page.setViewportSize(viewport);
            const ledger = await openThemeCase(page, id);
            if (id.endsWith('-profile')) {
                await expect(page.getByLabel('Username', {exact: true})).toHaveAttribute('readonly', '');
                await expect(page.getByLabel('Display name', {exact: true})).toHaveValue(id.startsWith('admin') ? 'Fixture Admin' : 'Fixture Member');
                await expect(page.locator('.account-profile-avatar .ant-avatar')).toHaveCount(1);
                await expect(page.locator('.account-center-hero')).toHaveCount(0);
            }
            if (id.endsWith('-access')) {
                const modules = page.getByRole('heading', {name: 'Module access', exact: true});
                const session = page.getByRole('heading', {name: 'Current session', exact: true});
                expect((await modules.boundingBox())!.y).toBeLessThan((await session.boundingBox())!.y);
                await expect(page.getByText('Access revision', {exact: true})).toBeHidden();
                await page.getByText('Times, revisions & versions', {exact: true}).click();
                await expect(page.getByText('Access revision', {exact: true})).toBeVisible();
                await page.getByText('Times, revisions & versions', {exact: true}).click();
            }
            if (id.endsWith('-help')) await expect(page.getByRole('region', {name: 'Help resources'})).toBeVisible();
            if (id === 'member-security') await expect(page.getByText('automation-client', {exact: true}).filter({visible: true})).toBeVisible();
            if (id === 'member-notifications') await expect(page.getByText('Alex Chen (@alex_demo)', {exact: true})).toBeVisible();
            await page.evaluate(() => window.scrollTo(0, 0));
            await assertThemeLayout(page);
            await page.mouse.move(0, 0);
            const screenshot = info.outputPath(`${id}-${viewport.name}.png`);
            await page.screenshot({path: screenshot, fullPage: true, animations: 'disabled'});
            await info.attach(`${id}-${viewport.name}`, {path: screenshot, contentType: 'image/png'});
            assertThemeLedger(ledger);
        });
    }
}

test('theme:identity profile draft survives conflict and leave dialog compares saved and draft', async ({page}) => {
    const {themeCases} = await import('./theme-refactor/cases');
    const {installThemeCase} = await import('./theme-refactor/routes');
    const scenario = structuredClone(themeCases.find(item => item.id === 'member-profile')!);
    scenario.replies.push({
        method: 'PUT',
        path: '/api/v1/account/11111111-1111-4111-8111-111111111111/profile',
        realm: 'member',
        status: 409,
        json: {message: 'Profile revision conflict'}
    });
    const ledger = await installThemeCase(page, scenario);
    await page.goto(`${process.env.ATHENA_UI_E2E_PATH_PREFIX || ''}/account/profile`);
    const input = page.getByLabel('Display name', {exact: true});
    await input.fill('My unsaved draft');
    const refreshedUser = scenario.replies.find(item => item.path === '/api/v1/session/userinfo')!.json as any;
    refreshedUser.profile = {...refreshedUser.profile, displayName: 'Latest saved name', revision: 5};
    await page.getByRole('button', {name: 'Save profile'}).click();
    await expect(page.getByText('Profile changed elsewhere', {exact: true})).toBeVisible();
    await expect(input).toHaveValue('My unsaved draft');
    expect(ledger.requests.filter(item => item.method === 'PUT').map(item => item.body)).toEqual([{displayName: 'My unsaved draft', expectedRevision: 4}]);
    await page.getByRole('button', {name: 'Access & session Permissions and versions'}).click();
    const dialog = page.getByRole('dialog');
    await expect(dialog.getByText('Latest saved name', {exact: true})).toBeVisible();
    await expect(dialog.getByText('My unsaved draft', {exact: true})).toBeVisible();
    await expect(dialog.getByRole('button', {name: 'Keep editing'})).toBeFocused();
    const keepBox = await dialog.getByRole('button', {name: 'Keep editing'}).boundingBox();
    const discardBox = await dialog.getByRole('button', {name: 'Discard and leave'}).boundingBox();
    expect(keepBox!.y).toBeCloseTo(discardBox!.y, 0);
    await page.keyboard.press('Escape');
    await expect(input).toHaveValue('My unsaved draft');
    await page.getByRole('button', {name: 'Reset', exact: true}).click();
    await expect(input).toHaveValue('Latest saved name');
    assertThemeLedger(ledger);
});

test('theme:identity key creation stays single-flight and clipboard rejection leaves the full secret selectable', async ({page}) => {
    await page.addInitScript(() =>
        Object.defineProperty(navigator, 'clipboard', {
            configurable: true,
            value: {
                writeText: async () => {
                    throw new Error('Clipboard denied');
                }
            }
        })
    );
    const ledger = await openThemeCase(page, 'member-security');
    await page.getByRole('button', {name: 'Create API key', exact: true}).click();
    const creation = page.getByRole('dialog', {name: 'Create API key', exact: true});
    await creation.getByLabel('Key ID', {exact: true}).fill('identity-test');
    await creation.getByRole('button', {name: 'Create key', exact: true}).click();
    await page.keyboard.press('Enter');
    await page.keyboard.press('Escape');
    await expect(creation).toBeVisible();
    await expect(creation.getByRole('button', {name: 'Cancel', exact: true})).toBeDisabled();
    const result = page.getByRole('dialog', {name: 'Copy your API key now'});
    await expect(result).toBeVisible();
    expect(ledger.requests.filter(item => item.method === 'POST').map(item => item.body)).toEqual([{expiresIn: 7776000, id: 'identity-test'}]);
    await result.getByRole('button', {name: 'Copy API key', exact: true}).click();
    await expect(page.getByText('Could not copy API key', {exact: true})).toBeVisible();
    const secret = result.getByRole('textbox');
    await expect(secret).toHaveValue('fixture-once-only-secret-full-value');
    await secret.selectText();
    expect(await secret.evaluate((node: HTMLTextAreaElement) => node.value.slice(node.selectionStart, node.selectionEnd))).toBe('fixture-once-only-secret-full-value');
    await page.keyboard.press('Escape');
    await expect(result).toBeVisible();
    await result.getByRole('button', {name: 'Done', exact: true}).click();
    await expect(result).toBeHidden();
    assertThemeLedger(ledger);
});

test('theme:identity failed replacement preserves Connected and sends one empty attempt body', async ({page}) => {
    const ledger = await openThemeCase(page, 'member-notifications');
    await page.getByRole('button', {name: 'Reconnect', exact: true}).click();
    await expect(page.getByText('Replacement setup unavailable', {exact: true})).toBeVisible();
    await expect(page.getByText('Connected', {exact: true}).filter({visible: true})).toHaveCount(2);
    await expect(page.getByText('Alex Chen (@alex_demo)', {exact: true})).toBeVisible();
    expect(ledger.requests.filter(item => item.method === 'POST').map(item => item.body)).toEqual([{}]);
    assertThemeLedger(ledger);
});

for (const realm of ['member', 'admin']) {
    test(`theme:identity ${realm} Help shows only configured resources and deployment links`, async ({page}) => {
        for (const configured of [false, true]) {
            const ledger = await openThemeCase(page, `${realm}-help${configured ? '-configured' : ''}`);
            const prefix = process.env.ATHENA_UI_E2E_PATH_PREFIX || '';
            await expect(page.getByRole('link', {name: /LLM discovery/})).toHaveAttribute('href', `${prefix}/llms.txt`);
            await expect(page.getByRole('link', {name: /Full-Account AI Access/})).toHaveAttribute('href', `${prefix}/docs/ai/safety.md`);
            await expect(page.getByRole('link', {name: /Swagger UI/})).toHaveAttribute('href', `${prefix}/swagger-ui`);
            await expect(page.getByRole('link', {name: 'Team chat'})).toHaveCount(configured ? 1 : 0);
            await expect(page.getByRole('link', {name: 'Desktop download'})).toHaveCount(configured ? 1 : 0);
            await expect(page.getByRole('button', {name: 'Connect an AI'})).toHaveCount(realm === 'member' ? 1 : 0);
            assertThemeLedger(ledger);
        }
    });
}

test('theme:identity missing Phantom and expired Google ticket retain recovery', async ({page}) => {
    const ledger = await openThemeCase(page, 'member-login');
    await page.getByRole('button', {name: 'Continue with Phantom'}).click();
    await expect(page.getByText('Phantom is not installed in this browser.', {exact: true})).toBeVisible();
    expect(ledger.requests.filter(item => item.method !== 'GET')).toEqual([]);
    assertThemeLedger(ledger);
    const {themeCases} = await import('./theme-refactor/cases');
    const {installThemeCase} = await import('./theme-refactor/routes');
    const scenario = structuredClone(themeCases.find(item => item.id === 'member-register')!);
    scenario.replies[0] = {...scenario.replies[0], status: 410, json: {reason: 'registration_expired'}};
    const expiredLedger = await installThemeCase(page, scenario);
    await page.goto(`${process.env.ATHENA_UI_E2E_PATH_PREFIX || ''}${scenario.route}`);
    await expect(page.getByText('Your registration session has expired. Return to sign in and try again.', {exact: true})).toBeVisible();
    await expect(page.getByRole('button', {name: 'Return to sign in'})).toBeEnabled();
    assertThemeLedger(expiredLedger);
});

for (const viewport of [{name: 'narrow', width: 320, height: 844, rootSize: 16}, {name: 'root-200', width: 720, height: 1000, rootSize: 32}]) {
    for (const id of ['member-profile', 'member-security', 'member-notifications', 'admin-help']) {
        test(`theme:identity ${id} ${viewport.name} text and boundaries`, async ({page}, info) => {
            await page.setViewportSize(viewport);
            const ledger = await openThemeCase(page, id);
            if (viewport.rootSize === 32) {
                const before = await page.locator('.app-page h1').evaluate(node => Number.parseFloat(getComputedStyle(node).fontSize));
                await page.evaluate(() => {document.documentElement.style.fontSize = '32px';});
                await expect(page.locator('.app-page h1')).toHaveCSS('font-size', `${before * 2}px`);
                await expect(page.locator('.app-page__heading > .ant-typography-secondary')).toHaveCSS('font-size', '32px');
            }
            await assertThemeLayout(page);
            await page.mouse.move(0, 0);
            const screenshot = info.outputPath(`${id}-${viewport.name}.png`);
            await page.screenshot({path: screenshot, fullPage: true, animations: 'disabled'});
            await info.attach(`${id}-${viewport.name}`, {path: screenshot, contentType: 'image/png'});
            assertThemeLedger(ledger);
        });
    }
}

test('theme:identity avatar rejects unsupported content before sending a write', async ({page}) => {
    const ledger = await openThemeCase(page, 'member-profile');
    await page.locator('input[type=file]').setInputFiles({name: 'avatar.txt', mimeType: 'text/plain', buffer: Buffer.from('not an image')});
    await expect(page.getByText('Unsupported avatar format', {exact: true})).toBeVisible();
    expect(ledger.requests.filter(item => item.method !== 'GET')).toEqual([]);
    assertThemeLedger(ledger);
});

test('theme:identity Access uses semantic success and neutral disabled capabilities', async ({page}) => {
    const memberLedger = await openThemeCase(page, 'member-access');
    await expect(page.locator('.account-access-flags .ant-tag').first()).toHaveCSS('color', 'rgb(85, 217, 161)');
    await expect(page.locator('.account-center-content .section-panel__extra .ant-tag').last()).toHaveCSS('color', 'rgb(85, 217, 161)');
    await expect(page.locator('.account-module-summary .ant-tag').filter({hasText: 'Read & write'})).toHaveCSS('color', 'rgb(85, 217, 161)');
    await expect(page.locator('.account-module-summary .ant-tag').filter({hasText: 'Read only'})).toBeVisible();
    await expect(page.locator('.account-module-summary .ant-tag').filter({hasText: 'No access'}).first()).toBeVisible();
    assertThemeLedger(memberLedger);
    const adminLedger = await openThemeCase(page, 'admin-access');
    for (const tag of await page.locator('.account-access-flags .ant-tag').all()) {
        await expect(tag).toHaveText('No');
        await expect(tag).not.toHaveClass(/ant-tag-error|ant-tag-red/);
    }
    assertThemeLedger(adminLedger);
});

test('theme:identity mobile AI instructions retain a visible action footer while the body scrolls', async ({page}, info) => {
    await page.setViewportSize({width: 390, height: 844});
    const errors: string[] = [];
    page.on('pageerror', error => errors.push(error.message));
    const ledger = await openThemeCase(page, 'member-security');
    let credentialChecks = 0;
    // Credential verification intentionally omits the session realm and cookie; verify this bearer boundary explicitly.
    await page.route('**/api/v1/session/userinfo', async route => {
        const request = route.request();
        expect(request.url()).toBe(`${process.env.ATHENA_UI_E2E_BASE_URL}${process.env.ATHENA_UI_E2E_PATH_PREFIX || ''}/api/v1/session/userinfo`);
        expect(request.method()).toBe('GET');
        expect(request.headers().authorization).toBe('Bearer fixture-once-only-secret-full-value');
        expect(request.headers()['x-athena-application-realm']).toBeUndefined();
        credentialChecks++;
        await route.fulfill({status: 200, json: {loggedIn: true, accountId: '11111111-1111-4111-8111-111111111111'}});
    });
    await page.getByRole('button', {name: 'Connect AI', exact: true}).click();
    const creation = page.getByRole('dialog', {name: 'Connect AI', exact: true});
    await creation.getByLabel('Connection name', {exact: true}).fill('identity-ai');
    await creation.getByRole('button', {name: 'Create connection', exact: true}).click();
    const result = page.getByRole('dialog', {name: 'AI connection instructions ready'});
    await expect(result.getByText('Credential ready', {exact: true})).toBeVisible();
    await page.getByRole('alert').filter({hasText: 'Connection instructions ready'}).getByRole('button', {name: 'Close', exact: true}).click();
    await page.evaluate(() => Promise.all(document.getAnimations().filter(animation => animation.effect?.getTiming().iterations !== Infinity).map(animation => animation.finished.catch(() => undefined))));
    const body = result.locator('.ant-modal-body');
    const footer = result.locator('.ant-modal-footer');
    const before = await footer.boundingBox();
    expect(before!.y + before!.height).toBeLessThanOrEqual(844);
    expect(await body.evaluate(node => node.scrollHeight > node.clientHeight)).toBe(true);
    await body.evaluate(node => node.scrollTop = node.scrollHeight);
    expect((await footer.boundingBox())!.y).toBeCloseTo(before!.y, 0);
    await expect(result.getByRole('textbox')).toHaveValue(/fixture-once-only-secret-full-value/);
    expect(ledger.requests.filter(item => item.method === 'POST').map(item => item.body)).toEqual([{expiresIn: 7776000, id: 'identity-ai'}]);
    await page.mouse.move(0, 0);
    await page.screenshot({path: info.outputPath('member-ai-instructions-mobile.png'), animations: 'disabled'});
    await result.getByRole('button', {name: 'Done', exact: true}).click();
    await expect(result).toBeHidden();
    expect(errors).toEqual([]);
    expect(credentialChecks).toBe(1);
    assertThemeLedger(ledger);
});

test('theme:identity review mobile Profile leave confirmation stacks safe actions and restores focus', async ({page}, info) => {
    await page.setViewportSize({width: 390, height: 844});
    const {themeCases} = await import('./theme-refactor/cases');
    const {installThemeCase} = await import('./theme-refactor/routes');
    const scenario = structuredClone(themeCases.find(item => item.id === 'member-profile')!);
    const ledger = await installThemeCase(page, scenario);
    await page.goto(`${process.env.ATHENA_UI_E2E_PATH_PREFIX || ''}/account/profile`);
    await page.getByLabel('Display name', {exact: true}).fill('Unsaved mobile name');
    const section = page.getByRole('combobox', {name: 'Account section'});
    await section.focus();
    await page.keyboard.press('ArrowDown');
    await page.locator('.ant-select-item-option[title="Access & session"]').click();
    const dialog = page.getByRole('dialog', {name: 'Discard unsaved profile changes?'});
    const keep = dialog.getByRole('button', {name: 'Keep editing'});
    const discard = dialog.getByRole('button', {name: 'Discard and leave'});
    await expect(keep).toBeFocused();
    await page.evaluate(() => Promise.all(document.getAnimations().filter(animation => animation.effect?.getTiming().iterations !== Infinity).map(animation => animation.finished.catch(() => undefined))));
    const [keepBox, discardBox] = await Promise.all([keep.boundingBox(), discard.boundingBox()]);
    expect(discardBox!.y).toBeGreaterThanOrEqual(keepBox!.y + keepBox!.height + 8);
    expect(keepBox!.width).toBeGreaterThan(280);
    expect(discardBox!.width).toBeCloseTo(keepBox!.width, 0);
    expect(discardBox!.x).toBeCloseTo(keepBox!.x, 0);
    expect(discardBox!.x + discardBox!.width).toBeCloseTo(keepBox!.x + keepBox!.width, 0);
    for (let i = 0; i < 5; i++) {
        await page.keyboard.press('Tab');
        const focus = await dialog.evaluate(node => ({inside: node.contains(document.activeElement), active: document.activeElement?.tagName, className: document.activeElement?.className}));
        expect(focus.inside, JSON.stringify({step: i, ...focus})).toBe(true);
    }
    await keep.focus();
    await page.keyboard.press('Shift+Tab');
    await expect(discard).toBeFocused();
    await page.keyboard.press('Shift+Tab');
    await expect(keep).toBeFocused();
    await page.mouse.move(0, 0);
    await page.screenshot({path: info.outputPath('profile-leave-mobile.png'), animations: 'disabled'});
    await page.keyboard.press('Escape');
    await expect(dialog).toBeHidden();
    await expect(section).toBeFocused();
    await expect(page.getByLabel('Display name', {exact: true})).toHaveValue('Unsaved mobile name');
    await section.press('ArrowDown');
    await page.locator('.ant-select-item-option[title="Access & session"]').click();
    await expect(dialog).toBeVisible();
    const replacement = scenario.replies.find(reply => reply.path === '/api/v1/session/userinfo')!.json as any;
    replacement.iss = 'replacement-issuer';
    replacement.profile.displayName = 'Replacement profile';
    await page.evaluate(() => window.dispatchEvent(new Event('focus')));
    await expect(dialog).toBeHidden();
    await expect(page.getByLabel('Display name', {exact: true})).toHaveValue('Replacement profile');
    assertThemeLedger(ledger);
});

for (const phase of ['issued', 'pending']) for (const change of ['account', 'issuer']) {
    test(`theme:identity review Security ${change} switch clears ${phase} credential`, async ({page}) => {
        const {themeCases} = await import('./theme-refactor/cases');
        const {installThemeCase} = await import('./theme-refactor/routes');
        const scenario = structuredClone(themeCases.find(item => item.id === 'member-security')!);
        const ledger = await installThemeCase(page, scenario);
        await page.goto(`${process.env.ATHENA_UI_E2E_PATH_PREFIX || ''}/account/security`);
        await page.getByRole('button', {name: 'Create API key', exact: true}).click();
        await page.getByLabel('Key ID', {exact: true}).fill('old-identity');
        const created = new Promise<void>(resolve => {
            const finished = (request: import('@playwright/test').Request) => {
                if (request.method() === 'POST' && request.url().endsWith('/api/v1/account/security/tokens')) resolve();
            };
            page.on('requestfinished', finished);
            page.on('requestfailed', finished);
        });
        await page.getByRole('button', {name: 'Create key', exact: true}).click();
        const result = page.getByRole('dialog', {name: 'Copy your API key now'});
        if (phase === 'issued') await expect(result).toBeVisible();
        else await expect.poll(() => ledger.requests.filter(item => item.method === 'POST').length).toBe(1);
        const user = scenario.replies.find(reply => reply.path === '/api/v1/session/userinfo')!.json as any;
        if (change === 'account') user.accountId = '33333333-3333-4333-8333-333333333333';
        else user.iss = 'replacement-issuer';
        user.profile.displayName = 'Replacement identity';
        await page.evaluate(() => window.dispatchEvent(new Event('focus')));
        await expect(page.getByRole('heading', {name: 'Replacement identity'})).toBeVisible();
        await created;
        await expect(result).toBeHidden();
        await expect(page.getByRole('dialog', {name: 'Create API key', exact: true})).toBeHidden();
        await expect(page.locator('.account-secret-value')).toHaveCount(0);
        expect(ledger.requests.filter(item => item.method === 'POST').map(item => item.body)).toEqual([{id: 'old-identity', expiresIn: 7776000}]);
        assertThemeLedger(ledger);
    });
}

for (const change of ['account', 'issuer']) {
    test(`theme:identity review Notifications ${change} switch destroys the old confirmation`, async ({page}) => {
        const {themeCases} = await import('./theme-refactor/cases');
        const {installThemeCase} = await import('./theme-refactor/routes');
        const scenario = structuredClone(themeCases.find(item => item.id === 'member-notifications')!);
        const user = structuredClone((scenario.replies.find(reply => reply.path.endsWith('/bootstrap'))!.json as any).session.userInfo);
        scenario.replies.push({method: 'GET', path: '/api/v1/session/userinfo', realm: 'member', status: 200, json: user});
        const ledger = await installThemeCase(page, scenario);
        await page.goto(`${process.env.ATHENA_UI_E2E_PATH_PREFIX || ''}/notifications`);
        await page.getByRole('button', {name: 'Disconnect', exact: true}).click();
        const dialog = page.getByRole('dialog', {name: 'Disconnect Telegram?'});
        await expect(dialog).toBeVisible();
        if (change === 'account') user.accountId = '33333333-3333-4333-8333-333333333333';
        else user.iss = 'replacement-issuer';
        const settings = scenario.replies.find(reply => reply.method === 'GET' && reply.path.endsWith('/notification-bindings/telegram'))!;
        settings.json = {botAvailable: true, botUsername: 'fixture_bot'};
        await page.evaluate(() => window.dispatchEvent(new Event('focus')));
        await expect(dialog).toBeHidden();
        await expect(page.getByText('Alex Chen (@alex_demo)', {exact: true})).toBeHidden();
        expect(ledger.requests.filter(item => item.method !== 'GET')).toEqual([]);
        assertThemeLedger(ledger);
    });
}
