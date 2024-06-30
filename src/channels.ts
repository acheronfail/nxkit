import { IpcMainInvokeEvent } from 'electron';
import { FSEntry, FSFile } from './nand/fatfs/fs';

export interface ProdKeys {
  location: string;
  data: string;
}

export interface Partition {
  id: string;
  name: string;
  mountable: boolean;
  size: number;
  sizeHuman: string;
  free?: number;
  freeHuman?: string;
}

// TODO: do we need the enum, or can we just make them all generic?
export enum NandError {
  None,
  Generic,
  NoProdKeys,
  InvalidProdKeys,
  InvalidPartitionTable,
  NoNandOpened,
  NoPartitionMounted,
  Readonly,
  AlreadyExists,
}

export type NandResult<T = void> =
  | (T extends void ? { error: NandError.None } : { error: NandError.None; data: T })
  | { error: NandError.Generic; description: string }
  | { error: Exclude<NandError, NandError.None | NandError.Generic> };

export interface ExposedPreloadAPIs extends NXKitBridge {
  openLink: RendererChannelImpl[Channels.OpenLink];
  pathDirname: RendererChannelImpl[Channels.PathDirname];
  pathJoin: RendererChannelImpl[Channels.PathJoin];

  runTegraRcmSmash: RendererChannelImpl[Channels.TegraRcmSmash];

  payloadsOpenDirectory: RendererChannelImpl[Channels.PayloadsOpenDirectory];
  payloadsReadFile: RendererChannelImpl[Channels.PayloadsReadFile];
  payloadsCopyIn: RendererChannelImpl[Channels.PayloadsCopyIn];
  payloadsFind: RendererChannelImpl[Channels.PayloadsFind];

  keysFind: RendererChannelImpl[Channels.ProdKeysFind];
  keysSearchPaths: RendererChannelImpl[Channels.ProdKeysSearchPaths];

  nandOpen: RendererChannelImpl[Channels.NandOpen];
  nandClose: RendererChannelImpl[Channels.NandClose];
  nandMount: RendererChannelImpl[Channels.NandMountPartition];
  nandReaddir: RendererChannelImpl[Channels.NandReaddir];
  nandCopyFileOut: RendererChannelImpl[Channels.NandCopyFileOut];
  nandCopyFilesIn: RendererChannelImpl[Channels.NandCopyFilesIn];
  nandCheckExists: RendererChannelImpl[Channels.NandCheckExists];
  nandMoveEntry: RendererChannelImpl[Channels.NandMoveEntry];
  nandDeleteEntry: RendererChannelImpl[Channels.NandDeleteEntry];
  nandFormatPartition: RendererChannelImpl[Channels.NandFormatPartition];
}

export const NXKitBridgeKey = 'nxkit';
export type NXKitBridgeKeyType = typeof NXKitBridgeKey;
export interface NXKitBridge {
  isWindows: boolean;
  isLinux: boolean;
  isOsx: boolean;
}

/**
 * List of all IPC channels between main and renderer processes.
 */
export enum Channels {
  PreloadBridge,

  OpenLink,
  PathDirname,
  PathJoin,

  TegraRcmSmash,

  PayloadsOpenDirectory,
  PayloadsReadFile,
  PayloadsCopyIn,
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
  NandCopyFileOut,
  /** Copy files into the currently mounted nand partition */
  NandCopyFilesIn,
  /** Recursively check for conflicts in the currently mounted nand partition */
  NandCheckExists,
  /** Move an entry inside of the currently mounted nand partition */
  NandMoveEntry,
  /** Delete an entry inside of the currently mounted nand partition */
  NandDeleteEntry,
  /** Reformat nand partition */
  NandFormatPartition,
}

/**
 * Utility type to assist in defining IPC channel type definitions.
 */
// eslint-disable-next-line @typescript-eslint/ban-types
type ChannelImpl<F extends (...args: never[]) => unknown> = [Parameters<F>, Promise<ReturnType<F>>];

/**
 * List of all IPC channel type definitions.
 */
export type ChannelImplDefinition<C extends Channels> = {
  [Channels.PreloadBridge]: ChannelImpl<() => NXKitBridge>;

  [Channels.OpenLink]: ChannelImpl<(link: string) => void>;
  [Channels.PathDirname]: ChannelImpl<(path: string) => string>;
  [Channels.PathJoin]: ChannelImpl<(...parts: string[]) => string>;

  [Channels.TegraRcmSmash]: ChannelImpl<
    (payloadPath: string) => {
      success: boolean;
      stdout: string;
      stderr: string;
    }
  >;

  [Channels.PayloadsOpenDirectory]: ChannelImpl<() => void>;
  [Channels.PayloadsReadFile]: ChannelImpl<(payloadPath: string) => Uint8Array>;
  [Channels.PayloadsCopyIn]: ChannelImpl<(filePaths: string[]) => void>;
  [Channels.PayloadsFind]: ChannelImpl<() => FSFile[]>;

  [Channels.ProdKeysFind]: ChannelImpl<() => ProdKeys | null>;
  [Channels.ProdKeysSearchPaths]: ChannelImpl<() => string[]>;

  [Channels.NandOpen]: ChannelImpl<(nandPath: string) => NandResult<Partition[]>>;
  [Channels.NandClose]: ChannelImpl<() => void>;
  [Channels.NandMountPartition]: ChannelImpl<(partName: string, readonly: boolean, keys?: ProdKeys) => NandResult>;
  [Channels.NandReaddir]: ChannelImpl<(path: string) => NandResult<FSEntry[]>>;
  [Channels.NandCopyFileOut]: ChannelImpl<(pathInNand: string) => NandResult>;
  [Channels.NandCopyFilesIn]: ChannelImpl<(dirPathInNand: string, filePaths: string[]) => NandResult>;
  [Channels.NandCheckExists]: ChannelImpl<(dirPathInNand: string, filePaths: string[]) => NandResult<boolean>>;
  [Channels.NandMoveEntry]: ChannelImpl<(oldPathInNand: string, newPathInNand: string) => NandResult>;
  [Channels.NandDeleteEntry]: ChannelImpl<(pathInNand: string) => NandResult>;
  [Channels.NandFormatPartition]: ChannelImpl<(partName: string, readonly: boolean, keys?: ProdKeys) => NandResult>;
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
