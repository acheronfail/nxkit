import { describe, beforeEach, test, expect } from 'vitest';
import * as FatFs from 'js-fatfs';
import { FSFile, Fat32FileSystem, FatType } from './fs';
import { BiosParameterblock } from './bpb';

class MockDisk implements FatFs.DiskIO {
  readonly sectorSize = 512;

  constructor(private readonly buf: Uint8Array) {}

  initialize(_ff: FatFs.FatFs, _pdrv: number): number {
    return 0;
  }
  status(_ff: FatFs.FatFs, _pdrv: number): number {
    return 0;
  }
  read(ff: FatFs.FatFs, _pdrv: number, buff: number, sector: number, count: number): number {
    ff.HEAPU8.set(new Uint8Array(this.buf.buffer, sector * this.sectorSize, count * this.sectorSize), buff);
    return FatFs.RES_OK;
  }
  write(ff: FatFs.FatFs, _pdrv: number, buff: number, sector: number, count: number): number {
    this.buf.set(new Uint8Array(ff.HEAPU8.buffer, buff, count * this.sectorSize), sector * this.sectorSize);
    return FatFs.RES_OK;
  }
  ioctl(ff: FatFs.FatFs, _pdrv: number, cmd: number, buff: number): number {
    switch (cmd) {
      case FatFs.CTRL_SYNC:
        return FatFs.RES_OK;
      case FatFs.GET_SECTOR_COUNT:
        ff.setValue(buff, this.buf.byteLength / this.sectorSize, 'i32');
        return FatFs.RES_OK;
      case FatFs.GET_SECTOR_SIZE:
        ff.setValue(buff, this.sectorSize, 'i16');
        return FatFs.RES_OK;
      case FatFs.GET_BLOCK_SIZE:
        ff.setValue(buff, 1, 'i32');
        return FatFs.RES_OK;
      default:
        console.warn(`ioctl(${cmd}): not implemented`);
        return FatFs.RES_ERROR;
    }
  }
}

const getABuffer = (size: number) => {
  const buf = Buffer.alloc(size);
  for (let i = 0; i < size; i++) buf[i] = size - i;
  return buf;
};

