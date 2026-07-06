import GPT from 'gpt';
import { Partition } from 'mbr';
import { findEfiPartition } from './mbr';
import { Io } from './fatfs/io';

export const BLOCK_SIZE = 512;

export function getPartitionTable(io: Io): GPT {
  const efiPart = findEfiPartition(io);
  const primaryGpt = readPrimaryGpt(io, BLOCK_SIZE, efiPart);
  return primaryGpt;
}

export function readBackupGpt(io: Io, primaryGpt: GPT): GPT {
  const backupGpt = new GPT({ blockSize: primaryGpt.blockSize });
  // the backup gpt has the partition table first, followed by the header, so start reading at the table
  const backupGptTableBlockOffset = Number(primaryGpt.backupLBA) - 32;
  const offset = backupGptTableBlockOffset * primaryGpt.blockSize;
  // table takes up 32 blocks, header takes up 1 block
  backupGpt.parseBackup(io.read(offset, 33 * primaryGpt.blockSize));

  return backupGpt;
}

export function repairBackupGptTable(io: Io, primaryGpt: GPT) {
  // create backup gpt table from primary
  const backupGpt = Buffer.alloc(33 * primaryGpt.blockSize);
  primaryGpt.writeBackupFromPrimary(backupGpt);

  // write backup gpt table
  io.write(Number(primaryGpt.backupLBA) * primaryGpt.blockSize - primaryGpt.tableSize, backupGpt);
}

function readPrimaryGpt(io: Io, blockSize: number, efiPart: Partition): GPT {
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
  const tableOffset = Number(gpt.tableOffset) * gpt.blockSize;
  const tableBuffer = io.read(tableOffset, gpt.tableSize);

  // We need to parse the first 4 partition entries & the rest separately
  // as the first 4 table entries always occupy one block,
  // with the rest following in subsequent blocks
  gpt.parseTable(tableBuffer, 0, gpt.blockSize);
  gpt.parseTable(tableBuffer, gpt.blockSize, gpt.tableSize);

  return gpt;
}
