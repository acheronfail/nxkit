/// <reference types="vite/client" />

/*
 * Imported as URLs by vite
 */

declare module '*.exe' {
  const src: string;
  export default src;
}

declare module '*.nso' {
  const src: string;
  export default src;
}

declare module '*.npdm' {
  const src: string;
  export default src;
}

declare module '*?raw' {
  const text: string;
  export default text;
}

/*
 * Types for untyped packages
 */

declare module 'chs' {
  class CHS {
    constructor(cylinder: number, head: number, sector: number);
  }

  export default CHS;
}

declare module 'mbr' {
  export class Partition {
    type: number;
    firstLBA: number;
    status: number;
    sectors: number;
    firstCHS: CHS;
    lastCHS: CHS;
  }

  // https://github.com/jhermsmeier/node-mbr/blob/master/lib/mbr.js
  class MBR {
    buffer: Buffer;
    partitions: Partition[];
    getEFIPart: () => EfiPartition;
    static createBuffer(): Buffer;
    static parse(buffer: Buffer): MBR;
  }

  export default MBR;
}

/**
 * https://github.com/jhermsmeier/node-gpt/blob/89036390dd401a295566ffdc7ca422f1f075f0af/lib/gpt.js#L15
 */
declare module 'gpt' {
  export class PartitionEntry {
    constructor(options: { type: string; guid: string; name: string; firstLBA: bigint; lastLBA: bigint; attr: bigint });
    type: string;
    guid: string;
    name: string;
    firstLBA: bigint;
    lastLBA: bigint;
    attr: bigint;
  }

  class GPT {
    constructor(options: {
      blockSize: number;
      guid?: string;
      headerSize?: number;
      currentLBA?: bigint;
      backupLBA?: bigint;
      firstLBA?: bigint;
      lastLBA?: bigint;
      tableOffset?: bigint;
      entrySize?: number;
    });
    blockSize: number;
    backupLBA: bigint;
    partitions: PartitionEntry[];
    tableSize: number;
    tableOffset: bigint;
    tableChecksum: number;
    headerChecksum: number;
    checksumTable: () => number;
    verify(): boolean;
    verifyHeader(): boolean;
    verifyTable(): boolean;
    parseHeader(buffer: Buffer): void;
    parseTable(buffer: Buffer, offset?: number, end?: number): void;
    parseBackup(buffer: Buffer): void;
    write(buffer: Buffer, offset?: number): Buffer;
    writeBackupFromPrimary(buffer: Buffer, offset?: number): Buffer;

    static PartitionEntry: typeof PartitionEntry;
  }

  export default GPT;
}

/*
 * Helpers
 */

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type PromiseIpc = Record<string, (...args: any[]) => Promise<any>>;
type PromiseIpcHandler = (...args: unknown[]) => Promise<unknown>;

/*
 * Misc
 */

interface TinfoilResponse {
  data: TinfoilTitle[];
}

interface TinfoilTitle {
  id: string;
  name: string;
  icon: string;
  release_date: string;
  publisher: string;
  size: string;
  user_rating: number;
}
