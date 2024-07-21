// See the Electron documentation for details on how to use preload scripts:
// https://www.electronjs.org/docs/latest/tutorial/process-model#preload-scripts

import { contextBridge, ipcRenderer } from 'electron';
import { NXKitBridgeKey, NXKitBridge, NXKitBridgeKeyType } from './channels';
import type { MainIpcDefinition } from './main';
import type { Progress } from './node/nand/explorer/worker';

type RendererBridge = <C extends keyof MainIpcDefinition>(
  channel: C,
  ...args: Parameters<MainIpcDefinition[C]>
) => ReturnType<MainIpcDefinition[C]>;

type OnProgress = (progress: Progress) => void;

declare global {
  interface Window {
    [NXKitBridgeKey]: NXKitBridge & {
      call: RendererBridge;
      progressSubscribe: (fn: OnProgress) => void;
    };
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const invoke: RendererBridge = (channel, ...args) => ipcRenderer.invoke(channel, ...args) as any;

invoke('preloadBridge').then((bridge) => {
  let onProgress: OnProgress | undefined;
  ipcRenderer.on('progress', (_, progress) => onProgress?.(progress));

  contextBridge.exposeInMainWorld(NXKitBridgeKey, {
    ...bridge,
    call: invoke,
    progressSubscribe: (fn) => (onProgress = fn),
  } satisfies Window[NXKitBridgeKeyType]);
});
