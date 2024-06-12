#!/usr/bin/env node

import fs from 'fs';

const p = 'node_modules/js-fatfs/package.json';
const pkg = JSON.parse(fs.readFileSync(p, 'utf-8'));

if (!Object.prototype.hasOwnProperty.call(pkg, 'type')) {
  console.log('Patching js-fatfs!')
  pkg.type = 'module';
  fs.writeFileSync(p, JSON.stringify(pkg, null, 2));
}
