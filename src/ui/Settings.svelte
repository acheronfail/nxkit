<script lang="ts">
  import { keys } from './stores/keys.svelte';
  import { readFile } from '../browser/file';

  let input = $state<HTMLInputElement | null>(null);
  let files = $state<FileList | null>(null);
  let keyFile = $derived(files && files[0]);

  $effect(() => {
    if (keyFile) {
      readFile(keyFile, 'string').then((data) => keys.setUserKeys({ location: keyFile.path, data }));
    } else {
      keys.setUserKeys(null);
    }
  });

  function resetKeys(event: Event) {
    event.preventDefault();
    files = null;
    input.value = '';
  }
</script>

<h4>Select Prod Keys</h4>

{#if keys.value}
  ✅️ Configured keys: <span class="code">{keys.value.location}</span>
{:else}
  ❌️ No keys found
{/if}

<p>
  <span class="manual">
    {#if keyFile}
      <button onclick={resetKeys}>Clear selected keys</button>
    {/if}
    <button hidden={!!keyFile}>
      <label for="prod-keys">Manually select keys</label>
    </button>
    <input type="file" id="prod-keys" name="prod-keys" bind:this={input} bind:files />
  </span>
</p>

<p>
  Prod keys are required for creating NSPs with the NRO Forwarder, and also for reading the Switch's NAND partition in
  the explorer.
</p>
<p>
  By default, NXKit searches the following places for <span class="code">prod.keys</span> files:
</p>
<!-- TODO: single source of truth for this -->
<!-- TODO: explain what `~` and `$CWD` mean (cross-platform, too) -->
<ul>
  <li><span class="code">~/.switch/prod.keys</span></li>
  <li><span class="code">$CWD/prod.keys</span></li>
</ul>

<h5>Where do I get <span class="code">prod.keys</span>?</h5>
<p>
  You must extract the keys from your Switch, if you have an unpatched Switch you can use
  <span class="code">Lockpick_RCM</span> to get them.
</p>

<style>
  li {
    margin-top: 1px;
  }

  input#prod-keys {
    display: none;
  }

  .manual {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
</style>
