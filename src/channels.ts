export interface ProdKeys {
  location: string;
  data: string;
}

export interface Partition {
  id: string;
  name: string;
  mountable: boolean;
  size: number;
  sizeHuman: string;
  free?: number;
  freeHuman?: string;
}

// TODO: do we need the enum, or can we just make them all generic?
export enum NandError {
  None,
  Generic,
  NoProdKeys,
  InvalidProdKeys,
  InvalidPartitionTable,
  NoNandOpened,
  NoPartitionMounted,
  Readonly,
  AlreadyExists,
}

export type NandResult<T = void> =
  | (T extends void ? { error: NandError.None } : { error: NandError.None; data: T })
  | { error: NandError.Generic; description: string }
  | { error: Exclude<NandError, NandError.None | NandError.Generic> };

export const NXKitBridgeKey = 'nxkit';
export type NXKitBridgeKeyType = typeof NXKitBridgeKey;
export interface NXKitBridge {
  isWindows: boolean;
  isLinux: boolean;
  isOsx: boolean;
}
