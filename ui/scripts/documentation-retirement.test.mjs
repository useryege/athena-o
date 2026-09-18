import assert from 'node:assert/strict';
import {spawn} from 'node:child_process';
import {once} from 'node:events';
import {fileURLToPath} from 'node:url';
import test from 'node:test';

const scriptPath = fileURLToPath(import.meta.url);
const retiredPaths = [
    '/swagger-ui',
    '/swagger-ui/index.html',
    '/swagger.json',
    '/llms.txt',
    '/docs/ai',
    '/docs/ai/overview.md',
    '/assets/scripts/redoc.standalone.js',
    '/assets/scripts/redoc-LICENSE.txt',
    '/assets/scripts/README.md'
];

if (process.argv[2] === '--serve') {
    const {createServer, preview} = await import('vite');
    const mode = process.argv[3];
    const port = Number(process.argv[4]);
    const options = {
        configFile: new URL('../vite.config.ts', import.meta.url).pathname,
        root: new URL('../src/app', import.meta.url).pathname,
        logLevel: 'silent'
    };
    const server = mode === 'dev'
        ? await createServer({...options, server: {host: '127.0.0.1', port, strictPort: true}})
        : await preview({...options, preview: {host: '127.0.0.1', port, strictPort: true}});
    if (mode === 'dev') {
        await server.listen();
    }
    const shutdown = async () => {
        await server.close();
        process.exit(0);
    };
    process.once('SIGTERM', shutdown);
    process.once('SIGINT', shutdown);
    process.stdout.write('ready\n');
    await new Promise(() => {});
}

const reservePort = async () => {
    const {createServer} = await import('node:net');
    const server = createServer();
    server.listen(0, '127.0.0.1');
    await once(server, 'listening');
    const {port} = server.address();
    await new Promise(resolve => server.close(resolve));
    return port;
};

const startVite = async (mode, prefix) => {
    const port = await reservePort();
    const child = spawn(process.execPath, [scriptPath, '--serve', mode, String(port)], {
        cwd: new URL('..', import.meta.url).pathname,
        env: {...process.env, ATHENA_SERVER_BASEHREF: prefix || '/', ATHENA_API_URL: 'http://127.0.0.1:1'},
        stdio: ['ignore', 'pipe', 'pipe']
    });
    let stderr = '';
    child.stderr.on('data', chunk => {
        stderr += chunk;
    });
    await Promise.race([
        once(child.stdout, 'data'),
        once(child, 'exit').then(([code]) => {
            throw new Error(`Vite ${mode} exited with ${code}: ${stderr}`);
        })
    ]);
    return {
        url: `http://127.0.0.1:${port}${prefix}`,
        close: async () => {
            child.kill('SIGTERM');
            await once(child, 'exit');
        }
    };
};

if (process.argv[2] !== '--serve') {
    for (const mode of ['dev', 'preview']) {
        for (const prefix of ['', '/athena']) {
            test(`${mode} retires documentation under ${prefix || 'root'}`, async t => {
                const server = await startVite(mode, prefix);
                t.after(server.close);
                for (const path of retiredPaths) {
                    for (const method of ['GET', 'HEAD']) {
                        for (const accept of ['application/json', 'text/html']) {
                            const response = await fetch(`${server.url}${path}`, {method, headers: {accept}, redirect: 'manual'});
                            assert.equal(response.status, 404, `${method} ${prefix}${path} (${accept})`);
                            assert.equal(response.headers.get('location'), null);
                            if (!prefix) {
                                const aliased = await fetch(`${server.url}/athena${path}`, {method, headers: {accept}, redirect: 'manual'});
                                assert.equal(aliased.status, 404, `${method} /athena${path} (${accept})`);
                                assert.equal(aliased.headers.get('location'), null);
                            }
                        }
                    }
                }
                const page = await fetch(`${server.url}/swagger-ui-guide`, {headers: {accept: 'text/html'}});
                assert.equal(page.status, 200);
                const asset = await fetch(`${server.url}/fonts.css`);
                assert.equal(asset.status, 200);
            });
        }
    }
}
