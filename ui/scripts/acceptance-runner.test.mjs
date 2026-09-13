import assert from 'node:assert/strict';
import {spawn} from 'node:child_process';
import fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import {fileURLToPath} from 'node:url';
import {test} from 'node:test';

const runner = fileURLToPath(new URL('./acceptance-runner.mjs', import.meta.url));
const launcher = fileURLToPath(new URL('../../hack/ui-acceptance.sh', import.meta.url));
const fakeTool = String.raw`
const fs = require('node:fs');
const path = require('node:path');
const tool = path.basename(process.argv[1]);
const args = process.argv.slice(2);
const env = process.env;
const record = (event) => fs.appendFileSync(env.EVENTS, JSON.stringify({tool, args, pid:process.pid, executable:process.argv[1], cwd:process.cwd(), chromePath:env.ATHENA_CHROME_PATH, ...event})+'\n');
record({});
if (env.HOLD_PREFLIGHT && tool === 'chrome') { fs.writeFileSync(env.CONTAINER,String(process.pid)); setInterval(()=>{},1000); return; }
if (env.HOLD_LOCK && tool === 'git') { setInterval(()=>{},1000); return; }
if (tool === 'git') process.exit(1);
if (args.includes('--version') || args[0] === 'version') { console.log(tool+' 1.0'); process.exit(0); }
if (tool === 'docker') {
  if (args[0] === 'image') { if (env.FAIL === 'image') process.exit(2); console.log('sha256:fixture-image'); }
  if (args[0] === 'run') { fs.writeFileSync(env.CONTAINER, args[args.indexOf('--name')+1]); if(env.FAIL === 'pg') process.exit(3); console.log('fixture-container'); }
  if (args[0] === 'port') console.log('127.0.0.1:54329');
  if (args[0] === 'exec' && env.FAIL === 'pg-ready') process.exit(1);
  if (args[0] === 'rm') { if(env.FAIL_CLEANUP) process.exit(7); fs.rmSync(env.CONTAINER, {force:true}); }
  process.exit(0);
}
if (tool === 'go') {
  if(env.FAIL === 'harness') process.exit(4);
  const dir=env.ATHENA_UI_E2E_DIR;
  record({dir, prefix: env.ATHENA_UI_E2E_PATH_PREFIX, dist: env.ATHENA_UI_DIST, dsn: env.ATHENA_TEST_PG_ADMIN_DSN, goProxy:env.GOPROXY, goNoProxy:env.GONOPROXY, goToolchain:env.GOTOOLCHAIN});
  if(env.FAIL !== 'startup-timeout') fs.writeFileSync(path.join(dir,'harness.json'), JSON.stringify({BaseURL:'http://127.0.0.1:40123', PathPrefix:env.ATHENA_UI_E2E_PATH_PREFIX, Database:dir, MemberAState:'a', MemberBState:'b', AdminState:'admin', ControlURL:'http://127.0.0.1:40124', Owners:[], HTMLHashes:{}}));
  if(env.FAIL === 'harness-during-test') setTimeout(()=>process.exit(8), 200);
  if(env.FAIL === 'stop-timeout') process.on('SIGTERM',()=>{});
  setInterval(()=> { if(env.FAIL !== 'stop-timeout' && fs.existsSync(path.join(dir,'stop'))) process.exit(0); }, 10);
} else if(tool === 'yarn.js') {
  if(env.HOLD_BUILD && args.includes('build')) { setInterval(()=>{},1000); return; }
  if(env.FAIL === 'build') process.exit(5);
  fs.mkdirSync('dist/app', {recursive:true}); fs.writeFileSync('dist/app/index.html','<html>built</html>');
} else if(tool === 'cli.js') {
  const project=args.find(a=>a.startsWith('--project='))?.split('=')[1];
  record({project, mode:env.ATHENA_UI_E2E_MODE, prefix:env.ATHENA_UI_E2E_PATH_PREFIX, manifest:env.ATHENA_UI_E2E_MANIFEST, output:env.ATHENA_UI_E2E_OUTPUT_DIR});
  if(env.FAIL === 'playwright') process.exit(6);
  if(project === 'a11y' && env.FAIL === 'a11y-playwright') {
    const attachment=path.join(env.ATHENA_UI_E2E_OUTPUT_DIR,'axe-results.json');
    fs.writeFileSync(attachment,JSON.stringify({violations:[{id:'color-contrast',nodes:[{target:['#example']}]}]}));
    fs.writeFileSync(path.join(env.ATHENA_UI_E2E_OUTPUT_DIR,'results.json'),JSON.stringify({
      errors:[],stats:{expected:0,unexpected:1,skipped:0,flaky:0},
      suites:[{specs:[{tests:[{status:'unexpected',results:[{
        status:'failed',errors:[{message:'Error: [ATHENA_A11Y_VIOLATION] controlled finding'}],
        attachments:[{name:'axe-results.json',path:attachment,contentType:'application/json'}]
      }]}]}]}]
    }));
    process.exit(1);
  }
  if(project === 'a11y' && env.FAIL === 'a11y-config') process.exit(1);
  if(project === 'a11y' && env.FAIL === 'a11y-browser') {
    fs.writeFileSync(path.join(env.ATHENA_UI_E2E_OUTPUT_DIR,'results.json'),JSON.stringify({
      errors:[],stats:{expected:0,unexpected:1,skipped:0,flaky:0},
      suites:[{specs:[{tests:[{status:'unexpected',results:[{
        status:'failed',errors:[{message:'browserType.launch: Executable does not exist'}],attachments:[]
      }]}]}]}]
    }));
    process.exit(1);
  }
  if(env.HOLD || env.FAIL === 'harness-during-test') setInterval(()=>{},1000);
}
`;

