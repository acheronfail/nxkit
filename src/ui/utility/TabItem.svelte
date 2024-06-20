<script lang="ts">
  import { getContext } from 'svelte';
  import { type TabContext, TabContextKey } from './Tabs.svelte';

  export let defaultOpen = false;

  // FIXME: hot reloads in Tab are broken with this...
  const { selected } = getContext<TabContext>(TabContextKey);
  let open = defaultOpen;

  function init(node: HTMLElement) {
    selected.set(node);
    const destroy = selected.subscribe((x) => (open = x === node));
    return { destroy };
  }
</script>

<li
  class="w-full overflow-hidden focus-within:z-40 focus-within:ring-2 focus-within:ring-blue-500 border first:border-l border-l-0 first:rounded-l-lg last:rounded-r-lg border-slate-200 dark:border-slate-900"
  role="presentation"
>
  <button
    class:dark:bg-gray-800={open}
    class="focus:outline-none inline-block w-full p-4 text-gray-900 bg-gray-100 dark:bg-gray-700 dark:text-white"
    type="button"
    on:click={() => (open = true)}
  >
    <slot name="label" />
  </button>

  {#if open}
    <div class="grow flex flex-col w-full h-full" use:init>
      <slot name="content" />
    </div>
  {/if}
</li>
