<script lang="ts">
  import { keysFromUser } from '../browser/keys';

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  async function openNand() {
    const { files } = document.querySelector<HTMLInputElement>('#nand-file');
    const [rawNand] = files;
    if (!rawNand) return;

    const userKeys = await keysFromUser().then((k) => k && new TextDecoder().decode(k));

    const partitions = await window.nxkit.nandOpen(rawNand.path);
    console.log(partitions);

    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
    const userPartition = partitions.find((p) => p.name === 'USER')!.name;
    await window.nxkit.nandMount(userPartition, userKeys);

    const entries = await window.nxkit.nandReaddir('/');
    console.log(entries);

    await window.nxkit.nandClose();
  }
</script>

<span>Choose your rawnand.bin</span>
<input type="file" name="rawnand.bin" id="nand-file" onchange={openNand} />
