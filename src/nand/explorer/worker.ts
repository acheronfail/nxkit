import { MessageEvent } from 'electron';
import { Message } from './messages';

function reply<In>(message: Message<In>) {
  process.parentPort.postMessage(message);
}

process.parentPort.on('message', (event: MessageEvent) => {
  // TODO: create types for main <-> explorer just like main <-> renderer
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const { id, data }: Message<any> = event.data;
  console.log('I received a message', data);
  reply({ id, data: 'foo' });
});
