<script lang="ts" context="module">
  import downloadHekateMd from '../markdown/download-hekate.md?raw';
  import downloadFuseeMd from '../markdown/download-fusee.md?raw';

  interface PayloadOption {
    link: string;
    displayName: string;
    markdown?: string;
  }

  const payloadOptions: Record<string, PayloadOption> = {
    hekate: {
      displayName: 'hekate_ctcaer.bin',
      markdown: downloadHekateMd,
      link: 'https://github.com/CTCaer/hekate/releases/latest',
    },
    fusee: {
      displayName: 'fusee.bin',
      markdown: downloadFuseeMd,
      link: 'https://github.com/Atmosphere-NX/Atmosphere/releases/latest',
    },
    tegraExplorer: {
      displayName: 'tegraexplorer.bin',
      link: 'https://github.com/suchmememanyskill/TegraExplorer/releases/latest',
    },
  };
</script>

<script lang="ts">
  import InputSelect, { type Option } from '../utility/InputSelect.svelte';
  import Tooltip from '../utility/Tooltip.svelte';
  import Code from '../utility/Code.svelte';
  import Button from '../utility/Button.svelte';
  import Markdown from '../utility/Markdown.svelte';

  const options: Option[] = [
    { displayName: 'select payload', value: '', disabled: true, selected: true },
    ...Object.entries(payloadOptions).map(([value, { displayName }]) => ({ displayName, value })),
  ];

  let selectedPayloadKey = $state<string | undefined>();
  let selectedPayload = $derived<PayloadOption | undefined>(
    selectedPayloadKey ? payloadOptions[selectedPayloadKey] : undefined,
  );

  function downloadPayload() {
    if (selectedPayload) {
      window.nxkit.call('openLink', selectedPayload.link);
    }
  }
</script>

<div class="text-center">
  Go download a payload:
  <InputSelect small {options} bind:value={selectedPayloadKey} />
  <Tooltip>
    {#snippet tooltip()}
      <div>
        {#if selectedPayload}
          <p class="text-center">
            Opens <Code>{selectedPayload.link}</Code>
          </p>
          {#if selectedPayload.markdown}
            <Markdown class="mt-2" content={selectedPayload.markdown} />
          {/if}
        {:else}
          Select an option!
        {/if}
      </div>
    {/snippet}
    <Button size="inline" appearance="primary" disabled={!selectedPayloadKey} onclick={downloadPayload}>open</Button>
  </Tooltip>
</div>
