<script lang="ts" context="module">
  import type { FSDirectory, FSEntry, FSFile } from '../../nand/fatfs/fs';
  import type { Node, ReloadFn } from '../utility/FileTree/FileTree.svelte';

  type FileNode = Node<FSFile, FSDirectory>;

  export function entryToNode(entry: FSEntry): FileNode {
    return {
      id: entry.path,
      name: entry.name,
      isDirectory: entry.type === 'd',
      data: entry,
    };
  }

  export interface Props {
    class?: string;
  }
</script>

<script lang="ts">
  import Tooltip from '../utility/Tooltip.svelte';

  import { ArrowDownTrayIcon, PencilSquareIcon, TrashIcon } from 'heroicons-svelte/24/solid';
  import ActionButton from '../utility/FileTree/ActionButton.svelte';
  import ActionButtons from '../utility/FileTree/ActionButtons.svelte';
  import Code from '../utility/Code.svelte';
  import { handleNandResult } from '../errors';
  import FileTree from '../utility/FileTree/FileTree.svelte';

  let { class: propClass = '' }: Props = $props();

  let isRenamingId = $state<string | null>(null);

  const root = entryToNode({
    type: 'd',
    path: '/',
    name: '<root>',
  }) as Node<FSFile, FSDirectory, true>;

  // TODO build it out and enable
  const renamingEnabled = false;

  const handlers = {
    toggleRename: (node: FileNode) => {
      isRenamingId = isRenamingId === node.id ? null : node.id;
    },
    doRename: async (node: FileNode, reloadDir: ReloadFn) => {
      // TODO: prompt for name somehow
      const parentDir = await window.nxkit.pathDirname(node.data.path);
      await window.nxkit
        .nandMoveEntry(node.data.path, node.data.path == '/PRF2SAFE.RCV' ? '/newname' : '/PRF2SAFE.RCV')
        .then((result) => handleNandResult(result, `rename ${node.data.name}`))
        .finally(() => reloadDir(parentDir));
    },
    delete: async (node: FileNode, reloadDir: ReloadFn) => {
      const yes = confirm(`Are you sure you want to delete ${node.data.name}?\n\nThis action cannot be undone!`);
      if (yes) {
        const parentDir = await window.nxkit.pathDirname(node.data.path);
        await window.nxkit
          .nandDeleteEntry(node.data.path)
          .then((result) => handleNandResult(result, `delete ${node.data.name}`))
          .finally(() => reloadDir(parentDir));
      }
    },
    downloadFile: async (file: FSFile) => {
      await window.nxkit.nandCopyFile(file.path);
    },
    openNandDirectory: async (path: string): Promise<FileNode[]> => {
      const result = handleNandResult(await window.nxkit.nandReaddir(path), `readdir /`);
      return (result ?? []).map(entryToNode);
    },
  };
</script>

{#snippet renderDeleteAction(node: FileNode, reloadDir: ReloadFn)}
  <Tooltip placement="left">
    {#snippet tooltip()}
      <span>
        <span class="text-red-500">Delete</span>
        <Code>{node.name}</Code>
      </span>
    {/snippet}
    <ActionButton onclick={() => handlers.delete(node, reloadDir)}>
      <TrashIcon class="h-4 cursor-pointer hover:fill-red-500" />
    </ActionButton>
  </Tooltip>
{/snippet}

{#snippet renderRenameAction(node: FileNode, reloadDir: ReloadFn)}
  {#if renamingEnabled}
    <Tooltip placement="left">
      {#snippet tooltip()}
        <span>Rename <Code>{node.name}</Code></span>
      {/snippet}
      <ActionButton onclick={() => handlers.toggleRename(node)}>
        <PencilSquareIcon class="h-4 cursor-pointer hover:fill-slate-900" />
      </ActionButton>
    </Tooltip>
  {/if}
{/snippet}

<FileTree class={propClass} {root} loadDirectory={handlers.openNandDirectory}>
  {#snippet name(node)}
    {#if isRenamingId === node.id}
      TODO: editable name here
    {:else}
      {node.name}
    {/if}
  {/snippet}
  {#snippet dirExtra(node, reloadDir)}
    <ActionButtons>
      {@render renderRenameAction(node, reloadDir)}
      {@render renderDeleteAction(node, reloadDir)}
    </ActionButtons>
  {/snippet}
  {#snippet fileExtra(node, reloadDir)}
    <ActionButtons>
      <span class="font-mono">{node.data.sizeHuman}</span>
      <Tooltip placement="left">
        {#snippet tooltip()}
          <span>Download <Code>{node.data.name}</Code></span>
        {/snippet}
        <ActionButton onclick={() => handlers.downloadFile(node.data)}>
          <ArrowDownTrayIcon class="h-4 cursor-pointer hover:stroke-slate-900 hover:stroke-2" />
        </ActionButton>
      </Tooltip>
      {@render renderRenameAction(node, reloadDir)}
      {@render renderDeleteAction(node, reloadDir)}
    </ActionButtons>
  {/snippet}
</FileTree>