async function fixture(t, extra = {}) {
  const root = await fs.mkdtemp(path.join(os.tmpdir(), 'athena-runner-test-'));
  const children = [];
  t.after(async () => {
    for (const child of children) if (child.exitCode === null && child.signalCode === null) child.kill('SIGKILL');
    const lines = (await fs.readFile(path.join(root, 'events'), 'utf8').catch(() => '')).trim().split('\n').filter(Boolean);
    for (const pid of new Set(lines.map(line => JSON.parse(line).pid))) {
      try { process.kill(-pid, 'SIGKILL'); } catch (error) { if (error.code !== 'ESRCH') throw error; }
    }
    await Promise.all(children.map(child => child.done));
    await fs.rm(root, {recursive: true, force: true});
  });
  const put = async (name, body, mode = 0o644) => {
    await fs.mkdir(path.dirname(path.join(root, name)), {recursive: true});
    await fs.writeFile(path.join(root, name), body, {mode});
  };
  await put('ui/package.json', JSON.stringify({engines: {node: '>=20'}}));
  for (const name of ['yarn', '@playwright/test', 'vite', 'semver']) {
    await put(`ui/node_modules/${name}/package.json`, JSON.stringify({name, version: '1.0.0', main: 'index.cjs', engines: {node: '>=20'}}));
  }
  await put('ui/node_modules/semver/index.cjs', `exports.satisfies = (version) => Number(version.replace(/^v/,'').split('.')[0]) >= 20;`);
  await put('ui/node_modules/@playwright/test/index.cjs', `exports.chromium = {executablePath:()=>${JSON.stringify(path.join(root, 'bin/chromium'))}};`);
  for (const name of ['go', 'docker', 'chromium', 'chrome', 'git']) await put(`bin/${name}`, `#!${process.execPath}\n${fakeTool}`, 0o755);
  for (const name of ['yarn/bin/yarn.js', '@playwright/test/cli.js']) await put(`ui/node_modules/${name}`, fakeTool);
  const env = {
    ...process.env,
    PATH: `${path.join(root, 'bin')}:${path.dirname(process.execPath)}:/usr/bin:/bin`,
    ATHENA_CHROME_PATH: path.join(root, 'bin/chrome'),
    EVENTS: path.join(root, 'events'),
    CONTAINER: path.join(root, 'container'),
    ...extra
  };
  const start = (override = {}, {cwd = process.cwd()} = {}) => {
    const child = spawn(
      process.execPath,
      [
        '--input-type=module',
        '-e',
        `import {runAcceptance} from ${JSON.stringify('file://' + runner)}; process.exitCode=await runAcceptance({root:${JSON.stringify(root)}, timeouts:{startup:450,postgres:180,stop:120,term:80,kill:80,poll:10,command:3000}});`
      ],
      {cwd, env: {...env, ...override}, stdio: ['ignore', 'pipe', 'pipe']}
    );
    let stdout = '',
      stderr = '';
    child.stdout.on('data', d => (stdout += d));
    child.stderr.on('data', d => (stderr += d));
    child.done = new Promise(resolve => child.on('close', (code, signal) => resolve({code, signal, stdout, stderr})));
    children.push(child);
    return child;
  };
  const events = async () => (await fs.readFile(env.EVENTS, 'utf8').catch(() => '')).trim().split('\n').filter(Boolean).map(JSON.parse);
  const reports = async () => {
    const dirs = await fs.readdir(path.join(root, '.tmp/athena-ui-acceptance'));
    return Promise.all(dirs.filter(d => !d.startsWith('.')).map(d => fs.readFile(path.join(root, '.tmp/athena-ui-acceptance', d, 'run.json'), 'utf8').then(JSON.parse)));
  };
  const until = async predicate => {
    const end = Date.now() + 5000;
    while (Date.now() < end) {
      if (await predicate()) return;
      await new Promise(r => setTimeout(r, 10));
    }
    throw Error('fixture condition timeout');
  };
  return {root, env, start, events, reports, until};
}

