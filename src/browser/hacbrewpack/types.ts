import { VideoCapture } from '@tootallnate/nacp';

export enum StartupUserAccount {
  Disabled,
  Enabled,
}

export enum Screenshots {
  Disabled,
  Enabled,
}

export enum LogoType {
  LicensedBy,
  DistributedBy,
  Nothing,
}

export interface BuildNSPArgs {
  /**
   * Title id
   */
  id: string;
  /**
   * Also known as the "publisher", shown in NSP info menu
   */
  author: string;
  /**
   * File name for the generated NSP file.
   * If unset, defaults to the id.
   */
  fileName?: string;
  /**
   * Image of the NSP title, this is what you see on the homescreen.
   */
  image?: Data;
  /**
   * prod.keys
   */
  keys: Data;
  /**
   * Logo shown when the NSP title is clicked.
   */
  logo?: Data;
  logoType?: LogoType;
  /**
   * Arguments to the NRO file that will be run.
   */
  nroArgv: string[];
  /**
   * Path to the NRO file that will be run.
   * This is the path that's on the Switch's SD card.
   */
  nroPath: string;
  /**
   * Configure how screenshots should work in the game.
   */
  screenshot?: Screenshots;
  /**
   * Optional start up animation to show while the NSP opens.
   */
  startupMovie?: Data;
  /**
   * Whether or not the user profile picker should appear when opening the title.
   */
  startupUserAccount?: StartupUserAccount;
  /**
   * The name of the NSP as shown on the home screen.
   */
  title: string;
  /**
   * The version of the NSP as shown in the home screen info.
   */
  version?: string;
  /**
   * Whether or not video capture support is enabled for the game.
   * Defaults to disabled, since it requires extra memory to be allocated.
   */
  videoCapture?: VideoCapture;
}

export type Data = string | ArrayBufferView;
export interface HacBrewPackArgs {
  /**
   * arguments to `hacbrewpack`
   */
  argv: string[];
  /**
   * prod.keys
   */
  keys: Data;
  /**
   * metadata about NSP title
   */
  controlNacp: Data;
  /**
   * NSP image (what's shown on title screen)
   */
  image: Data;
  /**
   * logo for startup animation
   */
  logo: Data;
  /**
   * gif for startup animation
   */
  startupMovie: Data;
  /**
   * path to the NRO file to run
   */
  nextArgv: Data;
  /**
   * args for the NRO file (remember to include arg0 as `nextNroPath` itself)
   *  `sdmc:${nroPath}`
   * or (e.g., for retroarch forwarding):
   *  `sdmc:${retroarchCoreNro} "sdmc:${romPath}"`
   */
  nextNroPath: Data;

  /**
   * Set the filename of the generated NSP
   */
  fileName?: string;

  /**
   * Files required to build the Nintendo Switch executable.
   * Compiled by vendor/Forwarder-Mod
   */
  main: Data;
  mainNpdm: Data;
}

export interface HacBrewPackResult {
  stdout: string;
  stderr: string;
  exitCode: number;
  nsp?: File;
}
