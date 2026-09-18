import react from '@vitejs/plugin-react';
import {defineConfig, type Connect, type Plugin} from 'vite';
import {resolve} from 'node:path';

const apiTarget = process.env.ATHENA_API_URL || `http://localhost:${process.env.ATHENA_SERVER_PORT || '8080'}`;

const normalizeDeploymentBase = (value: string | undefined) => {
    const segments = (value || '/').trim().split('/').filter(Boolean);
    return segments.length === 0 ? '/' : `/${segments.join('/')}/`;
};

const deploymentBase = normalizeDeploymentBase(process.env.ATHENA_SERVER_BASEHREF);
const deploymentPrefix = deploymentBase === '/' ? '' : deploymentBase.slice(0, -1);
const deploymentPath = (logicalPath: string) => `${deploymentPrefix}${logicalPath}` || '/';

const proxyConf = {
    target: apiTarget,
    secure: false,
    changeOrigin: true,
    rewrite: (pathname: string) => (deploymentPrefix && pathname.startsWith(`${deploymentPrefix}/`) ? pathname.slice(deploymentPrefix.length) : pathname)
};

const pathWithinDeployment = (pathname: string) => {
    if (!deploymentPrefix) {
        return pathname;
    }
    if (pathname === deploymentPrefix) {
        return '/';
    }
    return pathname.startsWith(`${deploymentPrefix}/`) ? pathname.slice(deploymentPrefix.length) : undefined;
};

const isHTMLNavigation = (request: {method?: string; headers: {accept?: string}}) =>
    (request.method === 'GET' || request.method === 'HEAD') && (request.headers.accept || '').split(',').some(value => value.trim().split(';', 1)[0] === 'text/html');

const isProxiedPath = (pathname: string) =>
    pathname === '/api' ||
    pathname.startsWith('/api/') ||
    pathname === '/auth' ||
    pathname.startsWith('/auth/');

const isRetiredDocumentationPath = (pathname: string) =>
    pathname === '/swagger-ui' ||
    pathname.startsWith('/swagger-ui/') ||
    pathname === '/swagger.json' ||
    pathname === '/llms.txt' ||
    pathname === '/docs/ai' ||
    pathname.startsWith('/docs/ai/') ||
    pathname === '/assets/scripts/redoc.standalone.js' ||
    pathname === '/assets/scripts/redoc-LICENSE.txt' ||
    pathname === '/assets/scripts/README.md';

const applicationIndexPath = (pathname: string) => (pathname === '/admin' || pathname.startsWith('/admin/') ? '/admin/index.html' : '/index.html');

const isStaticAssetPath = (pathname: string) =>
    pathname === '/fonts.css' ||
    pathname.startsWith('/images/') ||
    pathname.startsWith('/assets/') ||
    pathname.startsWith('/entry/');

const dualApplicationHistoryFallback = (): Plugin => {
    const install = (middlewares: Connect.Server) => {
        middlewares.use((request, _response, next) => {
            if (!request.url) {
                next();
                return;
            }
            const url = new URL(request.url, 'http://athena.local');
            const logicalPath = pathWithinDeployment(url.pathname);
            if (logicalPath && isRetiredDocumentationPath(logicalPath)) {
                _response.statusCode = 404;
                _response.end('Not Found');
                return;
            }
            if (!isHTMLNavigation(request)) {
                next();
                return;
            }
            if (!logicalPath || isProxiedPath(logicalPath)) {
                next();
                return;
            }
            const indexPath = applicationIndexPath(logicalPath);
            if (isStaticAssetPath(logicalPath)) {
                next();
                return;
            }
            const deployedIndexPath = deploymentPath(indexPath);
            if (url.pathname !== deployedIndexPath) {
                request.url = `${deployedIndexPath}${url.search}`;
            }
            next();
        });
    };

    return {
        name: 'athena-dual-application-history-fallback',
        configureServer: server => install(server.middlewares),
        configurePreviewServer: server => install(server.middlewares),
        transformIndexHtml: {
            order: 'pre',
            handler(html, context) {
                const admin = context.path.endsWith('/admin/index.html');
                const applicationBase = admin ? `${deploymentBase}admin/` : deploymentBase;
                return html
                    .replace(/<base href="[^"]*">/, `<base href="${applicationBase}">`)
                    .replace(/<meta name="athena-deployment-base-href" content="[^"]*">/, `<meta name="athena-deployment-base-href" content="${deploymentBase}">`);
            }
        }
    };
};

export default defineConfig(({command}) => ({
    root: 'src/app',
    publicDir: '../assets',
    base: command === 'serve' ? deploymentBase : './',
    appType: 'mpa',
    build: {
        outDir: '../../dist/app',
        emptyOutDir: false,
        sourcemap: true,
        manifest: true,
        rollupOptions: {
            input: {
                member: resolve(__dirname, 'src/app/index.html'),
                admin: resolve(__dirname, 'src/app/admin/index.html')
            }
        }
    },
    plugins: [dualApplicationHistoryFallback(), react()],
    server: {
        port: 4000,
        host: process.env.ATHENA_YARN_HOST || 'localhost',
        proxy: {
            [deploymentPath('/api')]: proxyConf,
            [deploymentPath('/auth')]: proxyConf
        }
    },
    define: {
        'SYSTEM_INFO': JSON.stringify({
            version: process.env.ATHENA_VERSION || 'latest'
        }),
        'process.env.NODE_ENV': JSON.stringify(process.env.NODE_ENV || 'development'),
        'process.env.NODE_ONLINE_ENV': JSON.stringify(process.env.NODE_ONLINE_ENV || 'offline'),
        'process.env.HOST_ARCH': JSON.stringify(process.env.HOST_ARCH || 'amd64'),
        'process.platform': JSON.stringify('browser')
    }
}));
