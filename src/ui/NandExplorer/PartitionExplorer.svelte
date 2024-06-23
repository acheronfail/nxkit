<script lang="ts" context="module">
  import type { Partition } from 'src/channels';
  import type { Node } from '../utility/FileTree/FileTreeNode.svelte';

  export interface Props {
    partitions: Partition[];
    onPartitionChoose: (partition: Partition) => void;
    class?: string;
  }

  function partitionToNode(partition: Partition): Node<Partition, never> {
    return {
      id: partition.id,
      name: partition.name,
      isDirectory: false,
      isDisabled: !partition.mountable,
      data: partition,
    };
  }
</script>

<script lang="ts">
  import { CircleStackIcon, TrashIcon } from 'heroicons-svelte/24/solid';
  import FileTreeRoot from '../utility/FileTree/FileTreeRoot.svelte';
  import ActionButtons from '../utility/FileTree/ActionButtons.svelte';
  import Tooltip from '../utility/Tooltip.svelte';
  import ActionButton from '../utility/FileTree/ActionButton.svelte';

  let { partitions = $bindable(), onPartitionChoose, class: cls = '' }: Props = $props();

  let handlers = {
    format: async (partition: Partition) => {
      // TODO: show loading/disabled state while formatting
      const result = await window.nxkit.nandFormatPartition(partition.name);
      console.log(result);
    },
  };
</script>

<FileTreeRoot class={cls} nodes={partitions.map(partitionToNode)} onFileClick={onPartitionChoose}>
  {#snippet icon({ iconClass })}
    <CircleStackIcon class="text-red-300 {iconClass}" />
  {/snippet}
  {#snippet fileExtra(file)}
    <ActionButtons>
      {#if !file.mountable}
        <span>unsupported</span>
      {:else}
        {file.sizeHuman}
        <Tooltip placement="left">
          {#snippet tooltip()}
            <span>Format partition {(file as Partition).name} (<span class="text-red-500">data loss!</span>)</span>
          {/snippet}
          <ActionButton onclick={() => handlers.format(file)}>
            <TrashIcon class="h-4 cursor-pointer hover:fill-red-500 hover:stroke-2" />
          </ActionButton>
        </Tooltip>
      {/if}
    </ActionButtons>
  {/snippet}
</FileTreeRoot>
