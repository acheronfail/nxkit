/**
 * This file can't be `.mts`, because of a limitation in electron-forge:
 * - https://github.com/electron/forge/blob/620a6ae48a6805846d7125fc28bef874ed7a8b37/packages/api/core/src/util/forge-config.ts#L116
 * - https://github.com/gulpjs/interpret/blob/c09bf70bc73d020b9d387223e7b74708687fdb47/index.js#L70
 */

import { MakerSquirrel } from '@electron-forge/maker-squirrel';
import { MakerZIP } from '@electron-forge/maker-zip';
import { MakerDeb } from '@electron-forge/maker-deb';
import { MakerRpm } from '@electron-forge/maker-rpm';
import { VitePlugin } from '@electron-forge/plugin-vite';
import { FusesPlugin } from '@electron-forge/plugin-fuses';
import { FuseV1Options, FuseVersion } from '@electron/fuses';
import cp from 'node:child_process';
import { createRequire } from 'node:module';
const pkgJson = createRequire(import.meta.url)('./package.json');

// *sigh*, the things we do for cjs<->esm incompatibilities...
const extraResource = JSON.parse(
  cp.execSync(
    `npm exec tsx -- --eval "import('./src/resources').then(m => {
      const extraResource = Object.values(m.getResources(false));
      console.log(JSON.stringify(extraResource));
    })"`,
    { encoding: 'utf-8' },
  ),
);

const asar = true;

/** @type {import('@electron-forge/shared-types').ForgeConfig} */
const config = {
  packagerConfig: {
    name: pkgJson.productName,
    executableName: pkgJson.name,
    asar,
    extraResource,
    appCategoryType: 'public.app-category.utilities',
  },
  rebuildConfig: {},
  makers: [
    new MakerSquirrel({}),
    new MakerZIP({}, ['darwin', 'linux']),
    new MakerRpm({
      options: {
        bin: pkgJson.name,
      },
    }),
    new MakerDeb({
      options: {
        bin: pkgJson.name,
      },
    }),
  ],
  plugins: [
    new VitePlugin({
      // `build` can specify multiple entry builds, which can be Main process, Preload scripts, Worker process, etc.
      // If you are familiar with Vite configuration, it will look really familiar.
      build: [
        {
          // `entry` is just an alias for `build.lib.entry` in the corresponding file of `config`.
          entry: 'src/main/index.ts',
          config: 'vite.main.config.ts',
        },
        {
          entry: 'src/preload.ts',
          config: 'vite.preload.config.ts',
        },
        {
          entry: 'src/node/nand/explorer/worker.ts',
          config: 'vite.main.config.ts',
        },
      ],
      renderer: [
        {
          name: 'renderer',
          config: 'vite.renderer.config.ts',
        },
      ],
    }),
    // Fuses are used to enable/disable various Electron functionality
    // at package time, before code signing the application
    new FusesPlugin({
      version: FuseVersion.V1,
      [FuseV1Options.RunAsNode]: false,
      [FuseV1Options.EnableCookieEncryption]: true,
      [FuseV1Options.EnableNodeOptionsEnvironmentVariable]: false,
      [FuseV1Options.EnableNodeCliInspectArguments]: false,
      // Disabled because it currently breaks windows applications
      [FuseV1Options.EnableEmbeddedAsarIntegrityValidation]: false,
      [FuseV1Options.OnlyLoadAppFromAsar]: asar,
    }),
  ],
};

export default config;
