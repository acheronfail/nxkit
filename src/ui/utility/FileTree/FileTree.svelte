<script context="module" lang="ts">
  import type { Snippet } from 'svelte';

  export type Props<File = any, Dir = any> = {
    class?: string;
    rootNodes: Node<File, Dir>[];

    onFileClick?: (file: File) => void;
    loadDirectory?: (dir: Dir) => Promise<Node<File, Dir>[]>;

    icon?: Snippet<[Node<File, Dir>]>;
    name?: Snippet<[Node<File, Dir>]>;
    dirExtra?: Snippet<[Dir, Node<File, Dir>]>;
    fileExtra?: Snippet<[File, Node<File, Dir>]>;
  };

  export interface Node<File = any, Dir = any, IsDirectory extends boolean = boolean> {
    id: string;
    name: string;
    isDirectory: IsDirectory;
    isDisabled?: boolean;
    data?: IsDirectory extends true ? Dir : File;
  }

  const LOADING_NODE: Node = {
    id: 'LOADING',
    name: 'Loading...',
    isDirectory: false,
    isDisabled: true,
  };
</script>

<script lang="ts" generics="File, Dir">
  import { ClockIcon, DocumentIcon, FolderIcon } from 'heroicons-svelte/24/outline';
  import { FolderOpenIcon } from 'heroicons-svelte/24/solid';

  let {
    rootNodes,
    onFileClick: openFile,
    loadDirectory: openDirectory,
    icon,
    name,
    dirExtra,
    fileExtra,
    class: propClass = '',
  }: Props<File, Dir> = $props();

  type NodeId = string;
  const expandedState = $state<Record<NodeId, boolean>>({});
  const loadingState = $state<Record<NodeId, boolean>>({});
  const childrenState = $state<Record<NodeId, string[]>>({});
  const nodes: Record<NodeId, Node<File, Dir>> = $state({});

  const handlers = {
    onclick: (node: Node<File, Dir>) => {
      if (node.isDirectory) {
        if (expandedState[node.id]) {
          expandedState[node.id] = false;
        } else {
          expandedState[node.id] = true;
          loadingState[node.id] = true;
          openDirectory?.(node.data as Dir)
            .then((childNodes) => {
              childrenState[node.id] = childNodes.map((node) => node.id);
              childNodes.forEach((node) => (nodes[node.id] = node));
            })
            .finally(() => {
              loadingState[node.id] = false;
            });
        }
      } else {
        openFile?.(node.data as File);
      }
    },
  };

  const iconClass = 'inline-block h-4';
  const itemClass = 'pr-2 flex justify-between items-center focus:outline-none';
</script>

{#snippet renderNode(node: Node, depth = 0)}
  {@const expanded = expandedState[node.id]}
  {@const loading = loadingState[node.id]}
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
      onkeypress={!node.isDisabled ? (e) => e.key === ' ' && handlers.onclick(node) : undefined}
      onclick={!node.isDisabled ? () => handlers.onclick(node) : undefined}
    >
      {#if node.isDirectory}
        <span>
          {#if icon}
            {@render icon(node)}
          {:else if expanded}
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
            {@render dirExtra(node.data, node)}
          {/if}
        </span>
      {:else}
        <span>
          {#if icon}
            {@render icon(node)}
          {:else if node === LOADING_NODE}
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
          {#if fileExtra && node !== LOADING_NODE}
            {@render fileExtra(node.data, node)}
          {/if}
        </span>
      {/if}
    </div>
  </li>
  {#if expanded}
    {#if loading}
      {@render renderNode(LOADING_NODE, depth + 1)}
    {:else}
      {#each childrenState[node.id] ?? [] as id}
        {@render renderNode(nodes[id], depth + 1)}
      {/each}
    {/if}
  {/if}
{/snippet}

<ul class="select-none font-mono m-2 border border-slate-900 bg-slate-900 {propClass}">
  {#each rootNodes as node}
    {@render renderNode(node, 1)}
  {/each}
</ul>

<style>
  li > div {
    padding-left: 100%;
    margin-left: -100%;
  }
</style>
