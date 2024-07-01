// See the Electron documentation for details on how to use preload scripts:
// https://www.electronjs.org/docs/latest/tutorial/process-model#preload-scripts

import { contextBridge, ipcRenderer } from 'electron';
import { NXKitBridgeKey, NXKitBridge, NXKitBridgeKeyType } from './channels';
import type { MainIpcDefinition } from './main';

type RendererBridge = <C extends keyof MainIpcDefinition>(
  channel: C,
  ...args: Parameters<MainIpcDefinition[C]>
) => ReturnType<MainIpcDefinition[C]>;

declare global {
  interface Window {
    [NXKitBridgeKey]: NXKitBridge & {
      call: RendererBridge;
    };
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const invoke: RendererBridge = (channel, ...args) => ipcRenderer.invoke(channel, ...args) as any;

invoke('preloadBridge').then((bridge) =>
  contextBridge.exposeInMainWorld(NXKitBridgeKey, {
    ...bridge,
    call: invoke,
  } satisfies Window[NXKitBridgeKeyType]),
);
