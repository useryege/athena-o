import fs from 'node:fs/promises';
import {constants, openSync, closeSync} from 'node:fs';
import path from 'node:path';
import {fileURLToPath, pathToFileURL} from 'node:url';
import {createRequire} from 'node:module';
import {spawn} from 'node:child_process';
import {createHash, randomBytes, randomUUID} from 'node:crypto';

const defaultRoot = fileURLToPath(new URL('../../', import.meta.url));
const delay = ms => new Promise(resolve => setTimeout(resolve, ms));
const defaults = {startup: 300_000, postgres: 60_000, stop: 30_000, term: 5_000, kill: 5_000, poll: 100, command: 1_800_000};

function failure(message, detail = {}) {
  return Object.assign(new Error(message), detail);
}
function detail(error) {
  return {message: error.message, ...(error.stage && {stage: error.stage}), ...(error.code !== undefined && {code: error.code}), ...(error.signal && {signal: error.signal})};
}
async function executable(file, label) {
  try {
    if (!(await fs.stat(file)).isFile()) throw Error();
    await fs.access(file, constants.X_OK);
  } catch {
    throw failure(`${label} unavailable or not executable: ${file}`);
  }
  return file;
}
async function findExecutable(name, env, explicit, fallback) {
  if (explicit) return executable(path.resolve(explicit), name);
  for (const dir of (env.PATH || '').split(path.delimiter)) {
    if (!dir) continue;
    try {
      return await executable(path.resolve(dir, name), name);
    } catch {
      /* next PATH directory */
    }
  }
  if (fallback) return executable(fallback, name);
  throw failure(`${name} unavailable in PATH`);
}

// Validate the raw authority too: URL normalizes 127.1 and decimal IPv4 to 127.0.0.1.
export function smokeTarget(value) {
  let target;
  try {
    target = new URL(value);
  } catch {
    throw failure('Invalid smoke target URL');
  }
  const authority = /^http:\/\/([^/]+)/.exec(value)?.[1];
  if (target.protocol !== 'http:' || !/^(localhost|127\.0\.0\.1|\[::1\])(?::\d+)?$/.test(authority || '') || target.username || target.password || target.search || target.hash) {
    throw failure('Smoke target requires an HTTP loopback URL (localhost, 127.0.0.1 or [::1]), without user information, query or fragment');
  }
  const prefix = target.pathname.replace(/\/+$/, '');
  return {baseURL: target.origin + prefix, prefix};
}

function launch(command, args, {cwd, env, log} = {}) {
  const fd = log ? openSync(log, 'a', 0o600) : undefined;
  let child;
  try {
    child = spawn(command, args, {cwd, env, detached: true, stdio: ['ignore', fd ?? 'pipe', fd ?? 'pipe']});
  } finally {
    if (fd !== undefined) closeSync(fd);
  }
  const proc = {child, command, args, result: null, stdout: '', stderr: ''};
  child.stdout?.on('data', data => {
    proc.stdout = (proc.stdout + data).slice(-1_000_000);
  });
  child.stderr?.on('data', data => {
    proc.stderr = (proc.stderr + data).slice(-1_000_000);
  });
  proc.done = new Promise(resolve => {
    const finish = value => {
      if (!proc.result) {
        proc.result = value;
        resolve(value);
      }
    };
    child.once('error', error => finish({code: null, signal: null, error: error.message}));
    child.once('exit', (code, signal) => finish({code, signal}));
  });
  return proc;
}
function groupAlive(proc) {
  if (!proc.child.pid) return false;
  try {
    process.kill(-proc.child.pid, 0);
    return true;
  } catch (error) {
    if (error.code === 'ESRCH') return false;
    throw error;
  }
}
function signalGroup(proc, signal) {
  if (!proc.child.pid) return;
  try {
    process.kill(-proc.child.pid, signal);
  } catch (error) {
    if (error.code !== 'ESRCH') throw error;
  }
}
async function until(predicate, timeout, poll) {
  const end = Date.now() + timeout;
  while (!(await predicate())) {
    if (Date.now() >= end) return false;
    await delay(poll);
  }
  return true;
}
async function terminate(proc, timeouts) {
  if (!groupAlive(proc)) return;
  signalGroup(proc, 'SIGTERM');
  if (await until(() => !groupAlive(proc), timeouts.term, timeouts.poll)) return;
  signalGroup(proc, 'SIGKILL');
  if (!(await until(() => !groupAlive(proc), timeouts.kill, timeouts.poll))) throw failure(`Process group ${proc.child.pid} did not exit after SIGKILL`);
}
async function checked(command, args, options, timeouts) {
  options.check?.();
  const proc = launch(command, args, options);
  try {
    const ended = await until(
      () => {
        options.check?.();
        return proc.result;
      },
      Math.min(timeouts.command, 30_000),
      timeouts.poll
    );
    if (!ended) throw failure(`${path.basename(command)} preflight timed out`);
    if (proc.result.code !== 0) throw failure(`${path.basename(command)} ${args.join(' ')} failed: ${proc.stderr || proc.result.error || proc.result.code}`);
    return proc.stdout.trim();
  } finally {
    await terminate(proc, timeouts);
  }
}

