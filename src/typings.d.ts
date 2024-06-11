declare module '*.keys' {
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
