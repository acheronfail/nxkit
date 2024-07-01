import { Crypto } from './crypto';
import { Io } from './io';

export interface NandIoOptions {
  io: Io;
  partitionStartOffset: number;
  partitionEndOffset: number;
  sectorSize: number;
  crypto?: Crypto;
}

/**
 * This is a class which allows reading partitions from Switch NAND dumps.
 * It handles split dumps as well as combined dumps, and also when configured
 * also handles decryption/encryption of reads/writes.
 */
export class NandIoLayer {
  private readonly io: Io;
  private readonly partitionStartOffset: number;
  private readonly partitionEndOffset: number;
  private readonly crypto?: Crypto;

  public readonly sectorSize: number;
  public readonly sectorCount: number;

  constructor(options: NandIoOptions) {
    this.io = options.io;
    this.partitionStartOffset = options.partitionStartOffset;
    this.partitionEndOffset = options.partitionEndOffset;
    this.crypto = options.crypto;

    this.sectorSize = options.sectorSize;
    this.sectorCount = Math.floor((options.partitionEndOffset - options.partitionStartOffset) / options.sectorSize);
  }

  public read(offset: number, size: number): Buffer {
    const diskOffset = this.partitionStartOffset + offset;

    // can't read past the end
    if (this.partitionStartOffset + offset > this.partitionEndOffset) {
      return Buffer.from([]);
    }

    // adjust size to make sure we don't read past the end
    if (offset + size > this.partitionEndOffset) {
      size = this.partitionEndOffset - offset;
    }

    if (!this.crypto) {
      return this.io.read(diskOffset, size);
    }

    // In order to decrypt, we need to start at a 16 byte offset (it uses aes-based
    // encryption which works in 128 bit chunks), so if we wanted to read `xxxx` in:
    //  11112222333344xx xx55666677778888
    //  BBBBBBBBBBBBBB     AAAAAAAAAAAAAA   B = before, A = after, ^ = offset
    //                ^
    // we need to get the offset of the start of the first 16 byte chunk, and the end of
    // the second chunk. We then read both chunks and decrypt them, and return the
    // desired bytes (discarding `before` and `after`)

    const blockSize = this.crypto.blockSize();

    const before = offset % blockSize;
    let after = (offset + size) % blockSize;
    if (after) after = blockSize - after;

    const readSize = before + size + after;
    const alignedDiskOffset = diskOffset - before;
    const partByteOffset = offset - before;

    const buf = this.io.read(alignedDiskOffset, readSize);
    return this.crypto.decrypt(buf, partByteOffset).subarray(before, before + size);
  }

  public write(offset: number, data: Uint8Array): number {
    const diskOffset = this.partitionStartOffset + offset;

    // don't write past the end
    if (this.partitionStartOffset + offset > this.partitionEndOffset) {
      return 0;
    }

    // trim the size of the data to write to make sure we don't write past the end
    const diskEndOffset = diskOffset + data.byteLength;
    const excess = diskEndOffset - this.partitionEndOffset;
    if (excess > 0) {
      data = Buffer.from(data.buffer.slice(0, -excess));
    }

    if (!this.crypto) {
      return this.io.write(diskOffset, Buffer.from(data));
    }

    // In order to write to an arbitrary location, we need to know which 16 byte
    // chunk(s) it's in, and we need to know their contents since the smallest
    // increment we can encrypt is a 16 byte chunk. If we wanted to replace bytes
    // `4455` with `xxxx`, it looks something like this:
    //  1111222233334444 5555666677778888
    //  BBBBBBBBBBBBBB^    AAAAAAAAAAAAAA   B = before, A = after, ^ = offset
    //  xxxxxxxxxxxxxxWW WWyyyyyyyyyyyyyy   x = before chunk, y = after chunk, W = data to write

    const blockSize = this.crypto.blockSize();
    const chunks: Uint8Array[] = [];

    const before = offset % blockSize;
    const partByteOffset = offset - before;
    if (before) {
      chunks.push(this.read(partByteOffset, before));
    }

    chunks.push(data);

    let after = (offset + data.byteLength) % blockSize;
    if (after) {
      after = blockSize - after;
      chunks.push(this.read(offset + data.byteLength, after));
    }

    const toWrite = Buffer.concat(chunks);
    const enc = this.crypto.encrypt(toWrite, partByteOffset);

    const alignedDiskOffset = diskOffset - before;
    return this.io.write(alignedDiskOffset, enc);
  }
}
