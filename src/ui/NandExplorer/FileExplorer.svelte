<script lang="ts" context="module">
  import type { FSDirectory, FSEntry, FSFile } from '../../nand/fatfs/fs';
  import type { Node, ReloadFn } from '../utility/FileTree/FileTree.svelte';

  export type FileNode<B extends boolean = boolean> = Node<FSFile, FSDirectory, B>;

  export function entryToNode(entry: FSEntry): FileNode {
    return {
      id: entry.path,
      name: entry.name,
      isDirectory: entry.type === 'd',
      data: entry,
    };
  }

  export const ROOT_NODE = Object.freeze(
    entryToNode({
      type: 'd',
      path: '/',
      name: '<root>',
    }),
  ) as Node<FSFile, FSDirectory, true>;

  export interface Props {
    class?: string;
    readonly: boolean;
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
  import InputTextInline from '../utility/InputTextInline.svelte';

  let { readonly, class: propClass = '' }: Props = $props();

  let isRenamingId = $state<string | null>(null);
  let disabled = $state(false);

  const handlers = {
    toggleRename: (node: FileNode) => {
      isRenamingId = isRenamingId === node.id ? null : node.id;
    },
    doRename: (node: FileNode, reloadDir: ReloadFn) => async (newName: string) => {
      const parentDir = await window.nxkit.pathDirname(node.data.path);
      const targetPath = await window.nxkit.pathJoin(parentDir, newName);
      await window.nxkit
        .nandMoveEntry(node.data.path, targetPath)
        .then((result) => handleNandResult(result, `rename ${node.data.name}`))
        .finally(() => reloadDir(parentDir));
    },
    moveEntry: async (target: FileNode<true>, item: FileNode, reloadDir: ReloadFn) => {
      disabled = true;
      const srcPath = item.data.path;
      const parentDir = await window.nxkit.pathDirname(srcPath);
      const targetPath = await window.nxkit.pathJoin(target.data.path, item.data.name);

      if (targetPath.startsWith(srcPath)) {
        disabled = false;
        return alert('Cannot move a directory into itself');
      }

      await window.nxkit
        .nandMoveEntry(item.data.path, targetPath)
        .then((result) => handleNandResult(result, `move ${item.data.name}`))
        .finally(() => {
          reloadDir(parentDir);
          reloadDir(target.data.path);
          disabled = false;
        });
    },
    copyFilesIn: async (target: FileNode<true>, files: FileList, reloadDir: ReloadFn) => {
      disabled = true;
      const filePaths: string[] = [];
      for (const file of files) {
        filePaths.push(file.path);
      }

      if (!filePaths.length) {
        disabled = false;
        return;
      }

      const exists = await window.nxkit
        .nandCheckExists(target.data.path, filePaths)
        .then((result) => handleNandResult(result, `copy in ${filePaths.join(', ')}`));

      if (exists) {
        const yes = confirm(
          'This operation will overwrite existing files in the NAND, are you sure you want to continue?',
        );
        if (!yes) {
          disabled = false;
          return;
        }
      }

      await window.nxkit
        .nandCopyFilesIn(target.data.path, filePaths)
        .then((result) => handleNandResult(result, `copy in ${filePaths.join(', ')}`))
        .finally(() => {
          reloadDir(target.data.path);
          disabled = false;
        });
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
      await window.nxkit.nandCopyFileOut(file.path);
    },
    openNandDirectory: async (path: string): Promise<FileNode[]> => {
      const result = handleNandResult(await window.nxkit.nandReaddir(path), `readdir /`);
      return (result ?? []).map(entryToNode);
    },
  };
</script>

{#snippet renderDeleteAction(node: FileNode, reloadDir: ReloadFn)}
  <Tooltip {disabled} placement="left">
    {#snippet tooltip()}
      <span>
        <span class="text-red-500">Delete</span>
        <Code>{node.name}</Code>
      </span>
    {/snippet}
    <ActionButton {disabled} onclick={() => handlers.delete(node, reloadDir)}>
      <TrashIcon class="h-4 cursor-pointer hover:fill-red-500" />
    </ActionButton>
  </Tooltip>
{/snippet}

{#snippet renderRenameAction(node: FileNode, reloadDir: ReloadFn)}
  <Tooltip {disabled} placement="left">
    {#snippet tooltip()}
      <span>Rename <Code>{node.name}</Code></span>
    {/snippet}
    <ActionButton {disabled} onclick={() => handlers.toggleRename(node)}>
      <PencilSquareIcon class="h-4 cursor-pointer hover:stroke-blue-400 hover:fill-blue-400" />
    </ActionButton>
  </Tooltip>
{/snippet}

<FileTree
  {disabled}
  class={propClass}
  loadDirectory={handlers.openNandDirectory}
  root={ROOT_NODE}
  onDragDrop={disabled || readonly
    ? undefined
    : (target, item, reloadDir) => {
        if (item instanceof FileList) {
          handlers.copyFilesIn(target, item, reloadDir);
        } else {
          handlers.moveEntry(target, item, reloadDir);
        }
      }}
>
  {#snippet name(node, reloadDir)}
    {#if isRenamingId === node.id}
      <InputTextInline
        autoFocus
        class="grow mr-2"
        initialValue={node.name}
        onEnter={handlers.doRename(node, reloadDir)}
        onEscape={() => (isRenamingId = null)}
      />
    {:else}
      {node.name}
    {/if}
  {/snippet}
  {#snippet dirExtra(node, reloadDir)}
    <ActionButtons>
      {#if !readonly}
        {@render renderRenameAction(node, reloadDir)}
        {@render renderDeleteAction(node, reloadDir)}
      {/if}
    </ActionButtons>
  {/snippet}
  {#snippet fileExtra(node, reloadDir)}
    <ActionButtons>
      <span class="font-mono">{node.data.sizeHuman}</span>
      <Tooltip {disabled} placement="left">
        {#snippet tooltip()}
          <span>Download <Code>{node.data.name}</Code></span>
        {/snippet}
        <ActionButton {disabled} onclick={() => handlers.downloadFile(node.data)}>
          <ArrowDownTrayIcon class="h-4 cursor-pointer hover:stroke-blue-400 hover:stroke-2" />
        </ActionButton>
      </Tooltip>
      {#if !readonly}
        {@render renderRenameAction(node, reloadDir)}
        {@render renderDeleteAction(node, reloadDir)}
      {/if}
    </ActionButtons>
  {/snippet}
</FileTree>
