import {defineConfig} from '@playwright/test';
import path from 'node:path';

const outputRoot = path.resolve(__dirname, '../.tmp/ai-test-tools/visual');

export default defineConfig({
    testDir: './test-tools/visual',
    testMatch: '**/*.spec.ts',
    workers: 1,
    fullyParallel: false,
    retries: 0,
    timeout: 15000,
    updateSnapshots: 'none',
    outputDir: path.join(outputRoot, 'artifacts'),
    reporter: [
        ['list'],
        ['json', {outputFile: path.join(outputRoot, 'results.json')}],
        ['html', {outputFolder: path.join(outputRoot, 'html'), open: 'never'}]
    ],
    use: {
        channel: 'chromium',
        viewport: {width: 800, height: 600},
        colorScheme: 'light',
        locale: 'en-US',
        timezoneId: 'UTC',
        contextOptions: {reducedMotion: 'reduce'},
        trace: 'retain-on-failure',
        screenshot: 'only-on-failure'
    }
});
