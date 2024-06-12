/* eslint-disable @typescript-eslint/no-explicit-any */
import { open } from 'fs/promises';
import { readSync } from 'fs';
import GPT from 'gpt';
import MBR from 'mbr';
import prettyBytes from 'pretty-bytes';
import * as FatFs from 'js-fatfs';
import { Partition as PartitionView, check_result } from './fatfs';
import { FileHandle } from './types';
import { XtsCrypto } from './xts';
import { Keys } from '../main/keys';

// TODO: support reading split dumps

// TODO: expose these to renderer
export type FSFile = { type: 'f'; name: string; path: string; size: number; sizeHuman: string };
export type FSDirectory = { type: 'd'; name: string; path: string };
export type FSEntry = FSFile | FSDirectory;

export async function nandExplorerPoc(nandPath: string, keys: Keys): Promise<string> {
  const blockSize = 512;
  const nandHandle = await open(nandPath, 'r');

  // read mbr
  const buffer = Buffer.alloc(512);
  readSync(nandHandle.fd, buffer, 0, buffer.length, 0);

  // find efi
  const mbr = MBR.parse(buffer);
  const efiPart = mbr.getEFIPart();

  // read gpt
  // TODO: also read backup gpt, and verify both
  const primaryGPT = readPrimaryGPT(nandHandle, blockSize, efiPart);

  // TODO: list and choose partitions
  const systemPart = primaryGPT.partitions.find((part: any) => part.name === 'SYSTEM');

  const partStart = Number(systemPart.firstLBA) * blockSize;
  const partEnd = Number(systemPart.lastLBA + 1n) * blockSize;

  // TODO: detect which bis keys to use https://github.com/ihaveamac/ninfs/blob/main/ninfs/mount/nandhac.py#L24
  const { crypto, tweak } = keys.getBisKey(2);
  const xts = new XtsCrypto(crypto, tweak);

  // TODO: verify keys are correct before attempting to boot up fat32

  // fatfs
  // TODO: wrapper around this API to make it more ergonomic

  const diskio = new PartitionView(nandHandle.fd, partStart, partEnd, nandPath.endsWith('.bin') ? xts : undefined);
  const ff = await FatFs.create({ diskio });

  const fatfs = ff.malloc(FatFs.sizeof_FATFS);
  check_result(ff.f_mount(fatfs, '', 1));

  const dir = ff.malloc(FatFs.sizeof_DIR);
  const fno = ff.malloc(FatFs.sizeof_FILINFO);
  check_result(ff.f_opendir(dir, '/'));

  const entries: FSEntry[] = [];
  for (;;) {
    check_result(ff.f_readdir(dir, fno));
    const name = ff.FILINFO_fname(fno);
    if (name === '') break;

    const path = `/${name}`;
    const size = ff.FILINFO_fsize(fno);
    const isDir = ff.FILINFO_fattrib(fno) & FatFs.AM_DIR;

    entries.push(isDir ? { type: 'd', name, path } : { type: 'f', name, path, size, sizeHuman: prettyBytes(size) });
  }

  // Clean up.
  ff.free(fno);
  check_result(ff.f_closedir(dir));
  ff.free(dir);
  check_result(ff.f_unmount(''));
  ff.free(fatfs);

  await nandHandle.close();

  return entries.map((ent) => (ent.type == 'd' ? ent.name : [ent.name, ent.sizeHuman].join(' - '))).join('\n');
}

function readPrimaryGPT(handle: FileHandle, blockSize: number, efiPart: any) {
  const gpt = new GPT({ blockSize });

  // NOTE: For protective GPTs (0xEF), the MBR's partitions
  // attempt to span as much of the device as they can to protect
  // against systems attempting to action on the device,
  // so the GPT is then located at LBA 1, not the EFI partition's first LBA
  const offset = efiPart.type == 0xee ? efiPart.firstLBA * gpt.blockSize : gpt.blockSize;

  // First, we need to read & parse the GPT header, which will declare various
  // sizes and offsets for us to calculate where & how long the table and backup are
  const headerBuffer = Buffer.alloc(gpt.blockSize);

  readSync(handle.fd, headerBuffer, 0, headerBuffer.length, offset);
  gpt.parseHeader(headerBuffer);

  // Now on to reading the actual partition table
  const tableBuffer = Buffer.alloc(gpt.tableSize);
  // NOTE: gpt.tableOffset is a BigInt
  const tableOffset = Number(gpt.tableOffset) * gpt.blockSize;
  readSync(handle.fd, tableBuffer, 0, tableBuffer.length, tableOffset);

  // We need to parse the first 4 partition entries & the rest separately
  // as the first 4 table entries always occupy one block,
  // with the rest following in subsequent blocks
  gpt.parseTable(tableBuffer, 0, gpt.blockSize);
  gpt.parseTable(tableBuffer, gpt.blockSize, gpt.tableSize);

  return gpt;
}
