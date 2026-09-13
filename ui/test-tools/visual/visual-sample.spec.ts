import {expect, test} from '@playwright/test';

test('renders a fixed visual sample', async ({page}) => {
    await page.setContent(`
        <style>
            *, *::before, *::after { animation: none !important; transition: none !important; }
            body { margin: 0; background: #f4f7fb; color: #162033; font-family: Arial, sans-serif; }
            #visual-sample { box-sizing: border-box; width: 640px; height: 320px; margin: 48px auto; padding: 32px; background: #ffffff; border: 1px solid #d8e1ef; border-radius: 16px; }
            .eyebrow { margin: 0 0 12px; color: #49637f; font-size: 12px; font-weight: 700; letter-spacing: 0.08em; text-transform: uppercase; }
            h1 { margin: 0; font-size: 28px; line-height: 36px; }
            p { margin: 12px 0 24px; color: #52647a; font-size: 16px; line-height: 24px; }
            .summary { display: flex; align-items: center; justify-content: space-between; padding: 16px; background: #edf5ff; border-radius: 10px; }
            .summary strong { font-size: 20px; }
            .status { padding: 6px 10px; color: #15613d; background: #d8f4e5; border-radius: 999px; font-size: 13px; font-weight: 700; }
        </style>
        <main id="visual-sample">
            <p class="eyebrow">Test tools</p>
            <h1>Deterministic visual sample</h1>
            <p>Static content at 2026-09-13T00:00:00Z.</p>
            <section class="summary" aria-label="Sample summary">
                <strong>42 checks</strong>
                <span class="status">Ready</span>
            </section>
        </main>
    `);

    await expect(page.locator('#visual-sample')).toHaveScreenshot('visual-sample.png', {
        animations: 'disabled',
        caret: 'hide',
        scale: 'css'
    });
});
