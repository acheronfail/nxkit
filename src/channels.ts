import { IpcMainInvokeEvent } from 'electron';
import { FSEntry } from './nand/fatfs/fs';
import { PartitionEntry } from './nand/gpt';

export interface NXKitTegraRcmSmashResult {
  success: boolean;
  stdout: string;
  stderr: string;
}

export interface ExposedPreloadAPIs extends NXKitBridge {
  runTegraRcmSmash: RendererChannelImpl[Channels.TegraRcmSmash];
  findProdKeys: RendererChannelImpl[Channels.findProdKeys];

  nandOpen: RendererChannelImpl[Channels.NandOpen];
  nandClose: RendererChannelImpl[Channels.NandClose];
  nandMount: RendererChannelImpl[Channels.NandMountPartition];
  nandReaddir: RendererChannelImpl[Channels.NandReaddir];
}

export interface NXKitBridge {
  isWindows: boolean;
}

/**
 * List of all IPC channels between main and renderer processes.
 */
export enum Channels {
  PreloadBrige = 'bridge',
  TegraRcmSmash = 'tegraRcmSmash',
  findProdKeys = 'findProdKeys',

  /** Open nand disk image */
  NandOpen = 'nandOpen',
  /** Close nand disk image */
  NandClose = 'nandClose',

  /** Read directory from mounted currently nand partition */
  NandMountPartition = 'NandMountPartition',

  /** Read directory from mounted currently nand partition */
  NandReaddir = 'nandReaddir',
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
  [Channels.findProdKeys]: ChannelImpl<() => string | null>;

  [Channels.NandOpen]: ChannelImpl<(nandPath: string) => PartitionEntry[]>;
  [Channels.NandClose]: ChannelImpl<() => void>;

  [Channels.NandMountPartition]: ChannelImpl<(partitionName: string, keys?: string) => void>;

  [Channels.NandReaddir]: ChannelImpl<(path: string) => FSEntry[]>;
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
