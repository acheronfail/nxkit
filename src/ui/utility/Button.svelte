<script context="module" lang="ts">
  import type { Snippet } from 'svelte';
  import type { EventHandler, HTMLAttributes } from 'svelte/elements';

  export type Appearance = 'primary' | 'default' | 'warning' | 'danger';
  export type Size = 'large' | 'default' | 'inline' | 'small';

  export interface Props extends HTMLAttributes<HTMLElement> {
    appearance?: Appearance;
    size?: Size;
    children: Snippet;
    loading?: boolean;
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
    loading = false,
    children,
    class: propClass,
    ...rest
  }: Props = $props();

  const innerClass = 'w-full h-full';
  const buttonClass = `select-none text-center focus:outline-none focus:ring-2 focus:ring-blue-500 rounded`;
  const disabledClass = `bg-slate-600 text-slate-400 hover:text-slate-400`;
  const loadingClass = 'cursor-default';

  const appearanceClass: Record<Appearance, string> = {
    primary: `text-white bg-blue-500`,
    default: `text-white bg-slate-700`,
    warning: `text-white bg-orange-600`,
    danger: `text-white bg-red-700`,
  };
  const appearanceClassInteraction: Record<Appearance, string> = {
    primary: `hover:bg-blue-700 active:bg-blue-500`,
    default: `hover:bg-slate-500 active:bg-slate-700`,
    warning: `hover:bg-orange-700 active:bg-orange-800`,
    danger: `hover:bg-red-900 active:bg-red-500`,
  };
  const sizeClass: Record<Size, string> = {
    large: 'py-2 px-4 font-bold',
    default: 'py-1 px-2 font-bold',
    inline: 'px-2 text-sm',
    small: 'px-1 text-xs',
  };

  type EventParam = Parameters<EventHandler<Event, HTMLElement>>[0];
  const onPress = (event: EventParam, isKeyEvent = false) => {
    if (disabled || loading) {
      event.preventDefault();
    } else if (rest.onclick) {
      rest.onclick?.(event);
    } else if (isKeyEvent) {
      event.currentTarget.click();
    }
  };
</script>

{#snippet spinner()}
  <div class="flex justify-center items-center">
    <svg class="animate-spin h-6 w-6" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
      <path
        class="opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
      ></path>
    </svg>
  </div>
{/snippet}

{#if typeof rest.for === 'string'}
  <!-- svelte-ignore a11y_label_has_associated_control -->
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <!-- svelte-ignore a11y_no_noninteractive_element_to_interactive_role -->
  <label
    onclick={onPress}
    onkeydown={(e) => e.code === 'Space' && onPress(e, true)}
    tabindex="0"
    role="button"
    class="{disabled ? disabledClass : appearanceClass[appearance]} {loading
      ? loadingClass
      : disabled
        ? ''
        : appearanceClassInteraction[appearance]} {sizeClass[size]} {buttonClass} {propClass}"
    {...rest}
  >
    <div class={innerClass}>
      {#if loading}
        {@render spinner()}
      {:else}
        {@render children()}
      {/if}
    </div>
  </label>
{:else}
  <button
    class="{disabled ? disabledClass : appearanceClass[appearance]} {loading
      ? loadingClass
      : disabled
        ? ''
        : appearanceClassInteraction[appearance]} {sizeClass[size]} {buttonClass} {propClass}"
    {disabled}
    {...rest}
  >
    <div class={innerClass}>
      {#if loading}
        {@render spinner()}
      {:else}
        {@render children()}
      {/if}
    </div>
  </button>
{/if}
