import './index.css';

import { readFile, downloadFile } from '../browser/file';
import { buildNsp, generateRandomId } from '../hacbrewpack/nsp';
import { findRCMDevices, injectPayload } from '../rcm/inject';

async function keysFromUser(): Promise<Uint8Array | null> {
  const { files } = document.querySelector<HTMLInputElement>('#keys');
  if (files[0]) return readFile(files[0]);
  return null;
}

async function getKeys(): Promise<string | Uint8Array> {
  let keys: string | Uint8Array = await keysFromUser();
  if (!keys) keys = await window.nxkit.findProdKeys();
  if (!keys) {
    throw new Error('prod.keys are required to create an NSP!');
  }

  return keys;
}

// payload injector
{
  // TODO: bundle in some payloads
  // TODO: fetch latest payloads
  // TODO: doc linux udev: `SUBSYSTEM=="usb", ATTR{idVendor}=="0955", MODE="0664", GROUP="plugdev"` @ `/etc/udev/rules.d/50-switch.rules`
  // TODO: doc windows usb driver install:
  //  - download https://zadig.akeo.ie/
  //  - connect Switch in RCM
  //  - choose `APX`
  //  - select `libusbK`
  //  - select `Install Driver`

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (window as any).inject = async function () {
    const injectOutput = document.querySelector('#inject-logs');
    injectOutput.textContent = '';

    const { files } = document.querySelector<HTMLInputElement>('#rcm-payload');
    const [payload] = files;
    if (!payload) return alert('Please select a payload to inject!');

    if (window.nxkit.isWindows) {
      const result = await window.nxkit.runTegraRcmSmash(payload.path);
      injectOutput.textContent = result.stdout;
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
  // TODO: dynamic ui for retroarch forwarding, including verification and/or auto-complete of fields, etc
  // TODO: choose mounted Switch SD card for path autocomplete and validation

  document.querySelector<HTMLInputElement>('#nsp-id').value = generateRandomId();

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (window as any).generate = async function () {
    const { value: id } = document.querySelector<HTMLInputElement>('#nsp-id');
    const { value: title } = document.querySelector<HTMLInputElement>('#nsp-title');
    const { value: author } = document.querySelector<HTMLInputElement>('#nsp-author');
    const { value: nroPath } = document.querySelector<HTMLInputElement>('#nsp-nroPath');
    const keys = await getKeys();

    try {
      const result = await buildNsp({
        id,
        title,
        author,
        keys,
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

// TODO: nand viewer
{
  // https://gitlab.com/roothorick/busehac

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (window as any).openNand = async function () {
    const { files } = document.querySelector<HTMLInputElement>('#nand-file');
    const [rawNand] = files;
    if (!rawNand) return;

    const userKeys = await keysFromUser().then((k) => k && new TextDecoder().decode(k));

    const partitions = await window.nxkit.nandOpen(rawNand.path);
    console.log(partitions);

    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
    const userPartition = partitions.find((p) => p.name === 'USER')!.name;
    await window.nxkit.nandMount(userPartition, userKeys);

    const entries = await window.nxkit.nandReaddir('/');
    console.log(entries);

    await window.nxkit.nandClose();
  };
}

// TODO: tool to split/merge files to/from fat32 chunks
