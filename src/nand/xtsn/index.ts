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

export class Xtsn {
  constructor(
    private readonly cryptoKey: Buffer,
    private readonly tweakKey: Buffer,
  ) {
    this.cryptoKey = cryptoKey;
    this.tweakKey = tweakKey;
  }

  private createTweak(clusterOffset: number) {
    const tweak = Buffer.alloc(16, 0);
    tweak.writeBigInt64BE(BigInt(clusterOffset), 8);

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
    sectorSize: number,
    skippedBytes: number,
    method: 'createCipheriv' | 'createDecipheriv',
  ) {
    const data = Buffer.from(input);
    let currentOffset = 0;
    const processBlock = (tweak: Buffer, runs: number) => {
      for (let i = 0; i < runs; i++) {
        const block = data.subarray(currentOffset, currentOffset + 16);
        this.applyTweak(tweak, block);
        const processedBlock = this.aesEcb(block, this.cryptoKey, method);
        this.applyTweak(tweak, processedBlock);
        this.updateTweak(tweak);
        processedBlock.copy(data, currentOffset);

        currentOffset += 16;
      }
    };

    if (skippedBytes > 0) {
      const fullSectorsToSkip = Math.floor(skippedBytes / sectorSize);
      sectorOffset += fullSectorsToSkip;
      skippedBytes %= sectorSize;
    }

    if (skippedBytes > 0) {
      const tweak = this.createTweak(sectorOffset);
      for (let i = 0; i < Math.floor(skippedBytes / 16); i++) {
        this.updateTweak(tweak);
      }

      processBlock(tweak, Math.floor((sectorSize - skippedBytes) / 16));
      sectorOffset++;
    }

    while (currentOffset < data.byteLength) {
      const tweak = this.createTweak(sectorOffset);
      processBlock(tweak, Math.floor(sectorSize / 16));
      sectorOffset++;
    }

    return data;
  }

  public encrypt(input: Buffer, byteOffset = 0, sectorSize = 0x200): Buffer {
    const sectorOffset = Math.floor(byteOffset / sectorSize);
    return this.doCrypt(input, sectorOffset, sectorSize, byteOffset % sectorSize, 'createCipheriv');
  }

  public decrypt(input: Buffer, byteOffset = 0, sectorSize = 0x200): Buffer {
    const sectorOffset = Math.floor(byteOffset / sectorSize);
    return this.doCrypt(input, sectorOffset, sectorSize, byteOffset % sectorSize, 'createDecipheriv');
  }

  // Expose APIs that are compatible with the python haccrypto lib

  public encryptHC(input: Buffer, sectorOffset: number, sectorSize = 0x200, skippedBytes = 0): Buffer {
    return this.doCrypt(input, sectorOffset, sectorSize, skippedBytes, 'createCipheriv');
  }

  public decryptHC(input: Buffer, sectorOffset: number, sectorSize = 0x200, skippedBytes = 0): Buffer {
    return this.doCrypt(input, sectorOffset, sectorSize, skippedBytes, 'createDecipheriv');
  }
}