test('check-only is read-only and smoke does not require Go, Docker or manifests', async t => {
  const f = await fixture(t, {UI_ACCEPTANCE_MODE: 'smoke', UI_ACCEPTANCE_CHECK_ONLY: '1'});
  await fs.rm(path.join(f.root, 'bin/go'));
  await fs.rm(path.join(f.root, 'bin/docker'));
  const result = await f.start().done;
  assert.equal(result.code, 0, result.stderr);
  assert.equal(JSON.parse(result.stdout).mode, 'smoke');
  assert.equal(await fs.stat(path.join(f.root, '.tmp')).catch(() => null), null);
  assert.ok((await f.events()).every(e => e.args.includes('--version')));
});

test('invalid modes and non-loopback or credential-bearing smoke targets fail before side effects', async t => {
  const f = await fixture(t, {UI_ACCEPTANCE_MODE: 'smoke'});
  for (const url of ['https://localhost:4000', 'http://example.com', 'http://user@localhost', 'http://localhost?q=x', 'http://localhost#x', 'http://127.1']) {
    const result = await f.start({UI_ACCEPTANCE_BASE_URL: url}).done;
    assert.notEqual(result.code, 0, url);
    assert.match(result.stderr, /loopback|target|URL/i);
  }
  const result = await f.start({UI_ACCEPTANCE_MODE: 'other'}).done;
  assert.match(result.stderr, /mode/i);
  assert.deepEqual(await f.events(), []);
});

test('missing local browser and missing isolated image are blockers without run directories', async t => {
  const f = await fixture(t, {UI_ACCEPTANCE_CHECK_ONLY: '1'});
  let result = await f.start({FAIL: 'image'}).done;
  assert.notEqual(result.code, 0);
  assert.match(result.stderr, /image/i);
  await fs.rm(path.join(f.root, 'bin/chromium'));
  result = await f.start().done;
  assert.notEqual(result.code, 0);
  assert.match(result.stderr, /browser|chromium/i);
  assert.equal(await fs.stat(path.join(f.root, '.tmp')).catch(() => null), null);
});

