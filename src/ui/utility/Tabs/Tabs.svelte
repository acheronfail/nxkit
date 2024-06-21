<script lang="ts" context="module">
  import type { Snippet } from 'svelte';

  export const TabContextKey = {};
  export interface TabContext {
    registerTab: (tab: {}) => void;
    registerPanel: (panel: {}) => void;
    selectTab: (tab: {}) => void;
    selectTabByIndex: (index: number) => void;
    selectedTab: Writable<{} | null>;
    selectedPanel: Writable<{} | null>;
  }

  export interface Props {
    children: Snippet;
    selected?: number;
  }
</script>

<script lang="ts">
  import { setContext, onDestroy } from 'svelte';
  import { type Writable, writable } from 'svelte/store';

  let { children, selected = $bindable(0) }: Props = $props();

  const tabs: {}[] = [];
  const panels: {}[] = [];
  const selectedTab = writable<{} | null>(null);
  const selectedPanel = writable<{} | null>(null);

  const context = setContext<TabContext>(TabContextKey, {
    registerTab: (tab: {}) => {
      tabs.push(tab);
      selectedTab.update((current) => current || tab);

      onDestroy(() => {
        const i = tabs.indexOf(tab);
        tabs.splice(i, 1);
        selectedTab.update((current) => (current === tab ? tabs[i] || tabs[tabs.length - 1] : current));
      });
    },

    registerPanel: (panel: {}) => {
      panels.push(panel);
      selectedPanel.update((current) => current || panel);

      onDestroy(() => {
        const i = panels.indexOf(panel);
        panels.splice(i, 1);
        selectedPanel.update((current) => (current === panel ? panels[i] || panels[panels.length - 1] : current));
      });
    },

    selectTab: (tab: {}) => {
      const i = tabs.indexOf(tab);
      selectedTab.set(tab);
      selectedPanel.set(panels[i]);
    },

    selectTabByIndex: (i: number) => {
      const idx = Math.min(tabs.length - 1, Math.max(i, 0));
      selectedTab.set(tabs[idx]);
      selectedPanel.set(panels[idx]);
    },

    selectedTab,
    selectedPanel,
  });

  $effect(() => {
    if (typeof selected === 'number') {
      context.selectTabByIndex(selected);
    }
  });
</script>

{@render children()}
