import * as FatFs from 'js-fatfs';
import { Xtsn } from '../xtsn';
import { XtsnLayer } from './layer';

export interface PartitionDriverOptions {
  /**
   * File descriptor
   */
  fd: number;
  /**
   * Starting byte offset of the partition
   */
  partitionStartOffset: number;
  /**
   * Last byte offset of the partition
   */
  partitionEndOffset: number;
  /**
   * Cryptographic module used for encrypted partitions
   */
  xtsn?: Xtsn;
  /**
   * Whether or not to treat the disk as readonly
   */
  readonly: boolean;
}

// eslint-disable-next-line import/namespace
export class PartitionDriver implements FatFs.DiskIO {
  private readonly sectorSize: number;
  private readonly xtsLayer: XtsnLayer;
  private readonly readonly: boolean;

  constructor(opts: PartitionDriverOptions) {
    this.readonly = opts.readonly;
    this.xtsLayer = new XtsnLayer(opts.fd, opts.partitionStartOffset, opts.partitionEndOffset, opts.xtsn);
    const buf = this.xtsLayer.read(0, 16);
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

    ff.HEAPU8.set(this.xtsLayer.read(start, size), buff);
    return FatFs.RES_OK;
  }

  write(ff: FatFs.FatFs, _pdrv: number, buff: number, sector: number, count: number): number {
    if (this.readonly) {
      return FatFs.RES_ERROR;
    }

    const data = ff.HEAPU8.subarray(buff, buff + count * this.sectorSize);
    this.xtsLayer.write(sector * this.sectorSize, data);
    return FatFs.RES_OK;
  }

  ioctl(ff: FatFs.FatFs, _pdrv: number, cmd: number, buff: number): number {
    switch (cmd) {
      case FatFs.CTRL_SYNC:
        return FatFs.RES_OK;
      case FatFs.GET_SECTOR_COUNT:
        // Use `ff.setValue` to write an integer to the FatFs memory.
        ff.setValue(buff, this.xtsLayer.size() / this.sectorSize, 'i32');
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
