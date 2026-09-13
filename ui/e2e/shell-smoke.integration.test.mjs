import assert from 'node:assert/strict';
import {mkdtemp, readFile, readdir, rm, stat} from 'node:fs/promises';
import http from 'node:http';
import os from 'node:os';
import path from 'node:path';
import {spawn} from 'node:child_process';
import test from 'node:test';

const uiDir = path.resolve(import.meta.dirname, '..');
const playwright = path.join(uiDir, 'node_modules', '@playwright', 'test', 'cli.js');
const chrome = process.env.ATHENA_CHROME_PATH || '/usr/bin/google-chrome';
const node = process.execPath;

const run = (args, options = {}) =>
    new Promise(resolve => {
        const env = {...process.env, ...options.env, FORCE_COLOR: '0'};
        delete env.NO_COLOR;
        const child = spawn(node, args, {
            cwd: uiDir,
            env,
            stdio: ['ignore', 'pipe', 'pipe']
        });
        let stdout = '';
        let stderr = '';
        child.stdout.on('data', chunk => (stdout += chunk));
        child.stderr.on('data', chunk => (stderr += chunk));
        child.on('close', code => resolve({code, stdout, stderr}));
    });

const makeOutput = async name => mkdtemp(path.join(os.tmpdir(), `athena-${name}-`));

const playwrightEnv = (baseURL, outputDir, extra = {}) => ({
    ATHENA_UI_E2E_MODE: 'smoke',
    ATHENA_UI_E2E_BASE_URL: baseURL,
    ATHENA_UI_E2E_PATH_PREFIX: '',
    ATHENA_UI_E2E_OUTPUT_DIR: outputDir,
    ATHENA_CHROME_PATH: chrome,
    ...extra
});

const runPlaywright = (env, args = []) => run([playwright, 'test', '--project=smoke', ...args], {env});

const settings = {
    url: 'http://127.0.0.1',
    statusBadgeEnabled: false,
    statusBadgeRootUrl: '',
    googleAnalytics: {},
    help: {chatUrl: '', chatText: '', binaryUrls: {}},
    userLoginsDisabled: false,
    kustomizeVersions: [],
    uiCssURL: '',
    uiBannerContent: '',
    uiBannerURL: '',
    uiBannerPermanent: false,
    uiBannerPosition: '',
    execEnabled: false,
    appsInAnyNamespaceEnabled: false,
    hydratorEnabled: false,
    syncWithReplaceAllowed: false
};

const userInfo = administrator => ({
    loggedIn: true,
    accountId: administrator ? '22222222-2222-4222-8222-222222222222' : '11111111-1111-4111-8111-111111111111',
    username: administrator ? 'local-admin' : 'local-user',
    iss: 'controlled-smoke',
    administrator,
    access: {loginEnabled: true, apiKeyEnabled: false, profitSharingEnabled: false, revision: '1', moduleAccess: []},
    identity: {provider: 'ACCOUNT_IDENTITY_PROVIDER_DEVELOPMENT', verifiedEmail: '', solanaAddress: '', createdAt: '1', lastLoginAt: '1'},
    profile: {displayName: administrator ? 'Local Administrator' : 'Local Member', tier: 'ACCOUNT_TIER_STANDARD', avatarUrl: '', revision: '1'},
    preferences: {theme: 'ACCOUNT_THEME_MODE_SYSTEM', revision: '1'}
});

