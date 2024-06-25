<script lang="ts" context="module">
  import type { HTMLInputAttributes } from 'svelte/elements';
  import type { Snippet } from 'svelte';
  import type { Placement } from '@floating-ui/dom';

  export interface Props extends HTMLInputAttributes {
    id?: string;
    label: Snippet;
    checked?: boolean;
    tooltip?: Snippet;
    tooltipPlacement?: Placement;
  }
</script>

<script lang="ts">
  import { CheckCircleIcon, XCircleIcon } from 'heroicons-svelte/24/solid';
  import Tooltip from './Tooltip.svelte';

  let {
    label,
    id,
    disabled,
    checked = $bindable(false),
    tooltip: tooltipProp,
    tooltipPlacement = 'left',
    ...rest
  }: Props = $props();

  const iconClass = 'absolute inline-block align-baseline h-[1.5rem]';
  const toggleClass = 'h-[1.5rem] w-[3rem]';

  const enabledClass = 'cursor-pointer';
  const disabledClass = 'text-slate-500';
</script>

{#snippet renderCheckbox()}
  <div class="select-none flex justify-between items-center">
    <label class="min-w-32 flex justify-between items-center gap-2 {disabled ? disabledClass : enabledClass}" for={id}>
      {@render label()}
      <div
        class="relative rounded-full bg-slate-900 shadow-inset-border {toggleClass} {disabled
          ? 'shadow-slate-500'
          : checked
            ? 'shadow-green-500'
            : 'shadow-red-500'}"
      >
        {#if checked}
          <CheckCircleIcon class="fill-green-500 right-0 {iconClass}" />
        {:else}
          <XCircleIcon class="fill-red-500 left-0 {iconClass}" />
        {/if}
      </div>
    </label>

    <input hidden type="checkbox" bind:checked {id} {disabled} {...rest} />
  </div>
{/snippet}

{#if tooltipProp}
  <span class="has-tooltip p-2">
    <Tooltip placement={tooltipPlacement}>
      {#snippet tooltip()}
        {@render tooltipProp()}
      {/snippet}
      {@render renderCheckbox()}
    </Tooltip>
  </span>
{:else}
  {@render renderCheckbox()}
{/if}
