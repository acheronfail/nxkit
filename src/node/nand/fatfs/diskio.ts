import * as FatFs from 'js-fatfs';
import { NandIoLayer } from './layer';

export interface PartitionDriverOptions {
  ioLayer: NandIoLayer;
  /**
   * Whether or not to treat the disk as readonly
   */
  readonly: boolean;
}

export class ReadonlyError extends Error {}

// eslint-disable-next-line import/namespace
export class NxDiskIo implements FatFs.DiskIO {
  private readonly blockSize: number;
  private readonly sectorSize: number;
  private readonly sectorCount: number;
  private readonly ioLayer: NandIoLayer;
  private readonly readonly: boolean;

  constructor({ readonly, ioLayer }: PartitionDriverOptions) {
    this.readonly = readonly;
    this.ioLayer = ioLayer;
    this.blockSize = ioLayer.blockSize;
    this.sectorSize = ioLayer.sectorSize;
    this.sectorCount = ioLayer.sectorCount;
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

    ff.HEAPU8.set(this.ioLayer.read(start, size), buff);
    return FatFs.RES_OK;
  }

  write(ff: FatFs.FatFs, _pdrv: number, buff: number, sector: number, count: number): number {
    if (this.readonly) {
      throw new ReadonlyError('Tried to write to a disk in readonly mode!');
    }

    const bytesToWrite = new Uint8Array(ff.HEAPU8.buffer, buff, count * this.sectorSize);
    this.ioLayer.write(sector * this.sectorSize, bytesToWrite);
    return FatFs.RES_OK;
  }

  ioctl(ff: FatFs.FatFs, _pdrv: number, cmd: number, buff: number): number {
    switch (cmd) {
      case FatFs.CTRL_SYNC:
        return FatFs.RES_OK;
      case FatFs.GET_SECTOR_COUNT:
        ff.setValue(buff, this.sectorCount, 'i32');
        return FatFs.RES_OK;
      case FatFs.GET_SECTOR_SIZE:
        ff.setValue(buff, this.sectorSize, 'i16');
        return FatFs.RES_OK;
      case FatFs.GET_BLOCK_SIZE:
        ff.setValue(buff, this.blockSize, 'i32');
        return FatFs.RES_OK;
      default:
        console.warn(`ioctl(${cmd}): not implemented`);
        return FatFs.RES_ERROR;
    }
  }
}
