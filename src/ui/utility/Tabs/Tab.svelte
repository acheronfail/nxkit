<script lang="ts">
  import { getContext } from 'svelte';
  import { TabContextKey, type TabContext } from './Tabs.svelte';

  const tab = {};
  const { registerTab, selectTab, selectedTab } = getContext<TabContext>(TabContextKey);

  registerTab(tab);

  const liClass =
    'w-full overflow-hidden focus-within:z-10 focus-within:ring-2 focus-within:ring-blue-500 border first:border-l border-l-0 first:rounded-l-lg last:rounded-r-lg border-slate-200 dark:border-slate-900';
  const buttonClass =
    'focus:outline-none inline-block w-full p-4 text-gray-900 bg-gray-100 dark:bg-gray-700 dark:text-white flex flex-row justify-center items-center';
</script>

<li data-testid="tab" class={liClass}>
  <button class:dark:bg-gray-800={$selectedTab === tab} class={buttonClass} on:click={() => selectTab(tab)}>
    <slot />
  </button>
</li>