test('isolated builds once, snapshots the build and runs fresh root/prefix harnesses sequentially', async t => {
  const f = await fixture(t, {GOPRIVATE: '*', GONOPROXY: '*'});
  const result = await f.start().done;
  assert.equal(result.code, 0, result.stderr);
  const events = await f.events();
  assert.equal(events.filter(e => e.tool === 'yarn.js' && e.args.includes('build')).length, 1);
  assert.deepEqual(
    events.filter(e => e.project).map(e => [e.prefix, e.project]),
    [
      ['', 'ui-fixtures'],
      ['', 'live'],
      ['/athena', 'ui-fixtures'],
      ['/athena', 'live']
    ]
  );
  const harnesses = events.filter(e => e.dir);
  assert.equal(harnesses.length, 2);
  assert.notEqual(harnesses[0].dir, harnesses[1].dir);
  assert.ok(
    harnesses.every(h => h.goProxy === 'off' && h.goNoProxy === 'none' && h.goToolchain === 'local'),
    'Go must use cached modules and the selected local toolchain, including for inherited private module patterns'
  );
  assert.equal(await fs.readFile(path.join(harnesses[0].dist, 'index.html'), 'utf8'), '<html>built</html>');
  const run = events.find(e => e.tool === 'docker' && e.args[0] === 'run');
  assert.ok(run.args.includes('--pull=never'));
  assert.ok(run.args.includes('127.0.0.1::5432'));
  assert.ok(!run.args.includes('-v'));
  const removal = events.find(e => e.tool === 'docker' && e.args[0] === 'rm');
  assert.ok(removal.args.includes('-v'));
  assert.ok(removal.args.includes(run.args[run.args.indexOf('--name') + 1]));
  const [report] = await f.reports();
  assert.equal(report.status, 'passed');
  assert.equal(report.cleanup.status, 'passed');
  assert.ok(report.build.files.length);
  assert.equal(await fs.stat(f.env.CONTAINER).catch(() => null), null);
});

test('a11y suite runs only the accessibility project for both isolated deployment prefixes', async t => {
  const f = await fixture(t, {UI_ACCEPTANCE_SUITE: 'a11y'});
  const result = await f.start().done;
  assert.equal(result.code, 0, result.stderr);
  assert.deepEqual(
    (await f.events()).filter(e => e.project).map(e => [e.prefix, e.project]),
    [['', 'a11y'], ['/athena', 'a11y']]
  );
  const [report] = await f.reports();
  assert.equal(report.suite, 'a11y');
  assert.equal(report.cleanup.status, 'passed');
  assert.equal(await fs.stat(f.env.CONTAINER).catch(() => null), null);
});

test('a11y Playwright failures collect the second prefix and clean all owned resources', async t => {
  const f = await fixture(t, {UI_ACCEPTANCE_SUITE: 'a11y', FAIL: 'a11y-playwright'});
  const result = await f.start().done;
  assert.notEqual(result.code, 0);
  assert.deepEqual(
    (await f.events()).filter(e => e.project).map(e => [e.prefix, e.project]),
    [['', 'a11y'], ['/athena', 'a11y']]
  );
  const [report] = await f.reports();
  assert.equal(report.status, 'failed');
  assert.equal(report.failure, null, 'test findings must remain distinct from infrastructure failures');
  assert.equal(report.testFailures.length, 2);
  assert.equal(report.cleanup.status, 'passed');
  assert.equal(report.stages.filter(stage => stage.name.endsWith('/a11y') && stage.status === 'failed').length, 2);
  assert.equal(await fs.stat(f.env.CONTAINER).catch(() => null), null);
});

test('smoke mode rejects a11y suite before browser or service side effects', async t => {
  const f = await fixture(t, {UI_ACCEPTANCE_MODE: 'smoke', UI_ACCEPTANCE_SUITE: 'a11y'});
  const result = await f.start().done;
  assert.notEqual(result.code, 0);
  assert.match(result.stderr, /a11y.*isolated|isolated.*a11y/i);
  assert.deepEqual(await f.events(), []);
});

