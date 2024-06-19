<script lang="ts" context="module">
  import type { PartitionEntry } from '../../nand/gpt';
  import type { Node } from '../utility/FileTreeNode.svelte';
  import prettyBytes from 'pretty-bytes';

  export interface Props {
    partitions: PartitionEntry[];
    onPartitionChoose: (partition: PartitionEntry) => void;
  }

  function getPartitionSize(partition: PartitionEntry): string {
    return prettyBytes(Number((partition.lastLBA - partition.firstLBA) * 512n));
  }

  function isPartitionDisabled(partition: PartitionEntry): boolean {
    // NOTE: currently all non FAT partitions are disabled, since we don't support
    // mounting them (yet!)
    return !['USER', 'SAFE', 'SYSTEM', 'PRODINFOF'].includes(partition.name);
  }

  function partitionToNode(partition: PartitionEntry): Node<PartitionEntry, never> {
    return {
      id: partition.type,
      name: partition.name,
      isDirectory: false,
      isDisabled: isPartitionDisabled(partition),
      data: partition,
    };
  }
</script>

<script lang="ts">
  import { CircleStackIcon, XCircleIcon } from 'heroicons-svelte/24/solid';
  import FileTreeRoot from '../utility/FileTreeRoot.svelte';
  import ActionButtons from './ActionButtons.svelte';
  import Tooltip from '../utility/Tooltip.svelte';
  import ActionButton from './ActionButton.svelte';

  let { partitions = $bindable(), onPartitionChoose }: Props = $props();

  let handlers = {
    format: async (partition: PartitionEntry) => {
      // TODO: do we need to disable anything in the UI while formatting?
      const result = await window.nxkit.nandFormatPartition(partition.name);
      console.log(result);
    },
  };
</script>

<FileTreeRoot nodes={partitions.map(partitionToNode)} onFileClick={onPartitionChoose}>
  <CircleStackIcon slot="icon" let:iconClass class="text-red-300 {iconClass}" />
  <ActionButtons slot="file-extra" let:file>
    {#if isPartitionDisabled(file)}
      <span>unsupported</span>
    {:else}
      <span>
        {getPartitionSize(file)}
        <Tooltip placement="left">
          <span slot="tooltip">Format {(file as PartitionEntry).name}</span>
          <ActionButton class="hidden" slot="content" onclick={() => handlers.format(file)}>
            <XCircleIcon class="h-4 cursor-pointer hover:fill-slate-900 hover:stroke-2" />
          </ActionButton>
        </Tooltip>
      </span>
    {/if}
  </ActionButtons>
</FileTreeRoot>
