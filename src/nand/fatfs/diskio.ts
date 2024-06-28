import * as FatFs from 'js-fatfs';
import { NandIoLayer } from './layer';
import timers from '../../timers';

export interface PartitionDriverOptions {
  nandIo: NandIoLayer;
  /**
   * Whether or not to treat the disk as readonly
   */
  readonly: boolean;
  sectorSize: number;
}

export class ReadonlyError extends Error {}

// eslint-disable-next-line import/namespace
export class PartitionDriver implements FatFs.DiskIO {
  private readonly sectorSize: number;
  private readonly nandIo: NandIoLayer;
  private readonly readonly: boolean;

  constructor({ readonly, nandIo, sectorSize }: PartitionDriverOptions) {
    this.readonly = readonly;
    this.nandIo = nandIo;
    this.sectorSize = sectorSize;
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

    const stop = timers.start('nandIoRead');
    ff.HEAPU8.set(this.nandIo.read(start, size), buff);
    stop();
    return FatFs.RES_OK;
  }

  write(ff: FatFs.FatFs, _pdrv: number, buff: number, sector: number, count: number): number {
    if (this.readonly) {
      throw new ReadonlyError('Tried to write to a disk in readonly mode!');
    }

    const data = ff.HEAPU8.subarray(buff, buff + count * this.sectorSize);
    const stop = timers.start('nandIoWrite');
    this.nandIo.write(sector * this.sectorSize, data);
    stop();
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
