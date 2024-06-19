import type { UserConfig } from 'vite';
import { defineConfig } from 'vite';
import wasm from 'vite-plugin-wasm';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { svelteTesting } from '@testing-library/svelte/vite';
import { assetsInclude } from './vite.renderer.config';

// https://vitejs.dev/config
export default defineConfig(() => {
  return {
    assetsInclude,
    plugins: [wasm(), svelte(), svelteTesting()],
    test: {
      environment: 'jsdom',
      setupFiles: ['vitest.setup.ts'],
      alias: {
        '@testing-library/svelte': '@testing-library/svelte/svelte5',
      },
    },
    resolve: {
      preserveSymlinks: true,
    },
    clearScreen: false,
  } as UserConfig;
});
