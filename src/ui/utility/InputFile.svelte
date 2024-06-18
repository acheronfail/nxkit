<script lang="ts" context="module">
  export interface Props {
    id?: string;
    label: string;
    files: FileList;

    // TODO: restrict this to HTMLInput props
    [key: string]: any;
  }
</script>

<script lang="ts">
  import { QuestionMarkCircleIcon } from 'heroicons-svelte/24/outline';
  import Tooltip from './Tooltip.svelte';

  let { label, id = label.toLowerCase(), files = $bindable(), ...rest }: Props = $props();
</script>

<!-- svelte-ignore slot_element_deprecated -->
<div class="flex justify-between items-center">
  <label class="min-w-32" for={id}>{label}:</label>
  <input
    type="file"
    class="grow rounded border border-slate-700 p-2 bg-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500 caret-blue-500"
    bind:files
    {...rest}
  />
  {#if $$slots.infoTooltip}
    <span class="has-tooltip p-2 cursor-default hover:text-slate-400">
      <Tooltip placement="left">
        <span slot="tooltip" class="text-white">
          <slot name="infoTooltip" />
        </span>
        <QuestionMarkCircleIcon slot="content" class="h-6" />
      </Tooltip>
    </span>
  {/if}
</div>
