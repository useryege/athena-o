import {defineConfig} from 'vite';
import react from '@vitejs/plugin-react';
import path from 'node:path';
export default defineConfig({root: path.resolve(__dirname), plugins: [react()], server: {host: '127.0.0.1', port: 34191, strictPort: true}});
