import { Xtsn } from '../xtsn';
import { Io } from './io';

/**
 * This is a class which allows reading partitions from Switch NAND dumps.
 * It handles split dumps as well as combined dumps, and also when configured
 * also handles decryption/encryption of reads/writes.
 */
export class NandIo {
  private readonly sectorSize = 0x4000;

  constructor(
    private readonly io: Io,
    private readonly partitionStartOffset: number,
    private readonly partitionEndOffset: number,
    private readonly xtsn?: Xtsn,
  ) {}

  public size(): number {
    return this.io.size();
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

    if (!this.xtsn) {
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

    const before = offset % 16;
    let after = (offset + size) % 16;
    if (after) after = 16 - after;

    const alignedDiskOffset = diskOffset - before;
    const readSize = before + size + after;

    const buf = this.io.read(alignedDiskOffset, readSize);
    return this.xtsn.decrypt(buf, offset - before, this.sectorSize).subarray(before, before + size);
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

    if (!this.xtsn) {
      return this.io.write(diskOffset, Buffer.from(data));
    }

    // In order to write to an arbitrary location, we need to know which 16 byte
    // chunk(s) it's in, and we need to know their contents since the smallest
    // increment we can encrypt is a 16 byte chunk. If we wanted to replace bytes
    // `4455` with `xxxx`, it looks something like this:
    //  1111222233334444 5555666677778888
    //  BBBBBBBBBBBBBB^    AAAAAAAAAAAAAA   B = before, A = after, ^ = offset
    //  xxxxxxxxxxxxxxWW WWyyyyyyyyyyyyyy   x = before chunk, y = after chunk, W = data to write

    const chunks: Uint8Array[] = [];

    const before = offset % this.sectorSize;
    const alignedPartOffset = offset - before;
    if (before) {
      chunks.push(this.read(alignedPartOffset, before));
    }

    chunks.push(data);

    let after = (offset + data.byteLength) % this.sectorSize;
    if (after) {
      after = this.sectorSize - after;
      chunks.push(this.read(offset + data.byteLength, after));
    }

    const enc = this.xtsn.encrypt(Buffer.concat(chunks), diskOffset - before, this.sectorSize);
    return this.io.write(diskOffset + alignedPartOffset, enc);
  }
}
