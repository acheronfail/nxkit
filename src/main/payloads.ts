import fs from 'node:fs/promises';
import { join, resolve } from 'node:path';
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

export async function findPayloads(): Promise<FSFile[]> {
  const { payloadDirectory } = getResources(app.isPackaged);
  await fs.mkdir(payloadDirectory, { recursive: true });

  const payloads = await fs.readdir(payloadDirectory);
  return Promise.all(
    payloads.map(async (name) => {
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
