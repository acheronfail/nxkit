/// <reference types="vite/client" />

/*
 * Imported as URLs by vite
 */

declare module '*.exe' {
  const src: string;
  export default src;
}

declare module '*.nso' {
  const src: string;
  export default src;
}

declare module '*.npdm' {
  const src: string;
  export default src;
}

declare module '*?raw' {
  const text: string;
  export default text;
}

/*
 * Types for untyped packages
 */

declare module 'chs' {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const mod: any;
  export default mod;
}

declare module 'mbr' {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const mod: any;
  export default mod;
}

declare module 'gpt' {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const mod: any;
  export default mod;
}

/*
 * Helpers
 */

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type PromiseIpc = Record<string, (...args: any[]) => Promise<any>>;
type PromiseIpcHandler = (...args: unknown[]) => Promise<unknown>;

/*
 * Misc
 */

interface TinfoilResponse {
  data: TinfoilTitle[];
}

interface TinfoilTitle {
  id: string;
  name: string;
  icon: string;
  release_date: string;
  publisher: string;
  size: string;
  user_rating: number;
}
