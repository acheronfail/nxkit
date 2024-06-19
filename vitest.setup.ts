import '@testing-library/jest-dom/vitest';
import { vi } from 'vitest';
import { NXKitBridgeKey, ExposedPreloadAPIs } from './src/channels';

declare global {
  interface Window {
    [NXKitBridgeKey]: ExposedPreloadAPIs;
  }
}

window[NXKitBridgeKey] = {
  isWindows: false,
  isLinux: false,
  isOsx: false,
  keysFind: vi.fn(),
  keysSearchPaths: vi.fn(),
  nandClose: vi.fn(),
  nandCopyFile: vi.fn(),
  nandFormatPartition: vi.fn(),
  nandMount: vi.fn(),
  nandOpen: vi.fn(),
  nandReaddir: vi.fn(),
  openLink: vi.fn(),
  payloadsFind: vi.fn(),
  payloadsOpenDirectory: vi.fn(),
  payloadsReadFile: vi.fn(),
  runTegraRcmSmash: vi.fn(),
};