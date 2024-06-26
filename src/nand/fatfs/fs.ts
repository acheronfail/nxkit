import { join } from 'node:path';
import prettyBytes from 'pretty-bytes';
import * as FatFs from 'js-fatfs';
import { BiosParameterblock } from './bpb';

const errorToString: Record<number, string> = {
  [FatFs.FR_DISK_ERR]: 'FR_DISK_ERR',
  [FatFs.FR_INT_ERR]: 'FR_INT_ERR',
  [FatFs.FR_NOT_READY]: 'FR_NOT_READY',
  [FatFs.FR_NO_FILE]: 'FR_NO_FILE',
  [FatFs.FR_NO_PATH]: 'FR_NO_PATH',
  [FatFs.FR_INVALID_NAME]: 'FR_INVALID_NAME',
  [FatFs.FR_DENIED]: 'FR_DENIED',
  [FatFs.FR_EXIST]: 'FR_EXIST',
  [FatFs.FR_INVALID_OBJECT]: 'FR_INVALID_OBJECT',
  [FatFs.FR_WRITE_PROTECTED]: 'FR_WRITE_PROTECTED',
  [FatFs.FR_INVALID_DRIVE]: 'FR_INVALID_DRIVE',
  [FatFs.FR_NOT_ENABLED]: 'FR_NOT_ENABLED',
  [FatFs.FR_NO_FILESYSTEM]: 'FR_NO_FILESYSTEM',
  [FatFs.FR_MKFS_ABORTED]: 'FR_MKFS_ABORTED',
  [FatFs.FR_TIMEOUT]: 'FR_TIMEOUT',
  [FatFs.FR_LOCKED]: 'FR_LOCKED',
  [FatFs.FR_NOT_ENOUGH_CORE]: 'FR_NOT_ENOUGH_CORE',
  [FatFs.FR_TOO_MANY_OPEN_FILES]: 'FR_TOO_MANY_OPEN_FILES',
  [FatFs.FR_INVALID_PARAMETER]: 'FR_INVALID_PARAMETER',
};

export class FatError extends Error {
  constructor(public readonly code: number) {
    super(`${errorToString[code] ?? 'Unknown'}`);
  }
}

export function check_result(result: number) {
  if (result === FatFs.FR_OK) {
    return;
  }

  throw new FatError(result);
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

    const chunkSize = 4096;
    let offset = 0;
    let bytesRead = Infinity;
    do {
      check_result(this.ff.f_read(filePtr, buff + offset, chunkSize, bytesReadPtr));
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

  /**
   * Create a new directory.
   * @param dirPath path to the directory
   */
  mkdir(dirPath: string) {
    check_result(this.ff.f_mkdir(dirPath));
  }

  /**
   * Delete a directory; the directory must be empty.
   * @param dirPath path to the directory
   */
  rmdir(dirPath: string) {
    check_result(this.ff.f_rmdir(dirPath));
  }

  /**
   * Delete (unlink) a file.
   * @param filePath path to the file
   */
  unlink(filePath: string) {
    check_result(this.ff.f_unlink(filePath));
  }

  /**
   * Renames (or moves) a file from src to dst.
   * @param srcPath path to the existing file/directory
   * @param dstPath path to the new location
   */
  rename(srcPath: string, dstPath: string) {
    check_result(this.ff.f_rename(srcPath, dstPath));
  }

  /**
   * Delete a file or directory. If it's a directory, then the directory is
   * recursively deleted (all files and folders inside it are deleted).
   * @param path path to file or directory to remove
   */
  remove(path: string) {
    const fno = this.ff.malloc(FatFs.sizeof_FILINFO);
    check_result(this.ff.f_stat(path, fno));

    const isDir = this.ff.FILINFO_fattrib(fno) & FatFs.AM_DIR;
    if (isDir) {
      this.readdir(path).forEach((entry) => this.remove(entry.path));
      this.rmdir(path);
    } else {
      this.unlink(path);
    }

    this.ff.free(fno);
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
