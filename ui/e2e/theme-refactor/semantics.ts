import {expect, test} from '@playwright/test';
import {assertThemeLedger, openThemeCase} from './routes';

for (const [id, role, name] of [
    ['markets-hot', 'navigation', 'Table pagination'],
    ['foundations-wallets', 'group', 'Private wallets compact view'],
    ['worm-assets-batch-paused', 'group', 'Cash Out batch controls'],
    ['member-notifications', 'region', 'Telegram connection ready']
] as const) {
    test(`theme:semantics ${id} exposes named ${name}`, async ({page}) => {
        await page.setViewportSize({width: 390, height: 844});
        const ledger = await openThemeCase(page, id);
        await expect(page.getByRole(role, {name, exact: true})).toBeVisible();
        assertThemeLedger(ledger);
    });
}

for (const viewport of [
    {width: 1440, height: 900, root: 16},
    {width: 390, height: 844, root: 16},
    {width: 320, height: 844, root: 16},
    {width: 720, height: 1000, root: 32}
]) {
    test(`theme:semantics wallet card ${viewport.width} root${viewport.root} has one label and toggles once by card checkbox and keyboard`, async ({page}, info) => {
        await page.setViewportSize(viewport);
        await page.addInitScript(root => {
            document.addEventListener('DOMContentLoaded', () => {
                document.documentElement.style.fontSize = `${root}px`;
            });
        }, viewport.root);
        const ledger = await openThemeCase(page, 'worm-assets-twenty');
        await page.getByRole('button', {name: /Manage selection/}).click();
        const dialog = page.getByRole('dialog');
        await expect(dialog).not.toHaveClass(/ant-zoom-(?:appear|enter)/);
        const close = dialog.getByRole('button', {name: 'Close', exact: true});
        const closeBox = (await close.boundingBox())!;
        const dialogBox = (await dialog.boundingBox())!;
        expect.soft(closeBox.x).toBeGreaterThanOrEqual(Math.max(0, dialogBox.x));
        expect.soft(closeBox.y).toBeGreaterThanOrEqual(Math.max(0, dialogBox.y));
        expect.soft(closeBox.x + closeBox.width).toBeLessThanOrEqual(Math.min(viewport.width, dialogBox.x + dialogBox.width));
        expect.soft(closeBox.y + closeBox.height).toBeLessThanOrEqual(Math.min(viewport.height, dialogBox.y + dialogBox.height));
        const card = dialog.locator('.worm-wallet-selection-card').first();
        const checkbox = card.getByRole('checkbox', {name: /^Select Wallet 1 /});
        await expect.soft(dialog.locator('label label'), 'native label elements must not be nested').toHaveCount(0);
        await expect(checkbox).toBeChecked();
        await expect(page.locator('html')).toHaveCSS('font-size', `${viewport.root}px`);
        if (viewport.root === 32) {
            const collisions = await dialog.evaluate(element => {
                const close = element.querySelector('.ant-modal-close')!.getBoundingClientRect();
                const titleText = [...element.querySelectorAll('.worm-wallet-selection-modal__title > span')].flatMap(node => {
                    const range = document.createRange();
                    range.selectNodeContents(node);
                    return [...range.getClientRects()];
                });
                const card = element.querySelector('.worm-wallet-selection-card')!;
                const range = document.createRange();
                range.selectNodeContents(card.querySelector('code')!);
                const status = [...card.querySelectorAll('.worm-wallet-selection-card__status .ant-tag')].map(tag => tag.getBoundingClientRect());
                const overlaps = (a: DOMRect, b: DOMRect) => Math.min(a.right, b.right) > Math.max(a.left, b.left) + 1 && Math.min(a.bottom, b.bottom) > Math.max(a.top, b.top) + 1;
                return {title: titleText.some(box => overlaps(box, close)), address: [...range.getClientRects()].some(box => status.some(tag => overlaps(box, tag)))};
            });
            expect.soft(collisions).toEqual({title: false, address: false});
        }
        await expect.soft(card).toHaveCSS('border-top-color', 'rgb(0, 255, 167)');
        expect.soft(await card.evaluate(element => getComputedStyle(element).gridTemplateColumns.split(' ').length)).toBe(viewport.width <= 520 ? 2 : 3);
        await card.locator('.worm-wallet-selection-card__identity').click();
        await expect(checkbox).not.toBeChecked();
        await expect(card).not.toHaveClass(/--selected/);
        await checkbox.click();
        await expect(checkbox).toBeChecked();
        await expect(card).toHaveClass(/--selected/);
        await checkbox.focus();
        await page.keyboard.press('Space');
        await expect(checkbox).not.toBeChecked();
        await page.keyboard.press('Space');
        await expect(checkbox).toBeChecked();
        const disabled = dialog.locator('.worm-wallet-selection-card').last();
        await expect(disabled.getByRole('checkbox')).toBeDisabled();
        await disabled.scrollIntoViewIfNeeded();
        const disabledBox = (await disabled.boundingBox())!;
        await page.mouse.click(disabledBox.x + disabledBox.width / 2, disabledBox.y + disabledBox.height / 2);
        await expect(disabled.getByRole('checkbox')).not.toBeChecked();
        await info.attach('wallet-selection-semantics', {body: await page.screenshot(), contentType: 'image/png'});
        await close.click();
        await expect(dialog).toHaveCount(0);
        await page.getByRole('button', {name: /Manage selection/}).click();
        await expect(dialog).not.toHaveClass(/ant-zoom-(?:appear|enter)/);
        await close.focus();
        await page.keyboard.press('Enter');
        await expect(dialog).toHaveCount(0);
        assertThemeLedger(ledger);
    });
}

