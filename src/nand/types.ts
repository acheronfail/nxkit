import { open } from 'fs/promises';

export type FileHandle = Awaited<ReturnType<typeof open>>;