const fixtureScript = (scenario, prefix) => `
(async () => {
  const admin = location.pathname.startsWith('${prefix}/admin');
  const realm = admin ? 'admin' : 'member';
  let bootstrap;
  const delays = [500, 1000, 2000, 3000];
  for (let attempt = 0; ; attempt++) {
    try {
      const response = await fetch('${prefix}/api/v1/app/bootstrap', {headers: {'X-Athena-Application-Realm': ${scenario === 'wrong-realm' ? "admin ? 'member' : 'admin'" : 'realm'}}});
      if (!response.ok) throw new Error('bootstrap HTTP ' + response.status);
      bootstrap = await response.json();
      if (!bootstrap.settings || !bootstrap.session || !bootstrap.session.status) throw new Error('invalid bootstrap response');
      break;
    } catch (error) {
      if (!'${scenario}'.startsWith('recover-') || attempt === delays.length) throw error;
      await new Promise(resolve => setTimeout(resolve, delays[attempt]));
    }
  }
  ${scenario === 'near-deadline' ? 'await new Promise(resolve => setTimeout(resolve, 14200));' : ''}
  const status = bootstrap.session.status;
  const authenticated = status === 'APP_BOOTSTRAP_SESSION_STATUS_AUTHENTICATED';
  const root = document.querySelector('#app');
  if (authenticated) {
    const user = bootstrap.session.userInfo || bootstrap.session.user_info;
    if (!user) throw new Error('missing authenticated user info');
    const nav = document.createElement('nav');
    nav.setAttribute('aria-label', admin ? 'Administration navigation' : 'Primary navigation');
    nav.textContent = admin ? 'Athena Admin' : 'Athena';
    root.append(nav);
    const heading = document.createElement('h1');
    heading.textContent = 'Trader Sync';
    root.append(heading);
  } else {
    history.replaceState(null, '', admin ? '${prefix}/admin/login' : '${prefix}/login');
    root.innerHTML = admin
      ? '<main><h1>Athena Admin</h1><button>Continue with Google</button></main>'
      : '<main><h1>Athena</h1><div role="group" aria-label="Sign-in methods"><button>Continue with Google</button><button>Continue with Phantom</button></div></main>';
  }
  ${
      scenario === 'slow-resource-404'
          ? `const link = document.createElement('link');
  link.rel = 'stylesheet';
  link.href = '${prefix}/assets/missing.css';
  document.head.append(link);`
          : ''
  }
  ${scenario === 'pending-bootstrap-error' ? `setTimeout(() => fetch('${prefix}/api/v1/app/bootstrap', {headers: {'X-Athena-Application-Realm': realm}}), 200);` : ''}
  ${scenario === 'auth-popup' ? `window.open('${prefix}/auth/google/login', '_blank');` : ''}
  ${scenario === 'pageerror' ? "setTimeout(() => { throw new Error('controlled delayed pageerror'); }, 200);" : ''}
})().catch(error => { setTimeout(() => { throw error; }); });
`;

async function startFixture(scenario, prefix = '') {
    let authHandlerHits = 0;
    const bootstrapRequests = {member: 0, admin: 0};
    const server = http.createServer((request, response) => {
        const url = new URL(request.url, 'http://127.0.0.1');
        if (url.pathname === `${prefix}/auth/google/login`) {
            authHandlerHits++;
            response.writeHead(200, {'content-type': 'text/html'}).end('<!doctype html><title>controlled identity provider</title>');
            return;
        }
        if (url.pathname === `${prefix}/api/v1/app/bootstrap`) {
            const realm = request.headers['x-athena-application-realm'];
            const attempt = ++bootstrapRequests[realm];
            if (scenario === 'pending-bootstrap-error' && attempt === 2) {
                setTimeout(() => response.writeHead(503, {'content-type': 'application/json'}).end(JSON.stringify({message: 'controlled late unavailable'})), 2200);
                return;
            }
            if (scenario === 'recover-json' && attempt === 1) {
                response.writeHead(200, {'content-type': 'application/json'}).end('{invalid-json');
                return;
            }
            if (scenario === 'bootstrap-error' || (scenario === 'recover-once' && attempt === 1) || (scenario === 'recover-many' && attempt <= 3)) {
                response.writeHead(503, {'content-type': 'application/json'}).end(JSON.stringify({message: 'controlled unavailable'}));
                return;
            }
            const body =
                scenario === 'invalid-bootstrap'
                    ? {settings}
                    : scenario === 'anonymous' || scenario === 'resource-404' || scenario === 'pageerror'
                      ? {settings, session: {status: 'APP_BOOTSTRAP_SESSION_STATUS_ANONYMOUS'}}
                      : {
                            settings,
                            session: {
                                status: 'APP_BOOTSTRAP_SESSION_STATUS_AUTHENTICATED',
                                userInfo: userInfo(scenario === 'wrong-role' ? realm !== 'admin' : realm === 'admin')
                            }
                        };
            response.writeHead(200, {'content-type': 'application/json'}).end(JSON.stringify(body));
            return;
        }
        if (url.pathname === `${prefix}/assets/app.js`) {
            response.writeHead(200, {'content-type': 'text/javascript'}).end(fixtureScript(scenario, prefix));
            return;
        }
        if (url.pathname === `${prefix}/assets/missing.css`) {
            const reply = () => response.writeHead(404, {'content-type': 'text/css'}).end('missing');
            if (scenario === 'slow-resource-404') setTimeout(reply, 2200);
            else reply();
            return;
        }
        const isAdmin = url.pathname === `${prefix}/admin/` || url.pathname === `${prefix}/admin` || url.pathname === `${prefix}/admin/login`;
        if (url.pathname === `${prefix}/` || url.pathname === `${prefix}/login` || isAdmin) {
            const base = isAdmin ? `${prefix}/admin/` : `${prefix}/`;
            const missing = scenario === 'resource-404' ? `<link rel="stylesheet" href="${prefix}/assets/missing.css">` : '';
            response
                .writeHead(200, {'content-type': 'text/html'})
                .end(
                    `<!doctype html><html><head><base href="${base}"><meta name="athena-deployment-base-href" content="${prefix}/">${missing}</head><body><div id="app"></div><script defer src="${prefix}/assets/app.js"></script></body></html>`
                );
            return;
        }
        response.writeHead(404).end('not found');
    });
    await new Promise((resolve, reject) => {
        server.once('error', reject);
        server.listen(0, '127.0.0.1', resolve);
    });
    const address = server.address();
    return {
        baseURL: `http://127.0.0.1:${address.port}`,
        authHandlerHits: () => authHandlerHits,
        bootstrapRequests,
        close: () => new Promise((resolve, reject) => server.close(error => (error ? reject(error) : resolve())))
    };
}

