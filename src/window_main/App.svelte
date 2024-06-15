<script lang="ts">
  import Tabs from '../ui/Tabs.svelte';
  import type { Tab } from '../ui/Tabs.types';
  import NroForwarder from '../ui/NroForwarder.svelte';
  import PayloadInjector from '../ui/PayloadInjector.svelte';
  import NandExplorer from '../ui/NandExplorer.svelte';

  const tabs: Tab[] = [
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
    // TODO: tool to split/merge files to/from fat32 chunks
  ];
</script>

<Tabs items={tabs} />
