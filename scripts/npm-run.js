#!/usr/bin/env node

import cp from 'node:child_process';
import os from 'node:os';
import chalk from 'chalk';

let windowsAllowFail = false;
for (const arg of process.argv.slice(2)) {
  if (arg === '--windows-allow-fail') {
    if (os.platform() === 'win32') {
      windowsAllowFail = true;
    }

    continue;
  }

  try {
    console.log(`${chalk.cyan('running script: ')}${chalk.yellow(arg)}`);
    cp.execSync(`npm run ${arg}`, { stdio: 'inherit' });
  } catch (err) {
    console.error(chalk.red.bold(err));
    if (windowsAllowFail) {
      console.log(chalk.yellow('Ignoring error due to --windows-allow-fail flag.'));
      continue;
    }

    process.exit(1);
  }
}
