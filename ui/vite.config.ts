import react from '@vitejs/plugin-react';
import {defineConfig} from 'vite';

const apiTarget = process.env.ATHENA_API_URL || `http://localhost:${process.env.ATHENA_SERVER_PORT || '8080'}`;

const proxyConf = {
    target: apiTarget,
    secure: false,
    changeOrigin: true
};

export default defineConfig({
    root: 'src/app',
    publicDir: '../assets',
    build: {
        outDir: '../../dist/app',
        emptyOutDir: false,
        sourcemap: true
    },
    plugins: [react()],
    server: {
        port: 4000,
        host: process.env.ATHENA_YARN_HOST || 'localhost',
        proxy: {
            '/api': proxyConf,
            '/auth': proxyConf,
            '/swagger-ui': proxyConf,
            '/swagger.json': proxyConf
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
});
