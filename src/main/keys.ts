import fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { Xtsn } from '../nand/xtsn';

const PROD_KEYS_SEARCH_PATHS: string[] = [
  path.join(os.homedir(), '.switch', 'prod.keys'),
  path.join(process.cwd(), 'prod.keys'),
];

export async function findProdKeys(): Promise<Keys | null> {
  for (const filePath of PROD_KEYS_SEARCH_PATHS) {
    const text = await fs.readFile(filePath, 'utf-8');
    try {
      return Keys.parseKeys(text);
    } catch (err) {
      console.log(`Failed to parse keys at ${filePath}: ${String(err)}`);
    }
  }

  return null;
}

export class Keys {
  static parseKeys(text: string): Keys | null {
    // TODO: validation of keys (zod?)
    return new Keys(
      Object.fromEntries(
        text
          .trim()
          .split('\n')
          .map((line) => line.split('=').map((s) => s.trim()))
      )
    );
  }

  constructor(public readonly raw: RawKeys) {}

  getBisKey(id: 0 | 1 | 2 | 3): { crypto: Buffer; tweak: Buffer } {
    const text = this.raw[`bis_key_0${id}`];
    return {
      crypto: Buffer.from(text.substring(0, 32), 'hex'),
      tweak: Buffer.from(text.substring(32), 'hex'),
    };
  }

  getXtsn(id: 0 | 1 | 2 | 3): Xtsn {
    const { crypto, tweak } = this.getBisKey(id);
    return new Xtsn(crypto, tweak);
  }

  toString(): string {
    return Object.entries(this.raw)
      .map((entry) => entry.join(' = '))
      .join('\n');
  }
}

