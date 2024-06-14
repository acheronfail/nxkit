import { readSync } from 'fs';
import GPT from 'gpt';
import { FileHandle } from './types';
import { EfiPartition, findEfiPartition } from './mbr';

export interface PartitionEntry {
  type: string;
  guid: string;
  name: string;
  firstLBA: bigint;
  lastLBA: bigint;
  attr: bigint;
}

export interface GptTable {
  partitions: PartitionEntry[];
}

export const BLOCK_SIZE = 512;

export function getPartitionTable(handle: FileHandle): GptTable {
  const efiPart = findEfiPartition(handle);
  return readPrimaryGpt(handle, BLOCK_SIZE, efiPart);
}

// TODO: also read backup gpt, and verify both
function readPrimaryGpt(handle: FileHandle, blockSize: number, efiPart: EfiPartition): GptTable {
  const gpt = new GPT({ blockSize });

  // NOTE: For protective GPTs (0xEF), the MBR's partitions
  // attempt to span as much of the device as they can to protect
  // against systems attempting to action on the device,
  // so the GPT is then located at LBA 1, not the EFI partition's first LBA
  const offset = efiPart.type == 0xee ? efiPart.firstLBA * gpt.blockSize : gpt.blockSize;

  // First, we need to read & parse the GPT header, which will declare various
  // sizes and offsets for us to calculate where & how long the table and backup are
  const headerBuffer = Buffer.alloc(gpt.blockSize);

  readSync(handle.fd, headerBuffer, 0, headerBuffer.length, offset);
  gpt.parseHeader(headerBuffer);

  // Now on to reading the actual partition table
  const tableBuffer = Buffer.alloc(gpt.tableSize);
  // NOTE: gpt.tableOffset is a BigInt
  const tableOffset = Number(gpt.tableOffset) * gpt.blockSize;
  readSync(handle.fd, tableBuffer, 0, tableBuffer.length, tableOffset);

  // We need to parse the first 4 partition entries & the rest separately
  // as the first 4 table entries always occupy one block,
  // with the rest following in subsequent blocks
  gpt.parseTable(tableBuffer, 0, gpt.blockSize);
  gpt.parseTable(tableBuffer, gpt.blockSize, gpt.tableSize);

  return gpt;
}
