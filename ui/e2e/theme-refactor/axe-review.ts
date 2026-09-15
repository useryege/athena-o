import {expect, type Page, type TestInfo} from '@playwright/test';
import type {AxeResults} from 'axe-core';
import fs from 'node:fs';

// Preserve measurements for manual checks; these never convert incomplete rules into passes.
export async function recordAxeReview(page: Page, results: AxeResults, info: TestInfo) {
    expect(
        results.incomplete
            .filter(rule => ['aria-prohibited-attr', 'form-field-multiple-labels', 'th-has-data-cells'].includes(rule.id))
            .map(rule => ({id: rule.id, nodes: rule.nodes.map(node => node.target)})),
        'Known semantic review findings must remain fixed'
    ).toEqual([]);
    if (!results.incomplete.length) return;
    const targets = results.incomplete.flatMap(rule => rule.nodes.map(node => ({rule: rule.id, target: node.target, html: node.html})));
    const measurements = await page.evaluate(items => {
        const canvas = document.createElement('canvas');
        canvas.width = canvas.height = 1;
        const context = canvas.getContext('2d')!;
        const rgba = (color: string) => {
            context.clearRect(0, 0, 1, 1);
            context.fillStyle = color;
            context.fillRect(0, 0, 1, 1);
            return [...context.getImageData(0, 0, 1, 1).data];
        };
        const luminance = (color: number[]) =>
            color
                .slice(0, 3)
                .map(c => c / 255)
                .map(c => (c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4))
                .reduce((s, c, i) => s + c * [0.2126, 0.7152, 0.0722][i], 0);
        return items.map(item => {
            const selector = item.target.length === 1 && typeof item.target[0] === 'string' ? item.target[0] : null;
            const element = selector ? document.querySelector(selector) : null;
            if (!element) return {...item, unavailable: true};
            const style = getComputedStyle(element);
            const box = element.getBoundingClientRect();
            const ancestors = [];
            for (let parent: Element | null = element; parent; parent = parent.parentElement) {
                const computed = getComputedStyle(parent);
                ancestors.push({
                    tag: parent.tagName,
                    className: parent.className,
                    background: computed.backgroundColor,
                    backgroundImage: computed.backgroundImage,
                    opacity: computed.opacity,
                    color: computed.color
                });
            }
            let background = [0, 0, 0];
            for (const ancestor of [...ancestors].reverse()) {
                const color = rgba(ancestor.background);
                background = background.map((c, i) => (color[i] * color[3]) / 255 + c * (1 - color[3] / 255));
            }
            const fg = rgba(style.color);
            const foreground = background.map((c, i) => (fg[i] * fg[3]) / 255 + c * (1 - fg[3] / 255));
            const a = luminance(foreground),
                b = luminance(background);
            const centerX = box.x + box.width / 2,
                centerY = box.y + box.height / 2;
            return {
                ...item,
                color: style.color,
                fontSize: style.fontSize,
                textDecoration: style.textDecorationLine,
                background,
                foreground,
                ancestorContrast: (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05),
                ancestors,
                box: box.toJSON(),
                centerStack: document
                    .elementsFromPoint(centerX, centerY)
                    .slice(0, 6)
                    .map(e => ({tag: e.tagName, className: e.className, text: e.textContent?.slice(0, 100)}))
            };
        });
    }, targets);
    fs.writeFileSync(
        info.outputPath('axe-manual-review.json'),
        JSON.stringify({note: 'Ancestor composite contrast is a measurement, not a pass: overlap, image, opacity and visibility require manual review.', measurements}, null, 2)
    );
    await page.screenshot({path: info.outputPath('axe-review-context.png'), fullPage: false});
}
