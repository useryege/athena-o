import {expect, test} from '@playwright/test';
import {themeCases} from './cases';
import {assertThemeLedger, installThemeCase} from './routes';

for (const realm of ['member', 'admin'] as const) {
    test(`theme:registration ${realm} canonical ticket expires back to its own login`, async ({page}) => {
        const scenario = structuredClone(themeCases.find(item => item.id === `${realm}-register`)!);
        scenario.route = `/register?athenaRealm=${realm}`;
        scenario.replies.find(item => item.path === '/auth/registration')!.json = {reason: 'registration_expired'};
        scenario.replies.find(item => item.path === '/auth/registration')!.status = 410;
        scenario.replies.push(...structuredClone(themeCases.find(item => item.id === `${realm}-login`)!.replies));
        const ledger = await installThemeCase(page, scenario);
        const prefix = process.env.ATHENA_UI_E2E_PATH_PREFIX || '';
        await page.goto(`${prefix}${scenario.route}`);
        await expect(page.getByText('Your registration session has expired. Return to sign in and try again.', {exact: true})).toBeVisible();
        await page.getByRole('button', {name: 'Return to sign in', exact: true}).click();
        await expect(page).toHaveURL(new RegExp(`${prefix}${realm === 'admin' ? '/admin' : ''}/login(?:\\?.*)?$`));
        await expect(page.getByRole('heading', {level: 1, name: realm === 'admin' ? 'Athena Admin' : 'Welcome to Athena', exact: true})).toBeVisible();
        assertThemeLedger(ledger);
    });
}
