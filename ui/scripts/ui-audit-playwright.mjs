import {chromium} from '@playwright/test';
import fs from 'fs';
import path from 'path';

const BASE = process.env.ATHENA_UI_URL || 'http://127.0.0.1:4000';
const OUT = process.env.ATHENA_UI_AUDIT_OUT || '/tmp/athena-ui-audit';
const ROUTES = [
    '/',
    '/user-info',
    '/projects',
    '/wallet',
    '/wallet-blacklist',
    '/worm',
    '/polymarket',
    '/polymarket/realtime',
    '/polymarket/movers',
    '/polymarket/sports-live',
    '/notifications',
    '/solidity/bytecodes',
    '/solidity/bytecode-blacklist',
    '/solidity/source-quality/prompts',
    '/settings',
    '/help'
];

fs.mkdirSync(OUT, {recursive: true});

const slug = (route) => (route === '/' ? 'root' : route.replace(/^\//, '').replace(/\//g, '_'));

const report = {
    base: BASE,
    routes: [],
    globalConsoleErrors: []
};

const browser = await chromium.launch({headless: true});
const context = await browser.newContext({viewport: {width: 1280, height: 800}});
const page = await context.newPage();

page.on('console', (msg) => {
    if (msg.type() === 'error') {
        report.globalConsoleErrors.push({text: msg.text(), location: msg.location()});
    }
});

for (const route of ROUTES) {
    const name = slug(route);
    const url = `${BASE}${route}`;
    const entry = {route, url, finalUrl: null, title: null, apiFailures: [], consoleErrors: [], alerts: [], loadingStuck: false, emptyHints: []};

    const routeConsole = [];
    const onConsole = (msg) => {
        if (msg.type() === 'error') routeConsole.push(msg.text());
    };
    page.on('console', onConsole);

    const apiFailures = [];
    page.on('response', (res) => {
        const u = res.url();
        if (!u.includes('/api/')) return;
        const status = res.status();
        if (status >= 400) apiFailures.push({status, url: u});
    });

    try {
        await page.goto(url, {waitUntil: 'networkidle', timeout: 45000});
    } catch (e) {
        entry.navigationError = String(e.message || e);
        try {
            await page.goto(url, {waitUntil: 'domcontentloaded', timeout: 20000});
        } catch (e2) {
            entry.navigationError = String(e2.message || e2);
        }
    }

    await page.waitForTimeout(1500);
    entry.finalUrl = page.url();
    entry.title = await page.title();

    const bodyText = await page.locator('body').innerText().catch(() => '');
    entry.alerts = await page
        .locator('.ant-alert-message, .ant-result-title, .ant-result-subtitle')
        .allTextContents()
        .catch(() => []);
    entry.loadingStuck = /loading/i.test(bodyText) && (await page.locator('.ant-spin').count()) > 0;
    if (/no data|暂无|empty/i.test(bodyText)) entry.emptyHints.push('empty-state-text');
    if ((await page.locator('.ant-empty').count()) > 0) entry.emptyHints.push('ant-empty');

    entry.apiFailures = apiFailures;
    entry.consoleErrors = [...new Set(routeConsole)];

    await page.screenshot({path: path.join(OUT, `${name}.png`), fullPage: true});
    const a11y = await page.locator('body').ariaSnapshot().catch(() => '');
    fs.writeFileSync(path.join(OUT, `${name}.aria.txt`), a11y || '');

    page.off('console', onConsole);
    report.routes.push(entry);
}

await page.setViewportSize({width: 390, height: 844});
await page.goto(`${BASE}/user-info`, {waitUntil: 'networkidle', timeout: 45000}).catch(() => {});
await page.waitForTimeout(1000);
await page.screenshot({path: path.join(OUT, 'user-info-mobile.png'), fullPage: true});
report.mobileUserInfo = {viewport: '390x844', url: page.url()};

fs.writeFileSync(path.join(OUT, 'report.json'), JSON.stringify(report, null, 2));
console.log(JSON.stringify(report, null, 2));
await browser.close();
