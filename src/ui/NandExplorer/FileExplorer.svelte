<script lang="ts" context="module">
  import type { FSDirectory, FSEntry, FSFile } from '../../nand/fatfs/fs';
  import type { Node } from '../utility/FileTree/FileTreeNode.svelte';

  export function entryToNode(entry: FSEntry): Node<FSDirectory, FSFile> {
    return {
      id: entry.path,
      name: entry.name,
      isDirectory: entry.type === 'd',
      data: entry,
    };
  }

  export interface Props {
    rootEntries: FSEntry[];
    class?: string;
  }
</script>

<script lang="ts">
  import Tooltip from '../utility/Tooltip.svelte';

  import { ArrowDownTrayIcon, PencilSquareIcon, TrashIcon } from 'heroicons-svelte/24/solid';
  import FileTreeRoot from '../utility/FileTree/FileTreeRoot.svelte';
  import ActionButton from '../utility/FileTree/ActionButton.svelte';
  import ActionButtons from '../utility/FileTree/ActionButtons.svelte';
  import { NandError } from '../../channels';
  import Code from '../utility/Code.svelte';
  import { handleNandResult } from '../errors';

  let { rootEntries, class: cls = '' }: Props = $props();

  let isRenamingId = $state<string | null>(null);

  // TODO build it out and enable
  const renamingEnabled = false;

  const handlers = {
    toggleRename: (entry: FSEntry) => {
      const node = entryToNode(entry);
      isRenamingId = isRenamingId === node.id ? null : node.id;
    },
    doRename: async (entry: FSEntry) => {
      // TODO: prompt for name
      await window.nxkit
        .nandMoveEntry(entry.path, entry.path == '/PRF2SAFE.RCV' ? '/newname' : '/PRF2SAFE.RCV')
        .then((result) => handleNandResult(result, `rename ${entry.name}`))
        .finally(() => {
          // TODO: reload view!
        });
    },
    delete: async (entry: FSEntry) => {
      const yes = confirm(`Are you sure you want to delete ${entry.name}?\n\nThis action cannot be undone!`);
      if (yes) {
        await window.nxkit
          .nandDeleteEntry(entry.path)
          .then((result) => handleNandResult(result, `delete ${entry.name}`))
          .finally(() => {
            // TODO: reload view!
          });
      }
    },
    downloadFile: async (file: FSFile) => {
      await window.nxkit.nandCopyFile(file.path);
    },
    openNandDirectory: async (dir: FSDirectory): Promise<Node<FSDirectory, FSFile>[]> => {
      return window.nxkit.nandReaddir(dir.path).then((result) => {
        if (result.error === NandError.None) {
          return result.data.map(entryToNode);
        }

        console.error(`Failed to readdir: ${result.error}`);
        return [];
      });
    },
  };
</script>

{#snippet renderDeleteAction(entry)}
  <Tooltip placement="left">
    {#snippet tooltip()}
      <span><span class="text-red-500">Delete</span> <Code>{entry.name}</Code></span>
    {/snippet}
    <ActionButton onclick={() => handlers.delete(entry)}>
      <TrashIcon class="h-4 cursor-pointer hover:fill-red-500" />
    </ActionButton>
  </Tooltip>
{/snippet}

{#snippet renderRenameAction(entry)}
  {#if renamingEnabled}
    <Tooltip placement="left">
      {#snippet tooltip()}
        <span>Rename <Code>{entry.name}</Code></span>
      {/snippet}
      <ActionButton onclick={() => handlers.toggleRename(entry)}>
        <PencilSquareIcon class="h-4 cursor-pointer hover:fill-slate-900" />
      </ActionButton>
    </Tooltip>
  {/if}
{/snippet}

<FileTreeRoot class={cls} nodes={rootEntries.map(entryToNode)} openDirectory={handlers.openNandDirectory}>
  {#snippet name(node)}
    {#if isRenamingId === node.id}
      TODO: editable name here
    {:else}
      {node.name}
    {/if}
  {/snippet}
  {#snippet dirExtra(dir)}
    <ActionButtons>
      {@render renderRenameAction(dir)}
      {@render renderDeleteAction(dir)}
    </ActionButtons>
  {/snippet}
  {#snippet fileExtra(file)}
    <ActionButtons>
      <span class="font-mono">{file.sizeHuman}</span>
      <Tooltip placement="left">
        {#snippet tooltip()}
          <span>Download <Code>{file.name}</Code></span>
        {/snippet}
        <ActionButton onclick={() => handlers.downloadFile(file)}>
          <ArrowDownTrayIcon class="h-4 cursor-pointer hover:stroke-slate-900 hover:stroke-2" />
        </ActionButton>
      </Tooltip>
      {@render renderRenameAction(file)}
      {@render renderDeleteAction(file)}
    </ActionButtons>
  {/snippet}
</FileTreeRoot>
