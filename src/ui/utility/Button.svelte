<script context="module" lang="ts">
  import type { Snippet } from 'svelte';
  import Tooltip from './Tooltip.svelte';

  export type Appearance = 'primary' | 'default' | 'warning' | 'danger';
  export type Size = 'large' | 'default' | 'inline';
  export type Props = {
    appearance?: Appearance;
    size?: Size;
    tooltip?: string;
    children: Snippet;
    [other: string]: any;
  };
</script>

<script lang="ts">
  let { appearance = 'default', size = 'default', disabled = false, children, tooltip, ...rest }: Props = $props();

  const innerClass = 'w-full h-full';
  const buttonClass = `select-none inline-block text-center focus:outline-none focus:ring-2 focus:ring-blue-500 rounded`;
  const disabledClass = `bg-slate-600 text-slate-400 hover:text-slate-400`;
  const appearanceClass: Record<Appearance, string> = {
    primary: `text-white bg-blue-500 hover:bg-blue-700 active:bg-blue-500`,
    default: `text-white bg-slate-700 hover:bg-slate-500 active:bg-slate-700`,
    warning: `text-white bg-orange-600 hover:bg-orange-700 active:bg-orange-600`,
    danger: `text-white bg-red-500 hover:bg-red-700 active:bg-red-500`,
  };
  const sizeClass: Record<Size, string> = {
    large: 'py-2 px-4 font-bold',
    default: 'py-1 px-2 font-bold',
    inline: 'px-1',
  };

  const preventDefault = (e: Event) => e.preventDefault();
</script>

<!-- svelte-ignore slot_element_deprecated -->
{#if typeof rest.for === 'string'}
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <label
    onclick={disabled ? preventDefault : rest.onclick}
    class="{disabled ? disabledClass : appearanceClass[appearance]} {sizeClass[size]} {buttonClass}"
    {...rest}
  >
    {#if tooltip}
      <Tooltip>
        <span slot="tooltip" class="text-white">{tooltip}</span>
        <div slot="content" class={innerClass}><slot /></div>
      </Tooltip>
    {:else}
      <Tooltip>
        <div slot="content" class={innerClass}><slot /></div>
      </Tooltip>
    {/if}
  </label>
{:else}
  <button
    class="{disabled ? disabledClass : appearanceClass[appearance]} {sizeClass[size]} {buttonClass}"
    {disabled}
    {...rest}
  >
    {#if tooltip}
      <Tooltip>
        <span slot="tooltip" class="text-white">{tooltip}</span>
        <div slot="content" class={innerClass}><slot /></div>
      </Tooltip>
    {:else}
      <Tooltip>
        <div slot="content" class={innerClass}><slot /></div>
      </Tooltip>
    {/if}
  </button>
{/if}
