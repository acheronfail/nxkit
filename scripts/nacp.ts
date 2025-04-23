import fsp from 'node:fs/promises';
import { NACP } from '@tootallnate/nacp';

const nacp = new NACP();
nacp.title = 'Test Title';

await fsp.writeFile('control.node.nacp', Buffer.from(nacp.buffer));
