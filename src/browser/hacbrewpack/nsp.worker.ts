import { HacBrewPackArgs, HacBrewPackResult } from './types';
import wasmHacBrewPack from './hacbrewpack';

onmessage = (event: MessageEvent<HacBrewPackArgs>) => {
  runHacBrewPack(event.data)
    .then((result) => postMessage(result))
    .catch((err) => {
      console.error(`Failed to build NSP: `, err);
      throw err;
    });
};

async function runHacBrewPack(args: HacBrewPackArgs): Promise<HacBrewPackResult> {
  const stdout: string[] = [];
  const stderr: string[] = [];
  const { FS, callMain } = await wasmHacBrewPack({
    // wasm will immediately call main when we initialise it if we don't disable it
    // we don't want it to run immediately, because we have some setup to do
    noInitialRun: true,
    print: (line) => stdout.push(line),
    printErr: (line) => stderr.push(line),
  });

  FS.writeFile('/keys.dat', args.keys);

  FS.mkdir('/control');
  FS.writeFile('/control/control.nacp', args.controlNacp);
  FS.writeFile('/control/icon_AmericanEnglish.dat', args.image);

  FS.mkdir('/exefs');
  FS.writeFile('/exefs/main', args.main);
  FS.writeFile('/exefs/main.npdm', args.mainNpdm);

  FS.mkdir('/logo');
  FS.writeFile('/logo/NintendoLogo.png', args.logo);
  FS.writeFile('/logo/StartupMovie.gif', args.startupMovie);

  FS.mkdir('/romfs');
  FS.writeFile('/romfs/nextArgv', args.nextArgv);
  FS.writeFile('/romfs/nextNroPath', args.nextNroPath);

  const exitCode = callMain(args.argv);
  if (exitCode !== 0) {
    return {
      stdout: stdout.join('\n'),
      stderr: stderr.join('\n'),
      exitCode,
    };
  }

  const NSP_OUT_DIRECTORY = '/hacbrewpack_nsp';
  const [nspFilename] = FS.readdir(NSP_OUT_DIRECTORY).filter((n) => n.endsWith('.nsp'));
  const data = FS.readFile(`${NSP_OUT_DIRECTORY}/${nspFilename}`);

  return {
    exitCode,
    stdout: stdout.join('\n'),
    stderr: stderr.join('\n'),
    nsp: new File([data], args.fileName ?? nspFilename),
  };
}
