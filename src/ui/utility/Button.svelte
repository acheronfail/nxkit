<script context="module" lang="ts">
  import type { Snippet } from "svelte";

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
  import { Tooltip } from "@svelte-plugins/tooltips";

  let {
    appearance = 'default',
    size = 'default',
    children,
    tooltip,
    ...rest
  }: Props = $props();

  const innerClass = 'w-full h-full';
  const buttonClass = 'inline-block text-center disabled:bg-slate-600 disabled:text-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 text-white rounded';
  const appearanceClass: Record<Appearance, string> = {
    'primary': 'bg-blue-500 hover:bg-blue-700 active:bg-blue-500',
    'default': 'bg-slate-700 hover:bg-slate-500 active:bg-slate-700',
    'warning': 'bg-orange-600 hover:bg-orange-700 active:bg-orange-600',
    'danger': 'bg-red-500 hover:bg-red-700 active:bg-red-500',
  };
  const sizeClass: Record<Size, string> = {
    'large': 'py-2 px-4 font-bold',
    'default': 'py-1 px-2 font-bold',
    'inline': 'px-1',
  };

</script>

{#if typeof rest.for === 'string'}
  <label class="{appearanceClass[appearance]} {sizeClass[size]} {buttonClass}" {...rest}>
    <Tooltip content={tooltip}>
      <div class={innerClass}>{@render children()}</div>
    </Tooltip>
  </label>
{:else}
  <button class="{appearanceClass[appearance]} {sizeClass[size]} {buttonClass}" {...rest}>
    <Tooltip content={tooltip}>
      <div class={innerClass}>{@render children()}</div>
    </Tooltip>
  </button>
{/if}
