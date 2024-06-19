import fs from 'node:fs';
import { basename } from 'node:path';
import { BrowserWindow, dialog } from 'electron';
import * as FatFs from 'js-fatfs';
import { PartitionDriver } from './fatfs/diskio';
import { FSEntry, Fat32FileSystem } from './fatfs/fs';
import { resolveKeys } from '../main/keys';
import { BLOCK_SIZE, GptTable, PartitionEntry, getPartitionTable } from './gpt';
import { NX_PARTITIONS } from './constants';
import { NandIo } from './fatfs/layer';
import { Io, createIo } from './fatfs/io';
import { NandError, NandResult, ProdKeys } from '../channels';

interface Nand {
  io: Io | null;
  fs: Fat32FileSystem | null;
}

const nand: Nand = {
  io: null,
  fs: null,
};

export async function close() {
  if (nand.fs) {
    nand.fs.close();
    nand.fs = null;
  }

  if (nand.io) {
    nand.io.close();
    nand.io = null;
  }
}

export async function open(nandPath: string): Promise<NandResult<PartitionEntry[]>> {
  if (nand.io) {
    await close();
  }

  nand.io = await createIo(nandPath);

  let gpt: GptTable;
  try {
    gpt = getPartitionTable(nand.io);
  } catch (err) {
    return { error: NandError.InvalidPartitionTable };
  }

  return { error: NandError.None, data: gpt.partitions };
}

export async function mount(partitionName: string, keysFromUser?: ProdKeys): Promise<NandResult> {
  const keys = await resolveKeys(keysFromUser);
  if (!keys) {
    return { error: NandError.NoProdKeys };
  }

  if (!nand.io) {
    return { error: NandError.NoNandOpened };
  }

  const { partitions } = getPartitionTable(nand.io);
  const partition = partitions.find((part) => part.name === partitionName);
  if (!partition) throw new Error(`No partition found with name: ${partitionName}`);

  const partStart = Number(partition.firstLBA) * BLOCK_SIZE;
  const partEnd = Number(partition.lastLBA + 1n) * BLOCK_SIZE;

  const { bisKeyId, magicOffset, magicBytes } = NX_PARTITIONS[partition.type];
  const xtsn = typeof bisKeyId === 'number' ? keys.getXtsn(bisKeyId) : undefined;

  const nandIo = new NandIo(nand.io, partStart, partEnd, xtsn);

  // verify magic if present, as a way to verify we've got the right prod.keys
  if (typeof magicOffset === 'number' && magicBytes) {
    const data = nandIo.read(magicOffset, magicBytes.byteLength);
    if (!data.equals(magicBytes)) {
      return { error: NandError.InvalidProdKeys };
    }
  }

  nand.fs = new Fat32FileSystem(
    await FatFs.create({
      diskio: new PartitionDriver({
        nandIo: new NandIo(nand.io, partStart, partEnd, xtsn),
        readonly: true,
      }),
    }),
  );

  return { error: NandError.None };
}

export async function readdir(fsPath: string): Promise<NandResult<FSEntry[]>> {
  if (!nand.fs) {
    return { error: NandError.NoPartitionMounted };
  }

  return { error: NandError.None, data: nand.fs.readdir(fsPath) };
}

export async function copyFile(pathInNand: string, window: BrowserWindow): Promise<NandResult> {
  if (!nand.fs) {
    return { error: NandError.NoPartitionMounted };
  }

  const result = await dialog.showSaveDialog(window, {
    defaultPath: basename(pathInNand),
  });
  if (result.canceled) return { error: NandError.None };

  const fd = fs.openSync(result.filePath, 'w+');
  nand.fs.readFile(pathInNand, (chunk) => {
    let bytesWritten = 0;
    while (bytesWritten < chunk.byteLength) {
      bytesWritten += fs.writeSync(fd, chunk, bytesWritten, chunk.byteLength - bytesWritten, null);
    }
  });
  fs.closeSync(fd);

  return { error: NandError.None };
}

export async function format(partitionName: string, keysFromUser?: ProdKeys): Promise<NandResult> {
  await mount(partitionName, keysFromUser);
  if (!nand.fs) {
    return { error: NandError.NoPartitionMounted };
  }

  try {
    nand.fs.format();
    return { error: NandError.None };
  } catch (err) {
    return { error: NandError.Unknown };
  }
}
