import { stat, readFile, writeFile, readdir } from 'node:fs/promises';
import path from 'node:path';
import { describe, expect, test } from 'vitest';
import { temporaryWriteTask, temporaryDirectoryTask } from 'tempy';
import { CombinedDumpIo, SplitDumpIo } from './io';

describe(CombinedDumpIo.name, () => {
  test('size', () => {
    return temporaryWriteTask<void>(Buffer.alloc(128), async (filePath) => {
      const io = new CombinedDumpIo(filePath);
      expect(io.size()).toBe((await stat(filePath)).size);
      io.close();
    });
  });

  test('read', () => {
    return temporaryWriteTask<void>(Buffer.alloc(128, 42), async (filePath) => {
      const io = new CombinedDumpIo(filePath);
      expect([...io.read(0, 128)]).toEqual([...(await readFile(filePath))]);
      io.close();
    });
  });

  test('read over end', () => {
    return temporaryWriteTask<void>(Buffer.alloc(10, 42), async (filePath) => {
      const io = new CombinedDumpIo(filePath);
      expect([...io.read(0, 100)]).toEqual([...(await readFile(filePath))]);
      io.close();
    });
  });

  test('read past end', () => {
    return temporaryWriteTask(Buffer.alloc(10), async (filePath) => {
      const io = new CombinedDumpIo(filePath);
      expect([...io.read(100, 10)]).toEqual([]);
      io.close();
    });
  });

  test('write', () => {
    return temporaryWriteTask(Buffer.alloc(10, 42), async (filePath) => {
      const io = new CombinedDumpIo(filePath);
      expect(io.write(0, Buffer.alloc(5, 0))).toBe(5);

      expect([...(await readFile(filePath))]).toEqual([...Buffer.alloc(5, 0), ...Buffer.alloc(5, 42)]);
      io.close();
    });
  });

  test('write over end', () => {
    return temporaryWriteTask<void>(Buffer.alloc(10), async (filePath) => {
      const io = new CombinedDumpIo(filePath);
      expect(io.write(0, Buffer.alloc(20, 1))).toBe(10);
      expect([...(await readFile(filePath))]).toEqual([...Buffer.alloc(10, 1)]);
      io.close();
    });
  });

  test('write past end', () => {
    return temporaryWriteTask<void>(Buffer.alloc(10), async (filePath) => {
      const io = new CombinedDumpIo(filePath);
      expect(io.write(100, Buffer.alloc(10))).toBe(0);
      io.close();
    });
  });
});

