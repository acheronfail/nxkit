import { app, BrowserWindow, ipcMain } from 'electron';
import { platform } from 'node:os';
import cp from 'node:child_process';
import { createRequire } from 'node:module';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
import { Channels, MainChannelImpl } from '../channels';
import { findProdKeys, Keys, PROD_KEYS_SEARCH_PATHS } from './keys';
import * as nand from '../nand/explorer';
import automaticContextMenus from 'electron-context-menu';
import { getResources } from '../resources';

automaticContextMenus({});

const require = createRequire(import.meta.url);
const __dirname = path.dirname(fileURLToPath(import.meta.url));

// Handle creating/removing shortcuts on Windows when installing/uninstalling.
if (require('electron-squirrel-startup')) {
  app.quit();
}

function loadWindow(window: BrowserWindow, name: string) {
  if (RENDERER_VITE_DEV_SERVER_URL) {
    window.loadURL(`${RENDERER_VITE_DEV_SERVER_URL}/src/${name}/index.html`);
  } else {
    window.loadFile(path.join(__dirname, `../${RENDERER_VITE_NAME}/src/${name}/index.html`));
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

  loadWindow(win, 'window_main');

  if (!app.isPackaged) {
    win.webContents.openDevTools();
  }

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
  let mainWindow: BrowserWindow | undefined = undefined;
  const mainChannelImpl: MainChannelImpl = {
    [Channels.PreloadBrige]: async () => ({
      isWindows: platform() === 'win32',
    }),
    [Channels.TegraRcmSmash]: async (_event, payloadFilePath) => {
      return new Promise((resolve) => {
        // TODO: compile TegraRcmSmash ourselves
        const { tegraRcmSmash } = getResources(app.isPackaged);
        cp.execFile(tegraRcmSmash, [payloadFilePath], { encoding: 'ucs-2' }, (err, stdout, stderr) => {
          resolve({
            success: !err,
            stdout: stdout.trim(),
            stderr: stderr.trim(),
          });
        });
      });
    },
    [Channels.ProdKeysFind]: (_event) =>
      findProdKeys().then((keys) => keys && { location: keys.path, data: keys.toString() }),
    [Channels.ProdKeysSearchPaths]: async (_event) => PROD_KEYS_SEARCH_PATHS,

    [Channels.NandOpen]: async (_event, path) => nand.open(path),
    [Channels.NandClose]: async (_event) => nand.close(),
    [Channels.NandMountPartition]: async (_event, paritionName, keysFromUser) =>
      nand.mount(
        paritionName,
        keysFromUser ? Keys.parseKeys(keysFromUser.location, keysFromUser.data) : await findProdKeys(),
      ),
    [Channels.NandReaddir]: async (_event, path) => nand.readdir(path),
    [Channels.NandCopyFile]: async (_event, pathInNand) => mainWindow && nand.copyFile(pathInNand, mainWindow),
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