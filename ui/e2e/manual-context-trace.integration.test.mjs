import assert from 'node:assert/strict';
import {mkdtemp, readFile, rm, writeFile} from 'node:fs/promises';
import http from 'node:http';
import os from 'node:os';
import path from 'node:path';
import {spawn} from 'node:child_process';
import test from 'node:test';

const ui = path.resolve(import.meta.dirname, '..');

for (const fails of [false, true]) {
    test(`native tracing handles early and final context closure on ${fails ? 'failure' : 'success'}`, async t => {
        let bootstraps = 0;
        const server = http.createServer((request, response) => {
            if (request.url === '/api/v1/app/bootstrap') {
                const status = fails && ++bootstraps === 2 ? 'ANONYMOUS' : 'AUTHENTICATED';
                response
                    .writeHead(200, {'content-type': 'application/json'})
                    .end(JSON.stringify({session: {status: `APP_BOOTSTRAP_SESSION_STATUS_${status}`, user_info: {administrator: request.headers['x-test-realm'] === 'admin'}}}));
            } else {
                response
                    .writeHead(200, {'content-type': 'text/html'})
                    .end(
                        '<!doctype html><h1>Trader Sync</h1><script>fetch("/api/v1/app/bootstrap", {headers: {"X-Test-Realm": location.pathname.startsWith("/admin/") ? "admin" : "member"}})</script>'
                    );
            }
        });
        await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
        const output = await mkdtemp(path.join(os.tmpdir(), 'athena-native-trace-'));
        t.after(async () => {
            await new Promise(resolve => server.close(resolve));
            await rm(output, {recursive: true, force: true});
        });
        const baseURL = `http://127.0.0.1:${server.address().port}`;
        const state = path.join(output, 'state.json');
        const manifest = path.join(output, 'manifest.json');
        await writeFile(state, JSON.stringify({cookies: [], origins: []}));
        await writeFile(manifest, JSON.stringify({BaseURL: baseURL, PathPrefix: '', MemberAState: state, MemberBState: state, AdminState: state}));
        const result = await new Promise((resolve, reject) => {
            const child = spawn(
                process.execPath,
                [path.join(ui, 'node_modules/@playwright/test/cli.js'), 'test', '--project=live', '--grep=three isolated real cookie sessions bootstrap'],
                {
                    cwd: ui,
                    env: {
                        ...process.env,
                        ATHENA_UI_E2E_MODE: 'isolated',
                        ATHENA_UI_E2E_BASE_URL: baseURL,
                        ATHENA_UI_E2E_PATH_PREFIX: '',
                        ATHENA_UI_E2E_MANIFEST: manifest,
                        ATHENA_UI_E2E_OUTPUT_DIR: output,
                        ATHENA_CHROME_PATH: '/not-used-by-isolated'
                    },
                    stdio: ['ignore', 'pipe', 'pipe']
                }
            );
            let log = '';
            child.stdout.on('data', chunk => (log += chunk));
            child.stderr.on('data', chunk => (log += chunk));
            child.on('error', reject);
            child.on('close', code => resolve({code, log}));
        });
        const report = JSON.parse(await readFile(path.join(output, 'results.json'), 'utf8'));
        const results = [];
        const visit = suite => {
            for (const spec of suite.specs || []) for (const entry of spec.tests) results.push(...entry.results);
            for (const nested of suite.suites || []) visit(nested);
        };
        for (const suite of report.suites) visit(suite);
        assert.equal(results.length, 1, result.log);
        assert.doesNotMatch(result.log, /Must start tracing|close failed|trace stop failed/);
        assert.equal(result.code, fails ? 1 : 0, result.log);
        assert.equal(results[0].errors.length, fails ? 1 : 0, result.log);
        if (fails) {
            assert.match(results[0].errors[0].message, /APP_BOOTSTRAP_SESSION_STATUS_AUTHENTICATED/);
            const trace = results[0].attachments.find(attachment => attachment.name === 'trace');
            assert(trace?.path, result.log);
            assert((await readFile(trace.path)).length > 100, 'native trace is retained');
        } else {
            assert.equal(results[0].status, 'passed');
            assert.equal(
                results[0].attachments.some(attachment => attachment.name === 'trace'),
                false,
                'successful test does not retain a trace'
            );
        }
    });
}
