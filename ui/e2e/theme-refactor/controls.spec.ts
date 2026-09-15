import {expect, test} from '@playwright/test';

for (const width of [1440, 390]) {
    test(`theme colors and 200% font scaling at ${width}px`, async ({page}, info) => {
        await page.setViewportSize({width, height: 1100});
        await page.goto('/controls.html');
        const primary = page.getByRole('button', {name: 'Save profile'});
        const danger = page.getByRole('button', {name: 'Remove avatar'});
        for (const [button, colors] of [
            [primary, ['rgb(0, 255, 167)', 'rgb(81, 255, 195)', 'rgb(9, 195, 133)']],
            [danger, ['rgb(245, 140, 155)', 'rgb(255, 176, 189)', 'rgb(229, 119, 139)']]
        ] as const) {
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
        const disabled = page.getByRole('button', {name: 'Disabled primary'});
        await expect(disabled).toBeDisabled();
        await expect(disabled).toHaveCSS('background-color', 'rgba(255, 255, 255, 0.08)');
        await expect(disabled).toHaveCSS('color', 'rgb(159, 160, 161)');
        await disabled.hover();
        await expect(disabled).toHaveCSS('background-color', 'rgba(255, 255, 255, 0.08)');
        await expect(page.getByRole('textbox', {name: 'Disabled input'})).toBeDisabled();
        await expect(page.getByRole('textbox', {name: 'Disabled input'})).toHaveCSS('border-top-style', 'dashed');
        const readonly = page.getByRole('textbox', {name: 'Username'});
        await expect(readonly).toHaveCSS('color', 'rgb(159, 160, 161)');
        await expect(readonly).toHaveCSS('background-color', 'rgb(24, 29, 34)');
        await expect(readonly).toHaveCSS('border-top-color', 'rgb(37, 42, 48)');
        await readonly.focus();
        await expect(readonly).toBeFocused();
        await expect(readonly).toHaveCSS('outline-color', 'rgb(0, 255, 167)');
        for (const size of [16, 32]) {
            await page.evaluate(value => (document.documentElement.style.fontSize = `${value}px`), size);
            await expect(primary).toHaveCSS('font-size', `${size * 0.875}px`);
            await expect(page.getByRole('textbox', {name: 'Display name'})).toHaveCSS('font-size', `${size * 0.875}px`);
            await expect(page.locator('.ant-alert-description').first()).toHaveCSS('font-size', `${size * 0.875}px`);
            await expect(page.getByTestId('body')).toHaveCSS('font-size', `${size}px`);
            const clipping = await page
                .locator('button, input, .ant-alert')
                .evaluateAll(nodes => nodes.map(node => ({width: node.clientWidth, scrollWidth: node.scrollWidth, height: node.clientHeight, scrollHeight: node.scrollHeight})));
            for (const box of clipping) {
                expect(box.scrollWidth).toBeLessThanOrEqual(box.width + 1);
                expect(box.scrollHeight).toBeLessThanOrEqual(box.height + 1);
            }
            await info.attach(`controls-${width}-${size}`, {body: await page.screenshot({fullPage: true}), contentType: 'image/png'});
        }
    });
}

for (const width of [390, 720]) {
    test(`all Ant size and portal consumers enlarge at ${width}px`, async ({page}, info) => {
        await page.setViewportSize({width, height: 1000});
        await page.goto('/controls.html');
        for (const root of [16, 32]) {
            await page.evaluate(size => (document.documentElement.style.fontSize = `${size}px`), root);
            for (const size of ['small', 'middle', 'large']) {
                const section = page.getByTestId(`size-${size}`);
                const font = size === 'large' ? root : root * 0.875;
                await expect(section.locator('.ant-btn')).toHaveCSS('font-size', `${font}px`);
                await expect(section.locator('.ant-input')).toHaveCSS('font-size', `${font}px`);
                await expect(section.locator('.ant-input-number-input')).toHaveCSS('font-size', `${font}px`);
                await expect(section.locator('.ant-select-content')).toHaveCSS('font-size', `${font}px`);
                await expect(section.locator('td')).toHaveCSS('font-size', `${root * 0.875}px`);
                for (const input of [section.locator('.ant-input'), section.locator('.ant-input-number-input')])
                    expect((await input.boundingBox())!.height).toBeGreaterThanOrEqual(root * 2.75);
            }
            await page.getByRole('button', {name: 'Open dropdown', exact: true}).click();
            await expect(page.getByRole('menuitem', {name: 'Profile option'})).toHaveCSS('font-size', `${root * 0.875}px`);
            await page.keyboard.press('Escape');
            await page.getByRole('button', {name: 'Show message', exact: true}).click();
            await expect(page.getByText('Message body', {exact: true}).last()).toHaveCSS('font-size', `${root * 0.875}px`);
            await page.getByRole('button', {name: 'Show notification', exact: true}).click();
            await expect(page.getByText('Notification title', {exact: true}).last()).toHaveCSS('font-size', `${root}px`);
            await expect(page.getByText('Notification body', {exact: true}).last()).toHaveCSS('font-size', `${root * 0.875}px`);
            await page.getByRole('button', {name: 'Open modal', exact: true}).click();
            const modal = page.getByRole('dialog', {name: 'Modal title'});
            await expect(modal.getByText('Modal title', {exact: true})).toHaveCSS('font-size', `${root * 1.25}px`);
            await expect(modal.getByText('Modal body', {exact: true})).toHaveCSS('font-size', `${root * 0.875}px`);
            await info.attach(`all-consumers-${root}`, {body: await page.screenshot({fullPage: true, animations: 'disabled'}), contentType: 'image/png'});
            await modal.getByRole('button', {name: 'Cancel', exact: true}).click();
        }
    });
}

for (const root of [16, 32]) {
    test(`selection controls retain off and disabled states with root ${root}`, async ({page}, info) => {
        await page.goto('/controls.html');
        await page.evaluate(size => (document.documentElement.style.fontSize = `${size}px`), root);
        const records = [];
        for (const kind of ['Checkbox', 'Radio', 'Switch']) {
            const role = kind.toLowerCase() as 'checkbox' | 'radio' | 'switch';
            const off = page.getByRole(role, {name: kind === 'Radio' ? 'Radio on' : `${kind} toggle`, exact: true});
            await expect(off).not.toBeChecked();
            const sample = async (input: typeof off) => {
                const marker = kind === 'Switch' ? input.locator('.ant-switch-handle') : input.locator('..');
                return marker.evaluate(async (e, kind) => {
                    const surface = e.closest('.ant-switch') || e;
                    getComputedStyle(surface).backgroundColor;
                    await Promise.all(surface.getAnimations({subtree: true}).map(a => a.finished));
                    const pseudo = getComputedStyle(e, kind === 'Switch' ? '::before' : '::after');
                    return {
                        foreground: kind === 'Checkbox' ? pseudo.borderRightColor : pseudo.backgroundColor,
                        background: getComputedStyle(surface).backgroundColor,
                        opacity: pseudo.opacity
                    };
                }, kind);
            };
            const before = await sample(off);
            if (kind === 'Switch') expect(before.foreground).toBe('rgb(255, 255, 255)');
            else expect(before.opacity).toBe('0');
            await off.focus();
            await expect(off).toBeFocused();
            await page.keyboard.press('Space');
            await expect(off).toBeChecked();
            const selected = await sample(off);
            expect(selected.foreground).toBe('rgb(6, 8, 11)');
            expect(selected.opacity).toBe('1');
            if (kind === 'Radio') await page.getByRole('radio', {name: 'Radio off', exact: true}).check();
            else await off.click();
            await expect(off).not.toBeChecked();
            for (const state of ['checked', 'off']) {
                const disabled = page.getByRole(role, {name: `${kind} disabled ${state}`, exact: true});
                await expect(disabled).toBeDisabled();
                await expect(disabled).toBeChecked({checked: state === 'checked'});
                const colors = await sample(disabled);
                expect(colors.foreground).toBe(kind === 'Switch' ? 'rgb(255, 255, 255)' : 'rgb(159, 160, 161)');
                if (kind !== 'Switch') expect(colors.opacity).toBe(state === 'checked' ? '1' : '0');
                const box = (await disabled.boundingBox())!;
                await page.mouse.click(box.x + box.width / 2, box.y + box.height / 2);
                await expect(disabled).toBeChecked({checked: state === 'checked'});
                records.push({kind, state, ...colors});
            }
            records.push({kind, before, selected});
        }
        await info.attach('selection-control-states', {body: JSON.stringify(records), contentType: 'application/json'});
        await info.attach('selection-control-states-visual', {body: await page.getByRole('region', {name: 'Selection control states'}).screenshot(), contentType: 'image/png'});
    });
}
