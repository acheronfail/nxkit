<script lang="ts">
  import { readFile } from '../browser/file';
  import { findRCMDevices, injectPayload } from '../rcm/inject';
  import Button from './utility/Button.svelte';
  import InputFile from './utility/InputFile.svelte';
  import Container from './utility/Container.svelte';

  // TODO: bundle in some payloads
  // TODO: fetch latest payloads
  // TODO: doc linux udev: `SUBSYSTEM=="usb", ATTR{idVendor}=="0955", MODE="0664", GROUP="plugdev"` @ `/etc/udev/rules.d/50-switch.rules`
  // TODO: doc windows usb driver install:
  //  - download https://zadig.akeo.ie/
  //  - connect Switch in RCM
  //  - choose `APX`
  //  - select `libusbK`
  //  - select `Install Driver`

  let files = $state<FileList | null>(null);
  let payload = $derived(files?.[0]);
  let output = $state('');

  let disabled = $derived(!payload);
  let tooltip = $derived(disabled && 'Please select a payload!');

  async function inject() {
    output = '';

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

<!-- svelte-ignore slot_element_deprecated -->
<Container>
  <div class="flex flex-col gap-2">
    <p>You can send payloads to a Switch in RCM mode directly.</p>
    <!-- TODO: description of how to enter RCM mode -->
    <InputFile label="Payload" bind:files>
      <div slot="infoTooltip" class="text-sm w-60 text-center">
        The payload to send to the Switch. E.g.,
        <span class="font-mono">tegraexplorer.bin</span>,
        <span class="font-mono">hekate.bin</span>, etc
      </div>
    </InputFile>
    <Button appearance="primary" size="large" {tooltip} {disabled} onclick={inject}>Inject</Button>
    <pre id="inject-logs">{output}</pre>
  </div>
</Container>
