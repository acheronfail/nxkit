import os from 'node:os';
import path from 'node:path';

export type Resources = ReturnType<typeof getResources>;

export function getResources(isPackaged: boolean) {
  const dotSwitch = path.join(os.homedir(), '.switch');
  const resources = {
    payloadDirectory: path.resolve(dotSwitch, 'payloads'),
    prodKeysSearchPaths: [path.resolve(dotSwitch, 'prod.keys'), path.resolve(process.cwd(), 'prod.keys')],
    tegraRcmSmash: isPackaged
      ? path.resolve(process.resourcesPath, 'TegraRcmSmash.exe')
      : path.resolve('vendor', 'TegraRcmSmash', 'TegraRcmSmash.exe'),
  };

  if (!isPackaged) {
    resources.prodKeysSearchPaths.unshift(path.resolve(process.cwd(), '.data', 'prod.keys'));
  }

  return resources;
}
