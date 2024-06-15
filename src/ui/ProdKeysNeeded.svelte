<script lang="ts">
  import { keys } from './stores/keys.svelte';
  import Code from './utility/Code.svelte';
</script>

{#if !keys.value}
  <p>
    Configuring <Code>prod.keys</Code> is required. Please select some in Settings!
  </p>
{/if}
