import { app, BrowserWindow, ipcMain } from 'electron';
import path from 'path';
import { platform } from 'os';
import cp from 'child_process';

// Handle creating/removing shortcuts on Windows when installing/uninstalling.
if (require('electron-squirrel-startup')) {
  app.quit();
}

const createWindow = () => {
  // Create the browser window.
  const mainWindow = new BrowserWindow({
    width: 800,
    height: 600,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
    },
  });

  // and load the index.html of the app.
  if (MAIN_WINDOW_VITE_DEV_SERVER_URL) {
    mainWindow.loadURL(MAIN_WINDOW_VITE_DEV_SERVER_URL);
  } else {
    mainWindow.loadFile(path.join(__dirname, `../renderer/${MAIN_WINDOW_VITE_NAME}/index.html`));
  }

  // Open the DevTools.
  if (!app.isPackaged) {
    mainWindow.webContents.openDevTools();
  }

  // Automatically select a pre-mariko Switch in RCM
  mainWindow.webContents.session.on('select-usb-device', (_event, details, callback) => {
    callback(details.deviceList.find((dev) => dev.vendorId === 0x0955)?.deviceId);
  });
};

// This method will be called when Electron has finished
// initialization and is ready to create browser windows.
// Some APIs can only be used after this event occurs.
app.on('ready', () => {
  ipcMain.handle('nxkit:bridge', () => {
    const bridgeApi: NXKitBridge = {
      isWindows: platform() === 'win32',
    };
    return bridgeApi;
  });

  ipcMain.handle('nxkit:tegra_rcm_smash', async (event, payloadFilePath: string) => {
    return new Promise<NXKitTegraRcmSmashResult>((resolve) => {
      // TODO: compile TegraRcmSmash ourselves
      const exePath = app.isPackaged
        ? path.join(process.resourcesPath, 'TegraRcmSmash.exe')
        : path.join('vendor', 'TegraRcmSmash', 'TegraRcmSmash.exe');

      cp.execFile(exePath, [payloadFilePath], { encoding: 'ucs-2' }, (err, stdout, stderr) => {
        resolve({ success: !err, stdout: stdout.trim(), stderr: stderr.trim() });
      });
    });
  });

  createWindow();
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
    createWindow();
  }
});

// In this file you can include the rest of your app's specific main process
// code. You can also put them in separate files and import them here.
