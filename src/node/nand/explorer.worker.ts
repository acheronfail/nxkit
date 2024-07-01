import { MessageEvent } from 'electron';
import fs from 'node:fs';
import fsp from 'node:fs/promises';
import { basename, join } from 'node:path';
import * as FatFs from 'js-fatfs';
import { PartitionDriver, ReadonlyError } from './fatfs/diskio';
import { FSEntry, Fat32FileSystem, FatError, FatType } from './fatfs/fs';
import { BLOCK_SIZE, GptTable, PartitionEntry, getPartitionTable } from './gpt';
import { NX_PARTITIONS, NxPartition, PartitionFormat, isFat } from './constants';
import { NandIoLayer } from './fatfs/layer';
import { Io, createIo } from './fatfs/io';
import { NandResult, Partition, ProdKeys } from '../../channels';
import prettyBytes from 'pretty-bytes';
import { BiosParameterblock } from './fatfs/bpb';
import { Crypto, NxCrypto } from './fatfs/crypto';
import { Xtsn } from './xtsn';
import { resolveKeys } from '../keys';
import { Keys } from '../keys.types';

export interface Progress {
  totalBytesCopied: number;
  totalFilesCopied: number;
  currentFilePath: string;
  currentFileSize: number;
  currentFileOffset: number;
}

type ProgressUpdate =
  | { action: 'start'; path: string; size: number }
  | { action: 'chunk'; offset: number; size: number }
  | { action: 'finish' };
type ProgressCb = (upd: ProgressUpdate) => void;

class ExplorerError<T> extends Error {
  constructor(public readonly result: Exclude<NandResult<T>, { type: 'success' }>) {
    super(result.error);
  }
}

class Explorer {
  private io: Io | null = null;
  private fs: Fat32FileSystem | null = null;
  private isPackaged = false;

  async close() {
    if (this.fs) {
      this.fs.close();
      this.fs = null;
    }

    if (this.io) {
      this.io.close();
      this.io = null;
    }
  }

  private async operation<T>(op: () => Promise<NandResult<T>>): Promise<NandResult<T>> {
    try {
      return await op();
    } catch (err) {
      console.error(err);

      if (err instanceof ExplorerError) {
        return err.result;
      }

      if (err instanceof ReadonlyError) {
        return { type: 'failure', error: 'Tried to write but the NAND is opened it Read-Only mode' };
      }

      if (err instanceof FatError) {
        if (err.code === FatFs.FR_EXIST) {
          return { type: 'failure', error: 'An item already exists with that name!' };
        }
      }

      return { type: 'failure', error: `An unexpected error occurred: ${err}` };
    }
  }

  private async resolveKeys(keysFromUser?: ProdKeys): Promise<Keys> {
    const keys = await resolveKeys(this.isPackaged, keysFromUser);
    if (!keys) {
      throw new ExplorerError({ type: 'failure', error: 'prod.keys are required but none were found' });
    }

    return keys;
  }

  private getIo(): Io {
    if (!this.io) {
      throw new ExplorerError({ type: 'failure', error: 'No NAND file is currently open.' });
    }

    return this.io;
  }

  private getFat(): Fat32FileSystem {
    if (!this.fs) {
      throw new ExplorerError({ type: 'failure', error: 'No partition is currently mounted.' });
    }

    return this.fs;
  }

  private getPartitionTable(io: Io): GptTable {
    try {
      return getPartitionTable(io);
    } catch (err) {
      console.error(err);
      throw new ExplorerError({
        type: 'failure',
        error: 'Failed to read partition table, is the file a valid NAND dump?',
      });
    }
  }

  private getPartition(io: Io, partName: string): PartitionEntry {
    const { partitions } = this.getPartitionTable(io);
    const partition = partitions.find((part) => part.name === partName);
    if (!partition) {
      throw new ExplorerError({ type: 'failure', error: `No partition found with name: '${partName}'` });
    }

    return partition;
  }

  //
  // Public api
  //

  public async open(nandPath: string, keysFromUser?: ProdKeys): Promise<NandResult<Partition[]>> {
    return this.operation(async () => {
      if (this.io) {
        await this.close();
      }

      this.io = await createIo(nandPath);
      const gpt = this.getPartitionTable(this.io);

      const data: Partition[] = [];
      for (const part of gpt.partitions) {
        const { name, format } = NX_PARTITIONS[part.type];
        const size = Number((part.lastLBA - part.firstLBA) * 512n);
        const sizeHuman = prettyBytes(size);
        const mountable = isFat(format);
        const partition: Partition = { id: part.type, name, mountable, size, sizeHuman };

        try {
          if (mountable) {
            await this.mount(name, true, keysFromUser);
            if (this.fs) {
              const free = this.fs.free();
              partition.free = free;
              partition.freeHuman = prettyBytes(free);
            }
          }
        } catch (err) {
          console.error(`Failed to detect free space on partition: ${name}`);
          console.error(err);
        }

        data.push(partition);
      }

      this.fs?.close();
      this.fs = null;

      return {
        type: 'success',
        data,
      };
    });
  }