async function preflight(root, env, timeouts, check, invocationCwd) {
  const mode = env.UI_ACCEPTANCE_MODE || 'isolated';
  if (!['isolated', 'smoke'].includes(mode)) throw failure(`Invalid UI_ACCEPTANCE_MODE: ${mode}`);
  const suite = env.UI_ACCEPTANCE_SUITE || 'acceptance';
  if (!['acceptance', 'a11y'].includes(suite)) throw failure(`Invalid UI_ACCEPTANCE_SUITE: ${suite}`);
  if (suite === 'a11y' && mode !== 'isolated') throw failure('The a11y suite requires isolated UI_ACCEPTANCE_MODE');
  const grep = env.UI_ACCEPTANCE_GREP || null;
  if (mode === 'smoke' && grep) throw failure('UI_ACCEPTANCE_GREP is not supported in smoke mode; use isolated mode for filtered acceptance');
  const target = mode === 'smoke' ? smokeTarget(env.UI_ACCEPTANCE_BASE_URL || 'http://localhost:4000') : null;
  if (mode === 'isolated' && env.UI_ACCEPTANCE_BASE_URL) throw failure('UI_ACCEPTANCE_BASE_URL is only valid in smoke mode; isolated uses its fresh harness manifest');
  if (process.platform !== 'linux') throw failure('Run ui-acceptance with Linux/WSL Node in the same runtime as the repository');
  const ui = path.join(root, 'ui');
  const require = createRequire(path.join(ui, 'package.json'));
  let pkg, semver, playwright, yarn, playwrightCLI, versions;
  try {
    pkg = JSON.parse(await fs.readFile(path.join(ui, 'package.json'), 'utf8'));
    for (const name of Object.keys({...pkg.dependencies, ...pkg.devDependencies})) {
      // Read from node_modules directly: some packages intentionally hide package.json in exports.
      try {
        await fs.access(path.join(ui, 'node_modules', name, 'package.json'), constants.R_OK);
      } catch {
        throw failure(`Declared UI dependency is missing: ${name}`);
      }
    }
    semver = require('semver');
    playwright = require('@playwright/test');
    yarn = require.resolve('yarn/bin/yarn.js');
    playwrightCLI = require.resolve('@playwright/test/cli');
    versions = {node: process.version, nodeExecutable: process.execPath};
    for (const name of ['yarn', '@playwright/test', 'vite']) {
      const dependency = require(`${name}/package.json`);
      versions[name] = dependency.version;
      if (dependency.engines?.node && !semver.satisfies(process.version, dependency.engines.node))
        throw failure(`Node ${process.version} does not satisfy ${name} engines ${dependency.engines.node}`);
    }
    if (pkg.engines?.node && !semver.satisfies(process.version, pkg.engines.node)) throw failure(`Node ${process.version} does not satisfy project engines ${pkg.engines.node}`);
  } catch (error) {
    throw failure(`Project-local UI dependencies/Node blocker: ${error.message}`);
  }
  const childEnv = {...env, PATH: `${path.dirname(process.execPath)}:${env.PATH || ''}`};
  // Never inherit an unrelated harness, deployment prefix or test output location.
  for (const key of Object.keys(childEnv)) if (key.startsWith('ATHENA_UI_E2E_') || key === 'ATHENA_UI_DIST') delete childEnv[key];
  const browser = await executable(mode === 'isolated' ? playwright.chromium.executablePath() : path.resolve(invocationCwd, env.ATHENA_CHROME_PATH || '/usr/bin/google-chrome'), 'browser');
  if (mode === 'smoke') childEnv.ATHENA_CHROME_PATH = browser;
  else delete childEnv.ATHENA_CHROME_PATH;
  versions.browser = await checked(browser, ['--version'], {cwd: ui, env: childEnv, check}, timeouts);
  versions.yarnCLI = await checked(process.execPath, [yarn, '--version'], {cwd: ui, env: childEnv, check}, timeouts);
  versions.playwrightCLI = await checked(process.execPath, [playwrightCLI, '--version'], {cwd: ui, env: childEnv, check}, timeouts);
  let go, docker, image;
  if (mode === 'isolated') {
    go = await findExecutable('go', childEnv, env.ATHENA_UI_ACCEPTANCE_GO, '/usr/local/go/bin/go');
    childEnv.PATH = `${path.dirname(go)}:${childEnv.PATH}`;
    childEnv.GOTOOLCHAIN = 'local';
    childEnv.GOPROXY = 'off';
    // GOPRIVATE/GONOPROXY can bypass GOPROXY=off with direct VCS downloads.
    childEnv.GONOPROXY = 'none';
    docker = await findExecutable('docker', childEnv);
    versions.go = await checked(go, ['version'], {cwd: root, env: childEnv, check}, timeouts);
    versions.docker = await checked(docker, ['version', '--format', '{{.Server.Version}}'], {cwd: root, env: childEnv, check}, timeouts);
    const tag = env.ATHENA_POSTGRES_IMAGE_TAG || '16';
    if (!/^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$/.test(tag)) throw failure('Invalid PostgreSQL image tag');
    image = `docker.io/library/postgres:${tag}`;
    versions.postgresImage = {name: image, id: await checked(docker, ['image', 'inspect', '--format', '{{.Id}}', image], {cwd: root, env: childEnv, check}, timeouts)};
  }
  return {mode, suite, grep, target, ui, env: childEnv, yarn, playwrightCLI, browser, go, docker, image, versions};
}