describe(SplitDumpIo.name, () => {
  const getFiles = async (dirPath: string) => {
    await writeFile(path.join(dirPath, 'file.00'), Buffer.alloc(16, 0));
    await writeFile(path.join(dirPath, 'file.01'), Buffer.alloc(16, 1));
    await writeFile(path.join(dirPath, 'file.02'), Buffer.alloc(16, 2));

    return await readdir(dirPath).then((names) => names.map((n) => path.join(dirPath, n)));
  };

  test('size', () => {
    return temporaryDirectoryTask<void>(async (dirPath) => {
      const files = await getFiles(dirPath);
      const io = new SplitDumpIo(files);
      expect(io.size()).toBe(48);
      io.close();
    });
  });

  test('read', () => {
    return temporaryDirectoryTask<void>(async (dirPath) => {
      const files = await getFiles(dirPath);
      const io = new SplitDumpIo(files);
      expect([...io.read(0, 16)]).toEqual([...(await readFile(files[0]))]);
      io.close();
    });
  });

  test('read over 1 boundary', () => {
    return temporaryDirectoryTask<void>(async (dirPath) => {
      const files = await getFiles(dirPath);
      const io = new SplitDumpIo(files);
      expect([...io.read(0, 32)]).toEqual([...(await readFile(files[0])), ...(await readFile(files[1]))]);
      io.close();
    });
  });

  test('read over 2 boundaries', () => {
    return temporaryDirectoryTask<void>(async (dirPath) => {
      const files = await getFiles(dirPath);
      const io = new SplitDumpIo(files);
      expect([...io.read(0, 48)]).toEqual([
        ...(await readFile(files[0])),
        ...(await readFile(files[1])),
        ...(await readFile(files[2])),
      ]);
      io.close();
    });
  });

  test('read over end', () => {
    return temporaryDirectoryTask<void>(async (dirPath) => {
      const files = await getFiles(dirPath);
      const io = new SplitDumpIo(files);
      expect([...io.read(32, 1000)]).toEqual([...(await readFile(files[2]))]);
      io.close();
    });
  });

  test('read past end', () => {
    return temporaryDirectoryTask<void>(async (dirPath) => {
      const files = await getFiles(dirPath);
      const io = new SplitDumpIo(files);
      expect([...io.read(100, 10)]).toEqual([]);
      io.close();
    });
  });

  test('write', () => {
    return temporaryDirectoryTask<void>(async (dirPath) => {
      const files = await getFiles(dirPath);
      const io = new SplitDumpIo(files);
      expect(io.write(0, Buffer.alloc(16, 42))).toBe(16);
      expect([...(await readFile(files[0]))]).toEqual([...Buffer.alloc(16, 42)]);
      expect([...(await readFile(files[1]))]).toEqual([...Buffer.alloc(16, 1)]);
      expect([...(await readFile(files[2]))]).toEqual([...Buffer.alloc(16, 2)]);
      io.close();
    });
  });

  test('write over 1 boundary', () => {
    return temporaryDirectoryTask<void>(async (dirPath) => {
      const files = await getFiles(dirPath);
      const io = new SplitDumpIo(files);
      expect(io.write(0, Buffer.alloc(32, 42))).toBe(32);
      expect([...(await readFile(files[0]))]).toEqual([...Buffer.alloc(16, 42)]);
      expect([...(await readFile(files[1]))]).toEqual([...Buffer.alloc(16, 42)]);
      expect([...(await readFile(files[2]))]).toEqual([...Buffer.alloc(16, 2)]);
      io.close();
    });
  });

  test('write over 2 boundaries', () => {
    return temporaryDirectoryTask<void>(async (dirPath) => {
      const files = await getFiles(dirPath);
      const io = new SplitDumpIo(files);
      expect(io.write(0, Buffer.alloc(48, 42))).toBe(48);
      expect([...(await readFile(files[0]))]).toEqual([...Buffer.alloc(16, 42)]);
      expect([...(await readFile(files[1]))]).toEqual([...Buffer.alloc(16, 42)]);
      expect([...(await readFile(files[2]))]).toEqual([...Buffer.alloc(16, 42)]);
      io.close();
    });
  });

  test('write over end', () => {
    return temporaryDirectoryTask<void>(async (dirPath) => {
      const files = await getFiles(dirPath);
      const io = new SplitDumpIo(files);
      expect(io.write(32, Buffer.alloc(100, 42))).toBe(16);
      expect([...(await readFile(files[0]))]).toEqual([...Buffer.alloc(16, 0)]);
      expect([...(await readFile(files[1]))]).toEqual([...Buffer.alloc(16, 1)]);
      expect([...(await readFile(files[2]))]).toEqual([...Buffer.alloc(16, 42)]);
      io.close();
    });
  });

  test('write past end', () => {
    return temporaryDirectoryTask<void>(async (dirPath) => {
      const files = await getFiles(dirPath);
      const io = new SplitDumpIo(files);
      expect(io.write(100, Buffer.alloc(10))).toBe(0);
      expect([...(await readFile(files[0]))]).toEqual([...Buffer.alloc(16, 0)]);
      expect([...(await readFile(files[1]))]).toEqual([...Buffer.alloc(16, 1)]);
      expect([...(await readFile(files[2]))]).toEqual([...Buffer.alloc(16, 2)]);
      io.close();
    });
  });
});
