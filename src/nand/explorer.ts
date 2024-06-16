/* eslint-disable @typescript-eslint/no-explicit-any */
import fsPromise from 'fs/promises';
import * as FatFs from 'js-fatfs';
import { PartitionDriver } from './fatfs/diskio';
import { FSEntry, Fat32FileSystem } from './fatfs/fs';
import { FileHandle } from './types';
import { Keys } from '../main/keys';
import { BLOCK_SIZE, PartitionEntry, getPartitionTable } from './gpt';
import { NX_PARTITIONS } from './constants';

// TODO: support reading split dumps

interface Nand {
  handle: FileHandle | null;
  fs: Fat32FileSystem | null;
}

const nand: Nand = {
  handle: null,
  fs: null,
};

export async function close() {
  if (nand.fs) {
    nand.fs.close();
    nand.fs = null;
  }

  if (nand.handle) {
    await nand.handle.close();
    nand.handle = null;
  }
}

export async function open(nandPath: string): Promise<PartitionEntry[]> {
  if (nand.handle) {
    await close();
  }

  const handle = await fsPromise.open(nandPath, 'r');
  nand.handle = handle;

  // NOTE: we don't support all the partitions right now, just the FAT32 ones
  return getPartitionTable(handle).partitions.filter((part) => ['SAFE', 'SYSTEM', 'USER'].includes(part.name));
}

export async function mount(partitionName: string, keys: Keys) {
  if (!nand.handle) throw new Error('No Nand has been opened yet!');

  const { partitions } = getPartitionTable(nand.handle);
  const partition = partitions.find((part) => part.name === partitionName);
  if (!partition) throw new Error(`No partition found with name: ${partitionName}`);

  const partStart = Number(partition.firstLBA) * BLOCK_SIZE;
  const partEnd = Number(partition.lastLBA + 1n) * BLOCK_SIZE;

  const { bisKeyId } = NX_PARTITIONS[partition.type];
  const xtsn = bisKeyId ? keys.getXtsn(bisKeyId) : undefined;

  // TODO: verify keys are correct before attempting to boot up fat32, perform a decryption test
  nand.fs = new Fat32FileSystem(
    await FatFs.create({
      diskio: new PartitionDriver({
        fd: nand.handle.fd,
        partitionStartOffset: partStart,
        partitionEndOffset: partEnd,
        xtsn,
        readonly: true,
      }),
    })
  );
}

export async function readdir(path: string): Promise<FSEntry[]> {
  if (!nand.fs) throw new Error('No partition mounted!');
  return nand.fs.readdir(path);
}
