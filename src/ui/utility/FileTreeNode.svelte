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

  const handlers = {
    handleDirectoryOpen: () => {
      isExpanded = expandedState[node.id] = !isExpanded;
      const loadingTimer = setTimeout(() => (isLoading = true), 100);

      // TODO: error handling
      openDirectory(node.data)
        .then((nodes) => (children = nodes))
        .finally(() => {
          clearTimeout(loadingTimer);
          isLoading = false;
        });
    },
  };
</script>

<li class="odd:dark:bg-slate-700" style="padding-left: {depth}ex;">
  {#if node.isDirectory}
    <div
      class={itemClass}
      role="button"
      tabindex="0"
      onkeypress={(e) => e.key === ' ' && handlers.handleDirectoryOpen()}
      onclick={handlers.handleDirectoryOpen}
    >
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
    </div>
  {:else}
    <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
    <div
      class={itemClass}
      class:text-slate-600={node === loadingFile}
      role={onFileClick ? 'button' : undefined}
      tabindex={onFileClick ? 0 : -1}
      onkeypress={(e) => e.key === ' ' && onFileClick(node.data)}
      onclick={() => onFileClick(node.data)}
    >
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
      <!-- FIXME: add actions -->
      <!-- <span>
        <Tooltip content="Download">
          <ArrowDownTrayIcon
            onclick={() => handlers.download(node)}
            class="cursor-pointer h-6 p-1 hover:text-black inline-block"
          />
        </Tooltip>
      </span> -->
    </div>
  {/if}
</li>
{#if isExpanded}
  {#if isLoading}
    <svelte:self {openDirectory} node={loadingFile} depth={depth + 1} />
  {:else}
    {#each children as e}
      <svelte:self {openDirectory} node={e} depth={depth + 1} />
    {/each}
  {/if}
{/if}

<style>
  li > div {
    padding-left: 100%;
    margin-left: -100%;
  }
</style>
