#!/usr/bin/env node

import cp from 'node:child_process';
import chalk from 'chalk';

for (const script of process.argv.slice(2)) {
  try {
    cp.execSync(`npm run ${script}`, { stdio: 'inherit' });
  } catch (err) {
    console.error(chalk.red.bold(err));
    process.exit(1);
  }
}
