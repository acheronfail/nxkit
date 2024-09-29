import '@testing-library/jest-dom/vitest';
import { vi } from 'vitest';
import { NXKitBridgeKey } from './src/channels';

window[NXKitBridgeKey] = {
  isWindows: false,
  isLinux: false,
  isOsx: false,

  call: vi.fn().mockResolvedValue(undefined),
  progressSubscribe: vi.fn(),
};
