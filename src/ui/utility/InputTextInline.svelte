<script lang="ts" context="module">
  export interface Props {
    class?: string;
    onEscape?: () => void;
    onEnter?: (value: string) => void;
    initialValue?: string;
    autoFocus?: boolean;
  }
</script>

<script lang="ts">
  import { onMount } from 'svelte';

  import type { KeyboardEventHandler, MouseEventHandler } from 'svelte/elements';

  let { autoFocus, onEnter, onEscape, initialValue, class: propClass = '' }: Props = $props();
  let node = $state<HTMLInputElement | null>(null);

  const onclick: MouseEventHandler<HTMLInputElement> = (e) => e.stopPropagation();
  const onkeydown: KeyboardEventHandler<HTMLInputElement> = (e) => {
    if (e.code === 'Enter') {
      onEnter?.(e.currentTarget.value);
    }
    if (e.code === 'Escape') {
      onEscape?.();
    }
  };

  onMount(() => {
    if (node) {
      if (autoFocus) {
        node.focus();
      }

      if (initialValue) {
        node.value = initialValue;
      }

      node.select();
    }
  });

  const shapeClass = 'rounded px-1 shadow-inset-border';
  const focusClass = 'focus:outline-none focus:ring-blue-500 focus:ring-1';
  const styleClass = 'bg-slate-900 border-slate-600 shadow-slate-600';
</script>

<input class="{shapeClass} {styleClass} {focusClass} {propClass}" type="text" bind:this={node} {onkeydown} {onclick} />
