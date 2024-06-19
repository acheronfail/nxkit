<script lang="ts" context="module">
  export interface Props {
    output?: string;
    title?: string;
  }
</script>

<script lang="ts">
  import Button from './Button.svelte';

  let { title, output = $bindable('') }: Props = $props();
</script>

{#if title}
  <h3 class="font-bold text-center">{title}</h3>
{/if}
<div class="overflow-auto rounded box-border border h-96 p-4 dark:border-slate-900 dark:bg-slate-700">
  <pre>{output}</pre>
</div>
<div class="text-center">
  <Button size="inline" onclick={() => (output = '')}>clear output</Button>
</div>
