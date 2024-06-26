// See the Electron documentation for details on how to use preload scripts:
// https://www.electronjs.org/docs/latest/tutorial/process-model#preload-scripts

import { contextBridge, ipcRenderer } from 'electron';
import { Channels, ChannelImplDefinition, ExposedPreloadAPIs, NXKitBridgeKey, NXKitBridgeKeyType } from './channels';

declare global {
  interface Window {
    [NXKitBridgeKey]: ExposedPreloadAPIs;
  }
}

function invoke<C extends Channels>(channel: C, ...args: ChannelImplDefinition<C>[0]): ChannelImplDefinition<C>[1] {
  return ipcRenderer.invoke(channel.toString(), ...args);
}

function exposeInMainWorld<K extends NXKitBridgeKeyType>(key: K, value: Window[K]) {
  contextBridge.exposeInMainWorld(key, value);
}

// -----------------------------------------------------------------------------

invoke(Channels.PreloadBridge).then((bridge) =>
  exposeInMainWorld(NXKitBridgeKey, {
    ...bridge,
    openLink: (link) => invoke(Channels.OpenLink, link),
    pathDirname: (path) => invoke(Channels.PathDirname, path),
    pathJoin: (...parts) => invoke(Channels.PathJoin, ...parts),

    runTegraRcmSmash: (payloadPath) => invoke(Channels.TegraRcmSmash, payloadPath),

    payloadsOpenDirectory: () => invoke(Channels.PayloadsOpenDirectory),
    payloadsReadFile: (payloadPath) => invoke(Channels.PayloadsReadFile, payloadPath),
    payloadsFind: () => invoke(Channels.PayloadsFind),

    keysFind: () => invoke(Channels.ProdKeysFind),
    keysSearchPaths: () => invoke(Channels.ProdKeysSearchPaths),

    nandOpen: (nandPath) => invoke(Channels.NandOpen, nandPath),
    nandClose: () => invoke(Channels.NandClose),
    nandMount: (partName, readonly, keys) => invoke(Channels.NandMountPartition, partName, readonly, keys),
    nandReaddir: (path) => invoke(Channels.NandReaddir, path),
    nandCopyFileOut: (pathInNand) => invoke(Channels.NandCopyFileOut, pathInNand),
    nandCopyFilesIn: (pathInNand, filePaths) => invoke(Channels.NandCopyFilesIn, pathInNand, filePaths),
    nandMoveEntry: (oldPathInNand, newPathInNand) => invoke(Channels.NandMoveEntry, oldPathInNand, newPathInNand),
    nandDeleteEntry: (pathInNand) => invoke(Channels.NandDeleteEntry, pathInNand),
    nandFormatPartition: (partName, readonly, keys) => invoke(Channels.NandFormatPartition, partName, readonly, keys),
  }),
);
