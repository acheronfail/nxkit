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
      // FIXME: get slots or snippets working to be able to style these and remove this cast
      icon: CircleStackIcon as any as Component,
    };
  }
</script>

<script lang="ts">
  import { keys } from './stores/keys.svelte';
  import Button from './utility/Button.svelte';
  import Container from './utility/Container.svelte';
  import FileTreeRoot from './utility/FileTreeRoot.svelte';
  import Code from './utility/Code.svelte';
  import { CircleStackIcon } from 'heroicons-svelte/24/outline';
  import { ArrowDownTrayIcon } from 'heroicons-svelte/24/solid';
  import { NandError } from '../channels';
  import { Tooltip } from '@svelte-plugins/tooltips';

  let input = $state<HTMLInputElement | null>(null);
  let files = $state<FileList | null>(null);
  let nandFile = $derived(files?.[0]);
  let loading = $state(false);
  let disabled = $derived(!keys.value || loading);
  let tooltip = $derived(loading ? 'Loading...' : disabled && 'Please select your prod.keys in Settings!');

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

      rootEntries = await window.nxkit.nandReaddir('/');
    },
    closePartition: () => {
      rootEntries = null;
      selectedPartition = null;
    },
    openNandDirectory: async (dir: FSDirectory): Promise<Node<FSDirectory, FSFile>[]> => {
      return window.nxkit.nandReaddir(dir.path).then((entries) => entries.map(entryToNode));
    },
    downloadFile: async (file: FSFile) => {
      await window.nxkit.nandCopyFile(file.path);
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
</script>

<Container>
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
      <FileTreeRoot nodes={rootEntries.map(entryToNode)} openDirectory={handlers.openNandDirectory}>
        <div slot="file-extra" let:file class="flex flex-row justify-end items-center gap-2">
          <span class="font-mono">{file.sizeHuman}</span>
          <Tooltip content="Download {file.name}" position="left">
            <ArrowDownTrayIcon
              class="h-4 cursor-pointer hover:text-black"
              onclick={() => handlers.downloadFile(file)}
            />
          </Tooltip>
        </div>
      </FileTreeRoot>
    {:else if partitions}
      <p class="text-center">Choose a partition to explore</p>
      <FileTreeRoot nodes={partitions.map(partitionToNode)} onFileClick={handlers.onPartitionChoose}>
        <div slot="file-extra"></div>
      </FileTreeRoot>
    {:else}
      <p class="text-center">
        Choose your <Code>rawnand.bin</Code> file to begin!
      </p>
    {/if}
  </div>
</Container>
