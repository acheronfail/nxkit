import { describe, expect, test } from 'vitest';
import { NandIoLayer } from './layer';
import { CLUSTER_SIZE, DISK_SIZE, XorOffsetCrypto, getLayer } from './layer.test.util';

describe(NandIoLayer.name, () => {
  describe('no crypto', () => {
    test('read', () => {
      const disk = Buffer.alloc(DISK_SIZE, 0);
      disk.set([1, 2, 3, 4, 5], 0);

      const layer = getLayer(disk);

      expect(Array.from(layer.read(0, 5))).toEqual([1, 2, 3, 4, 5]);
      expect(Array.from(layer.read(5, 5))).toEqual([0, 0, 0, 0, 0]);
    });

    test('write', () => {
      const disk = Buffer.alloc(DISK_SIZE, 0);
      const layer = getLayer(disk);

      layer.write(0, Buffer.from([1, 2, 3, 4, 5]));
      expect(Array.from(disk.subarray(0, 5))).toEqual([1, 2, 3, 4, 5]);
      expect(Array.from(disk.subarray(5, 10))).toEqual([0, 0, 0, 0, 0]);
    });
  });

  describe('crypto', () => {
    const createEmptyCryptoDisk = (size: number) => {
      const disk = Buffer.alloc(size);
      for (let i = 0; i * CLUSTER_SIZE < size; i++) {
        disk.set(Buffer.alloc(CLUSTER_SIZE, 0 ^ i), CLUSTER_SIZE * i);
      }

      return disk;
    };

    test('read', () => {
      const disk = createEmptyCryptoDisk(DISK_SIZE);
      // set first 5 bytes of 1st and 2nd cluster to `42`
      disk.set([42, 42, 42, 42, 42], 0);
      disk.set([42, 42, 42, 42, 42], CLUSTER_SIZE);

      const layer = getLayer(disk, new XorOffsetCrypto());

      // since 1st cluster just XOR's with 0, it's still 42
      expect(Array.from(layer.read(0, 5))).toEqual([42, 42, 42, 42, 42]);
      // but 2nd cluster XOR's with 1, so it's 43
      expect(Array.from(layer.read(CLUSTER_SIZE, 5))).toEqual([43, 43, 43, 43, 43]);
    });

    test('read over cluster bounds', () => {
      const disk = createEmptyCryptoDisk(DISK_SIZE);
      // 1st cluster: all `0`
      // 2nd cluster: all `42` (XOR'd value is 43)
      // 3rd cluster: all `0`
      // (rest all `1`)
      disk.set(Buffer.alloc(CLUSTER_SIZE, 43), CLUSTER_SIZE);

      const layer = getLayer(disk, new XorOffsetCrypto());

      // read 1st cluster + first byte from 2nd cluster
      expect([...layer.read(0, CLUSTER_SIZE + 1)]).toEqual([...Buffer.alloc(CLUSTER_SIZE, 0), 42]);
      // read 2nd cluster + last byte from 1st cluster
      expect([...layer.read(CLUSTER_SIZE - 1, CLUSTER_SIZE + 1)]).toEqual([0, ...Buffer.alloc(CLUSTER_SIZE, 42)]);
      // read last byte from 1st cluster, entire 2nd cluster, first byte from 3rd cluster
      expect([...layer.read(CLUSTER_SIZE - 1, CLUSTER_SIZE + 2)]).toEqual([0, ...Buffer.alloc(CLUSTER_SIZE, 42), 0]);
    });

    test('write', () => {
      const disk = createEmptyCryptoDisk(DISK_SIZE);
      const layer = getLayer(disk, new XorOffsetCrypto());

      // set first 5 bytes of 1st and 2nd cluster to `42`
      expect(layer.write(0, Buffer.from([42, 42, 42, 42, 42]))).toBe(CLUSTER_SIZE);
      expect(layer.write(CLUSTER_SIZE, Buffer.from([42, 42, 42, 42, 42]))).toBe(CLUSTER_SIZE);

      expect([...disk.subarray(0, CLUSTER_SIZE)]).toEqual([42, 42, 42, 42, 42, ...Buffer.alloc(CLUSTER_SIZE - 5, 0)]);
      expect([...disk.subarray(CLUSTER_SIZE, CLUSTER_SIZE * 2)]).toEqual([
        43,
        43,
        43,
        43,
        43,
        ...Buffer.alloc(CLUSTER_SIZE - 5, 1),
      ]);
    });

    test('write: entire 1st cluster + first byte of 2nd cluster', () => {
      const disk = createEmptyCryptoDisk(DISK_SIZE);
      const layer = getLayer(disk, new XorOffsetCrypto());

      expect(layer.write(0, Buffer.alloc(CLUSTER_SIZE + 1, 42))).toBe(CLUSTER_SIZE * 2);
      expect([...disk.subarray(0, CLUSTER_SIZE + 1)]).toEqual([...Buffer.alloc(CLUSTER_SIZE, 42), 43]);
    });

    test('write: last byte of 1st cluster + entire 2nd cluster', () => {
      const disk = createEmptyCryptoDisk(DISK_SIZE);
      const layer = getLayer(disk, new XorOffsetCrypto());

      expect(layer.write(CLUSTER_SIZE - 1, Buffer.alloc(CLUSTER_SIZE + 1, 42))).toBe(CLUSTER_SIZE * 2);
      expect([...disk.subarray(0, CLUSTER_SIZE * 2)]).toEqual([
        ...Buffer.alloc(CLUSTER_SIZE - 1, 0),
        42,
        ...Buffer.alloc(CLUSTER_SIZE, 43),
      ]);
    });

    test('write: last byte of 1st cluster + entire 2nd cluster + first byte of 3rd cluster', () => {
      const disk = createEmptyCryptoDisk(DISK_SIZE);
      const layer = getLayer(disk, new XorOffsetCrypto());

      expect(layer.write(CLUSTER_SIZE - 1, Buffer.alloc(CLUSTER_SIZE + 2, 42))).toBe(CLUSTER_SIZE * 3);
      expect([...disk.subarray(0, CLUSTER_SIZE * 3)]).toEqual([
        ...Buffer.alloc(CLUSTER_SIZE - 1, 0),
        42,
        ...Buffer.alloc(CLUSTER_SIZE, 43),
        40,
        ...Buffer.alloc(CLUSTER_SIZE - 1, 2),
      ]);
    });
  });
});
