import path from 'node:path';

export type Resources = ReturnType<typeof getResources>;

export function getResources(isPackaged: boolean) {
  return {
    tegraRcmSmash: isPackaged
      ? path.join(process.resourcesPath, 'TegraRcmSmash.exe')
      : path.join('vendor', 'TegraRcmSmash', 'TegraRcmSmash.exe'),
  };
}
