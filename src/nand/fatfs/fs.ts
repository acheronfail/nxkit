import { join } from 'node:path';
import prettyBytes from 'pretty-bytes';
import * as FatFs from 'js-fatfs';
import { BiosParameterblock } from './bpb';

export function check_result(r: number) {
  switch (r) {
    case FatFs.FR_OK:
      return;
    case FatFs.FR_DISK_ERR:
      throw new Error('FatFs Error: FR_DISK_ERR');
    case FatFs.FR_INT_ERR:
      throw new Error('FatFs Error: FR_INT_ERR');
    case FatFs.FR_NOT_READY:
      throw new Error('FatFs Error: FR_NOT_READY');
    case FatFs.FR_NO_FILE:
      throw new Error('FatFs Error: FR_NO_FILE');
    case FatFs.FR_NO_PATH:
      throw new Error('FatFs Error: FR_NO_PATH');
    case FatFs.FR_INVALID_NAME:
      throw new Error('FatFs Error: FR_INVALID_NAME');
    case FatFs.FR_DENIED:
      throw new Error('FatFs Error: FR_DENIED');
    case FatFs.FR_EXIST:
      throw new Error('FatFs Error: FR_EXIST');
    case FatFs.FR_INVALID_OBJECT:
      throw new Error('FatFs Error: FR_INVALID_OBJECT');
    case FatFs.FR_WRITE_PROTECTED:
      throw new Error('FatFs Error: FR_WRITE_PROTECTED');
    case FatFs.FR_INVALID_DRIVE:
      throw new Error('FatFs Error: FR_INVALID_DRIVE');
    case FatFs.FR_NOT_ENABLED:
      throw new Error('FatFs Error: FR_NOT_ENABLED');
    case FatFs.FR_NO_FILESYSTEM:
      throw new Error('FatFs Error: FR_NO_FILESYSTEM');
    case FatFs.FR_MKFS_ABORTED:
      throw new Error('FatFs Error: FR_MKFS_ABORTED');
    case FatFs.FR_TIMEOUT:
      throw new Error('FatFs Error: FR_TIMEOUT');
    case FatFs.FR_LOCKED:
      throw new Error('FatFs Error: FR_LOCKED');
    case FatFs.FR_NOT_ENOUGH_CORE:
      throw new Error('FatFs Error: FR_NOT_ENOUGH_CORE');
    case FatFs.FR_TOO_MANY_OPEN_FILES:
      throw new Error('FatFs Error: FR_TOO_MANY_OPEN_FILES');
    case FatFs.FR_INVALID_PARAMETER:
      throw new Error('FatFs Error: FR_INVALID_PARAMETER');
    default:
      throw new Error(`FatFs Error: ${r}`);
  }
}

export type FSFile = {
  type: 'f';
  name: string;
  path: string;
  size: number;
  sizeHuman: string;
};
export type FSDirectory = { type: 'd'; name: string; path: string };
export type FSEntry = FSFile | FSDirectory;

export enum FatType {
  Fat = FatFs.FM_FAT,
  Fat32 = FatFs.FM_FAT32,
}

export class Fat32FileSystem {
  private readonly fsHandle: number;

  constructor(
    private readonly ff: FatFs.FatFs,
    private readonly bpb: BiosParameterblock,
  ) {
    this.fsHandle = this.ff.malloc(FatFs.sizeof_FATFS);
    check_result(this.ff.f_mount(this.fsHandle, '', 1));
  }

  close() {
    check_result(this.ff.f_unmount(''));
    this.ff.free(this.fsHandle);
  }

  /**
   * Performs a full disk format, re-creating the existing FAT volume
   * @param fatType specify which type of FAT volume to be created
   */
  format(fatType: FatType) {
    // http://elm-chan.org/fsw/ff/doc/mkfs.html
    const opts = {
      // Format as requested FAT type (`fatType`) but without partitioning `FM_SFD`
      fmt: fatType | FatFs.FM_SFD,
      // keep number of fats the same
      n_fat: this.bpb.numFats,
      // keep au size the same,
      au_size: this.bpb.bytsPerSec * this.bpb.secPerClus,
      // leave as default values
      align: 0,
      n_root: 0,
    };

    const work = this.ff.malloc(FatFs.FF_MAX_SS);
    check_result(this.ff.f_mkfs('', opts, work, FatFs.FF_MAX_SS));
    this.ff.free(work);
  }

