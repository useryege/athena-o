import {expect, test, type Page, type Request, type Response} from '@playwright/test';

const target = new URL(process.env.ATHENA_UI_E2E_BASE_URL!);
const pathPrefix = (process.env.ATHENA_UI_E2E_PATH_PREFIX || '').replace(/\/$/, '');
const deploymentBase = `${pathPrefix}/`;

type Realm = 'member' | 'admin';

const entryPath = (realm: Realm) => `${pathPrefix}${realm === 'admin' ? '/admin' : ''}/`;
const loginPath = (realm: Realm) => `${pathPrefix}${realm === 'admin' ? '/admin/login' : '/login'}`;
const bootstrapPath = `${pathPrefix}/api/v1/app/bootstrap`;

const readinessTimeoutMs = 15000;
const resourceKinds = new Set(['document', 'script', 'stylesheet', 'font', 'image']);
const resourceQuietWindowMs = 1000;
const resourceSettleTimeoutMs = 5000;

const requiredSameOriginResource = (request: Request) => {
    const url = new URL(request.url());
    return url.origin === target.origin && resourceKinds.has(request.resourceType());
};

const isBootstrapRequest = (request: Request) => {
    const url = new URL(request.url());
    return url.origin === target.origin && url.pathname === bootstrapPath;
};

type BootstrapAttempt = {request: Request; response?: Response; complete: boolean; body?: any; parseError?: string; networkError?: string};

function bootstrapProblem(attempt: BootstrapAttempt | undefined, realm: Realm): string | null {
    if (!attempt) return 'no bootstrap response';
    if (attempt.networkError) return `bootstrap request failed: ${attempt.networkError}`;
    if (!attempt.response) return 'bootstrap response is still pending';
    if (attempt.response.status() !== 200) return `HTTP ${attempt.response.status()}`;
    if (attempt.request.headers()['x-athena-application-realm'] !== realm) return 'bootstrap request realm mismatch';
    if (!attempt.complete) return 'bootstrap response body is still pending';
    if (attempt.parseError) return `bootstrap JSON: ${attempt.parseError}`;
    const bootstrap = attempt.body;
    if (!bootstrap?.settings) return 'bootstrap settings are missing';
    if (
        !['APP_BOOTSTRAP_SESSION_STATUS_ANONYMOUS', 'APP_BOOTSTRAP_SESSION_STATUS_AUTHENTICATED', 'APP_BOOTSTRAP_SESSION_STATUS_ACCOUNT_MAINTENANCE'].includes(
            bootstrap?.session?.status
        )
    ) {
        return 'invalid bootstrap session status';
    }
    if (bootstrap.session.status === 'APP_BOOTSTRAP_SESSION_STATUS_AUTHENTICATED') {
        const userInfo = bootstrap.session.userInfo || bootstrap.session.user_info;
        if (!userInfo) return 'authenticated bootstrap user info is missing';
        if (Boolean(userInfo.administrator) !== (realm === 'admin')) return 'bootstrap role mismatch';
    }
    return null;
}

async function waitForRequiredResources(page: Page, pending: Set<Request>, lastActivity: () => number, shellReadyAt: number) {
    await page.waitForLoadState('load', {timeout: resourceSettleTimeoutMs});
    try {
        await expect
            .poll(() => pending.size === 0 && Date.now() - Math.max(shellReadyAt, lastActivity()) >= resourceQuietWindowMs, {
                message: 'required same-origin resources finish and remain quiet after the application shell is ready',
                timeout: resourceSettleTimeoutMs,
                intervals: [50, 100, 250]
            })
            .toBe(true);
    } catch (error) {
        const urls = [...pending].map(request => `${request.resourceType()} ${request.url()}`);
        throw new Error(`Required same-origin resources did not settle within ${resourceSettleTimeoutMs}ms${urls.length ? `:\n${urls.join('\n')}` : ''}`, {cause: error});
    }
}

