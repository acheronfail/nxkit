import { IpcMainInvokeEvent } from 'electron';
import { FSEntry } from './nand/fatfs/fs';
import { PartitionEntry } from './nand/gpt';

export interface ProdKeys {
  location: string;
  data: string;
}

export enum NandError {
  None,
  InvalidProdKeys,
  InvalidPartitionTable,
}

export type NandResult<T = void> =
  | (T extends void ? { error: NandError.None } : { error: NandError.None; data: T })
  | { error: Exclude<NandError, NandError.None> };

export interface ExposedPreloadAPIs extends NXKitBridge {
  runTegraRcmSmash: RendererChannelImpl[Channels.TegraRcmSmash];
  keysFind: RendererChannelImpl[Channels.ProdKeysFind];
  keysSearchPaths: RendererChannelImpl[Channels.ProdKeysSearchPaths];

  nandOpen: RendererChannelImpl[Channels.NandOpen];
  nandClose: RendererChannelImpl[Channels.NandClose];
  nandMount: RendererChannelImpl[Channels.NandMountPartition];
  nandReaddir: RendererChannelImpl[Channels.NandReaddir];
  nandCopyFile: RendererChannelImpl[Channels.NandCopyFile];
}

export interface NXKitBridge {
  isWindows: boolean;
}

/**
 * List of all IPC channels between main and renderer processes.
 */
export enum Channels {
  PreloadBrige,
  TegraRcmSmash,
  ProdKeysFind,
  ProdKeysSearchPaths,

  /** Open nand disk image */
  NandOpen,
  /** Close nand disk image */
  NandClose,
  /** Read directory from currently mounted nand partition */
  NandMountPartition,
  /** Read directory from currently mounted nand partition */
  NandReaddir,
  /** Copy a file out of the currently mounted nand partition */
  NandCopyFile,
}

/**
 * Utility type to assist in defining IPC channel type definitions.
 */
// eslint-disable-next-line @typescript-eslint/ban-types
type ChannelImpl<F extends (...args: unknown[]) => unknown> = [Parameters<F>, Promise<ReturnType<F>>];

/**
 * List of all IPC channel type definitions.
 */
export type ChannelImplDefinition<C extends Channels> = {
  [Channels.PreloadBrige]: ChannelImpl<() => NXKitBridge>;
  [Channels.TegraRcmSmash]: ChannelImpl<
    (payloadPath: string) => {
      success: boolean;
      stdout: string;
      stderr: string;
    }
  >;
  [Channels.ProdKeysFind]: ChannelImpl<() => ProdKeys | null>;
  [Channels.ProdKeysSearchPaths]: ChannelImpl<() => string[]>;

  [Channels.NandOpen]: ChannelImpl<(nandPath: string) => NandResult<PartitionEntry[]>>;
  [Channels.NandClose]: ChannelImpl<() => void>;
  [Channels.NandMountPartition]: ChannelImpl<(partitionName: string, keys?: ProdKeys) => NandResult>;
  [Channels.NandReaddir]: ChannelImpl<(path: string) => FSEntry[]>;
  [Channels.NandCopyFile]: ChannelImpl<(pathInNand: string) => void>;
}[C];

/**
 * Represents all the main IPC channel implementations with their corresponding types.
 */
export type MainChannelImpl = {
  [C in Channels]: (event: IpcMainInvokeEvent, ...args: ChannelImplDefinition<C>[0]) => ChannelImplDefinition<C>[1];
};

export type RendererChannelImpl = {
  [C in Channels]: (...args: ChannelImplDefinition<C>[0]) => ChannelImplDefinition<C>[1];
};
