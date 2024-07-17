import MBR, { Partition } from 'mbr';
import { Io } from './fatfs/io';

export function findEfiPartition(io: Io): Partition {
  const mbr: MBR = MBR.parse(io.read(0, 512));
  return mbr.getEFIPart();
}
