import { describe, expect, test } from 'vitest';
import { temporaryDirectoryTask } from 'tempy';
import fsp from 'node:fs/promises';
import path from 'node:path';
import { merge, split } from './split';

function sequentialBuffer(size: number, start = 0): Buffer {
  const buf = Buffer.alloc(size);
  for (let i = 0; i < size; i++) {
    buf[i] = start + i;
  }

  return buf;
}

describe(split.name, () => {
  test('copy', () => {
    return temporaryDirectoryTask<void>(async (dir) => {
      const filePath = path.join(dir, 'file');
      await fsp.writeFile(filePath, sequentialBuffer(100));

      await expect(split(filePath, false, false, 30)).resolves.toEqual(expect.objectContaining({ type: 'success' }));

      expect(await fsp.readdir(dir)).toEqual(['file', 'file.00', 'file.01', 'file.02', 'file.03']);
      expect(await fsp.readFile(path.join(dir, 'file.00'))).toEqual(sequentialBuffer(30, 0));
      expect(await fsp.readFile(path.join(dir, 'file.01'))).toEqual(sequentialBuffer(30, 30));
      expect(await fsp.readFile(path.join(dir, 'file.02'))).toEqual(sequentialBuffer(30, 60));
      expect(await fsp.readFile(path.join(dir, 'file.03'))).toEqual(sequentialBuffer(10, 90));
    });
  });

  test('in place', () => {
    return temporaryDirectoryTask<void>(async (dir) => {
      const filePath = path.join(dir, 'file');
      await fsp.writeFile(filePath, sequentialBuffer(100));

      await expect(split(filePath, false, true, 30)).resolves.toEqual(expect.objectContaining({ type: 'success' }));

      expect(await fsp.readdir(dir)).toEqual(['file.00', 'file.01', 'file.02', 'file.03']);
      expect(await fsp.readFile(path.join(dir, 'file.00'))).toEqual(sequentialBuffer(30, 0));
      expect(await fsp.readFile(path.join(dir, 'file.01'))).toEqual(sequentialBuffer(30, 30));
      expect(await fsp.readFile(path.join(dir, 'file.02'))).toEqual(sequentialBuffer(30, 60));
      expect(await fsp.readFile(path.join(dir, 'file.03'))).toEqual(sequentialBuffer(10, 90));
    });
  });

  test('in place; extension', () => {
    return temporaryDirectoryTask<void>(async (dir) => {
      const filePath = path.join(dir, 'file.bin');
      await fsp.writeFile(filePath, sequentialBuffer(100));

      await expect(split(filePath, false, true, 30)).resolves.toEqual(expect.objectContaining({ type: 'success' }));

      expect(await fsp.readdir(dir)).toEqual(['file.bin.00', 'file.bin.01', 'file.bin.02', 'file.bin.03']);
      expect(await fsp.readFile(path.join(dir, 'file.bin.00'))).toEqual(sequentialBuffer(30, 0));
      expect(await fsp.readFile(path.join(dir, 'file.bin.01'))).toEqual(sequentialBuffer(30, 30));
      expect(await fsp.readFile(path.join(dir, 'file.bin.02'))).toEqual(sequentialBuffer(30, 60));
      expect(await fsp.readFile(path.join(dir, 'file.bin.03'))).toEqual(sequentialBuffer(10, 90));
    });
  });

  test('as archive; copy', () => {
    return temporaryDirectoryTask<void>(async (dir) => {
      const filePath = path.join(dir, 'file');
      await fsp.writeFile(filePath, sequentialBuffer(100));

      await expect(split(filePath, true, false, 30)).resolves.toEqual(expect.objectContaining({ type: 'success' }));

      expect(await fsp.readdir(dir)).toEqual(['file', 'file_split']);
      expect(await fsp.readdir(path.join(dir, 'file_split'))).toEqual(['00', '01', '02', '03']);
      expect(await fsp.readFile(path.join(dir, 'file_split/00'))).toEqual(sequentialBuffer(30, 0));
      expect(await fsp.readFile(path.join(dir, 'file_split/01'))).toEqual(sequentialBuffer(30, 30));
      expect(await fsp.readFile(path.join(dir, 'file_split/02'))).toEqual(sequentialBuffer(30, 60));
      expect(await fsp.readFile(path.join(dir, 'file_split/03'))).toEqual(sequentialBuffer(10, 90));
    });
  });

  test('as archive; in place', () => {
    return temporaryDirectoryTask<void>(async (dir) => {
      const filePath = path.join(dir, 'file');
      await fsp.writeFile(filePath, sequentialBuffer(100));

      await expect(split(filePath, true, true, 30)).resolves.toEqual(expect.objectContaining({ type: 'success' }));

      expect(await fsp.readdir(dir)).toEqual(['file_split']);
      expect(await fsp.readdir(path.join(dir, 'file_split'))).toEqual(['00', '01', '02', '03']);
      expect(await fsp.readFile(path.join(dir, 'file_split/00'))).toEqual(sequentialBuffer(30, 0));
      expect(await fsp.readFile(path.join(dir, 'file_split/01'))).toEqual(sequentialBuffer(30, 30));
      expect(await fsp.readFile(path.join(dir, 'file_split/02'))).toEqual(sequentialBuffer(30, 60));
      expect(await fsp.readFile(path.join(dir, 'file_split/03'))).toEqual(sequentialBuffer(10, 90));
    });
  });

  test('as archive; in place; extension', () => {
    return temporaryDirectoryTask<void>(async (dir) => {
      const filePath = path.join(dir, 'file.nsp');
      await fsp.writeFile(filePath, sequentialBuffer(100));

      await expect(split(filePath, true, true, 30)).resolves.toEqual(expect.objectContaining({ type: 'success' }));

      expect(await fsp.readdir(dir)).toEqual(['file_split.nsp']);
      expect(await fsp.readdir(path.join(dir, 'file_split.nsp'))).toEqual(['00', '01', '02', '03']);
      expect(await fsp.readFile(path.join(dir, 'file_split.nsp/00'))).toEqual(sequentialBuffer(30, 0));
      expect(await fsp.readFile(path.join(dir, 'file_split.nsp/01'))).toEqual(sequentialBuffer(30, 30));
      expect(await fsp.readFile(path.join(dir, 'file_split.nsp/02'))).toEqual(sequentialBuffer(30, 60));
      expect(await fsp.readFile(path.join(dir, 'file_split.nsp/03'))).toEqual(sequentialBuffer(10, 90));
    });
  });
});

