<script lang="ts">
  import { downloadFile } from '../browser/file';
  import { buildNsp, generateRandomId } from '../hacbrewpack/nsp';
  import ProdKeysNeeded from './ProdKeysNeeded.svelte';
  import { keys } from './stores/keys.svelte';

  // TODO: image selection
  // TODO: dynamic ui for retroarch forwarding, including verification and/or auto-complete of fields, etc
  // TODO: choose mounted Switch SD card for path autocomplete and validation

  let id = $state(generateRandomId());
  let title = $state('');
  let author = $state('');
  let nroPath = $state('');

  let stdout = $state('');
  let stderr = $state('');

  async function generate() {
    try {
      const result = await buildNsp({
        id,
        title,
        author,
        keys: keys.value.data,
        nroPath,
        nroArgv: [],
      });

      stdout = result.stdout;
      stderr = result.stderr;

      if (result.exitCode !== 0) {
        alert(`Error generating NSP, please check the logs`);
      }

      if (result.nsp) {
        downloadFile(result.nsp);
      }
    } catch (err) {
      console.error(err);
      alert(String(err));
    }
  }
</script>

<div style="display: flex; flex-direction: column">
  <input type="text" name="id" placeholder="id" bind:value={id} />
  <input type="text" name="title" placeholder="title" bind:value={title} />
  <input type="text" name="author" placeholder="author" bind:value={author} />
  <input type="text" name="nroPath" placeholder="sdmc:/switch/your_app.nro" bind:value={nroPath} />
  <ProdKeysNeeded />
  <input type="submit" value="Generate NSP" disabled={!keys.value} onclick={generate} />
</div>

<pre>{stdout}</pre>
<pre>{stderr}</pre>
