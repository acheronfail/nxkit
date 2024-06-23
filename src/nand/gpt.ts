import GPT from 'gpt';
import { EfiPartition, findEfiPartition } from './mbr';
import { Io } from './fatfs/io';

export interface PartitionEntry {
  type: string;
  guid: string;
  name: string;
  firstLBA: bigint;
  lastLBA: bigint;
  attr: bigint;
}

/**
 * https://github.com/jhermsmeier/node-gpt/blob/89036390dd401a295566ffdc7ca422f1f075f0af/lib/gpt.js#L15
 */
export interface GptTable {
  blockSize: number;
  backupLBA: bigint;
  partitions: PartitionEntry[];
  verify(): boolean;
  verifyHeader(): boolean;
  verifyTable(): boolean;
  writeBackupFromPrimary(): Buffer;
}

export const BLOCK_SIZE = 512;

export function getPartitionTable(io: Io): GptTable {
  const efiPart = findEfiPartition(io);
  const primaryGpt = readPrimaryGpt(io, BLOCK_SIZE, efiPart);
  return primaryGpt;
}

function readPrimaryGpt(io: Io, blockSize: number, efiPart: EfiPartition): GptTable {
  const gpt = new GPT({ blockSize });

  // NOTE: For protective GPTs (0xEF), the MBR's partitions
  // attempt to span as much of the device as they can to protect
  // against systems attempting to action on the device,
  // so the GPT is then located at LBA 1, not the EFI partition's first LBA
  const offset = efiPart.type == 0xee ? efiPart.firstLBA * gpt.blockSize : gpt.blockSize;

  // First, we need to read & parse the GPT header, which will declare various
  // sizes and offsets for us to calculate where & how long the table and backup are

  const headerBuffer = io.read(offset, gpt.blockSize);
  gpt.parseHeader(headerBuffer);

  // Now on to reading the actual partition table
  // NOTE: gpt.tableOffset is a BigInt
  const tableOffset = Number(gpt.tableOffset) * gpt.blockSize;
  const tableBuffer = io.read(tableOffset, gpt.tableSize);

  // We need to parse the first 4 partition entries & the rest separately
  // as the first 4 table entries always occupy one block,
  // with the rest following in subsequent blocks
  gpt.parseTable(tableBuffer, 0, gpt.blockSize);
  gpt.parseTable(tableBuffer, gpt.blockSize, gpt.tableSize);

  return gpt;
}

// @ts-expect-error TODO: support repairing the backup gpt + all gpt tables (are they unique or static?)
function _readBackupGpt(io: Io, primaryGpt: GptTable): GptTable {
  const backupGpt = new GPT({ blockSize: primaryGpt.blockSize });
  const offset = (Number(primaryGpt.backupLBA) - 32) * primaryGpt.blockSize;
  const buffer = io.read(offset, 33 * primaryGpt.blockSize);
  backupGpt.parseBackup(buffer);

  return backupGpt;
}
