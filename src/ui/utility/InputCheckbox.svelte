<script lang="ts" context="module">
  import type { HTMLInputAttributes } from 'svelte/elements';
  import type { Snippet } from 'svelte';
  import type { Placement } from '@floating-ui/dom';

  export type Appearance = 'default' | 'pronounced';
  export interface Props extends HTMLInputAttributes {
    id: string;
    label: string | Snippet;
    appearance?: Appearance;
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
    appearance = 'default',
    disabled,
    checked = $bindable(false),
    tooltip: tooltipProp,
    tooltipPlacement = 'left',
    ...rest
  }: Props = $props();

  const iconClass = 'absolute inline-block align-baseline h-[1.5rem]';
  const toggleClass = 'h-[1.5rem] w-[3rem]';

  const checkedIconClass = appearance == 'pronounced' ? 'fill-green-500' : 'fill-green-700 dark:fill-green-300';
  const checkedBorderClass = appearance == 'pronounced' ? 'shadow-green-500' : 'shadow-slate-400';
  const uncheckedIconClass = appearance == 'pronounced' ? 'fill-red-500' : 'fill-red-700 dark:fill-red-300';
  const uncheckedBorderClass = appearance == 'pronounced' ? 'shadow-red-500' : 'shadow-slate-400';

  const enabledClass = 'cursor-pointer';
  const disabledClass = 'text-slate-500';
</script>

{#snippet renderCheckbox()}
  <div class="select-none inline-flex justify-between items-center">
    <label
      class="min-w-32 inline-flex justify-between items-center gap-2 {disabled ? disabledClass : enabledClass}"
      for={id}
    >
      {#if typeof label === 'string'}
        {label}:
      {:else}
        {@render label()}
      {/if}
      <div
        class="relative rounded-full bg-slate-300 dark:bg-slate-900 shadow-inset-border {toggleClass} {disabled
          ? 'shadow-slate-500'
          : checked
            ? checkedBorderClass
            : uncheckedBorderClass}"
      >
        {#if checked}
          <CheckCircleIcon class="{checkedIconClass} right-0 {iconClass}" />
        {:else}
          <XCircleIcon class="{uncheckedIconClass} left-0 {iconClass}" />
        {/if}
      </div>
    </label>

    <input hidden type="checkbox" bind:checked {id} {disabled} {...rest} />
  </div>
{/snippet}

{#if tooltipProp}
  <span class="p-2">
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
