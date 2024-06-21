<script context="module" lang="ts">
  import type { Snippet } from 'svelte';
  import type { EventHandler, HTMLAttributes } from 'svelte/elements';

  export type Appearance = 'primary' | 'default' | 'warning' | 'danger';
  export type Size = 'large' | 'default' | 'inline' | 'small';

  export interface Props extends HTMLAttributes<HTMLElement> {
    appearance?: Appearance;
    size?: Size;
    children: Snippet;
    disabled?: boolean;
    for?: string;
    onclick?: EventHandler<Event, HTMLElement>;
  }
</script>

<script lang="ts">
  let {
    appearance = 'default',
    size = 'default',
    disabled = false,
    children,
    class: propClass,
    ...rest
  }: Props = $props();

  const innerClass = 'w-full h-full';
  const buttonClass = `select-none text-center focus:outline-none focus:ring-2 focus:ring-blue-500 rounded`;
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
    inline: 'px-2 text-sm',
    small: 'px-1 text-xs',
  };

  type EventParam = Parameters<EventHandler<Event, HTMLElement>>[0];
  const onPress = (event: EventParam, isKeyEvent = false) => {
    if (disabled) {
      event.preventDefault();
    } else if (rest.onclick) {
      rest.onclick?.(event);
    } else if (isKeyEvent) {
      event.currentTarget.click();
    }
  };
</script>

{#if typeof rest.for === 'string'}
  <!-- svelte-ignore a11y_label_has_associated_control -->
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <!-- svelte-ignore a11y_no_noninteractive_element_to_interactive_role -->
  <label
    onclick={onPress}
    onkeydown={(e) => e.code === 'Space' && onPress(e, true)}
    tabindex="0"
    role="button"
    class="{disabled ? disabledClass : appearanceClass[appearance]} {sizeClass[size]} {buttonClass} {propClass}"
    {...rest}
  >
    <div class={innerClass}>
      {@render children()}
    </div>
  </label>
{:else}
  <button
    class="{disabled ? disabledClass : appearanceClass[appearance]} {sizeClass[size]} {buttonClass} {propClass}"
    {disabled}
    {...rest}
  >
    <div class={innerClass}>
      {@render children()}
    </div>
  </button>
{/if}
