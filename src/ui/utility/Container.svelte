<script lang="ts" context="module">
  import type { HTMLAttributes } from 'svelte/elements';

  export interface Props extends HTMLAttributes<HTMLDivElement> {
    class?: string;
    fillContainer?: boolean;
  }
</script>

<script lang="ts">
  let { class: cls, fillContainer, ...rest }: Props = $props();

  const fillClass = fillContainer ? 'h-full w-full' : '';
</script>

<!-- svelte-ignore slot_element_deprecated -->
<div class="flex flex-col gap-px gap-3 my-2 {fillClass} {cls}" {...rest}>
  <slot />
</div>
