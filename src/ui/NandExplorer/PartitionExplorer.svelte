<script lang="ts" context="module">
  import type { Partition } from 'src/channels';
  import type { Node } from '../utility/FileTree/FileTreeNode.svelte';

  export interface Props {
    partitions: Partition[];
    onPartitionChoose: (partition: Partition) => void;
    class?: string;
    disabled?: boolean;
    readonly: boolean;
  }
</script>

<script lang="ts">
  import { CircleStackIcon, TrashIcon } from 'heroicons-svelte/24/solid';
  import FileTreeRoot from '../utility/FileTree/FileTreeRoot.svelte';
  import ActionButtons from '../utility/FileTree/ActionButtons.svelte';
  import Tooltip from '../utility/Tooltip.svelte';
  import ActionButton from '../utility/FileTree/ActionButton.svelte';
  import { handleNandResult } from '../errors';
  import { keys } from '../stores/keys.svelte';

  let {
    readonly,
    partitions = $bindable(),
    disabled = $bindable(false),
    onPartitionChoose,
    class: cls = '',
  }: Props = $props();

  let formattingPartitionId = $state<string | null>(null);

  let handlers = {
    format: async (partition: Partition) => {
      const yes = confirm(
        `Are you sure you want to format ${partition.name}?\n\nThis will delete all files and folders on this partition!`,
      );
      if (yes) {
        formattingPartitionId = partition.id;
        disabled = true;
        window.nxkit
          .nandFormatPartition(partition.name, readonly, $state.snapshot(keys.value))
          .then((result) => handleNandResult(result, `format partition '${partition.name}'`))
          .finally(() => {
            disabled = false;
            formattingPartitionId = null;
          });
      }
    },
  };

  function partitionToNode(partition: Partition): Node<Partition, never> {
    return {
      id: partition.id,
      name: partition.name,
      isDirectory: false,
      isDisabled: disabled || !partition.mountable,
      data: partition,
    };
  }
</script>

<FileTreeRoot class={cls} nodes={partitions.map(partitionToNode)} onFileClick={onPartitionChoose}>
  {#snippet icon({ iconClass })}
    <CircleStackIcon class="text-red-300 {iconClass}" />
  {/snippet}
  {#snippet fileExtra(file)}
    <ActionButtons>
      {#if !file.mountable}
        <span>unsupported</span>
      {:else if file.id === formattingPartitionId}
        <span class="text-red-500">formatting...</span>
      {:else}
        {file.sizeHuman}
        <Tooltip placement="left">
          {#snippet tooltip()}
            <span>Format partition {(file as Partition).name} (<span class="text-red-500">data loss!</span>)</span>
          {/snippet}
          <ActionButton {disabled} onclick={() => handlers.format(file)}>
            <TrashIcon class="h-4 cursor-pointer hover:fill-red-500 hover:stroke-2" />
          </ActionButton>
        </Tooltip>
      {/if}
    </ActionButtons>
  {/snippet}
</FileTreeRoot>
