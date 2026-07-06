import fsp from 'node:fs/promises';

export async function getUniquePath(desiredPath: string): Promise<string> {
  let count = 0;
  let current = desiredPath;
  for (;;) {
    const exists = await fsp.stat(current).then(
      () => true,
      () => false,
    );

    if (!exists) {
      return current;
    }

    current = `${desiredPath}.${(++count).toString().padStart(2, '0')}`;
  }
}
