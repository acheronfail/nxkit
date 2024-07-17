<script lang="ts" context="module">
  export interface Props {
    nandFilePath?: string;
    partitionName?: string;
    readonlyDefault?: boolean;
  }
</script>

<script lang="ts">
  import { keys } from './stores/keys.svelte';
  import Button from './utility/Button.svelte';
  import Container from './utility/Container.svelte';
  import Code from './utility/Code.svelte';
  import { type Partition } from '../channels';
  import FileExplorer from './NandExplorer/FileExplorer.svelte';
  import PartitionExplorer from './NandExplorer/PartitionExplorer.svelte';
  import Tooltip from './utility/Tooltip.svelte';
  import { onMount } from 'svelte';
  import { handleNandResult } from './errors';
  import InputCheckbox from './utility/InputCheckbox.svelte';
  import Markdown from './utility/Markdown.svelte';
  import readonlyMd from './markdown/nand-readonly.md?raw';

  // TODO: support working directly with a connected switch, not just with dumps

  let { nandFilePath, partitionName, readonlyDefault = true }: Props = $props();

  let input = $state<HTMLInputElement | null>(null);
  let loading = $state(false);
  let readonly = $state(readonlyDefault);
  let partDisabled = $state(false);

  let disabled = $derived(!keys.value || loading || partDisabled);
  let readonlyDisabled = $derived(!!nandFilePath);

  let partitions = $state<Partition[] | null>(null);
  let selectedPartition = $state<Partition | null>(null);

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
        await handlers.onPartitionChoose(part);
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
        const result = handleNandResult(await window.nxkit.call('nandOpen', nandFilePath), 'Open NAND');
        if (result) {
          partitions = result;
        } else {
          handlers.reset();
        }
      }
    },
    verifyPartitionTable: async () => {
      const result = handleNandResult(await window.nxkit.call('nandVerifyPartitionTable'), 'Verify partition table');

      if (result !== null) {
        alert('Partition table is valid!');
      }
    },
    repairBackupPartitionTable: async () => {
      const result = handleNandResult(
        await window.nxkit.call('nandRepairBackupPartitionTable'),
        'Repair backup partition table',
      );

      if (result !== null) {
        alert('Repair of backup partition table complete!');
      }
    },
    onPartitionChoose: async (partition: Partition) => {
      const result = handleNandResult(
        await window.nxkit.call(
          'nandMountPartition',
          partition.name,
          $state.snapshot(readonly),
          $state.snapshot(keys.value),
        ),
        `Mount partition '${partition.name}'`,
      );

      if (result !== null) {
        selectedPartition = partition;
      }
    },
    closePartition: () => {
      selectedPartition = null;
    },
    reset: () => {
      loading = true;
      window.nxkit.call('nandClose').finally(() => {
        loading = false;
        partitions = null;
        selectedPartition = null;
        nandFilePath = undefined;
        if (input) input.value = '';
      });
    },
  };
</script>

<Container fillParent data-testid="nandexplorer">
  <div class="flex flex-col">
    <Tooltip>
      {#snippet tooltip()}
        <p class="text-center w-96 leading-6">
          {#if !keys.value}
            Please select your prod.keys in Settings!
          {:else if loading}
            Loading...
          {:else if nandFilePath}
            Currently exploring:
            {#each nandFilePath.split('/').filter(Boolean) as part}
              {' / '}<Code>{part}</Code>
            {/each}
          {:else}
            Choose either a complete dump <Code>rawnand.bin</Code>
            <br />
            Or the first part of a split dump <Code>rawnand.bin.00</Code>
          {/if}
        </p>
      {/snippet}
      <span class="block">
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
    {#if selectedPartition}
      <p class="text-center">
        Currently exploring <strong class="font-mono text-red-300">{selectedPartition.name}</strong>
      </p>
      <div class="flex justify-center items-center gap-2 mt-2">
        <Button size="inline" onclick={handlers.closePartition}>choose another partition</Button>
      </div>
      <FileExplorer {readonly} class="overflow-auto grow h-0" />
    {:else if partitions}
      <p class="text-center">Choose a partition to explore</p>
      <div class="flex justify-center items-center gap-2 mt-2">
        <Button size="inline" onclick={handlers.verifyPartitionTable}>verify partition table</Button>
        <Tooltip>
          <Button size="inline" onclick={handlers.repairBackupPartitionTable} appearance="danger" disabled={readonly}>
            repair backup gpt table
          </Button>
          {#snippet tooltip()}
            <div class="w-96">
              <p class="text-center">
                Attempt to repair the Backup GPT partition table by re-creating it from the Primary GPT partition table.
              </p>
              <p class="text-center">Only do this if you know what you're doing!</p>
            </div>
          {/snippet}
        </Tooltip>
      </div>
      <PartitionExplorer
        class="overflow-auto grow h-0"
        {readonly}
        {partitions}
        reloadPartitions={() => {
          partitions = null;
          handlers.openNand();
        }}
        bind:disabled={partDisabled}
        onPartitionChoose={handlers.onPartitionChoose}
      />
    {:else if nandFilePath}
      <p class="text-center">Loading...</p>
    {:else}
      <p class="text-center">
        Choose your <Code>rawnand.bin</Code> file to begin!
      </p>
    {/if}
  </div>

  <div class="flex justify-center">
    <InputCheckbox
      id="readonly"
      appearance="pronounced"
      disabled={readonlyDisabled}
      bind:checked={readonly}
      tooltipPlacement="top"
    >
      {#snippet tooltip()}
        <div class="w-96 text-center">
          <Markdown content={readonlyMd} />
          {#if readonlyDisabled}
            <br />
            <p class="text-yellow-600">
              Close NAND to {#if readonly}disable{:else}enable{/if}!
            </p>
          {/if}
        </div>
      {/snippet}
      {#snippet label()}
        Read-Only:
      {/snippet}
    </InputCheckbox>
  </div>
</Container>
