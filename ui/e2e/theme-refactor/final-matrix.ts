import {expect, test} from '@playwright/test';
import fs from 'node:fs';
import {createHash} from 'node:crypto';
import path from 'node:path';
import mainCases from './main-cases.json';
import {assertThemeLayout, assertThemeLedger, openThemeCase} from './routes';

export const matrixViewports = [
    {name: 'desktop', width: 1440, height: 900, root: 16},
    {name: 'mobile', width: 390, height: 844, root: 16},
    {name: 'narrow', width: 320, height: 844, root: 16},
    {name: 'text-200', width: 720, height: 1000, root: 32}
];
for (const scenario of mainCases) {
    for (const viewport of matrixViewports) {
        test(`theme:final ${scenario.id} ${viewport.name}`, async ({page}, info) => {
            await page.setViewportSize({width: viewport.width, height: viewport.height});
            if (['worm-preview', 'worm-executions', 'worm-execution-detail'].includes(scenario.id)) await page.clock.setFixedTime(new Date('2026-09-14T08:00:30Z'));
            const errors: string[] = [];
            page.on('pageerror', error => errors.push(error.message));
            const ledger = await openThemeCase(page, scenario.id);
            await page.waitForLoadState('networkidle');
            if (scenario.id === 'member-register' || scenario.id === 'admin-register') {
                await page.getByLabel('Username', {exact: true}).fill(scenario.id === 'admin-register' ? 'admin.alex' : 'alex.chen');
                await expect(page.getByText('Username is available.', {exact: true})).toBeVisible();
            }
            if (scenario.id === 'member-profile' && viewport.name !== 'desktop') await page.getByLabel('Display name', {exact: true}).fill('Alex Chen Research');
            const samples = page
                .locator('h1, .app-page p, .ant-btn, .ant-input, .ant-input-number-input, .ant-select-content, .ant-table-cell, .athena-account-avatar')
                .filter({visible: true});
            const before = await samples.evaluateAll(nodes =>
                nodes.map(node => ({font: parseFloat(getComputedStyle(node).fontSize), text: node.textContent?.trim(), tag: node.tagName}))
            );
            if (viewport.root === 32) {
                await page.evaluate(() => (document.documentElement.style.fontSize = '32px'));
                await expect(page.locator('html')).toHaveCSS('font-size', '32px');
                await expect
                    .poll(async () =>
                        (await samples.evaluateAll(nodes => nodes.map(node => parseFloat(getComputedStyle(node).fontSize)))).map((font, index) =>
                            Math.round((font / before[index].font) * 100)
                        )
                    )
                    .toEqual(before.map(() => 200));
            }
            await page.mouse.move(0, 0);
            await page.evaluate(() => window.scrollTo(0, 0));
            await assertThemeLayout(page);
            // Mobile page copy owns a complete row; its actions follow it. Notifications has an approved title/refresh grid.
            if (viewport.width <= 720 && scenario.id !== 'member-notifications') {
                const header = page.locator('.app-page__header').first();
                const heading = header.locator('.app-page__heading');
                const actions = header.locator('.app-page__actions');
                if ((await heading.count()) && (await actions.locator('button, a').count())) {
                    const copy = (await heading.boundingBox())!;
                    const action = (await actions.boundingBox())!;
                    const frame = (await header.boundingBox())!;
                    expect(copy.width, 'Page title and description retain the full mobile row').toBeGreaterThanOrEqual(frame.width - 4);
                    expect(action.y, 'Page actions follow the complete title and description').toBeGreaterThanOrEqual(copy.y + copy.height);
                }
            }
            const splitActionWords = await page
                .locator('.ant-btn, .radar-windows th, .radar-windows td')
                .filter({visible: true})
                .evaluateAll(buttons => {
                    const split: {label: string; word: string; lines: number}[] = [];
                    for (const button of buttons) {
                        const walker = document.createTreeWalker(button, NodeFilter.SHOW_TEXT);
                        let node: Node | null;
                        while ((node = walker.nextNode())) {
                            for (const match of (node.textContent || '').matchAll(/[+-]?\d+(?:\.\d+)?%?|[A-Za-z]{2,}/g)) {
                                const range = document.createRange();
                                range.setStart(node, match.index!);
                                range.setEnd(node, match.index! + match[0].length);
                                const lines = new Set(Array.from(range.getClientRects(), rect => Math.round(rect.y))).size;
                                if (lines > 1) split.push({label: button.textContent?.trim() || '', word: match[0], lines});
                            }
                        }
                    }
                    return split;
                });
            expect(splitActionWords, 'Action labels and price cells may wrap between words but must preserve whole words and numbers').toEqual([]);
            const overlappingKeyLabels = await page.locator('.account-token-card dl > div').evaluateAll(rows =>
                rows.filter(row => {
                    const label = row.querySelector('dt')!;
                    const value = row.querySelector('dd')!;
                    const range = document.createRange();
                    range.selectNodeContents(label);
                    return range.getBoundingClientRect().right > value.getBoundingClientRect().left;
                }).map(row => row.textContent)
            );
            expect(overlappingKeyLabels, 'API key date labels do not overlap their values after text enlargement').toEqual([]);
            const boundaries = await page
                .locator('.ant-btn, .ant-alert, .athena-account-avatar')
                .filter({visible: true})
                .evaluateAll(nodes =>
                    nodes.map(node => {
                        const element = node as HTMLElement;
                        return {
                            label: element.getAttribute('aria-label') || element.textContent?.trim(),
                            width: element.clientWidth,
                            contentWidth: element.scrollWidth,
                            height: element.clientHeight,
                            contentHeight: element.scrollHeight
                        };
                    })
                );
            expect(
                boundaries.filter(item => item.contentWidth > item.width + 1 || item.contentHeight > item.height + 1),
                'Visible controls, errors and avatars must contain their local content'
            ).toEqual([]);
            const references = scenario.references.map(reference => {
                const digest = createHash('sha256')
                    .update(fs.readFileSync(path.resolve(__dirname, '../../..', reference.file)))
                    .digest('hex');
                expect(digest).toBe(reference.sha256);
                return reference;
            });
            const screenshot = info.outputPath(`${scenario.id}-${viewport.name}.png`);
            await page.screenshot({path: screenshot, fullPage: true, animations: 'disabled'});
            await info.attach(`${scenario.id}-${viewport.name}`, {path: screenshot, contentType: 'image/png'});
            await info.attach('matrix-evidence', {
                body: JSON.stringify({id: scenario.id, viewport, url: page.url(), fontsBefore: before, boundaries, references, errors, requests: ledger.requests}, null, 2),
                contentType: 'application/json'
            });
            expect(errors).toEqual([]);
            assertThemeLedger(ledger);
        });
    }
}
