import fsp from 'node:fs/promises';
import CHS from 'chs';
import MBR from 'mbr';
import GPT from 'gpt';
import * as FatFs from 'js-fatfs';
import { Keys, RawKeys, RawKeysSchema } from '../src/main/keys.types';
import { NX_PARTITIONS, PartitionFormat } from '../src/nand/constants';
import { Xtsn } from '../src/nand/xtsn';
import { NxCrypto, Crypto } from '../src/nand/fatfs/crypto';
import { NandIoLayer } from '../src/nand/fatfs/layer';
import { createIo } from '../src/nand/fatfs/io';
import { FatType, check_result } from '../src/nand/fatfs/fs';
import { PartitionDriver } from '../src/nand/fatfs/diskio';
import { PartitionEntry, getPartitionTable } from '../src/nand/gpt';

const argv = process.argv.slice(2);
const split = argv.includes('--split'); // create a split dump
const clear = argv.includes('--clear'); // do not encrypt partitions

const emptyKeys = (Object.keys(RawKeysSchema.shape) as (keyof RawKeys)[]).reduce<RawKeys>((keys, prop) => {
  switch (prop) {
    case 'bis_key_00':
    case 'bis_key_01':
    case 'bis_key_02':
    case 'bis_key_03':
    case 'bis_key_source_00':
    case 'bis_key_source_01':
    case 'bis_key_source_02':
    case 'header_key':
    case 'header_key_source':
    case 'sd_card_custom_storage_key_source':
    case 'sd_card_nca_key_source':
    case 'sd_card_save_key_source':
      keys[prop] = '0'.repeat(64);
      break;
    case 'keyblob_00':
    case 'keyblob_01':
    case 'keyblob_02':
    case 'keyblob_03':
    case 'keyblob_04':
    case 'keyblob_05':
      keys[prop] = '0'.repeat(288);
      break;
    case 'ssl_rsa_key':
      keys[prop] = '0'.repeat(512);
      break;
    case 'eticket_rsa_keypair':
      keys[prop] = '0'.repeat(1056);
      break;
    default:
      keys[prop] = '0'.repeat(32);
      break;
  }
  return keys;
}, {} as RawKeys);

const keys = new Keys('prod.keys', emptyKeys);

await fsp.rm('.data', { recursive: true, force: true });
await fsp.mkdir('.data', { recursive: true });
await fsp.writeFile('.data/prod.keys', keys.toString());

const imageSize = 31268536320;
const chunkSize = 2147483648;

{
  const handle = await fsp.open('.data/rawnand.bin', 'w');
  await handle.truncate(imageSize);

  const gpt = new GPT({
    blockSize: 512,
    guid: 'EDD7049E-B2D3-4067-B3D9-E5A8F398258F',
    headerSize: 92,
    currentLBA: 1n,
    backupLBA: 61071359n,
    firstLBA: 34n,
    lastLBA: 61071326n,
    tableOffset: 2n,
    entrySize: 128,
  });
  gpt.partitions.push(
    new GPT.PartitionEntry({
      type: '98109E25-64E2-4C95-8A77-414916F5BCEB',
      guid: '5561E2D3-9B30-4D80-A546-10EB7C0151FC',
      name: 'PRODINFO',
      firstLBA: 34n,
      lastLBA: 8191n,
      attr: 1n,
    }),
    new GPT.PartitionEntry({
      type: 'F3056AEC-5449-494C-9F2C-5FDCB75B6E6E',
      guid: '5561E2D3-9B30-4D80-A546-10EB7C0151FC',
      name: 'PRODINFOF',
      firstLBA: 8192n,
      lastLBA: 16383n,
      attr: 1n,
    }),
    new GPT.PartitionEntry({
      type: '5365DE36-911B-4BB4-8FF9-AA1EBCD73990',
      guid: '755272B7-445C-46A3-987B-D40E5D25EB83',
      name: 'BCPKG2-1-Normal-Main',
      firstLBA: 16384n,
      lastLBA: 32767n,
      attr: 1n,
    }),
    new GPT.PartitionEntry({
      type: '8455717B-BD2B-4162-8454-91695218FC38',
      guid: 'EAD904D9-61A3-4DBA-BB11-6E516A1F4093',
      name: 'BCPKG2-2-Normal-Sub',
      firstLBA: 32768n,
      lastLBA: 49151n,
      attr: 1n,
    }),
    new GPT.PartitionEntry({
      type: '8ED6C9A6-9C48-490B-BBEB-001D17A4C0F7',
      guid: 'EF78007A-D02C-4BF8-9BEF-B5B5CB3F2B76',
      name: 'BCPKG2-3-SafeMode-Main',
      firstLBA: 49152n,
      lastLBA: 65535n,
      attr: 1n,
    }),
    new GPT.PartitionEntry({
      type: '5E99751C-56C9-47CC-AA30-B65039888917',
      guid: 'DACB7CD3-5624-41D9-85BF-DB61AE5A0096',
      name: 'BCPKG2-4-SafeMode-Sub',
      firstLBA: 65536n,
      lastLBA: 81919n,
      attr: 1n,
    }),
    new GPT.PartitionEntry({
      type: 'C447D9A2-24B7-468A-98C8-595CD077165A',
      guid: '1C58F253-945E-4F24-95F2-29091B775F56',
      name: 'BCPKG2-5-Repair-Main',
      firstLBA: 81920n,
      lastLBA: 98303n,
      attr: 1n,
    }),
    new GPT.PartitionEntry({
      type: '9586E1A1-3AA2-4C90-91B3-2F4A5195B4D2',
      guid: '5561E2D3-9B30-4D80-A546-10EB7C0151FC',
      name: 'BCPKG2-6-Repair-Sub',
      firstLBA: 98304n,
      lastLBA: 114687n,
      attr: 1n,
    }),
    new GPT.PartitionEntry({
      type: 'A44F9F6B-4ED3-441F-A34A-56AAA136BC6A',
      guid: '5561E2D3-9B30-4D80-A546-10EB7C0151FC',
      name: 'SAFE',
      firstLBA: 114688n,
      lastLBA: 245759n,
      attr: 1n,
    }),
    new GPT.PartitionEntry({
      type: 'ACB0CDF0-4F72-432D-AA0D-5388C733B224',
      guid: '5561E2D3-9B30-4D80-A546-10EB7C0151FC',
      name: 'SYSTEM',
      firstLBA: 245760n,
      lastLBA: 5488639n,
      attr: 1n,
    }),
    new GPT.PartitionEntry({
      type: '2B777F63-E842-47AF-94C4-25A7F18B2280',
      guid: '5561E2D3-9B30-4D80-A546-10EB7C0151FC',
      name: 'USER',
      firstLBA: 5488640n,
      lastLBA: 60014591n,
      attr: 1n,
    }),
  );

  const mbr = MBR.parse(MBR.createBuffer());
  mbr.partitions[0].status = 0;
  mbr.partitions[0].type = 238;
  mbr.partitions[0].sectors = 4294967295;
  mbr.partitions[0].firstLBA = 1;
  mbr.partitions[0].firstCHS = new CHS(8, 0, 0);
  mbr.partitions[0].lastCHS = new CHS(1023, 255, 63);

  // Create GPT partition table
  const primaryGpt = Buffer.alloc(gpt.blockSize + 33 * gpt.blockSize);
  const backupGpt = Buffer.alloc(33 * gpt.blockSize);

  primaryGpt.set(mbr.buffer, 0);
  gpt.write(primaryGpt, gpt.blockSize);
  gpt.writeBackupFromPrimary(backupGpt);

  await handle.write(primaryGpt, 0, primaryGpt.byteLength, 0);
  await handle.write(backupGpt, 0, backupGpt.byteLength, Number(gpt.backupLBA) * gpt.blockSize);
  await handle.close();
}

