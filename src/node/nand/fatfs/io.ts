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

  const fd = fs.openSync(nandPath, 'r+');
  return new CombinedDumpIo(fd);
}

/**
 * IO implementation for a combined NAND dump, i.e. `rawnand.bin`.
 */
export class CombinedDumpIo implements Io {
  constructor(private readonly fd: number) {}

  close() {
    fs.closeSync(this.fd);
  }

  size(): number {
    return fs.fstatSync(this.fd).size;
  }

  read(offset: number, length: number): Buffer {
    const buf = Buffer.alloc(length, 0);
    fs.readSync(this.fd, buf, 0, length, offset);
    return buf;
  }

  write(offset: number, data: Buffer): number {
    return fs.writeSync(this.fd, data, 0, data.byteLength, offset);
  }
}

interface SplitFile {
  path: string;
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
      const file: SplitFile = { path, offset, length: stat.size };
      offset += stat.size;
      return file;
    });

    this.totalSize = this.files.reduce((size, { length }) => size + length, 0);
  }

  close() {
    // we don't keep any descriptors open, so nothing to close
  }

  size(): number {
    return this.totalSize;
  }

  findStartingFileIdx(offset: number): number {
    for (let i = 0; i < this.files.length; i++) {
      const file = this.files[i];
      if (offset == file.offset) return i;
      if (offset < file.offset) return i - 1;
      if (offset < file.offset + file.length) return i;
    }

    throw new Error(`Failed to find file for offset: ${offset}`);
  }

  read(offset: number, length: number): Buffer {
    let currentFileIdx = this.findStartingFileIdx(offset);
    let bytesLeftToRead = length;

    const buf = Buffer.alloc(length);
    while (bytesLeftToRead > 0) {
      const splitFile = this.files[currentFileIdx];
      if (!splitFile) {
        // EOF
        break;
      }

      const fd = fs.openSync(splitFile.path, 'r');
      const bytesRemaining = length - bytesLeftToRead;
      const bytesRead = fs.readSync(fd, buf, bytesRemaining, length, offset + bytesRemaining - splitFile.offset);
      bytesLeftToRead -= bytesRead;
      currentFileIdx++;
    }

    return buf;
  }

  write(offset: number, data: Buffer): number {
    let currentFileIdx = this.findStartingFileIdx(offset);
    const length = data.byteLength;
    let bytesLeftToWrite = length;

    while (bytesLeftToWrite > 0) {
      const splitFile = this.files[currentFileIdx];
      if (!splitFile) {
        // EOF
        break;
      }

      const fd = fs.openSync(splitFile.path, 'w+');
      const bytesRemaining = length - bytesLeftToWrite;
      const bytesWritten = fs.writeSync(fd, data, bytesRemaining, length, offset + bytesRemaining - splitFile.offset);
      bytesLeftToWrite -= bytesWritten;
      currentFileIdx++;
    }

    return length;
  }
}
