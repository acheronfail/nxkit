<script lang="ts">
  import { downloadFile } from '../browser/file';
  import { buildNsp } from '../hacbrewpack/nsp';
  import { generateTitleId } from '../hacbrewpack/id';
  import { keys } from './stores/keys.svelte';
  import Button from './utility/Button.svelte';
  import Container from './utility/Container.svelte';
  import TabItem from './utility/TabItem.svelte';
  import Tabs from './utility/Tabs.svelte';
  import InputFile from './utility/InputText.svelte';
  import InputImage, { type Image } from './utility/InputImage.svelte';
  import LogOutput from './utility/LogOutput.svelte';

  // TODO: choose mounted Switch SD card for path autocomplete and validation?

  const nroPathDesc = (name: string) => `File path to the ${name} NRO file on the Nintendo Switch SD card.`;
  const descriptions = {
    id: `A hexadecimal id for the title. This value isn't shown in the UI, but it makes sure titles don't conflict with each other.`,
    title: 'The name which is displayed on the Nintendo Switch home screen.',
    author: `Name of the publisher displayed in the title's details screen.`,
    nroPath: {
      app: nroPathDesc("homebrew application's"),
      rom: nroPathDesc("RetroArch core's"),
    },
    romPath: "File path to the game's ROM file on the Nintendo Switch SD card",
  };

  let id = $state(generateTitleId());
  let title = $state('');
  let author = $state('');
  let nroPath = $state('');
  let romPath = $state('');
  let image = $state<Image | null>(null);

  let disabled = $derived(!keys.value || !image);
  let tooltip = $derived(
    !keys.value ? 'Please select your prod.keys in Settings!' : !image ? 'Please select an image!' : undefined,
  );

  let stdout = $state('');
  let stderr = $state('');

  async function generate() {
    try {
      if (!image) {
        throw new Error('Cannot generate an NSP without an image!');
      }

      if (!keys.value) {
        throw new Error('Cannot generate an NSP without prod.keys!');
      }

      const imageBlob = await image.toBlob();
      const imageData = new Uint8Array(await imageBlob.arrayBuffer());

      const result = await buildNsp({
        id,
        title,
        author,
        image: imageData,
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

  const tooltipClass = 'w-60 text-center';
</script>

<Container class="gap-4" data-testid="nro-forwarder">
  <Tabs>
    <TabItem defaultOpen>
      <span slot="label">Application</span>
      <Container slot="content">
        <InputImage onCropComplete={(img) => (image = img)} />
        <InputFile label="App ID" placeholder="01..........0000" bind:value={id}>
          <div slot="infoTooltip" class={tooltipClass}>{descriptions.id}</div>
        </InputFile>
        <InputFile label="App Title" placeholder="NX Shell" bind:value={title}>
          <div slot="infoTooltip" class={tooltipClass}>{descriptions.title}</div>
        </InputFile>
        <InputFile label="App Publisher" placeholder="joel16" bind:value={author}>
          <div slot="infoTooltip" class={tooltipClass}>{descriptions.author}</div>
        </InputFile>
        <InputFile label="NRO Path" placeholder="/switch/NX-Shell.nro" bind:value={nroPath}>
          <div slot="infoTooltip" class={tooltipClass}>{descriptions.nroPath.app}</div>
        </InputFile>
      </Container>
    </TabItem>
    <TabItem>
      <span slot="label">RetroArch ROM</span>
      <Container slot="content">
        <InputImage onCropComplete={(img) => (image = img)} />
        <InputFile label="App ID" placeholder="01..........0000" bind:value={id}>
          <div slot="infoTooltip" class={tooltipClass}>{descriptions.id}</div>
        </InputFile>
        <InputFile label="Game Title" placeholder="Kirby's Adventure" bind:value={title}>
          <div slot="infoTooltip" class={tooltipClass}>{descriptions.title}</div>
        </InputFile>
        <InputFile label="Game Publisher" placeholder="Nintendo" bind:value={author}>
          <div slot="infoTooltip" class={tooltipClass}>{descriptions.author}</div>
        </InputFile>
        <InputFile label="Core Path" placeholder="/retroarch/cores/nestopia_libretro_libnx.nro" bind:value={nroPath}>
          <div slot="infoTooltip" class={tooltipClass}>{descriptions.nroPath.rom}</div>
        </InputFile>
        <InputFile label="ROM Path" placeholder="/roms/nes/Kirby's Adventure.zip" bind:value={romPath}>
          <div slot="infoTooltip" class={tooltipClass}>{descriptions.romPath}</div>
        </InputFile>
      </Container>
    </TabItem>
  </Tabs>

  <Button class="mt-4" appearance="primary" size="large" onclick={generate} {disabled} {tooltip}>Generate NSP</Button>
</Container>

{#if stdout}<LogOutput title="Standard Out" bind:output={stdout} />{/if}
{#if stderr}<LogOutput title="Standard Err" bind:output={stderr} />{/if}
