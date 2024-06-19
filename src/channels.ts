import { IpcMainInvokeEvent } from 'electron';
import { FSEntry, FSFile } from './nand/fatfs/fs';
import { PartitionEntry } from './nand/gpt';

export interface ProdKeys {
  location: string;
  data: string;
}

export enum NandError {
  None,
  Unknown,
  InvalidProdKeys,
  InvalidPartitionTable,
}

export type NandResult<T = void> =
  | (T extends void ? { error: NandError.None } : { error: NandError.None; data: T })
  | { error: Exclude<NandError, NandError.None> };

export interface ExposedPreloadAPIs extends NXKitBridge {
  runTegraRcmSmash: RendererChannelImpl[Channels.TegraRcmSmash];

  payloadsOpenDirectory: RendererChannelImpl[Channels.PayloadsOpenDirectory];
  payloadsReadFile: RendererChannelImpl[Channels.PayloadsReadFile];
  payloadsFind: RendererChannelImpl[Channels.PayloadsFind];

  keysFind: RendererChannelImpl[Channels.ProdKeysFind];
  keysSearchPaths: RendererChannelImpl[Channels.ProdKeysSearchPaths];

  nandOpen: RendererChannelImpl[Channels.NandOpen];
  nandClose: RendererChannelImpl[Channels.NandClose];
  nandMount: RendererChannelImpl[Channels.NandMountPartition];
  nandReaddir: RendererChannelImpl[Channels.NandReaddir];
  nandCopyFile: RendererChannelImpl[Channels.NandCopyFile];
  nandFormatPartition: RendererChannelImpl[Channels.NandFormatPartition];
}

export interface NXKitBridge {
  isWindows: boolean;
  isLinux: boolean;
  isOsx: boolean;
}

/**
 * List of all IPC channels between main and renderer processes.
 */
export enum Channels {
  PreloadBrige,

  TegraRcmSmash,

  PayloadsOpenDirectory,
  PayloadsReadFile,
  PayloadsFind,

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
  /** Reformat nand partition */
  NandFormatPartition,
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

  [Channels.PayloadsOpenDirectory]: ChannelImpl<() => void>;
  [Channels.PayloadsReadFile]: ChannelImpl<(payloadPath: string) => Uint8Array>;
  [Channels.PayloadsFind]: ChannelImpl<() => FSFile[]>;

  [Channels.ProdKeysFind]: ChannelImpl<() => ProdKeys | null>;
  [Channels.ProdKeysSearchPaths]: ChannelImpl<() => string[]>;

  [Channels.NandOpen]: ChannelImpl<(nandPath: string) => NandResult<PartitionEntry[]>>;
  [Channels.NandClose]: ChannelImpl<() => void>;
  [Channels.NandMountPartition]: ChannelImpl<(partitionName: string, keys?: ProdKeys) => NandResult>;
  [Channels.NandReaddir]: ChannelImpl<(path: string) => FSEntry[]>;
  [Channels.NandCopyFile]: ChannelImpl<(pathInNand: string) => void>;
  [Channels.NandFormatPartition]: ChannelImpl<(partitionName: string, keys?: ProdKeys) => NandResult>;
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