async function applicationStateReady(page: Page, realm: Realm, bootstrap: any) {
    if (bootstrap.session.status === 'APP_BOOTSTRAP_SESSION_STATUS_AUTHENTICATED') {
        return page.getByRole('navigation', {name: realm === 'admin' ? 'Administration navigation' : 'Primary navigation'}).isVisible();
    }
    const loginURL = new RegExp(`${loginPath(realm).replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}(?:[?#]|$)`);
    if (!loginURL.test(page.url())) return false;
    const heading = page.getByRole('heading', {name: realm === 'admin' ? 'Athena Admin' : 'Athena', exact: true});
    const methods = realm === 'admin' ? page.getByRole('button', {name: 'Continue with Google', exact: true}) : page.getByRole('group', {name: 'Sign-in methods'});
    return (await heading.isVisible()) && (await methods.isVisible());
}

async function waitForBootstrapReady(page: Page, realm: Realm, attempts: BootstrapAttempt[], deadline: number) {
    let problem = 'no bootstrap response';
    try {
        await expect
            .poll(
                async () => {
                    const latest = attempts[attempts.length - 1];
                    problem = bootstrapProblem(latest, realm);
                    if (problem) return false;
                    // Navigation can replace the document between observations, particularly at admin login.
                    try {
                        const ready = (await applicationStateReady(page, realm, latest.body)) && (await page.locator('#app > *').count()) > 0;
                        problem = ready ? '' : 'application shell does not match bootstrap yet';
                        return ready && latest === attempts[attempts.length - 1];
                    } catch (error) {
                        problem = error instanceof Error ? error.message : String(error);
                        return false;
                    }
                },
                {message: 'bootstrap and application shell recover within the readiness deadline', timeout: Math.max(1, deadline - Date.now()), intervals: [50, 100, 250]}
            )
            .toBe(true);
    } catch (error) {
        const latest = attempts[attempts.length - 1];
        throw new Error(`Bootstrap readiness failed within ${readinessTimeoutMs}ms; latest HTTP ${latest?.response?.status() ?? 'none'}: ${problem}`, {cause: error});
    }
}

