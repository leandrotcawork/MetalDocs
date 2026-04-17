import { defineConfig } from 'vitest/config';
import { resolve } from 'node:path';

export default defineConfig({
  resolve: {
    alias: {
      '@metaldocs/mddm-layout-tokens': resolve(__dirname, '../../../shared/mddm-layout-tokens/index.ts'),
      '@metaldocs/mddm-pagination-types': resolve(__dirname, '../../../shared/mddm-pagination-types/index.ts'),
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./vitest.setup.ts'],
    testTimeout: 15000,
    globals: true,
  },
});