export interface RawKeys {
  aes_kek_generation_source: string;
  aes_key_generation_source: string;
  bis_kek_source: string;
  bis_key_00: string;
  bis_key_01: string;
  bis_key_02: string;
  bis_key_03: string;
  bis_key_source_00: string;
  bis_key_source_01: string;
  bis_key_source_02: string;
  device_key: string;
  device_key_4x: string;
  eticket_rsa_kek: string;
  eticket_rsa_kek_source: string;
  eticket_rsa_kekek_source: string;
  eticket_rsa_keypair: string;
  header_kek_source: string;
  header_key: string;
  header_key_source: string;
  key_area_key_application_00: string;
  key_area_key_application_01: string;
  key_area_key_application_02: string;
  key_area_key_application_03: string;
  key_area_key_application_04: string;
  key_area_key_application_05: string;
  key_area_key_application_06: string;
  key_area_key_application_07: string;
  key_area_key_application_08: string;
  key_area_key_application_09: string;
  key_area_key_application_0a: string;
  key_area_key_application_0b: string;
  key_area_key_application_0c: string;
  key_area_key_application_0d: string;
  key_area_key_application_0e: string;
  key_area_key_application_0f: string;
  key_area_key_application_10: string;
  key_area_key_application_11: string;
  key_area_key_application_source: string;
  key_area_key_ocean_00: string;
  key_area_key_ocean_01: string;
  key_area_key_ocean_02: string;
  key_area_key_ocean_03: string;
  key_area_key_ocean_04: string;
  key_area_key_ocean_05: string;
  key_area_key_ocean_06: string;
  key_area_key_ocean_07: string;
  key_area_key_ocean_08: string;
  key_area_key_ocean_09: string;
  key_area_key_ocean_0a: string;
  key_area_key_ocean_0b: string;
  key_area_key_ocean_0c: string;
  key_area_key_ocean_0d: string;
  key_area_key_ocean_0e: string;
  key_area_key_ocean_0f: string;
  key_area_key_ocean_10: string;
  key_area_key_ocean_11: string;
  key_area_key_ocean_source: string;
  key_area_key_system_00: string;
  key_area_key_system_01: string;
  key_area_key_system_02: string;
  key_area_key_system_03: string;
  key_area_key_system_04: string;
  key_area_key_system_05: string;
  key_area_key_system_06: string;
  key_area_key_system_07: string;
  key_area_key_system_08: string;
  key_area_key_system_09: string;
  key_area_key_system_0a: string;
  key_area_key_system_0b: string;
  key_area_key_system_0c: string;
  key_area_key_system_0d: string;
  key_area_key_system_0e: string;
  key_area_key_system_0f: string;
  key_area_key_system_10: string;
  key_area_key_system_11: string;
  key_area_key_system_source: string;
  keyblob_00: string;
  keyblob_01: string;
  keyblob_02: string;
  keyblob_03: string;
  keyblob_04: string;
  keyblob_05: string;
  keyblob_key_00: string;
  keyblob_key_01: string;
  keyblob_key_02: string;
  keyblob_key_03: string;
  keyblob_key_04: string;
  keyblob_key_05: string;
  keyblob_key_source_00: string;
  keyblob_key_source_01: string;
  keyblob_key_source_02: string;
  keyblob_key_source_03: string;
  keyblob_key_source_04: string;
  keyblob_key_source_05: string;
  keyblob_mac_key_00: string;
  keyblob_mac_key_01: string;
  keyblob_mac_key_02: string;
  keyblob_mac_key_03: string;
  keyblob_mac_key_04: string;
  keyblob_mac_key_05: string;
  keyblob_mac_key_source: string;
  mariko_master_kek_source_05: string;
  mariko_master_kek_source_06: string;
  mariko_master_kek_source_07: string;
  mariko_master_kek_source_08: string;
  mariko_master_kek_source_09: string;
  mariko_master_kek_source_0a: string;
  mariko_master_kek_source_0b: string;
  mariko_master_kek_source_0c: string;
  mariko_master_kek_source_0d: string;
  mariko_master_kek_source_0e: string;
  mariko_master_kek_source_0f: string;
  mariko_master_kek_source_10: string;
  mariko_master_kek_source_11: string;
  master_kek_00: string;
  master_kek_01: string;
  master_kek_02: string;
  master_kek_03: string;
  master_kek_04: string;
  master_kek_05: string;
  master_kek_08: string;
  master_kek_09: string;
  master_kek_0a: string;
  master_kek_0b: string;
  master_kek_0c: string;
  master_kek_0d: string;
  master_kek_0e: string;
  master_kek_0f: string;
  master_kek_10: string;
  master_kek_11: string;
  master_kek_source_06: string;
  master_kek_source_07: string;
  master_kek_source_08: string;
  master_kek_source_09: string;
  master_kek_source_0a: string;
  master_kek_source_0b: string;
  master_kek_source_0c: string;
  master_kek_source_0d: string;
  master_kek_source_0e: string;
  master_kek_source_0f: string;
  master_kek_source_10: string;
  master_kek_source_11: string;
  master_key_00: string;
  master_key_01: string;
  master_key_02: string;
  master_key_03: string;
  master_key_04: string;
  master_key_05: string;
  master_key_06: string;
  master_key_07: string;
  master_key_08: string;
  master_key_09: string;
  master_key_0a: string;
  master_key_0b: string;
  master_key_0c: string;
  master_key_0d: string;
  master_key_0e: string;
  master_key_0f: string;
  master_key_10: string;
  master_key_11: string;
  master_key_source: string;
  package1_key_00: string;
  package1_key_01: string;
  package1_key_02: string;
  package1_key_03: string;
  package1_key_04: string;
  package1_key_05: string;
  package2_key_00: string;
  package2_key_01: string;
  package2_key_02: string;
  package2_key_03: string;
  package2_key_04: string;
  package2_key_05: string;
  package2_key_06: string;
  package2_key_07: string;
  package2_key_08: string;
  package2_key_09: string;
  package2_key_0a: string;
  package2_key_0b: string;
  package2_key_0c: string;
  package2_key_0d: string;
  package2_key_0e: string;
  package2_key_0f: string;
  package2_key_10: string;
  package2_key_11: string;
  package2_key_source: string;
  per_console_key_source: string;
  retail_specific_aes_key_source: string;
  save_mac_kek_source: string;
  save_mac_key: string;
  save_mac_key_source: string;
  save_mac_sd_card_kek_source: string;
  save_mac_sd_card_key_source: string;
  sd_card_custom_storage_key_source: string;
  sd_card_kek_source: string;
  sd_card_nca_key_source: string;
  sd_card_save_key_source: string;
  sd_seed: string;
  secure_boot_key: string;
  ssl_rsa_kek: string;
  ssl_rsa_kek_source: string;
  ssl_rsa_kekek_source: string;
  ssl_rsa_key: string;
  titlekek_00: string;
  titlekek_01: string;
  titlekek_02: string;
  titlekek_03: string;
  titlekek_04: string;
  titlekek_05: string;
  titlekek_06: string;
  titlekek_07: string;
  titlekek_08: string;
  titlekek_09: string;
  titlekek_0a: string;
  titlekek_0b: string;
  titlekek_0c: string;
  titlekek_0d: string;
  titlekek_0e: string;
  titlekek_0f: string;
  titlekek_10: string;
  titlekek_11: string;
  titlekek_source: string;
  tsec_key: string;
  tsec_root_key_02: string;
}
