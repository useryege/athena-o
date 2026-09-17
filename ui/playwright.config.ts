import {defineConfig, type PlaywrightTestConfig} from '@playwright/test';
import path from 'node:path';

type AcceptanceMode = 'isolated' | 'smoke';

const mode = (process.env.ATHENA_UI_E2E_MODE || 'isolated') as AcceptanceMode;
if (mode !== 'isolated' && mode !== 'smoke') {
    throw new Error(`ATHENA_UI_E2E_MODE must be isolated or smoke, got ${mode}`);
}
const suite = process.env.UI_ACCEPTANCE_SUITE || 'acceptance';
if (suite !== 'acceptance' && suite !== 'a11y') {
    throw new Error(`UI_ACCEPTANCE_SUITE must be acceptance or a11y, got ${suite}`);
}
if (suite === 'a11y' && mode !== 'isolated') {
    throw new Error('The a11y suite requires isolated UI acceptance mode');
}

const baseURL = process.env.ATHENA_UI_E2E_BASE_URL;
if (!baseURL) {
    throw new Error('ATHENA_UI_E2E_BASE_URL is required');
}
const target = new URL(baseURL);
if (target.protocol !== 'http:' || target.username || target.password || target.search || target.hash) {
    throw new Error('UI acceptance target must be a plain HTTP URL without credentials, query, or fragment');
}
const loopbackHosts = new Set(['localhost', '127.0.0.1', '[::1]']);
if (!loopbackHosts.has(target.hostname)) {
    throw new Error('UI acceptance target must use a loopback hostname');
}
if (mode === 'isolated' && target.hostname !== '127.0.0.1') {
    throw new Error('Isolated UI acceptance requires a 127.0.0.1 harness URL');
}

const pathPrefix = process.env.ATHENA_UI_E2E_PATH_PREFIX || '';
if (pathPrefix && (!pathPrefix.startsWith('/') || pathPrefix.endsWith('/'))) {
    throw new Error('ATHENA_UI_E2E_PATH_PREFIX must be empty or a leading-slash path without a trailing slash');
}

const outputRoot = process.env.ATHENA_UI_E2E_OUTPUT_DIR;
if (!outputRoot || !path.isAbsolute(outputRoot)) {
    throw new Error('ATHENA_UI_E2E_OUTPUT_DIR must be an absolute path');
}

const commonUse: PlaywrightTestConfig['use'] = {
    baseURL,
    viewport: {width: 1440, height: 900},
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure'
};

export default defineConfig({
    testDir: './e2e',
    workers: 1,
    fullyParallel: false,
    retries: 0,
    timeout: 45000,
    outputDir: path.join(outputRoot, 'artifacts'),
    reporter: [['list'], ['json', {outputFile: path.join(outputRoot, 'results.json')}], ['html', {outputFolder: path.join(outputRoot, 'html'), open: 'never'}]],
    use: commonUse,
    projects:
        mode === 'smoke'
            ? [
                  {
                      name: 'smoke',
                      testMatch: 'shell-smoke.spec.ts',
                      use: {launchOptions: {executablePath: process.env.ATHENA_CHROME_PATH || '/usr/bin/google-chrome'}}
                  }
              ]
            : suite === 'a11y'
              ? [{name: 'a11y', testMatch: /(?:trader-sync-a11y|theme-refactor-a11y)\.spec\.ts$/, use: {channel: 'chromium'}}]
              : [
                    {name: 'ui-fixtures', testMatch: /(?:trader-sync|theme-refactor|module-removal|module-access)\.spec\.ts$/, use: {channel: 'chromium'}},
                    {name: 'live', testMatch: 'trader-sync-live.spec.ts', use: {channel: 'chromium'}}
                ]
});
