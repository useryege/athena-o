import {defineConfig} from '@playwright/test';
import fs from 'node:fs';
import path from 'node:path';
import base from './playwright.config';

if (process.env.ATHENA_UI_E2E_MODE !== 'smoke') throw new Error('Worm retirement live acceptance requires smoke mode');
const state = (name: string) => {
    const value = process.env[name];
    if (!value || !path.isAbsolute(value) || !fs.statSync(value).isFile()) throw new Error(`${name} must identify an existing absolute storageState file`);
    return value;
};
export default defineConfig({
    ...base,
    timeout: 90000,
    projects: [
        {
            name: 'worm-retirement-member',
            testMatch: 'worm-trading-retirement.spec.ts',
            grep: /@member/,
            use: {storageState: state('ATHENA_WORM_RETIREMENT_MEMBER_STATE'), launchOptions: {executablePath: process.env.ATHENA_CHROME_PATH || '/usr/bin/google-chrome'}}
        },
        {
            name: 'worm-retirement-admin',
            testMatch: 'worm-trading-retirement.spec.ts',
            grep: /@admin/,
            use: {storageState: state('ATHENA_WORM_RETIREMENT_ADMIN_STATE'), launchOptions: {executablePath: process.env.ATHENA_CHROME_PATH || '/usr/bin/google-chrome'}}
        }
    ]
});
