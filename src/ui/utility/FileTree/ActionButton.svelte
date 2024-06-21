<script lang="ts" context="module">
  import type { Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';

  export interface Props extends HTMLAttributes<HTMLElement> {
    disabled?: boolean;
    onclick: () => void;
    children: Snippet;
  }
</script>

<script lang="ts">
  let { children, onclick, disabled = false, ...rest }: Props = $props();

  function handler(e: Event) {
    e.stopPropagation();
    onclick();
  }
</script>

{#if disabled}
  <span>
    {@render children()}
  </span>
{:else}
  <span
    role="button"
    tabindex="0"
    onkeypress={(e) => e.key === ' ' && handler(e)}
    onclick={(e) => handler(e)}
    {...rest}
  >
    {@render children()}
  </span>
{/if}
