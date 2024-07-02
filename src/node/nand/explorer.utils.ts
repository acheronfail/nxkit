import fs from 'node:fs';
import fsp from 'node:fs/promises';
import { basename, join } from 'node:path';
import { FSEntry, Fat32FileSystem } from './fatfs/fs';

export interface WalkParams {
  pathInNand: string;
  pathInNandEntry: FSEntry | null;
  pathOnHost: string;
  pathOnHostStats: fs.Stats;
}

export enum WalkAction {
  /** continue walking */
  Continue,
  /** stop walking altogether */
  Stop,
  /** skip this directory tree, but keep walking elsewhere */
  Skip,
}

export type WalkCallback = (params: WalkParams) => Promise<WalkAction>;

async function _walk(fat: Fat32FileSystem, pathOnHost: string, dirPathInNand: string, fn: WalkCallback) {
  const pathInNand = join(dirPathInNand, basename(pathOnHost));

  const pathOnHostStats = await fsp.stat(pathOnHost);
  const pathInNandEntry = fat.read(pathInNand);
  const action = await fn({ pathInNand, pathInNandEntry, pathOnHost, pathOnHostStats });
  if (action !== WalkAction.Continue) return action;

  if (pathOnHostStats.isDirectory()) {
    for (const entry of await fsp.readdir(pathOnHost)) {
      const action = await _walk(fat, join(pathOnHost, entry), pathInNand, fn);
      if (action === WalkAction.Stop) return WalkAction.Stop;
    }
  }

  return WalkAction.Continue;
}

export async function walkHostPaths(
  fat: Fat32FileSystem,
  pathsOnHost: string[],
  rootDirInNand: string,
  fn: WalkCallback,
) {
  for (const pathOnHost of pathsOnHost) {
    switch (await _walk(fat, pathOnHost, rootDirInNand, fn)) {
      case WalkAction.Continue:
      case WalkAction.Skip:
        continue;
      case WalkAction.Stop:
        break;
    }
  }
}

export interface CopyPath {
  pathOnHostStats: fs.Stats;
  pathOnHost: string;
  pathInNand: string;
}
export interface PreCopyCheck {
  totalBytes: number;
  totalFiles: number;
  totalDirectories: number;
  copyPaths: CopyPath[];
}

export async function preCopyCheck(
  fat: Fat32FileSystem,
  hostPaths: string[],
  rootInNand: string,
): Promise<PreCopyCheck> {
  let totalBytes = 0;
  let totalFiles = 0;
  let totalDirectories = 0;
  const copyPaths: CopyPath[] = [];

  await walkHostPaths(fat, hostPaths, rootInNand, async ({ pathOnHostStats, pathOnHost, pathInNand }) => {
    if (pathOnHostStats.isFile()) {
      totalBytes += pathOnHostStats.size;
      totalFiles++;
    } else if (pathOnHostStats.isDirectory()) {
      totalDirectories++;
    }

    copyPaths.push({ pathOnHost, pathOnHostStats, pathInNand });

    return WalkAction.Continue;
  });

  return { totalBytes, totalFiles, totalDirectories, copyPaths };
}

export async function checkExistsRecursively(
  fat: Fat32FileSystem,
  hostPaths: string[],
  rootInNand: string,
): Promise<boolean> {
  let conflictExists = false;

  await walkHostPaths(fat, hostPaths, rootInNand, async ({ pathInNandEntry, pathOnHostStats }) => {
    if (!pathInNandEntry) return WalkAction.Continue;

    // if there's a file and an entry of any kind, that's a conflict
    if (pathOnHostStats.isFile() && pathInNandEntry) {
      conflictExists = true;
      return WalkAction.Stop;
    }

    // conflict if exists in the nand and it's not a directory
    if (pathOnHostStats.isDirectory() && pathInNandEntry.type !== 'd') {
      conflictExists = true;
      return WalkAction.Stop;
    }

    return WalkAction.Continue;
  });

  return conflictExists;
}
