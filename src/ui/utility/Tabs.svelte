<script lang="ts" context="module">
  import type { Writable } from 'svelte/store';
  export interface TabContext {
    selected: Writable<HTMLElement | null>;
  }

  export const TabContextKey = Symbol();

  export interface Props {
    fillContainer?: boolean;
    class?: string;
  }
</script>

<script lang="ts">
  import { writable } from 'svelte/store';
  import { onMount, setContext } from 'svelte';

  let { fillContainer = false, class: cls = '' }: Props = $props();
  let stickyHeader = $state<HTMLElement | null>(null);

  const tabContext = setContext<TabContext>(TabContextKey, {
    selected: writable(null),
  });

  function init(node: HTMLElement) {
    const destroy = tabContext.selected.subscribe((x) => {
      if (x) {
        node.replaceChildren(x);
      }
    });

    return { destroy };
  }

  const fillClass = 'grow flex flex-col h-full w-full relative';

  function onScroll() {
    if (stickyHeader) {
      const needsBorder = window.document.documentElement.scrollTop > 0;
      stickyHeader!.style.borderBottomStyle = needsBorder ? 'solid' : 'none';
    }
  }
  onMount(() => {
    if ($$slots.header && stickyHeader) {
      onScroll();
      window.addEventListener('scroll', onScroll, { passive: true });
      return () => window.removeEventListener('scroll', onScroll);
    }
  });
</script>

<!-- svelte-ignore slot_element_deprecated -->
{#if $$slots.header}
  <div bind:this={stickyHeader} class="sticky top-0 z-50 border-b pb-1 dark:border-slate-900 dark:bg-slate-800">
    <slot name="header" />
    <ul class="px-4 text-sm font-medium text-center text-gray-500 flex dark:text-gray-400">
      <slot />
    </ul>
  </div>
{:else}
  <ul class="text-sm font-medium text-center text-gray-500 flex dark:text-gray-400">
    <slot />
  </ul>
{/if}

<div class="{fillContainer ? fillClass : ''} {cls}" use:init></div>
