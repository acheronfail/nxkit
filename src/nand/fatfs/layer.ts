import fs from 'fs';
import { Xtsn } from '../xtsn';

export class XtsnLayer {
  private readonly sectorSize = 0x4000;

  constructor(
    private readonly fd: number,
    private readonly partitionStartOffset: number,
    private readonly partitionEndOffset: number,
    private readonly xtsn?: Xtsn
  ) {}

  public size(): number {
    return fs.fstatSync(this.fd).size;
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
      const buf = Buffer.alloc(size, 0);
      fs.readSync(this.fd, buf, 0, size, diskOffset);
      return buf;
    }

    const before = offset % 16;
    let after = (offset + size) % 16;
    if (after) after = 16 - after;

    const alignedDiskOffset = diskOffset - before;
    const readSize = before + size + after;

    const buf = Buffer.alloc(readSize, 0);
    fs.readSync(this.fd, buf, 0, readSize, alignedDiskOffset);
    return this.xtsn.decrypt(buf, offset - before, this.sectorSize).subarray(before, before + size);
  }

  // TODO: test writes
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
      return fs.writeSync(this.fd, data, 0, data.byteLength, diskOffset);
    }

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
    return fs.writeSync(this.fd, enc, 0, enc.byteLength, diskOffset + alignedPartOffset);
  }
}
