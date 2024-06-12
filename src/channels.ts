import { IpcMainInvokeEvent } from 'electron';

export interface NXKitTegraRcmSmashResult {
  success: boolean;
  stdout: string;
  stderr: string;
}

export interface ExposedPreloadAPIs extends NXKitBridge {
  runTegraRcmSmash: RendererChannelImpl[Channels.TegraRcmSmash];
  openNand: RendererChannelImpl[Channels.OpenNand];
  findProdKeys: RendererChannelImpl[Channels.findProdKeys];
}

export interface NXKitBridge {
  isWindows: boolean;
}

/**
 * List of all IPC channels between main and renderer processes.
 */
export enum Channels {
  PreloadBrige = 'bridge',
  TegraRcmSmash = 'tegra_rcm_smash',
  OpenNand = 'open_nand',
  findProdKeys = 'findProdKeys',
}

/**
 * Utility type to assist in defining IPC channel type definitions.
 */
type ChannelImpl<Args extends Array<unknown> = [], Result = void> = [Args, Promise<Result>];

/**
 * List of all IPC channel type definitions.
 */
export type ChannelImplDefinition<C extends Channels> = {
  [Channels.PreloadBrige]: ChannelImpl<[], NXKitBridge>;
  [Channels.TegraRcmSmash]: ChannelImpl<
    [string],
    {
      success: boolean;
      stdout: string;
      stderr: string;
    }
  >;
  [Channels.OpenNand]: ChannelImpl<[string, string | undefined], string>;
  [Channels.findProdKeys]: ChannelImpl<[], string | null>;
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
