<script lang="ts">
  import { Tooltip } from '@svelte-plugins/tooltips';
  import { QuestionMarkCircleIcon } from 'heroicons-svelte/24/outline';

  /** Label for the input */
  export let label: string;
  /** The input's value */
  export let value: string;
  /** Optionally explicitly set the id of the input */
  export let id = label.toLowerCase();
  /** Optional information, shown as a (?) next to the input */
  export let infoTooltip = '';
</script>

<div class="flex justify-between items-center">
  <label class="min-w-32" for={id}>{label}:</label>
  <input
    type="text"
    class="grow rounded border border-slate-700 p-2 bg-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500 caret-blue-500"
    bind:value
    {...$$props}
  />
  {#if infoTooltip}
    <span class="has-tooltip p-2 cursor-default hover:text-slate-400">
      <Tooltip position='left' content={infoTooltip}>
        <QuestionMarkCircleIcon class="h-6" />
      </Tooltip>
    </span>
  {/if}
</div>
