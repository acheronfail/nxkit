import { type Socket } from 'node:net';

export function setupSocket<T>(socket: Socket, messageHandler: (t: T) => void, closeHandler?: () => void) {
  const headerSize = 4;
  let msgData = Buffer.alloc(0);
  let msgLength: number | null = null;

  socket.on('data', (chunk) => {
    msgData = Buffer.concat([msgData, chunk]);

    while (
      (msgLength === null && msgData.length >= headerSize) ||
      (msgLength !== null && msgData.length >= headerSize + msgLength)
    ) {
      if (msgLength === null) {
        msgLength = msgData.readUInt32BE(0);
      }

      // this indicates we must exit
      if (msgLength === 0) {
        closeHandler?.();
        msgData = msgData.subarray(headerSize + msgLength);
        msgLength = null;
        continue;
      }

      // read the message if we have it all
      if (msgData.length >= headerSize + msgLength) {
        const msgBytes = msgData.subarray(headerSize, headerSize + msgLength);
        let msg: T;
        try {
          msg = JSON.parse(msgBytes.toString('utf-8'));
        } catch (error) {
          console.error({
            error,
            msgLength,
            msgData: msgData.toString('utf-8'),
            messageBytes: msgBytes.toString('utf-8'),
          });
          closeHandler?.();
          break;
        }

        messageHandler(msg);

        // remove processed message, and reset length
        msgData = msgData.subarray(headerSize + msgLength);
        msgLength = null;
      }
    }
  });
}

export function sendSocketMessage(socket: Socket, data: string) {
  const msgBuf = Buffer.from(data, 'utf-8');
  const msgHeader = Buffer.alloc(4);
  msgHeader.writeUint32BE(msgBuf.length, 0);
  socket.write(Buffer.concat([msgHeader, msgBuf]));
}
