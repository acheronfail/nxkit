<script context="module" lang="ts">
  import type { Snippet } from 'svelte';

  export interface Node<FileData = any, DirData = any, IsDirectory extends boolean = boolean> {
    id: string;
    name: string;
    isDirectory: IsDirectory;
    isDisabled?: boolean;
    data?: IsDirectory extends true ? DirData : FileData;
  }

  const loadingFile: Node = {
    id: 'LOADING',
    name: 'Loading...',
    isDirectory: false,
    isDisabled: true,
  };

  export interface Snippets<FileData = any, DirData = any> {
    icon?: Snippet<
      [
        {
          node: Node<FileData, DirData>;
          iconClass: string;
        },
      ]
    >;
    name?: Snippet<[Node<FileData, DirData>]>;
    dirExtra?: Snippet<[DirData]>;
    fileExtra?: Snippet<[FileData]>;
  }

  interface CommonProps<FileData = any, DirData = any> extends Snippets<FileData, DirData> {
    node: Node<FileData, DirData>;
    depth: number;
    isExpanded?: boolean;
    onFileClick?: (data: FileData) => void;
  }

  export type Props<FileData = any, DirData = any> = DirData extends never
    ? CommonProps & { openDirectory?: undefined }
    : CommonProps & {
        openDirectory: (dir: DirData) => Promise<Node<FileData, DirData>[]>;
      };
</script>

<script lang="ts">
  import { FolderIcon, DocumentIcon, ClockIcon } from 'heroicons-svelte/24/outline';
  import { FolderOpenIcon } from 'heroicons-svelte/24/solid';

  // Record<id, isExpanded>
  const expandedState: Record<string, boolean> = {};

  let {
    node,
    depth,
    onFileClick,
    openDirectory,
    name,
    icon,
    dirExtra,
    fileExtra,
    isExpanded = expandedState[node.id] ?? false,
  }: Props = $props();
  let isLoading = $state(false);
  let children = $state<Node[] | null>(null);

  const handleDirectoryOpen = () => {
    isExpanded = expandedState[node.id] = !isExpanded;
    const loadingTimer = setTimeout(() => (isLoading = true), 100);

    openDirectory?.(node.data)
      .then((nodes) => (children = nodes))
      .catch((err) => alert(`Failed to open "${node.id}": ${String(err)}`))
      .finally(() => {
        clearTimeout(loadingTimer);
        isLoading = false;
      });
  };

  const handler = () => (node.isDirectory ? handleDirectoryOpen() : onFileClick?.(node.data));

  const iconClass = 'inline-block h-4';
  const itemClass = 'pr-2 flex justify-between items-center focus:outline-none';
</script>

<li class="dark:bg-slate-800 odd:dark:bg-slate-700" style="padding-left: {depth}ex;">
  <div
    class={itemClass}
    class:text-slate-500={node.isDisabled}
    class:focus:bg-blue-600={!node.isDisabled}
    class:focus:hover:bg-blue-500={!node.isDisabled}
    class:hover:dark:bg-slate-600={!node.isDisabled}
    role={node.isDisabled ? '' : 'button'}
    tabIndex={node.isDisabled ? -1 : 0}
    aria-disabled={node.isDisabled}
    onkeypress={!node.isDisabled ? (e) => e.key === ' ' && handler() : undefined}
    onclick={!node.isDisabled ? () => handler() : undefined}
  >
    {#if node.isDirectory}
      <span>
        {#if icon}
          {@render icon({ node, iconClass })}
        {:else if isExpanded}
          <FolderOpenIcon class="text-blue-300 {iconClass}" />
        {:else}
          <FolderIcon class="text-blue-300 {iconClass}" />
        {/if}
        {#if name}
          {@render name(node)}
        {:else}
          {node.name}
        {/if}
      </span>
      <span>
        {#if dirExtra}
          {@render dirExtra(node.data)}
        {/if}
      </span>
    {:else}
      <span>
        {#if icon}
          {@render icon({ node, iconClass })}
        {:else if node === loadingFile}
          <ClockIcon class={iconClass} />
        {:else}
          <DocumentIcon class={iconClass} />
        {/if}
        {#if name}
          {@render name(node)}
        {:else}
          {node.name}
        {/if}
      </span>
      <span>
        {#if fileExtra}
          {@render fileExtra(node.data)}
        {/if}
      </span>
    {/if}
  </div>
</li>
{#if isExpanded}
  {#if isLoading}
    <svelte:self {openDirectory} node={loadingFile} depth={depth + 1} />
  {:else if children}
    {#each children as e}
      <svelte:self {openDirectory} {fileExtra} {dirExtra} {icon} node={e} depth={depth + 1} />
    {/each}
  {/if}
{/if}

<style>
  li > div {
    padding-left: 100%;
    margin-left: -100%;
  }
</style>
