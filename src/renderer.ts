/**
 * This file will automatically be loaded by vite and run in the "renderer" context.
 * To learn more about the differences between the "main" and the "renderer" context in
 * Electron, visit:
 *
 * https://electronjs.org/docs/tutorial/application-architecture#main-and-renderer-processes
 *
 * By default, Node.js integration in this file is disabled. When enabling Node.js integration
 * in a renderer process, please be aware of potential security implications. You can read
 * more about security risks here:
 *
 * https://electronjs.org/docs/tutorial/security
 *
 * To enable Node.js integration in this file, open up `main.ts` and enable the `nodeIntegration`
 * flag:
 *
 * ```
 *  // Create the browser window.
 *  mainWindow = new BrowserWindow({
 *    width: 800,
 *    height: 600,
 *    webPreferences: {
 *      nodeIntegration: true
 *    }
 *  });
 * ```
 */

import './index.css';

import { buildNsp, generateRandomId } from './hacbrewpack/nsp';
import { downloadFile } from './browser/download';

document.querySelector<HTMLInputElement>('#nsp-id').value = generateRandomId();

// eslint-disable-next-line @typescript-eslint/no-explicit-any
(window as any).generate = async function () {
  const { value: id } = document.querySelector<HTMLInputElement>('#nsp-id');
  const { value: title } = document.querySelector<HTMLInputElement>('#nsp-title');
  const { value: author } = document.querySelector<HTMLInputElement>('#nsp-author');
  const { value: nroPath } = document.querySelector<HTMLInputElement>('#nsp-nroPath');
  const {
    files: [keys],
  } = document.querySelector<HTMLInputElement>('#nsp-keys');

  try {
    const result = await buildNsp({
      id,
      title,
      author,
      keys: await readFile(keys),
      nroPath,
      nroArgv: [],
    });

    document.querySelector('#nsp-stdout').textContent = result.stdout;
    document.querySelector('#nsp-stderr').textContent = result.stderr;

    if (result.exitCode !== 0) {
      alert(`Error generating NSP, please check the logs`);
    }

    if (result.nsp) {
      downloadFile(result.nsp);
    }
  } catch (err) {
    alert(String(err));
  }
};

async function readFile(file: File): Promise<Uint8Array> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onerror = reject;
    reader.onload = (event) => resolve(new Uint8Array(event.target.result as ArrayBuffer));

    reader.readAsArrayBuffer(file);
  });
}