test('a11y infrastructure failure stops before a second harness and cleans resources', async t => {
  const f = await fixture(t, {UI_ACCEPTANCE_SUITE: 'a11y', FAIL: 'startup-timeout'});
  const result = await f.start().done;
  assert.notEqual(result.code, 0);
  assert.equal((await f.events()).filter(e => e.dir).length, 1);
  assert.deepEqual((await f.events()).filter(e => e.project), []);
  const [report] = await f.reports();
  assert.equal(report.status, 'failed');
  assert.match(report.failure.message, /startup/i);
  assert.deepEqual(report.testFailures, []);
  assert.equal(report.cleanup.status, 'passed');
  assert.equal(await fs.stat(f.env.CONTAINER).catch(() => null), null);
});

for (const failure of ['a11y-config', 'a11y-browser'])
  test(`${failure} exit one is infrastructure, stops the second prefix, and cleans resources`, async t => {
    const f = await fixture(t, {UI_ACCEPTANCE_SUITE: 'a11y', FAIL: failure});
    const result = await f.start().done;
    assert.notEqual(result.code, 0);
    assert.deepEqual((await f.events()).filter(e => e.project).map(e => [e.prefix, e.project]), [['', 'a11y']]);
    const [report] = await f.reports();
    assert.equal(report.status, 'failed');
    assert.ok(report.failure, 'infrastructure error must be reported as the failure');
    assert.deepEqual(report.testFailures, []);
    assert.equal(report.cleanup.status, 'passed');
    assert.equal(await fs.stat(f.env.CONTAINER).catch(() => null), null);
  });

for (const failure of ['build', 'pg', 'pg-ready', 'harness', 'startup-timeout', 'playwright', 'harness-during-test', 'stop-timeout'])
  test(`${failure} failure stops later stages and cleans owned resources`, async t => {
    const f = await fixture(t, {FAIL: failure});
    const result = await f.start().done;
    assert.notEqual(result.code, 0, result.stdout);
    const [report] = await f.reports();
    assert.equal(report.status, 'failed');
    assert.ok(report.failure);
    assert.ok(
      report.stages.some(stage => stage.status === 'failed'),
      'failed lifecycle phase must appear in stage evidence'
    );
    assert.equal(await fs.stat(f.env.CONTAINER).catch(() => null), null);
    assert.ok((await f.events()).filter(e => e.project).length < 4);
  });

test('cleanup failure is retained alongside first test failure', async t => {
  const f = await fixture(t, {FAIL: 'playwright', FAIL_CLEANUP: '1'});
  const result = await f.start().done;
  assert.notEqual(result.code, 0);
  const [report] = await f.reports();
  assert.match(report.failure.message, /ui-fixtures|playwright/i);
  assert.equal(report.cleanup.status, 'failed');
  assert.ok(report.cleanup.errors.length);
});

for (const signal of ['SIGINT', 'SIGTERM'])
  test(`${signal} cleans owned resources and leaves unrelated processes alone`, async t => {
    const f = await fixture(t, {HOLD: '1'});
    const other = spawn(process.execPath, ['-e', 'setInterval(()=>{},1000)']);
    t.after(() => other.kill());
    const child = f.start();
    await f.until(async () => (await f.events()).some(e => e.project));
    child.kill(signal);
    const result = await child.done;
    assert.equal(result.code, signal === 'SIGINT' ? 130 : 143, result.stderr);
    assert.equal(other.exitCode, null);
    assert.equal(await fs.stat(f.env.CONTAINER).catch(() => null), null);
    const [report] = await f.reports();
    assert.equal(report.failure.signal, signal);
  });

test('concurrent isolated run cannot build or clean the first run resources', async t => {
  const f = await fixture(t, {HOLD: '1'});
  const first = f.start();
  await f.until(async () => (await f.events()).some(e => e.project));
  const second = await f.start().done;
  assert.notEqual(second.code, 0);
  assert.match(second.stderr, /lock|running/i);
  assert.equal((await f.events()).filter(e => e.tool === 'yarn.js' && e.args.includes('build')).length, 1);
  assert.ok(await fs.stat(f.env.CONTAINER));
  first.kill('SIGINT');
  await first.done;
});

