import { fileURLToPath } from 'node:url';
import path from 'node:path';
import { UtilityProcess, app, utilityProcess } from 'electron';
import type { IncomingMessage, ExplorerIpcDefinition, OutgoingMessage } from './explorer.worker';

export class ExplorerWorker {
  private messageId = 0;
  private readonly proc: UtilityProcess;

  constructor() {
    const __dirname = path.dirname(fileURLToPath(import.meta.url));
    this.proc = utilityProcess.fork(path.join(__dirname, 'src_node_nand_explorer.worker.js'));
  }

  public call<C extends keyof ExplorerIpcDefinition>(
    channel: C,
    ...args: Parameters<ExplorerIpcDefinition[C]>
  ): ReturnType<ExplorerIpcDefinition[C]> {
    const id = this.messageId++;
    return new Promise((resolve) => {
      const handler = (message: OutgoingMessage<C>) => {
        if (message.id === id) {
          this.proc.removeListener('message', handler);
          resolve(message.value);
        }
      };

      this.proc.postMessage({ id, channel, args, isPackaged: app.isPackaged } satisfies IncomingMessage<C>);
      this.proc.on('message', handler);
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
    }) as any;
  }
}