test('theme:semantics active workflow number uses readable ink on mint', async ({page}, info) => {
    const ledger = await openThemeCase(page, 'worm-preview');
    const number = page.locator('.ant-steps-item-process .ant-steps-item-icon-number');
    await expect(number).toBeVisible();
    await expect(number).toHaveCSS('color', 'rgb(6, 8, 11)');
    const colors = await number.evaluate(e => ({foreground: getComputedStyle(e).color, background: getComputedStyle(e.closest('.ant-steps-item-icon')!).backgroundColor}));
    const luminance = (color: string) =>
        color
            .match(/[\d.]+/g)!
            .slice(0, 3)
            .map(Number)
            .map(c => c / 255)
            .map(c => (c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4))
            .reduce((sum, c, i) => sum + c * [0.2126, 0.7152, 0.0722][i], 0);
    const foreground = luminance(colors.foreground),
        background = luminance(colors.background);
    const contrast = (Math.max(foreground, background) + 0.05) / (Math.min(foreground, background) + 0.05);
    expect(contrast).toBeGreaterThanOrEqual(4.5);
    await info.attach('workflow-number-contrast', {body: JSON.stringify({...colors, contrast}), contentType: 'application/json'});
    assertThemeLedger(ledger);
});

test('theme:semantics settlement rules link has a persistent non-color cue', async ({page}) => {
    const ledger = await openThemeCase(page, 'markets-corners');
    const link = page.getByRole('link', {name: 'market-specific rules', exact: true});
    await expect(link).toHaveCSS('text-decoration-line', 'underline');
    await link.focus();
    await expect(link).toBeFocused();
    await expect(link).toHaveCSS('text-decoration-line', 'underline');
    assertThemeLedger(ledger);
});

test('theme:semantics workflow number doubles with root text and fits its icon', async ({page}, info) => {
    await page.setViewportSize({width: 720, height: 1000});
    await page.clock.setFixedTime(new Date('2026-09-14T08:00:30Z'));
    const ledger = await openThemeCase(page, 'worm-preview');
    const number = page.locator('.ant-steps-item-process .ant-steps-item-icon-number');
    const before = await number.evaluate(e => parseFloat(getComputedStyle(e).fontSize));
    await page.evaluate(() => (document.documentElement.style.fontSize = '32px'));
    await expect(number).toHaveCSS('font-size', `${before * 2}px`);
    const geometry = await number.evaluate(e => {
        const range = document.createRange();
        range.selectNodeContents(e);
        return {text: range.getBoundingClientRect().toJSON(), icon: e.closest('.ant-steps-item-icon')!.getBoundingClientRect().toJSON()};
    });
    expect(geometry.text.left).toBeGreaterThanOrEqual(geometry.icon.left - 1);
    expect(geometry.text.right).toBeLessThanOrEqual(geometry.icon.right + 1);
    expect(geometry.text.top).toBeGreaterThanOrEqual(geometry.icon.top - 1);
    expect(geometry.text.bottom).toBeLessThanOrEqual(geometry.icon.bottom + 1);
    await info.attach('workflow-root32', {body: await page.screenshot(), contentType: 'image/png'});
    assertThemeLedger(ledger);
});

for (const control of ['checkbox', 'radio', 'switch'] as const) {
    test(`theme:semantics ${control} selected marker remains distinguishable in normal hover and focus`, async ({page}, info) => {
        const ledger = await openThemeCase(page, control === 'checkbox' ? 'worm-assets-twenty' : control === 'radio' ? 'foundations-member-voting' : 'admin-accounts');
        if (control === 'checkbox') await page.getByRole('button', {name: /Manage selection/}).click();
        const input =
            control === 'checkbox'
                ? page.getByRole('dialog').getByRole('checkbox', {checked: true}).first()
                : control === 'radio'
                  ? page.locator('input[type="radio"]:not(:disabled)').first()
                  : page.getByRole('switch', {checked: true}).first();
        await expect(input).toBeVisible();
        if (control === 'radio') await input.check();
        const marker = control === 'checkbox' ? input.locator('..') : control === 'radio' ? input.locator('..') : input.locator('.ant-switch-handle');
        const results = [];
        for (const state of ['normal', 'hover', 'focus']) {
            await page.mouse.move(0, 0);
            if (state === 'hover') await input.hover();
            if (state === 'focus') {
                await input.focus();
                await expect(input).toBeFocused();
            }
            await marker.evaluate(async e => {
                const surface = e.closest('.ant-switch') || e;
                getComputedStyle(surface).backgroundColor;
                await Promise.all(surface.getAnimations({subtree: true}).map(animation => animation.finished));
            });
            const sample = await marker.evaluate((e, control) => {
                const pseudo = getComputedStyle(e, control === 'switch' ? '::before' : '::after');
                return {
                    foreground: control === 'checkbox' ? pseudo.borderRightColor : pseudo.backgroundColor,
                    background: getComputedStyle(control === 'switch' ? e.closest('.ant-switch')! : e).backgroundColor,
                    opacity: pseudo.opacity
                };
            }, control);
            const l = (c: string) =>
                c
                    .match(/[\d.]+/g)!
                    .slice(0, 3)
                    .map(Number)
                    .map(v => v / 255)
                    .map(v => (v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4))
                    .reduce((s, v, i) => s + v * [0.2126, 0.7152, 0.0722][i], 0);
            const a = l(sample.foreground),
                b = l(sample.background),
                contrast = (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05);
            results.push({state, ...sample, contrast});
            expect.soft(contrast).toBeGreaterThanOrEqual(3);
            expect(sample.opacity).toBe('1');
        }
        await info.attach('selected-control-visual', {body: await page.screenshot(), contentType: 'image/png'});
        await info.attach('non-text-contrast', {body: JSON.stringify(results), contentType: 'application/json'});
        assertThemeLedger(ledger);
    });
}
