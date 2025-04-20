import fsp from 'node:fs/promises';
import { getResources } from '../src/resources';

await fsp.writeFile('resources.json', JSON.stringify(Object.values(getResources(false)), null, 2));