async function filesBelow(directory) {
    const entries = await readdir(directory, {withFileTypes: true});
    const files = [];
    for (const entry of entries) {
        const value = path.join(directory, entry.name);
        if (entry.isDirectory()) files.push(...(await filesBelow(value)));
        else files.push(value);
    }
    return files;
}

test('smoke mode collects only the shell smoke spec without a live manifest', async t => {
    const outputDir = await makeOutput('smoke-list');
    t.after(() => rm(outputDir, {recursive: true, force: true}));
    const result = await runPlaywright(playwrightEnv('http://localhost:4000', outputDir), ['--list']);
    assert.equal(result.code, 0, result.stdout + result.stderr);
    assert.match(result.stdout, /shell-smoke\.spec\.ts/);
    assert.doesNotMatch(result.stdout + result.stderr, /trader-sync-live/);
});

test('smoke rejects non-loopback targets and isolated mode rejects localhost', async t => {
    const outputDir = await makeOutput('target-validation');
    t.after(() => rm(outputDir, {recursive: true, force: true}));
    const remote = await runPlaywright(playwrightEnv('http://example.com', outputDir), ['--list']);
    assert.notEqual(remote.code, 0);
    assert.match(remote.stdout + remote.stderr, /loopback hostname/);
    const isolated = await runPlaywright(
        {
            ...playwrightEnv('http://localhost:4000', outputDir),
            ATHENA_UI_E2E_MODE: 'isolated',
            ATHENA_UI_E2E_MANIFEST: path.join(outputDir, 'unused-manifest.json')
        },
        ['--list']
    );
    assert.notEqual(isolated.code, 0);
    assert.match(isolated.stdout + isolated.stderr, /127\.0\.0\.1 harness/);
});

for (const scenario of ['authenticated', 'anonymous']) {
    test(`real browser accepts the controlled ${scenario} member and admin shells`, async t => {
        const fixture = await startFixture(scenario);
        const outputDir = await makeOutput(`smoke-${scenario}`);
        t.after(async () => {
            await fixture.close();
            await rm(outputDir, {recursive: true, force: true});
        });
        const result = await runPlaywright(playwrightEnv(fixture.baseURL, outputDir));
        assert.equal(result.code, 0, result.stdout + result.stderr);
        assert.match(result.stdout, /2 passed/);
        await stat(path.join(outputDir, 'results.json'));
        await stat(path.join(outputDir, 'html', 'index.html'));
    });
}

test('real browser verifies both application bases under a deployment prefix', async t => {
    const fixture = await startFixture('anonymous', '/athena');
    const outputDir = await makeOutput('smoke-prefix');
    t.after(async () => {
        await fixture.close();
        await rm(outputDir, {recursive: true, force: true});
    });
    const result = await runPlaywright(playwrightEnv(`${fixture.baseURL}/athena`, outputDir, {ATHENA_UI_E2E_PATH_PREFIX: '/athena'}));
    assert.equal(result.code, 0, result.stdout + result.stderr);
    assert.match(result.stdout, /2 passed/);
});

