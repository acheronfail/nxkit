<script lang="ts" context="module">
  export interface Props {
    nandFilePath?: string;
    partitionName?: string;
  }
</script>

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
  import Tooltip from './utility/Tooltip.svelte';
  import { onMount } from 'svelte';

  let { nandFilePath, partitionName }: Props = $props();

  // FIXME: when navigating away, state is lost (probably fixing Tabs component is best way)

  let input = $state<HTMLInputElement | null>(null);
  let loading = $state(false);
  let disabled = $derived(!keys.value || loading);

  let partitions = $state<PartitionEntry[] | null>(null);
  let selectedPartition = $state<PartitionEntry | null>(null);
  let rootEntries = $state<FSEntry[] | null>(null);

  // handle initial props
  onMount(async () => {
    // if a nand path was already set, try to open it
    if (nandFilePath) {
      await handlers.openNand();
    }

    // if a partition name was given, try to find it and mount it
    if (partitionName) {
      const part = partitions?.find((part) => part.name === partitionName);
      if (part) {
        handlers.onPartitionChoose(part);
      }
    }

    // clear partition name now since we don't need it anymore
    partitionName = undefined;
  });

  const handlers = {
    onNandChoose: async (newNandPath: string | undefined) => {
      nandFilePath = newNandPath;
      await handlers.openNand();
    },
    openNand: async () => {
      if (nandFilePath) {
        const result = await window.nxkit.nandOpen(nandFilePath);
        switch (result.error) {
          case NandError.None:
            partitions = result.data;
            break;
          case NandError.InvalidPartitionTable:
            return alert('Failed to read partition table, did you select a Nand dump?');
          default:
            console.log(result);
            return alert('An unknown error occurred when trying to open the Nand!');
        }
      }
    },
    onPartitionChoose: async (partition: PartitionEntry) => {
      selectedPartition = partition;
      {
        const result = await window.nxkit.nandMount(selectedPartition.name, $state.snapshot(keys.value));
        switch (result.error) {
          case NandError.None:
            break;
          case NandError.InvalidProdKeys:
            return alert("Failed to read partition, please ensure you're using the right prod.keys!");
          default:
            console.log(result);
            return alert(`An unknown error occurred when trying to mount ${selectedPartition.name}!`);
        }
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
        nandFilePath = undefined;
        if (input) input.value = '';
      });
    },
  };
</script>

<Container fillContainer data-testid="nandexplorer">
  <div class="flex flex-col">
    <Tooltip>
      <p slot="tooltip" class="text-center w-96">
        {#if !keys.value}
          Please select your prod.keys in Settings!
        {:else if loading}
          Loading...
        {:else if nandFilePath}
          Currently exploring <Code>{nandFilePath}</Code>
        {:else}
          Choose either a complete dump <Code>rawnand.bin</Code>
          <br />Or the first part of a split dump <Code>rawnand.bin.00</Code>
        {/if}
      </p>
      <span slot="content" class="block">
        {#if nandFilePath}
          <Button class="w-full block" appearance="warning" size="large" {disabled} onclick={handlers.reset}>
            Close NAND
          </Button>
        {:else}
          <Button class="w-full block" appearance="primary" size="large" for="rawnand-file" {disabled}>
            Choose your rawnand.bin
          </Button>
        {/if}
      </span>
    </Tooltip>

    <input
      hidden
      id="rawnand-file"
      type="file"
      bind:this={input}
      onchange={(e) => handlers.onNandChoose(e.currentTarget.files?.[0].path)}
    />
  </div>

  <div data-testid="explorer-wrapper" class="flex flex-col grow">
    {#if rootEntries && selectedPartition}
      <p class="text-center">
        Currently exploring <strong class="font-mono text-red-300">{selectedPartition.name}</strong>
      </p>
      <div class="text-center">
        <Button size="inline" onclick={handlers.closePartition}>choose another partition</Button>
      </div>
      <FileExplorer class="overflow-auto grow h-0" {rootEntries} />
    {:else if partitions}
      <p class="text-center">Choose a partition to explore</p>
      <PartitionExplorer
        class="overflow-auto grow h-0"
        bind:partitions
        onPartitionChoose={handlers.onPartitionChoose}
      />
    {:else}
      <p class="text-center">
        Choose your <Code>rawnand.bin</Code> file to begin!
      </p>
    {/if}
  </div>
</Container>
