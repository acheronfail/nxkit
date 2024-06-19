<script lang="ts" context="module">
  import type { Writable } from 'svelte/store';
  export interface TabContext {
    selected: Writable<HTMLElement | null>;
  }

  export const TabContextKey = Symbol();

  export interface Props {
    fillContainer?: boolean;
  }
</script>

<script lang="ts">
  import { writable } from 'svelte/store';
  import { setContext } from 'svelte';

  let { fillContainer = false }: Props = $props();

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

  const fillClass = 'h-full w-full relative';
</script>

<!-- svelte-ignore slot_element_deprecated -->
<ul class="text-sm font-medium text-center text-gray-500 shadow flex dark:text-gray-400">
  <slot />
</ul>
<div class={fillContainer ? fillClass : ''} use:init></div>
