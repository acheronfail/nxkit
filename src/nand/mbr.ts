import MBR from 'mbr';
import { Io } from './fatfs/io';

export interface EfiPartition {
  type: number;
  firstLBA: bigint;
}

export function findEfiPartition(io: Io): EfiPartition {
  const mbr = MBR.parse(io.read(0, 512));
  return mbr.getEFIPart();
}