  public async mount(partName: string, readonly: boolean, keysFromUser?: ProdKeys): Promise<NandResult> {
    return this.operation(async () => {
      const [{ magicOffset, magicBytes }, nandIo] = await this._createLayer(partName, keysFromUser);
      // verify magic if present, as a way to verify we've got the right prod.keys early
      if (typeof magicOffset === 'number' && magicBytes) {
        const data = nandIo.read(magicOffset, magicBytes.byteLength);
        if (!data.equals(magicBytes)) {
          console.error({
            message: 'magic byte mismatch',
            expected: magicBytes,
            actual: data,
          });
          throw new ExplorerError({
            type: 'failure',
            error: "Failed to decrypt partition, make sure you're using the right prod.keys file",
          });
        }
      }

      const bpb = new BiosParameterblock(nandIo.read(0, 512));
      const diskio = new PartitionDriver({ nandIo, readonly, sectorSize: bpb.bytsPerSec });
      const fatFs = await FatFs.create({ diskio });
      this.fs = new Fat32FileSystem(fatFs, bpb);

      return { type: 'success', data: undefined };
    });
  }

  private async _createLayer(partName: string, keysFromUser?: ProdKeys): Promise<[NxPartition, NandIoLayer]> {
    const io = this.getIo();
    const keys = await this.resolveKeys(keysFromUser);

    const partition = this.getPartition(io, partName);
    const partitionStartOffset = Number(partition.firstLBA) * BLOCK_SIZE;
    const partitionEndOffset = Number(partition.lastLBA + 1n) * BLOCK_SIZE;

    const nxPartition = NX_PARTITIONS[partition.type];
    const { bisKeyId, magicOffset, magicBytes, format } = nxPartition;

    const isClearText =
      magicOffset && magicBytes
        ? io.read(magicOffset + partitionStartOffset, magicBytes.byteLength).equals(magicBytes)
        : false;

    let crypto: Crypto | undefined = undefined;
    if (!isClearText && typeof bisKeyId === 'number') {
      const bisKey = keys.getBisKey(bisKeyId);
      crypto = new NxCrypto(new Xtsn(bisKey.crypto, bisKey.tweak));
    }

    if (!isFat(format)) {
      throw new ExplorerError({
        type: 'failure',
        error: `Unsupported partition format, cannot mount '${partition.name}'`,
      });
    }

    return [
      nxPartition,
      new NandIoLayer({
        io,
        partitionStartOffset,
        partitionEndOffset,
        crypto,
      }),
    ];
  }

  public async readdir(fsPath: string): Promise<NandResult<FSEntry[]>> {
    return this.operation(async () => {
      const fs = this.getFat();
      return { type: 'success', data: fs.readdir(fsPath) };
    });
  }

  public async checkExists(dirPathInNand: string, filePathsOnHost: string[]): Promise<NandResult<boolean>> {
    return this.operation(async () => {
      const fs = this.getFat();

      for (const filePath of filePathsOnHost) {
        if (await checkExistsRecursively(fs, filePath, dirPathInNand)) {
          return { type: 'success', data: true };
        }
      }

      return { type: 'success', data: false };
    });
  }

  // FIXME: fail if combined size of files won't fit in NAND
  public async copyFilesIn(dirPathInNand: string, filePathsOnHost: string[]): Promise<NandResult> {
    const progress: Progress = {
      totalBytesCopied: 0,
      totalFilesCopied: 0,
      currentFilePath: '',
      currentFileSize: 0,
      currentFileOffset: 0,
    };

    return this.operation(async () => {
      const fat = this.getFat();
      for (const filePath of filePathsOnHost) {
        await copyEntryOverwriting(fat, filePath, dirPathInNand, (update) => {
          switch (update.action) {
            case 'start':
              progress.totalFilesCopied++;
              progress.currentFilePath = update.path;
              progress.currentFileSize = update.size;
              progress.currentFileOffset = 0;
              break;
            case 'chunk':
              progress.totalBytesCopied += update.size;
              progress.currentFileOffset = update.offset;
              break;
            case 'finish':
              break;
          }

          process.parentPort.postMessage({ id: 'progress', progress } satisfies OutgoingMessage);
        });
      }

      process.parentPort.postMessage({ id: 'progress', progress: null } satisfies OutgoingMessage);
      return { type: 'success', data: undefined };
    });
  }

