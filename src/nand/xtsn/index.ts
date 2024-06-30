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

const require = createRequire(import.meta.url);
const XtsnCipher: NativeCipherConstructor = require('./build/Release/xtsn.node');

interface NativeCipher {
  /**
   * Run the cipher on `data` at `offset`. Only does 16 bytes at a time.
   * @param data data to run the cipher on, only 16 bytes are done at a time.
   * @param offset optional offset to start when running the cipher on data.
   * @returns the number of bytes processed (should always be 16 unless an error occurred)
   */
  update(data: NodeJS.ArrayBufferView, offset?: number): number;
}

interface NativeCipherConstructor {
  /**
   * Create a new instance of the cipher class.
   * @param key the 16 byte Bis Key (no shorter, no longer)
   * @param encrypt whether this cipher is used for encrypting or decrypting
   */
  new (key: Buffer, encrypt: boolean): NativeCipher;
}

/**
 * We use the largest view possible when representing the tweak so when we need
 * to apply it to the data we can do it in as few cycles as possible. Doing it
 * this way helps node to do the XOR operation in larger chunks.
 */
type Tweak = BigUint64Array;

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
  private readonly tweakCipher: NativeCipher;
  private readonly cryptoCipher: NativeCipher;
  private readonly cryptoDecipher: NativeCipher;

  constructor(
    private readonly cryptoKey: Buffer,
    private readonly tweakKey: Buffer,
    public readonly sectorSize = 0x4000,
  ) {
    this.tweakCipher = new XtsnCipher(tweakKey, true);
    this.cryptoCipher = new XtsnCipher(cryptoKey, true);
    this.cryptoDecipher = new XtsnCipher(cryptoKey, false);
  }

  private createTweak(sectorOffset: number): Tweak {
    const tweak = Buffer.alloc(16, 0);
    tweak.writeBigInt64BE(BigInt(sectorOffset), 8);

    this.tweakCipher.update(tweak);
    return new BigUint64Array(tweak.buffer, tweak.byteOffset, 2);
  }

  private updateTweak(tweak: Tweak) {
    const tweakView = new Uint8Array(tweak.buffer, tweak.byteOffset, 16);

    const lastHigh = tweakView[15] & 0x80;
    for (let j = 15; j > 0; j--) {
      tweakView[j] = ((tweakView[j] << 1) & ~1) | (tweakView[j - 1] & 0x80 ? 1 : 0);
    }

    tweakView[0] = ((tweakView[0] << 1) & ~1) ^ (lastHigh ? 0x87 : 0);
  }

  private doCrypt(input: Buffer, sectorOffset: number, skippedBytes: number, encrypt: boolean) {
    const cipher = encrypt ? this.cryptoCipher : this.cryptoDecipher;
    const input8 = new Uint8Array(input.buffer, input.byteOffset, input.byteLength);
    const input64 = new BigInt64Array(input8.buffer, input.byteOffset, input.byteLength / 8);

    let offset8 = 0;
    let offset64 = 0;
    const processBlock = (tweak: Tweak, runs: number) => {
      for (let i = 0; i < runs; i++) {
        if (offset8 >= input.length) return;

        input64[offset64 + 0] ^= tweak[0];
        input64[offset64 + 1] ^= tweak[1];
        cipher.update(input8, offset8);
        input64[offset64 + 0] ^= tweak[0];
        input64[offset64 + 1] ^= tweak[1];

        this.updateTweak(tweak);

        offset8 += 16;
        offset64 += 2;
      }
    };

    if (skippedBytes > 0) {
      const fullSectorsToSkip = Math.floor(skippedBytes / this.sectorSize);
      sectorOffset += fullSectorsToSkip;
      skippedBytes %= this.sectorSize;
    }

    if (skippedBytes > 0) {
      const tweak = this.createTweak(sectorOffset);
      for (let i = 0; i < Math.floor(skippedBytes / 16); i++) {
        this.updateTweak(tweak);
      }

      processBlock(tweak, Math.floor((this.sectorSize - skippedBytes) / 16));
      sectorOffset++;
    }

    while (offset8 < input.byteLength) {
      const tweak = this.createTweak(sectorOffset);
      processBlock(tweak, Math.floor(this.sectorSize / 16));
      sectorOffset++;
    }

    return input;
  }

  public encrypt(input: Buffer, byteOffset = 0): Buffer {
    const sectorOffset = Math.floor(byteOffset / this.sectorSize);
    return this.doCrypt(input, sectorOffset, byteOffset % this.sectorSize, true);
  }

  public decrypt(input: Buffer, byteOffset = 0): Buffer {
    const sectorOffset = Math.floor(byteOffset / this.sectorSize);
    return this.doCrypt(input, sectorOffset, byteOffset % this.sectorSize, false);
  }

  // Expose APIs that are compatible with the python haccrypto lib

  public encryptHC(input: Buffer, sectorOffset: number, sectorSize = 0x200, skippedBytes = 0): Buffer {
    const xtsn = new Xtsn(this.cryptoKey, this.tweakKey, sectorSize);
    return xtsn.doCrypt(input, sectorOffset, skippedBytes, true);
  }

  public decryptHC(input: Buffer, sectorOffset: number, sectorSize = 0x200, skippedBytes = 0): Buffer {
    const xtsn = new Xtsn(this.cryptoKey, this.tweakKey, sectorSize);
    return xtsn.doCrypt(input, sectorOffset, skippedBytes, false);
  }
}
