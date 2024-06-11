// See the Electron documentation for details on how to use preload scripts:
// https://www.electronjs.org/docs/latest/tutorial/process-model#preload-scripts

import { contextBridge, ipcRenderer } from 'electron';

declare global {
  interface Window {
    nxkit: NXKitBridge;
    nxkitTegraRcmSmash: NXKitTegraRcmSmash;
  }
}

ipcRenderer.invoke('nxkit:bridge').then((bridge) => contextBridge.exposeInMainWorld('nxkit', bridge));

const nxkitTegraRcmSmash: NXKitTegraRcmSmash = {
  run: async (payloadFilePath: string) => {
    return ipcRenderer.invoke('nxkit:tegra_rcm_smash', payloadFilePath);
  },
};
contextBridge.exposeInMainWorld('nxkitTegraRcmSmash', nxkitTegraRcmSmash);
