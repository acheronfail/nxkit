import { fileURLToPath } from 'node:url';
import path from 'node:path';
import { BrowserWindow, UtilityProcess, app, utilityProcess, dialog } from 'electron';
import { ExplorerIpcDefinition, ExplorerIpcKey, IncomingMessage, OutgoingMessage } from '../node/nand/explorer.worker';

export class ExplorerWorker {
  private messageId = 0;
  private exited = false;
  private readonly proc: UtilityProcess;
  private readonly mainWindow: BrowserWindow;

  constructor(mainWindow: BrowserWindow) {
    const __dirname = path.dirname(fileURLToPath(import.meta.url));
    const workerPath = path.join(__dirname, 'src_node_nand_explorer.worker.js');
    this.proc = utilityProcess.fork(workerPath, [], { stdio: 'inherit' });
    this.proc.on('exit', (code) => {
      console.log(`Worker process exited with code: ${code}`);
      this.exited = true;
    });
    this.proc.postMessage({ id: 'bootstrap', isPackaged: app.isPackaged } satisfies IncomingMessage<ExplorerIpcKey>);
    this.mainWindow = mainWindow;
  }

  public call<C extends keyof ExplorerIpcDefinition>(
    channel: C,
    ...args: Parameters<ExplorerIpcDefinition[C]>
  ): ReturnType<ExplorerIpcDefinition[C]> {
    if (this.exited) {
      const error = new Error('The NAND worker process has unexpectedly exited');
      dialog.showMessageBox(this.mainWindow, {
        type: 'error',
        message: error.message,
      });
      throw error;
    }

    const id = this.messageId++;
    return new Promise((resolve) => {
      const handler = (msg: OutgoingMessage<C>) => {
        if (msg.id === id) {
          resolve(msg.value);
          this.proc.removeListener('message', handler);
        } else if (msg.id === 'progress') {
          // TODO: types around this api
          this.mainWindow.webContents.postMessage('progress', msg.progress);
        }
      };

      this.proc.postMessage({ id, channel, args } satisfies IncomingMessage<C>);
      this.proc.on('message', handler);
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
    }) as any;
  }
}
