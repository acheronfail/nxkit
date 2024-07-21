import { describe, expect, test, vi } from 'vitest';
import fsp from 'node:fs/promises';
import { join, relative } from 'node:path';
import { temporaryDirectoryTask } from 'tempy';
import { WalkAction, preCopyCheck, checkExistsRecursively, walkHostPaths } from './utils';
import { Fat32FileSystem } from '../fatfs/fs';

const mockFat = () => {
  return vi.mocked({
    read: vi.fn(),
  } satisfies Partial<Fat32FileSystem> as unknown as Fat32FileSystem);
};

const setupMockHostFiles = async (dirPath: string) => {
  await fsp.writeFile(join(dirPath, 'file1'), 'aaa');
  await fsp.writeFile(join(dirPath, 'file2'), 'bbb');
  await fsp.mkdir(join(dirPath, 'dir1'));
  await fsp.mkdir(join(dirPath, 'dir2'));
  await fsp.writeFile(join(dirPath, 'dir2', 'file3'), 'ccc');
  await fsp.writeFile(join(dirPath, 'dir2', 'file4'), 'ddd');
  await fsp.mkdir(join(dirPath, 'dir2', 'dir3'));
  await fsp.writeFile(join(dirPath, 'dir2', 'dir3', 'file5'), 'eee');

  const entries = await fsp.readdir(dirPath);
  return entries.map((e) => join(dirPath, e));
};

describe(preCopyCheck.name, () => {
  test('calculates total size', () => {
    return temporaryDirectoryTask<void>(async (dirPath) => {
      const hostPaths = await setupMockHostFiles(dirPath);
      const result = await preCopyCheck(mockFat(), hostPaths, '/');

      expect(result.totalDirectories).toBe(3);
      expect(result.totalFiles).toBe(5);
      expect(result.totalBytes).toBe(15);
      expect(result.copyPaths).toHaveLength(8);
    });
  });
});

describe(checkExistsRecursively.name, () => {
  test('returns false if nand is empty', () => {
    return temporaryDirectoryTask<void>(async (dirPath) => {
      const hostPaths = await setupMockHostFiles(dirPath);
      await expect(checkExistsRecursively(mockFat(), hostPaths, '/')).resolves.toBe(false);
    });
  });

  test('returns false if nand has files but none are the same', () => {
    return temporaryDirectoryTask<void>(async (dirPath) => {
      const hostPaths = await setupMockHostFiles(dirPath);
      const fat = mockFat();

      fat.read.mockImplementation((path) => {
        switch (path) {
          case '/dir2':
          case '/dir2/dir3':
          case '/dir2/dir3/file5':
          case '/dir2/file3':
          case '/dir2/file4':
          default:
            return null;
        }
      });

      await expect(checkExistsRecursively(fat, hostPaths, '/')).resolves.toBe(false);
    });
  });

  test('returns false if nand has files but directories are the same', () => {
    return temporaryDirectoryTask<void>(async (dirPath) => {
      const hostPaths = await setupMockHostFiles(dirPath);
      const fat = mockFat();

      fat.read.mockImplementation((path) => {
        switch (path) {
          case '/dir2':
            return { type: 'd', name: 'dir2', path };
          case '/dir2/dir3':
            return { type: 'd', name: 'dir3', path };
          case '/dir2/dir3/file5':
          case '/dir2/file3':
          case '/dir2/file4':
          default:
            return null;
        }
      });

      await expect(checkExistsRecursively(fat, hostPaths, '/')).resolves.toBe(false);
    });
  });

  test('returns true if nand has files that are the same', () => {
    return temporaryDirectoryTask<void>(async (dirPath) => {
      const hostPaths = await setupMockHostFiles(dirPath);
      const fat = mockFat();

      fat.read.mockImplementation((path) => {
        switch (path) {
          case '/dir2':
          case '/dir2/dir3':
          case '/dir2/dir3/file5':
          case '/dir2/file3':
            return { type: 'f', path, name: 'file3', size: 0, sizeHuman: '0 B' };
          case '/dir2/file4':
            return { type: 'f', path, name: 'file4', size: 0, sizeHuman: '0 B' };
          default:
            return null;
        }
      });

      await expect(checkExistsRecursively(fat, hostPaths, '/')).resolves.toBe(true);
    });
  });

  test('returns true if nand has a file that is the same path as a directory', () => {
    return temporaryDirectoryTask<void>(async (dirPath) => {
      const hostPaths = await setupMockHostFiles(dirPath);
      const fat = mockFat();

      fat.read.mockImplementation((path) => {
        switch (path) {
          case '/dir2':
            return { type: 'f', path, name: 'dir2', size: 0, sizeHuman: '0 B' };
          case '/dir2/dir3':
          case '/dir2/dir3/file5':
          case '/dir2/file3':
          case '/dir2/file4':
          default:
            return null;
        }
      });

      await expect(checkExistsRecursively(fat, hostPaths, '/')).resolves.toBe(true);
    });
  });
});

describe(walkHostPaths.name, () => {
  test('walks a directory tree', () => {
    return temporaryDirectoryTask<void>(async (dirPath) => {
      const mockHostPaths = await setupMockHostFiles(dirPath);

      const hostPaths: string[] = [];
      const nandPaths: string[] = [];
      const fat = mockFat();
      const nandRoot = '/nand/path';

      await walkHostPaths(fat, mockHostPaths, nandRoot, async ({ pathOnHost, pathInNand }) => {
        hostPaths.push(`/${relative(dirPath, pathOnHost)}`);
        nandPaths.push(pathInNand);
        return WalkAction.Continue;
      });

      expect(hostPaths).toEqual([
        '/dir1',
        '/dir2',
        '/dir2/dir3',
        '/dir2/dir3/file5',
        '/dir2/file3',
        '/dir2/file4',
        '/file1',
        '/file2',
      ]);

      expect(nandPaths).toEqual([
        '/nand/path/dir1',
        '/nand/path/dir2',
        '/nand/path/dir2/dir3',
        '/nand/path/dir2/dir3/file5',
        '/nand/path/dir2/file3',
        '/nand/path/dir2/file4',
        '/nand/path/file1',
        '/nand/path/file2',
      ]);
    });
  });

  test('short circuits', () => {
    return temporaryDirectoryTask<void>(async (dirPath) => {
      const mockHostPaths = await setupMockHostFiles(dirPath);

      const nandPaths: string[] = [];
      const fat = mockFat();
      const nandRoot = '/nand/path';

      await walkHostPaths(fat, mockHostPaths, nandRoot, async ({ pathInNand }) => {
        nandPaths.push(pathInNand);
        return pathInNand.endsWith('/dir2') ? WalkAction.Skip : WalkAction.Stop;
      });

      expect(nandPaths).toEqual(['/nand/path/dir1', '/nand/path/dir2', '/nand/path/file1', '/nand/path/file2']);
    });
  });
});
