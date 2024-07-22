<script lang="ts">
  const { class: className, ...props } = $$props;
</script>

<span
  class={`font-mono break-words align-text-bottom text-xs border rounded p-1 border-slate-400 dark:border-slate-900 bg-slate-200 dark:bg-slate-700 ${className}`}
  {...props}
>
  <slot />
</span>
