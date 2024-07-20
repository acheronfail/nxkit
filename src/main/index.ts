import { app, BrowserWindow, dialog, ipcMain, shell } from 'electron';
import { platform } from 'node:os';
import cp from 'node:child_process';
import { createRequire } from 'node:module';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import { ProdKeys } from '../channels';
import { findProdKeys } from '../node/keys';
import * as payloads from './payloads';
import automaticContextMenus from 'electron-context-menu';
import { getResources } from '../resources';
import { ExplorerController } from './explorer';
import { merge, split } from '../node/split';

const require = createRequire(import.meta.url);
const __dirname = path.dirname(fileURLToPath(import.meta.url));

// Handle creating/removing shortcuts on Windows when installing/uninstalling.
if (require('electron-squirrel-startup')) {
  app.quit();
}

automaticContextMenus({});

function loadWindow(window: BrowserWindow, name: string, params?: URLSearchParams) {
  if (RENDERER_VITE_DEV_SERVER_URL) {
    window.loadURL(`${RENDERER_VITE_DEV_SERVER_URL}/src/${name}/index.html?${params}`);
  } else {
    window.loadFile(path.join(__dirname, `../${RENDERER_VITE_NAME}/src/${name}/index.html?${params}`));
  }
}

const createMainWindow = (): BrowserWindow => {
  const win = new BrowserWindow({
    width: app.isPackaged ? 800 : 1600,
    height: 800,
    show: false,
    backgroundColor: '#1e293b',
    titleBarStyle: 'hiddenInset',
    webPreferences: {
      preload: path.join(__dirname, 'preload.cjs'),
    },
  });

  win.once('ready-to-show', () => win.show());

  // quick 'n' dirty cli handling
  const argString = process.argv.slice(2).join(' ');
  const params = new URLSearchParams();
  const addMatch = (name: string, re: RegExp) => {
    const match = re.exec(argString);
    if (match) {
      params.set(name, match[1]);
    }
  };

  addMatch('tab', /--tab[= ](\S+)/);
  addMatch('rawnand', /--rawnand[= ](\S+)/);
  loadWindow(win, 'window_main', params);

  if (!app.isPackaged) {
    win.webContents.openDevTools();
  }

  // open links externally (requires target="_blank" on all links in renderer)
  win.webContents.setWindowOpenHandler(({ url }) => {
    shell.openExternal(url);
    return { action: 'deny' };
  });

  // Automatically select a pre-mariko Switch in RCM
  win.webContents.session.on('select-usb-device', (_event, details, callback) => {
    callback(details.deviceList.find((dev) => dev.vendorId === 0x0955)?.deviceId);
  });

  return win;
};

const { tegraRcmSmash, payloadDirectory, prodKeysSearchPaths } = getResources(app.isPackaged);
let explorerController: ExplorerController;
let mainWindow: BrowserWindow;

const mainChannelImpl = {
  preloadBridge: async () => {
    const plat = platform();
    return {
      isWindows: plat === 'win32',
      isLinux: plat === 'linux',
      isOsx: plat === 'darwin',
    };
  },

  openLink: async (link: string) => shell.openExternal(link),
  openPath: async (p: string) => shell.openPath(p),
  pathDirname: async (p: string) => path.dirname(p),
  pathJoin: async (...parts: string[]) => path.join(...parts),

  tegraRcmSmash: async (payloadFilePath: string) => {
    interface SmashResult {
      success: boolean;
      stdout: string;
      stderr: string;
    }

    return new Promise<SmashResult>((resolve) => {
      // TODO: compile TegraRcmSmash ourselves
      cp.execFile(tegraRcmSmash, [payloadFilePath], { encoding: 'ucs-2' }, (err, stdout, stderr) => {
        resolve({
          success: !err,
          stdout: stdout.trim(),
          stderr: stderr.trim(),
        });
      });
    });
  },

  splitFile: async (filePath: string, asArchive: boolean, inPlace: boolean) => split(filePath, asArchive, inPlace),
  mergeFile: async (filePath: string, inPlace: boolean) => merge(filePath, inPlace),

  payloadsOpenDirectory: async () => shell.showItemInFolder(payloadDirectory),
  payloadsReadFile: (payloadPath: string) => payloads.readPayload(payloadPath),
  payloadsCopyIn: (filePaths: string[]) => payloads.copyInFiles(filePaths),
  payloadsFind: () => payloads.findPayloads(),

  prodKeysFind: () =>
    findProdKeys(app.isPackaged).then((keys) => keys && { location: keys.path, data: keys.toString() }),
  prodKeysSearchPaths: async () => prodKeysSearchPaths,

  nandOpenDisk: async (nandPath: string, keysFromUser?: ProdKeys) =>
    explorerController.open({ nandPath, keysFromUser, asSudo: true }),
  nandOpenDump: async (nandPath: string, keysFromUser?: ProdKeys) =>
    explorerController.open({ nandPath, keysFromUser, asSudo: false }),
  nandClose: async () => explorerController.close(),

  nandVerifyPartitionTable: async () => explorerController.call('verifyPartitionTable'),
  nandRepairBackupPartitionTable: async () => explorerController.call('repairBackupPartitionTable'),
  nandMountPartition: async (partName: string, readonly: boolean, keys?: ProdKeys) =>
    explorerController.call('mount', partName, readonly, keys),
  nandReaddir: async (path: string) => explorerController.call('readdir', path),
  nandCopyFileOut: async (pathInNand: string) => {
    const result = await dialog.showSaveDialog(mainWindow, { defaultPath: path.basename(pathInNand) });
    if (result.canceled) return;

    return explorerController.call('copyFileOut', pathInNand, result.filePath);
  },
  nandCopyFilesIn: async (dirPathInNand: string, filePaths: string[]) =>
    explorerController.call('copyFilesIn', dirPathInNand, filePaths),
  nandCheckExists: async (dirPathInNand: string, filePaths: string[]) =>
    explorerController.call('checkExists', dirPathInNand, filePaths),
  nandMoveEntry: async (oldPathInNand: string, newPathInNand: string) =>
    explorerController.call('move', oldPathInNand, newPathInNand),
  nandDeleteEntry: async (pathInNand: string) => explorerController.call('del', pathInNand),
  nandFormatPartition: async (partName: string, readonly: boolean, keys?: ProdKeys) =>
    explorerController.call('format', partName, readonly, keys),
} satisfies PromiseIpc;
export type MainIpcDefinition = typeof mainChannelImpl;

// This method will be called when Electron has finished
// initialization and is ready to create browser windows.
// Some APIs can only be used after this event occurs.
app.on('ready', () => {
  for (const [channel, impl] of Object.entries(mainChannelImpl)) {
    ipcMain.handle(channel, (_event, ...args) => (impl as PromiseIpcHandler)(...args));
  }

  mainWindow = createMainWindow();
  explorerController = new ExplorerController(mainWindow);
});

// Quit when all windows are closed, except on macOS. There, it's common
// for applications and their menu bar to stay active until the user quits
// explicitly with Cmd + Q.
app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('activate', () => {
  // On OS X it's common to re-create a window in the app when the
  // dock icon is clicked and there are no other windows open.
  if (BrowserWindow.getAllWindows().length === 0) {
    mainWindow = createMainWindow();
    explorerController.setMainWindow(mainWindow);
  }
});

// In this file you can include the rest of your app's specific main process
// code. You can also put them in separate files and import them here.
