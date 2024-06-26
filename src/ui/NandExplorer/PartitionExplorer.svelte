<script lang="ts" context="module">
  import type { Partition } from 'src/channels';
  import type { Node } from '../utility/FileTree/FileTree.svelte';

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
  import FileTree from '../utility/FileTree/FileTree.svelte';
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

  function partitionToNode(partition: Partition): Node<Partition, undefined> {
    return {
      id: partition.id,
      name: partition.name,
      isDirectory: false,
      isDisabled: disabled || !partition.mountable,
      data: partition,
    };
  }

  const root: Node<Partition, undefined, true> = {
    id: 'root',
    name: '<root>',
    isDirectory: true,
    isDisabled: false,
    data: undefined,
  };
</script>

<FileTree
  class={cls}
  {root}
  loadDirectory={async () => partitions.map(partitionToNode)}
  onFileClick={disabled ? undefined : onPartitionChoose}
>
  {#snippet icon()}
    <CircleStackIcon class="inline-block h-4 text-red-300" />
  {/snippet}
  {#snippet fileExtra(node)}
    <ActionButtons>
      {#if !node.data.mountable}
        <span>unsupported</span>
      {:else if node.id === formattingPartitionId}
        <span class="text-red-500">formatting...</span>
      {:else}
        {node.data.sizeHuman}
        <Tooltip placement="left">
          {#snippet tooltip()}
            <span>Format partition {node.data.name} (<span class="text-red-500">data loss!</span>)</span>
          {/snippet}
          <ActionButton {disabled} onclick={() => handlers.format(node.data)}>
            <TrashIcon class="h-4 cursor-pointer hover:fill-red-500 hover:stroke-2" />
          </ActionButton>
        </Tooltip>
      {/if}
    </ActionButtons>
  {/snippet}
</FileTree>
