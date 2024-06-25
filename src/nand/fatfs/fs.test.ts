import { describe, beforeEach, test, expect } from 'vitest';
import * as FatFs from 'js-fatfs';
import { Fat32FileSystem, FatType, check_result } from './fs';
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

describe(Fat32FileSystem.name, () => {
  let fs: Fat32FileSystem;

  beforeEach(async () => {
    const ff = await FatFs.create({ diskio: new MockDisk(new Uint8Array(128 * 1024)) });

    const work = ff.malloc(FatFs.FF_MAX_SS);
    check_result(ff.f_mkfs('', 0, work, FatFs.FF_MAX_SS));
    ff.free(work);

    fs = new Fat32FileSystem(ff, {
      numFats: 2,
      bytsPerSec: 512,
      secPerClus: 1,
    } as unknown as BiosParameterblock);
  });

  test('readdir', () => {
    expect(fs.readdir('/')).toEqual([]);
    fs.writeFile('/empty');
    fs.writeFile('/stuff', Buffer.from('foobar'));
    expect(fs.readdir('/').map(({ path }) => path)).toEqual(['/empty', '/stuff']);
  });

  test('mkdir', () => {
    fs.mkdir('/dir');
    expect(fs.readdir('/')).toEqual([
      {
        type: 'd',
        name: 'dir',
        path: '/dir',
      },
    ]);
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

  test('writeFile empty', () => {
    fs.writeFile('/empty');
    const data = fs.readFile('/empty');
    expect(data.byteLength).toBe(0);
  });

  test('writeFile contents', () => {
    fs.writeFile('/stuff', Buffer.from('asdf'));
    const data = fs.readFile('/stuff');
    expect(data.byteLength).toBe(4);
    expect(data).toEqual(Buffer.from('asdf'));
  });

  test('readFile chunks', () => {
    fs.writeFile('/stuff', Buffer.from('asdf'));
    let data = Buffer.alloc(0);
    fs.readFile('/stuff', (chunk) => (data = Buffer.concat([data, chunk])));
    expect(fs.readFile('/stuff')).toEqual(data);
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
