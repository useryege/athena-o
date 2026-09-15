import {defineConfig} from '@playwright/test';
import path from 'node:path';
const output = process.env.THEME_CONTROLS_OUTPUT || path.resolve(__dirname, '../../../.tmp/ui-theme-refactor/controls');
export default defineConfig({
    testDir: __dirname,
    testMatch: 'controls.spec.ts',
    workers: 1,
    retries: 0,
    use: {baseURL: 'http://127.0.0.1:34191', viewport: {width: 1440, height: 1000}, channel: 'chromium', trace: 'retain-on-failure'},
    outputDir: path.join(output, 'artifacts'),
    reporter: [['list'], ['json', {outputFile: path.join(output, 'results.json')}]],
    webServer: process.env.THEME_CONTROLS_EXTERNAL === '1' ? undefined : {
        command: 'yarn vite --config e2e/theme-refactor/controls.vite.config.ts',
        url: 'http://127.0.0.1:34191/controls.html',
        reuseExistingServer: false
    }
});
