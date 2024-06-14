import { readSync } from 'fs';
import MBR from 'mbr';
import { FileHandle } from './types';

export interface EfiPartition {
  type: number;
  firstLBA: bigint;
}

export function findEfiPartition(handle: FileHandle): EfiPartition {
  // read mbr
  const buffer = Buffer.alloc(512);
  readSync(handle.fd, buffer, 0, buffer.length, 0);

  // find efi
  const mbr = MBR.parse(buffer);
  return mbr.getEFIPart();
}
