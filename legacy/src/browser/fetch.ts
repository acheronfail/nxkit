export async function fetchBinary(url: string): Promise<Uint8Array> {
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(`Failed to find "${url}": ${await response.text()}`);
  }

  return new Uint8Array(await response.arrayBuffer());
}
