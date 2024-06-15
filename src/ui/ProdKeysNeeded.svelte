<script lang="ts">
  import { keys } from './stores/keys.svelte';
</script>

{#if !keys.value}
  <p>
    Configuring <span class="code">prod.keys</span> is required. Please select some in Settings!
  </p>
{/if}
