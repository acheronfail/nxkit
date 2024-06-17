import path from 'node:path';
import type { ConfigEnv, UserConfig } from 'vite';
import { defineConfig } from 'vite';
import wasm from 'vite-plugin-wasm';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { pluginExposeRenderer } from './vite.base.config';

// https://vitejs.dev/config
export default defineConfig((env) => {
  const forgeEnv = env as ConfigEnv<'renderer'>;
  const { root, mode, forgeConfigSelf } = forgeEnv;
  const name = forgeConfigSelf.name ?? '';

  return {
    root,
    mode,
    assetsInclude: ['**/exefs/*'],
    base: './',
    build: {
      outDir: `.vite/${name}`,
      rollupOptions: {
        input: {
          window_main: path.join(root, 'src', 'window_main', 'index.html'),
        },
      },
    },
    plugins: [pluginExposeRenderer(name), wasm(), svelte()],
    resolve: {
      preserveSymlinks: true,
    },
    clearScreen: false,
  } as UserConfig;
});
