/**
 * Switch NAND crypto routines for NodeJS.
 *
 * See:
 *  - https://github.com/DacoTaco/YASDU/blob/master/R.N.D/xts_crypto.cpp
 *  - https://github.com/luigoalma/haccrypto/blob/master/haccrypto/_crypto.cpp
 *  - https://github.com/eliboa/NxNandManager/blob/master/NxNandManager/NxCrypto.cpp
 *  - https://github.com/ihaveamac/ninfs/blob/main/ninfs/mount/nandhac.py
 *  - https://gitlab.com/roothorick/busehac
 *  - https://github.com/ihaveamac/switchfs
 */

import crypto from 'node:crypto';

export class XtsCrypto {
  // TODO: sector size? is it necessary?
  constructor(private readonly cryptoKey: Buffer, private readonly tweakKey: Buffer) {}

  private createTweak(offset: number): Buffer {
    const tweak = Buffer.alloc(16, 0);
    for (let i = 0; i < 4; i++) {
      tweak[15 - i] = (offset >> (i * 8)) & 0xff;
    }

    const cipher = crypto.createCipheriv('aes-128-ecb', this.tweakKey, null).setAutoPadding(false);
    return Buffer.concat([cipher.update(tweak), cipher.final()]);
  }

  private applyTweak(tweak: Buffer, data: Buffer): void {
    const buf = Buffer.from(tweak);
    for (let i = 0; i < data.length; i += 16) {
      for (let j = 0; j < 16; j++) {
        data[i + j] ^= buf[j];
      }

      const lastHigh = buf[15] & 0x80;
      for (let j = 15; j > 0; j--) {
        buf[j] = ((buf[j] << 1) & ~1) | (buf[j - 1] & 0x80 ? 1 : 0);
      }

      buf[0] = ((buf[0] << 1) & ~1) ^ (lastHigh ? 0x87 : 0);
    }
  }

  private doCrypt(input: Buffer, offset: number, method: 'createCipheriv' | 'createDecipheriv'): Buffer {
    const tweak = this.createTweak(offset);

    const data = Buffer.from(input);
    this.applyTweak(tweak, data);

    const cipher = crypto[method]('aes-128-ecb', this.cryptoKey, null).setAutoPadding(false);
    const encrypted = Buffer.concat([cipher.update(data), cipher.final()]);

    this.applyTweak(tweak, encrypted);

    return encrypted;
  }

  public encrypt(input: Buffer, offset: number): Buffer {
    return this.doCrypt(input, offset, 'createCipheriv');
  }

  public decrypt(input: Buffer, offset: number): Buffer {
    return this.doCrypt(input, offset, 'createDecipheriv');
  }
}
