import fsp from 'node:fs/promises';
import fs from 'node:fs';
import chalk from 'chalk';
import { randomBytes } from 'node:crypto';
import * as FatFs from 'js-fatfs';
import { Bench, Task, TaskResult } from 'tinybench';
import { Crypto, NxCrypto } from '../src/node/nand/fatfs/crypto';
import { createIo } from '../src/node/nand/fatfs/io';
import { NandIoLayer } from '../src/node/nand/fatfs/layer';
import { Xtsn } from '../src/node/nand/xtsn';
import { NxDiskIo } from '../src/node/nand/fatfs/diskio';
import { Fat32FileSystem, FatError } from '../src/node/nand/fatfs/fs';
import { BiosParameterBlock } from '../src/node/nand/fatfs/bpb';
import { getLayer, XorOffsetCrypto } from '../src/node/nand/fatfs/layer.test.util';

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
    diskio: new NxDiskIo({ ioLayer: nandIo, readonly: false }),
  });

  // format the blank disk with an empty FAT32 filesystem
  const work = ff.malloc(FatFs.FF_MAX_SS);
  const res = ff.f_mkfs('', 0, work, FatFs.FF_MAX_SS);
  if (res !== FatFs.RES_OK) {
    throw new FatError(res, 'Failed to format partition for benchmarking');
  }
  ff.free(work);

  // setup my "Fat32FileSystem" overlay, which has high level "read and write apis", etc
  const bpb = new BiosParameterBlock(nandIo.read(0, 512));
  return new Fat32FileSystem(ff, bpb);
};

//
// benchmark code
//

const benchmark = async (name: string, fat32: Fat32FileSystem) => {
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
};

//
// run benchmarks
//

let layer: NandIoLayer;
let fat32: Fat32FileSystem;

const xtsn = new NxCrypto(new Xtsn(Buffer.alloc(16), Buffer.alloc(16)));
const diskSize = 524288;
const dataToWrite = randomBytes(diskSize);

const simpleBench = new Bench({ time: 100 })
  .add(
    'plaintext',
    () => {
      layer.write(0, dataToWrite);
    },
    {
      beforeAll: () => {
        layer = getLayer(Buffer.alloc(diskSize));
      },
    },
  )
  .add(
    'xor',
    () => {
      layer.write(0, dataToWrite);
    },
    {
      beforeAll: () => {
        layer = getLayer(Buffer.alloc(diskSize), new XorOffsetCrypto());
      },
    },
  )
  .add(
    'encryption',
    () => {
      layer.write(0, dataToWrite);
    },
    {
      beforeAll: () => {
        layer = getLayer(Buffer.alloc(diskSize), xtsn);
      },
    },
  );

const integrationBench = new Bench({ time: 10000 })
  .add(
    '100mb file, plaintext',
    async () => {
      await benchmark('clear', fat32);
    },
    {
      async beforeAll() {
        fat32 = await createFs(disk.size);
      },
    },
  )
  .add(
    '100mb file, encrypted',
    async () => {
      await benchmark('xtsn', fat32);
    },
    {
      async beforeAll() {
        fat32 = await createFs(disk.size, xtsn);
      },
    },
  );

type BenchmarkResults = {
  title: string;
  results: { name: string; tasks: { name: string; result: TaskResult }[] }[];
}[];

const results: BenchmarkResults[number]['results'] = [];
async function doBenchmark(benchmark: Bench, name: string) {
  process.stdout.write(`Running benchmark: ${name}...`);
  await benchmark.run();

  const f = (n: number) =>
    `${n.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`.padStart(14);

  const m = (task: Task) => ({
    name: task.name,
    'ops/sec': task.result ? parseInt(task.result.hz.toString(), 10).toLocaleString() : NaN,
    'mean:ms': task.result ? f(task.result.mean) : NaN,
    'mean:us': task.result ? f(task.result.mean * 1_000) : NaN,
    'mean:ns': task.result ? f(task.result.mean * 1_000_000) : NaN,
  });

  process.stdout.write('\r');
  console.table(benchmark.table(m));

  type TaskWithResult = Task & { result: TaskResult };

  const tasksSorted = (benchmark.tasks as TaskWithResult[]).slice().sort((a, b) => a.result.mean - b.result.mean);
  const fastest = tasksSorted[0];

  console.log(`> ${fastest.name} ${chalk.gray('is the fastest')}`);
  for (const task of tasksSorted.slice(1)) {
    const factor = f(task.result.mean / fastest.result.mean).trim();
    console.log(`> ${task.name} ${chalk.cyan(`${factor}x`)} ${chalk.gray('slower than')} ${fastest.name}`);
  }

  results.push({
    name,
    tasks: tasksSorted.map((task) => ({ name: task.name, result: task.result })),
  });
}

// run benchmarks
await doBenchmark(simpleBench, 'simple');
await doBenchmark(integrationBench, 'integration');

// write out results to a file if a title was provided
const title = process.argv[2];
if (title) {
  const outputFile = 'bench.json';
  const output: BenchmarkResults = JSON.parse(await fsp.readFile(outputFile, 'utf-8').catch(() => '[]'));
  output.push({ title, results });
  await fsp.writeFile(outputFile, JSON.stringify(output));

  type Chart = {
    title: string;
    // the name of each different test run
    xValues: string[];
    // a benchmark name and plot for each value in the run
    yValues: [string, number[]][];
    timeUnit: string;
  };

  const formattedFile = 'bench.formatted.json';
  await fsp.writeFile(
    formattedFile,
    JSON.stringify([
      {
        title: 'XTSN - Copy 50k buffer of random bytes',
        timeUnit: 'us',
        xValues: output.map((run) => run.title),
        yValues: simpleBench.tasks.map((_, i) => [
          output[0].results[0].tasks[i].name,
          output.map((run) => {
            return run.results[0].tasks[i].result.mean;
          }),
        ]),
      },
      {
        title: 'XTSN - Copy 100MB File into Encrypted Fat32 Partition',
        timeUnit: 'ms',
        xValues: output.map((run) => run.title),
        yValues: integrationBench.tasks.map((_, i) => [
          output[0].results[1].tasks[i].name,
          output.map((run) => {
            return run.results[1].tasks[i].result.mean;
          }),
        ]),
      },
    ] satisfies Chart[]),
  );
}
