<script lang="ts" context="module">
  import type { Writable } from 'svelte/store';
  export interface TabContext {
    selected: Writable<HTMLElement | null>;
  }

  export const TabContextKey = Symbol();
</script>

<script lang="ts">
  import { writable } from 'svelte/store';
  import { setContext } from 'svelte';

  const tabContext = setContext<TabContext>(TabContextKey, { selected: writable(null) });

  function init(node: HTMLElement) {
    const destroy = tabContext.selected.subscribe((x) => {
      if (x) {
        node.replaceChildren(x);
      }
    });

    return { destroy };
  }
</script>

<ul class="text-sm font-medium text-center text-gray-500 shadow flex dark:text-gray-400">
  <slot />
</ul>
<div use:init></div>
