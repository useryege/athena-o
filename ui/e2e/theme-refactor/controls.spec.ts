import {expect, test} from '@playwright/test';

for (const width of [1440, 390]) {
    test(`theme colors and 200% font scaling at ${width}px`, async ({page}, info) => {
        await page.setViewportSize({width, height: 1100});
        await page.goto('/controls.html');
        const primary = page.getByRole('button', {name: 'Save profile'});
        const danger = page.getByRole('button', {name: 'Remove avatar'});
        for (const [button, colors] of [[primary, ['rgb(0, 255, 167)', 'rgb(81, 255, 195)', 'rgb(9, 195, 133)']], [danger, ['rgb(245, 140, 155)', 'rgb(255, 176, 189)', 'rgb(229, 119, 139)']]] as const) {
            await page.mouse.move(0, 0);
            await expect(button).toHaveCSS('background-color', colors[0]);
            await expect(button).toHaveCSS('color', 'rgb(6, 8, 11)');
            await button.hover();
            await expect(button).toHaveCSS('background-color', colors[1]);
            await expect(button).toHaveCSS('color', 'rgb(6, 8, 11)');
            await page.mouse.down();
            await expect(button).toHaveCSS('background-color', colors[2]);
            await expect(button).toHaveCSS('color', 'rgb(6, 8, 11)');
            await page.mouse.up();
        }
        for (const [type, bg, border, icon] of [
            ['success', 'rgb(14, 32, 27)', 'rgb(45, 87, 71)', 'rgb(85, 217, 161)'],
            ['warning', 'rgb(33, 28, 19)', 'rgb(102, 82, 49)', 'rgb(230, 191, 114)'],
            ['error', 'rgb(36, 23, 28)', 'rgb(103, 59, 71)', 'rgb(245, 140, 155)'],
            ['info', 'rgb(20, 29, 39)', 'rgb(53, 81, 109)', 'rgb(155, 198, 243)']
        ]) {
            const alert = page.locator(`.ant-alert-${type}`);
            await expect(alert).toHaveCSS('background-color', bg);
            await expect(alert).toHaveCSS('border-top-color', border);
            await expect(alert.locator('.ant-alert-icon')).toHaveCSS('color', icon);
        }
        const readonly = page.getByRole('textbox', {name: 'Username'});
        await expect(readonly).toHaveCSS('color', 'rgb(159, 160, 161)');
        await expect(readonly).toHaveCSS('background-color', 'rgb(24, 29, 34)');
        await expect(readonly).toHaveCSS('border-top-color', 'rgb(37, 42, 48)');
        await readonly.focus();
        await expect(readonly).toBeFocused();
        for (const size of [16, 32]) {
            await page.evaluate(value => document.documentElement.style.fontSize = `${value}px`, size);
            await expect(primary).toHaveCSS('font-size', `${size * .875}px`);
            await expect(page.getByRole('textbox', {name: 'Display name'})).toHaveCSS('font-size', `${size * .875}px`);
            await expect(page.locator('.ant-alert-description').first()).toHaveCSS('font-size', `${size * .875}px`);
            await expect(page.getByTestId('body')).toHaveCSS('font-size', `${size}px`);
            const clipping = await page.locator('button, input, .ant-alert').evaluateAll(nodes => nodes.map(node => ({width: node.clientWidth, scrollWidth: node.scrollWidth, height: node.clientHeight, scrollHeight: node.scrollHeight})));
            for (const box of clipping) {
                expect(box.scrollWidth).toBeLessThanOrEqual(box.width + 1);
                expect(box.scrollHeight).toBeLessThanOrEqual(box.height + 1);
            }
            await info.attach(`controls-${width}-${size}`, {body: await page.screenshot({fullPage: true}), contentType: 'image/png'});
        }
    });
}
