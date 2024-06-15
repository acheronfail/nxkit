<script lang="ts">
  import ProdKeysNeeded from './ProdKeysNeeded.svelte';
  import { keys } from './stores/keys.svelte';
  import Button from './utility/Button.svelte';
  import TabContent from './utility/TabContent.svelte';

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

<TabContent>
  <ProdKeysNeeded />
  <div class="picker">
    <Button disabled={!keys.value}>
      <label for="rawnand-file">Choose your rawnand.bin</label>
    </Button>
    <input hidden type="file" bind:files />
  </div>
</TabContent>

<style>
  .picker {
    display: flex;
    flex-direction: column;
  }
</style>
