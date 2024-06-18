<script lang="ts" context="module">
  export interface Props {
    onclick: () => void;
  }
</script>

<script lang="ts">
  let { onclick }: Props = $props();

  function handler(e: Event) {
    e.stopPropagation();
    onclick();
  }
</script>

<!-- svelte-ignore slot_element_deprecated -->
<span role="button" tabindex="0" onkeypress={(e) => e.key === ' ' && handler(e)} onclick={(e) => handler(e)}>
  <slot />
</span>
