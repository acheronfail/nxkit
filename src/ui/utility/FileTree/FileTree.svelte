<script context="module" lang="ts">
  import { onMount, type Snippet } from 'svelte';

  export type ReloadFn = (id: string) => Promise<void>;
  export interface Props<File = any, Dir = any> {
    class?: string;

    root: Node<File, Dir, true> | Node<File, Dir>[];

    onFileClick?: (file: File) => void;
    loadDirectory?: (id: string) => Promise<Node<File, Dir>[]>;

    icon?: Snippet<[Node<File, Dir>]>;
    name?: Snippet<[Node<File, Dir>]>;
    dirExtra?: Snippet<[Node<File, Dir, true>, ReloadFn]>;
    fileExtra?: Snippet<[Node<File, Dir, false>, ReloadFn]>;
  }

  export interface Node<File = any, Dir = any, IsDirectory extends boolean = boolean> {
    id: string;
    name: string;
    isDirectory: IsDirectory;
    isDisabled?: boolean;
    data: IsDirectory extends true ? Dir : File;
  }

  const LOADING_NODE: Node = {
    id: 'LOADING',
    name: 'Loading...',
    isDirectory: false,
    isDisabled: true,
    data: undefined,
  };
</script>

<script lang="ts" generics="F, D">
  import { ClockIcon, DocumentIcon, FolderIcon } from 'heroicons-svelte/24/outline';
  import { FolderOpenIcon } from 'heroicons-svelte/24/solid';

  let {
    root,
    onFileClick,
    loadDirectory,
    icon,
    name,
    dirExtra,
    fileExtra,
    class: propClass = '',
  }: Props<F, D> = $props();

  type NodeId = string;
  const expandedState = $state<Record<NodeId, boolean>>({});
  const loadingState = $state<Record<NodeId, boolean>>({});
  const childrenState = $state<Record<NodeId, string[]>>({});
  const nodes: Record<NodeId, Node<F, D>> = $state({});

  onMount(async () => {
    if (Array.isArray(root)) {
      root.forEach((node) => (nodes[node.id] = node));
    } else {
      nodes[root.id] = root;
      await handlers.toggleDir(root.id, true);
    }
  });

  const handlers = {
    reloadDir: async (id: string) => {
      const node = nodes[id];
      if (node && node.isDirectory) {
        await handlers.toggleDir(id, true);
      }
    },
    reloadParent: async () => {
      console.log('TODO reload');
    },
    toggleDir: async (id: string, expand?: boolean) => {
      console.log({ id, nodes: nodes[id] });

      expand ??= !expandedState[id];
      expandedState[id] = expand;
      if (expand) {
        const loadingTimer = setTimeout(() => (loadingState[id] = true), 100);
        await loadDirectory?.(id)
          .then((childNodes) => {
            childrenState[id] = childNodes.map((node) => node.id);
            childNodes.forEach((node) => (nodes[node.id] = node));
          })
          .finally(() => {
            clearTimeout(loadingTimer);
            loadingState[id] = false;
          });
      }
    },
    onclick: (node: Node<F, D>) => {
      if (node.isDirectory) {
        handlers.toggleDir(node.id);
      } else {
        onFileClick?.(node.data as F);
      }
    },
  };

  const iconClass = 'inline-block h-4';
  const itemClass = 'pr-2 flex justify-between items-center focus:outline-none';
</script>

{#snippet renderNode(node: Node, depth = 1)}
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
            {@render dirExtra(node as Node<F, D, true>, handlers.reloadDir)}
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
            {@render fileExtra(node as Node<F, D, false>, handlers.reloadDir)}
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
  {#if Array.isArray(root)}
    {#each root as node}
      {@render renderNode(node, 1)}
    {/each}
  {:else if loadingState[root.id]}
    {@render renderNode(LOADING_NODE, 1)}
  {:else}
    {#each childrenState[root.id] ?? [] as id}
      {@render renderNode(nodes[id], 1)}
    {/each}
  {/if}
</ul>

<style>
  li > div {
    padding-left: 100%;
    margin-left: -100%;
  }
</style>
