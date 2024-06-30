import { app, BrowserWindow, ipcMain, shell } from 'electron';
import { platform } from 'node:os';
import cp from 'node:child_process';
import { createRequire } from 'node:module';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import { Channels, MainChannelImpl, NandError } from '../channels';
import { findProdKeys } from './keys';
import explorer from '../nand/explorer';
import * as payloads from './payloads';
import automaticContextMenus from 'electron-context-menu';
import { getResources } from '../resources';

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

// This method will be called when Electron has finished
// initialization and is ready to create browser windows.
// Some APIs can only be used after this event occurs.
app.on('ready', () => {
  const { tegraRcmSmash, payloadDirectory, prodKeysSearchPaths } = getResources(app.isPackaged);

  let mainWindow: BrowserWindow | undefined = undefined;
  const mainChannelImpl: MainChannelImpl = {
    [Channels.PreloadBridge]: async () => {
      const plat = platform();
      return {
        isWindows: plat === 'win32',
        isLinux: plat === 'linux',
        isOsx: plat === 'darwin',
      };
    },

    [Channels.OpenLink]: async (_event, link) => shell.openExternal(link),
    [Channels.PathDirname]: async (_event, p) => path.dirname(p),
    [Channels.PathJoin]: async (_event, ...parts) => path.join(...parts),

    [Channels.TegraRcmSmash]: async (_event, payloadFilePath) => {
      return new Promise((resolve) => {
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

    [Channels.PayloadsOpenDirectory]: async (_event) => shell.showItemInFolder(payloadDirectory),
    [Channels.PayloadsReadFile]: (_event, payloadPath) => payloads.readPayload(payloadPath),
    [Channels.PayloadsCopyIn]: (_event, filePaths) => payloads.copyInFiles(filePaths),
    [Channels.PayloadsFind]: (_event) => payloads.findPayloads(),

    [Channels.ProdKeysFind]: (_event) =>
      findProdKeys().then((keys) => keys && { location: keys.path, data: keys.toString() }),
    [Channels.ProdKeysSearchPaths]: async (_event) => prodKeysSearchPaths,

    [Channels.NandOpen]: async (_event, path) => explorer.open(path),
    [Channels.NandClose]: async (_event) => explorer.close(),
    [Channels.NandMountPartition]: async (_event, partName, readonly, keysFromUser) =>
      explorer.mount(partName, readonly, keysFromUser),
    [Channels.NandReaddir]: async (_event, path) => explorer.readdir(path),
    [Channels.NandCopyFileOut]: async (_event, pathInNand) => {
      if (!mainWindow) {
        return { error: NandError.Generic, description: 'Failed to find the main application window' };
      }

      return explorer.copyFileOut(pathInNand, mainWindow);
    },
    [Channels.NandCopyFilesIn]: async (_event, dirPathInNand, filePaths) =>
      explorer.copyFilesIn(dirPathInNand, filePaths),
    [Channels.NandCheckExists]: async (_event, dirPathInNand, filePaths) =>
      explorer.checkExists(dirPathInNand, filePaths),
    [Channels.NandMoveEntry]: async (_event, oldPathInNand, newPathInNand) =>
      explorer.move(oldPathInNand, newPathInNand),
    [Channels.NandDeleteEntry]: async (_event, pathInNand) => explorer.del(pathInNand),
    [Channels.NandFormatPartition]: async (_event, partName, readonly, keysFromUser) =>
      explorer.format(partName, readonly, keysFromUser),
  };

  for (const [channel, impl] of Object.entries(mainChannelImpl)) {
    ipcMain.handle(channel, impl);
  }

  mainWindow = createMainWindow();
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
    createMainWindow();
  }
});

// In this file you can include the rest of your app's specific main process
// code. You can also put them in separate files and import them here.
