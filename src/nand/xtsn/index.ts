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

/**
 * Some notes:
 *  It's AES-XTS encryption, and AES works in 128 bit (16 byte) blocks
 *  Sector size is 0x4000
 *  XTS tweaks are updated (consecutively) once per sector
 *
 * See https://switchbrew.org/wiki/Flash_Filesystem for more information.
 */
export class Xtsn {
  constructor(
    private readonly cryptoKey: Buffer,
    private readonly tweakKey: Buffer,
    public readonly sectorSize = 0x4000,
  ) {}

  private createTweak(sectorOffset: number) {
    const tweak = Buffer.alloc(16, 0);
    tweak.writeBigInt64BE(BigInt(sectorOffset), 8);

    return this.aesEcb(tweak, this.tweakKey, 'createCipheriv');
  }

  private applyTweak(tweak: Buffer, data: Buffer) {
    for (let j = 0; j < 16; j++) {
      data[j] ^= tweak[j];
    }
  }

  private updateTweak(tweak: Buffer) {
    const lastHigh = tweak[15] & 0x80;
    for (let j = 15; j > 0; j--) {
      tweak[j] = ((tweak[j] << 1) & ~1) | (tweak[j - 1] & 0x80 ? 1 : 0);
    }

    tweak[0] = ((tweak[0] << 1) & ~1) ^ (lastHigh ? 0x87 : 0);
  }

  private aesEcb(data: Buffer, key: Buffer, method: 'createCipheriv' | 'createDecipheriv') {
    const cipher = crypto[method]('aes-128-ecb', key, null).setAutoPadding(false);
    return Buffer.concat([cipher.update(data), cipher.final()]);
  }

  private doCrypt(
    input: Buffer,
    sectorOffset: number,
    skippedBytes: number,
    method: 'createCipheriv' | 'createDecipheriv',
  ) {
    const data = Buffer.from(input);
    let currentSectorOffset = 0;
    const processBlock = (tweak: Buffer, runs: number) => {
      for (let i = 0; i < runs; i++) {
        const block = data.subarray(currentSectorOffset, currentSectorOffset + 16);
        this.applyTweak(tweak, block);
        const processedBlock = this.aesEcb(block, this.cryptoKey, method);
        this.applyTweak(tweak, processedBlock);
        this.updateTweak(tweak);
        processedBlock.copy(data, currentSectorOffset);

        currentSectorOffset += 16;
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

    while (currentSectorOffset < data.byteLength) {
      const tweak = this.createTweak(sectorOffset);
      processBlock(tweak, Math.floor(this.sectorSize / 16));
      sectorOffset++;
    }

    return data;
  }

  public encrypt(input: Buffer, byteOffset = 0): Buffer {
    const sectorOffset = Math.floor(byteOffset / this.sectorSize);
    return this.doCrypt(input, sectorOffset, byteOffset % this.sectorSize, 'createCipheriv');
  }

  public decrypt(input: Buffer, byteOffset = 0): Buffer {
    const sectorOffset = Math.floor(byteOffset / this.sectorSize);
    return this.doCrypt(input, sectorOffset, byteOffset % this.sectorSize, 'createDecipheriv');
  }

  // Expose APIs that are compatible with the python haccrypto lib

  public encryptHC(input: Buffer, sectorOffset: number, sectorSize = 0x200, skippedBytes = 0): Buffer {
    const xtsn = new Xtsn(this.cryptoKey, this.tweakKey, sectorSize);
    return xtsn.doCrypt(input, sectorOffset, skippedBytes, 'createCipheriv');
  }

  public decryptHC(input: Buffer, sectorOffset: number, sectorSize = 0x200, skippedBytes = 0): Buffer {
    const xtsn = new Xtsn(this.cryptoKey, this.tweakKey, sectorSize);
    return xtsn.doCrypt(input, sectorOffset, skippedBytes, 'createDecipheriv');
  }
}
