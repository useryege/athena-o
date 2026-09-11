import {defineConfig} from '@playwright/test';
const baseURL = process.env.ATHENA_UI_E2E_BASE_URL;
if (!baseURL) throw new Error('ATHENA_UI_E2E_BASE_URL is required');
const target = new URL(baseURL);
if (target.protocol !== 'http:' || target.hostname !== '127.0.0.1') throw new Error('UI acceptance requires the explicitly selected loopback harness');
export default defineConfig({
    testDir: './e2e',
    workers: 1,
    fullyParallel: false,
    timeout: 45000,
    outputDir: '../.superpowers/trader-sync-acceptance/playwright',
    reporter: [['list']],
    use: {baseURL, trace: 'retain-on-failure', screenshot: 'only-on-failure', launchOptions: {executablePath: process.env.ATHENA_CHROME_PATH || '/usr/bin/google-chrome'}},
    projects: [
        {name: 'ui-fixtures', testMatch: 'trader-sync.spec.ts'},
        {name: 'live', testMatch: 'trader-sync-live.spec.ts'}
    ]
});