async function buildSummary(dir) {
  const files = [];
  async function walk(current) {
    for (const entry of (await fs.readdir(current, {withFileTypes: true})).sort((a, b) => a.name.localeCompare(b.name))) {
      const full = path.join(current, entry.name);
      if (entry.isDirectory()) await walk(full);
      else if (entry.isFile()) {
        const content = await fs.readFile(full);
        files.push({path: path.relative(dir, full), bytes: content.length, sha256: createHash('sha256').update(content).digest('hex')});
      } else throw failure(`Unexpected non-regular build artifact: ${full}`);
    }
  }
  await walk(dir);
  if (!files.some(file => file.path === 'index.html')) throw failure('UI build snapshot is missing index.html');
  return {directory: dir, sha256: createHash('sha256').update(JSON.stringify(files)).digest('hex'), files};
}

async function verifyAxeViolationEvidence(output) {
  let results;
  try {
    results = JSON.parse(await fs.readFile(path.join(output, 'results.json'), 'utf8'));
  } catch {
    throw failure(`a11y Playwright exited without readable results.json: ${output}`);
  }
  if (!Array.isArray(results.errors) || results.errors.length) throw failure(`a11y Playwright reported global errors: ${output}`);
  const specs = [];
  const collect = suite => {
    specs.push(...(suite.specs || []));
    for (const child of suite.suites || []) collect(child);
  };
  for (const suite of results.suites || []) collect(suite);
  const tests = specs.flatMap(spec => spec.tests || []);
  const unexpected = tests.filter(test => test.status === 'unexpected');
  if (
    !tests.length ||
    !unexpected.length ||
    tests.some(test => !['expected', 'unexpected'].includes(test.status) || test.results?.length !== 1) ||
    results.stats?.unexpected !== unexpected.length ||
    results.stats?.expected !== tests.length - unexpected.length ||
    results.stats?.skipped !== 0 ||
    results.stats?.flaky !== 0
  ) throw failure(`a11y Playwright did not complete all discovered tests: ${output}`);
  for (const test of unexpected) {
    const result = test.results[0];
    if (result.status !== 'failed' || !result.errors?.length || result.errors.some(error => !error.message?.includes('[ATHENA_A11Y_VIOLATION]')))
      throw failure(`a11y Playwright failed outside the axe violation assertion: ${output}`);
    const attachment = result.attachments?.find(item => item.name === 'axe-results.json' && item.contentType === 'application/json');
    const relative = attachment?.path && path.relative(output, attachment.path);
    if (!relative || relative.startsWith('..') || path.isAbsolute(relative)) throw failure(`a11y violation is missing its local axe attachment: ${output}`);
    let axe;
    try {
      axe = JSON.parse(await fs.readFile(attachment.path, 'utf8'));
    } catch {
      throw failure(`a11y violation has an unreadable axe attachment: ${output}`);
    }
    if (!Array.isArray(axe.violations) || !axe.violations.some(violation => violation.nodes?.length))
      throw failure(`a11y violation assertion has no axe violation evidence: ${output}`);
  }
}

