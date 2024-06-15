<script lang="ts">
  import { downloadFile } from '../browser/file';
  import { buildNsp, generateRandomId } from '../hacbrewpack/nsp';
  import ProdKeysNeeded from './ProdKeysNeeded.svelte';
  import { keys } from './stores/keys.svelte';
  import Button from './utility/Button.svelte';
  import TabContent from './utility/TabContent.svelte';
  import TabItem from './utility/TabItem.svelte';
  import Tabs from './utility/Tabs.svelte';
  import TextInput from './utility/TextInput.svelte';

  // TODO: image selection
  // TODO: choose mounted Switch SD card for path autocomplete and validation?

  let id = $state(generateRandomId());
  let title = $state('');
  let author = $state('');
  let nroPath = $state('');
  let romPath = $state('');

  let stdout = $state('');
  let stderr = $state('');

  async function generate() {
    try {
      const result = await buildNsp({
        id,
        title,
        author,
        keys: keys.value.data,
        nroPath: `sdmc:${nroPath}`,
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

<TabContent>
  <Tabs>
    <TabItem title="Application" defaultOpen>
      <TabContent>
        <TextInput label="App ID" placeholder="01..........0000" bind:value={id} />
        <TextInput label="App Title" placeholder="NX Shell" bind:value={title} />
        <TextInput label="App Publisher" placeholder="joel16" bind:value={author} />
        <TextInput label="NRO Path" placeholder="/switch/NX-Shell.nro" bind:value={nroPath} />
      </TabContent>
    </TabItem>
    <TabItem title="RetroArch ROM">
      <TabContent>
        <TextInput label="App ID" placeholder="01..........0000" bind:value={id} />
        <TextInput label="Game Title" placeholder="Kirby's Adventure" bind:value={title} />
        <TextInput label="Game Publisher" placeholder="Nintendo" bind:value={author} />
        <TextInput label="Core Path" placeholder="/retroarch/cores/nestopia_libretro_libnx.nro" bind:value={nroPath} />
        <TextInput label="ROM Path" placeholder="/roms/nes/Kirby's Adventure.zip" bind:value={romPath} />
      </TabContent>
    </TabItem>
  </Tabs>

  <ProdKeysNeeded />
  <Button onclick={generate} disabled={!keys.value}>Generate NSP</Button>
</TabContent>

<pre>{stdout}</pre>
<pre>{stderr}</pre>
