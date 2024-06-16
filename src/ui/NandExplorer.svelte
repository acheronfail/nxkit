<script lang="ts" context="module">
  import type { FSDirectory, FSEntry, FSFile } from '../nand/fatfs/fs';
  import type { PartitionEntry } from '../nand/gpt';
  import type { Node } from './utility/FileTreeNode.svelte';
  import type { Component } from 'svelte';

  function entryToNode(entry: FSEntry): Node<FSDirectory, FSFile> {
    return {
      id: entry.path,
      name: entry.name,
      isDirectory: entry.type === 'd',
      data: entry,
    };
  }

  function partitionToNode(partition: PartitionEntry): Node<PartitionEntry, never> {
    return {
      id: partition.type,
      name: partition.name,
      isDirectory: false,
      data: partition,
      // TODO: what's the proper way to pass a svelte component like this? can I send props with it? (class:text-red)
      icon: CircleStackIcon as any as Component,
    };
  }
</script>

<script lang="ts">
  import { keys } from './stores/keys.svelte';
  import Button from './utility/Button.svelte';
  import TabContent from './utility/TabContent.svelte';
  import FileTreeRoot from './utility/FileTreeRoot.svelte';
  import Code from './utility/Code.svelte';
  import { CircleStackIcon } from 'heroicons-svelte/24/outline';

  let input = $state<HTMLInputElement | null>(null);
  let files = $state<FileList | null>(null);
  let nandFile = $derived(files?.[0]);
  let loading = $state(false);
  let disabled = $derived(loading || !keys.value);
  let tooltip = $derived(loading ? 'Loading...' : disabled && 'Please select your prod.keys in Settings!');

  let partitions = $state<PartitionEntry[] | null>(null);
  let selectedPartition = $state<PartitionEntry | null>(null);
  let rootEntries = $state<FSEntry[] | null>(null);

  const handlers = {
    onNandChoose: async () => {
      if (nandFile) {
        partitions = await window.nxkit.nandOpen(nandFile.path);
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
        input.value = '';
      });
    },
  };

  async function openNandDirectory(dir: FSDirectory): Promise<Node<FSDirectory, FSFile>[]> {
    return window.nxkit.nandReaddir(dir.path).then((entries) => entries.map(entryToNode));
  }

  async function onPartitionClick(partition: PartitionEntry) {
    selectedPartition = partition;
    await window.nxkit.nandMount(selectedPartition.name, $state.snapshot(keys.value));
    rootEntries = await window.nxkit.nandReaddir('/');
  }
</script>

<TabContent>
  <div class="flex flex-col">
    {#if nandFile}
      <Button appearance="warning" size="large" {disabled} {tooltip} onclick={handlers.reset}>Close Nand</Button>
    {:else}
      <Button appearance="primary" size="large" for="rawnand-file" {disabled} {tooltip}>Choose your rawnand.bin</Button>
    {/if}

    <input hidden id="rawnand-file" type="file" bind:files bind:this={input} onchange={handlers.onNandChoose} />
  </div>

  <div class="m-2">
    {#if rootEntries}
      <p class="text-center">
        Currently exploring <strong>{selectedPartition.name}</strong>
        <Button size="inline" onclick={handlers.closePartition}>choose another partition</Button>
      </p>
      <FileTreeRoot nodes={rootEntries.map(entryToNode)} openDirectory={openNandDirectory} />
    {:else if partitions}
      <p class="text-center">Choose a partition to explore</p>
      <FileTreeRoot nodes={partitions.map(partitionToNode)} onFileClick={onPartitionClick} />
    {:else}
      <p class="text-center">
        Choose your <Code>rawnand.bin</Code> file to begin!
      </p>
    {/if}
  </div>
</TabContent>
