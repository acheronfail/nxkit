import MBR from 'mbr';
import { Io } from './fatfs/io';

export interface EfiPartition {
  type: number;
  firstLBA: bigint;
}

// https://github.com/jhermsmeier/node-mbr/blob/master/lib/mbr.js
export interface Mbr {
  getEFIPart: () => EfiPartition;
}

export function findEfiPartition(io: Io): EfiPartition {
  const mbr: Mbr = MBR.parse(io.read(0, 512));
  return mbr.getEFIPart();
}
