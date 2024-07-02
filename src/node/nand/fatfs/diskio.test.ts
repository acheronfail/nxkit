import { describe, expect, test } from 'vitest';
import * as FatFs from 'js-fatfs';
import { NxDiskIo, ReadonlyError } from './diskio';
import { SECTOR_SIZE, getLayer } from './layer.test.util';

const HEAP_SIZE = 1024;
const mockFatFs = () =>
  ({
    HEAPU8: new Uint8Array(Buffer.alloc(HEAP_SIZE, 0)),
  }) as FatFs.FatFs;

describe(NxDiskIo.name, () => {
  test('reads from the disk', () => {
    // first half of disk is 0xff, second half is 0x88
    const disk = Buffer.alloc(256);
    disk.set(Buffer.alloc(128, 0xff), 0);
    disk.set(Buffer.alloc(128, 0x80), 128);
    const diskio = new NxDiskIo({ readonly: true, ioLayer: getLayer(disk) });

    // set every value in the ff heap to 0
    const ff = mockFatFs();
    ff.HEAPU8.set(Buffer.alloc(HEAP_SIZE, 0));

    // read a single sector from the start of the disk into the heap
    diskio.read(ff, 0, 0, 0, 1);
    expect([...ff.HEAPU8.subarray(0, SECTOR_SIZE)]).toEqual([...Buffer.alloc(SECTOR_SIZE, 0xff)]);

    // read 2 sectors from the disk into the heap
    const heapPos = 512;
    diskio.read(ff, 0, heapPos, 1, 2);
    expect([...ff.HEAPU8.subarray(heapPos, heapPos + SECTOR_SIZE)]).toEqual([...Buffer.alloc(SECTOR_SIZE, 0xff)]);
    expect([...ff.HEAPU8.subarray(heapPos + SECTOR_SIZE, heapPos + SECTOR_SIZE * 2)]).toEqual([
      ...Buffer.alloc(SECTOR_SIZE, 0x80),
    ]);
  });

  test('writes from the heap', () => {
    // set the disk all to 0
    const disk = Buffer.alloc(256, 0);
    const diskio = new NxDiskIo({ readonly: false, ioLayer: getLayer(disk) });

    // first half of heap is 0xff, second half of heap is 0x80
    const ff = mockFatFs();
    const middleOfHeap = Math.floor(HEAP_SIZE / 2);
    ff.HEAPU8.set(Buffer.alloc(HEAP_SIZE, 0xff));
    ff.HEAPU8.set(Buffer.alloc(middleOfHeap, 0x80), middleOfHeap);

    // write a single sector from the start of the heap into the disk's first sector
    diskio.write(ff, 0, 0, 0, 1);
    expect([...disk.subarray(0, SECTOR_SIZE)]).toEqual([...Buffer.alloc(SECTOR_SIZE, 0xff)]);

    // the heap has values like this: (where XX is a sector filled with bytes of XX)
    // ff ff ff ff 80 80 80 80
    //          ^^ ^^
    // we're writing the sectors outlined with ^^ into the 2nd and 3rd sectors of the disk
    diskio.write(ff, 0, middleOfHeap - SECTOR_SIZE, 1, 2);
    expect([...disk.subarray(SECTOR_SIZE, SECTOR_SIZE * 3)]).toEqual([
      ...Buffer.alloc(SECTOR_SIZE, 0xff),
      ...Buffer.alloc(SECTOR_SIZE, 0x80),
    ]);
  });

  test('it throws ReadonlyError if readonly is enabled and asked to write', () => {
    const disk = Buffer.alloc(128);
    // set readonly
    const diskio = new NxDiskIo({ readonly: true, ioLayer: getLayer(disk) });

    // set every value in the ff heap to 0xff
    const ff = mockFatFs();
    ff.HEAPU8.set(Buffer.alloc(ff.HEAPU8.byteLength, 0xff));

    // try to write a single sector from the start of the heap
    expect(() => diskio.write(ff, 0, 0, 0, 1)).toThrowError(
      new ReadonlyError('Tried to write to a disk in readonly mode!'),
    );

    // also verify nothing was written to disk
    expect([...disk]).toEqual([...Buffer.alloc(128, 0)]);
  });
});
