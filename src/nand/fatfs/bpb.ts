/**
 * http://elm-chan.org/docs/fat_e.html#bpb
 */
export class BiosParameterblock {
  public readonly oemName: string;
  public readonly bytsPerSec: number;
  public readonly secPerClus: number;
  public readonly rsvdSecCnt: number;
  public readonly numFats: number;
  public readonly rootEntCnt: number;
  public readonly totSec16: number;
  public readonly media: number;
  public readonly fatsZ16: number;
  public readonly secPerTrk: number;
  public readonly numHeads: number;
  public readonly hiddSec: number;
  public readonly totSec32: number;

  constructor(bootSector: Buffer) {
    const read = (off: number, len: number) => bootSector.subarray(off, off + len);
    this.oemName = read(3, 8).toString();
    this.bytsPerSec = read(11, 2).readInt16LE();
    this.secPerClus = read(13, 1).readUInt8();
    this.rsvdSecCnt = read(14, 2).readInt16LE();
    this.numFats = read(16, 1).readUInt8();
    this.rootEntCnt = read(17, 2).readInt16LE();
    this.totSec16 = read(19, 2).readInt16LE();
    this.media = read(21, 1).readUInt8();
    this.fatsZ16 = read(22, 2).readInt16LE();
    this.secPerTrk = read(24, 2).readInt16LE();
    this.numHeads = read(26, 2).readInt16LE();
    this.hiddSec = read(28, 4).readInt32LE();
    this.totSec32 = read(32, 4).readInt32LE();
  }
}