async function probeApplicationShell(page: Page, realm: Realm) {
    const pageErrors: string[] = [];
    const resourceFailures: string[] = [];
    const identityProviderRequests: string[] = [];
    const bootstrapAttempts: BootstrapAttempt[] = [];
    const bootstrapIdentityFailures: string[] = [];
    const pendingResources = new Set<Request>();
    let lastResourceActivity = Date.now();
    const context = page.context();

    const observePage = (observed: Page) => observed.on('pageerror', error => pageErrors.push(error.message));
    observePage(page);
    context.on('page', observePage);
    await context.route('**/*', async route => {
        const request = route.request();
        const url = new URL(request.url());
        const authenticationPath = url.origin === target.origin && (url.pathname === `${pathPrefix}/auth` || url.pathname.startsWith(`${pathPrefix}/auth/`));
        const externalDocument = url.origin !== target.origin && request.resourceType() === 'document';
        if (authenticationPath || externalDocument) {
            identityProviderRequests.push(request.url());
            await route.abort('blockedbyclient');
            return;
        }
        await route.continue();
    });
    context.on('request', request => {
        if (isBootstrapRequest(request)) {
            bootstrapAttempts.push({request, complete: false});
            if (request.headers()['x-athena-application-realm'] !== realm) bootstrapIdentityFailures.push('bootstrap request realm mismatch');
        }
        if (requiredSameOriginResource(request) || isBootstrapRequest(request)) {
            pendingResources.add(request);
            lastResourceActivity = Date.now();
        }
    });
    context.on('requestfinished', request => {
        if (pendingResources.delete(request)) {
            lastResourceActivity = Date.now();
        }
    });
    context.on('requestfailed', request => {
        if (pendingResources.delete(request)) lastResourceActivity = Date.now();
        const attempt = bootstrapAttempts.find(item => item.request === request);
        if (attempt) {
            attempt.networkError = request.failure()?.errorText || 'request failed';
            attempt.complete = true;
        }
        if (requiredSameOriginResource(request)) {
            resourceFailures.push(`${request.resourceType()} ${request.url()}: ${request.failure()?.errorText || 'request failed'}`);
        }
    });
    context.on('response', response => {
        const url = new URL(response.url());
        const attempt = bootstrapAttempts.find(item => item.request === response.request());
        if (attempt) {
            attempt.response = response;
            attempt.complete = response.status() !== 200;
            if (response.status() === 200) {
                void response
                    .json()
                    .then(body => {
                        attempt.body = body;
                        const userInfo = body?.session?.userInfo || body?.session?.user_info;
                        if (body?.session?.status === 'APP_BOOTSTRAP_SESSION_STATUS_AUTHENTICATED' && userInfo && Boolean(userInfo.administrator) !== (realm === 'admin')) {
                            bootstrapIdentityFailures.push('bootstrap role mismatch');
                        }
                    })
                    .catch(error => {
                        attempt.parseError = error instanceof Error ? error.message : String(error);
                    })
                    .finally(() => {
                        attempt.complete = true;
                    });
            }
        }
        if (url.origin === target.origin && resourceKinds.has(response.request().resourceType()) && response.status() >= 400) {
            resourceFailures.push(`${response.request().resourceType()} ${response.url()}: HTTP ${response.status()}`);
        }
    });

    try {
        const readinessDeadline = Date.now() + readinessTimeoutMs;
        const response = await page.goto(new URL(entryPath(realm), target.origin).href, {waitUntil: 'domcontentloaded', timeout: readinessTimeoutMs});
        expect(response, 'main document response').not.toBeNull();
        expect(response!.status(), 'main document status').toBe(200);
        expect(response!.headers()['content-type'] || '', 'main document content type').toContain('text/html');

        await waitForBootstrapReady(page, realm, bootstrapAttempts, readinessDeadline);

        await expect(page.locator('base')).toHaveAttribute('href', entryPath(realm));
        await expect(page.locator('meta[name="athena-deployment-base-href"]')).toHaveAttribute('content', deploymentBase);

        const shellReadyAt = Date.now();
        await waitForRequiredResources(page, pendingResources, () => lastResourceActivity, shellReadyAt);
        // Initial readiness owns the 15-second budget. The resource quiet window can end later;
        // verify the latest state once without giving the application another retry budget.
        const latest = bootstrapAttempts[bootstrapAttempts.length - 1];
        expect(bootstrapProblem(latest, realm), 'latest bootstrap after resources settle').toBeNull();
        expect(await applicationStateReady(page, realm, latest.body), 'latest bootstrap matches the application shell').toBe(true);
        expect(await page.locator('#app > *').count(), 'application remains mounted').toBeGreaterThan(0);
        expect(bootstrapIdentityFailures, 'bootstrap realm and role boundaries').toEqual([]);

        const loadedResources = await page.evaluate(() =>
            performance
                .getEntriesByType('resource')
                .map(entry => entry.name)
                .filter(name => {
                    const url = new URL(name);
                    return url.origin === location.origin && /\.(?:css|js|mjs)(?:$|\?)/.test(url.pathname + url.search);
                })
        );
        expect(loadedResources.length, 'same-origin script or stylesheet loaded').toBeGreaterThan(0);
        expect(resourceFailures, 'required same-origin resources').toEqual([]);
        expect(pageErrors, 'uncaught browser errors').toEqual([]);
        expect(identityProviderRequests, 'smoke must not start an external identity-provider flow').toEqual([]);
    } finally {
        await test.info().attach('bootstrap-attempts', {
            body: Buffer.from(
                JSON.stringify(
                    bootstrapAttempts.map(attempt => ({
                        url: attempt.request.url(),
                        status: attempt.response?.status(),
                        realm: attempt.request.headers()['x-athena-application-realm'],
                        complete: attempt.complete,
                        parseError: attempt.parseError,
                        networkError: attempt.networkError,
                        sessionStatus: attempt.body?.session?.status
                    })),
                    null,
                    2
                )
            ),
            contentType: 'application/json'
        });
    }
}

test('member application entry, bootstrap and session shell are healthy', async ({page}) => {
    await probeApplicationShell(page, 'member');
});

test('administrator application entry, bootstrap and session shell are healthy', async ({page}) => {
    await probeApplicationShell(page, 'admin');
});