test('smoke selects only smoke and never controls the existing environment', async t => {
  const f = await fixture(t, {UI_ACCEPTANCE_MODE: 'smoke', UI_ACCEPTANCE_BASE_URL: 'http://[::1]:4000/athena/'});
  const result = await f.start().done;
  assert.equal(result.code, 0, result.stderr);
  const events = await f.events();
  assert.deepEqual(
    events.filter(e => e.project).map(e => [e.project, e.prefix, e.manifest]),
    [['smoke', '/athena', undefined]]
  );
  assert.ok(!events.some(e => ['docker', 'go'].includes(e.tool)));
  const [report] = await f.reports();
  assert.equal(report.status, 'passed');
});

test('shell rejects explicit missing Node without falling back', async () => {
  const child = spawn('/bin/bash', [launcher], {env: {...process.env, ATHENA_UI_ACCEPTANCE_NODE: '/missing/node'}, stdio: ['ignore', 'pipe', 'pipe']});
  let stderr = '';
  child.stderr.on('data', d => (stderr += d));
  const code = await new Promise(resolve => child.on('close', resolve));
  assert.notEqual(code, 0);
  assert.match(stderr, /ATHENA_UI_ACCEPTANCE_NODE|Node/);
});

test('shell chooses explicit Node, PATH Node, then resolved NVM default without changing caller PATH', async t => {
  const f = await fixture(t);
  const bin = path.join(f.root, 'selection-bin');
  await fs.mkdir(bin);
  await fs.symlink('/usr/bin/dirname', path.join(bin, 'dirname'));
  const nodeScript = label => `#!/bin/bash\nprintf '%s\\n' '${label}'\n`;
  const explicit = path.join(bin, 'explicit');
  await fs.writeFile(explicit, nodeScript('explicit'), {mode: 0o755});
  const nvm = path.join(f.root, 'nvm');
  await fs.mkdir(nvm);
  await fs.writeFile(path.join(nvm, 'nvm.sh'), `nvm() { printf '%s\\n' '${explicit}'; }\n`);
  const invoke = async extra => {
    const child = spawn('/bin/bash', [launcher], {env: {...process.env, PATH: bin, NVM_DIR: nvm, ATHENA_UI_ACCEPTANCE_NODE: '', ...extra}, stdio: ['ignore', 'pipe', 'pipe']});
    let out = '';
    child.stdout.on('data', d => (out += d));
    const code = await new Promise(r => child.on('close', r));
    return {code, out: out.trim()};
  };
  await fs.writeFile(path.join(bin, 'node'), nodeScript('path'), {mode: 0o755});
  assert.deepEqual(await invoke({ATHENA_UI_ACCEPTANCE_NODE: explicit}), {code: 0, out: 'explicit'});
  assert.deepEqual(await invoke({}), {code: 0, out: 'path'});
  await fs.rm(path.join(bin, 'node'));
  assert.deepEqual(await invoke({}), {code: 0, out: 'explicit'});
  await fs.rm(path.join(nvm, 'nvm.sh'));
  assert.notEqual((await invoke({})).code, 0);
});

test('check-only blocks a missing declared UI dependency before creating evidence', async t => {
  const f = await fixture(t, {UI_ACCEPTANCE_MODE: 'smoke', UI_ACCEPTANCE_CHECK_ONLY: '1'});
  await fs.writeFile(path.join(f.root, 'ui/package.json'), JSON.stringify({dependencies: {react: '^18.3.1'}}));
  const result = await f.start().done;
  assert.notEqual(result.code, 0);
  assert.match(result.stderr, /react/);
  assert.equal(await fs.stat(path.join(f.root, '.tmp')).catch(() => null), null);
});