// Format partitions with blank keys
async function formatPartition(name: string) {
  console.log(`Formatting ${name}...`);
  const io = await createIo('.data/rawnand.bin');
  const gpt = getPartitionTable(io);

  const partition = gpt.partitions.find((part: PartitionEntry) => part.name === name);
  if (!partition) throw new Error(`Failed to find partition: ${name}`);

  const nxPartition = NX_PARTITIONS[partition.type];
  if (!nxPartition) throw new Error(`Failed to find nx partition: ${partition.type}`);

  const partitionStartOffset = Number(partition.firstLBA) * gpt.blockSize;
  const partitionEndOffset = Number(partition.lastLBA + 1n) * gpt.blockSize;

  const { bisKeyId } = NX_PARTITIONS[partition.type];
  let crypto: Crypto | undefined = undefined;
  if (!clear && typeof bisKeyId === 'number') {
    const bisKey = keys.getBisKey(bisKeyId);
    crypto = new NxCrypto(new Xtsn(bisKey.crypto, bisKey.tweak));
  }

  const nandIo = new NandIoLayer({
    io,
    partitionStartOffset,
    partitionEndOffset,
    crypto,
  });

  const ff = await FatFs.create({ diskio: new PartitionDriver({ nandIo, readonly: false, sectorSize: 512 }) });

  const opts = {
    fmt: nxPartition.format === PartitionFormat.Fat12 ? FatType.Fat : FatType.Fat32 | FatFs.FM_SFD,
    n_fat: 2,
    au_size: 512 * (nxPartition.format === PartitionFormat.Fat12 ? 1 : 32),
    align: 0,
    n_root: 0,
  };

  const work = ff.malloc(FatFs.FF_MAX_SS);
  check_result(ff.f_mkfs('', opts, work, FatFs.FF_MAX_SS), 'f_mkfs');
  ff.free(work);

  io.close();
}

// FIXME: PRODINFOF isn't formatting correctly
console.error('FIXME: the PRODINFOF partition will not work (needs to be formatted)');
// await formatPartition('PRODINFOF');

// FIXME: why do these all show 31.3 GB free space? did I format wrong?
await formatPartition('SAFE');
await formatPartition('SYSTEM');
await formatPartition('USER');

if (split) {
  const fullHandle = await fsp.open('.data/rawnand.bin', 'r+');
  let offset = imageSize - (imageSize % chunkSize);
  let count = Math.floor(imageSize / chunkSize);

  while (offset > -1) {
    const splitPath = `.data/rawnand.bin.${count.toString().padStart(2, '0')}`;
    console.log(`Creating ${splitPath}...`);

    const start = offset;
    const end = Math.min(imageSize, offset + chunkSize - 1);
    const splitHandle = await fsp.open(splitPath, 'w');
    const src = fullHandle.createReadStream({ start, end, autoClose: false });
    const dst = splitHandle.createWriteStream();
    await new Promise((resolve, reject) => {
      src.pipe(dst);
      dst.on('finish', resolve);
      src.on('error', reject);
      dst.on('error', reject);
    });

    await splitHandle.close();
    await fullHandle.truncate(offset);
    offset -= chunkSize;
    count--;
  }

  await fullHandle.close();
  await fsp.unlink('.data/rawnand.bin');
}
