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
  const config: UserConfig = {
    build: {
      lib: {
        entry,
        fileName: `${(entry as string).replace(/\//g, '_').replace(/\.ts$/, '')}`,
        formats: ['es'],
      },
      rollupOptions: {
        external,
      },
    },
    plugins: [copyNativeNodesModules(buildConfig.build?.outDir), pluginHotRestart('restart')],
    define,
    resolve: {
      // Load the Node.js entry.
      mainFields: ['module', 'jsnext:main', 'jsnext'],
    },
  };

  return mergeConfig(buildConfig, config);
});

function copyNativeNodesModules(outDir: string): PluginOption {
  let unique = 0;
  const nodeDir = join(outDir, '.node');

  return {
    enforce: 'pre',
    name: 'copy-native-nodes-modules',
    buildStart: (_options) => {
      fs.mkdirSync(nodeDir, { recursive: true });
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

        const dotNodePath = join(dirname(id), requirePath);
        const bundledPath = join(nodeDir, `${basename(dotNodePath, '.node')}.${unique++}.node`);
        const newRequirePath = relative(outDir, bundledPath);
        fs.copyFileSync(dotNodePath, bundledPath);

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
