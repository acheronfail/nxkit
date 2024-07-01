// See the Electron documentation for details on how to use preload scripts:
// https://www.electronjs.org/docs/latest/tutorial/process-model#preload-scripts

import { contextBridge, ipcRenderer } from 'electron';
import { NXKitBridgeKey, NXKitBridge, NXKitBridgeKeyType } from './channels';
import type { ChannelDef } from './main';

type RendererBridge = <C extends keyof ChannelDef>(
  channel: C,
  ...args: Parameters<ChannelDef[C]>
) => ReturnType<ChannelDef[C]>;

declare global {
  interface Window {
    [NXKitBridgeKey]: NXKitBridge & {
      call: RendererBridge;
    };
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const invoke: RendererBridge = (channel, ...args) => ipcRenderer.invoke(channel, ...args) as any;

invoke('PreloadBridge').then((bridge) =>
  contextBridge.exposeInMainWorld(NXKitBridgeKey, {
    ...bridge,
    call: invoke,
  } satisfies Window[NXKitBridgeKeyType]),
);
