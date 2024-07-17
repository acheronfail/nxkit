<script lang="ts" context="module">
  import type { Snippet } from 'svelte';
  import type { HTMLAttributes } from 'svelte/elements';

  export interface Props extends HTMLAttributes<HTMLDivElement> {
    class?: string;
    fillParent?: boolean;
    children: Snippet;
  }
</script>

<script lang="ts">
  let { children, class: cls, fillParent, ...rest }: Props = $props();

  const fillClass = fillParent ? 'h-full w-full' : '';
</script>

<div class="grow flex flex-col gap-3 py-2 {fillClass} {cls}" {...rest}>
  {@render children()}
</div>
