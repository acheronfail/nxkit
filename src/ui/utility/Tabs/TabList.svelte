<script lang="ts" context="module">
  import type { Snippet } from 'svelte';
  export interface Props {
    children: Snippet;
    header?: Snippet;
    class?: string;
  }
</script>

<script lang="ts">
  import { onMount } from 'svelte';

  let { children, header, class: propClass = '' }: Props = $props();
  let stickyHeader = $state<HTMLElement | null>(null);

  function onScroll() {
    if (stickyHeader) {
      const needsBorder = window.document.documentElement.scrollTop > 0;
      stickyHeader.style.borderBottomStyle = needsBorder ? 'solid' : 'none';
      stickyHeader.classList[needsBorder ? 'add' : 'remove'](shadowClass);
    }
  }

  onMount(() => {
    if (header && stickyHeader) {
      onScroll();
      window.addEventListener('scroll', onScroll, { passive: true });
      return () => window.removeEventListener('scroll', onScroll);
    }
  });

  const shadowClass = 'shadow-lg';
</script>

{#if header}
  <div
    data-testid="tablist"
    bind:this={stickyHeader}
    class="sticky {shadowClass} top-0 z-50 border-b pb-1 dark:border-slate-900 dark:bg-slate-800"
  >
    {@render header()}
    <ul class="px-4 text-sm font-medium text-center text-gray-500 flex dark:text-gray-400 {propClass}">
      {@render children()}
    </ul>
  </div>
{:else}
  <ul data-testid="tablist" class="text-sm font-medium text-center text-gray-500 flex dark:text-gray-400 {propClass}">
    {@render children()}
  </ul>
{/if}
