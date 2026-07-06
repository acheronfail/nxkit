import { NandIoLayer } from './layer';
import { Io } from './io';
import { Crypto } from './crypto';

export class MockIo implements Io {
  constructor(private readonly buf: Buffer) {}

  close() {
    return;
  }
  size() {
    return this.buf.byteLength;
  }
  read(offset: number, length: number): Buffer {
    return this.buf.subarray(offset, offset + length);
  }
  write(offset: number, data: Buffer): number {
    this.buf.set(data, offset);
    return data.byteLength;
  }
}

export const BLOCK_SIZE = 16;
export const CLUSTER_SIZE = 16;
export const DISK_SIZE = CLUSTER_SIZE * 8;

export class XorOffsetCrypto implements Crypto {
  public blockSize(): number {
    return BLOCK_SIZE;
  }

  public encrypt(input: Buffer, byteOffset = 0): Buffer {
    return this.xorWithClusterOffset(input, byteOffset);
  }

  public decrypt(input: Buffer, byteOffset = 0): Buffer {
    return this.xorWithClusterOffset(input, byteOffset);
  }

  private xorWithClusterOffset(input: Buffer, byteOffset: number) {
    const data = Buffer.from(input);
    let clusterOffset = Math.floor(byteOffset / CLUSTER_SIZE);
    for (let i = 0; i < data.byteLength; i += CLUSTER_SIZE) {
      for (let j = 0; j < CLUSTER_SIZE; j++) data[i + j] ^= clusterOffset;
      clusterOffset++;
    }

    return data;
  }
}

export const SECTOR_SIZE = 0x40;
export const getLayer = (disk: Buffer, crypto?: Crypto) =>
  new NandIoLayer({
    io: new MockIo(disk),
    partitionStartOffset: 0,
    partitionEndOffset: disk.byteLength,
    sectorSize: SECTOR_SIZE,
    crypto,
  });
