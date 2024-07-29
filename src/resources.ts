import os from 'node:os';
import path from 'node:path';

/**
 * Returns paths used by the application, which can change if the app is running
 * in development mode or not.
 */
export function getPaths(isPackaged: boolean) {
  const dotSwitch = path.join(os.homedir(), '.switch');
  const paths = {
    payloadDirectory: path.resolve(dotSwitch, 'payloads'),
    prodKeysSearchPaths: [path.resolve(dotSwitch, 'prod.keys'), path.resolve(process.cwd(), 'prod.keys')],
  };

  if (!isPackaged) {
    paths.prodKeysSearchPaths.unshift(path.resolve(process.cwd(), '.data', 'prod.keys'));
  }

  return paths;
}
export type Paths = ReturnType<typeof getPaths>;

/**
 * Returns "resources"; bundled in files that must be copied when packaging the
 * application.
 */
export function getResources(isPackaged: boolean) {
  const resources = {
    tegraRcmSmash: isPackaged
      ? path.resolve(process.resourcesPath, 'TegraRcmSmash.exe')
      : path.resolve('vendor', 'TegraRcmSmash', 'TegraRcmSmash.exe'),
  } satisfies Record<string, string>;

  return resources;
}
export type Resources = ReturnType<typeof getResources>;
