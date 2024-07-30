<script lang="ts" context="module">
  import type { Snippet } from 'svelte';

  export const TabContextKey = {};

  export type TabType = {};
  export interface TabContext {
    registerTab: (tab: TabType) => void;
    registerPanel: (panel: TabType) => void;
    selectTab: (tab: TabType) => void;
    selectTabByIndex: (index: number) => void;
    selectedTab: Writable<TabType | null>;
    selectedPanel: Writable<TabType | null>;
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

  const tabs: TabType[] = [];
  const panels: TabType[] = [];
  const selectedTab = writable<TabType | null>(null);
  const selectedPanel = writable<TabType | null>(null);

  const context = setContext<TabContext>(TabContextKey, {
    registerTab: (tab: TabType) => {
      tabs.push(tab);
      selectedTab.update((current: TabType | null) => current || tab);

      onDestroy(() => {
        const i = tabs.indexOf(tab);
        tabs.splice(i, 1);
        selectedTab.update((current: TabType | null) => (current === tab ? tabs[i] || tabs[tabs.length - 1] : current));
      });
    },

    registerPanel: (panel: TabType) => {
      panels.push(panel);
      selectedPanel.update((current: TabType | null) => current || panel);

      onDestroy(() => {
        const i = panels.indexOf(panel);
        panels.splice(i, 1);
        selectedPanel.update((current: TabType | null) =>
          current === panel ? panels[i] || panels[panels.length - 1] : current,
        );
      });
    },

    selectTab: (tab: TabType) => {
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
