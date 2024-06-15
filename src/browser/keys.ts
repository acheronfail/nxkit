import { readFile } from "./file";

export async function keysFromUser(): Promise<Uint8Array | null> {
  const { files } = document.querySelector<HTMLInputElement>('#keys');
  if (files[0]) return readFile(files[0]);
  return null;
}

export async function getKeys(): Promise<string | Uint8Array> {
  let keys: string | Uint8Array = await keysFromUser();
  if (!keys) keys = await window.nxkit.findProdKeys();
  if (!keys) {
    throw new Error('prod.keys are required to create an NSP!');
  }

  return keys;
}