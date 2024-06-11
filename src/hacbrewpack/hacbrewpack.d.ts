/* eslint-disable @typescript-eslint/no-explicit-any */

// Community maintained types don't export FS properly, so we have to redefine them...
interface FSModule {
    ignorePermissions: boolean;
    trackingDelegate: {
        onOpenFile(path: string, trackingFlags: number): unknown;
        onCloseFile(path: string): unknown;
        onSeekFile(path: string, position: number, whence: number): unknown;
        onReadFile(path: string, bytesRead: number): unknown;
        onWriteToFile(path: string, bytesWritten: number): unknown;
        onMakeDirectory(path: string, mode: number): unknown;
        onMakeSymlink(oldpath: string, newpath: string): unknown;
        willMovePath(old_path: string, new_path: string): unknown;
        onMovePath(old_path: string, new_path: string): unknown;
        willDeletePath(path: string): unknown;
        onDeletePath(path: string): unknown;
    };
    tracking: any;
    genericErrors: Record<number, ErrnoError>;

    //
    // paths
    //
    lookupPath(
        path: string,
        opts: Partial<{
            follow_mount: boolean;
            /**
             * by default, lookupPath will not follow a symlink if it is the final path component.
             * setting opts.follow = true will override this behavior.
             */
            follow: boolean;
            recurse_count: number;
            parent: boolean;
        }>,
    ): Lookup;
    getPath(node: FSNode): string;
    analyzePath(path: string, dontResolveLastLink?: boolean): Analyze;

    //
    // nodes
    //
    isFile(mode: number): boolean;
    isDir(mode: number): boolean;
    isLink(mode: number): boolean;
    isChrdev(mode: number): boolean;
    isBlkdev(mode: number): boolean;
    isFIFO(mode: number): boolean;
    isSocket(mode: number): boolean;

    //
    // devices
    //
    major(dev: number): number;
    minor(dev: number): number;
    makedev(ma: number, mi: number): number;
    registerDevice(dev: number, ops: Partial<StreamOps>): void;
    getDevice(dev: number): { stream_ops: StreamOps };

    //
    // core
    //
    getMounts(mount: Mount): Mount[];
    syncfs(populate: boolean, callback: (e: any) => any): void;
    syncfs(callback: (e: any) => any, populate?: boolean): void;
    mount(type: Emscripten.FileSystemType, opts: any, mountpoint: string): any;
    unmount(mountpoint: string): void;

    mkdir(path: string, mode?: number): FSNode;
    mkdev(path: string, mode?: number, dev?: number): FSNode;
    symlink(oldpath: string, newpath: string): FSNode;
    rename(old_path: string, new_path: string): void;
    rmdir(path: string): void;
    readdir(path: string): string[];
    unlink(path: string): void;
    readlink(path: string): string;
    stat(path: string, dontFollow?: boolean): Stats;
    lstat(path: string): Stats;
    chmod(path: string, mode: number, dontFollow?: boolean): void;
    lchmod(path: string, mode: number): void;
    fchmod(fd: number, mode: number): void;
    chown(path: string, uid: number, gid: number, dontFollow?: boolean): void;
    lchown(path: string, uid: number, gid: number): void;
    fchown(fd: number, uid: number, gid: number): void;
    truncate(path: string, len: number): void;
    ftruncate(fd: number, len: number): void;
    utime(path: string, atime: number, mtime: number): void;
    open(path: string, flags: string, mode?: number, fd_start?: number, fd_end?: number): FSStream;
    close(stream: FSStream): void;
    llseek(stream: FSStream, offset: number, whence: number): number;
    read(stream: FSStream, buffer: ArrayBufferView, offset: number, length: number, position?: number): number;
    write(
        stream: FSStream,
        buffer: ArrayBufferView,
        offset: number,
        length: number,
        position?: number,
        canOwn?: boolean,
    ): number;
    allocate(stream: FSStream, offset: number, length: number): void;
    mmap(
        stream: FSStream,
        buffer: ArrayBufferView,
        offset: number,
        length: number,
        position: number,
        prot: number,
        flags: number,
    ): {
        allocated: boolean;
        ptr: number;
    };
    ioctl(stream: FSStream, cmd: any, arg: any): any;
    readFile(path: string, opts: { encoding: "binary"; flags?: string | undefined }): Uint8Array;
    readFile(path: string, opts: { encoding: "utf8"; flags?: string | undefined }): string;
    readFile(path: string, opts?: { flags?: string | undefined }): Uint8Array;
    writeFile(path: string, data: string | ArrayBufferView, opts?: { flags?: string | undefined }): void;

    //
    // module-level FS code
    //
    cwd(): string;
    chdir(path: string): void;
    init(
        input: null | (() => number | null),
        output: null | ((c: number) => any),
        error: null | ((c: number) => any),
    ): void;

    createLazyFile(
        parent: string | FSNode,
        name: string,
        url: string,
        canRead: boolean,
        canWrite: boolean,
    ): FSNode;
    createPreloadedFile(
        parent: string | FSNode,
        name: string,
        url: string,
        canRead: boolean,
        canWrite: boolean,
        onload?: () => void,
        onerror?: () => void,
        dontCreateFile?: boolean,
        canOwn?: boolean,
    ): void;
    createDataFile(
        parent: string | FSNode,
        name: string,
        data: ArrayBufferView,
        canRead: boolean,
        canWrite: boolean,
        canOwn: boolean,
    ): FSNode;
}

// Community maintained types don't have everything...
interface ExtendedEmscriptenModule extends EmscriptenModule {
  onExit(code: number): void;
  callMain(argv: string[]): number;

  FS: FSModule;
}

const loadHacBrewPack: EmscriptenModuleFactory<ExtendedEmscriptenModule>;
export default loadHacBrewPack;
