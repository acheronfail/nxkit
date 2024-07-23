import { basename, dirname, join } from 'node:path';
import fs from 'node:fs';
import fsp from 'node:fs/promises';

/**
 * The interface for a wrapper around reading disk images.
 * NOTE: we have to be sync here and not async, because FatFs only exposes a
 * sync API.
 */
export interface Io {
  close(): void;
  size(): number;
  read(offset: number, length: number): Buffer;
  write(offset: number, data: Buffer): number;
}

export async function createIo(nandPath: string): Promise<Io> {
  const splitFileSuffix = '.00';

  if (nandPath.endsWith(splitFileSuffix)) {
    const dir = dirname(nandPath);
    const prefix = basename(nandPath).slice(0, -splitFileSuffix.length);
    const entries = await fsp.readdir(dir);
    return new SplitDumpIo(
      entries
        .filter((name) => name.startsWith(prefix) && /\.\d\d$/.test(name))
        .map((name) => join(dir, name))
        .sort(),
    );
  }

  return new CombinedDumpIo(nandPath);
}

/**
 * IO implementation for a combined NAND dump, i.e. `rawnand.bin`.
 */
export class CombinedDumpIo implements Io {
  private readonly fd: FdWrapper;

  constructor(filePath: string) {
    this.fd = new FdWrapper(filePath);
  }

  close() {
    this.fd.close();
  }

  size(): number {
    return fs.fstatSync(this.fd.get(false)).size;
  }

  read(offset: number, length: number): Buffer {
    const fileSize = this.size();
    const end = offset + length;
    const readSize = Math.min(fileSize, end) - offset;
    if (readSize <= 0) {
      return Buffer.alloc(0);
    }

    const buf = Buffer.alloc(readSize, 0);
    fs.readSync(this.fd.get(false), buf, 0, readSize, offset);
    return buf;
  }

  write(offset: number, data: Buffer): number {
    const writeLength = Math.min(this.size() - offset, data.byteLength);
    if (writeLength <= 0) {
      return 0;
    }

    return fs.writeSync(this.fd.get(true), data, 0, writeLength, offset);
  }
}

class FdWrapper {
  private fd: number | null;
  private openedForWriting: boolean;
  public readonly filePath: string;

  constructor(filePath: string) {
    this.fd = null;
    this.openedForWriting = false;
    this.filePath = filePath;
  }

  get(forWriting = false): number {
    // if we already have a descriptor...
    if (this.fd !== null) {
      // and it's opened for writing, just use it
      if (this.openedForWriting) {
        return this.fd;
      }

      // if it's only for reading, but we don't want to write, just use it
      if (!forWriting) {
        return this.fd;
      }

      // otherwise we'll need to close it and re-open it for writing
      if (process.env.DEBUG) {
        console.log(`Upgrading file descriptor (${this.fd}) for writing: (${this.filePath})`);
      }
      this.close();
    }

    // https://nodejs.org/api/fs.html#file-system-flags
    this.fd = fs.openSync(this.filePath, forWriting ? 'r+' : 'r');
    this.openedForWriting = forWriting;

    return this.fd;
  }

  close() {
    if (this.fd !== null) {
      fs.closeSync(this.fd);
      this.fd = null;
    }
  }
}

interface SplitFile {
  fd: FdWrapper;
  offset: number;
  length: number;
}

/**
 * IO implementation for a split NAND dump, i.e. `rawnand.bin.00, rawnand.bin.01, ...`
 */
export class SplitDumpIo implements Io {
  private readonly totalSize: number;
  private readonly files: SplitFile[];

  constructor(parts: string[]) {
    let offset = 0;
    this.files = parts.map((path) => {
      const stat = fs.statSync(path);
      const file: SplitFile = { offset, length: stat.size, fd: new FdWrapper(path) };
      offset += stat.size;
      return file;
    });

    this.totalSize = this.files.reduce((size, { length }) => size + length, 0);
  }

  close() {
    this.files.forEach(({ fd }) => fd.close());
  }

  size(): number {
    return this.totalSize;
  }

  findStartingFileIdx(offset: number): number | null {
    for (let i = 0; i < this.files.length; i++) {
      const file = this.files[i];
      if (offset == file.offset) return i;
      if (offset < file.offset) return i - 1;
      if (offset < file.offset + file.length) return i;
    }

    return null;
  }

  read(globalOffset: number, length: number): Buffer {
    const realLength = Math.min(length, this.size() - globalOffset);
    let currentFileIdx = this.findStartingFileIdx(globalOffset);
    if (currentFileIdx === null) {
      // EOF
      return Buffer.alloc(0);
    }

    let bytesRead = 0;
    const buf = Buffer.alloc(realLength);
    while (bytesRead < realLength) {
      const splitFile = this.files[currentFileIdx];
      if (!splitFile) {
        // EOF
        break;
      }

      const fd = splitFile.fd.get(false);
      const localOffset = globalOffset - splitFile.offset + bytesRead;

      const { size } = fs.fstatSync(fd);
      const bytesLeftInFile = size - localOffset;
      if (bytesLeftInFile <= 0) {
        continue;
      }

      const bytesToRead = Math.min(bytesLeftInFile, realLength - bytesRead);
      bytesRead += fs.readSync(fd, buf, bytesRead, bytesToRead, localOffset);
      currentFileIdx++;
    }

    return buf;
  }

  write(globalOffset: number, data: Buffer): number {
    const realLength = Math.min(data.byteLength, this.size() - globalOffset);
    let currentFileIdx = this.findStartingFileIdx(globalOffset);
    if (currentFileIdx === null) {
      // EOF
      return 0;
    }

    let bytesWritten = 0;
    while (bytesWritten < realLength) {
      const splitFile = this.files[currentFileIdx];
      if (!splitFile) {
        // EOF
        break;
      }

      const fd = splitFile.fd.get(true);
      const localOffset = globalOffset - splitFile.offset + bytesWritten;

      const { size } = fs.fstatSync(fd);
      const bytesLeftInFile = size - localOffset;
      if (bytesLeftInFile <= 0) {
        continue;
      }

      const bytesToWrite = Math.min(bytesLeftInFile, realLength - bytesWritten);
      bytesWritten += fs.writeSync(fd, data, bytesWritten, bytesToWrite, localOffset);
      currentFileIdx++;
    }

    return realLength;
  }
}
