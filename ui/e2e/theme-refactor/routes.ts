import {expect, type Page, type Route} from '@playwright/test';
import {themeCases} from './cases';
import type {ThemeCase, ThemeLedger, ThemeReply} from './contracts';

const pathPrefix = () => (process.env.ATHENA_UI_E2E_PATH_PREFIX || '').replace(/\/$/, '');
const deploymentPath = (value: string) => `${pathPrefix()}${value.startsWith('/') ? value : `/${value}`}`;
const baseOrigin = () => new URL(process.env.ATHENA_UI_E2E_BASE_URL || 'http://127.0.0.1').origin;
const installedRoutes = new WeakMap<Page, (route: Route) => Promise<void>>();

const bodyOf = (request: ReturnType<Route['request']>) => {
    const text = request.postData();
    if (text === null) return undefined;
    try {
        return JSON.parse(text);
    } catch {
        return text;
    }
};

const relativePath = (pathname: string) => {
    const prefix = pathPrefix();
    if (!prefix) return pathname;
    if (pathname === prefix) return '/';
    return pathname.startsWith(`${prefix}/`) ? pathname.slice(prefix.length) : undefined;
};

const requestRealm = (url: URL, headers: Record<string, string>) => headers['x-athena-application-realm'] || url.searchParams.get('realm') || undefined;

const validateReply = (reply: ThemeReply) => {
    if (!/^(GET|POST|PUT|PATCH|DELETE)$/.test(reply.method) || !reply.path.startsWith('/') || reply.path.includes('?') || reply.path.includes('#')) {
        throw new Error(`Invalid theme reply declaration: ${reply.method} ${reply.path}`);
    }
    if (!Number.isInteger(reply.status) || reply.status < 100 || reply.status > 599) throw new Error(`Invalid theme reply status for ${reply.method} ${reply.path}`);
    if (reply.delayMs !== undefined && (!Number.isFinite(reply.delayMs) || reply.delayMs < 0 || reply.delayMs > 10_000)) {
        throw new Error(`Theme reply delay must be between 0 and 10000ms for ${reply.method} ${reply.path}`);
    }
};

export async function installThemeCase(page: Page, scenario: ThemeCase): Promise<ThemeLedger> {
    if (!scenario.replies.length) throw new Error(`Theme case ${scenario.id} must declare at least one response`);
    scenario.replies.forEach(validateReply);
    const prior = installedRoutes.get(page);
    if (prior) await page.unroute('**/*', prior);
    const ledger: ThemeLedger = {requests: [], unexpected: []};
    const ticketValue = `theme-registration-${scenario.registrationTicket?.realm}`;
    if (scenario.registrationTicket) {
        await page.context().addCookies([
            {
                name: 'athena.registration',
                value: ticketValue,
                domain: new URL(baseOrigin()).hostname,
                path: deploymentPath('/auth/registration'),
                httpOnly: true,
                sameSite: 'Lax'
            }
        ]);
    }
    const handler = async (route: Route) => {
        const request = route.request();
        const url = new URL(request.url());
        const relative = relativePath(url.pathname);
        if (url.origin !== baseOrigin()) {
            if (url.protocol === 'http:' || url.protocol === 'https:') {
                ledger.unexpected.push(`cross-origin request ${request.method()} ${request.url()}`);
                await route.abort('blockedbyclient');
                return;
            }
            await route.continue();
            return;
        }
        if (!relative) {
            if (url.pathname.startsWith('/api/') || url.pathname.startsWith('/auth/')) {
                ledger.unexpected.push(`request escaped deployment prefix ${request.method()} ${url.pathname}${url.search}`);
                await route.abort('blockedbyclient');
                return;
            }
            await route.continue();
            return;
        }
        if (!relative.startsWith('/api/') && !relative.startsWith('/auth/')) {
            await route.continue();
            return;
        }
        // Real registration uses a browser-bound HttpOnly ticket. URL/header claims cannot choose its authority.
        const registration = relative === '/auth/registration' || relative.startsWith('/auth/registration/');
        const cookies = registration ? await page.context().cookies(request.url()) : [];
        const sentHeaders = await request.allHeaders();
        const realm = registration
            ? scenario.registrationTicket &&
              sentHeaders.cookie?.split(';').some(cookie => cookie.trim() === `athena.registration=${ticketValue}`) &&
              cookies.some(cookie => cookie.name === 'athena.registration' && cookie.value === ticketValue && cookie.httpOnly)
                ? scenario.registrationTicket.realm
                : undefined
            : requestRealm(url, request.headers());
        const recorded = {method: request.method(), path: relative, query: url.search, realm, body: bodyOf(request)};
        ledger.requests.push(recorded);
        if (recorded.method === 'GET' && recorded.path === '/api/v1/module-access-states' && (realm === 'member' || realm === 'admin')) {
            await route.fulfill({json: {states: ['trader_sync', 'solana', 'market_radar', 'managed_oo', 'profit_sharing', 'worm'].map(module_key => ({module_key, state: 1}))}});
            return;
        }
        const reply = scenario.replies.find(item => item.method === recorded.method && item.path === recorded.path && item.realm === recorded.realm);
        if (!reply) {
            const message = `unexpected ${recorded.method} ${recorded.path}${recorded.query} realm=${recorded.realm || 'missing'}`;
            ledger.unexpected.push(message);
            await route.fulfill({status: 500, json: {error: {code: 'UNEXPECTED_THEME_REQUEST', message}}});
            return;
        }
        if (reply.delayMs) await new Promise(resolve => setTimeout(resolve, reply.delayMs));
        await route.fulfill({status: reply.status, json: reply.json});
    };
    installedRoutes.set(page, handler);
    await page.route('**/*', handler);
    return ledger;
}