for (const scenario of ['bootstrap-error', 'invalid-bootstrap', 'wrong-realm', 'wrong-role', 'resource-404', 'slow-resource-404', 'pageerror']) {
    test(`controlled ${scenario} exits nonzero and retains native failure evidence`, async t => {
        const fixture = await startFixture(scenario);
        const outputDir = await makeOutput(`smoke-${scenario}`);
        t.after(async () => {
            await fixture.close();
            await rm(outputDir, {recursive: true, force: true});
        });
        const result = await runPlaywright(playwrightEnv(fixture.baseURL, outputDir));
        assert.notEqual(result.code, 0, result.stdout + result.stderr);
        const files = await filesBelow(outputDir);
        assert(
            files.some(file => file.endsWith('results.json')),
            files.join('\n')
        );
        assert(
            files.some(file => file.endsWith(path.join('html', 'index.html'))),
            files.join('\n')
        );
        assert(
            files.some(file => file.endsWith('trace.zip')),
            files.join('\n')
        );
        assert(
            files.some(file => file.endsWith('.png')),
            files.join('\n')
        );
    });
}

for (const prefix of ['', '/athena']) {
    for (const [scenario, attempts] of [
        ['recover-once', 2],
        ['recover-many', 4],
        ['recover-json', 2]
    ]) {
        test(`smoke waits for ${scenario} application retry under ${prefix || 'root'}`, async t => {
            const fixture = await startFixture(scenario, prefix);
            const outputDir = await makeOutput('smoke-recovery');
            t.after(async () => {
                await fixture.close();
                await rm(outputDir, {recursive: true, force: true});
            });
            const result = await runPlaywright(playwrightEnv(`${fixture.baseURL}${prefix}`, outputDir, {ATHENA_UI_E2E_PATH_PREFIX: prefix}));
            assert.equal(result.code, 0, result.stdout + result.stderr);
            assert.deepEqual(fixture.bootstrapRequests, {member: attempts, admin: attempts}, 'only application retries should issue bootstrap requests');
            const report = JSON.parse(await readFile(path.join(outputDir, 'results.json'), 'utf8'));
            assert.equal(report.stats.expected, 2);
            assert.equal(report.stats.flaky, 0);
        });
    }
}

test('an authentication popup fails smoke without reaching the auth handler', async t => {
    const fixture = await startFixture('auth-popup');
    const outputDir = await makeOutput('smoke-auth-popup');
    t.after(async () => {
        await fixture.close();
        await rm(outputDir, {recursive: true, force: true});
    });
    const result = await runPlaywright(playwrightEnv(fixture.baseURL, outputDir));
    assert.notEqual(result.code, 0, result.stdout + result.stderr);
    assert.equal(fixture.authHandlerHits(), 0, result.stdout + result.stderr);
});

test('an unreachable service exits nonzero and retains a trace', async t => {
    const fixture = await startFixture('anonymous');
    const unreachable = fixture.baseURL;
    await fixture.close();
    const outputDir = await makeOutput('smoke-unreachable');
    t.after(() => rm(outputDir, {recursive: true, force: true}));
    const result = await runPlaywright(playwrightEnv(unreachable, outputDir));
    assert.notEqual(result.code, 0, result.stdout + result.stderr);
    const files = await filesBelow(outputDir);
    assert(
        files.some(file => file.endsWith('results.json')),
        files.join('\n')
    );
    assert(
        files.some(file => file.endsWith('trace.zip')),
        files.join('\n')
    );
});

for (const [scenario, expectedCode] of [
    ['near-deadline', 0],
    ['pending-bootstrap-error', 1]
]) {
    test(`final bootstrap verification handles ${scenario}`, async t => {
        const fixture = await startFixture(scenario);
        const outputDir = await makeOutput('smoke-final-bootstrap');
        t.after(async () => {
            await fixture.close();
            await rm(outputDir, {recursive: true, force: true});
        });
        const result = await runPlaywright(playwrightEnv(fixture.baseURL, outputDir), ['--grep=member application entry']);
        assert.equal(result.code, expectedCode, result.stdout + result.stderr);
        if (scenario === 'pending-bootstrap-error') {
            assert.equal(fixture.bootstrapRequests.member, 2);
            assert.match(result.stdout + result.stderr, /HTTP 503/);
        }
    });
}
