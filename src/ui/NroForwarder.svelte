<script lang="ts">
  import { downloadFile } from '../browser/file';
  import { getKeys } from '../browser/keys';
  import { buildNsp, generateRandomId } from '../hacbrewpack/nsp';

  // TODO: dynamic ui for retroarch forwarding, including verification and/or auto-complete of fields, etc
  // TODO: choose mounted Switch SD card for path autocomplete and validation

  let id = generateRandomId();

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  async function generate() {
    const { value: id } = document.querySelector<HTMLInputElement>('#nsp-id');
    const { value: title } = document.querySelector<HTMLInputElement>('#nsp-title');
    const { value: author } = document.querySelector<HTMLInputElement>('#nsp-author');
    const { value: nroPath } = document.querySelector<HTMLInputElement>('#nsp-nroPath');
    const keys = await getKeys();

    try {
      const result = await buildNsp({
        id,
        title,
        author,
        keys,
        nroPath,
        nroArgv: [],
      });

      document.querySelector('#nsp-stdout').textContent = result.stdout;
      document.querySelector('#nsp-stderr').textContent = result.stderr;

      if (result.exitCode !== 0) {
        alert(`Error generating NSP, please check the logs`);
      }

      if (result.nsp) {
        downloadFile(result.nsp);
      }
    } catch (err) {
      alert(String(err));
    }
  }
</script>

<div style="display: flex; flex-direction: column">
  <input type="text" name="id" id="nsp-id" placeholder="id" value={id} />
  <input type="text" name="title" id="nsp-title" placeholder="title" />
  <input type="text" name="author" id="nsp-author" placeholder="author" />
  <input type="text" name="nroPath" id="nsp-nroPath" placeholder="sdmc:/switch/your_app.nro" />
  <input type="submit" value="Generate NSP" onclick={generate} />
</div>

<pre id="nsp-stdout"></pre>
<pre id="nsp-stderr"></pre>
