<script context="module" lang="ts">
  import { onMount, type Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';

  export type ReloadFn = (id: string) => Promise<void>;
  export interface Props<File = any, Dir = any> extends HTMLAttributes<HTMLUListElement> {
    class?: string;
    disabled?: boolean;

    root: Node<File, Dir, true>;

    onDragDrop?: (target: Node<File, Dir, true>, item: Node<File, Dir> | FileList, reloadDir: ReloadFn) => void;
    onFileClick?: (file: File) => void;
    loadDirectory: (id: string) => Promise<Node<File, Dir>[]>;

    icon?: Snippet<[Node<File, Dir>, ReloadFn]>;
    name?: Snippet<[Node<File, Dir>, ReloadFn]>;
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
    disabled,
    root,
    onDragDrop,
    onFileClick,
    loadDirectory,
    icon,
    name,
    dirExtra,
    fileExtra,
    class: propClass = '',
    ...ulProps
  }: Props<F, D> = $props();

  type NodeId = string;
  const loadingState = $state<Record<NodeId, boolean>>({});
  const expandedState = $state<Record<NodeId, boolean>>({});
  const childrenState = $state<Record<NodeId, string[]>>({});
  const nodes: Record<NodeId, Node<F, D>> = $state({});
  let dragTargetId = $state<string | undefined>();

  onMount(async () => {
    nodes[root.id] = root;
    await handlers.toggleDir(root.id, true);
  });

  const handlers = {
    reloadDir: async (id: string) => {
      const node = nodes[id];
      if (node && node.isDirectory) {
        await handlers.toggleDir(id, true);
      }
    },
    toggleDir: async (id: string, expand?: boolean) => {
      expand ??= !expandedState[id];
      expandedState[id] = expand;
      if (expand) {
        const loadingTimer = setTimeout(() => (loadingState[id] = true), 100);
        await loadDirectory(id)
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

  function getNodeParent(nodeId: string): string | null {
    const entry = Object.entries(childrenState).find(([_, childrenList]) => childrenList.includes(nodeId));
    if (entry) {
      return entry[0];
    }

    return null;
  }

  const DRAG_MIME_TYPE = 'application/text+nxkit';
  const ALLOWED_DRAG_TYPES = [DRAG_MIME_TYPE, 'Files'];
  let draggingId: string | undefined = undefined;
  let draggingEnabled = false;

  const ondragstart = (ev: DragEvent) => {
    if (ev.dataTransfer) {
      const { id } = (ev.target as HTMLLIElement).dataset;
      if (id) {
        draggingEnabled = true;
        draggingId = id;
        ev.dataTransfer.setData(DRAG_MIME_TYPE, id);
      }
    }
  };

  const ondragend = (ev: DragEvent) => {
    draggingId = undefined;
    draggingEnabled = false;
  };

  const ondragenter = (ev: DragEvent) => {
    if (ev.dataTransfer) {
      draggingEnabled = ALLOWED_DRAG_TYPES.some((type) => ev.dataTransfer?.types.includes(type));
    }
  };

  const ondragover = (ev: DragEvent) => {
    if (!onDragDrop || !draggingEnabled) return;
    ev.preventDefault();

    if (ev.dataTransfer) {
      if (ev.dataTransfer.types.includes('Files')) {
        ev.dataTransfer.dropEffect = 'copy';
      } else if (ev.dataTransfer.types.includes(DRAG_MIME_TYPE)) {
        ev.dataTransfer.dropEffect = 'move';
      }
    }

    // find node we're dragging over
    let dom: HTMLElement | null = ev.target as HTMLElement;
    while (dom) {
      const tag = dom.tagName.toLowerCase();
      if (tag === 'li') break;
      if (tag === 'ul') break;
      dom = dom.parentElement;
    }

    if (!dom) return;
    const { id } = dom.dataset;
    if (typeof id !== 'string') return;

    // compute drop target directory
    if (nodes[id].isDirectory) {
      dragTargetId = dom.dataset.id;
    } else {
      const parent = getNodeParent(id);
      if (parent) {
        dragTargetId = parent;
      }
    }

    // if we're dragging an item and it's targeting itself or its parent, don't do anything
    if (draggingId) {
      if (draggingId === dragTargetId) {
        dragTargetId = undefined;
        ev.dataTransfer!.dropEffect = 'none';
      } else if (getNodeParent(draggingId) === dragTargetId) {
        dragTargetId = undefined;
        ev.dataTransfer!.dropEffect = 'none';
      }
    }
  };

  const ondragleave = (_: DragEvent) => {};
  const ondrop = (ev: DragEvent) => {
    ev.preventDefault();

    if (!draggingEnabled) return;
    if (!dragTargetId) return;
    if (!ev.dataTransfer) return;

    const target = nodes[dragTargetId] as Node<F, D, true>;
    if (draggingId) {
      onDragDrop?.(target, nodes[draggingId], handlers.reloadDir);
    } else if (ev.dataTransfer.files.length) {
      onDragDrop?.(target, ev.dataTransfer.files, handlers.reloadDir);
    }

    dragTargetId = undefined;
  };

  const ulClass = 'select-none font-mono m-2 border border-slate-900 bg-slate-900';
  const iconClass = 'inline-block h-4';
  const spanClass = 'grow flex items-center justify-start gap-2';
</script>

{#snippet renderNode(node: Node, depth = 1)}
  {@const loading = loadingState[node.id]}
  {@const expanded = expandedState[node.id]}
  {@const dragging = dragTargetId === node.id}
  {@const isDisabled = node.isDisabled || disabled}
  <li
    data-id={node.id}
    draggable="true"
    class="dark:bg-slate-800 odd:dark:bg-slate-700"
    style="padding-left: {depth}ex;"
  >
    <div
      class="pr-2 flex justify-between items-center focus:outline-none"
      class:text-slate-500={isDisabled}
      class:focus:bg-blue-600={!isDisabled && !dragging}
      class:focus:hover:bg-blue-500={!isDisabled && !dragging}
      class:hover:dark:bg-slate-600={!isDisabled && !dragging}
      class:bg-blue-600={dragging}
      role={isDisabled ? '' : 'button'}
      tabIndex={isDisabled ? -1 : 0}
      aria-disabled={isDisabled}
      onkeypress={!isDisabled ? (e) => e.key === ' ' && handlers.onclick(node) : undefined}
      onclick={!isDisabled ? () => handlers.onclick(node) : undefined}
    >
      {#if node.isDirectory}
        <span class={spanClass}>
          {#if icon}
            {@render icon(node, handlers.reloadDir)}
          {:else if expanded}
            <FolderOpenIcon class="text-blue-300 {iconClass}" />
          {:else}
            <FolderIcon class="text-blue-300 {iconClass}" />
          {/if}
          {#if name}
            {@render name(node, handlers.reloadDir)}
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
        <span class={spanClass}>
          {#if icon}
            {@render icon(node, handlers.reloadDir)}
          {:else if node === LOADING_NODE}
            <ClockIcon class={iconClass} />
          {:else}
            <DocumentIcon class={iconClass} />
          {/if}
          {#if name}
            {@render name(node, handlers.reloadDir)}
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

<ul
  data-id={root.id}
  class="relative {ulClass} {propClass} {dragTargetId === root.id ? 'bg-blue-600' : ''}"
  {ondragstart}
  {ondragend}
  {ondragenter}
  {ondragover}
  {ondragleave}
  {ondrop}
  {...ulProps}
>
  {#if loadingState[root.id]}
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