describe(merge.name, () => {
  test('copy', () => {
    return temporaryDirectoryTask<void>(async (dir) => {
      const basePath = path.join(dir, 'file');
      await fsp.writeFile(`${basePath}.00`, sequentialBuffer(30, 0));
      await fsp.writeFile(`${basePath}.01`, sequentialBuffer(30, 30));
      await fsp.writeFile(`${basePath}.02`, sequentialBuffer(30, 60));
      await fsp.writeFile(`${basePath}.03`, sequentialBuffer(10, 90));

      await expect(merge(`${basePath}.00`, false)).resolves.toEqual(expect.objectContaining({ type: 'success' }));

      expect(await fsp.readdir(dir)).toEqual(['file', 'file.00', 'file.01', 'file.02', 'file.03']);
      expect(await fsp.readFile(path.join(dir, 'file'))).toEqual(sequentialBuffer(100, 0));
    });
  });

  test('in place', () => {
    return temporaryDirectoryTask<void>(async (dir) => {
      const basePath = path.join(dir, 'file');
      await fsp.writeFile(`${basePath}.00`, sequentialBuffer(30, 0));
      await fsp.writeFile(`${basePath}.01`, sequentialBuffer(30, 30));
      await fsp.writeFile(`${basePath}.02`, sequentialBuffer(30, 60));
      await fsp.writeFile(`${basePath}.03`, sequentialBuffer(10, 90));

      await expect(merge(`${basePath}.00`, true)).resolves.toEqual(expect.objectContaining({ type: 'success' }));

      expect(await fsp.readdir(dir)).toEqual(['file']);
      expect(await fsp.readFile(path.join(dir, 'file'))).toEqual(sequentialBuffer(100, 0));
    });
  });

  test('in place; extension', () => {
    return temporaryDirectoryTask<void>(async (dir) => {
      const basePath = path.join(dir, 'file.bin');
      await fsp.writeFile(`${basePath}.00`, sequentialBuffer(30, 0));
      await fsp.writeFile(`${basePath}.01`, sequentialBuffer(30, 30));
      await fsp.writeFile(`${basePath}.02`, sequentialBuffer(30, 60));
      await fsp.writeFile(`${basePath}.03`, sequentialBuffer(10, 90));

      await expect(merge(`${basePath}.00`, true)).resolves.toEqual(expect.objectContaining({ type: 'success' }));

      expect(await fsp.readdir(dir)).toEqual(['file.bin']);
      expect(await fsp.readFile(path.join(dir, 'file.bin'))).toEqual(sequentialBuffer(100, 0));
    });
  });

  test('archive; copy', () => {
    return temporaryDirectoryTask<void>(async (dir) => {
      const basePath = path.join(dir, 'file');
      await fsp.mkdir(basePath);
      await fsp.writeFile(`${basePath}/00`, sequentialBuffer(30, 0));
      await fsp.writeFile(`${basePath}/01`, sequentialBuffer(30, 30));
      await fsp.writeFile(`${basePath}/02`, sequentialBuffer(30, 60));
      await fsp.writeFile(`${basePath}/03`, sequentialBuffer(10, 90));

      await expect(merge(`${basePath}/00`, false)).resolves.toEqual(expect.objectContaining({ type: 'success' }));

      expect(await fsp.readdir(dir)).toEqual(['file', 'file_merged']);
      expect(await fsp.readFile(path.join(dir, 'file_merged'))).toEqual(sequentialBuffer(100, 0));
    });
  });

  test('archive; in place', () => {
    return temporaryDirectoryTask<void>(async (dir) => {
      const basePath = path.join(dir, 'file');
      await fsp.mkdir(basePath);
      await fsp.writeFile(`${basePath}/00`, sequentialBuffer(30, 0));
      await fsp.writeFile(`${basePath}/01`, sequentialBuffer(30, 30));
      await fsp.writeFile(`${basePath}/02`, sequentialBuffer(30, 60));
      await fsp.writeFile(`${basePath}/03`, sequentialBuffer(10, 90));

      await merge(`${basePath}/00`, true);

      expect(await fsp.readdir(dir)).toEqual(['file_merged']);
      expect(await fsp.readFile(path.join(dir, 'file_merged'))).toEqual(sequentialBuffer(100, 0));
    });
  });

  test('archive; copy; extension', () => {
    return temporaryDirectoryTask<void>(async (dir) => {
      const basePath = path.join(dir, 'file.nsp');
      await fsp.mkdir(basePath);
      await fsp.writeFile(`${basePath}/00`, sequentialBuffer(30, 0));
      await fsp.writeFile(`${basePath}/01`, sequentialBuffer(30, 30));
      await fsp.writeFile(`${basePath}/02`, sequentialBuffer(30, 60));
      await fsp.writeFile(`${basePath}/03`, sequentialBuffer(10, 90));

      await expect(merge(`${basePath}/00`, false)).resolves.toEqual(expect.objectContaining({ type: 'success' }));

      expect(await fsp.readdir(dir)).toEqual(['file.nsp', 'file_merged.nsp']);
      expect(await fsp.readFile(path.join(dir, 'file_merged.nsp'))).toEqual(sequentialBuffer(100, 0));
    });
  });
});
