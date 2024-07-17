import fsp from 'node:fs/promises';
import path from 'node:path';

export type SplitMergeResult = { type: 'success'; outputPath: string } | { type: 'failure'; description: string };

export async function split(
  filePath: string,
  asArchive: boolean,
  inPlace: boolean,
  splitSize = asArchive ? 0xffff0000 : 0x80000000,
): Promise<SplitMergeResult> {
  const inner = async () => {
    const fileHandle = await fsp.open(filePath, inPlace ? 'r+' : 'r');
    const fileStats = await fileHandle.stat();

    // move from the back to the front
    let offset = fileStats.size - (fileStats.size % splitSize);
    let count = Math.floor(fileStats.size / splitSize);

    const getSplitPath = () => {
      const countSuffix = `${count.toString().padStart(2, '0')}`;

      if (asArchive) {
        const parsed = path.parse(filePath);
        if (parsed.ext) {
          return path.join(parsed.dir, `${parsed.name}_split${parsed.ext}`, countSuffix);
        } else {
          return path.join(parsed.dir, `${parsed.base}_split`, countSuffix);
        }
      }

      return `${filePath}.${countSuffix}`;
    };

    // if we're not copying, the original file becomes the first part
    const limit = inPlace ? 0 : -1;
    while (offset > limit) {
      const splitPath = getSplitPath();
      console.log(`Creating ${splitPath}...`);
      if (asArchive) {
        await fsp.mkdir(path.dirname(splitPath), { recursive: true });
      }

      const start = offset;
      const end = Math.min(fileStats.size, offset + splitSize - 1);
      const splitHandle = await fsp.open(splitPath, 'w');
      const src = fileHandle.createReadStream({ start, end, autoClose: false });
      const dst = splitHandle.createWriteStream();
      await new Promise((resolve, reject) => {
        dst.on('finish', resolve);
        dst.on('error', reject);
        src.on('error', reject);
        src.pipe(dst);
      });

      await splitHandle.close();
      if (inPlace) {
        await fileHandle.truncate(offset);
      }

      offset -= splitSize;
      count--;
    }

    await fileHandle.close();

    // move remaining part into destination
    if (inPlace) {
      const splitPath = getSplitPath();
      console.log(`Creating ${splitPath}...`);
      await fsp.rename(filePath, splitPath);
    }

    return path.dirname(getSplitPath());
  };

  try {
    return { type: 'success', outputPath: await inner() };
  } catch (err) {
    console.error(err);
    const description = `Failed to split file: ${err instanceof Error ? err.message : String(err)}`;
    return { type: 'failure', description };
  }
}

export async function merge(filePath: string, inPlace: boolean): Promise<SplitMergeResult> {
  const inner = async (): Promise<SplitMergeResult> => {
    const firstFileName = path.basename(filePath);
    if (!/^00$|.*\.00$/.test(firstFileName)) {
      return {
        type: 'failure',
        description: "The selected file doesn't appear to be the first part of a split! Select 00 or myfile.00",
      };
    }

    const isArchive = firstFileName === '00';
    const dir = path.dirname(filePath);
    const entries = await fsp.readdir(dir);
    const splitFiles = entries
      .filter((name) => (isArchive ? /^\d\d$/ : /\.\d\d$/).test(name))
      .map((name) => path.join(dir, name))
      .sort();

    // figure out destination path
    let mergedFileName: string;
    if (isArchive) {
      const parsed = path.parse(dir);
      if (parsed.ext) {
        mergedFileName = `${parsed.name}_merged${parsed.ext}`;
      } else {
        mergedFileName = `${parsed.base}_merged`;
      }
    } else {
      mergedFileName = path.basename(filePath, '.00');
    }

    const mergedFilePath = path.join(isArchive ? path.dirname(dir) : dir, mergedFileName);
    const mergedHandle = await fsp.open(mergedFilePath, 'w');
    const dst = mergedHandle.createWriteStream({ autoClose: false });
    dst.setMaxListeners(splitFiles.length + 1);
    for (const splitFile of splitFiles) {
      console.log(`Merging ${splitFile}...`);
      const splitHandle = await fsp.open(splitFile, 'r');
      const src = splitHandle.createReadStream();
      await new Promise((resolve, reject) => {
        dst.on('error', reject);
        src.on('error', reject);
        src.on('end', resolve);
        src.pipe(dst, { end: false });
      });

      await splitHandle.close();
      if (inPlace) {
        await fsp.unlink(splitFile);
      }
    }

    await mergedHandle.close();
    if (inPlace && isArchive) {
      // NOTE: by not failing here we allow there to be other files inside the archive
      // bundles, e.g.: file/00, file/01, file/something_else
      await fsp.rmdir(dir).catch((err) => console.warn(err));
    }

    return { type: 'success', outputPath: path.dirname(mergedFilePath) };
  };

  try {
    return await inner();
  } catch (err) {
    console.error(err);
    const description = `Failed to merge file: ${err instanceof Error ? err.message : String(err)}`;
    return { type: 'failure', description };
  }
}