async function acquireLock(root, env, timeouts, runId, check) {
  let parent;
  try {
    parent = path.resolve(root, await checked('git', ['rev-parse', '--git-common-dir'], {cwd: root, env, check}, timeouts));
  } catch (error) {
    check();
    parent = path.join(root, '.tmp/athena-ui-acceptance');
    await fs.mkdir(parent, {recursive: true});
  }
  const dir = path.join(parent, '.athena-ui-acceptance.lock');
  check();
  try {
    await fs.mkdir(dir);
  } catch (error) {
    if (error.code === 'EEXIST') throw failure(`Another isolated acceptance run holds the repository lock: ${dir}`);
    throw error;
  }
  try {
    check();
    await fs.writeFile(path.join(dir, 'owner.json'), JSON.stringify({runId, pid: process.pid}), {mode: 0o600});
    check();
  } catch (error) {
    await fs.rm(path.join(dir, 'owner.json'), {force: true});
    await fs.rmdir(dir);
    throw error;
  }
  return async () => {
    const owner = JSON.parse(await fs.readFile(path.join(dir, 'owner.json'), 'utf8'));
    if (owner.runId !== runId) throw failure('Repository lock ownership changed; refusing to remove it');
    await fs.unlink(path.join(dir, 'owner.json'));
    await fs.rmdir(dir);
  };
}

