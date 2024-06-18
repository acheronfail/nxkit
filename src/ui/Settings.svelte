<script lang="ts">
  import { keys } from './stores/keys.svelte';
  import { readFile } from '../browser/file';
  import Button from './utility/Button.svelte';
  import Container from './utility/Container.svelte';
  import Code from './utility/Code.svelte';
  import { onMount } from 'svelte';

  let input = $state<HTMLInputElement | null>(null);
  let files = $state<FileList | null>(null);
  let keyFile = $derived(files && files[0]);

  let searchPaths = $state<string[]>([]);

  $effect(() => {
    if (keyFile) {
      readFile(keyFile, 'string').then((data) => keys.setUserKeys({ location: keyFile.path, data }));
    }
  });

  onMount(() => {
    window.nxkit.keysSearchPaths().then((paths) => (searchPaths = paths));
  });

  function resetKeys() {
    keys.setUserKeys(null);
    files = null;
    input.value = '';
  }
</script>

<Container>
  <h4 class="font-bold">Select Prod Keys</h4>

  <div class="flex gap-2">
    {#if keys.value}
      ✅️ Configured keys: <Code class="grow">{keys.value.location}</Code>
    {:else}
      ❌️ No keys found
    {/if}
  </div>

  <p>
    <span class="flex justify-between items-center">
      {#if keys.userKeysSelected}
        <Button appearance="warning" onclick={resetKeys}>Clear selected keys</Button>
      {:else}
        <Button appearance="primary" for="prod-keys">Manually select keys</Button>
      {/if}

      <input hidden type="file" id="prod-keys" name="prod-keys" bind:this={input} bind:files />
    </span>
  </p>

  <p>
    Prod keys are required for creating NSPs with the NRO Forwarder, and also for reading the Switch's NAND partition in
    the explorer.
  </p>
  <p>
    By default, NXKit searches the following places for <Code>prod.keys</Code> files:
  </p>

  <ul class="list-disc ml-4">
    {#each searchPaths as path}
      <li class="m-1"><Code>{path}</Code></li>
    {/each}
  </ul>

  <h4 class="font-bold">Where do I get <Code>prod.keys</Code>?</h4>
  <p>
    You must extract the keys from your Switch, if you have an unpatched Switch you can use
    <Code>Lockpick_RCM</Code> to get them.
  </p>
</Container>
