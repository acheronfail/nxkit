<script lang="ts" context="module">
  import { type Snippet } from 'svelte';

  export interface Props {
    children: Snippet;
    class?: string;
  }
</script>

<script lang="ts">
  import { getContext } from 'svelte';
  import { TabContextKey, type TabContext } from './Tabs.svelte';

  let { children, class: propClass = '' }: Props = $props();

  const panel = {};
  const { registerPanel, selectedPanel } = getContext<TabContext>(TabContextKey);
  registerPanel(panel);
</script>

<div class:hidden={$selectedPanel !== panel} class="grow flex flex-col px-4 {propClass}">
  {@render children()}
</div>
