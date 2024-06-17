import * as FatFs from 'js-fatfs';
import { NandIo } from './layer';

export interface PartitionDriverOptions {
  nandIo: NandIo;
  /**
   * Whether or not to treat the disk as readonly
   */
  readonly: boolean;
}

// eslint-disable-next-line import/namespace
export class PartitionDriver implements FatFs.DiskIO {
  private readonly sectorSize: number;
  private readonly nandIo: NandIo;
  private readonly readonly: boolean;

  constructor({ readonly, nandIo }: PartitionDriverOptions) {
    this.readonly = readonly;
    this.nandIo = nandIo;
    const buf = this.nandIo.read(0, 16);
    this.sectorSize = buf.readUInt16LE(11); // BPB_BytsPerSec
  }

  initialize(_ff: FatFs.FatFs, _pdrv: number) {
    // http://elm-chan.org/fsw/ff/doc/dinit.html
    return 0;
  }

  status(_ff: FatFs.FatFs, _pdrv: number) {
    // http://elm-chan.org/fsw/ff/doc/dstat.html
    return 0;
  }

  read(ff: FatFs.FatFs, _pdrv: number, buff: number, sector: number, count: number): number {
    const start = sector * this.sectorSize;
    const end = (sector + count) * this.sectorSize;
    const size = end - start;

    ff.HEAPU8.set(this.nandIo.read(start, size), buff);
    return FatFs.RES_OK;
  }

  write(ff: FatFs.FatFs, _pdrv: number, buff: number, sector: number, count: number): number {
    if (this.readonly) {
      return FatFs.RES_ERROR;
    }

    const data = ff.HEAPU8.subarray(buff, buff + count * this.sectorSize);
    this.nandIo.write(sector * this.sectorSize, data);
    return FatFs.RES_OK;
  }

  ioctl(ff: FatFs.FatFs, _pdrv: number, cmd: number, buff: number): number {
    switch (cmd) {
      case FatFs.CTRL_SYNC:
        return FatFs.RES_OK;
      case FatFs.GET_SECTOR_COUNT:
        // Use `ff.setValue` to write an integer to the FatFs memory.
        ff.setValue(buff, this.nandIo.size() / this.sectorSize, 'i32');
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
