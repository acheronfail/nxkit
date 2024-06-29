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

import crypto from 'node:crypto';

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
  private readonly tweakCipher: crypto.Cipher;
  private readonly cryptoCipher: crypto.Cipher;
  private readonly cryptoDecipher: crypto.Decipher;

  constructor(
    private readonly cryptoKey: Buffer,
    private readonly tweakKey: Buffer,
    public readonly sectorSize = 0x4000,
  ) {
    this.tweakCipher = crypto.createCipheriv('aes-128-ecb', tweakKey, null).setAutoPadding(false);
    this.cryptoCipher = crypto.createCipheriv('aes-128-ecb', cryptoKey, null).setAutoPadding(false);
    this.cryptoDecipher = crypto.createDecipheriv('aes-128-ecb', cryptoKey, null).setAutoPadding(false);
  }

  private createTweak(sectorOffset: number): Tweak {
    const tweak = Buffer.alloc(16, 0);
    tweak.writeBigInt64BE(BigInt(sectorOffset), 8);

    const tmp = this.tweakCipher.update(tweak);
    return new BigUint64Array(tmp.buffer, tmp.byteOffset, 2);
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

    let block8Offset = 0;
    let block64Offset = 0;
    const processBlock = (tweak: Tweak, runs: number) => {
      for (let i = 0; i < runs; i++) {
        if (block8Offset >= input.length) return;

        const block = new Uint8Array(input8.buffer, input8.byteOffset + block8Offset, 16);

        input64[block64Offset + 0] ^= tweak[0];
        input64[block64Offset + 1] ^= tweak[1];
        cipher.update(block).copy(block);
        input64[block64Offset + 0] ^= tweak[0];
        input64[block64Offset + 1] ^= tweak[1];

        this.updateTweak(tweak);
        input.set(block, block8Offset);

        block8Offset += 16;
        block64Offset += 2;
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

    while (block8Offset < input.byteLength) {
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
