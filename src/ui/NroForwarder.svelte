<script lang="ts">
  import { downloadFile } from '../browser/file';
  import { buildNsp, generateRandomId } from '../hacbrewpack/nsp';
  import { keys } from './stores/keys.svelte';
  import Button from './utility/Button.svelte';
  import TabContent from './utility/TabContent.svelte';
  import TabItem from './utility/TabItem.svelte';
  import Tabs from './utility/Tabs.svelte';
  import InputFile from './utility/InputText.svelte';

  // TODO: image selection
  // TODO: choose mounted Switch SD card for path autocomplete and validation?

  const descriptions = {
    id: 'TODO',
    title: 'TODO',
    author: 'TODO',
    nroPath: 'TODO',
    romPath: 'TODO',
  };

  let id = $state(generateRandomId());
  let title = $state('');
  let author = $state('');
  let nroPath = $state('');
  let romPath = $state('');

  let disabled = $derived(!keys.value);
  let tooltip = $derived(disabled && 'Please select your prod.keys in Settings!');

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
        <InputFile label="App ID" placeholder="01..........0000" bind:value={id} infoTooltip={descriptions.id} />
        <InputFile label="App Title" placeholder="NX Shell" bind:value={title} infoTooltip={descriptions.title} />
        <InputFile label="App Publisher" placeholder="joel16" bind:value={author} infoTooltip={descriptions.author} />
        <InputFile
          label="NRO Path"
          placeholder="/switch/NX-Shell.nro"
          bind:value={nroPath}
          infoTooltip={descriptions.nroPath}
        />
      </TabContent>
    </TabItem>
    <TabItem title="RetroArch ROM">
      <TabContent>
        <InputFile label="App ID" placeholder="01..........0000" bind:value={id} infoTooltip={descriptions.id} />
        <InputFile
          label="Game Title"
          placeholder="Kirby's Adventure"
          bind:value={title}
          infoTooltip={descriptions.title}
        />
        <InputFile
          label="Game Publisher"
          placeholder="Nintendo"
          bind:value={author}
          infoTooltip={descriptions.author}
        />
        <InputFile
          label="Core Path"
          placeholder="/retroarch/cores/nestopia_libretro_libnx.nro"
          bind:value={nroPath}
          infoTooltip={descriptions.nroPath}
        />
        <InputFile
          label="ROM Path"
          placeholder="/roms/nes/Kirby's Adventure.zip"
          bind:value={romPath}
          infoTooltip={descriptions.romPath}
        />
      </TabContent>
    </TabItem>
  </Tabs>

  <Button appearance="primary" size="large" onclick={generate} {disabled} {tooltip}>Generate NSP</Button>
</TabContent>

<pre>{stdout}</pre>
<pre>{stderr}</pre>
