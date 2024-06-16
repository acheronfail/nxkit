<script lang="ts">
  import { Tooltip } from "@svelte-plugins/tooltips";
  import { QuestionMarkCircleIcon } from "heroicons-svelte/24/outline";

  export let label: string;
  export let id = label.toLowerCase();
  export let files: FileList;
  export let infoTooltip = '';
</script>

<div class="flex justify-between items-center">
  <label class="min-w-32" for={id}>{label}:</label>
  <input
    type="file"
    class="grow rounded border border-slate-700 p-2 bg-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500 caret-blue-500"
    bind:files
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
