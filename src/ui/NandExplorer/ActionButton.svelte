<script lang="ts" context="module">
  export interface Props {
    disabled?: boolean;
    onclick: () => void;
  }
</script>

<script lang="ts">
  let { onclick, disabled = false }: Props = $props();

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
  <span role="button" tabindex="0" onkeypress={(e) => e.key === ' ' && handler(e)} onclick={(e) => handler(e)}>
    <slot />
  </span>
{/if}
