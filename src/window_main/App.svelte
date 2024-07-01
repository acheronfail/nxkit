<script lang="ts">
  import NroForwarder from '../ui/NroForwarder.svelte';
  import PayloadInjector from '../ui/PayloadInjector.svelte';
  import NandExplorer from '../ui/NandExplorer.svelte';
  import Settings from '../ui/Settings.svelte';
  import { keys } from '../ui/stores/keys.svelte';
  import { onMount } from 'svelte';
  import { ExclamationTriangleIcon } from 'heroicons-svelte/24/solid';
  import { Tabs, TabList, TabContent, Tab } from '../ui/utility/Tabs';

  const params = new URLSearchParams(window.location.search);
  const [nandFilePath, partitionName, nandReadonlyString = '1'] = params.get('rawnand')?.split(':') ?? [];
  const nandReadonly = Boolean(parseInt(nandReadonlyString));
  const defaultOpen = ['forwarder', 'injector', 'explorer', 'settings'].findIndex((n) => n === params.get('tab'));

  let selected = $state(defaultOpen);

  function keyboardTabHandler(event: KeyboardEvent) {
    const hasModifier = window.nxkit.isOsx ? event.metaKey : event.ctrlKey;
    if (!hasModifier) return;

    switch (event.key) {
      case '1':
      case '2':
      case '3':
      case '4':
        event.preventDefault();
        selected = parseInt(event.key) - 1;
        if (document.activeElement instanceof HTMLElement) {
          document.activeElement.blur();
        }
      default:
        break;
    }
  }

  async function loadKeysFromMain() {
    const keysFromMain = await window.nxkit.call('prodKeysFind');
    if (keysFromMain) {
      keys.setMainKeys(keysFromMain);
    }
  }

  onMount(() => {
    loadKeysFromMain();
    window.addEventListener('keydown', keyboardTabHandler);
    return () => window.removeEventListener('keydown', keyboardTabHandler);
  });

  const widthClass = 'w-[90vw] m-auto';
</script>

{#snippet header()}
  <h1 class="font-bold text-2xl text-center pt-2" style="-webkit-app-region: drag">
    <span class="bg-gradient-to-br from-red-500 to-yellow-500 bg-clip-text text-transparent box-decoration-clone">
      NXKit
    </span>
  </h1>
{/snippet}

<Tabs bind:selected>
  <TabList class={widthClass} {header}>
    <Tab>NRO Forwarder</Tab>
    <Tab>Payload Injector</Tab>
    <Tab>NAND Explorer</Tab>
    <Tab>
      Settings
      {#if !keys.value}
        <ExclamationTriangleIcon class="pl-2 h-5 text-yellow-500" />
      {/if}
    </Tab>
  </TabList>

  <TabContent class={widthClass}>
    <NroForwarder />
  </TabContent>
  <TabContent class={widthClass}>
    <PayloadInjector />
  </TabContent>
  <TabContent class={widthClass}>
    <NandExplorer {nandFilePath} {partitionName} readonlyDefault={nandReadonly} />
  </TabContent>
  <!-- TODO: tool to split/merge files to/from fat32 chunks-->
  <TabContent class={widthClass}>
    <Settings />
  </TabContent>
</Tabs>
