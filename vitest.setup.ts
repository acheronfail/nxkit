import '@testing-library/jest-dom/vitest';
import { vi } from 'vitest';
import { NXKitBridgeKey } from './src/channels';

window[NXKitBridgeKey] = {
  isWindows: false,
  isLinux: false,
  isOsx: false,

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  call: vi.fn<any>().mockResolvedValue(undefined),
};
