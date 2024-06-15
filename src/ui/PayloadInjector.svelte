<script lang="ts">
  import { readFile } from "../browser/file";
  import { findRCMDevices, injectPayload } from "../rcm/inject";

  // TODO: bundle in some payloads
  // TODO: fetch latest payloads
  // TODO: doc linux udev: `SUBSYSTEM=="usb", ATTR{idVendor}=="0955", MODE="0664", GROUP="plugdev"` @ `/etc/udev/rules.d/50-switch.rules`
  // TODO: doc windows usb driver install:
  //  - download https://zadig.akeo.ie/
  //  - connect Switch in RCM
  //  - choose `APX`
  //  - select `libusbK`
  //  - select `Install Driver`


  let output = '';

  async function inject() {
    output = '';

    const { files } = document.querySelector<HTMLInputElement>('#rcm-payload');
    const [payload] = files;
    if (!payload) return alert('Please select a payload to inject!');

    if (window.nxkit.isWindows) {
      const result = await window.nxkit.runTegraRcmSmash(payload.path);
      output = result.stdout;
      if (result.stderr) output += ' -- -- -- \n' + result.stderr;
    } else {
      const [dev] = await findRCMDevices();
      if (!dev) return alert('No Switch found in RCM mode!');

      await injectPayload(dev, await readFile(payload, 'arrayBuffer'), (log) => (output += `${log}\n`));
    }
  }
</script>

<span>Choose payload to inject</span>
<input type="file" name="rcm-payload" id="rcm-payload" />
<input type="submit" value="Inject" onclick={inject} />
<pre id="inject-logs">{output}</pre>
