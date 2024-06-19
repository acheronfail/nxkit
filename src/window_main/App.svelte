<script lang="ts">
  import Tabs from '../ui/utility/Tabs.svelte';
  import NroForwarder from '../ui/NroForwarder.svelte';
  import PayloadInjector from '../ui/PayloadInjector.svelte';
  import NandExplorer from '../ui/NandExplorer.svelte';
  import Settings from '../ui/Settings.svelte';
  import { keys } from '../ui/stores/keys.svelte';
  import { onMount } from 'svelte';
  import TabItem from '../ui/utility/TabItem.svelte';
  import { ExclamationTriangleIcon } from 'heroicons-svelte/24/solid';

  // TODO: be able to programmatically control selected tab
  // TODO: select tabs with `cmdCtrl+number`

  const defaultOpen = new URLSearchParams(window.location.search).get('tab');

  onMount(async () => {
    const keysFromMain = await window.nxkit.keysFind();
    if (keysFromMain) {
      keys.setMainKeys(keysFromMain);
    }
  });
</script>

<Tabs fillContainer>
  <!-- TODO: tool to split/merge files to/from fat32 chunks -->
  <TabItem defaultOpen={!defaultOpen || defaultOpen === 'forwarder'}>
    <span slot="label">NRO Forwarder</span>
    <NroForwarder slot="content" />
  </TabItem>
  <TabItem defaultOpen={defaultOpen === 'injector'}>
    <span slot="label">Payload Injector</span>
    <PayloadInjector slot="content" />
  </TabItem>
  <TabItem defaultOpen={defaultOpen === 'explorer'}>
    <span slot="label">NAND Explorer</span>
    <NandExplorer slot="content" />
  </TabItem>
  <TabItem defaultOpen={defaultOpen === 'settings'}>
    <span slot="label" class="flex flex-row justify-center items-center">
      Settings
      {#if !keys.value}
        <ExclamationTriangleIcon class="pl-2 h-5 text-yellow-500" />
      {/if}
    </span>
    <Settings slot="content" />
  </TabItem>
</Tabs>
