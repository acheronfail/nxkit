import type { ConfigEnv, PluginOption, UserConfig } from 'vite';
import { defineConfig, mergeConfig } from 'vite';
import { join, basename, dirname, relative } from 'node:path';
import fs from 'node:fs';
import { SourceMapGenerator } from 'source-map';
import { getBuildConfig, getBuildDefine, external, pluginHotRestart } from './vite.base.config';

// https://vitejs.dev/config
export default defineConfig((env) => {
  const forgeEnv = env as ConfigEnv<'build'>;
  const { forgeConfigSelf } = forgeEnv;
  const define = getBuildDefine(forgeEnv);

  if (!forgeConfigSelf.entry) {
    throw new Error('Failed to config forge config entrypoint for main!');
  }

  const buildConfig = getBuildConfig(forgeEnv);
  if (!buildConfig.build?.outDir) {
    throw new Error('Failed to find build out directory');
  }

  const { entry } = forgeConfigSelf;
  const fileName = `${(entry as string).replace(/\//g, '_').replace(/\.ts$/, '')}`;
  const config: UserConfig = {
    build: {
      lib: {
        entry,
        fileName,
        formats: ['es'],
      },
      rollupOptions: {
        external,
      },
    },
    plugins: [
      copyNativeNodesModules(buildConfig.build!.outDir),
      copyWasmFiles(buildConfig.build!.outDir),
      pluginHotRestart('restart'),
    ],
    define,
    resolve: {
      // Load the Node.js entry.
      mainFields: ['module', 'jsnext:main', 'jsnext'],
    },
  };

  return mergeConfig(buildConfig, config);
});

interface CopyContext {
  uniqueId: number;
  copyDir: string;
  rootDir: string;
  extension: string;
}

// Needed to properly import `js-fatfs`'s wasm file (which is built by emscripten).
// If we don't do this manual copy then vite inlines the file as a base64 string which breaks things.
// Since this file is dynamically imported I can't find a way to stop vite from doing this.
function copyWasmFiles(rootDir: string): PluginOption {
  const ctx: CopyContext = {
    uniqueId: 0,
    copyDir: join(rootDir, '.wasm'),
    rootDir,
    extension: '.wasm',
  };

  const re = /new\s+URL\(\s*"(?<name>[a-zA-Z0-9_-]+)\.wasm",\s*import\.meta\.url\s*\).href/;
  return {
    enforce: 'pre',
    name: 'copy-wasm-files',
    buildStart: (_options) => {
      fs.mkdirSync(ctx.copyDir, { recursive: true });
    },
    transform: (code, id) => {
      const match = re.exec(code);
      if (match) {
        const newPath = copyFile(ctx, id, `${match[1]}${ctx.extension}`);
        return code.replace(match[0], `import.meta.url.replace(/\\/[^/]*$/, "/${newPath}")`);
      }
    },
  };
}

function copyNativeNodesModules(rootDir: string): PluginOption {
  const ctx: CopyContext = {
    uniqueId: 0,
    copyDir: join(rootDir, '.node'),
    rootDir,
    extension: '.node',
  };

  return {
    enforce: 'pre',
    name: 'copy-native-nodes-modules',
    buildStart: (_options) => {
      fs.mkdirSync(ctx.copyDir, { recursive: true });
    },
    transform: (code, id) => {
      const requireRe = /require\(['"](.*?)['"]\)/g;

      const map = new SourceMapGenerator({ file: id, sourceRoot: '' });
      map.setSourceContent(id, code);

      let match: RegExpExecArray | null;
      while ((match = requireRe.exec(code))) {
        const requirePath = match[1];
        if (!requirePath || !requirePath.endsWith('.node')) continue;

        const line = code.substring(0, match.index).split('\n').length;
        map.addMapping({
          generated: { line, column: 0 },
          original: { line, column: 0 },
          source: id,
        });

        const newRequirePath = copyFile(ctx, id, requirePath);

        code = [
          code.slice(0, match.index),
          `require("./${newRequirePath}")`,
          code.slice(match.index + match[0].length),
        ].join('');
      }

      return {
        code,
        map: map.toString(),
      };
    },
  };
}

function copyFile(ctx: CopyContext, id: string, requirePath: string) {
  const idPath = join(dirname(id), requirePath);
  const bundledPath = join(ctx.copyDir, `${basename(idPath, ctx.extension)}.${ctx.uniqueId++}${ctx.extension}`);
  const newRequirePath = relative(ctx.rootDir, bundledPath);
  fs.copyFileSync(idPath, bundledPath);
  return newRequirePath;
}