  public async copyFileOut(pathInNand: string, destPath: string): Promise<NandResult> {
    return this.operation(async () => {
      const fat = this.getFat();
      const handle = await fsp.open(destPath, 'w+');
      fat.readFile(pathInNand, (chunk) => {
        let bytesWritten = 0;
        while (bytesWritten < chunk.byteLength) {
          bytesWritten += fs.writeSync(handle.fd, chunk, bytesWritten, chunk.byteLength - bytesWritten, null);
        }
      });

      await handle.close();

      return { type: 'success', data: undefined };
    });
  }

  public async format(partName: string, readonly: boolean, keysFromUser?: ProdKeys): Promise<NandResult> {
    return this.operation(async () => {
      // mount fat first so we can format it
      await this.mount(partName, readonly, keysFromUser);
      const [{ format }] = await this._createLayer(partName, keysFromUser);
      const fat = this.getFat();
      fat.format(format === PartitionFormat.Fat32 ? FatType.Fat32 : FatType.Fat);
      return { type: 'success', data: undefined };

      // FIXME: create a "repair" format mode, which doesn't read from the BPB but resets the partition completely
      // e.g., Fat32FileSystem.prototype.format.call(...)
    });
  }

  public async move(oldPathInNand: string, newPathInNand: string): Promise<NandResult> {
    return this.operation(async () => {
      console.log(`move: ${oldPathInNand} -> ${newPathInNand}`);
      this.getFat().rename(oldPathInNand, newPathInNand);
      return { type: 'success', data: undefined };
    });
  }

  public async del(pathInNand: string): Promise<NandResult> {
    return this.operation(async () => {
      this.getFat().remove(pathInNand);
      return { type: 'success', data: undefined };
    });
  }
}

async function checkExistsRecursively(
  fs: Fat32FileSystem,
  pathOnHost: string,
  dirPathInNand: string,
): Promise<boolean> {
  const pathInNand = join(dirPathInNand, basename(pathOnHost));

  const stats = await fsp.stat(pathOnHost);
  const entry = fs.read(pathInNand);

  // there's nothing in the nand here, so no conflict
  if (!entry) return false;

  // if there's a file and an entry of any kind, that's a conflict
  if (stats.isFile() && entry) {
    return true;
  }

  if (stats.isDirectory()) {
    // conflict if exists in the nand and it's not a directory
    if (entry.type !== 'd') return true;

    for (const entry of await fsp.readdir(pathOnHost)) {
      if (await checkExistsRecursively(fs, join(pathOnHost, entry), pathInNand)) {
        return true;
      }
    }
  }

  return false;
}

async function copyEntryOverwriting(
  nandFs: Fat32FileSystem,
  pathOnHost: string,
  dirPathInNand: string,
  onProgress: ProgressCb,
) {
  const pathInNand = join(dirPathInNand, basename(pathOnHost));

  const stats = await fsp.stat(pathOnHost);
  if (stats.isFile()) {
    onProgress({ action: 'start', path: pathOnHost, size: stats.size });
    let offset = 0;
    const handle = await fsp.open(pathOnHost);
    nandFs.writeFile(
      pathInNand,
      (size) => {
        const buf = Buffer.alloc(size);
        const bytesRead = fs.readSync(handle.fd, buf, 0, size, offset);
        offset += bytesRead;

        onProgress({ action: 'chunk', offset, size: stats.size });

        return buf.subarray(0, bytesRead);
      },
      true,
    );
    await handle.close();

    onProgress({ action: 'finish' });
  } else if (stats.isDirectory()) {
    nandFs.mkdir(pathInNand, true);
    for (const entry of await fsp.readdir(pathOnHost)) {
      const entryPath = join(pathOnHost, entry);
      await copyEntryOverwriting(nandFs, entryPath, pathInNand, onProgress);
    }
  } else {
    console.error(`Unsupported type: ${pathOnHost}`);
    return;
  }
}

/**
 * Worker IPC
 */

const explorer = new Explorer();
export type ExplorerIpcDefinition = typeof explorer;
export type ExplorerIpcKey = keyof ExplorerIpcDefinition;

export type IncomingMessage<C extends ExplorerIpcKey = ExplorerIpcKey> =
  | { id: 'bootstrap'; isPackaged: boolean }
  | {
      id: number;
      channel: C;
      args: Parameters<ExplorerIpcDefinition[C]>;
    };

export type OutgoingMessage<C extends ExplorerIpcKey = ExplorerIpcKey> =
  | {
      id: 'progress';
      progress: Progress | null;
    }
  | {
      id: number;
      value: ReturnType<ExplorerIpcDefinition[C]>;
    };

async function handleMessage<C extends ExplorerIpcKey>(msg: IncomingMessage<C>) {
  const { id } = msg;
  if (id === 'bootstrap') {
    explorer['isPackaged'] = msg.isPackaged;
  } else {
    const value = await (explorer[msg.channel] as PromiseIpcHandler)(...msg.args);
    return process.parentPort.postMessage({ id, value });
  }
}

process.parentPort.on('message', (event: MessageEvent) => handleMessage(event.data));
