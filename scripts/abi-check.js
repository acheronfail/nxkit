// Used for checking ABI compatibility post build.
import { createRequire } from 'node:module';
import glob from 'glob';
const require = createRequire(import.meta.url);

const die = (msg) => {
  console.error(msg);
  process.exit(1);
};

const input = process.argv[2];
if (!input) {
  die('Usage: electron abi-check.js <node addon path>');
}

const paths = glob.sync(input, { absolute: true }).filter((p) => p.endsWith('.node'));
if (paths.length === 0) {
  die(`No files found matching ${input}`);
}

let code = 0;
const results = paths.map((path) => {
  try {
    require(path);
    return { path, error: null };
  } catch (e) {
    code++;
    return { path, error: String(e) };
  }
});

console.log(JSON.stringify(results, null, 2));
process.exit(code);