test('a timed out smoke command is terminated and reported as failure', async t => {
  const f = await fixture(t, {UI_ACCEPTANCE_MODE: 'smoke', HOLD: '1'});
  const result = await f.start().done;
  assert.notEqual(result.code, 0);
  const [report] = await f.reports();
  assert.match(report.failure.message, /timed out/);
  assert.equal(report.cleanup.status, 'passed');
});

test('cleanup failure alone makes an otherwise successful run fail', async t => {
  const f = await fixture(t, {FAIL_CLEANUP: '1'});
  const result = await f.start().done;
  assert.notEqual(result.code, 0);
  const [report] = await f.reports();
  assert.equal(report.failure, null);
  assert.equal(report.status, 'failed');
  assert.equal(report.cleanup.status, 'failed');
});

test('interrupting preflight also reaps the selected version probe process', async t => {
  const f = await fixture(t, {UI_ACCEPTANCE_MODE: 'smoke', HOLD_PREFLIGHT: '1'});
  const child = f.start();
  await f.until(async () => (await fs.readFile(f.env.CONTAINER, 'utf8').catch(() => '')) !== '');
  const pid = Number(await fs.readFile(f.env.CONTAINER, 'utf8'));
  t.after(() => {
    try {
      process.kill(pid, 'SIGKILL');
    } catch {}
  });
  child.kill('SIGTERM');
  const result = await child.done;
  assert.equal(result.code, 143);
  assert.throws(() => process.kill(pid, 0), {code: 'ESRCH'});
});

test('SIGHUP during browser preflight exits 129 and reaps the version probe', async t => {
  const f = await fixture(t, {UI_ACCEPTANCE_MODE: 'smoke', HOLD_PREFLIGHT: '1'});
  const child = f.start();
  await f.until(async () => (await f.events()).some(e => e.tool === 'chrome'));
  const probe = (await f.events()).find(e => e.tool === 'chrome');
  child.kill('SIGHUP');
  const result = await child.done;
  assert.equal(result.code, 129, result.stderr);
  assert.throws(() => process.kill(probe.pid, 0), {code: 'ESRCH'});
  assert.equal(await fs.stat(path.join(f.root, '.tmp')).catch(() => null), null);
});

test('SIGHUP while acquiring the repository lock exits promptly without an orphaned probe', async t => {
  const f = await fixture(t, {HOLD_LOCK: '1'});
  const child = f.start();
  await f.until(async () => (await f.events()).some(e => e.tool === 'git'));
  const probe = (await f.events()).find(e => e.tool === 'git');
  const started = Date.now();
  child.kill('SIGHUP');
  const result = await child.done;
  assert.equal(result.code, 129, result.stderr);
  assert.ok(Date.now() - started < 1500, 'lock acquisition must observe the signal before its command deadline');
  assert.throws(() => process.kill(probe.pid, 0), {code: 'ESRCH'});
});

test('SIGHUP during build releases the lock and allows a subsequent isolated run', async t => {
  const f = await fixture(t, {HOLD_BUILD: '1'});
  const child = f.start();
  await f.until(async () => (await f.events()).some(e => e.tool === 'yarn.js' && e.args.includes('build')));
  const build = (await f.events()).find(e => e.tool === 'yarn.js' && e.args.includes('build'));
  child.kill('SIGHUP');
  const result = await child.done;
  assert.equal(result.code, 129, result.stderr);
  assert.throws(() => process.kill(build.pid, 0), {code: 'ESRCH'});
  const [report] = await f.reports();
  assert.equal(report.failure.signal, 'SIGHUP');
  assert.equal(report.cleanup.status, 'passed');
  assert.equal(await fs.stat(f.env.CONTAINER).catch(() => null), null);
  const again = await f.start({HOLD_BUILD: ''}).done;
  assert.equal(again.code, 0, again.stderr);
});

