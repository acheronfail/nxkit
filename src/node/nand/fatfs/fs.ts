import { basename, join } from 'node:path';
import prettyBytes from 'pretty-bytes';
import * as FatFs from 'js-fatfs';
import { BiosParameterblock } from './bpb';
import timers from '../../../timers';

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
  constructor(
    public readonly code: number,
    description: string,
  ) {
    super(`Failed to ${description}: ${errorToString[code] ?? 'Unknown Error'}`);
  }
}

// FIXME: each time this throws, there's a memory leak if malloc'd resources aren't freed
export function check_result(result: number, description: string) {
  if (result === FatFs.FR_OK) {
    return;
  }

  throw new FatError(result, description);
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

export type WriteGetChunk = (size: number) => Uint8Array;

export class Fat32FileSystem {
  private readonly fsHandle: number;

  constructor(
    private readonly ff: FatFs.FatFs,
    private readonly bpb: BiosParameterblock,
    public readonly chunkSize = 16384,
  ) {
    this.fsHandle = this.ff.malloc(FatFs.sizeof_FATFS);
    check_result(this.ff.f_mount(this.fsHandle, '', 1), 'f_mount');
  }

  close() {
    check_result(this.ff.f_unmount(''), 'f_close');
    this.ff.free(this.fsHandle);
  }

  free(): number {
    const freeClustersPtr = this.ff.malloc(4);
    check_result(this.ff.f_getfree('', freeClustersPtr, 0), 'f_getfree');
    const free = this.bpb.bytsPerSec * this.bpb.secPerClus * this.ff.getValue(freeClustersPtr, 'i32');
    this.ff.free(freeClustersPtr);
    return free;
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
    check_result(this.ff.f_mkfs('', opts, work, FatFs.FF_MAX_SS), 'f_mkfs');
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
    check_result(this.ff.f_open(filePtr, filePath, FatFs.FA_READ), 'f_open');
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
      check_result(this.ff.f_read(filePtr, buff + offset, Math.min(this.chunkSize, size), bytesReadPtr), 'f_read');
      bytesRead = this.ff.getValue(bytesReadPtr, 'i32');
      onRead(this.ff.HEAPU8.slice(buff + offset, buff + offset + bytesRead));
      offset += bytesRead;
    } while (bytesRead > 0);

    this.ff.free(bytesReadPtr);
    this.ff.free(buff);
    check_result(this.ff.f_close(filePtr), 'f_close');
    this.ff.free(filePtr);

    return ret;
  }

  /**
   * Create a new directory.
   * @param dirPath path to the directory
   */
  mkdir(dirPath: string, recursive = false) {
    if (!recursive) {
      check_result(this.ff.f_mkdir(dirPath), `f_mkdir ${dirPath}`);
      return;
    }

    const fno = this.ff.malloc(FatFs.sizeof_FILINFO);
    const parts = dirPath.split('/');
    for (let i = 1; i <= parts.length; i++) {
      const partialPath = parts.slice(0, i + 1).join('/');
      const result = this.ff.f_stat(partialPath, fno);
      switch (result) {
        case FatFs.FR_OK:
          // something existed and it wasn't a file
          if (!(this.ff.FILINFO_fattrib(fno) & FatFs.AM_DIR)) {
            throw new FatError(FatFs.FR_EXIST, `f_mkdir ${partialPath}`);
          }
          break;
        // nothing existed, create the directory
        case FatFs.FR_NO_FILE:
          check_result(this.ff.f_mkdir(partialPath), `f_mkdir ${partialPath}`);
          break;
        // something else happened
        default:
          check_result(result, `f_stat ${partialPath}`);
      }
    }

    this.ff.free(fno);
  }

  /**
   * Delete a directory; the directory must be empty.
   * @param dirPath path to the directory
   */
  rmdir(dirPath: string) {
    check_result(this.ff.f_rmdir(dirPath), `f_rmdir ${dirPath}`);
  }

  /**
   * Delete (unlink) a file.
   * @param filePath path to the file
   */
  unlink(filePath: string) {
    check_result(this.ff.f_unlink(filePath), `f_unlink ${filePath}`);
  }

  /**
   * Renames (or moves) a file from src to dst.
   * @param srcPath path to the existing file/directory
   * @param dstPath path to the new location
   */
  rename(srcPath: string, dstPath: string) {
    check_result(this.ff.f_rename(srcPath, dstPath), `f_rename ${srcPath} -> ${dstPath}`);
  }

  /**
   * Delete a file or directory. If it's a directory, then the directory is
   * recursively deleted (all files and folders inside it are deleted).
   * @param path path to file or directory to remove
   */
  remove(path: string) {
    const fno = this.ff.malloc(FatFs.sizeof_FILINFO);
    check_result(this.ff.f_stat(path, fno), `f_stat ${path}`);

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
   * Return information about an entry in the filesystem.
   * @param path path of file or directory inside FAT filesystem
   */
  read(path: string): FSEntry | null {
    const fno = this.ff.malloc(FatFs.sizeof_FILINFO);
    const result = this.ff.f_stat(path, fno);

    let entry: FSEntry | null = null;
    switch (result) {
      case FatFs.FR_OK:
        entry = this.createEntry(path, fno);
        break;
      case FatFs.FR_NO_FILE:
      case FatFs.FR_NO_PATH:
        break;
      default:
        this.ff.free(fno);
        check_result(result, `f_stat ${path}`);
        return null;
    }

    this.ff.free(fno);
    return entry;
  }

  private createEntry(path: string, fno: number): FSEntry {
    const name = basename(path);
    const size = this.ff.FILINFO_fsize(fno);
    const isDir = this.ff.FILINFO_fattrib(fno) & FatFs.AM_DIR;
    return isDir ? { type: 'd', name, path } : { type: 'f', name, path, size, sizeHuman: prettyBytes(size) };
  }

  /**
   * Read directory inside FAT filesystem
   * @param dirPath path of directory inside FAT filesystem
   */
  readdir(dirPath: string): FSEntry[] {
    const dir = this.ff.malloc(FatFs.sizeof_DIR);
    const fno = this.ff.malloc(FatFs.sizeof_FILINFO);
    check_result(this.ff.f_opendir(dir, dirPath), `f_opendir ${dirPath}`);

    const entries: FSEntry[] = [];
    for (;;) {
      check_result(this.ff.f_readdir(dir, fno), `f_readdir ${dirPath}`);
      const name = this.ff.FILINFO_fname(fno);
      if (name === '') break;

      entries.push(this.createEntry(join(dirPath, name), fno));
    }

    // Clean up.
    this.ff.free(fno);
    check_result(this.ff.f_closedir(dir), `f_closedir ${dirPath}`);
    this.ff.free(dir);

    return entries;
  }

  /**
   * Create a file inside FAT filesystem, optionally writing data to it
   * @param filePath path of file inside FAT filesystem
   * @param contents optional contents to write (file is created empty if not provided)
   */
  writeFile(filePath: string, getChunk?: WriteGetChunk, overwrite?: boolean): void;
  writeFile(filePath: string, contents?: Uint8Array, overwrite?: boolean): void;
  writeFile(filePath: string, contentsOrFn?: Uint8Array | WriteGetChunk, overwrite = false): void {
    const filePtr = this.ff.malloc(FatFs.sizeof_FIL);
    check_result(
      this.ff.f_open(filePtr, filePath, FatFs.FA_WRITE | (overwrite ? FatFs.FA_CREATE_ALWAYS : FatFs.FA_CREATE_NEW)),
      `f_open ${filePath}`,
    );

    if (contentsOrFn) {
      const bytesWrittenPtr = this.ff.malloc(4);
      const checkBytesWritten = (expectedLength: number) => {
        const bytesWritten = this.ff.getValue(bytesWrittenPtr, '*');
        if (bytesWritten != expectedLength) {
          throw new Error(`expected to write ${expectedLength} bytes, but wrote ${bytesWritten}`);
        }
      };

      // we've been given the entire data, write it all in one go
      if (typeof contentsOrFn === 'function') {
        const fn: WriteGetChunk = contentsOrFn;
        const bufOffset = this.ff.malloc(this.chunkSize);

        let chunk: Uint8Array | null;
        while ((chunk = fn(this.chunkSize)) && chunk.byteLength > 0) {
          if (chunk.byteLength > this.chunkSize) {
            throw new Error(
              `Provided chunk was too large, expected ${this.chunkSize} or smaller, but got ${chunk.byteLength}`,
            );
          }

          {
            const stop = timers.start('fatfsSetChunk');
            this.ff.HEAPU8.set(chunk, bufOffset);
            stop();
          }

          const stop = timers.start('fatfsWriteChunk');
          check_result(this.ff.f_write(filePtr, bufOffset, chunk.byteLength, bytesWrittenPtr), `f_write ${filePath}`);
          checkBytesWritten(chunk.byteLength);
          stop();
        }

        this.ff.free(bufOffset);
      } else {
        const data: Uint8Array = contentsOrFn;
        const bufOffset = this.ff.malloc(data.byteLength);
        this.ff.HEAPU8.set(data, bufOffset);
        check_result(this.ff.f_write(filePtr, bufOffset, data.byteLength, bytesWrittenPtr), `f_write ${filePath}`);

        checkBytesWritten(data.byteLength);
        this.ff.free(bufOffset);
      }

      this.ff.free(bytesWrittenPtr);
    }

    check_result(this.ff.f_close(filePtr), `f_close ${filePath}`);
    this.ff.free(filePtr);
  }
}
