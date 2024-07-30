<script lang="ts">
  import { ArrowPathIcon } from 'heroicons-svelte/24/solid';
  import { downloadFile } from '../browser/file';
  import { buildNsp } from '../browser/hacbrewpack/nsp';
  import { generateTitleId, titleIdsPromise } from '../browser/hacbrewpack/id';
  import { keys } from './stores/keys.svelte';
  import Button from './utility/Button.svelte';
  import Container from './utility/Container.svelte';
  import InputText from './utility/InputText.svelte';
  import InputImage, { type Image } from './utility/InputImage.svelte';
  import LogOutput from './utility/LogOutput.svelte';
  import { Tabs, TabList, TabContent, Tab } from './utility/Tabs';
  import Tooltip from './utility/Tooltip.svelte';
  import { onMount } from 'svelte';

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

  let id = $state('');
  let title = $state('');
  let author = $state('');
  let nroPath = $state('');
  let romPath = $state('');
  let image = $state<Image | null>(null);

  const regenerateId = () => generateTitleId().then((randomId) => (id = randomId));

  onMount(() => {
    regenerateId();
  });

  let disabled = $derived(!keys.value || !image);
  let tooltipText = $derived(
    !keys.value ? 'Please select your prod.keys in Settings!' : !image ? 'Please select an image!' : undefined,
  );

  let stdout = $state('');
  let stderr = $state('');

  async function generate() {
    const titleIds = await titleIdsPromise;

    try {
      if (titleIds.has(id)) {
        throw new Error(
          `The provided id: '${id}' conflicts with a known system or game id! Please choose another one.`,
        );
      }

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

{#snippet renderId()}
  <InputText id="appId" placeholder="01..........0000" bind:value={id}>
    {#snippet label()}
      <div class="flex justify-between items-center pr-2">
        App ID:
        <Tooltip placement="top">
          <Button size="small" onclick={regenerateId}>
            <ArrowPathIcon class="w-6" />
          </Button>
          {#snippet tooltip()}
            <p class="w-96 text-center">
              Regenerate id. The IDs generated here should be safe: they're checked against a list of known system ids,
              as well as a list of known game ids.
            </p>
          {/snippet}
        </Tooltip>
      </div>
    {/snippet}
    {#snippet infoTooltip()}
      <div class={tooltipClass}>{descriptions.id}</div>
    {/snippet}
  </InputText>
{/snippet}

<Container data-testid="nro-forwarder">
  <Tabs>
    <TabList>
      <Tab>Application</Tab>
      <Tab>RetroArch ROM</Tab>
    </TabList>

    <TabContent class="justify-around gap-2">
      <InputImage onCropComplete={(img) => (image = img)} />
      {@render renderId()}
      <InputText id="appTitle" label="App Title" placeholder="NX Shell" bind:value={title}>
        {#snippet infoTooltip()}
          <div class={tooltipClass}>{descriptions.title}</div>
        {/snippet}
      </InputText>
      <InputText id="appPublisher" label="App Publisher" placeholder="joel16" bind:value={author}>
        {#snippet infoTooltip()}
          <div class={tooltipClass}>{descriptions.author}</div>
        {/snippet}
      </InputText>
      <InputText id="nroPath" label="NRO Path" placeholder="/switch/NX-Shell.nro" bind:value={nroPath}>
        {#snippet infoTooltip()}
          <div class={tooltipClass}>{descriptions.nroPath.app}</div>
        {/snippet}
      </InputText>
    </TabContent>
    <TabContent class="justify-around gap-2">
      <InputImage onCropComplete={(img) => (image = img)} />
      {@render renderId()}
      <InputText id="gameTitle" label="Game Title" placeholder="Kirby's Adventure" bind:value={title}>
        {#snippet infoTooltip()}
          <div class={tooltipClass}>{descriptions.title}</div>
        {/snippet}
      </InputText>
      <InputText id="gamePublisher" label="Game Publisher" placeholder="Nintendo" bind:value={author}>
        {#snippet infoTooltip()}
          <div class={tooltipClass}>{descriptions.author}</div>
        {/snippet}
      </InputText>
      <InputText
        id="corePath"
        label="Core Path"
        placeholder="/retroarch/cores/nestopia_libretro_libnx.nro"
        bind:value={nroPath}
      >
        {#snippet infoTooltip()}
          <div class={tooltipClass}>{descriptions.nroPath.rom}</div>
        {/snippet}
      </InputText>
      <InputText id="romPath" label="ROM Path" placeholder="/roms/nes/Kirby's Adventure.zip" bind:value={romPath}>
        {#snippet infoTooltip()}
          <div class={tooltipClass}>{descriptions.romPath}</div>
        {/snippet}
      </InputText>
    </TabContent>
  </Tabs>

  <div>
    <Tooltip disabled={!tooltipText}>
      {#snippet tooltip()}
        <p>
          {tooltipText}
        </p>
      {/snippet}
      <Button class="w-full mt-4" appearance="primary" size="large" onclick={generate} {disabled}>Generate NSP</Button>
    </Tooltip>
  </div>
</Container>

{#if stdout}<LogOutput title="Standard Out" bind:output={stdout} />{/if}
{#if stderr}<LogOutput title="Standard Err" bind:output={stderr} />{/if}
