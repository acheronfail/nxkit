/* eslint-disable import/namespace */
import * as FatFs from 'js-fatfs';
import fs from 'fs';
import { XtsCrypto } from './xts';

export function check_result(r: number) {
  if (r !== FatFs.FR_OK) throw new Error(`FatFs error: ${r}`);
}

export class Partition implements FatFs.DiskIO {
  private readonly sectorSize: number;

  constructor(
    private readonly fd: number,
    private readonly start: number,
    private readonly end: number,
    private readonly xts?: XtsCrypto
  ) {
    const buf = this._read(0, 16);
    this.sectorSize = buf.readUInt16LE(11); // BPB_BytsPerSec
  }

  private _read(offset: number, size: number): Buffer {
    const realOffset = this.start + offset;

    if (this.start + offset > this.end) {
      return Buffer.from([]);
    }
    if (offset + size > this.end) {
      size = this.end - offset;
    }

    if (!this.xts) {
      const buf = Buffer.alloc(size, 0);
      fs.readSync(this.fd, buf, 0, size, realOffset);
      return buf;
    }

    const before = offset % 16;
    let after = (offset + size) % 16;
    if (after) after = 16 - after;

    const alignedRealOffset = realOffset - before;
    size = before + size;

    const buf = Buffer.alloc(size, 0);
    fs.readSync(this.fd, buf, 0, size + after, alignedRealOffset);

    return this.xts.decrypt(buf, offset / 0x4000).subarray(before, before + size);
  }

  initialize(_ff: FatFs.FatFs, _pdrv: number) {
    // http://elm-chan.org/fsw/ff/doc/dinit.html
    return 0;
  }

  status(_ff: FatFs.FatFs, _pdrv: number) {
    // http://elm-chan.org/fsw/ff/doc/dstat.html
    return 0;
  }

  read(ff: FatFs.FatFs, pdrv: number, buff: number, sector: number, count: number): number {
    const start = sector * this.sectorSize;
    const end = (sector + count) * this.sectorSize;
    const size = end - start;

    ff.HEAPU8.set(this._read(start, size), buff);
    return FatFs.RES_OK;
  }

  write(ff: FatFs.FatFs, pdrv: number, buff: number, sector: number, count: number): number {
    // TODO: impl
    return FatFs.RES_ERROR;
    // const data = ff.HEAPU8.subarray(buff, buff + count * this.sectorSize);
    // fs.writeSync(this.fd, data, undefined, undefined, sector * this.sectorSize);
    // return FatFs.RES_OK;
  }

  ioctl(ff: FatFs.FatFs, pdrv: number, cmd: number, buff: number): number {
    switch (cmd) {
      case FatFs.CTRL_SYNC:
        return FatFs.RES_OK;
      case FatFs.GET_SECTOR_COUNT:
        // Use `ff.setValue` to write an integer to the FatFs memory.
        ff.setValue(buff, fs.fstatSync(this.fd).size / this.sectorSize, 'i32');
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
