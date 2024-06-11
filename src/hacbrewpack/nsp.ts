// eslint-disable-next-line import/no-unresolved
import NSPWorker from './nsp.worker?worker';

import { NACP, VideoCapture } from '@tootallnate/nacp';
import { fetchBinary } from '../browser/fetch';
import { BuildNSPArgs, HacBrewPackResult, StartupUserAccount, Screenshots, LogoType, HacBrewPackArgs } from './types';

import defaultStartupMovie from '../public/StartupMovie.gif';
import defaultLogo from '../public/NintendoLogo.png';
import defaultImage from '../public/DefaultNSPImage.png';
import exefsMain from '../public/exefs/main';
import exefsMainNpdm from '../public/exefs/main.npdm';

export async function buildNsp(args: BuildNSPArgs): Promise<HacBrewPackResult> {
  const nacp = new NACP();
  nacp.id = args.id;
  nacp.title = args.title;
  nacp.author = args.author;
  nacp.version = args.version ?? '1.0.0';
  nacp.startupUserAccount = args.startupUserAccount ?? StartupUserAccount.Disabled;
  nacp.screenshot = args.screenshot ?? Screenshots.Enabled;
  nacp.videoCapture = args.videoCapture ?? VideoCapture.Disabled;
  nacp.logoType = args.logoType ?? LogoType.Nothing;
  nacp.logoHandling = 0;

  const worker = new NSPWorker();
  const result = new Promise<HacBrewPackResult>((resolve, reject) => {
    worker.onmessage = (event: MessageEvent<HacBrewPackResult>) => {
      resolve(event.data);
      worker.terminate();
    };
    worker.onerror = (event) => {
      reject(event);
      worker.terminate();
    };
  });

  const message: HacBrewPackArgs = {
    argv: ['--nopatchnacplogo', '--titleid', args.id],
    controlNacp: new Uint8Array(nacp.buffer),
    keys: args.keys,
    fileName: args.fileName,
    image: args.image ?? (await fetchBinary(defaultImage)),
    logo: args.logo ?? (await fetchBinary(defaultLogo)),
    startupMovie: args.startupMovie ?? (await fetchBinary(defaultStartupMovie)),
    nextNroPath: args.nroPath,
    nextArgv: [args.nroPath, ...args.nroArgv].join(' '),
    main: await fetchBinary(exefsMain),
    mainNpdm: await fetchBinary(exefsMainNpdm),
  };

  worker.postMessage(message);

  return result;
}

export function generateRandomId() {
  return ['01', ...new Array(10).fill(0).map(() => Math.floor(Math.random() * 16).toString(16)), '0000'].join('');
}
