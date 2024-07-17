<script lang="ts" context="module">
  import type { HTMLInputAttributes } from 'svelte/elements';
  import type { Snippet } from 'svelte';

  export interface Props extends HTMLInputAttributes {
    /** id of the input */
    id?: string;
    /** label of the input */
    label: string;
    /** you can bind to this to check if the input has selected anything, setting it does nothing */
    selected?: boolean;
    /** optionally provide a (?) icon with a tooltip about this input */
    infoTooltip?: Snippet;

    files: FileList | null;
  }
</script>

<script lang="ts">
  import { QuestionMarkCircleIcon, XCircleIcon } from 'heroicons-svelte/24/outline';
  import Tooltip from './Tooltip.svelte';

  let {
    label,
    id = label.toLowerCase(),
    infoTooltip,
    selected = $bindable(false),
    files = $bindable(null),
    ...rest
  }: Props = $props();

  let inputNode = $state<HTMLInputElement | undefined>();

  function init(node: HTMLInputElement) {
    node.addEventListener('change', () => (selected = true));
    selected = false;
  }

  function clear() {
    if (!inputNode) return;

    inputNode.files = null;
    inputNode.value = '';
    selected = false;
  }
</script>

<div class="flex justify-between items-center">
  <label class="min-w-32" for={id}>{label}:</label>
  <input
    use:init
    bind:this={inputNode}
    bind:files
    type="file"
    class="grow rounded border border-slate-700 p-2 bg-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500 caret-blue-500"
    {...rest}
  />
  {#if selected}
    <span class="p-2 cursor-pointer hover:text-slate-400">
      <XCircleIcon onclick={clear} class="h-6" />
    </span>
  {/if}
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
