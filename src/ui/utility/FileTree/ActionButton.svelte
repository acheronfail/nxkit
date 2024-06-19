<script lang="ts" context="module">
  import type { HTMLAttributes } from 'svelte/elements';

  export interface Props extends HTMLAttributes<HTMLElement> {
    disabled?: boolean;
    onclick: () => void;
  }
</script>

<script lang="ts">
  let { onclick, disabled = false, ...rest }: Props = $props();

  function handler(e: Event) {
    e.stopPropagation();
    onclick();
  }
</script>

<!-- svelte-ignore slot_element_deprecated -->
{#if disabled}
  <span>
    <slot />
  </span>
{:else}
  <span
    role="button"
    tabindex="0"
    onkeypress={(e) => e.key === ' ' && handler(e)}
    onclick={(e) => handler(e)}
    {...rest}
  >
    <slot />
  </span>
{/if}
