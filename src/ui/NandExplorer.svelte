<script lang="ts">
  import { keys } from './stores/keys.svelte';
  import Button from './utility/Button.svelte';
  import Container from './utility/Container.svelte';
  import Code from './utility/Code.svelte';
  import { NandError } from '../channels';
  import FileExplorer from './NandExplorer/FileExplorer.svelte';
  import PartitionExplorer from './NandExplorer/PartitionExplorer.svelte';
  import type { FSEntry } from '../nand/fatfs/fs';
  import type { PartitionEntry } from '../nand/gpt';
  import { onMount } from 'svelte';
  import Tooltip from './utility/Tooltip.svelte';

  let input = $state<HTMLInputElement | null>(null);
  let files = $state<FileList | null>(null);
  let nandFile = $derived(files?.[0]);
  let loading = $state(false);
  let disabled = $derived(!keys.value || loading);
  let tooltip = $derived(loading ? 'Loading...' : disabled ? 'Please select your prod.keys in Settings!' : undefined);

  let partitions = $state<PartitionEntry[] | null>(null);
  let selectedPartition = $state<PartitionEntry | null>(null);
  let rootEntries = $state<FSEntry[] | null>(null);

  const handlers = {
    onNandChoose: async () => {
      if (nandFile) {
        const result = await window.nxkit.nandOpen(nandFile.path);
        switch (result.error) {
          case NandError.None:
            partitions = result.data;
            break;
          case NandError.InvalidPartitionTable:
            return alert('Failed to read partition table, did you select a Nand dump?');
          default:
            return alert('An unknown error occurred when trying to open the Nand!');
        }
      }
    },
    onPartitionChoose: async (partition: PartitionEntry) => {
      selectedPartition = partition;
      const { error } = await window.nxkit.nandMount(selectedPartition.name, $state.snapshot(keys.value));
      switch (error) {
        case NandError.None:
          break;
        case NandError.InvalidProdKeys:
          return alert("Failed to read partition, please ensure you're using the right prod.keys!");
        default:
          return alert(`An unknown error occurred when trying to mount ${selectedPartition.name}!`);
      }

      const result = await window.nxkit.nandReaddir('/');
      if (result.error === NandError.None) {
        rootEntries = result.data;
      }
    },
    closePartition: () => {
      rootEntries = null;
      selectedPartition = null;
    },
    reset: () => {
      loading = true;
      window.nxkit.nandClose().finally(() => {
        loading = false;
        rootEntries = null;
        partitions = null;
        selectedPartition = null;
        files = null;
        if (input) input.value = '';
      });
    },
  };
</script>

<Container>
  <div class="flex flex-col">
    {#if nandFile}
      <Button appearance="warning" size="large" {disabled} {tooltip} onclick={handlers.reset}>Close NAND</Button>
    {:else}
      <Tooltip>
        <p slot="tooltip" class="text-center w-96">
          Choose either a complete dump <Code>rawnand.bin</Code>
          <br />Or the first part of a split dump <Code>rawnand.bin.00</Code>
        </p>
        <Button slot="content" appearance="primary" size="large" class="block" for="rawnand-file" {disabled} {tooltip}>
          Choose your rawnand.bin
        </Button>
      </Tooltip>
    {/if}

    <input hidden id="rawnand-file" type="file" bind:files bind:this={input} onchange={handlers.onNandChoose} />
  </div>

  <div class="m-2">
    {#if rootEntries && selectedPartition}
      <p class="text-center">
        Currently exploring <strong class="font-mono text-red-300">{selectedPartition.name}</strong>
      </p>
      <div class="text-center">
        <Button size="inline" onclick={handlers.closePartition}>choose another partition</Button>
      </div>
      <FileExplorer {rootEntries} />
    {:else if partitions}
      <p class="text-center">Choose a partition to explore</p>
      <PartitionExplorer bind:partitions onPartitionChoose={handlers.onPartitionChoose} />
    {:else}
      <p class="text-center">
        Choose your <Code>rawnand.bin</Code> file to begin!
      </p>
    {/if}
  </div>
</Container>
