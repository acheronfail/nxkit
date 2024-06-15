<script lang="ts">
  import Tabs from '../ui/Tabs.svelte';
  import type { Tab } from '../ui/Tabs.types';
  import NroForwarder from '../ui/NroForwarder.svelte';
  import PayloadInjector from '../ui/PayloadInjector.svelte';
  import NandExplorer from '../ui/NandExplorer.svelte';
  import Settings from '../ui/Settings.svelte';
  import { keys } from '../ui/stores/keys.svelte';
  import { onMount } from 'svelte';

  let settingsLabel = $state('Settings');
  $effect(() => {
    if (keys.value) {
      settingsLabel = 'Settings';
    } else {
      settingsLabel = 'Settings ⚠️️';
    }
  });

  const tabs: Tab[] = $derived([
    {
      id: 'nro',
      displayName: 'NRO Forwarder',
      component: NroForwarder,
    },
    {
      id: 'injector',
      displayName: 'Payload Injector',
      component: PayloadInjector,
    },
    {
      id: 'nand',
      displayName: 'Nand Explorer',
      component: NandExplorer,
    },
    {
      id: 'settings',
      displayName: settingsLabel,
      component: Settings,
    },
    // TODO: tool to split/merge files to/from fat32 chunks
  ]);


  onMount(async () => {
    const keysFromMain = await window.nxkit.findProdKeys();
    if (keysFromMain) {
      keys.setMainKeys(keysFromMain);
    }
  });
</script>

<h1>💖 NXKit</h1>
<Tabs items={tabs} />
