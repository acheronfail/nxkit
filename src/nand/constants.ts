export enum PartitionFormat {
  Unknown,
  Fat12,
  Fat32,
}

export function isFat(pf: PartitionFormat): boolean {
  return [PartitionFormat.Fat12, PartitionFormat.Fat32].includes(pf);
}

/** https://switchbrew.org/wiki/Flash_Filesystem */
export interface NxPartition {
  /** Partition name */
  name: string;
  /** Partition format */
  format: PartitionFormat;
  /** Encryption key id (if present) */
  bisKeyId?: 0 | 1 | 2 | 3;
  /** Offset of magic bytes (used for verifying crypto) */
  magicOffset?: number;
  /** Magic bytes to check */
  magicBytes?: Buffer;
}

const p = (
  format: PartitionFormat,
  name: string,
  bisKeyId?: NxPartition['bisKeyId'],
  magicOffset?: NxPartition['magicOffset'],
  magicBytes?: string,
): NxPartition => ({
  name,
  format,
  bisKeyId,
  magicOffset,
  magicBytes: magicBytes ? Buffer.from(magicBytes) : undefined,
});

// TODO: support BOOT0 and BOOT1 ?
export const NX_PARTITIONS: { [guid: string]: NxPartition } = {
  '98109E25-64E2-4C95-8A77-414916F5BCEB': p(PartitionFormat.Unknown, 'PRODINFO', 0, 0, 'CAL0'),
  'F3056AEC-5449-494C-9F2C-5FDCB75B6E6E': p(PartitionFormat.Fat12, 'PRODINFOF', 0, 0x680, 'CERTIF'),
  '5365DE36-911B-4BB4-8FF9-AA1EBCD73990': p(PartitionFormat.Unknown, 'BCPKG2-1-Normal-Main'),
  '8455717B-BD2B-4162-8454-91695218FC38': p(PartitionFormat.Unknown, 'BCPKG2-2-Normal-Sub'),
  '8ED6C9A6-9C48-490B-BBEB-001D17A4C0F7': p(PartitionFormat.Unknown, 'BCPKG2-3-SafeMode-Main'),
  '5E99751C-56C9-47CC-AA30-B65039888917': p(PartitionFormat.Unknown, 'BCPKG2-4-SafeMode-Sub'),
  'C447D9A2-24B7-468A-98C8-595CD077165A': p(PartitionFormat.Unknown, 'BCPKG2-5-Repair-Main'),
  '9586E1A1-3AA2-4C90-91B3-2F4A5195B4D2': p(PartitionFormat.Unknown, 'BCPKG2-6-Repair-Sub'),
  'A44F9F6B-4ED3-441F-A34A-56AAA136BC6A': p(PartitionFormat.Fat32, 'SAFE', 1, 0x47, 'NO NAME'),
  'ACB0CDF0-4F72-432D-AA0D-5388C733B224': p(PartitionFormat.Fat32, 'SYSTEM', 2, 0x47, 'NO NAME'),
  '2B777F63-E842-47AF-94C4-25A7F18B2280': p(PartitionFormat.Fat32, 'USER', 3, 0x47, 'NO NAME'),
};
