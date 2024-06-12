declare module '*.exe' {
  const src: string;
  export default src;
}

declare module '*.gif' {
  const src: string;
  export default src;
}

declare module '*.png' {
  const src: string;
  export default src;
}

declare module '*/exefs/main' {
  const src: string;
  export default src;
}

declare module '*/exefs/main.npdm' {
  const src: string;
  export default src;
}

declare module '*?worker' {
  const workerConstructor: {
    new (options?: { name?: string }): Worker;
  };
  export default workerConstructor;
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