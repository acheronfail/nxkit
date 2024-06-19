<script lang="ts" context="module">
  import type { HTMLSelectAttributes } from 'svelte/elements';

  export interface Option {
    value?: string;
    selected?: boolean;
    disabled?: boolean;
    displayName: string;
  }

  export interface Props extends HTMLSelectAttributes {
    options: Option[];
    small?: boolean;
    value?: string;
  }
</script>

<script lang="ts">
  let { options = $bindable(), value = $bindable(), small = false, ...rest }: Props = $props();

  const dispClass = small ? 'inline-block' : 'w-full h-full';
  const sizeClass = small ? 'p-1 min-w-20 text-xs pr-6' : 'p-2';
</script>

<div class="relative {dispClass}">
  <svg class="absolute top-1/3 right-2 h-1/3 dark:fill-white" viewBox="0 0 16 16" xmlns="http://www.w3.org/2000/svg">
    <path
      d="M7.247 11.14 2.451 5.658C1.885 5.013 2.345 4 3.204 4h9.592a1 1 0 0 1 .753 1.659l-4.796 5.48a1 1 0 0 1-1.506 0z"
    />
  </svg>
  <select
    class="{dispClass} border rounded appearance-none {sizeClass} focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-slate-700 dark:border-slate-900"
    class:text-slate-400={value === ''}
    onchange={(e) => (value = e.currentTarget.value)}
    bind:value
    {...rest}
  >
    {#each options as { value, disabled, selected, displayName }}
      <option {value} {selected} {disabled}>
        {displayName}
      </option>
    {/each}
  </select>
</div>
