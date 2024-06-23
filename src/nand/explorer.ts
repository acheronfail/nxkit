import fs from 'node:fs';
import { basename } from 'node:path';
import { BrowserWindow, dialog } from 'electron';
import * as FatFs from 'js-fatfs';
import { PartitionDriver, ReadonlyError } from './fatfs/diskio';
import { FSEntry, Fat32FileSystem, FatType } from './fatfs/fs';
import { resolveKeys } from '../main/keys';
import { BLOCK_SIZE, GptTable, PartitionEntry, getPartitionTable } from './gpt';
import { NX_PARTITIONS, PartitionFormat, isFat } from './constants';
import { NandIoLayer } from './fatfs/layer';
import { Io, createIo } from './fatfs/io';
import { NandError, NandResult, Partition, ProdKeys } from '../channels';
import prettyBytes from 'pretty-bytes';
import { BiosParameterblock } from './fatfs/bpb';
import { Crypto, NxCrypto } from './fatfs/crypto';
import { Xtsn } from './xtsn';

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

export async function open(nandPath: string): Promise<NandResult<Partition[]>> {
  if (nand.io) {
    await close();
  }

  nand.io = await createIo(nandPath);

  let gpt: GptTable;
  try {
    gpt = getPartitionTable(nand.io);
  } catch (err) {
    console.error(err);
    return { error: NandError.InvalidPartitionTable };
  }

  return {
    error: NandError.None,
    data: gpt.partitions.map((part) => {
      const { name, format } = NX_PARTITIONS[part.type];
      const mountable = isFat(format);
      const size = Number((part.lastLBA - part.firstLBA) * 512n);
      const sizeHuman = prettyBytes(size);
      return { id: part.type, name, mountable, size, sizeHuman };
    }),
  };
}

function findPartition(partitionName: string): NandResult<PartitionEntry> {
  if (!nand.io) {
    return { error: NandError.NoNandOpened };
  }

  const { partitions } = getPartitionTable(nand.io);
  const partition = partitions.find((part) => part.name === partitionName);
  if (!partition) throw new Error(`No partition found with name: ${partitionName}`);

  return { error: NandError.None, data: partition };
}

export async function mount(partitionName: string, keysFromUser?: ProdKeys): Promise<NandResult> {
  const keys = await resolveKeys(keysFromUser);
  if (!keys) {
    return { error: NandError.NoProdKeys };
  }

  if (!nand.io) {
    return { error: NandError.NoNandOpened };
  }

  const result = findPartition(partitionName);
  if (result.error !== NandError.None) {
    return result;
  }

  const { data: partition } = result;
  const partitionStartOffset = Number(partition.firstLBA) * BLOCK_SIZE;
  const partitionEndOffset = Number(partition.lastLBA + 1n) * BLOCK_SIZE;

  const { bisKeyId, magicOffset, magicBytes, format } = NX_PARTITIONS[partition.type];
  let crypto: Crypto | undefined = undefined;
  if (typeof bisKeyId === 'number') {
    const bisKey = keys.getBisKey(bisKeyId);
    crypto = new NxCrypto(new Xtsn(bisKey.crypto, bisKey.tweak));
  }

  const nandIo = new NandIoLayer({
    io: nand.io,
    partitionStartOffset,
    partitionEndOffset,
    crypto,
  });

  if (!isFat(format)) {
    throw new Error(`Unsupported partition format, cannot mount ${partition.name}`);
  }

  // verify magic if present, as a way to verify we've got the right prod.keys
  if (typeof magicOffset === 'number' && magicBytes) {
    const data = nandIo.read(magicOffset, magicBytes.byteLength);
    if (!data.equals(magicBytes)) {
      return { error: NandError.InvalidProdKeys };
    }
  }

  // TODO: pass readonly in from renderer (with warnings!)
  const readonly = false;

  const bpb = new BiosParameterblock(nandIo.read(0, 512));
  nand.fs = new Fat32FileSystem(
    await FatFs.create({ diskio: new PartitionDriver({ nandIo, readonly, sectorSize: bpb.bytsPerSec }) }),
    bpb,
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

  const result = findPartition(partitionName);
  if (result.error !== NandError.None) {
    return result;
  }

  const { name, format } = NX_PARTITIONS[result.data.type];
  if (!isFat(format)) {
    throw new Error(`Unsupported partition format, cannot format ${name}`);
  }

  try {
    nand.fs.format(format === PartitionFormat.Fat32 ? FatType.Fat32 : FatType.Fat);
    return { error: NandError.None };
  } catch (err) {
    if (err instanceof ReadonlyError) {
      return { error: NandError.Readonly };
    }

    console.error(err);
    return { error: NandError.Unknown };
  }
}
