interface NXKitBridge {
  isWindows: boolean;
}

interface NXKitTegraRcmSmashResult {
  success: boolean;
  stdout: string;
  stderr: string;
}

interface NXKitTegraRcmSmash {
  run: (payloadFilePath: string) => Promise<NXKitTegraRcmSmashResult>;
}

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
