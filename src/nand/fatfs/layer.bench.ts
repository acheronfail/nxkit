import { describe, bench } from 'vitest';
import crypto from 'node:crypto';
import { XorOffsetCrypto, getLayer } from './layer.test';
import { NxCrypto } from './crypto';
import { Xtsn } from '../xtsn';

describe('benchmarks', () => {
  const size = 16384;

  const data = crypto.randomBytes(size);

  const clear = getLayer(Buffer.alloc(size));
  bench('clear', () => {
    clear.write(0, data);
  });

  const xor = getLayer(Buffer.alloc(size), new XorOffsetCrypto());
  bench('xor', () => {
    xor.write(0, data);
  });

  const xtsn = getLayer(Buffer.alloc(size), new NxCrypto(new Xtsn(Buffer.alloc(16), Buffer.alloc(16), 0x4000)));
  bench('xtsn', () => {
    xtsn.write(0, data);
  });
});
