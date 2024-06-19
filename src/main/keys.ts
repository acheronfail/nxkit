import fs from 'node:fs/promises';
import { Xtsn } from '../nand/xtsn';
import { RawKeys, RawKeysSchema } from './keys.types';
import { getResources } from '../resources';
import { app } from 'electron';
import { ProdKeys } from '../channels';

export type BisKeyId = 0 | 1 | 2 | 3;

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

export class Keys {
  static parseKeys(path: string, text: string): Keys | null {
    return new Keys(
      path,
      RawKeysSchema.parse(
        Object.fromEntries(
          text
            .trim()
            .split('\n')
            .map((line) => line.split('=').map((s) => s.trim())),
        ),
      ),
    );
  }

  constructor(
    public readonly path: string,
    public readonly raw: RawKeys,
  ) {}

  getBisKey(id: BisKeyId): { crypto: Buffer; tweak: Buffer } {
    const text = this.raw[`bis_key_0${id}`];
    return {
      crypto: Buffer.from(text.substring(0, 32), 'hex'),
      tweak: Buffer.from(text.substring(32), 'hex'),
    };
  }

  getXtsn(id: BisKeyId): Xtsn {
    const { crypto, tweak } = this.getBisKey(id);
    return new Xtsn(crypto, tweak);
  }

  toString(): string {
    return Object.entries(this.raw)
      .map((entry) => entry.join(' = '))
      .join('\n');
  }
}
