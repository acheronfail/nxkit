import './index.css';

import { buildNsp, generateRandomId } from './hacbrewpack/nsp';
import { readFile, downloadFile } from './browser/file';
import { findRCMDevices, injectPayload } from './rcm/inject';

// payload injector
{
  // TODO: bundle in some payloads
  // TODO: fetch latest payloads

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (window as any).inject = async function () {
    const injectOutput = document.querySelector('#inject-logs');
    injectOutput.textContent = '';

    const { files } = document.querySelector<HTMLInputElement>('#rcm-payload');
    const [payload] = files;
    if (!payload) return alert('Please select a payload to inject!');

    if (window.nxkit.isWindows) {
      const result = await window.nxkitTegraRcmSmash.run(payload.path);
      if (result.stderr) injectOutput.textContent += ' -- -- -- \n' + result.stderr;
    } else {
      const [dev] = await findRCMDevices();
      if (!dev) return alert('No Switch found in RCM mode!');

      await injectPayload(dev, await readFile(payload), (log) => (injectOutput.textContent += `${log}\n`));
    }
  };
}

// nro forwarder
{
  document.querySelector<HTMLInputElement>('#nsp-id').value = generateRandomId();

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (window as any).generate = async function () {
    const { value: id } = document.querySelector<HTMLInputElement>('#nsp-id');
    const { value: title } = document.querySelector<HTMLInputElement>('#nsp-title');
    const { value: author } = document.querySelector<HTMLInputElement>('#nsp-author');
    const { value: nroPath } = document.querySelector<HTMLInputElement>('#nsp-nroPath');
    const { files } = document.querySelector<HTMLInputElement>('#nsp-keys');
    const [keys] = files;
    if (!keys) return alert('prod.keys are required to create an NSP!');

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
}