describe(Fat32FileSystem.name, () => {
  let fs: Fat32FileSystem;

  beforeEach(async () => {
    const ff = await FatFs.create({ diskio: new MockDisk(new Uint8Array(128 * 1024)) });

    const work = ff.malloc(FatFs.FF_MAX_SS);
    expect(ff.f_mkfs('', 0, work, FatFs.FF_MAX_SS)).toBe(FatFs.FR_OK);
    ff.free(work);

    const chunkSize = 128;
    const bpb = {
      numFats: 2,
      bytsPerSec: 512,
      secPerClus: 1,
    } as unknown as BiosParameterblock;

    fs = new Fat32FileSystem(ff, bpb, chunkSize);
  });

  test('free', () => {
    expect(fs.free()).toEqual(81920 - 512);
    fs.writeFile('/stuff', getABuffer(512));
    expect(fs.free()).toEqual(81920 - 512 - 512);
    fs.writeFile('/stuff', getABuffer(513), true);
    expect(fs.free()).toEqual(81920 - 512 - 512 - 512);
  });

  test('readdir', () => {
    expect(fs.readdir('/')).toEqual([]);
    fs.writeFile('/empty');
    fs.writeFile('/stuff', Buffer.from('foobar'));
    expect(fs.readdir('/').map(({ path }) => path)).toEqual(['/empty', '/stuff']);
  });

  describe('mkdir', () => {
    test('normal', () => {
      fs.mkdir('/dir');
      expect(fs.readdir('/')).toEqual([
        {
          type: 'd',
          name: 'dir',
          path: '/dir',
        },
      ]);
    });

    test('recursive', () => {
      fs.mkdir('/path/to/sub/directory', true);
      expect(fs.readdir('/').map(({ name }) => name)).toEqual(['path']);
      expect(fs.readdir('/path').map(({ name }) => name)).toEqual(['to']);
      expect(fs.readdir('/path/to').map(({ name }) => name)).toEqual(['sub']);
      expect(fs.readdir('/path/to/sub').map(({ name }) => name)).toEqual(['directory']);
      expect(fs.readdir('/path/to/sub/directory')).toEqual([]);
    });

    test('recursive with pre-existing dirs', () => {
      fs.mkdir('/path');
      fs.mkdir('/path/to');

      fs.mkdir('/path/to/sub/directory', true);
      expect(fs.readdir('/').map(({ name }) => name)).toEqual(['path']);
      expect(fs.readdir('/path').map(({ name }) => name)).toEqual(['to']);
      expect(fs.readdir('/path/to').map(({ name }) => name)).toEqual(['sub']);
      expect(fs.readdir('/path/to/sub').map(({ name }) => name)).toEqual(['directory']);
      expect(fs.readdir('/path/to/sub/directory')).toEqual([]);
    });

    test('recursive with file conflict', () => {
      fs.mkdir('/path');
      fs.mkdir('/path/to');
      fs.writeFile('/path/to/sub');

      expect(() => fs.mkdir('/path/to/sub/directory', true)).toThrowError('Failed to f_mkdir /path/to/sub: FR_EXIST');
    });
  });

  describe('read', () => {
    test('not exists', () => {
      expect(fs.read('/asdf')).toBe(null);
    });

    test('a file', () => {
      fs.writeFile('/asdf');
      expect(fs.read('/asdf')).toEqual({
        type: 'f',
        name: 'asdf',
        path: '/asdf',
        size: 0,
        sizeHuman: '0 B',
      });
    });

    test('a directory', () => {
      fs.mkdir('/asdf');
      expect(fs.read('/asdf')).toEqual({
        type: 'd',
        name: 'asdf',
        path: '/asdf',
      });
    });
  });

  test('rmdir', () => {
    fs.mkdir('/dir');
    expect(fs.readdir('/').map(({ path }) => path)).toEqual(['/dir']);
    fs.rmdir('/dir');
    expect(fs.readdir('/')).toEqual([]);
  });

  describe('rename', () => {
    test('file', () => {
      fs.writeFile('/file');
      expect(fs.readdir('/').map(({ path }) => path)).toEqual(['/file']);
      fs.rename('/file', '/new');
      expect(fs.readdir('/').map(({ path }) => path)).toEqual(['/new']);
    });

    test('dir', () => {
      fs.mkdir('/dir');
      expect(fs.readdir('/').map(({ path }) => path)).toEqual(['/dir']);
      fs.rename('/dir', '/new');
      expect(fs.readdir('/').map(({ path }) => path)).toEqual(['/new']);
    });

    test('across directories', () => {
      fs.mkdir('/dir');
      fs.writeFile('/file');
      expect(fs.readdir('/').map(({ path }) => path)).toEqual(['/dir', '/file']);
      fs.rename('/file', '/dir/new');
      expect(fs.readdir('/').map(({ name }) => name)).toEqual(['dir']);
      expect(fs.readdir('/dir').map(({ name }) => name)).toEqual(['new']);
    });
  });

  test('remove', () => {
    fs.mkdir('/first');
    fs.mkdir('/first/second');
    fs.writeFile('/first/empty');
    fs.writeFile('/first/stuff', Buffer.from('asdf'));
    fs.writeFile('/first/second/stuff', Buffer.from('asdf'));

    expect(fs.readdir('/').map(({ path }) => path)).toEqual(['/first']);
    expect(fs.readdir('/first').map(({ name }) => name)).toEqual(['second', 'empty', 'stuff']);
    expect(fs.readdir('/first/second').map(({ name }) => name)).toEqual(['stuff']);

    fs.remove('/first');
    expect(fs.readdir('/')).toEqual([]);
  });

  describe('writeFile', () => {
    test('empty', () => {
      fs.writeFile('/empty');
      const data = fs.readFile('/empty');
      expect(data.byteLength).toBe(0);
    });

    // TODO: test conflict and overwrite

    test('contents', () => {
      fs.writeFile('/stuff', Buffer.from('asdf'));
      const data = fs.readFile('/stuff');
      expect(data.byteLength).toBe(4);
      expect(data).toEqual(Buffer.from('asdf'));
    });

    test('chunks', () => {
      const buf = getABuffer(10_000);
      let offset = 0;
      fs.writeFile('/stuff', (size) => {
        const slice = buf.subarray(offset, offset + size);
        offset += size;
        return slice;
      });

      const data = fs.read('/stuff') as FSFile;
      expect(data?.type).toBe('f');
      expect(data?.size).toBe(10_000);
    });
  });

  test('readFile chunks', () => {
    const buf = getABuffer(10_000);
    fs.writeFile('/stuff', buf);

    const readInOneGo = fs.readFile('/stuff');
    const readInChunks = Buffer.alloc(buf.byteLength);
    let offset = 0;
    fs.readFile('/stuff', (chunk) => {
      readInChunks.set(chunk, offset);
      offset += chunk.byteLength;
    });

    expect(readInChunks).toEqual(buf);
    expect(readInOneGo).toEqual(readInChunks);
  });

  test('format', () => {
    fs.writeFile('/foo', Buffer.from('1'));
    fs.mkdir('/dir');
    fs.writeFile('/dir/foo', Buffer.from('2'));
    expect(fs.readdir('/').map(({ path }) => path)).toEqual(['/foo', '/dir']);
    fs.format(FatType.Fat);
    expect(fs.readdir('/')).toEqual([]);
  });
});
