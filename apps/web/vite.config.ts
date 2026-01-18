import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api/hello': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/api/todo': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  // @ts-expect-error - Vitest config is valid but not in Vite types
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
  },
});

