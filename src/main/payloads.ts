import fs from 'node:fs/promises';
import { basename, join, resolve } from 'node:path';
import { app } from 'electron';
import prettyBytes from 'pretty-bytes';
import { getResources } from '../resources';
import { FSFile } from '../nand/fatfs/fs';

export async function readPayload(payloadPath: string): Promise<Uint8Array> {
  const { payloadDirectory } = getResources(app.isPackaged);
  const resolvedPath = resolve(payloadPath);
  if (!resolvedPath.startsWith(payloadDirectory)) {
    throw new Error(`Expected ${resolvedPath} to be within ${payloadDirectory}!`);
  }

  return fs.readFile(resolvedPath);
}

async function pathExists(path: string): Promise<boolean> {
  try {
    await fs.stat(path);
    return true;
  } catch {
    return false;
  }
}

export async function copyInFiles(filePaths: string[]): Promise<void> {
  const { payloadDirectory } = getResources(app.isPackaged);
  await Promise.all(
    filePaths.map(async (filePath) => {
      const stats = await fs.stat(filePath);
      if (!stats.isFile()) return;

      // find a path that doesn't exist
      let targetPath = join(payloadDirectory, basename(filePath));
      if (await pathExists(targetPath)) {
        let count = 1;
        let nextPath: string;
        do {
          nextPath = `${targetPath.replace(/\.bin$/, '')}.copy_${count}.bin`;
          count++;
        } while (await pathExists(nextPath));
        targetPath = nextPath;
      }

      // copy file into path
      await fs.copyFile(filePath, targetPath);
    }),
  );
}

export async function findPayloads(): Promise<FSFile[]> {
  const { payloadDirectory } = getResources(app.isPackaged);
  await fs.mkdir(payloadDirectory, { recursive: true });

  const payloads = await fs.readdir(payloadDirectory);
  return Promise.all(
    payloads
      .filter((name) => name.endsWith('.bin'))
      .map(async (name) => {
        const path = join(payloadDirectory, name);
        const stat = await fs.stat(path);
        return {
          name,
          path,
          size: stat.size,
          sizeHuman: prettyBytes(stat.size),
          type: 'f',
        };
      }),
  );
}
