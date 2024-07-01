import { fileURLToPath } from 'node:url';
import path from 'node:path';
import { UtilityProcess, utilityProcess } from 'electron';
import { NandResult, Partition, ProdKeys } from '../../channels';
import { Message } from './messages';
const __dirname = path.dirname(fileURLToPath(import.meta.url));

export class ExplorerWorker {
  private messageId = 0;
  private readonly proc: UtilityProcess;

  constructor() {
    this.proc = utilityProcess.fork(path.join(__dirname, 'src_nand_explorer_worker.js'));
  }

  private async postMessage<In, Out>(data: In): Promise<Out> {
    const id = this.messageId++;
    return new Promise((resolve) => {
      const handler = (message: Message<Out>) => {
        if (message.id === id) {
          this.proc.removeListener('message', handler);
          resolve(message.data);
        }
      };

      this.proc.postMessage({ id, data } satisfies Message<In>);
      this.proc.on('message', handler);
    });
  }

  public async open(nandPath: string, keysFromUser?: ProdKeys): Promise<NandResult<Partition[]>> {
    return this.postMessage({
      action: 'open',
      args: [nandPath, keysFromUser],
    });
  }
}
