import fs from 'node:fs/promises';
import { getResources } from '../resources';
import { app } from 'electron';
import { ProdKeys } from '../channels';
import { Keys } from './keys.types';

export async function resolveKeys(keysFromUser?: ProdKeys): Promise<Keys | null> {
  if (keysFromUser) {
    const parsed = Keys.parseKeys(keysFromUser.location, keysFromUser.data);
    if (parsed) {
      return parsed;
    }
  }

  return findProdKeys();
}

export async function findProdKeys(): Promise<Keys | null> {
  for (const filePath of getResources(app.isPackaged).prodKeysSearchPaths) {
    try {
      const text = await fs.readFile(filePath, 'utf-8');
      return Keys.parseKeys(filePath, text);
    } catch (err) {
      console.log(`Failed to find keys at ${filePath}: ${String(err)}`);
    }
  }

  return null;
}
