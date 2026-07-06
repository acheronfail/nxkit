import { describe, expect, test } from 'vitest';
import fsp from 'node:fs/promises';
import path from 'node:path';
import { temporaryDirectoryTask } from 'tempy';
import { getUniquePath } from './paths';

describe(getUniquePath.name, () => {
  test('files exist', () => {
    return temporaryDirectoryTask<void>(async (dir) => {
      await expect(getUniquePath(path.join(dir, 'file'))).resolves.toBe(path.join(dir, 'file'));
      await fsp.writeFile(path.join(dir, 'file'), '0');
      await expect(getUniquePath(path.join(dir, 'file'))).resolves.toBe(path.join(dir, 'file.01'));
      await fsp.writeFile(path.join(dir, 'file.01'), '1');
      await expect(getUniquePath(path.join(dir, 'file'))).resolves.toBe(path.join(dir, 'file.02'));
    });
  });

  test('dir exists', () => {
    return temporaryDirectoryTask<void>(async (dir) => {
      await expect(getUniquePath(path.join(dir, 'file'))).resolves.toBe(path.join(dir, 'file'));
      await fsp.mkdir(path.join(dir, 'file'));
      await expect(getUniquePath(path.join(dir, 'file'))).resolves.toBe(path.join(dir, 'file.01'));
      await fsp.mkdir(path.join(dir, 'file.01'));
      await expect(getUniquePath(path.join(dir, 'file'))).resolves.toBe(path.join(dir, 'file.02'));
    });
  });
});
