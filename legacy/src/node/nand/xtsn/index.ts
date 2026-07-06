/**
 * Switch NAND crypto routines for NodeJS.
 *
 * Many thanks to all the open source implementations around the place:
 *  - https://github.com/DacoTaco/YASDU/blob/master/R.N.D/xts_crypto.cpp
 *  - https://github.com/luigoalma/haccrypto/blob/master/haccrypto/_crypto.cpp
 *  - https://github.com/eliboa/NxNandManager/blob/master/NxNandManager/NxCrypto.cpp
 *  - https://github.com/ihaveamac/ninfs/blob/main/ninfs/mount/nandhac.py
 *  - https://gitlab.com/roothorick/busehac
 *  - https://github.com/ihaveamac/switchfs
 */

import { createRequire } from 'node:module';

const require = globalThis.require ?? createRequire(import.meta.url);
const XtsnCipher: NativeCipherConstructor = require('./build/Release/xtsn.node');

interface NativeCipherConstructor {
  /**
   * Create a new instance of the cipher class.
   * @param cryptoKey the 16 byte Crypto Bis Key
   * @param tweakKey the 16 byte Tweak Bis Key
   * @param sectorSize sector size, defaults to 0x4000
   */
  new (cryptoKey: Buffer, tweakKey: Buffer, sectorSize?: number): NativeCipher;
}

interface NativeCipher {
  /**
   * Perform the cipher operation and encrypt or decrypt the provided data.
   * The operation runs in-place; that is, it mutates the input buffer.
   * @param input data to run the cipher on
   * @param sectorOffset starting sector offset
   * @param skippedBytes number of bytes skipped in the current sector offset
   * @param encrypt whether to encrypt or decrypt
   * @returns the input buffer
   */
  run(input: Buffer, sectorOffset: number, skippedBytes: number, encrypt: boolean): void;
}

/**
 * Some notes:
 *  It's AES-XTS encryption, and AES works in 128 bit (16 byte) blocks
 *  Sector size is 0x4000
 *  XTS tweaks are updated (consecutively) once per sector
 *  When encrypting/decrypting, the input buffer must be 16-byte aligned
 *
 * See https://switchbrew.org/wiki/Flash_Filesystem for more information.
 */
export class Xtsn {
  private readonly cipher: NativeCipher;

  constructor(
    private readonly cryptoKey: Buffer,
    private readonly tweakKey: Buffer,
    public readonly sectorSize = 0x4000,
  ) {
    this.cipher = new XtsnCipher(cryptoKey, tweakKey, sectorSize);
  }

  public encrypt(input: Buffer, byteOffset = 0): Buffer {
    const sectorOffset = Math.floor(byteOffset / this.sectorSize);
    this.cipher.run(input, sectorOffset, byteOffset % this.sectorSize, true);
    return input;
  }

  public decrypt(input: Buffer, byteOffset = 0): Buffer {
    const sectorOffset = Math.floor(byteOffset / this.sectorSize);
    this.cipher.run(input, sectorOffset, byteOffset % this.sectorSize, false);
    return input;
  }

  // Expose APIs that are compatible with the python haccrypto lib

  public encryptHC(input: Buffer, sectorOffset: number, sectorSize = 0x200, skippedBytes = 0): Buffer {
    const xtsn = new Xtsn(this.cryptoKey, this.tweakKey, sectorSize);
    xtsn.cipher.run(input, sectorOffset, skippedBytes, true);
    return input;
  }

  public decryptHC(input: Buffer, sectorOffset: number, sectorSize = 0x200, skippedBytes = 0): Buffer {
    const xtsn = new Xtsn(this.cryptoKey, this.tweakKey, sectorSize);
    xtsn.cipher.run(input, sectorOffset, skippedBytes, false);
    return input;
  }
}
