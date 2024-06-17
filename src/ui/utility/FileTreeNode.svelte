<script context="module" lang="ts">
  import type { Component } from 'svelte';

  export interface Node<FileData = any, DirData = any, IsDirectory extends boolean = boolean> {
    id: string;
    name: string;
    isDirectory: IsDirectory;
    data?: IsDirectory extends true ? DirData : FileData;
    icon?: Component;
  }

  const loadingFile: Node = {
    id: 'LOADING',
    name: 'Loading...',
    isDirectory: false,
  };

  interface CommonProps<FileData = any, DirData = any> {
    node: Node<FileData, DirData>;
    depth: number;
    isExpanded?: boolean;
    onFileClick?: (data: FileData) => void;
  }

  export type Props<FileData = any, DirData = any> = DirData extends never
    ? CommonProps & { openDirectory?: undefined }
    : CommonProps & { openDirectory: (dir: DirData) => Promise<Node<FileData, DirData>[]> };
</script>

<script lang="ts">
  import { FolderIcon, DocumentIcon, ClockIcon } from 'heroicons-svelte/24/outline';
  import { FolderOpenIcon } from 'heroicons-svelte/24/solid';

  const iconClass = 'inline-block h-4';
  const itemClass =
    'pr-2 flex justify-between items-center hover:dark:bg-slate-600 focus:outline-none focus:bg-blue-500';

  // Record<id, isExpanded>
  const expandedState: Record<string, boolean> = {};

  let { node, depth, onFileClick, openDirectory, isExpanded = expandedState[node.id] ?? false }: Props = $props();
  let isLoading = $state(false);
  let children = $state<Node[] | null>(null);

  const handleDirectoryOpen = () => {
    isExpanded = expandedState[node.id] = !isExpanded;
    const loadingTimer = setTimeout(() => (isLoading = true), 100);

    // TODO: error handling
    openDirectory(node.data)
      .then((nodes) => (children = nodes))
      .finally(() => {
        clearTimeout(loadingTimer);
        isLoading = false;
      });
  };

  const handler = () => (node.isDirectory ? handleDirectoryOpen() : onFileClick?.(node.data));
</script>

<li class="odd:dark:bg-slate-700" style="padding-left: {depth}ex;">
  <div
    class={itemClass}
    class:text-slate-600={node === loadingFile}
    role="button"
    tabindex="0"
    onkeypress={(e) => e.key === ' ' && handler()}
    onclick={() => handler()}
  >
    {#if node.isDirectory}
      <span>
        {#if node.icon}
          <svelte:component this={node.icon} class="text-blue-300 {iconClass}" />
        {:else if isExpanded}
          <FolderOpenIcon class="text-blue-300 {iconClass}" />
        {:else}
          <FolderIcon class="text-blue-300 {iconClass}" />
        {/if}
        {node.name}
      </span>
    {:else}
      <span>
        {#if node.icon}
          <svelte:component this={node.icon} class={iconClass} />
        {:else if node === loadingFile}
          <ClockIcon class={iconClass} />
        {:else}
          <DocumentIcon class={iconClass} />
        {/if}
        {node.name}
      </span>
      <!-- svelte-ignore slot_element_deprecated -->
      <span>
        <slot name="file-extra" file={node.data} />
      </span>
    {/if}
  </div>
</li>
{#if isExpanded}
  {#if isLoading}
    <svelte:self {openDirectory} node={loadingFile} depth={depth + 1} />
  {:else}
    {#each children as e}
      <!-- svelte-ignore slot_element_deprecated -->
      <svelte:self {openDirectory} node={e} depth={depth + 1}>
        <slot slot="file-extra" name="file-extra" let:file {file} />
      </svelte:self>
    {/each}
  {/if}
{/if}

<style>
  li > div {
    padding-left: 100%;
    margin-left: -100%;
  }
</style>
