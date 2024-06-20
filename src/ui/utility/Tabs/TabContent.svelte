<script lang="ts" context="module">
  export interface Props {
    class?: string;
  }
</script>

<script lang="ts">
  import { getContext } from 'svelte';
  import { TabContextKey, type TabContext } from './Tabs.svelte';

  let { class: propClass = '' }: Props = $props();

  const panel = {};
  const { registerPanel, selectedPanel } = getContext<TabContext>(TabContextKey);
  registerPanel(panel);
</script>

<!-- svelte-ignore slot_element_deprecated -->
{#if $selectedPanel === panel}
  <div class="grow flex flex-col px-4 {propClass}">
    <slot />
  </div>
{/if}
