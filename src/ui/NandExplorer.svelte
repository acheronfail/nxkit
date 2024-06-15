<script lang="ts">
  import ProdKeysNeeded from './ProdKeysNeeded.svelte';
  import { keys } from './stores/keys.svelte';

  let files = $state<FileList | null>(null);
  let nandFile = $derived(files?.[0]);

  async function loadNand(nandPath: string) {
    const partitions = await window.nxkit.nandOpen(nandPath);
    console.log(partitions);

    // TODO: list partitions to choose
    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
    const userPartition = partitions.find((p) => p.name === 'USER')!.name;
    await window.nxkit.nandMount(userPartition, $state.snapshot(keys.value));

    // TODO: file explorer
    const entries = await window.nxkit.nandReaddir('/');
    console.log(entries);

    await window.nxkit.nandClose();
  }

  $effect(() => {
    if (nandFile) {
      loadNand(nandFile.path);
    }
  });
</script>

<ProdKeysNeeded />
<div class="picker">
  <button disabled={!keys.value}><label for="rawnand-file">Choose your rawnand.bin</label></button>
  <input hidden type="file" name="rawnand-file" id="rawnand-file" bind:files />
</div>

<style>
  .picker {
    display: flex;
    flex-direction: column;
  }
</style>