export async function runAcceptance({root = defaultRoot, env = process.env, timeouts: overrides = {}} = {}) {
  const timeouts = {...defaults, ...overrides};
  const invocationCwd = process.cwd();
  let interrupted;
  const signalHandlers = new Map(
    ['SIGHUP', 'SIGINT', 'SIGTERM'].map(signal => [
      signal,
      () => {
        interrupted ||= failure(`Interrupted by ${signal}`, {signal, code: {SIGHUP: 129, SIGINT: 130, SIGTERM: 143}[signal]});
      }
    ])
  );
  for (const [signal, handler] of signalHandlers) process.on(signal, handler);
  const removeSignalHandlers = () => {
    for (const [signal, handler] of signalHandlers) process.off(signal, handler);
  };
  const checkInterrupted = () => {
    if (interrupted) throw interrupted;
  };
  let config;
  try {
    config = await preflight(root, env, timeouts, checkInterrupted, invocationCwd);
    checkInterrupted();
  } catch (error) {
    removeSignalHandlers();
    console.error(error.message);
    if (env.UI_ACCEPTANCE_CHECK_ONLY === '1') console.log(JSON.stringify({mode: env.UI_ACCEPTANCE_MODE || 'isolated', status: 'blocked', blocker: error.message}));
    return interrupted?.code || 1;
  }
  if (env.UI_ACCEPTANCE_CHECK_ONLY === '1') {
    removeSignalHandlers();
    console.log(JSON.stringify({mode: config.mode, suite: config.suite, filtered: Boolean(config.grep), grep: config.grep, status: 'ready', versions: config.versions, browser: config.browser, target: config.target}));
    return 0;
  }
  const runId = `${new Date().toISOString().replace(/[:.]/g, '-')}-${randomUUID().slice(0, 8)}`;
  const runDir = path.join(root, '.tmp/athena-ui-acceptance', runId);
  const report = {
    runId,
    runDir,
    mode: config.mode,
    suite: config.suite,
    filtered: Boolean(config.grep),
    grep: config.grep,
    startedAt: new Date().toISOString(),
    status: 'running',
    versions: config.versions,
    browser: config.browser,
    stages: [],
    failure: null,
    testFailures: [],
    cleanup: {status: 'pending', errors: []},
    evidence: [
      ...(config.grep ? [`filtered：局部运行 Playwright grep ${JSON.stringify(config.grep)}；不包含 live 项目，不能作为完整验收。`] : []),
      ...(
      config.mode === 'smoke'
        ? ['smoke：仅验证既有开发环境的会员/管理员应用壳，不证明业务回归；不启停该环境。']
        : config.suite === 'a11y'
          ? ['a11y：在两个隔离部署前缀中，以受控 API fixture 扫描指定页面与状态；每个扫描保留原始 axe 结果。', '扫描违规继续收集剩余状态；基础设施故障中止并清理。']
        : ['ui-fixtures：模拟 API 响应下的真实页面。', 'live：真实产品组件与临时 PostgreSQL；链、资料、Telegram 为本地替身。', '仅已完成的阶段构成证据；未执行的检查不能报告通过。'])
    ]
  };
  let releaseLock, container, harness;
  const check = () => {
    if (interrupted) throw interrupted;
    if (harness?.proc.result && !harness.stopping)
      throw failure(`Harness exited unexpectedly (${harness.proc.result.code ?? harness.proc.result.signal})`, {stage: harness.stage.name, ...harness.proc.result});
  };
  async function phase(name, action) {
    const stage = {name, startedAt: new Date().toISOString(), status: 'running'};
    report.stages.push(stage);
    try {
      const value = await action();
      stage.status = 'passed';
      return value;
    } catch (error) {
      error.stage ||= name;
      stage.status = 'failed';
      stage.error = detail(error);
      throw error;
    } finally {
      stage.completedAt = new Date().toISOString();
    }
  }
  async function command(name, bin, args, {cwd = root, extraEnv = {}, cleanup = false, collectTestFailure = false, log} = {}) {
    if (!cleanup) check();
    const stage = {name, startedAt: new Date().toISOString(), status: 'running', log: log || path.join(runDir, `${name.replaceAll('/', '-')}.log`)};
    report.stages.push(stage);
    const proc = launch(bin, args, {cwd, env: {...config.env, ...extraEnv}, log: stage.log});
    let collectedFailure;
    try {
      const end = Date.now() + (cleanup ? 30_000 : timeouts.command);
      while (!proc.result) {
        if (!cleanup) check();
        if (Date.now() > end) throw failure(`${name} timed out`);
        await delay(timeouts.poll);
      }
      if (!cleanup) check();
      stage.exit = proc.result;
      if (proc.result.code !== 0) {
        const error = failure(`${name} failed (${proc.result.code ?? proc.result.signal ?? proc.result.error}); see ${stage.log}`, proc.result);
        if (collectTestFailure && proc.result.code === 1) {
          error.stage = name;
          stage.error = detail(error);
          collectedFailure = error;
        } else throw error;
      }
      stage.status = collectedFailure ? 'failed' : 'passed';
    } catch (error) {
      stage.status = 'failed';
      stage.error = detail(error);
      error.stage ||= name;
      throw error;
    } finally {
      try {
        await terminate(proc, timeouts);
      } catch (error) {
        report.cleanup.errors.push(detail(error));
      }
      stage.exit ||= proc.result;
      stage.completedAt = new Date().toISOString();
    }
    return collectedFailure ? {failure: collectedFailure} : fs.readFile(stage.log, 'utf8');
  }
  async function stopHarness() {
    if (!harness) return;
    const current = harness;
    let stopError;
    try {
      if (current.proc.result) {
        throw failure(`Harness exited before stop (${current.proc.result.code ?? current.proc.result.signal})`);
      } else {
        current.stopping = true;
        await fs.writeFile(path.join(current.dir, 'stop'), 'stop\n', {mode: 0o600});
        if (!(await until(() => current.proc.result, timeouts.stop, timeouts.poll))) throw failure('Harness did not stop within its graceful stop deadline');
        if (current.proc.result.code !== 0) throw failure(`Harness stop failed (${current.proc.result.code ?? current.proc.result.signal})`);
      }
    } catch (error) {
      stopError = error;
    }
    try {
      await terminate(current.proc, timeouts);
    } catch (error) {
      report.cleanup.errors.push(detail(error));
      stopError ||= error;
    }
    current.stage.exit = current.proc.result;
    current.stage.status = stopError ? 'failed' : 'passed';
    current.stage.completedAt = new Date().toISOString();
    if (stopError) current.stage.error = detail(stopError);
    harness = null;
    if (stopError) throw stopError;
  }
  try {
    if (config.mode === 'isolated') releaseLock = await acquireLock(root, config.env, timeouts, runId, check);
    await fs.mkdir(runDir, {recursive: true, mode: 0o700});
    console.error(`ATHENA acceptance evidence: ${runDir}`);
    if (config.mode === 'isolated') {
      await command('build', process.execPath, [config.yarn, 'build'], {cwd: config.ui});
      const snapshot = path.join(runDir, 'build/app');
      report.build = await phase('build-snapshot', async () => {
        await fs.cp(path.join(config.ui, 'dist/app'), snapshot, {recursive: true});
        return buildSummary(snapshot);
      });
      check();
      container = `athena-ui-acceptance-${runId.toLowerCase()}`;
      const password = randomBytes(24).toString('hex');
      report.postgres = {name: container, image: config.image, label: `io.athena.ui-acceptance.run=${runId}`};
      await command('postgres-start', config.docker, [
        'run',
        '--detach',
        '--pull=never',
        '--name',
        container,
        '--label',
        report.postgres.label,
        '--publish',
        '127.0.0.1::5432',
        '--env',
        'POSTGRES_USER=postgres',
        '--env',
        `POSTGRES_PASSWORD=${password}`,
        '--env',
        'POSTGRES_DB=postgres',
        config.image
      ]);
      const binding = (await command('postgres-port', config.docker, ['port', container, '5432/tcp'])).trim();
      const port = /^127\.0\.0\.1:(\d+)$/.exec(binding)?.[1];
      if (!port || Number(port) > 65535 || Number(port) < 1) throw failure(`Unexpected PostgreSQL port binding: ${binding}`);
      await phase('postgres-ready', async () => {
        const end = Date.now() + timeouts.postgres;
        while (true) {
          check();
          // pg_isready can fail during startup; only the overall deadline is a failed stage.
          const ready = launch(config.docker, ['exec', container, 'pg_isready', '-U', 'postgres'], {cwd: root, env: config.env, log: path.join(runDir, 'postgres-ready.log')});
          try {
            while (!ready.result) {
              check();
              if (Date.now() > end) throw failure('PostgreSQL readiness timed out');
              await delay(timeouts.poll);
            }
            if (ready.result.code === 0) break;
          } finally {
            await terminate(ready, timeouts);
          }
          if (Date.now() > end) throw failure('PostgreSQL readiness timed out');
          await delay(timeouts.poll);
        }
      });
      for (const prefix of ['', '/athena']) {
        check();
        const name = prefix ? 'athena' : 'root';
        const dir = path.join(runDir, name, 'harness');
        await fs.mkdir(dir, {recursive: true, mode: 0o700});
        const manifestFile = path.join(dir, 'harness.json');
        const stage = {name: `${name}/harness`, status: 'running', startedAt: new Date().toISOString(), log: path.join(dir, 'harness.log')};
        report.stages.push(stage);
        const proc = launch(config.go, ['test', '-v', '-tags=integration,uiharness', './internal/tradersync/acceptance', '-run', '^TestUIHarness$', '-count=1', '-timeout=30m'], {
          cwd: root,
          env: {
            ...config.env,
            ATHENA_UI_E2E_DIR: dir,
            ATHENA_UI_DIST: snapshot,
            ATHENA_UI_E2E_PATH_PREFIX: prefix,
            ATHENA_UI_E2E_FIXTURE_ONLY: config.grep || config.suite === 'a11y' ? '1' : undefined,
            ATHENA_TEST_PG_ADMIN_DSN: `postgres://postgres:${password}@127.0.0.1:${port}/postgres?sslmode=disable`
          },
          log: stage.log
        });
        harness = {proc, dir, stage, stopping: false};
        let manifest;
        await phase(`${name}/startup`, async () => {
          const ready = await until(
            async () => {
              check();
              try {
                manifest = JSON.parse(await fs.readFile(manifestFile, 'utf8'));
              } catch (error) {
                if (error.code === 'ENOENT' || error instanceof SyntaxError) return false;
                throw error;
              }
              const target = smokeTarget(manifest.BaseURL);
              if (target.prefix || manifest.PathPrefix !== prefix) throw failure('Harness manifest has an unexpected deployment target');
              return true;
            },
            timeouts.startup,
            timeouts.poll
          );
          if (!ready) throw failure('Harness startup timed out');
        });
        check();
        report.stages.push({name: `${name}/ready`, status: 'passed', baseURL: manifest.BaseURL, prefix, manifest: manifestFile, database: manifest.Database});
        const projects = config.suite === 'a11y' ? ['a11y'] : config.grep ? ['ui-fixtures'] : ['ui-fixtures', 'live'];
        for (const project of projects) {
          const output = path.join(runDir, name, project);
          await fs.mkdir(output, {recursive: true});
          const playwrightArgs = [config.playwrightCLI, 'test', `--project=${project}`, ...(config.grep ? ['--grep', config.grep] : [])];
          const result = await command(`${name}/${project}`, process.execPath, playwrightArgs, {
            cwd: config.ui,
            collectTestFailure: config.suite === 'a11y',
            extraEnv: {
              ATHENA_UI_E2E_MODE: 'isolated',
              ATHENA_UI_E2E_BASE_URL: manifest.BaseURL,
              ATHENA_UI_E2E_PATH_PREFIX: prefix,
              ATHENA_UI_E2E_MANIFEST: manifestFile,
              ATHENA_UI_E2E_OUTPUT_DIR: output
            },
            log: path.join(output, 'runner.log')
          });
          if (result?.failure) {
            await phase(`${name}/scan-evidence`, () => verifyAxeViolationEvidence(output));
            report.testFailures.push(detail(result.failure));
          }
        }
        await stopHarness();
      }
    } else {
      const output = path.join(runDir, 'smoke');
      await fs.mkdir(output);
      await command('smoke', process.execPath, [config.playwrightCLI, 'test', '--project=smoke'], {
        cwd: config.ui,
        extraEnv: {ATHENA_UI_E2E_MODE: 'smoke', ATHENA_UI_E2E_BASE_URL: config.target.baseURL, ATHENA_UI_E2E_PATH_PREFIX: config.target.prefix, ATHENA_UI_E2E_OUTPUT_DIR: output},
        log: path.join(output, 'runner.log')
      });
    }
  } catch (error) {
    report.failure = detail(error);
    console.error(error.message);
  } finally {
    try {
      await stopHarness();
    } catch (error) {
      report.cleanup.errors.push(detail(error));
    }
    if (container) {
      try {
        await command('cleanup-postgres', config.docker, ['rm', '--force', '-v', container], {cleanup: true});
      } catch (error) {
        report.cleanup.errors.push(detail(error));
      }
    }
    if (releaseLock) {
      try {
        await releaseLock();
      } catch (error) {
        report.cleanup.errors.push(detail(error));
      }
    }
    report.failure ||= interrupted ? detail(interrupted) : null;
    report.cleanup.status = report.cleanup.errors.length ? 'failed' : 'passed';
    report.status = report.failure || report.testFailures.length || report.cleanup.errors.length ? 'failed' : 'passed';
    report.completedAt = new Date().toISOString();
    // Lock contention must leave the other run untouched, including its evidence directory.
    if (await fs.stat(runDir).catch(() => null)) {
      await fs.writeFile(path.join(runDir, 'run.json'), JSON.stringify(report, null, 2) + '\n');
      const lines = [
        '# ATHENA 浏览器验收',
        '',
        `模式：${report.mode}；范围：${report.filtered ? 'filtered（局部）' : 'complete'}；结果：${report.status}；清理：${report.cleanup.status}`,
        '',
        `开始：${report.startedAt}`,
        `结束：${report.completedAt}`,
        '',
        '## 证据边界',
        '',
        ...report.evidence.map(text => `- ${text}`),
        '',
        '## 阶段',
        '',
        ...report.stages.map(
          stage =>
            `- ${stage.name}：${stage.status}${stage.exit ? `（退出：${stage.exit.code ?? stage.exit.signal ?? stage.exit.error}）` : ''}${stage.log ? `；日志：${path.relative(runDir, stage.log)}` : ''}`
        ),
        '',
        '## 失败与清理',
        '',
        `基础设施失败：${report.failure?.message || '无'}`,
        ...report.testFailures.map(error => `- 扫描失败：${error.message}`),
        ...report.cleanup.errors.map(error => `- ${error.message}`),
        '',
        '## 工具与构建',
        '',
        '```json',
        JSON.stringify({browser: report.browser, versions: report.versions, build: report.build}, null, 2),
        '```',
        ''
      ];
      await fs.writeFile(path.join(runDir, 'report.md'), lines.join('\n'));
    }
    removeSignalHandlers();
  }
  return interrupted?.code || (report.status === 'passed' ? 0 : 1);
}

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) {
  if (process.argv.length > 2) {
    console.error('ui-acceptance does not accept additional Playwright arguments; select mode with UI_ACCEPTANCE_MODE.');
    process.exitCode = 1;
  } else {
    try {
      process.exitCode = await runAcceptance();
    } catch (error) {
      console.error(error.message);
      process.exitCode = 1;
    }
  }
}