test('repeated SIGHUP during test cleans owned resources once and leaves unrelated processes running', async t => {
  const f = await fixture(t, {HOLD: '1'});
  const unrelated = spawn(process.execPath, ['-e', 'setInterval(()=>{},1000)']);
  t.after(() => unrelated.kill('SIGKILL'));
  const child = f.start();
  await f.until(async () => (await f.events()).some(e => e.project));
  child.kill('SIGHUP');
  child.kill('SIGHUP');
  const result = await child.done;
  assert.equal(result.code, 129, result.stderr);
  assert.equal(unrelated.exitCode, null);
  assert.equal(await fs.stat(f.env.CONTAINER).catch(() => null), null);
  const [report] = await f.reports();
  assert.equal(report.failure.signal, 'SIGHUP');
  assert.equal(report.cleanup.status, 'passed');
  assert.equal((await f.events()).filter(e => e.tool === 'docker' && e.args[0] === 'rm').length, 1);
});

test('SIGHUP retains its first failure and cleanup errors when PostgreSQL removal fails', async t => {
  const f = await fixture(t, {HOLD: '1', FAIL_CLEANUP: '1'});
  const child = f.start();
  await f.until(async () => (await f.events()).some(e => e.project));
  child.kill('SIGHUP');
  const result = await child.done;
  assert.equal(result.code, 129, result.stderr);
  const [report] = await f.reports();
  assert.equal(report.failure.signal, 'SIGHUP');
  assert.equal(report.cleanup.status, 'failed');
  assert.ok(report.cleanup.errors.some(error => /cleanup-postgres/.test(error.message)));
});

for (const location of ['absolute', 'relative', 'spaces'])
  test(`smoke uses one ${location} Chrome path for validation, probe, child environment and report`, async t => {
    const f = await fixture(t, {UI_ACCEPTANCE_MODE: 'smoke'});
    let absolute = f.env.ATHENA_CHROME_PATH;
    if (location === 'spaces') {
      absolute = path.join(f.root, 'browser files', 'chrome');
      await fs.mkdir(path.dirname(absolute));
      await fs.copyFile(f.env.ATHENA_CHROME_PATH, absolute);
      await fs.chmod(absolute, 0o755);
    }
    const selected = location === 'absolute' ? absolute : path.relative(f.root, absolute);
    const result = await f.start({ATHENA_CHROME_PATH: selected}, {cwd: f.root}).done;
    assert.equal(result.code, 0, result.stderr);
    const events = await f.events();
    const probe = events.find(e => e.executable === absolute && e.args.includes('--version'));
    assert.ok(probe, `browser probe did not execute ${absolute}`);
    assert.equal(probe.chromePath, absolute);
    const playwright = events.find(e => e.tool === 'cli.js' && e.project === 'smoke');
    assert.equal(playwright.chromePath, absolute);
    const [report] = await f.reports();
    assert.equal(report.browser, absolute);
    assert.equal(report.versions.browser, 'chrome 1.0');
  });

test('an explicitly missing relative Chrome path blocks smoke before evidence creation', async t => {
  const f = await fixture(t, {UI_ACCEPTANCE_MODE: 'smoke'});
  const missing = 'missing chrome';
  const result = await f.start({ATHENA_CHROME_PATH: missing}, {cwd: f.root}).done;
  assert.notEqual(result.code, 0);
  assert.match(result.stderr, /browser unavailable or not executable/);
  assert.equal(await fs.stat(path.join(f.root, '.tmp')).catch(() => null), null);
  assert.ok(!(await f.events()).some(e => e.project));
});

test('isolated mode probes bundled Chromium even when an explicit Chrome path is supplied', async t => {
  const f = await fixture(t, {UI_ACCEPTANCE_CHECK_ONLY: '1', ATHENA_CHROME_PATH: '/missing/chrome'});
  const result = await f.start().done;
  assert.equal(result.code, 0, result.stderr);
  const ready = JSON.parse(result.stdout);
  assert.equal(ready.browser, path.join(f.root, 'bin/chromium'));
  assert.ok((await f.events()).some(e => e.tool === 'chromium' && e.args.includes('--version')));
});
