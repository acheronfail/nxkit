#!/usr/bin/env node

import cp from 'node:child_process';

for (const script of process.argv.slice(2)) {
  cp.execSync(`npm run ${script}`, { stdio: 'inherit' });
}
