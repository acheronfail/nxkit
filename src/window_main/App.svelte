<script lang="ts">
  import Tabs from '../ui/utility/Tabs.svelte';
  import NroForwarder from '../ui/NroForwarder.svelte';
  import PayloadInjector from '../ui/PayloadInjector.svelte';
  import NandExplorer from '../ui/NandExplorer.svelte';
  import Settings from '../ui/Settings.svelte';
  import { keys } from '../ui/stores/keys.svelte';
  import { onMount } from 'svelte';
  import TabItem from '../ui/utility/TabItem.svelte';

  let settingsLabel = $state('Settings');
  $effect(() => {
    if (keys.value) {
      settingsLabel = 'Settings';
    } else {
      settingsLabel = 'Settings ⚠️️';
    }
  });

  onMount(async () => {
    const keysFromMain = await window.nxkit.findProdKeys();
    if (keysFromMain) {
      keys.setMainKeys(keysFromMain);
    }
  });
</script>

<h1 class="font-bold text-xl m-2 text-center">💖 NXKit</h1>
<Tabs>
  <TabItem title="Nro Forwarder" defaultOpen>
    <NroForwarder />
  </TabItem>
  <TabItem title="Payload Injector">
    <PayloadInjector />
  </TabItem>
  <TabItem title="Nand Explorer">
    <NandExplorer />
  </TabItem>
  <TabItem title={settingsLabel}>
    <Settings />
  </TabItem>
  <!-- TODO: tool to split/merge files to/from fat32 chunks -->
</Tabs>
