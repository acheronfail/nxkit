/// <reference types="vite/client" />

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

/*
 * Types for untyped packages
 */

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
