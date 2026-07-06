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

export type NandResult<T = void> = { type: 'success'; data: T } | { type: 'failure'; error: string };

export const NXKitBridgeKey = 'nxkit';
export type NXKitBridgeKeyType = typeof NXKitBridgeKey;
export interface NXKitBridge {
  isWindows: boolean;
  isLinux: boolean;
  isOsx: boolean;
}
