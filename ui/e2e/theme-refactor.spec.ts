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
        const result = await fetch(target, {method: 'POST', headers: {'Content-Type': 'application/json', 'X-Athena-Application-Realm': 'member'}, body: JSON.stringify({write: true})});
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
    const identityClipping = await page.locator('.athena-account-menu__summary strong, .athena-account-menu__summary small').evaluateAll(nodes =>
        nodes.map(node => ({clientWidth: node.clientWidth, scrollWidth: node.scrollWidth}))
    );
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
