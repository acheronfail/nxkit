import { Xtsn } from '../xtsn';

export interface Crypto {
  /** defines boundaries (in bytes) where reads and writes can start from */
  blockSize(): number;
  encrypt(input: Buffer, byteOffset?: number): Buffer;
  decrypt(input: Buffer, byteOffset?: number): Buffer;
}

export class NxCrypto implements Crypto {
  constructor(private readonly xtsn: Xtsn) {}

  blockSize(): number {
    // algorithm is aes-128-ecb (the '128' is 128 bits, thus 16 bytes)
    return 16;
  }

  encrypt(input: Buffer, byteOffset?: number): Buffer {
    return this.xtsn.encrypt(input, byteOffset);
  }

  decrypt(input: Buffer, byteOffset?: number): Buffer {
    return this.xtsn.decrypt(input, byteOffset);
  }
}
