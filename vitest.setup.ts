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
  keysSearchPaths: vi.fn().mockResolvedValue(['path1', 'path2']),
  nandClose: vi.fn(),
  nandCopyFileOut: vi.fn(),
  nandCopyFilesIn: vi.fn(),
  nandFormatPartition: vi.fn(),
  nandMount: vi.fn(),
  nandOpen: vi.fn(),
  nandReaddir: vi.fn(),
  nandDeleteEntry: vi.fn(),
  nandMoveEntry: vi.fn(),
  nandCheckExists: vi.fn(),
  openLink: vi.fn(),
  pathDirname: vi.fn(),
  pathJoin: vi.fn(),
  payloadsFind: vi.fn(),
  payloadsOpenDirectory: vi.fn(),
  payloadsReadFile: vi.fn(),
  payloadsCopyIn: vi.fn(),
  runTegraRcmSmash: vi.fn(),
};
