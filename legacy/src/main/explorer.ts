import { fileURLToPath } from 'node:url';
import path from 'node:path';
import { EventEmitter } from 'node:events';
import net, { Server, Socket } from 'node:net';
import { app, BrowserWindow, UtilityProcess, utilityProcess, dialog } from 'electron';
import { exec } from '@vscode/sudo-prompt';
import chalk from 'chalk';
import { ExplorerIpcDefinition, ExplorerIpcKey, IncomingMessage, OutgoingMessage } from '../node/nand/explorer/worker';
import { ProdKeys } from '../channels';
import { sendSocketMessage, setupSocket } from '../node/nand/explorer/socket';
import { getUniquePath } from '../node/paths';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const WORKER_PATH = path.join(__dirname, 'src_node_nand_explorer_worker.js');

interface WorkerProcess {
  close(): void;

  addExitListener(fn: () => void): void;

  addMessageListener(fn: (msg: OutgoingMessage<ExplorerIpcKey>) => void): void;
  removeMessageListener(fn: (msg: OutgoingMessage<ExplorerIpcKey>) => void): void;

  sendMessage(message: IncomingMessage<ExplorerIpcKey>): void;
}

class UtilityProcessWorker implements WorkerProcess {
  private constructor(private readonly proc: UtilityProcess) {}

  static create(): Promise<UtilityProcessWorker> {
    return new Promise((resolve) => {
      const proc = utilityProcess.fork(WORKER_PATH, [], { stdio: 'pipe' });
      proc.stdout?.on('data', (chunk) => process.stdout.write(chalk.yellow(chunk.toString())));
      proc.stderr?.on('data', (chunk) => process.stderr.write(chalk.red(chunk.toString())));
      proc.once('spawn', () => {
        resolve(new UtilityProcessWorker(proc));
      });
    });
  }

  close() {
    this.proc.kill();
  }

  addExitListener(fn: () => void): void {
    this.proc.on('exit', fn);
  }

  addMessageListener(fn: (msg: OutgoingMessage<ExplorerIpcKey>) => void): void {
    this.proc.addListener('message', fn);
  }

  removeMessageListener(fn: (msg: OutgoingMessage<ExplorerIpcKey>) => void): void {
    this.proc.removeListener('message', fn);
  }

  sendMessage(message: IncomingMessage<ExplorerIpcKey>): void {
    this.proc.postMessage(message);
  }
}
class SudoProcessWorker implements WorkerProcess {
  private readonly emitter = new EventEmitter();
  private constructor(
    private readonly server: Server,
    private readonly client: Socket,
  ) {
    setupSocket(client, (msg) => this.emitter.emit('message', msg));
  }

  static create(): Promise<SudoProcessWorker> {
    return getUniquePath(`${process.platform === 'win32' ? '//./pipe' : ''}/tmp/nxkit.explorer.worker`).then(
      (pipePath) =>
        new Promise((resolve, reject) => {
          const server = net.createServer();
          const exePath = app.getPath('exe');

          const cmd: string[] = [`"${exePath}"`, '--no-sandbox', `"${WORKER_PATH}"`];

          // NOTE: `sudo-prompt` doesn't pass `env` correctly on linux
          const env: Record<string, string> = { NXKIT_DISK_PIPE: pipePath };
          if (process.platform === 'linux') {
            const add = (key: string) => {
              const value = process.env[key];
              if (value) env[key] = value;
            };

            add('DISPLAY');
            add('XAUTHORITY');

            for (const [key, value] of Object.entries(env)) {
              cmd.unshift(`${key}="${value.replace(/"/g, '\\"')}"`);
            }

            // NOTE: `sudo-prompt` also doesn't maintain the CWD properly, either
            cmd.unshift(`cd "${process.cwd()}";`);
          }

          server.listen(pipePath, () => {
            server.once('connection', (client) => resolve(new SudoProcessWorker(server, client)));
            exec(cmd.join(' '), { name: app.getName(), env }, (err, stdout, stderr) => {
              if (err) {
                console.error('sudo-prompt:err', err);
                reject(err);
              }

              if (stdout?.length) console.log('sudo-prompt:stdout:', stdout);
              if (stderr?.length) console.log('sudo-prompt:stderr:', stderr);
            });
          });
        }),
    );
  }

  close(): void {
    // sending an empty message tells the client to exit
    this.client.write(Buffer.alloc(4, 0));
    this.server.close();
  }

  addExitListener(fn: () => void): void {
    this.client.on('close', fn);
  }

  addMessageListener(fn: (msg: OutgoingMessage<ExplorerIpcKey>) => void): void {
    this.emitter.addListener('message', fn);
  }

  removeMessageListener(fn: (msg: OutgoingMessage<ExplorerIpcKey>) => void): void {
    this.emitter.removeListener('message', fn);
  }

  sendMessage(message: IncomingMessage<ExplorerIpcKey>): void {
    sendSocketMessage(this.client, JSON.stringify(message));
  }
}

export class ExplorerController {
  private messageId = 0;
  private proc: WorkerProcess | null = null;
  private mainWindow: BrowserWindow;

  constructor(mainWindow: BrowserWindow) {
    this.mainWindow = mainWindow;
  }

  public setMainWindow(mainWindow: BrowserWindow) {
    this.mainWindow = mainWindow;
  }

  public async open(options: { asSudo: boolean; nandPath: string; keysFromUser?: ProdKeys }) {
    if (this.proc) {
      await this.close();
    }

    this.proc = options.asSudo ? await SudoProcessWorker.create() : await UtilityProcessWorker.create();

    this.proc.sendMessage({ id: 'bootstrap', isPackaged: app.isPackaged } satisfies IncomingMessage<ExplorerIpcKey>);
    this.proc.addMessageListener((msg) => {
      if (msg.id === 'progress') {
        // TODO: types around this api
        this.mainWindow.webContents.postMessage('progress', msg.progress);
      }
    });

    return this.call('open', options.nandPath, options.keysFromUser);
  }

  public async close() {
    const proc = this.proc;
    if (!proc) {
      return;
    }

    // tell filesystem to close
    const ret = await this.call('close');

    // tell process to close
    proc.close();

    // wait for process to have closed
    await new Promise<void>((resolve) => {
      proc.addExitListener(() => {
        this.proc = null;
        resolve();
      });
    });

    // return now we're sure it's all closed
    return ret;
  }

  public call<C extends ExplorerIpcKey>(
    channel: C,
    ...args: Parameters<ExplorerIpcDefinition[C]>
  ): ReturnType<ExplorerIpcDefinition[C]> {
    return new Promise((resolve) => {
      if (!this.proc) {
        const error = new Error('Failed to communicate with the NAND worker process');
        dialog.showMessageBox(this.mainWindow, { type: 'error', message: error.message });
        throw error;
      }

      const id = this.messageId++;
      const handler = (msg: OutgoingMessage<ExplorerIpcKey>) => {
        if (msg.id === id) {
          this.proc?.removeMessageListener(handler);
          resolve(msg.value);
        }
      };

      this.proc.addMessageListener(handler);
      this.proc.sendMessage({ id, channel, args } satisfies IncomingMessage<C>);
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
    }) as any;
  }
}