export async function openThemeCase(page: Page, id: string): Promise<ThemeLedger> {
    const scenario = themeCases.find(item => item.id === id);
    if (!scenario) throw new Error(`Unknown theme case: ${id}`);
    const ledger = await installThemeCase(page, scenario);
    await page.goto(deploymentPath(scenario.route));
    await expect(page.getByRole('heading', {level: 1, name: scenario.heading, exact: true})).toBeVisible({timeout: 10_000});
    await page.evaluate(() => document.fonts.ready);
    return ledger;
}

export function assertThemeLedger(ledger: ThemeLedger): void {
    expect(ledger.requests.length, 'theme case must exercise at least one declared request').toBeGreaterThan(0);
    expect(ledger.unexpected, 'theme case issued undeclared, wrong-method, wrong-realm, or cross-origin business requests').toEqual([]);
}

export async function assertThemeLayout(page: Page): Promise<void> {
    const result = await page.evaluate(() => {
        const visible = (element: Element) => {
            const style = getComputedStyle(element);
            return style.display !== 'none' && style.visibility !== 'hidden' && (element as HTMLElement).getClientRects().length > 0;
        };
        const headings = [...document.querySelectorAll('h1')].filter(visible);
        const controls = [...document.querySelectorAll<HTMLElement>('.ant-btn-primary, .login-provider-button, .athena-shell__header button, .app-page__actions .ant-btn')].filter(
            element => visible(element) && !element.matches(':disabled')
        );
        const undersized = controls
            .map(element => ({label: element.getAttribute('aria-label') || element.textContent?.trim() || element.tagName, rect: element.getBoundingClientRect()}))
            .filter(item => item.rect.width < 43.99 || item.rect.height < 43.99)
            .map(item => `${item.label} (${item.rect.width}x${item.rect.height})`);
        const header = document.querySelector<HTMLElement>('.athena-shell__header');
        const pageHeader = document.querySelector<HTMLElement>('.app-page__header');
        const overlaps = Boolean(
            header && pageHeader && visible(header) && visible(pageHeader) && header.getBoundingClientRect().bottom > pageHeader.getBoundingClientRect().top + 1
        );
        return {
            rootOverflow: document.documentElement.scrollWidth - document.documentElement.clientWidth,
            headingCount: headings.length,
            undersized,
            overlaps
        };
    });
    expect(result.rootOverflow, 'document root has horizontal overflow').toBeLessThanOrEqual(1);
    expect(result.headingCount, 'exactly one visible h1 is required').toBe(1);
    expect(result.undersized, 'visible primary and shell controls must be at least 44px in both dimensions').toEqual([]);
    expect(result.overlaps, 'sticky application header overlaps the visible page header').toBe(false);
}
