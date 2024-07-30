<script lang="ts" context="module">
  import type { HTMLInputAttributes } from 'svelte/elements';
  import type { Snippet } from 'svelte';

  export interface Props extends HTMLInputAttributes {
    id: string;
    label: string | Snippet;
    value?: string;
    infoTooltip?: Snippet;
  }
</script>

<script lang="ts">
  import { QuestionMarkCircleIcon } from 'heroicons-svelte/24/outline';
  import Tooltip from './Tooltip.svelte';

  let { label, id, value = $bindable(), infoTooltip, ...rest }: Props = $props();
</script>

<div class="flex justify-between items-center">
  <label class="min-w-32" for={id}>
    {#if typeof label === 'string'}
      {label}:
    {:else}
      {@render label()}
    {/if}
  </label>
  <input
    type="text"
    class="grow rounded p-2 border border-slate-400 dark:border-slate-700 bg-slate-200 dark:bg-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500 caret-blue-500"
    bind:value
    {...rest}
  />
  {#if infoTooltip}
    <span class="p-2 cursor-default hover:text-slate-400">
      <Tooltip placement="left">
        {#snippet tooltip()}
          <span class="text-white">
            {@render infoTooltip()}
          </span>
        {/snippet}
        <QuestionMarkCircleIcon class="h-6" />
      </Tooltip>
    </span>
  {/if}
</div>
