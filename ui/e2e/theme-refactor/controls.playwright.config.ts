import {defineConfig} from '@playwright/test';
import path from 'node:path';
const output = path.resolve(__dirname, '../../../.tmp/ui-theme-refactor/t1/browser');
export default defineConfig({
    testDir: __dirname, testMatch: 'controls.spec.ts', workers: 1,
    outputDir: path.join(output, 'artifacts'),
    reporter: [['list'], ['json', {outputFile: path.join(output, 'results.json')}]],
    use: {baseURL: 'http://127.0.0.1:34191', channel: 'chromium', trace: 'retain-on-failure'},
    webServer: {command: 'yarn vite --config e2e/theme-refactor/controls.vite.config.ts', url: 'http://127.0.0.1:34191/controls.html', reuseExistingServer: false}
});
