<script lang="ts" context="module">
  import type { HTMLAttributes } from 'svelte/elements';

  export interface Props extends HTMLAttributes<HTMLDivElement> {
    class?: string;
  }
</script>

<script lang="ts">
  let { class: cls, ...rest }: Props = $props();
</script>

<!-- svelte-ignore slot_element_deprecated -->
<div class="flex flex-col gap-px gap-3 my-2 h-full w-full {cls}" {...rest}>
  <slot />
</div>
