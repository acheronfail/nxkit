import fsp from 'node:fs/promises';
import fs from 'node:fs';
import * as FatFs from 'js-fatfs';
import { Crypto, NxCrypto } from '../src/node/nand/fatfs/crypto';
import { createIo } from '../src/node/nand/fatfs/io';
import { NandIoLayer } from '../src/node/nand/fatfs/layer';
import { Xtsn } from '../src/node/nand/xtsn';
import { PartitionDriver } from '../src/node/nand/fatfs/diskio';
import { Fat32FileSystem, check_result } from '../src/node/nand/fatfs/fs';
import { BiosParameterBlock } from '../src/node/nand/fatfs/bpb';
import timers from '../src/timers';

//
// create files
//

const createFile = async (path: string, size: number) => {
  const handle = await fsp.open(path, 'w+');
  await handle.truncate(size);
  await handle.close();
  return { path, size };
};

const disk = await createFile('/tmp/disk.img', 2 ** 31);
const file = await createFile('/tmp/100m.dat', 100_000_000);

//
// create file system overlay
//

const createFs = async (size: number, crypto?: Crypto) => {
  const sectorSize = 512;

  // the "NandIoLayer" wraps the encryption logic (and reading writing to partition offsets)
  const nandIo = new NandIoLayer({
    // the "io" wraps a file descriptor
    io: await createIo(disk.path),
    partitionStartOffset: 0,
    partitionEndOffset: size,
    sectorSize,
    // this is "undefined" when no encryption used
    crypto,
  });

  // create FAT32 WASM driver
  const ff = await FatFs.create({
    diskio: new PartitionDriver({ nandIo, readonly: false }),
  });

  // format the blank disk with an empty FAT32 filesystem
  const work = ff.malloc(FatFs.FF_MAX_SS);
  check_result(ff.f_mkfs('', 0, work, FatFs.FF_MAX_SS), 'f_mkfs');
  ff.free(work);

  // setup my "Fat32FileSystem" overlay, which has high level "read and write apis", etc
  const bpb = new BiosParameterBlock(nandIo.read(0, 512));
  return new Fat32FileSystem(ff, bpb);
};

//
// benchmark code
//

const benchmark = async (name: string, crypto?: Crypto) => {
  const fat32 = await createFs(disk.size, crypto);
  const stop = timers.start(name);
  let offset = 0;

  const handle = await fsp.open(file.path);
  fat32.writeFile(
    '/file',
    (size) => {
      const buf = Buffer.alloc(size);
      const bytesRead = fs.readSync(handle.fd, buf, 0, size, offset);
      offset += bytesRead;
      return buf.subarray(0, bytesRead);
    },
    true,
  );
  await handle.close();
  stop();
};

//
// run benchmarks
//

for (let i = 0; i < 5; i++) {
  await benchmark('clear', undefined);
  await benchmark('xtsn', new NxCrypto(new Xtsn(Buffer.alloc(16), Buffer.alloc(16))));
}

timers.completeAll();
