import {expect, test} from '@playwright/test';
import {assertThemeLedger, openThemeCase} from './routes';

for (const viewport of [
    {width: 390, height: 844, root: 16},
    {width: 320, height: 844, root: 16},
    {width: 390, height: 400, root: 32}
]) {
    test(`theme:confirm profile mobile frame ${viewport.width}x${viewport.height} root${viewport.root}`, async ({page}, info) => {
        await page.setViewportSize(viewport);
        const ledger = await openThemeCase(page, 'member-profile');
        const input = page.getByLabel('Display name', {exact: true});
        await input.fill('Unsaved research display name with additional context');
        await page.evaluate(size => (document.documentElement.style.fontSize = `${size}px`), viewport.root);
        const section = page.getByLabel('Account section');
        await section.click();
        await page.getByText('Access & session', {exact: true}).filter({visible: true}).click();
        const modal = page.getByRole('dialog', {name: 'Discard unsaved profile changes?'});
        await expect(modal).toBeVisible();
        await expect(modal).not.toHaveClass(/ant-zoom-(?:appear|enter)/);
        const box = (await modal.boundingBox())!;
        expect(box.x).toBeCloseTo(20, 0);
        expect(box.width).toBeCloseTo(viewport.width - 40, 0);
        expect(box.y + box.height / 2).toBeCloseTo(viewport.height / 2, 0);
        const keep = modal.getByRole('button', {name: 'Keep editing', exact: true});
        const discard = modal.getByRole('button', {name: 'Discard and leave', exact: true});
        await expect(keep).toBeFocused();
        await page.keyboard.press('Shift+Tab');
        await expect(discard).toBeFocused();
        await page.keyboard.press('Tab');
        await expect(keep).toBeFocused();
        const body = modal.locator('.ant-modal-confirm-body');
        const before = (await keep.boundingBox())!;
        if (viewport.root === 32) {
            expect(await body.evaluate(e => e.scrollHeight > e.clientHeight)).toBe(true);
            await body.evaluate(e => (e.scrollTop = e.scrollHeight));
            expect((await keep.boundingBox())!.y).toBeCloseTo(before.y, 0);
        }
        expect(before.y + before.height).toBeLessThan(viewport.height - 19);
        await info.attach('confirmation-frame', {body: await page.screenshot({animations: 'disabled'}), contentType: 'image/png'});
        await page.keyboard.press('Escape');
        await expect(modal).toBeHidden();
        await expect(section).toBeFocused();
        await expect(input).toHaveValue('Unsaved research display name with additional context');
        assertThemeLedger(ledger);
    });
}