  /**
   * Extract a file out of the FAT filesystem
   * @param filePath path of file inside FAT filesystem
   * @param onRead callback that's fired each time a chunk of the file is read
   */
  readFile(filePath: string): Uint8Array;
  readFile(filePath: string, onRead: (chunk: Uint8Array) => void): void;
  readFile(filePath: string, onRead?: (chunk: Uint8Array) => void): Uint8Array | void {
    const filePtr = this.ff.malloc(FatFs.sizeof_FIL);
    check_result(this.ff.f_open(filePtr, filePath, FatFs.FA_READ));
    const size = this.ff.f_size(filePtr);
    const buff = this.ff.malloc(size);

    const bytesReadPtr = this.ff.malloc(4);
    this.ff.setValue(bytesReadPtr, 0, 'i32');

    let ret: Buffer | undefined = undefined;
    onRead =
      onRead ??
      (() => {
        let off = 0;
        const buf = (ret = Buffer.alloc(size));
        return (chunk: Uint8Array) => {
          buf.set(chunk, off);
          off += chunk.byteLength;
        };
      })();

    let offset = 0;
    let bytesRead = Infinity;
    do {
      check_result(this.ff.f_read(filePtr, buff, size, bytesReadPtr));
      bytesRead = this.ff.getValue(bytesReadPtr, 'i32');
      onRead(this.ff.HEAPU8.slice(buff + offset, buff + offset + bytesRead));
      offset += bytesRead;
    } while (bytesRead > 0);

    this.ff.free(bytesReadPtr);
    this.ff.free(buff);
    check_result(this.ff.f_close(filePtr));
    this.ff.free(filePtr);

    return ret;
  }

  mkdir(dirPath: string) {
    check_result(this.ff.f_mkdir(dirPath));
  }

  rmdir(dirPath: string) {
    check_result(this.ff.f_rmdir(dirPath));
  }

  /**
   * Read directory inside FAT filesystem
   * @param dirPath path of directory inside FAT filesystem
   */
  readdir(dirPath: string): FSEntry[] {
    const dir = this.ff.malloc(FatFs.sizeof_DIR);
    const fno = this.ff.malloc(FatFs.sizeof_FILINFO);
    check_result(this.ff.f_opendir(dir, dirPath));

    const entries: FSEntry[] = [];
    for (;;) {
      check_result(this.ff.f_readdir(dir, fno));
      const name = this.ff.FILINFO_fname(fno);
      if (name === '') break;

      const path = join(dirPath, name);
      const size = this.ff.FILINFO_fsize(fno);
      const isDir = this.ff.FILINFO_fattrib(fno) & FatFs.AM_DIR;

      entries.push(isDir ? { type: 'd', name, path } : { type: 'f', name, path, size, sizeHuman: prettyBytes(size) });
    }

    // Clean up.
    this.ff.free(fno);
    check_result(this.ff.f_closedir(dir));
    this.ff.free(dir);

    return entries;
  }

  /**
   * Create a file inside FAT filesystem, optionally writing data to it
   * @param filePath path of file inside FAT filesystem
   * @param contents optional contents to write (file is created empty if not provided)
   */
  writeFile(filePath: string, contents?: Uint8Array) {
    const filePtr = this.ff.malloc(FatFs.sizeof_FIL);
    check_result(this.ff.f_open(filePtr, filePath, FatFs.FA_WRITE | FatFs.FA_CREATE_NEW));
    if (contents) {
      const bufOffset = this.ff.malloc(contents.byteLength);
      this.ff.HEAPU8.set(contents, bufOffset);
      const bytesWrittenPtr = this.ff.malloc(4);
      check_result(this.ff.f_write(filePtr, bufOffset, contents.byteLength, bytesWrittenPtr));

      const bytesWritten = this.ff.getValue(bytesWrittenPtr, 'i32');
      if (bytesWritten != contents.byteLength) {
        throw new Error(`expected to write ${contents.byteLength} bytes, but wrote ${bytesWritten}`);
      }

      this.ff.free(bytesWrittenPtr);
      this.ff.free(bufOffset);
    }

    check_result(this.ff.f_close(filePtr));
    this.ff.free(filePtr);
  }
}
