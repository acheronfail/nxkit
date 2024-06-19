<script lang="ts">
  import { findRCMDevices, injectPayload } from '../rcm/inject';
  import Button from './utility/Button.svelte';
  import Container from './utility/Container.svelte';
  import { onMount } from 'svelte';
  import type { FSFile } from '../nand/fatfs/fs';
  import FileTreeRoot from './utility/FileTree/FileTreeRoot.svelte';
  import { entryToNode } from './NandExplorer/FileExplorer.svelte';
  import ActionButtons from './utility/FileTree/ActionButtons.svelte';
  import Tooltip from './utility/Tooltip.svelte';
  import DownloadPayloads from './PayloadInjector/DownloadPayloads.svelte';
  import LogOutput from './utility/LogOutput.svelte';

  // TODO: description of how to enter RCM mode
  // TODO: doc linux udev: `SUBSYSTEM=="usb", ATTR{idVendor}=="0955", MODE="0664", GROUP="plugdev"` @ `/etc/udev/rules.d/50-switch.rules`
  // TODO: doc windows usb driver install:
  //  - download https://zadig.akeo.ie/
  //  - connect Switch in RCM
  //  - choose `APX`
  //  - select `libusbK`
  //  - select `Install Driver`

  let output = $state('');
  let payloads = $state<FSFile[] | null>(null);

  async function updatePayloads() {
    payloads = await window.nxkit.payloadsFind();
  }

  onMount(() => {
    updatePayloads();
    window.addEventListener('focus', updatePayloads);
    return () => window.removeEventListener('focus', updatePayloads);
  });

  const handlers = {
    openPayloadDir: () => window.nxkit.payloadsOpenDirectory(),
    injectPayload: async (payloadPath: string) => {
      output = '';

      if (window.nxkit.isWindows) {
        const result = await window.nxkit.runTegraRcmSmash(payloadPath);
        output = result.stdout;
        if (result.stderr) output += ' -- -- -- \n' + result.stderr;
      } else {
        const [dev] = await findRCMDevices();
        if (!dev) return alert('No Switch found in RCM mode!');

        const payloadBytes = await window.nxkit.payloadsReadFile(payloadPath);
        await injectPayload(dev, payloadBytes, (log) => (output += `${log}\n`));
      }
    },
  };
</script>

<!-- svelte-ignore slot_element_deprecated -->
<Container fillContainer>
  <div class="flex flex-col gap-2 h-full">
    {#if payloads?.length}
      <p class="text-center">Choose a payload to inject to a Switch in RCM mode</p>
      <FileTreeRoot nodes={payloads.map(entryToNode)}>
        <ActionButtons slot="file-extra" let:file>
          <span>{file.sizeHuman}</span>
          <Tooltip placement="left">
            <div slot="tooltip">Inject {file.name}</div>
            <div slot="content">
              <Button size="small" appearance="primary" onclick={() => handlers.injectPayload(file.path)}>
                inject
              </Button>
            </div>
          </Tooltip>
        </ActionButtons>
      </FileTreeRoot>
      <DownloadPayloads />
      <div class="text-center">
        <Button onclick={handlers.openPayloadDir}>Open Payload Folder</Button>
      </div>
    {:else}
      <div class="h-full flex flex-col gap-2 justify-center items-center">
        <h3 class="font-bold">No payloads found!</h3>
        <DownloadPayloads />
        <Button appearance="primary" onclick={handlers.openPayloadDir}>Open Payload Folder</Button>
      </div>
    {/if}

    {#if output}
      <LogOutput title="Payload Injection Log" bind:output />
    {/if}
  </div>
</Container>
