/// <reference types="../../src/typings.d.ts" />

import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const outputPath = path.join(__dirname, 'titles.json');

async function fetchGameTitles(): Promise<void> {
  const url = new URL('/Title/ApiJson/', 'https://tinfoil.media');
  url.searchParams.set(
    'region',
    'ar,at,au,be,bg,br,ca,ch,cl,cn,co,cy,cz,de,dk,ee,es,fi,fr,gb,gr,hk,hr,hu,ar,at,au,be,bg,br,ca,ch,cl,cn,co,cy,cz,de,dk,ee,es,fi,fr,gb,gr,hk,hr,hu,',
  );
  url.searchParams.set('_', Date.now().toString());
  url.searchParams.set('rating_content', '');
  url.searchParams.set('rating', '');
  url.searchParams.set('category', '');
  url.searchParams.set('language', '');

  const response = await fetch(url, {
    method: 'GET',
    headers: {
      Accept: 'application/json, text/javascript, */*; q=0.01',
      'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/115.0',
      Origin: 'https://tinfoil.io',
      Referer: 'https://tinfoil.io/',
    },
  });

  if (!response.ok) {
    throw new Error(`Unexpected response ${response.status}: ${await response.text()}`);
  }

  const { data }: TinfoilResponse = await response.json();
  await fs.writeFile(outputPath, JSON.stringify(data, null, 2));
}

fetchGameTitles();
