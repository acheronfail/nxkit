<script lang="ts" context="module">
  import DescriptionHekate from './DescriptionHekate.svelte';
  import DescriptionFusee from './DescriptionFusee.svelte';
  import type { Component } from 'svelte';

  interface PayloadOption {
    link: string;
    displayName: string;
    description?: Component;
  }

  const payloadOptions: Record<string, PayloadOption> = {
    hekate: {
      displayName: 'hekate_ctcaer.bin',
      description: DescriptionHekate,
      link: 'https://github.com/CTCaer/hekate/releases/latest',
    },
    fusee: {
      displayName: 'fusee.bin',
      description: DescriptionFusee,
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
      window.nxkit.openLink(selectedPayload.link);
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
          {#if selectedPayload.description}
            <svelte:component this={selectedPayload.description} />
          {/if}
        {:else}
          Select an option!
        {/if}
      </div>
    {/snippet}
    <Button size="inline" appearance="primary" disabled={!selectedPayloadKey} onclick={downloadPayload}>open</Button>
  </Tooltip>
</div>
