import type { UserConfig } from 'vite';
import { defineConfig } from 'vite';

// https://vitejs.dev/config
export default defineConfig(() => {
  return {
    test: {
      poolOptions: {
        forks: {
          execArgv: ['--expose-gc'],
        },
      },
    },
  } as UserConfig;
});
