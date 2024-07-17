<script lang="ts" context="module">
  import type { HTMLInputAttributes } from 'svelte/elements';
  import type { Snippet } from 'svelte';

  export interface Props extends HTMLInputAttributes {
    id?: string;
    label: string;
    value?: string;
    infoTooltip?: Snippet;
  }
</script>

<script lang="ts">
  import { QuestionMarkCircleIcon } from 'heroicons-svelte/24/outline';
  import Tooltip from './Tooltip.svelte';

  let { label, id = label.toLowerCase(), value = $bindable(), infoTooltip, ...rest }: Props = $props();
</script>

<div class="flex justify-between items-center">
  <label class="min-w-32" for={id}>{label}:</label>
  <input
    type="text"
    class="grow rounded border border-slate-700 p-2 bg-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500 caret-blue-500"
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
