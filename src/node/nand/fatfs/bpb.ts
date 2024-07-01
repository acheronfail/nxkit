export enum FatType {
  Fat12 = 'Fat12',
  Fat16 = 'Fat16',
  Fat32 = 'Fat32',
}

/**
 * http://elm-chan.org/docs/fat_e.html#bpb
 */
export class BiosParameterBlock {
  public readonly fatType: FatType;

  /*
   * Common section
   */

  public readonly bs_jumpBoot: Buffer;
  public readonly bs_oemName: string;
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

  public readonly bs_drvNum: number;
  public readonly bs_reserved: number;
  public readonly bs_bootSig: number;
  public readonly bs_volId: number;
  public readonly bs_volLab: string;
  public readonly bs_filSysType: string;

  public readonly bs_sign: number;

  /*
   * Fat12 and Fat16 only
   */

  public readonly bs_bootCode: Buffer | null = null;

  /*
   * Fat32 only
   */

  public readonly fatsZ32: number | null = null;
  public readonly extFlags: Buffer | null = null;
  public readonly fsVer: Buffer | null = null;
  public readonly rootClus: number | null = null;
  public readonly fsInfo: number | null = null;
  public readonly bkBootSec: number | null = null;
  public readonly reserved: Buffer | null = null;
  public readonly bs_bootCode32: Buffer | null = null;

  constructor(bootSector: Buffer) {
    if (bootSector.byteLength !== 512) {
      throw new Error(
        `BiosParameterBlock input buffer must be 512 bytes, long, but received buffer of length: ${bootSector.byteLength}`,
      );
    }

    const read = (off: number, len: number) => bootSector.subarray(off, off + len);
    this.bs_jumpBoot = read(0, 3);
    this.bs_oemName = read(3, 8).toString();
    this.bytsPerSec = read(11, 2).readUint16LE();
    this.secPerClus = read(13, 1).readUInt8();
    this.rsvdSecCnt = read(14, 2).readUint16LE();
    this.numFats = read(16, 1).readUInt8();
    this.rootEntCnt = read(17, 2).readUint16LE();
    this.totSec16 = read(19, 2).readUint16LE();
    this.media = read(21, 1).readUInt8();
    this.fatsZ16 = read(22, 2).readUint16LE();
    this.secPerTrk = read(24, 2).readUint16LE();
    this.numHeads = read(26, 2).readUint16LE();
    this.hiddSec = read(28, 4).readUint32LE();
    this.totSec32 = read(32, 4).readUint32LE();

    this.bs_sign = read(510, 2).readUInt16LE();

    // if `fatsZ16` it's an indication to read `fatsZ32`, but it still doesn't
    // mean it's Fat32 format!
    if (this.fatsZ16 === 0) {
      this.fatsZ32 = read(36, 4).readUint32LE();
    }

    // this determine which fat type we are
    this.fatType = this.determineFatType();

    switch (this.fatType) {
      case FatType.Fat12:
      case FatType.Fat16:
        this.bs_drvNum = read(36, 1).readUint8();
        this.bs_reserved = read(37, 1).readUint8();
        this.bs_bootSig = read(38, 1).readUint8();
        this.bs_volId = read(39, 4).readUint32LE();
        this.bs_volLab = read(43, 11).toString();
        this.bs_filSysType = read(54, 8).toString();

        this.bs_bootCode = read(62, 448);
        break;
      case FatType.Fat32:
        this.bs_drvNum = read(64, 1).readUint8();
        this.bs_reserved = read(65, 1).readUint8();
        this.bs_bootSig = read(66, 1).readUint8();
        this.bs_volId = read(67, 4).readUint32LE();
        this.bs_volLab = read(71, 11).toString();
        this.bs_filSysType = read(82, 8).toString();

        this.extFlags = read(40, 2);
        this.fsVer = read(42, 2);
        this.rootClus = read(44, 4).readUInt32LE();
        this.fsInfo = read(48, 2).readUint16LE();
        this.bkBootSec = read(50, 2).readUint16LE();
        this.reserved = read(52, 12);
        this.bs_bootCode32 = read(90, 420);
        break;
    }
  }

  public clusterCount(): number {
    const fatStartSector = this.rsvdSecCnt;
    const fatSectors = (this.fatsZ32 ?? this.fatsZ16) * this.numFats;

    const rootDirStartSector = fatStartSector + fatSectors;
    const rootDirSectors = Math.ceil((32 * this.rootEntCnt + this.bytsPerSec - 1) / this.bytsPerSec);

    const dataStartSector = rootDirStartSector + rootDirSectors;
    const dataSectors = (this.totSec16 > 0 ? this.totSec16 : this.totSec32) - dataStartSector;

    return Math.ceil(dataSectors / this.secPerClus);
  }

  public totalSize(): number {
    return this.clusterCount() * this.secPerClus * this.bytsPerSec;
  }

  /**
   * http://elm-chan.org/docs/fat_e.html#fat_determination
   */
  private determineFatType(): FatType {
    const clusterCount = this.clusterCount();
    if (clusterCount <= 4085) return FatType.Fat12;
    if (clusterCount <= 65525) return FatType.Fat16;
    return FatType.Fat32;
  }
}
