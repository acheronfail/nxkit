import { join } from 'node:path';
import prettyBytes from 'pretty-bytes';
import * as FatFs from 'js-fatfs';

export function check_result(r: number) {
  if (r !== FatFs.FR_OK) throw new Error(`FatFs error: ${r}`);
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

export class Fat32FileSystem {
  private readonly fsHandle: number;

  constructor(private readonly ff: FatFs.FatFs) {
    this.fsHandle = this.ff.malloc(FatFs.sizeof_FATFS);
    check_result(this.ff.f_mount(this.fsHandle, '', 1));
  }

  // TODO: ability to format partitions
  // https://github.com/eliboa/NxNandManager/blob/6204efa9d4ace30e7debdeff718c82fda39f1b09/NxNandManager/NxPartition.cpp#L429
  // https://github.com/irori/js-fatfs/blob/2d8ad4f56b39ae3cee569976245d25ed2ba09e8d/src/fatfs.test.ts#L42

  close() {
    check_result(this.ff.f_unmount(''));
    this.ff.free(this.fsHandle);
  }

  /**
   * Extract a file out of the FAT filesystem
   * @param filePath path of file inside FAT filesystem
   * @param onRead callback that's fired each time a chunk of the file is read
   */
  readFile(filePath: string, onRead: (chunk: Uint8Array) => void) {
    const filePtr = this.ff.malloc(FatFs.sizeof_FIL);
    check_result(this.ff.f_open(filePtr, filePath, FatFs.FA_READ));
    const size = this.ff.f_size(filePtr);
    const buff = this.ff.malloc(size);

    const bytesReadPtr = this.ff.malloc(4);
    this.ff.setValue(bytesReadPtr, 0, 'i32');

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
}
