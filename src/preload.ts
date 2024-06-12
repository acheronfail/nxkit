// See the Electron documentation for details on how to use preload scripts:
// https://www.electronjs.org/docs/latest/tutorial/process-model#preload-scripts

import { contextBridge, ipcRenderer } from 'electron';
import { Channels, ChannelImplDefinition, ExposedPreloadAPIs } from './channels';

enum ExposedAPIs {
  Bridge = 'nxkit',
}

declare global {
  interface Window {
    [ExposedAPIs.Bridge]: ExposedPreloadAPIs;
  }
}

function invoke<C extends Channels>(channel: C, ...args: ChannelImplDefinition<C>[0]): ChannelImplDefinition<C>[1] {
  return ipcRenderer.invoke(channel, ...args);
}

function exposeInMainWorld<K extends ExposedAPIs>(key: K, value: Window[K]) {
  contextBridge.exposeInMainWorld(key, value);
}

// -----------------------------------------------------------------------------

invoke(Channels.PreloadBrige).then((bridge) =>
  exposeInMainWorld(ExposedAPIs.Bridge, {
    ...bridge,
    runTegraRcmSmash: (payloadPath) => invoke(Channels.TegraRcmSmash, payloadPath),
    openNand: (nandPath, keysFromUser) => invoke(Channels.OpenNand, nandPath, keysFromUser),
    findProdKeys: () => invoke(Channels.findProdKeys),
  })
);
